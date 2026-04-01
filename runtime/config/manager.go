package config

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	adapterhealth "github.com/FelixSeptem/baymax/adapter/health"
	"github.com/FelixSeptem/baymax/core/types"
	runtimediag "github.com/FelixSeptem/baymax/runtime/diagnostics"
	"github.com/FelixSeptem/baymax/runtime/security/redaction"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// ManagerOptions controls config loading source and hot reload behavior.
type ManagerOptions struct {
	FilePath        string
	EnvPrefix       string
	EnableHotReload bool
}

// Snapshot is an immutable runtime configuration snapshot used by library consumers.
type Snapshot struct {
	Config   Config         `json:"config"`
	LoadedAt time.Time      `json:"loaded_at"`
	Source   SnapshotSource `json:"source"`
}

// SnapshotSource describes where a snapshot came from.
type SnapshotSource struct {
	FilePath  string `json:"file_path,omitempty"`
	EnvPrefix string `json:"env_prefix"`
}

// Manager owns runtime config snapshots, hot reload, and diagnostics sinks.
type Manager struct {
	filePath  string
	envPrefix string

	snap atomic.Value // *Snapshot
	diag *runtimediag.Store

	readinessMu          sync.RWMutex
	readinessComponents  RuntimeReadinessComponentSnapshot
	reactReadiness       ReactReadinessDependencySnapshot
	adapterHealthMu      sync.RWMutex
	adapterHealthTargets map[string]AdapterHealthTarget
	adapterHealthRunner  *adapterhealth.Runner
	sandboxMu            sync.RWMutex
	sandboxExecutor      types.SandboxExecutor
	sandboxRolloutMu     sync.RWMutex
	sandboxRolloutState  SandboxRolloutRuntimeState

	watchStarted atomic.Bool
	stopOnce     sync.Once
	stopCh       chan struct{}
}

// MailboxDiagnosticRecord is a runtime/config-level projection used by orchestration modules
// to avoid direct dependency on runtime/diagnostics internals.
type MailboxDiagnosticRecord struct {
	Time                  time.Time
	MessageID             string
	IdempotencyKey        string
	CorrelationID         string
	Kind                  string
	State                 string
	FromAgent             string
	ToAgent               string
	RunID                 string
	TaskID                string
	WorkflowID            string
	TeamID                string
	Attempt               int
	ConsumerID            string
	ReasonCode            string
	Backend               string
	ConfiguredBackend     string
	BackendFallback       bool
	BackendFallbackReason string
	PublishPath           string
	Reclaimed             bool
	PanicRecovered        bool
}

// NewManager builds a runtime config manager with env/file/default precedence and optional hot reload.
func NewManager(opts ManagerOptions) (*Manager, error) {
	loadOpts := LoadOptions{FilePath: opts.FilePath, EnvPrefix: opts.EnvPrefix}
	cfg, _, err := loadWithSnapshot(loadOpts)
	if err != nil {
		return nil, err
	}

	m := &Manager{
		filePath:  strings.TrimSpace(opts.FilePath),
		envPrefix: strings.TrimSpace(opts.EnvPrefix),
		diag: runtimediag.NewStore(
			cfg.Diagnostics.MaxCallRecords,
			cfg.Diagnostics.MaxRunRecords,
			cfg.Diagnostics.MaxReloadErrors,
			cfg.Diagnostics.MaxSkillRecords,
			runtimediag.TimelineTrendConfig{
				Enabled:    cfg.Diagnostics.TimelineTrend.Enabled,
				LastNRuns:  cfg.Diagnostics.TimelineTrend.LastNRuns,
				TimeWindow: cfg.Diagnostics.TimelineTrend.TimeWindow,
			},
			runtimediag.CA2ExternalTrendConfig{
				Enabled: cfg.Diagnostics.CA2ExternalTrend.Enabled,
				Window:  cfg.Diagnostics.CA2ExternalTrend.Window,
				Thresholds: runtimediag.CA2ExternalThresholds{
					P95LatencyMs: cfg.Diagnostics.CA2ExternalTrend.Thresholds.P95LatencyMs,
					ErrorRate:    cfg.Diagnostics.CA2ExternalTrend.Thresholds.ErrorRate,
					HitRate:      cfg.Diagnostics.CA2ExternalTrend.Thresholds.HitRate,
				},
			},
		),
		adapterHealthTargets: map[string]AdapterHealthTarget{},
		adapterHealthRunner: adapterhealth.NewRunner(adapterhealth.RunnerOptions{
			ProbeTimeout: cfg.Adapter.Health.ProbeTimeout,
			CacheTTL:     cfg.Adapter.Health.CacheTTL,
			Backoff: adapterhealth.BackoffOptions{
				Enabled:     cfg.Adapter.Health.Backoff.Enabled,
				Initial:     cfg.Adapter.Health.Backoff.Initial,
				Max:         cfg.Adapter.Health.Backoff.Max,
				Multiplier:  cfg.Adapter.Health.Backoff.Multiplier,
				JitterRatio: cfg.Adapter.Health.Backoff.JitterRatio,
			},
			Circuit: adapterhealth.CircuitOptions{
				Enabled:                  cfg.Adapter.Health.Circuit.Enabled,
				FailureThreshold:         cfg.Adapter.Health.Circuit.FailureThreshold,
				OpenDuration:             cfg.Adapter.Health.Circuit.OpenDuration,
				HalfOpenMaxProbe:         cfg.Adapter.Health.Circuit.HalfOpenMaxProbe,
				HalfOpenSuccessThreshold: cfg.Adapter.Health.Circuit.HalfOpenSuccessThreshold,
			},
		}, nil),
		stopCh: make(chan struct{}),
	}
	m.diag.SetCardinalityConfig(runtimediag.CardinalityConfig{
		Enabled:        cfg.Diagnostics.Cardinality.Enabled,
		MaxMapEntries:  cfg.Diagnostics.Cardinality.MaxMapEntries,
		MaxListEntries: cfg.Diagnostics.Cardinality.MaxListEntries,
		MaxStringBytes: cfg.Diagnostics.Cardinality.MaxStringBytes,
		OverflowPolicy: cfg.Diagnostics.Cardinality.OverflowPolicy,
	})
	if m.envPrefix == "" {
		m.envPrefix = DefaultEnvPrefix
	}
	m.snap.Store(&Snapshot{
		Config:   cfg,
		LoadedAt: time.Now(),
		Source: SnapshotSource{
			FilePath:  m.filePath,
			EnvPrefix: m.envPrefix,
		},
	})
	if opts.EnableHotReload || cfg.Reload.Enabled {
		if err := m.Watch(context.Background()); err != nil {
			return nil, err
		}
	}
	return m, nil
}

// Close stops background reload watchers.
func (m *Manager) Close() error {
	m.stopOnce.Do(func() { close(m.stopCh) })
	return nil
}

// Watch starts file-based hot reload loop if not already started.
func (m *Manager) Watch(ctx context.Context) error {
	if strings.TrimSpace(m.filePath) == "" {
		return fmt.Errorf("hot reload requires a config file path")
	}
	if !m.watchStarted.CompareAndSwap(false, true) {
		return nil
	}

	w := viper.New()
	w.SetConfigFile(m.filePath)
	w.SetConfigType("yaml")
	if err := w.ReadInConfig(); err != nil {
		return fmt.Errorf("watch runtime config: %w", err)
	}

	events := make(chan struct{}, 1)
	w.OnConfigChange(func(_ fsnotify.Event) {
		select {
		case events <- struct{}{}:
		default:
		}
	})
	w.WatchConfig()

	go m.watchLoop(ctx, events)
	return nil
}

func (m *Manager) watchLoop(ctx context.Context, events <-chan struct{}) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case <-events:
			debounce := m.EffectiveConfig().Reload.Debounce
			if debounce <= 0 {
				debounce = 100 * time.Millisecond
			}
			timer := time.NewTimer(debounce)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-m.stopCh:
				timer.Stop()
				return
			case <-timer.C:
			}
			m.reload()
		}
	}
}

func (m *Manager) reload() {
	cfg, _, err := loadWithSnapshot(LoadOptions{FilePath: m.filePath, EnvPrefix: m.envPrefix})
	if err != nil {
		m.diag.AddReload(runtimediag.ReloadRecord{Time: time.Now(), Success: false, Error: err.Error()})
		return
	}
	current := m.EffectiveConfig()
	if err := validateSandboxRolloutPhaseTransition(current.Security.Sandbox.Rollout.Phase, cfg.Security.Sandbox.Rollout.Phase); err != nil {
		m.diag.AddReload(runtimediag.ReloadRecord{Time: time.Now(), Success: false, Error: err.Error()})
		return
	}
	if err := validateSandboxRolloutUnfreezeTransition(
		current.Security.Sandbox,
		cfg.Security.Sandbox,
		m.SandboxRolloutRuntimeState(),
		time.Now().UTC(),
	); err != nil {
		m.diag.AddReload(runtimediag.ReloadRecord{Time: time.Now(), Success: false, Error: err.Error()})
		return
	}
	m.snap.Store(&Snapshot{
		Config:   cfg,
		LoadedAt: time.Now(),
		Source: SnapshotSource{
			FilePath:  m.filePath,
			EnvPrefix: m.envPrefix,
		},
	})
	m.diag.Resize(
		cfg.Diagnostics.MaxCallRecords,
		cfg.Diagnostics.MaxRunRecords,
		cfg.Diagnostics.MaxReloadErrors,
		cfg.Diagnostics.MaxSkillRecords,
	)
	m.diag.SetTrendConfig(runtimediag.TimelineTrendConfig{
		Enabled:    cfg.Diagnostics.TimelineTrend.Enabled,
		LastNRuns:  cfg.Diagnostics.TimelineTrend.LastNRuns,
		TimeWindow: cfg.Diagnostics.TimelineTrend.TimeWindow,
	})
	m.diag.SetCA2ExternalTrendConfig(runtimediag.CA2ExternalTrendConfig{
		Enabled: cfg.Diagnostics.CA2ExternalTrend.Enabled,
		Window:  cfg.Diagnostics.CA2ExternalTrend.Window,
		Thresholds: runtimediag.CA2ExternalThresholds{
			P95LatencyMs: cfg.Diagnostics.CA2ExternalTrend.Thresholds.P95LatencyMs,
			ErrorRate:    cfg.Diagnostics.CA2ExternalTrend.Thresholds.ErrorRate,
			HitRate:      cfg.Diagnostics.CA2ExternalTrend.Thresholds.HitRate,
		},
	})
	m.diag.SetCardinalityConfig(runtimediag.CardinalityConfig{
		Enabled:        cfg.Diagnostics.Cardinality.Enabled,
		MaxMapEntries:  cfg.Diagnostics.Cardinality.MaxMapEntries,
		MaxListEntries: cfg.Diagnostics.Cardinality.MaxListEntries,
		MaxStringBytes: cfg.Diagnostics.Cardinality.MaxStringBytes,
		OverflowPolicy: cfg.Diagnostics.Cardinality.OverflowPolicy,
	})
	m.updateAdapterHealthRunnerOptions(cfg.Adapter.Health)
	m.diag.AddReload(runtimediag.ReloadRecord{Time: time.Now(), Success: true})
}

// EffectiveConfig returns the current effective runtime configuration.
func (m *Manager) EffectiveConfig() Config {
	s := m.snapshot()
	if s == nil {
		return DefaultConfig()
	}
	return s.Config
}

// CurrentSnapshot returns the current immutable snapshot metadata.
func (m *Manager) CurrentSnapshot() Snapshot {
	s := m.snapshot()
	if s == nil {
		return Snapshot{Config: DefaultConfig(), LoadedAt: time.Now()}
	}
	return *s
}

// EffectiveConfigSanitized returns redacted effective config for diagnostics surfaces.
func (m *Manager) EffectiveConfigSanitized() map[string]any {
	s := m.snapshot()
	if s == nil {
		raw, _ := toMap(DefaultConfig())
		return m.redactor().SanitizeMap(raw)
	}
	raw, err := toMap(s.Config)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return m.redactor().SanitizeMap(raw)
}

// ResolvePolicy resolves MCP runtime policy by profile and optional override.
func (m *Manager) ResolvePolicy(profile string, override *types.MCPRuntimePolicy) (types.MCPRuntimePolicy, error) {
	return ResolveMCPPolicyWithConfig(m.EffectiveConfig(), profile, override)
}

// RecordCall appends an MCP call diagnostics record.
func (m *Manager) RecordCall(rec runtimediag.CallRecord) {
	m.diag.AddCall(rec)
}

// RecordRun appends a run diagnostics record.
func (m *Manager) RecordRun(rec runtimediag.RunRecord) {
	m.diag.AddRun(rec)
	m.updateSandboxRolloutGovernanceFromRun(rec)
}

// RecordRunTimelineEvent appends a normalized timeline event sample for run aggregation.
func (m *Manager) RecordRunTimelineEvent(runID, phase, status string, sequence int64, ts time.Time) {
	m.diag.AddTimelineEvent(runID, phase, status, sequence, ts)
}

// RecordReload appends a config reload diagnostics record.
func (m *Manager) RecordReload(rec runtimediag.ReloadRecord) {
	m.diag.AddReload(rec)
}

// RecordSkill appends a skill diagnostics record after payload redaction.
func (m *Manager) RecordSkill(rec runtimediag.SkillRecord) {
	if len(rec.Payload) > 0 {
		rec.Payload = m.redactor().SanitizeMap(rec.Payload)
	}
	m.diag.AddSkill(rec)
}

// RecentCalls returns recent MCP call diagnostics records.
func (m *Manager) RecentCalls(n int) []runtimediag.CallRecord {
	return m.diag.RecentCalls(n)
}

// RecentRuns returns recent run diagnostics records.
func (m *Manager) RecentRuns(n int) []runtimediag.RunRecord {
	return m.diag.RecentRuns(n)
}

// QueryRuns executes unified diagnostics query with filters/pagination/sort/cursor.
func (m *Manager) QueryRuns(req runtimediag.UnifiedRunQueryRequest) (runtimediag.UnifiedRunQueryResult, error) {
	return m.diag.QueryRuns(req)
}

// RecordMailbox appends a mailbox diagnostics record.
func (m *Manager) RecordMailbox(rec runtimediag.MailboxRecord) {
	m.diag.AddMailbox(rec)
}

// RecordMailboxDiagnostic appends a mailbox diagnostics record using runtime/config-level DTO.
func (m *Manager) RecordMailboxDiagnostic(rec MailboxDiagnosticRecord) {
	m.diag.AddMailbox(runtimediag.MailboxRecord{
		Time:                  rec.Time,
		MessageID:             rec.MessageID,
		IdempotencyKey:        rec.IdempotencyKey,
		CorrelationID:         rec.CorrelationID,
		Kind:                  rec.Kind,
		State:                 rec.State,
		FromAgent:             rec.FromAgent,
		ToAgent:               rec.ToAgent,
		RunID:                 rec.RunID,
		TaskID:                rec.TaskID,
		WorkflowID:            rec.WorkflowID,
		TeamID:                rec.TeamID,
		Attempt:               rec.Attempt,
		ConsumerID:            rec.ConsumerID,
		ReasonCode:            rec.ReasonCode,
		Backend:               rec.Backend,
		ConfiguredBackend:     rec.ConfiguredBackend,
		BackendFallback:       rec.BackendFallback,
		BackendFallbackReason: rec.BackendFallbackReason,
		PublishPath:           rec.PublishPath,
		Reclaimed:             rec.Reclaimed,
		PanicRecovered:        rec.PanicRecovered,
	})
}

// RecentMailbox returns recent mailbox diagnostics records.
func (m *Manager) RecentMailbox(n int) []runtimediag.MailboxRecord {
	return m.diag.RecentMailbox(n)
}

// QueryMailbox executes mailbox diagnostics query with filters/pagination/sort/cursor.
func (m *Manager) QueryMailbox(req runtimediag.MailboxQueryRequest) (runtimediag.MailboxQueryResult, error) {
	return m.diag.QueryMailbox(req)
}

// MailboxAggregates returns mailbox aggregate counters for observability composition.
func (m *Manager) MailboxAggregates(req runtimediag.MailboxAggregateRequest) runtimediag.MailboxAggregate {
	return m.diag.MailboxAggregates(req)
}

// TimelineTrends returns cross-run timeline trend aggregates.
func (m *Manager) TimelineTrends(query runtimediag.TimelineTrendQuery) []runtimediag.TimelineTrendRecord {
	return m.diag.TimelineTrends(query)
}

// CA2ExternalTrends returns provider-scoped CA2 external retriever trend aggregates.
func (m *Manager) CA2ExternalTrends(query runtimediag.CA2ExternalTrendQuery) []runtimediag.CA2ExternalTrendRecord {
	return m.diag.CA2ExternalTrends(query)
}

// RecentReloads returns recent hot-reload diagnostics records.
func (m *Manager) RecentReloads(n int) []runtimediag.ReloadRecord {
	return m.diag.RecentReloads(n)
}

// RecentSkills returns recent skill lifecycle diagnostics records.
func (m *Manager) RecentSkills(n int) []runtimediag.SkillRecord {
	return m.diag.RecentSkills(n)
}

func (m *Manager) snapshot() *Snapshot {
	v := m.snap.Load()
	if v == nil {
		return nil
	}
	s, _ := v.(*Snapshot)
	return s
}

func (m *Manager) SetSandboxExecutor(executor types.SandboxExecutor) {
	if m == nil {
		return
	}
	m.sandboxMu.Lock()
	m.sandboxExecutor = executor
	m.sandboxMu.Unlock()
}

func (m *Manager) SandboxExecutor() types.SandboxExecutor {
	if m == nil {
		return nil
	}
	m.sandboxMu.RLock()
	defer m.sandboxMu.RUnlock()
	return m.sandboxExecutor
}

func (m *Manager) updateSandboxRolloutGovernanceFromRun(rec runtimediag.RunRecord) {
	if m == nil {
		return
	}
	sandboxCfg := m.EffectiveConfig().Security.Sandbox
	if !sandboxCfg.Enabled {
		return
	}

	m.sandboxRolloutMu.Lock()
	defer m.sandboxRolloutMu.Unlock()

	state := cloneSandboxRolloutRuntimeState(m.sandboxRolloutState)
	if rec.SchedulerQueueTotal > 0 {
		state.CapacityQueueDepth = rec.SchedulerQueueTotal
	}
	if rec.InflightPeak > 0 {
		state.CapacityInflight = rec.InflightPeak
	}
	if rec.SandboxEgressViolationTotal > 0 {
		state.EgressViolationTotal += rec.SandboxEgressViolationTotal
	}
	state.CapacityAction = evaluateSandboxCapacityAction(sandboxCfg.Capacity, state.CapacityQueueDepth, state.CapacityInflight)
	state.HealthBudgetStatus = evaluateSandboxHealthBudgetStatus(sandboxCfg.Rollout, rec)
	if state.HealthBudgetStatus == SandboxHealthBudgetBreached {
		state.HealthBudgetBreachTotal++
		if sandboxCfg.Rollout.FreezeOnBreach {
			if !state.FreezeState {
				state.UpdatedAt = time.Now().UTC()
			}
			state.FreezeState = true
			state.FreezeReasonCode = ReadinessCodeSandboxRolloutHealthBreached
			state.CapacityAction = SandboxCapacityActionDeny
		}
	}
	if state.HealthBudgetStatus == SandboxHealthBudgetWithinBudget && state.HealthBudgetBreachTotal > 0 {
		state.HealthBudgetBreachTotal--
	}
	if strings.ToLower(strings.TrimSpace(sandboxCfg.Rollout.Phase)) == SecuritySandboxRolloutPhaseFrozen {
		state.FreezeState = true
		if strings.TrimSpace(state.FreezeReasonCode) == "" {
			state.FreezeReasonCode = ReadinessCodeSandboxRolloutFrozen
		}
		state.CapacityAction = SandboxCapacityActionDeny
	}
	if state.UpdatedAt.IsZero() {
		state.UpdatedAt = time.Now().UTC()
	}
	m.sandboxRolloutState = state
}

func evaluateSandboxHealthBudgetStatus(cfg SecuritySandboxRolloutConfig, rec runtimediag.RunRecord) string {
	if cfg.ErrorBudget <= 0 {
		return SandboxHealthBudgetWithinBudget
	}
	launchFailure := rec.SandboxLaunchFailedTotal > 0
	timeoutBreach := rec.SandboxTimeoutTotal > 0
	violation := rec.SandboxCapabilityMismatchTotal > 0
	admissionDeny := rec.RuntimeReadinessAdmissionBlockedTotal > 0
	if code := strings.ToLower(strings.TrimSpace(rec.RuntimeReadinessAdmissionPrimaryCode)); code == ReadinessCodeSandboxRolloutFrozen || code == ReadinessCodeSandboxRolloutCapacityBlocked {
		admissionDeny = true
	}

	p95LatencyDelta := false
	if rec.SandboxExecLatencyMsP95 > 0 && rec.LatencyMs > 0 && rec.SandboxExecLatencyMsP95 > rec.LatencyMs {
		delta := float64(rec.SandboxExecLatencyMsP95-rec.LatencyMs) / float64(rec.LatencyMs)
		p95LatencyDelta = delta > cfg.ErrorBudget
	}

	breaches := 0
	for _, hit := range []bool{launchFailure, timeoutBreach, violation, p95LatencyDelta, admissionDeny} {
		if hit {
			breaches++
		}
	}
	if breaches == 0 {
		return SandboxHealthBudgetWithinBudget
	}
	rate := float64(breaches) / 5.0
	if rate > cfg.ErrorBudget {
		return SandboxHealthBudgetBreached
	}
	if rate >= cfg.ErrorBudget*0.8 {
		return SandboxHealthBudgetNearBudget
	}
	return SandboxHealthBudgetWithinBudget
}

func validateSandboxRolloutUnfreezeTransition(
	current SecuritySandboxConfig,
	next SecuritySandboxConfig,
	state SandboxRolloutRuntimeState,
	now time.Time,
) error {
	from := strings.ToLower(strings.TrimSpace(current.Rollout.Phase))
	to := strings.ToLower(strings.TrimSpace(next.Rollout.Phase))
	if from != SecuritySandboxRolloutPhaseFrozen || to == SecuritySandboxRolloutPhaseFrozen {
		return nil
	}
	if cooldown := next.Rollout.Cooldown; cooldown > 0 && !state.UpdatedAt.IsZero() {
		if now.Before(state.UpdatedAt.Add(cooldown)) {
			return fmt.Errorf(
				"security.sandbox.rollout transition from frozen requires cooldown completion, remaining=%s",
				state.UpdatedAt.Add(cooldown).Sub(now),
			)
		}
	}
	expectedToken := strings.TrimSpace(current.Rollout.ManualUnfreezeToken)
	providedToken := strings.TrimSpace(next.Rollout.ManualUnfreezeToken)
	if expectedToken == "" || providedToken == "" || expectedToken != providedToken {
		return errors.New("security.sandbox.rollout transition from frozen requires matching manual_unfreeze_token")
	}
	return nil
}

// RedactPayload applies configured redaction to generic payload maps.
func (m *Manager) RedactPayload(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	return m.redactor().SanitizeMap(in)
}

// RedactJSONText applies configured redaction to serialized JSON payload text.
func (m *Manager) RedactJSONText(in string) string {
	return m.redactor().SanitizeJSONText(in)
}

// PrecheckStage2External validates external retriever wiring before runtime usage.
func (m *Manager) PrecheckStage2External(provider string, external ContextAssemblerCA2ExternalConfig) ExternalPrecheckResult {
	return PrecheckStage2External(provider, external)
}

func (m *Manager) redactor() *redaction.Redactor {
	s := m.snapshot()
	if s == nil {
		base := DefaultConfig()
		return redaction.New(base.Security.Redaction.Enabled, base.Security.Redaction.Keywords)
	}
	return redaction.New(s.Config.Security.Redaction.Enabled, s.Config.Security.Redaction.Keywords)
}
