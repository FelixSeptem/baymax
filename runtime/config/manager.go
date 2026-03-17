package config

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

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

	watchStarted atomic.Bool
	stopOnce     sync.Once
	stopCh       chan struct{}
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
		stopCh: make(chan struct{}),
	}
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
