package config

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	adapterhealth "github.com/FelixSeptem/baymax/adapter/health"
	"github.com/FelixSeptem/baymax/core/types"
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
	ReadinessCodeConfigInvalid                 = "runtime.config.invalid"
	ReadinessCodeStrictEscalated               = "runtime.readiness.strict_escalated"
	ReadinessCodeArbitrationVersionUnsupported = "runtime.arbitration.version.unsupported"
	ReadinessCodeArbitrationVersionMismatch    = "runtime.arbitration.version.compatibility_mismatch"
	ReadinessCodeSchedulerFallback             = "scheduler.backend.fallback"
	ReadinessCodeSchedulerActivationError      = "scheduler.backend.activation_failed"
	ReadinessCodeMailboxFallback               = "mailbox.backend.fallback"
	ReadinessCodeMailboxActivationError        = "mailbox.backend.activation_failed"
	ReadinessCodeRecoveryFallback              = "recovery.backend.fallback"
	ReadinessCodeRecoveryActivationError       = "recovery.backend.activation_failed"
	ReadinessCodeRuntimeManagerUnavailable     = "runtime.manager.unavailable"
	ReadinessCodeAdapterRequiredUnavailable    = "adapter.health.required_unavailable"
	ReadinessCodeAdapterOptionalUnavailable    = "adapter.health.optional_unavailable"
	ReadinessCodeAdapterDegraded               = "adapter.health.degraded"
	ReadinessCodeAdapterRequiredCircuitOpen    = "adapter.health.required_circuit_open"
	ReadinessCodeAdapterOptionalCircuitOpen    = "adapter.health.optional_circuit_open"
	ReadinessCodeAdapterHalfOpenDegraded       = "adapter.health.half_open_degraded"
	ReadinessCodeAdapterGovernanceRecovered    = "adapter.health.governance_recovered"
	ReadinessCodeSandboxRequiredUnavailable    = "sandbox.required_unavailable"
	ReadinessCodeSandboxOptionalUnavailable    = "sandbox.optional_unavailable"
	ReadinessCodeSandboxProfileInvalid         = "sandbox.profile_invalid"
	ReadinessCodeSandboxCapabilityMismatch     = "sandbox.capability_mismatch"
	ReadinessCodeSandboxSessionModeUnsupported = "sandbox.session_mode_unsupported"
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
	Status                          ReadinessStatus                 `json:"status"`
	Findings                        []ReadinessFinding              `json:"findings"`
	AdapterHealth                   []AdapterHealthEvaluation       `json:"adapter_health,omitempty"`
	EvaluatedAt                     time.Time                       `json:"evaluated_at"`
	ArbitrationRuleRequestedVersion string                          `json:"runtime_arbitration_rule_requested_version,omitempty"`
	ArbitrationRuleEffectiveVersion string                          `json:"runtime_arbitration_rule_effective_version,omitempty"`
	ArbitrationRuleVersionSource    string                          `json:"runtime_arbitration_rule_version_source,omitempty"`
	ArbitrationRulePolicyAction     string                          `json:"runtime_arbitration_rule_policy_action,omitempty"`
	ArbitrationRuleUnsupportedTotal int                             `json:"runtime_arbitration_rule_unsupported_total,omitempty"`
	ArbitrationRuleMismatchTotal    int                             `json:"runtime_arbitration_rule_mismatch_total,omitempty"`
	arbitrationVersionConfig        RuntimeArbitrationVersionConfig `json:"-"`
}

type ReadinessSummary struct {
	Status                             string   `json:"runtime_readiness_status"`
	FindingTotal                       int      `json:"runtime_readiness_finding_total"`
	BlockingTotal                      int      `json:"runtime_readiness_blocking_total"`
	DegradedTotal                      int      `json:"runtime_readiness_degraded_total"`
	PrimaryDomain                      string   `json:"runtime_primary_domain,omitempty"`
	PrimaryCode                        string   `json:"runtime_readiness_primary_code"`
	PrimarySource                      string   `json:"runtime_primary_source,omitempty"`
	PrimaryConflictTotal               int      `json:"runtime_primary_conflict_total,omitempty"`
	SecondaryReasonCodes               []string `json:"runtime_secondary_reason_codes,omitempty"`
	SecondaryReasonCount               int      `json:"runtime_secondary_reason_count,omitempty"`
	ArbitrationRuleVersion             string   `json:"runtime_arbitration_rule_version,omitempty"`
	ArbitrationRuleRequestedVersion    string   `json:"runtime_arbitration_rule_requested_version,omitempty"`
	ArbitrationRuleEffectiveVersion    string   `json:"runtime_arbitration_rule_effective_version,omitempty"`
	ArbitrationRuleVersionSource       string   `json:"runtime_arbitration_rule_version_source,omitempty"`
	ArbitrationRulePolicyAction        string   `json:"runtime_arbitration_rule_policy_action,omitempty"`
	ArbitrationRuleUnsupportedTotal    int      `json:"runtime_arbitration_rule_unsupported_total,omitempty"`
	ArbitrationRuleMismatchTotal       int      `json:"runtime_arbitration_rule_mismatch_total,omitempty"`
	RemediationHintCode                string   `json:"runtime_remediation_hint_code,omitempty"`
	RemediationHintDomain              string   `json:"runtime_remediation_hint_domain,omitempty"`
	AdapterHealthStatus                string   `json:"adapter_health_status,omitempty"`
	AdapterHealthProbeTotal            int      `json:"adapter_health_probe_total,omitempty"`
	AdapterHealthDegradedTotal         int      `json:"adapter_health_degraded_total,omitempty"`
	AdapterHealthUnavailableTotal      int      `json:"adapter_health_unavailable_total,omitempty"`
	AdapterHealthPrimaryCode           string   `json:"adapter_health_primary_code,omitempty"`
	AdapterHealthBackoffAppliedTotal   int      `json:"adapter_health_backoff_applied_total,omitempty"`
	AdapterHealthCircuitOpenTotal      int      `json:"adapter_health_circuit_open_total,omitempty"`
	AdapterHealthCircuitHalfOpenTotal  int      `json:"adapter_health_circuit_half_open_total,omitempty"`
	AdapterHealthCircuitRecoverTotal   int      `json:"adapter_health_circuit_recover_total,omitempty"`
	AdapterHealthCircuitState          string   `json:"adapter_health_circuit_state,omitempty"`
	AdapterHealthGovernancePrimaryCode string   `json:"adapter_health_governance_primary_code,omitempty"`
}

type ReadinessAdmissionDecision struct {
	Enabled                                  bool                      `json:"enabled"`
	Mode                                     string                    `json:"mode"`
	BlockOn                                  string                    `json:"block_on"`
	DegradedPolicy                           string                    `json:"degraded_policy"`
	Outcome                                  ReadinessAdmissionOutcome `json:"outcome"`
	ReasonCode                               string                    `json:"reason_code"`
	ReadinessStatus                          ReadinessStatus           `json:"readiness_status"`
	ReadinessPrimaryDomain                   string                    `json:"readiness_primary_domain,omitempty"`
	ReadinessPrimaryCode                     string                    `json:"readiness_primary_code,omitempty"`
	ReadinessPrimarySource                   string                    `json:"readiness_primary_source,omitempty"`
	ReadinessSecondaryReasonCodes            []string                  `json:"readiness_secondary_reason_codes,omitempty"`
	ReadinessSecondaryReasonCount            int                       `json:"readiness_secondary_reason_count,omitempty"`
	ReadinessArbitrationRuleVersion          string                    `json:"readiness_arbitration_rule_version,omitempty"`
	ReadinessArbitrationRuleRequestedVersion string                    `json:"readiness_arbitration_rule_requested_version,omitempty"`
	ReadinessArbitrationRuleEffectiveVersion string                    `json:"readiness_arbitration_rule_effective_version,omitempty"`
	ReadinessArbitrationRuleVersionSource    string                    `json:"readiness_arbitration_rule_version_source,omitempty"`
	ReadinessArbitrationRulePolicyAction     string                    `json:"readiness_arbitration_rule_policy_action,omitempty"`
	ReadinessArbitrationRuleUnsupportedTotal int                       `json:"readiness_arbitration_rule_unsupported_total,omitempty"`
	ReadinessArbitrationRuleMismatchTotal    int                       `json:"readiness_arbitration_rule_mismatch_total,omitempty"`
	ReadinessRemediationHintCode             string                    `json:"readiness_remediation_hint_code,omitempty"`
	ReadinessRemediationHintDomain           string                    `json:"readiness_remediation_hint_domain,omitempty"`
	Bypass                                   bool                      `json:"bypass"`
}

type AdapterHealthTarget struct {
	Name     string              `json:"name"`
	Required bool                `json:"required"`
	Probe    adapterhealth.Probe `json:"-"`
	Metadata map[string]any      `json:"metadata,omitempty"`
}

type AdapterHealthEvaluation struct {
	Name                  string         `json:"name"`
	Required              bool           `json:"required"`
	Status                string         `json:"status"`
	Code                  string         `json:"code"`
	Message               string         `json:"message"`
	Metadata              map[string]any `json:"metadata"`
	BackoffAppliedTotal   int            `json:"backoff_applied_total,omitempty"`
	CircuitOpenTotal      int            `json:"circuit_open_total,omitempty"`
	CircuitHalfOpenTotal  int            `json:"circuit_half_open_total,omitempty"`
	CircuitRecoverTotal   int            `json:"circuit_recover_total,omitempty"`
	CircuitState          string         `json:"circuit_state,omitempty"`
	GovernancePrimaryCode string         `json:"governance_primary_code,omitempty"`
	CheckedAt             time.Time      `json:"checked_at"`
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
	return m.ReadinessPreflightWithRequest("")
}

func (m *Manager) ReadinessPreflightWithRequest(requestedRuleVersion string) ReadinessResult {
	evaluatedAt := time.Now().UTC()
	resolvedVersion, versionErr := ResolveArbitrationRuleVersion(DefaultConfig().Runtime.Arbitration.Version, requestedRuleVersion)
	if m == nil {
		result := ReadinessResult{
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
		result.ArbitrationRuleRequestedVersion = strings.TrimSpace(resolvedVersion.RequestedVersion)
		result.ArbitrationRuleEffectiveVersion = strings.TrimSpace(resolvedVersion.EffectiveVersion)
		result.ArbitrationRuleVersionSource = strings.TrimSpace(resolvedVersion.VersionSource)
		result.ArbitrationRulePolicyAction = strings.TrimSpace(resolvedVersion.PolicyAction)
		result.ArbitrationRuleUnsupportedTotal = resolvedVersion.UnsupportedTotal
		result.ArbitrationRuleMismatchTotal = resolvedVersion.MismatchTotal
		if versionErr != nil {
			if finding, ok := readinessFindingForArbitrationVersionError(versionErr, resolvedVersion); ok {
				result.Findings = append(result.Findings, finding)
				result.Findings = canonicalizeReadinessFindings(result.Findings)
			}
		}
		return result
	}

	cfg := m.EffectiveConfig()
	resolvedVersion, versionErr = ResolveArbitrationRuleVersion(cfg.Runtime.Arbitration.Version, requestedRuleVersion)
	if !cfg.Runtime.Readiness.Enabled {
		return ReadinessResult{
			Status:                          ReadinessStatusReady,
			Findings:                        nil,
			EvaluatedAt:                     evaluatedAt,
			ArbitrationRuleRequestedVersion: strings.TrimSpace(resolvedVersion.RequestedVersion),
			ArbitrationRuleEffectiveVersion: strings.TrimSpace(resolvedVersion.EffectiveVersion),
			ArbitrationRuleVersionSource:    strings.TrimSpace(resolvedVersion.VersionSource),
			ArbitrationRulePolicyAction:     strings.TrimSpace(resolvedVersion.PolicyAction),
			ArbitrationRuleUnsupportedTotal: resolvedVersion.UnsupportedTotal,
			ArbitrationRuleMismatchTotal:    resolvedVersion.MismatchTotal,
			arbitrationVersionConfig:        cfg.Runtime.Arbitration.Version,
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
	if versionErr != nil {
		if finding, ok := readinessFindingForArbitrationVersionError(versionErr, resolvedVersion); ok {
			findings = append(findings, finding)
		}
	}

	componentSnapshot := m.ReadinessComponentSnapshot()
	findings = append(findings, componentReadinessFindings("scheduler", componentSnapshot.Scheduler)...)
	findings = append(findings, componentReadinessFindings("mailbox", componentSnapshot.Mailbox)...)
	findings = append(findings, componentReadinessFindings("recovery", componentSnapshot.Recovery)...)
	findings = append(findings, m.sandboxReadinessFindings(cfg)...)
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
		Status:                          status,
		Findings:                        findings,
		AdapterHealth:                   adapterResults,
		EvaluatedAt:                     evaluatedAt,
		ArbitrationRuleRequestedVersion: strings.TrimSpace(resolvedVersion.RequestedVersion),
		ArbitrationRuleEffectiveVersion: strings.TrimSpace(resolvedVersion.EffectiveVersion),
		ArbitrationRuleVersionSource:    strings.TrimSpace(resolvedVersion.VersionSource),
		ArbitrationRulePolicyAction:     strings.TrimSpace(resolvedVersion.PolicyAction),
		ArbitrationRuleUnsupportedTotal: resolvedVersion.UnsupportedTotal,
		ArbitrationRuleMismatchTotal:    resolvedVersion.MismatchTotal,
		arbitrationVersionConfig:        cfg.Runtime.Arbitration.Version,
	}
}

func (m *Manager) EvaluateReadinessAdmission() ReadinessAdmissionDecision {
	return m.EvaluateReadinessAdmissionWithRequest("")
}

func (m *Manager) EvaluateReadinessAdmissionWithRequest(requestedRuleVersion string) ReadinessAdmissionDecision {
	defaultResolved, _ := ResolveArbitrationRuleVersion(DefaultConfig().Runtime.Arbitration.Version, requestedRuleVersion)
	if m == nil {
		hintCode, hintDomain := mustRemediationHintForPrimaryCode(ReadinessAdmissionCodeManagerNotReady)
		return ReadinessAdmissionDecision{
			Enabled:                                  false,
			Mode:                                     ReadinessAdmissionModeFailFast,
			BlockOn:                                  ReadinessAdmissionBlockOnBlockedOnly,
			DegradedPolicy:                           ReadinessAdmissionDegradedPolicyAllowAndRecord,
			Outcome:                                  ReadinessAdmissionOutcomeAllow,
			ReasonCode:                               ReadinessAdmissionCodeManagerNotReady,
			ReadinessStatus:                          ReadinessStatusBlocked,
			ReadinessPrimaryDomain:                   ReadinessDomainRuntime,
			ReadinessPrimaryCode:                     ReadinessAdmissionCodeManagerNotReady,
			ReadinessPrimarySource:                   RuntimePrimarySourceAdmission,
			ReadinessArbitrationRuleVersion:          strings.TrimSpace(defaultResolved.EffectiveVersion),
			ReadinessArbitrationRuleRequestedVersion: strings.TrimSpace(defaultResolved.RequestedVersion),
			ReadinessArbitrationRuleEffectiveVersion: strings.TrimSpace(defaultResolved.EffectiveVersion),
			ReadinessArbitrationRuleVersionSource:    strings.TrimSpace(defaultResolved.VersionSource),
			ReadinessArbitrationRulePolicyAction:     strings.TrimSpace(defaultResolved.PolicyAction),
			ReadinessArbitrationRuleUnsupportedTotal: defaultResolved.UnsupportedTotal,
			ReadinessArbitrationRuleMismatchTotal:    defaultResolved.MismatchTotal,
			ReadinessRemediationHintCode:             hintCode,
			ReadinessRemediationHintDomain:           hintDomain,
			Bypass:                                   true,
		}
	}

	runtimeCfg := m.EffectiveConfig().Runtime
	resolvedVersion, _ := ResolveArbitrationRuleVersion(runtimeCfg.Arbitration.Version, requestedRuleVersion)
	cfg := runtimeCfg.Readiness.Admission
	hintCode, hintDomain := mustRemediationHintForPrimaryCode(ReadinessAdmissionCodeBypassDisabled)
	decision := ReadinessAdmissionDecision{
		Enabled:                                  cfg.Enabled,
		Mode:                                     normalizeReadinessAdmissionMode(cfg.Mode),
		BlockOn:                                  normalizeReadinessAdmissionBlockOn(cfg.BlockOn),
		DegradedPolicy:                           normalizeReadinessAdmissionDegradedPolicy(cfg.DegradedPolicy),
		Outcome:                                  ReadinessAdmissionOutcomeAllow,
		ReasonCode:                               ReadinessAdmissionCodeBypassDisabled,
		ReadinessStatus:                          ReadinessStatusReady,
		ReadinessPrimaryDomain:                   ReadinessDomainRuntime,
		ReadinessPrimaryCode:                     ReadinessAdmissionCodeBypassDisabled,
		ReadinessPrimarySource:                   RuntimePrimarySourceAdmission,
		ReadinessArbitrationRuleVersion:          strings.TrimSpace(resolvedVersion.EffectiveVersion),
		ReadinessArbitrationRuleRequestedVersion: strings.TrimSpace(resolvedVersion.RequestedVersion),
		ReadinessArbitrationRuleEffectiveVersion: strings.TrimSpace(resolvedVersion.EffectiveVersion),
		ReadinessArbitrationRuleVersionSource:    strings.TrimSpace(resolvedVersion.VersionSource),
		ReadinessArbitrationRulePolicyAction:     strings.TrimSpace(resolvedVersion.PolicyAction),
		ReadinessArbitrationRuleUnsupportedTotal: resolvedVersion.UnsupportedTotal,
		ReadinessArbitrationRuleMismatchTotal:    resolvedVersion.MismatchTotal,
		ReadinessRemediationHintCode:             hintCode,
		ReadinessRemediationHintDomain:           hintDomain,
		Bypass:                                   true,
	}
	if !decision.Enabled {
		return decision
	}

	preflight := m.ReadinessPreflightWithRequest(requestedRuleVersion)
	summary := preflight.Summary()
	decision.Bypass = false
	decision.ReadinessStatus = preflight.Status
	decision.ReadinessPrimaryDomain = strings.TrimSpace(summary.PrimaryDomain)
	decision.ReadinessPrimaryCode = strings.TrimSpace(summary.PrimaryCode)
	decision.ReadinessPrimarySource = strings.TrimSpace(summary.PrimarySource)
	decision.ReadinessSecondaryReasonCodes = cloneStringSlice(summary.SecondaryReasonCodes)
	decision.ReadinessSecondaryReasonCount = summary.SecondaryReasonCount
	decision.ReadinessArbitrationRuleVersion = strings.TrimSpace(summary.ArbitrationRuleVersion)
	decision.ReadinessArbitrationRuleRequestedVersion = strings.TrimSpace(summary.ArbitrationRuleRequestedVersion)
	decision.ReadinessArbitrationRuleEffectiveVersion = strings.TrimSpace(summary.ArbitrationRuleEffectiveVersion)
	decision.ReadinessArbitrationRuleVersionSource = strings.TrimSpace(summary.ArbitrationRuleVersionSource)
	decision.ReadinessArbitrationRulePolicyAction = strings.TrimSpace(summary.ArbitrationRulePolicyAction)
	decision.ReadinessArbitrationRuleUnsupportedTotal = summary.ArbitrationRuleUnsupportedTotal
	decision.ReadinessArbitrationRuleMismatchTotal = summary.ArbitrationRuleMismatchTotal
	decision.ReadinessRemediationHintCode = strings.TrimSpace(summary.RemediationHintCode)
	decision.ReadinessRemediationHintDomain = strings.TrimSpace(summary.RemediationHintDomain)
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
	if decision.ReadinessPrimaryDomain == "" {
		decision.ReadinessPrimaryDomain = ReadinessDomainRuntime
	}
	if decision.ReadinessPrimarySource == "" {
		decision.ReadinessPrimarySource = RuntimePrimarySourceAdmission
	}
	if decision.ReadinessArbitrationRuleVersion == "" {
		decision.ReadinessArbitrationRuleVersion = strings.TrimSpace(resolvedVersion.EffectiveVersion)
	}
	if decision.ReadinessArbitrationRuleEffectiveVersion == "" {
		decision.ReadinessArbitrationRuleEffectiveVersion = strings.TrimSpace(decision.ReadinessArbitrationRuleVersion)
	}
	if decision.ReadinessArbitrationRulePolicyAction == "" {
		decision.ReadinessArbitrationRulePolicyAction = RuntimeArbitrationPolicyActionNone
	}
	if decision.Outcome == ReadinessAdmissionOutcomeDeny && isArbitrationVersionFindingCode(decision.ReadinessPrimaryCode) {
		decision.ReasonCode = strings.TrimSpace(decision.ReadinessPrimaryCode)
	}
	if decision.ReadinessRemediationHintCode == "" && decision.ReadinessPrimaryCode != "" {
		hintCode, hintDomain := mustRemediationHintForPrimaryCode(decision.ReadinessPrimaryCode)
		decision.ReadinessRemediationHintCode = hintCode
		decision.ReadinessRemediationHintDomain = hintDomain
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
		case ReadinessSeverityWarning:
			summary.DegradedTotal++
		}
	}
	primary := ArbitratePrimaryReason(PrimaryReasonArbitrationInput{
		ReadinessFindings:    r.Findings,
		RequestedRuleVersion: r.ArbitrationRuleRequestedVersion,
		VersionConfig:        r.arbitrationVersionConfig,
	})
	summary.PrimaryDomain = strings.TrimSpace(primary.Domain)
	summary.PrimaryCode = strings.TrimSpace(primary.Code)
	summary.PrimarySource = strings.TrimSpace(primary.Source)
	summary.PrimaryConflictTotal = primary.ConflictTotal
	summary.SecondaryReasonCodes = cloneStringSlice(primary.SecondaryCodes)
	summary.SecondaryReasonCount = primary.SecondaryCount
	summary.ArbitrationRuleVersion = strings.TrimSpace(primary.RuleVersion)
	summary.ArbitrationRuleRequestedVersion = strings.TrimSpace(primary.RuleRequestedVersion)
	summary.ArbitrationRuleEffectiveVersion = strings.TrimSpace(primary.RuleEffectiveVersion)
	summary.ArbitrationRuleVersionSource = strings.TrimSpace(primary.RuleVersionSource)
	summary.ArbitrationRulePolicyAction = strings.TrimSpace(primary.RulePolicyAction)
	summary.ArbitrationRuleUnsupportedTotal = primary.RuleUnsupportedTotal
	summary.ArbitrationRuleMismatchTotal = primary.RuleMismatchTotal
	summary.RemediationHintCode = strings.TrimSpace(primary.RemediationHintCode)
	summary.RemediationHintDomain = strings.TrimSpace(primary.RemediationHintDomain)
	if summary.ArbitrationRuleVersion == "" {
		summary.ArbitrationRuleVersion = strings.TrimSpace(r.ArbitrationRuleEffectiveVersion)
	}
	if summary.ArbitrationRuleRequestedVersion == "" {
		summary.ArbitrationRuleRequestedVersion = strings.TrimSpace(r.ArbitrationRuleRequestedVersion)
	}
	if summary.ArbitrationRuleEffectiveVersion == "" {
		summary.ArbitrationRuleEffectiveVersion = strings.TrimSpace(r.ArbitrationRuleEffectiveVersion)
	}
	if summary.ArbitrationRuleVersionSource == "" {
		summary.ArbitrationRuleVersionSource = strings.TrimSpace(r.ArbitrationRuleVersionSource)
	}
	if summary.ArbitrationRulePolicyAction == "" {
		summary.ArbitrationRulePolicyAction = strings.TrimSpace(r.ArbitrationRulePolicyAction)
	}
	if summary.ArbitrationRuleUnsupportedTotal == 0 {
		summary.ArbitrationRuleUnsupportedTotal = r.ArbitrationRuleUnsupportedTotal
	}
	if summary.ArbitrationRuleMismatchTotal == 0 {
		summary.ArbitrationRuleMismatchTotal = r.ArbitrationRuleMismatchTotal
	}
	if strings.TrimSpace(summary.Status) == "" {
		summary.Status = string(ReadinessStatusReady)
	}
	if len(r.AdapterHealth) > 0 {
		adapterStatus := string(adapterhealth.StatusHealthy)
		primaryRank := -1
		circuitRank := -1
		governanceRank := -1
		for i := range r.AdapterHealth {
			item := r.AdapterHealth[i]
			status := normalizeAdapterHealthStatus(item.Status)
			summary.AdapterHealthProbeTotal++
			summary.AdapterHealthBackoffAppliedTotal += item.BackoffAppliedTotal
			summary.AdapterHealthCircuitOpenTotal += item.CircuitOpenTotal
			summary.AdapterHealthCircuitHalfOpenTotal += item.CircuitHalfOpenTotal
			summary.AdapterHealthCircuitRecoverTotal += item.CircuitRecoverTotal
			switch strings.ToLower(strings.TrimSpace(item.CircuitState)) {
			case string(adapterhealth.CircuitStateOpen):
				if circuitRank < 2 {
					summary.AdapterHealthCircuitState = string(adapterhealth.CircuitStateOpen)
					circuitRank = 2
				}
			case string(adapterhealth.CircuitStateHalfOpen):
				if circuitRank < 1 {
					summary.AdapterHealthCircuitState = string(adapterhealth.CircuitStateHalfOpen)
					circuitRank = 1
				}
			case string(adapterhealth.CircuitStateClosed):
				if circuitRank < 0 {
					summary.AdapterHealthCircuitState = string(adapterhealth.CircuitStateClosed)
					circuitRank = 0
				}
			}
			if code := strings.TrimSpace(item.GovernancePrimaryCode); code != "" {
				rank := 0
				switch strings.ToLower(strings.TrimSpace(item.CircuitState)) {
				case string(adapterhealth.CircuitStateOpen):
					rank = 2
				case string(adapterhealth.CircuitStateHalfOpen):
					rank = 1
				}
				if rank >= governanceRank {
					summary.AdapterHealthGovernancePrimaryCode = code
					governanceRank = rank
				}
			}
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

func (m *Manager) sandboxReadinessFindings(cfg Config) []ReadinessFinding {
	if m == nil || !cfg.Security.Sandbox.Enabled {
		return nil
	}
	sandboxCfg := cfg.Security.Sandbox
	severity := ReadinessSeverityWarning
	if sandboxCfg.Required {
		severity = ReadinessSeverityError
	}
	metadataBase := map[string]any{
		"sandbox_enabled":      true,
		"sandbox_required":     sandboxCfg.Required,
		"sandbox_mode":         strings.ToLower(strings.TrimSpace(sandboxCfg.Mode)),
		"sandbox_backend":      strings.ToLower(strings.TrimSpace(sandboxCfg.Executor.Backend)),
		"sandbox_session_mode": strings.ToLower(strings.TrimSpace(sandboxCfg.Executor.SessionMode)),
	}
	selectedProfile := ResolveSandboxProfile(sandboxCfg, "")
	metadataBase["sandbox_profile"] = selectedProfile
	findings := make([]ReadinessFinding, 0, 3)

	profile, ok := sandboxCfg.Profiles[selectedProfile]
	if !ok {
		findings = append(findings, ReadinessFinding{
			Code:     ReadinessCodeSandboxProfileInvalid,
			Domain:   ReadinessDomainRuntime,
			Severity: severity,
			Message:  fmt.Sprintf("sandbox profile %q is not configured", selectedProfile),
			Metadata: metadataBase,
		})
		return findings
	}

	executor := m.SandboxExecutor()
	if executor == nil {
		code := ReadinessCodeSandboxOptionalUnavailable
		message := "sandbox executor is unavailable but sandbox is optional"
		if sandboxCfg.Required {
			code = ReadinessCodeSandboxRequiredUnavailable
			message = "sandbox executor is unavailable while required=true"
		}
		findings = append(findings, ReadinessFinding{
			Code:     code,
			Domain:   ReadinessDomainRuntime,
			Severity: severity,
			Message:  message,
			Metadata: metadataBase,
		})
		return findings
	}

	probeTimeout := profile.Timeouts.LaunchTimeout
	if probeTimeout <= 0 {
		probeTimeout = 1500 * time.Millisecond
	}
	probeCtx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	probe, err := executor.Probe(probeCtx)
	cancel()
	if err != nil {
		metadata := cloneAnyMap(metadataBase)
		metadata["probe_error"] = strings.TrimSpace(err.Error())
		metadata["probe_timeout_ms"] = probeTimeout.Milliseconds()
		code := ReadinessCodeSandboxOptionalUnavailable
		message := "sandbox executor probe failed but sandbox is optional"
		if sandboxCfg.Required {
			code = ReadinessCodeSandboxRequiredUnavailable
			message = "sandbox executor probe failed while required=true"
		}
		findings = append(findings, ReadinessFinding{
			Code:     code,
			Domain:   ReadinessDomainRuntime,
			Severity: severity,
			Message:  message,
			Metadata: metadata,
		})
		return findings
	}

	probeBackend := strings.ToLower(strings.TrimSpace(probe.Backend))
	missingCapabilities := make([]string, 0)
	for i := range sandboxCfg.Executor.RequiredCapabilities {
		capability := strings.ToLower(strings.TrimSpace(sandboxCfg.Executor.RequiredCapabilities[i]))
		if capability == "" {
			continue
		}
		if !probe.Supports(capability) {
			missingCapabilities = append(missingCapabilities, capability)
		}
	}
	if len(missingCapabilities) > 0 {
		metadata := cloneAnyMap(metadataBase)
		metadata["required_capabilities"] = append([]string(nil), sandboxCfg.Executor.RequiredCapabilities...)
		metadata["missing_capabilities"] = append([]string(nil), missingCapabilities...)
		if probeBackend != "" {
			metadata["probe_backend"] = probeBackend
		}
		findings = append(findings, ReadinessFinding{
			Code:     ReadinessCodeSandboxCapabilityMismatch,
			Domain:   ReadinessDomainRuntime,
			Severity: severity,
			Message:  "sandbox executor capabilities do not satisfy required_capabilities",
			Metadata: metadata,
		})
	}

	sessionMode := types.SandboxSessionMode(strings.ToLower(strings.TrimSpace(sandboxCfg.Executor.SessionMode)))
	if sessionMode != "" && !probe.SupportsSessionMode(sessionMode) {
		metadata := cloneAnyMap(metadataBase)
		metadata["supported_session_modes"] = append([]string(nil), probe.SupportedModes...)
		if probeBackend != "" {
			metadata["probe_backend"] = probeBackend
		}
		findings = append(findings, ReadinessFinding{
			Code:     ReadinessCodeSandboxSessionModeUnsupported,
			Domain:   ReadinessDomainRuntime,
			Severity: severity,
			Message:  "sandbox executor does not support configured session_mode",
			Metadata: metadata,
		})
	}

	return findings
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
			Name:                  strings.ToLower(strings.TrimSpace(target.Name)),
			Required:              target.Required,
			Status:                string(normalizeAdapterHealthStatus(string(probeResult.Status))),
			Code:                  strings.TrimSpace(probeResult.Code),
			Message:               strings.TrimSpace(probeResult.Message),
			Metadata:              cloneAnyMap(probeResult.Metadata),
			BackoffAppliedTotal:   probeResult.Governance.BackoffAppliedTotal,
			CircuitOpenTotal:      probeResult.Governance.CircuitOpenTotal,
			CircuitHalfOpenTotal:  probeResult.Governance.CircuitHalfOpenTotal,
			CircuitRecoverTotal:   probeResult.Governance.CircuitRecoverTotal,
			CircuitState:          strings.ToLower(strings.TrimSpace(probeResult.Governance.CircuitState)),
			GovernancePrimaryCode: strings.TrimSpace(probeResult.Governance.PrimaryCode),
			CheckedAt:             probeResult.CheckedAt.UTC(),
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
	if backoff := probeResult.Governance.BackoffAppliedTotal; backoff > 0 {
		metadata["governance_backoff_applied_total"] = backoff
	}
	if openTotal := probeResult.Governance.CircuitOpenTotal; openTotal > 0 {
		metadata["governance_circuit_open_total"] = openTotal
	}
	if halfOpenTotal := probeResult.Governance.CircuitHalfOpenTotal; halfOpenTotal > 0 {
		metadata["governance_circuit_half_open_total"] = halfOpenTotal
	}
	if recoverTotal := probeResult.Governance.CircuitRecoverTotal; recoverTotal > 0 {
		metadata["governance_circuit_recover_total"] = recoverTotal
	}
	if circuitState := strings.ToLower(strings.TrimSpace(probeResult.Governance.CircuitState)); circuitState != "" {
		metadata["governance_circuit_state"] = circuitState
	}
	if code := strings.TrimSpace(probeResult.Governance.PrimaryCode); code != "" {
		metadata["governance_primary_code"] = code
	}
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
	circuitState := strings.ToLower(strings.TrimSpace(probeResult.Governance.CircuitState))
	switch status {
	case adapterhealth.StatusHealthy:
		if strings.TrimSpace(probeResult.Governance.PrimaryCode) == adapterhealth.CodeCircuitRecover {
			return ReadinessFinding{
				Code:     ReadinessCodeAdapterGovernanceRecovered,
				Domain:   ReadinessDomainAdapter,
				Severity: ReadinessSeverityInfo,
				Message:  fmt.Sprintf("adapter %s recovered after governance half-open probes", name),
				Metadata: metadata,
			}, true
		}
		return ReadinessFinding{}, false
	case adapterhealth.StatusDegraded:
		code := ReadinessCodeAdapterDegraded
		message := fmt.Sprintf("adapter %s is degraded", name)
		if circuitState == string(adapterhealth.CircuitStateHalfOpen) {
			code = ReadinessCodeAdapterHalfOpenDegraded
			message = fmt.Sprintf("adapter %s half-open probe is degraded", name)
		}
		return ReadinessFinding{
			Code:     code,
			Domain:   ReadinessDomainAdapter,
			Severity: ReadinessSeverityWarning,
			Message:  message,
			Metadata: metadata,
		}, true
	default:
		severity := ReadinessSeverityWarning
		code := ReadinessCodeAdapterOptionalUnavailable
		message := fmt.Sprintf("optional adapter %s is unavailable", name)
		if circuitState == string(adapterhealth.CircuitStateOpen) {
			code = ReadinessCodeAdapterOptionalCircuitOpen
			message = fmt.Sprintf("optional adapter %s is unavailable while circuit is open", name)
		}
		if target.Required {
			code = ReadinessCodeAdapterRequiredUnavailable
			message = fmt.Sprintf("required adapter %s is unavailable", name)
			if circuitState == string(adapterhealth.CircuitStateOpen) {
				code = ReadinessCodeAdapterRequiredCircuitOpen
				message = fmt.Sprintf("required adapter %s is unavailable while circuit is open", name)
			}
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

func readinessFindingForArbitrationVersionError(err error, resolved ArbitrationRuleVersionResolution) (ReadinessFinding, bool) {
	typed, ok := err.(*ArbitrationRuleVersionError)
	if !ok {
		return ReadinessFinding{}, false
	}
	code := ReadinessCodeArbitrationVersionUnsupported
	message := "requested arbitration rule version is unsupported"
	if typed.Code == ArbitrationRuleVersionErrorMismatch {
		code = ReadinessCodeArbitrationVersionMismatch
		message = "requested arbitration rule version mismatches compatibility window"
	}
	if detail := strings.TrimSpace(typed.Message); detail != "" {
		message = detail
	}
	metadata := map[string]any{
		"runtime_arbitration_rule_requested_version": strings.TrimSpace(resolved.RequestedVersion),
		"runtime_arbitration_rule_effective_version": strings.TrimSpace(resolved.EffectiveVersion),
		"runtime_arbitration_rule_version_source":    strings.TrimSpace(resolved.VersionSource),
		"runtime_arbitration_rule_policy_action":     strings.TrimSpace(resolved.PolicyAction),
		"runtime_arbitration_rule_unsupported_total": resolved.UnsupportedTotal,
		"runtime_arbitration_rule_mismatch_total":    resolved.MismatchTotal,
	}
	return ReadinessFinding{
		Code:     code,
		Domain:   ReadinessDomainRuntime,
		Severity: ReadinessSeverityError,
		Message:  message,
		Metadata: metadata,
	}, true
}

func isArbitrationVersionFindingCode(code string) bool {
	switch strings.TrimSpace(code) {
	case ReadinessCodeArbitrationVersionUnsupported, ReadinessCodeArbitrationVersionMismatch:
		return true
	default:
		return false
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
			Backoff: adapterhealth.BackoffOptions{
				Enabled:     cfg.Backoff.Enabled,
				Initial:     cfg.Backoff.Initial,
				Max:         cfg.Backoff.Max,
				Multiplier:  cfg.Backoff.Multiplier,
				JitterRatio: cfg.Backoff.JitterRatio,
			},
			Circuit: adapterhealth.CircuitOptions{
				Enabled:                  cfg.Circuit.Enabled,
				FailureThreshold:         cfg.Circuit.FailureThreshold,
				OpenDuration:             cfg.Circuit.OpenDuration,
				HalfOpenMaxProbe:         cfg.Circuit.HalfOpenMaxProbe,
				HalfOpenSuccessThreshold: cfg.Circuit.HalfOpenSuccessThreshold,
			},
		}, nil)
		return
	}
	m.adapterHealthRunner.UpdateOptions(adapterhealth.RunnerOptions{
		ProbeTimeout: cfg.ProbeTimeout,
		CacheTTL:     cfg.CacheTTL,
		Backoff: adapterhealth.BackoffOptions{
			Enabled:     cfg.Backoff.Enabled,
			Initial:     cfg.Backoff.Initial,
			Max:         cfg.Backoff.Max,
			Multiplier:  cfg.Backoff.Multiplier,
			JitterRatio: cfg.Backoff.JitterRatio,
		},
		Circuit: adapterhealth.CircuitOptions{
			Enabled:                  cfg.Circuit.Enabled,
			FailureThreshold:         cfg.Circuit.FailureThreshold,
			OpenDuration:             cfg.Circuit.OpenDuration,
			HalfOpenMaxProbe:         cfg.Circuit.HalfOpenMaxProbe,
			HalfOpenSuccessThreshold: cfg.Circuit.HalfOpenSuccessThreshold,
		},
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

func cloneStringSlice(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for i := range in {
		item := strings.TrimSpace(in[i])
		if item == "" {
			continue
		}
		out = append(out, item)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
