package config

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	adapterhealth "github.com/FelixSeptem/baymax/adapter/health"
)

type ReadinessStatus string

const (
	ReadinessStatusReady    ReadinessStatus = "ready"
	ReadinessStatusDegraded ReadinessStatus = "degraded"
	ReadinessStatusBlocked  ReadinessStatus = "blocked"
)

const (
	ReadinessDomainRuntime   = "runtime"
	ReadinessDomainConfig    = "config"
	ReadinessDomainScheduler = "scheduler"
	ReadinessDomainMailbox   = "mailbox"
	ReadinessDomainRecovery  = "recovery"
	ReadinessDomainAdapter   = "adapter"
)

const (
	ReadinessSeverityInfo    = "info"
	ReadinessSeverityWarning = "warning"
	ReadinessSeverityError   = "error"
)

const (
	ReadinessCodeConfigInvalid              = "runtime.config.invalid"
	ReadinessCodeStrictEscalated            = "runtime.readiness.strict_escalated"
	ReadinessCodeSchedulerFallback          = "scheduler.backend.fallback"
	ReadinessCodeSchedulerActivationError   = "scheduler.backend.activation_failed"
	ReadinessCodeMailboxFallback            = "mailbox.backend.fallback"
	ReadinessCodeMailboxActivationError     = "mailbox.backend.activation_failed"
	ReadinessCodeRecoveryFallback           = "recovery.backend.fallback"
	ReadinessCodeRecoveryActivationError    = "recovery.backend.activation_failed"
	ReadinessCodeRuntimeManagerUnavailable  = "runtime.manager.unavailable"
	ReadinessCodeAdapterRequiredUnavailable = "adapter.health.required_unavailable"
	ReadinessCodeAdapterOptionalUnavailable = "adapter.health.optional_unavailable"
	ReadinessCodeAdapterDegraded            = "adapter.health.degraded"
)

type ReadinessAdmissionOutcome string

const (
	ReadinessAdmissionOutcomeAllow ReadinessAdmissionOutcome = "allow"
	ReadinessAdmissionOutcomeDeny  ReadinessAdmissionOutcome = "deny"
)

const (
	ReadinessAdmissionCodeBypassDisabled  = "runtime.readiness.admission.disabled"
	ReadinessAdmissionCodeReady           = "runtime.readiness.admission.ready"
	ReadinessAdmissionCodeBlocked         = "runtime.readiness.admission.blocked"
	ReadinessAdmissionCodeDegradedAllow   = "runtime.readiness.admission.degraded_allow"
	ReadinessAdmissionCodeDegradedDeny    = "runtime.readiness.admission.degraded_fail_fast"
	ReadinessAdmissionCodeUnknownStatus   = "runtime.readiness.admission.unknown_status"
	ReadinessAdmissionCodeManagerNotReady = "runtime.readiness.admission.manager_unavailable"
)

type ReadinessFinding struct {
	Code     string         `json:"code"`
	Domain   string         `json:"domain"`
	Severity string         `json:"severity"`
	Message  string         `json:"message"`
	Metadata map[string]any `json:"metadata"`
}

type ReadinessResult struct {
	Status        ReadinessStatus           `json:"status"`
	Findings      []ReadinessFinding        `json:"findings"`
	AdapterHealth []AdapterHealthEvaluation `json:"adapter_health,omitempty"`
	EvaluatedAt   time.Time                 `json:"evaluated_at"`
}

type ReadinessSummary struct {
	Status                        string `json:"runtime_readiness_status"`
	FindingTotal                  int    `json:"runtime_readiness_finding_total"`
	BlockingTotal                 int    `json:"runtime_readiness_blocking_total"`
	DegradedTotal                 int    `json:"runtime_readiness_degraded_total"`
	PrimaryCode                   string `json:"runtime_readiness_primary_code"`
	AdapterHealthStatus           string `json:"adapter_health_status,omitempty"`
	AdapterHealthProbeTotal       int    `json:"adapter_health_probe_total,omitempty"`
	AdapterHealthDegradedTotal    int    `json:"adapter_health_degraded_total,omitempty"`
	AdapterHealthUnavailableTotal int    `json:"adapter_health_unavailable_total,omitempty"`
	AdapterHealthPrimaryCode      string `json:"adapter_health_primary_code,omitempty"`
}

type ReadinessAdmissionDecision struct {
	Enabled              bool                      `json:"enabled"`
	Mode                 string                    `json:"mode"`
	BlockOn              string                    `json:"block_on"`
	DegradedPolicy       string                    `json:"degraded_policy"`
	Outcome              ReadinessAdmissionOutcome `json:"outcome"`
	ReasonCode           string                    `json:"reason_code"`
	ReadinessStatus      ReadinessStatus           `json:"readiness_status"`
	ReadinessPrimaryCode string                    `json:"readiness_primary_code,omitempty"`
	Bypass               bool                      `json:"bypass"`
}

type AdapterHealthTarget struct {
	Name     string              `json:"name"`
	Required bool                `json:"required"`
	Probe    adapterhealth.Probe `json:"-"`
	Metadata map[string]any      `json:"metadata,omitempty"`
}

type AdapterHealthEvaluation struct {
	Name      string         `json:"name"`
	Required  bool           `json:"required"`
	Status    string         `json:"status"`
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	Metadata  map[string]any `json:"metadata"`
	CheckedAt time.Time      `json:"checked_at"`
}

type RuntimeReadinessComponentState struct {
	Enabled           bool   `json:"enabled"`
	ConfiguredBackend string `json:"configured_backend,omitempty"`
	EffectiveBackend  string `json:"effective_backend,omitempty"`
	Fallback          bool   `json:"fallback,omitempty"`
	FallbackReason    string `json:"fallback_reason,omitempty"`
	ActivationError   string `json:"activation_error,omitempty"`
}

type RuntimeReadinessComponentSnapshot struct {
	Scheduler RuntimeReadinessComponentState `json:"scheduler"`
	Mailbox   RuntimeReadinessComponentState `json:"mailbox"`
	Recovery  RuntimeReadinessComponentState `json:"recovery"`
	UpdatedAt time.Time                      `json:"updated_at,omitempty"`
}

func (m *Manager) SetReadinessComponentSnapshot(snapshot RuntimeReadinessComponentSnapshot) {
	if m == nil {
		return
	}
	m.readinessMu.Lock()
	m.readinessComponents = cloneReadinessComponentSnapshot(snapshot)
	m.readinessMu.Unlock()
}

func (m *Manager) ReadinessComponentSnapshot() RuntimeReadinessComponentSnapshot {
	if m == nil {
		return RuntimeReadinessComponentSnapshot{}
	}
	m.readinessMu.RLock()
	defer m.readinessMu.RUnlock()
	return cloneReadinessComponentSnapshot(m.readinessComponents)
}

func (m *Manager) SetAdapterHealthTargets(targets []AdapterHealthTarget) {
	if m == nil {
		return
	}
	m.adapterHealthMu.Lock()
	defer m.adapterHealthMu.Unlock()
	m.adapterHealthTargets = normalizeAdapterHealthTargets(targets)
}

func (m *Manager) RegisterAdapterHealthTarget(target AdapterHealthTarget) error {
	if m == nil {
		return nil
	}
	name := strings.ToLower(strings.TrimSpace(target.Name))
	if name == "" {
		return fmt.Errorf("adapter health target name is required")
	}
	if target.Probe == nil {
		return fmt.Errorf("adapter health target %q probe is required", name)
	}
	normalized := target
	normalized.Name = name
	normalized.Metadata = cloneAnyMap(target.Metadata)
	m.adapterHealthMu.Lock()
	defer m.adapterHealthMu.Unlock()
	if m.adapterHealthTargets == nil {
		m.adapterHealthTargets = map[string]AdapterHealthTarget{}
	}
	m.adapterHealthTargets[name] = normalized
	return nil
}

func (m *Manager) RemoveAdapterHealthTarget(name string) {
	if m == nil {
		return
	}
	normalized := strings.ToLower(strings.TrimSpace(name))
	if normalized == "" {
		return
	}
	m.adapterHealthMu.Lock()
	defer m.adapterHealthMu.Unlock()
	delete(m.adapterHealthTargets, normalized)
}

func (m *Manager) AdapterHealthTargets() []AdapterHealthTarget {
	if m == nil {
		return nil
	}
	m.adapterHealthMu.RLock()
	defer m.adapterHealthMu.RUnlock()
	if len(m.adapterHealthTargets) == 0 {
		return nil
	}
	out := make([]AdapterHealthTarget, 0, len(m.adapterHealthTargets))
	for _, target := range m.adapterHealthTargets {
		item := target
		item.Metadata = cloneAnyMap(target.Metadata)
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func (m *Manager) ReadinessPreflight() ReadinessResult {
	evaluatedAt := time.Now().UTC()
	if m == nil {
		return ReadinessResult{
			Status: ReadinessStatusBlocked,
			Findings: []ReadinessFinding{
				{
					Code:     ReadinessCodeRuntimeManagerUnavailable,
					Domain:   ReadinessDomainRuntime,
					Severity: ReadinessSeverityError,
					Message:  "runtime manager is unavailable",
					Metadata: map[string]any{},
				},
			},
			EvaluatedAt: evaluatedAt,
		}
	}

	cfg := m.EffectiveConfig()
	if !cfg.Runtime.Readiness.Enabled {
		return ReadinessResult{
			Status:      ReadinessStatusReady,
			Findings:    nil,
			EvaluatedAt: evaluatedAt,
		}
	}

	findings := make([]ReadinessFinding, 0, 6)
	var adapterResults []AdapterHealthEvaluation
	if err := Validate(cfg); err != nil {
		findings = append(findings, ReadinessFinding{
			Code:     ReadinessCodeConfigInvalid,
			Domain:   ReadinessDomainConfig,
			Severity: ReadinessSeverityError,
			Message:  "effective runtime config is invalid",
			Metadata: map[string]any{"error": strings.TrimSpace(err.Error())},
		})
	}

	componentSnapshot := m.ReadinessComponentSnapshot()
	findings = append(findings, componentReadinessFindings("scheduler", componentSnapshot.Scheduler)...)
	findings = append(findings, componentReadinessFindings("mailbox", componentSnapshot.Mailbox)...)
	findings = append(findings, componentReadinessFindings("recovery", componentSnapshot.Recovery)...)
	adapterResults, adapterFindings := m.adapterHealthReadinessFindings(cfg)
	findings = append(findings, adapterFindings...)
	findings = canonicalizeReadinessFindings(findings)

	status := classifyReadinessStatus(findings)
	if cfg.Runtime.Readiness.Strict && status == ReadinessStatusDegraded {
		findings = append(findings, ReadinessFinding{
			Code:     ReadinessCodeStrictEscalated,
			Domain:   ReadinessDomainRuntime,
			Severity: ReadinessSeverityError,
			Message:  "strict readiness policy escalated degraded findings to blocked",
			Metadata: map[string]any{"strict": true},
		})
		findings = canonicalizeReadinessFindings(findings)
		status = ReadinessStatusBlocked
	}

	return ReadinessResult{
		Status:        status,
		Findings:      findings,
		AdapterHealth: adapterResults,
		EvaluatedAt:   evaluatedAt,
	}
}

func (m *Manager) EvaluateReadinessAdmission() ReadinessAdmissionDecision {
	if m == nil {
		return ReadinessAdmissionDecision{
			Enabled:         false,
			Mode:            ReadinessAdmissionModeFailFast,
			BlockOn:         ReadinessAdmissionBlockOnBlockedOnly,
			DegradedPolicy:  ReadinessAdmissionDegradedPolicyAllowAndRecord,
			Outcome:         ReadinessAdmissionOutcomeAllow,
			ReasonCode:      ReadinessAdmissionCodeManagerNotReady,
			ReadinessStatus: ReadinessStatusBlocked,
			Bypass:          true,
		}
	}

	cfg := m.EffectiveConfig().Runtime.Readiness.Admission
	decision := ReadinessAdmissionDecision{
		Enabled:         cfg.Enabled,
		Mode:            normalizeReadinessAdmissionMode(cfg.Mode),
		BlockOn:         normalizeReadinessAdmissionBlockOn(cfg.BlockOn),
		DegradedPolicy:  normalizeReadinessAdmissionDegradedPolicy(cfg.DegradedPolicy),
		Outcome:         ReadinessAdmissionOutcomeAllow,
		ReasonCode:      ReadinessAdmissionCodeBypassDisabled,
		ReadinessStatus: ReadinessStatusReady,
		Bypass:          true,
	}
	if !decision.Enabled {
		return decision
	}

	preflight := m.ReadinessPreflight()
	summary := preflight.Summary()
	decision.Bypass = false
	decision.ReadinessStatus = preflight.Status
	decision.ReadinessPrimaryCode = strings.TrimSpace(summary.PrimaryCode)
	switch preflight.Status {
	case ReadinessStatusReady:
		decision.Outcome = ReadinessAdmissionOutcomeAllow
		decision.ReasonCode = ReadinessAdmissionCodeReady
	case ReadinessStatusBlocked:
		decision.Outcome = ReadinessAdmissionOutcomeDeny
		decision.ReasonCode = ReadinessAdmissionCodeBlocked
	case ReadinessStatusDegraded:
		if decision.DegradedPolicy == ReadinessAdmissionDegradedPolicyFailFast {
			decision.Outcome = ReadinessAdmissionOutcomeDeny
			decision.ReasonCode = ReadinessAdmissionCodeDegradedDeny
		} else {
			decision.Outcome = ReadinessAdmissionOutcomeAllow
			decision.ReasonCode = ReadinessAdmissionCodeDegradedAllow
		}
	default:
		decision.Outcome = ReadinessAdmissionOutcomeDeny
		decision.ReasonCode = ReadinessAdmissionCodeUnknownStatus
	}
	if decision.ReadinessPrimaryCode == "" {
		decision.ReadinessPrimaryCode = decision.ReasonCode
	}
	return decision
}

func (r ReadinessResult) Summary() ReadinessSummary {
	summary := ReadinessSummary{
		Status: string(r.Status),
	}
	for i := range r.Findings {
		finding := r.Findings[i]
		summary.FindingTotal++
		switch strings.ToLower(strings.TrimSpace(finding.Severity)) {
		case ReadinessSeverityError:
			summary.BlockingTotal++
			if summary.PrimaryCode == "" {
				summary.PrimaryCode = strings.TrimSpace(finding.Code)
			}
		case ReadinessSeverityWarning:
			summary.DegradedTotal++
			if summary.PrimaryCode == "" {
				summary.PrimaryCode = strings.TrimSpace(finding.Code)
			}
		}
	}
	if strings.TrimSpace(summary.Status) == "" {
		summary.Status = string(ReadinessStatusReady)
	}
	if len(r.AdapterHealth) > 0 {
		adapterStatus := string(adapterhealth.StatusHealthy)
		primaryRank := -1
		for i := range r.AdapterHealth {
			item := r.AdapterHealth[i]
			status := normalizeAdapterHealthStatus(item.Status)
			summary.AdapterHealthProbeTotal++
			switch status {
			case adapterhealth.StatusUnavailable:
				summary.AdapterHealthUnavailableTotal++
				if primaryRank < 2 && strings.TrimSpace(item.Code) != "" {
					summary.AdapterHealthPrimaryCode = strings.TrimSpace(item.Code)
					primaryRank = 2
				}
				adapterStatus = string(adapterhealth.StatusUnavailable)
			case adapterhealth.StatusDegraded:
				summary.AdapterHealthDegradedTotal++
				if primaryRank < 1 && strings.TrimSpace(item.Code) != "" {
					summary.AdapterHealthPrimaryCode = strings.TrimSpace(item.Code)
					primaryRank = 1
				}
				if adapterStatus != string(adapterhealth.StatusUnavailable) {
					adapterStatus = string(adapterhealth.StatusDegraded)
				}
			default:
			}
		}
		summary.AdapterHealthStatus = adapterStatus
	}
	return summary
}

func componentReadinessFindings(component string, state RuntimeReadinessComponentState) []ReadinessFinding {
	component = strings.ToLower(strings.TrimSpace(component))
	if component == "" {
		return nil
	}

	metadata := map[string]any{}
	if configured := strings.ToLower(strings.TrimSpace(state.ConfiguredBackend)); configured != "" {
		metadata["configured_backend"] = configured
	}
	if effective := strings.ToLower(strings.TrimSpace(state.EffectiveBackend)); effective != "" {
		metadata["effective_backend"] = effective
	}

	if activationErr := strings.TrimSpace(state.ActivationError); activationErr != "" {
		metadata["activation_error"] = activationErr
		return []ReadinessFinding{
			{
				Code:     readinessActivationCode(component),
				Domain:   component,
				Severity: ReadinessSeverityError,
				Message:  fmt.Sprintf("%s backend activation failed", component),
				Metadata: metadata,
			},
		}
	}

	if state.Fallback {
		if reason := strings.TrimSpace(state.FallbackReason); reason != "" {
			metadata["fallback_reason"] = reason
		}
		return []ReadinessFinding{
			{
				Code:     readinessFallbackCode(component),
				Domain:   component,
				Severity: ReadinessSeverityWarning,
				Message:  fmt.Sprintf("%s backend fell back to memory", component),
				Metadata: metadata,
			},
		}
	}

	return nil
}

func readinessFallbackCode(component string) string {
	switch component {
	case "scheduler":
		return ReadinessCodeSchedulerFallback
	case "mailbox":
		return ReadinessCodeMailboxFallback
	case "recovery":
		return ReadinessCodeRecoveryFallback
	default:
		return component + ".backend.fallback"
	}
}

func readinessActivationCode(component string) string {
	switch component {
	case "scheduler":
		return ReadinessCodeSchedulerActivationError
	case "mailbox":
		return ReadinessCodeMailboxActivationError
	case "recovery":
		return ReadinessCodeRecoveryActivationError
	default:
		return component + ".backend.activation_failed"
	}
}

func (m *Manager) adapterHealthReadinessFindings(cfg Config) ([]AdapterHealthEvaluation, []ReadinessFinding) {
	if m == nil || !cfg.Adapter.Health.Enabled {
		return nil, nil
	}
	targets := m.AdapterHealthTargets()
	if len(targets) == 0 {
		return nil, nil
	}
	m.updateAdapterHealthRunnerOptions(cfg.Adapter.Health)
	runner := m.adapterHealthRunnerSnapshot()
	if runner == nil {
		return nil, nil
	}

	results := make([]AdapterHealthEvaluation, 0, len(targets))
	findings := make([]ReadinessFinding, 0, len(targets))
	for i := range targets {
		target := targets[i]
		probeResult := runner.Probe(context.Background(), target.Name, target.Probe)
		eval := AdapterHealthEvaluation{
			Name:      strings.ToLower(strings.TrimSpace(target.Name)),
			Required:  target.Required,
			Status:    string(normalizeAdapterHealthStatus(string(probeResult.Status))),
			Code:      strings.TrimSpace(probeResult.Code),
			Message:   strings.TrimSpace(probeResult.Message),
			Metadata:  cloneAnyMap(probeResult.Metadata),
			CheckedAt: probeResult.CheckedAt.UTC(),
		}
		if eval.Metadata == nil {
			eval.Metadata = map[string]any{}
		}
		delete(eval.Metadata, "cache_hit")
		results = append(results, eval)

		finding, ok := adapterHealthReadinessFinding(target, probeResult, cfg.Adapter.Health.Strict, cfg.Runtime.Readiness.Strict)
		if ok {
			findings = append(findings, finding)
		}
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Name != results[j].Name {
			return results[i].Name < results[j].Name
		}
		if results[i].Required != results[j].Required {
			return results[i].Required && !results[j].Required
		}
		return results[i].Code < results[j].Code
	})
	return results, findings
}

func adapterHealthReadinessFinding(target AdapterHealthTarget, probeResult adapterhealth.Result, adapterStrict bool, runtimeStrict bool) (ReadinessFinding, bool) {
	name := strings.ToLower(strings.TrimSpace(target.Name))
	if name == "" {
		return ReadinessFinding{}, false
	}
	metadata := cloneAnyMap(target.Metadata)
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["adapter"] = name
	metadata["required"] = target.Required
	metadata["health_status"] = string(normalizeAdapterHealthStatus(string(probeResult.Status)))
	metadata["health_code"] = strings.TrimSpace(probeResult.Code)
	if !probeResult.CheckedAt.IsZero() {
		metadata["checked_at"] = probeResult.CheckedAt.UTC().Format(time.RFC3339Nano)
	}
	for key, value := range probeResult.Metadata {
		if key == "cache_hit" {
			continue
		}
		metadata[key] = value
	}

	status := normalizeAdapterHealthStatus(string(probeResult.Status))
	switch status {
	case adapterhealth.StatusHealthy:
		return ReadinessFinding{}, false
	case adapterhealth.StatusDegraded:
		return ReadinessFinding{
			Code:     ReadinessCodeAdapterDegraded,
			Domain:   ReadinessDomainAdapter,
			Severity: ReadinessSeverityWarning,
			Message:  fmt.Sprintf("adapter %s is degraded", name),
			Metadata: metadata,
		}, true
	default:
		severity := ReadinessSeverityWarning
		code := ReadinessCodeAdapterOptionalUnavailable
		message := fmt.Sprintf("optional adapter %s is unavailable", name)
		if target.Required {
			code = ReadinessCodeAdapterRequiredUnavailable
			message = fmt.Sprintf("required adapter %s is unavailable", name)
			if adapterStrict || runtimeStrict {
				severity = ReadinessSeverityError
			}
		}
		return ReadinessFinding{
			Code:     code,
			Domain:   ReadinessDomainAdapter,
			Severity: severity,
			Message:  message,
			Metadata: metadata,
		}, true
	}
}

func classifyReadinessStatus(findings []ReadinessFinding) ReadinessStatus {
	status := ReadinessStatusReady
	for i := range findings {
		switch strings.ToLower(strings.TrimSpace(findings[i].Severity)) {
		case ReadinessSeverityError:
			return ReadinessStatusBlocked
		case ReadinessSeverityWarning:
			status = ReadinessStatusDegraded
		}
	}
	return status
}

func canonicalizeReadinessFindings(findings []ReadinessFinding) []ReadinessFinding {
	if len(findings) == 0 {
		return nil
	}
	out := make([]ReadinessFinding, 0, len(findings))
	for i := range findings {
		item := findings[i]
		item.Code = strings.TrimSpace(item.Code)
		item.Domain = strings.ToLower(strings.TrimSpace(item.Domain))
		item.Severity = normalizeReadinessSeverity(item.Severity)
		item.Message = strings.TrimSpace(item.Message)
		item.Metadata = cloneAnyMap(item.Metadata)
		if item.Metadata == nil {
			item.Metadata = map[string]any{}
		}
		if item.Code == "" {
			continue
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		li := readinessSeverityRank(out[i].Severity)
		lj := readinessSeverityRank(out[j].Severity)
		if li != lj {
			return li > lj
		}
		if out[i].Domain != out[j].Domain {
			return out[i].Domain < out[j].Domain
		}
		if out[i].Code != out[j].Code {
			return out[i].Code < out[j].Code
		}
		return out[i].Message < out[j].Message
	})
	return out
}

func normalizeReadinessSeverity(in string) string {
	switch strings.ToLower(strings.TrimSpace(in)) {
	case ReadinessSeverityInfo:
		return ReadinessSeverityInfo
	case ReadinessSeverityWarning:
		return ReadinessSeverityWarning
	case ReadinessSeverityError:
		return ReadinessSeverityError
	default:
		return ReadinessSeverityInfo
	}
}

func readinessSeverityRank(severity string) int {
	switch normalizeReadinessSeverity(severity) {
	case ReadinessSeverityError:
		return 3
	case ReadinessSeverityWarning:
		return 2
	default:
		return 1
	}
}

func (m *Manager) updateAdapterHealthRunnerOptions(cfg AdapterHealthConfig) {
	if m == nil {
		return
	}
	m.adapterHealthMu.Lock()
	defer m.adapterHealthMu.Unlock()
	if m.adapterHealthRunner == nil {
		m.adapterHealthRunner = adapterhealth.NewRunner(adapterhealth.RunnerOptions{
			ProbeTimeout: cfg.ProbeTimeout,
			CacheTTL:     cfg.CacheTTL,
		}, nil)
		return
	}
	m.adapterHealthRunner.UpdateOptions(adapterhealth.RunnerOptions{
		ProbeTimeout: cfg.ProbeTimeout,
		CacheTTL:     cfg.CacheTTL,
	})
}

func (m *Manager) adapterHealthRunnerSnapshot() *adapterhealth.Runner {
	if m == nil {
		return nil
	}
	m.adapterHealthMu.RLock()
	defer m.adapterHealthMu.RUnlock()
	return m.adapterHealthRunner
}

func normalizeAdapterHealthTargets(targets []AdapterHealthTarget) map[string]AdapterHealthTarget {
	if len(targets) == 0 {
		return map[string]AdapterHealthTarget{}
	}
	out := make(map[string]AdapterHealthTarget, len(targets))
	for i := range targets {
		item := targets[i]
		name := strings.ToLower(strings.TrimSpace(item.Name))
		if name == "" || item.Probe == nil {
			continue
		}
		item.Name = name
		item.Metadata = cloneAnyMap(item.Metadata)
		out[name] = item
	}
	return out
}

func normalizeAdapterHealthStatus(in string) adapterhealth.Status {
	switch adapterhealth.Status(strings.ToLower(strings.TrimSpace(in))) {
	case adapterhealth.StatusHealthy:
		return adapterhealth.StatusHealthy
	case adapterhealth.StatusDegraded:
		return adapterhealth.StatusDegraded
	default:
		return adapterhealth.StatusUnavailable
	}
}

func normalizeReadinessAdmissionMode(in string) string {
	if strings.ToLower(strings.TrimSpace(in)) == ReadinessAdmissionModeFailFast {
		return ReadinessAdmissionModeFailFast
	}
	return ReadinessAdmissionModeFailFast
}

func normalizeReadinessAdmissionBlockOn(in string) string {
	if strings.ToLower(strings.TrimSpace(in)) == ReadinessAdmissionBlockOnBlockedOnly {
		return ReadinessAdmissionBlockOnBlockedOnly
	}
	return ReadinessAdmissionBlockOnBlockedOnly
}

func normalizeReadinessAdmissionDegradedPolicy(in string) string {
	switch strings.ToLower(strings.TrimSpace(in)) {
	case ReadinessAdmissionDegradedPolicyFailFast:
		return ReadinessAdmissionDegradedPolicyFailFast
	case ReadinessAdmissionDegradedPolicyAllowAndRecord:
		return ReadinessAdmissionDegradedPolicyAllowAndRecord
	default:
		return ReadinessAdmissionDegradedPolicyAllowAndRecord
	}
}

func cloneReadinessComponentSnapshot(in RuntimeReadinessComponentSnapshot) RuntimeReadinessComponentSnapshot {
	out := in
	out.Scheduler = cloneReadinessComponentState(in.Scheduler)
	out.Mailbox = cloneReadinessComponentState(in.Mailbox)
	out.Recovery = cloneReadinessComponentState(in.Recovery)
	return out
}

func cloneReadinessComponentState(in RuntimeReadinessComponentState) RuntimeReadinessComponentState {
	return RuntimeReadinessComponentState{
		Enabled:           in.Enabled,
		ConfiguredBackend: strings.TrimSpace(in.ConfiguredBackend),
		EffectiveBackend:  strings.TrimSpace(in.EffectiveBackend),
		Fallback:          in.Fallback,
		FallbackReason:    strings.TrimSpace(in.FallbackReason),
		ActivationError:   strings.TrimSpace(in.ActivationError),
	}
}

func cloneAnyMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
