package config

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"runtime"
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
	ReadinessCodeConfigInvalid                        = "runtime.config.invalid"
	ReadinessCodeStrictEscalated                      = "runtime.readiness.strict_escalated"
	ReadinessCodeReactLoopDisabled                    = "react.loop_disabled"
	ReadinessCodeReactStreamDispatchUnavailable       = "react.stream_dispatch_unavailable"
	ReadinessCodeReactProviderToolCallingUnsupported  = "react.provider_tool_calling_unsupported"
	ReadinessCodeReactToolRegistryUnavailable         = "react.tool_registry_unavailable"
	ReadinessCodeReactSandboxDependencyUnavailable    = "react.sandbox_dependency_unavailable"
	ReadinessCodeArbitrationVersionUnsupported        = "runtime.arbitration.version.unsupported"
	ReadinessCodeArbitrationVersionMismatch           = "runtime.arbitration.version.compatibility_mismatch"
	ReadinessCodeSchedulerFallback                    = "scheduler.backend.fallback"
	ReadinessCodeSchedulerActivationError             = "scheduler.backend.activation_failed"
	ReadinessCodeMailboxFallback                      = "mailbox.backend.fallback"
	ReadinessCodeMailboxActivationError               = "mailbox.backend.activation_failed"
	ReadinessCodeRecoveryFallback                     = "recovery.backend.fallback"
	ReadinessCodeRecoveryActivationError              = "recovery.backend.activation_failed"
	ReadinessCodeRuntimeManagerUnavailable            = "runtime.manager.unavailable"
	ReadinessCodeAdapterRequiredUnavailable           = "adapter.health.required_unavailable"
	ReadinessCodeAdapterOptionalUnavailable           = "adapter.health.optional_unavailable"
	ReadinessCodeAdapterDegraded                      = "adapter.health.degraded"
	ReadinessCodeAdapterRequiredCircuitOpen           = "adapter.health.required_circuit_open"
	ReadinessCodeAdapterOptionalCircuitOpen           = "adapter.health.optional_circuit_open"
	ReadinessCodeAdapterHalfOpenDegraded              = "adapter.health.half_open_degraded"
	ReadinessCodeAdapterGovernanceRecovered           = "adapter.health.governance_recovered"
	ReadinessCodeSandboxRequiredUnavailable           = "sandbox.required_unavailable"
	ReadinessCodeSandboxOptionalUnavailable           = "sandbox.optional_unavailable"
	ReadinessCodeSandboxProfileInvalid                = "sandbox.profile_invalid"
	ReadinessCodeSandboxCapabilityMismatch            = "sandbox.capability_mismatch"
	ReadinessCodeSandboxSessionModeUnsupported        = "sandbox.session_mode_unsupported"
	ReadinessCodeSandboxAdapterProfileMissing         = "sandbox.adapter.profile_missing"
	ReadinessCodeSandboxAdapterBackendNotSupported    = "sandbox.adapter.backend_not_supported"
	ReadinessCodeSandboxAdapterHostMismatch           = "sandbox.adapter.host_mismatch"
	ReadinessCodeSandboxAdapterSessionModeUnsupported = "sandbox.adapter.session_mode_unsupported"
	ReadinessCodeSandboxRolloutPhaseInvalid           = "sandbox.rollout.phase_invalid"
	ReadinessCodeSandboxRolloutHealthBreached         = "sandbox.rollout.health_budget_breached"
	ReadinessCodeSandboxRolloutFrozen                 = "sandbox.rollout.frozen"
	ReadinessCodeSandboxRolloutCapacityBlocked        = "sandbox.rollout.capacity_unavailable"
	ReadinessCodeSandboxEgressPolicyInvalid           = "sandbox.egress.policy_invalid"
	ReadinessCodeSandboxEgressAllowlistInvalid        = "sandbox.egress.allowlist_invalid"
	ReadinessCodeSandboxEgressRuleConflict            = "sandbox.egress.rule_conflict"
	ReadinessCodeSandboxEgressViolationBudgetBreached = "sandbox.egress.violation_budget_breached"
	ReadinessCodeAdapterAllowlistMissingEntry         = "adapter.allowlist.missing_entry"
	ReadinessCodeAdapterAllowlistSignatureInvalid     = "adapter.allowlist.signature_invalid"
	ReadinessCodeAdapterAllowlistPolicyConflict       = "adapter.allowlist.policy_conflict"
	ReadinessCodeMemoryModeInvalid                    = "memory.mode_invalid"
	ReadinessCodeMemoryProfileMissing                 = "memory.profile_missing"
	ReadinessCodeMemoryProviderNotSupported           = "memory.provider_not_supported"
	ReadinessCodeMemorySPIUnavailable                 = "memory.spi_unavailable"
	ReadinessCodeMemoryFilesystemPathInvalid          = "memory.filesystem_path_invalid"
	ReadinessCodeMemoryContractVersionMismatch        = "memory.contract_version_mismatch"
	ReadinessCodeMemoryFallbackPolicyConflict         = "memory.fallback_policy_conflict"
	ReadinessCodeMemoryFallbackTargetUnavailable      = "memory.fallback_target_unavailable"
	ReadinessCodeObservabilityExportProfileInvalid    = "observability.export.profile_invalid"
	ReadinessCodeObservabilityExportSinkUnavailable   = "observability.export.sink_unavailable"
	ReadinessCodeObservabilityExportAuthInvalid       = "observability.export.auth_invalid"
	ReadinessCodeDiagnosticsBundleOutputUnavailable   = "diagnostics.bundle.output_unavailable"
	ReadinessCodeDiagnosticsBundlePolicyInvalid       = "diagnostics.bundle.policy_invalid"
)

type ReadinessAdmissionOutcome string

const (
	ReadinessAdmissionOutcomeAllow ReadinessAdmissionOutcome = "allow"
	ReadinessAdmissionOutcomeDeny  ReadinessAdmissionOutcome = "deny"
)

const (
	ReadinessAdmissionCodeBypassDisabled       = "runtime.readiness.admission.disabled"
	ReadinessAdmissionCodeReady                = "runtime.readiness.admission.ready"
	ReadinessAdmissionCodeBlocked              = "runtime.readiness.admission.blocked"
	ReadinessAdmissionCodeDegradedAllow        = "runtime.readiness.admission.degraded_allow"
	ReadinessAdmissionCodeDegradedDeny         = "runtime.readiness.admission.degraded_fail_fast"
	ReadinessAdmissionCodeSandboxFrozen        = "runtime.readiness.admission.sandbox_rollout_frozen"
	ReadinessAdmissionCodeSandboxThrottle      = "runtime.readiness.admission.sandbox_capacity_throttle_allow"
	ReadinessAdmissionCodeSandboxThrottledDeny = "runtime.readiness.admission.sandbox_capacity_throttle_fail_fast"
	ReadinessAdmissionCodeSandboxCapacityDeny  = "runtime.readiness.admission.sandbox_capacity_deny"
	ReadinessAdmissionCodeUnknownStatus        = "runtime.readiness.admission.unknown_status"
	ReadinessAdmissionCodeManagerNotReady      = "runtime.readiness.admission.manager_unavailable"
)

const (
	SandboxHealthBudgetWithinBudget = "within_budget"
	SandboxHealthBudgetNearBudget   = "near_budget"
	SandboxHealthBudgetBreached     = "breached"
)

const (
	SandboxCapacityActionAllow    = "allow"
	SandboxCapacityActionThrottle = "throttle"
	SandboxCapacityActionDeny     = "deny"
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
	SandboxRolloutPhase                      string                    `json:"sandbox_rollout_phase,omitempty"`
	SandboxCapacityAction                    string                    `json:"sandbox_capacity_action,omitempty"`
	SandboxCapacityDegradedPolicy            string                    `json:"sandbox_capacity_degraded_policy,omitempty"`
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

type ReactReadinessDependencySnapshot struct {
	ToolRegistryChecked          bool      `json:"tool_registry_checked"`
	ToolRegistryAvailable        bool      `json:"tool_registry_available"`
	ToolRegistryReason           string    `json:"tool_registry_reason,omitempty"`
	ProviderChecked              bool      `json:"provider_checked"`
	ProviderName                 string    `json:"provider_name,omitempty"`
	ProviderToolCallingSupported bool      `json:"provider_tool_calling_supported"`
	ProviderReason               string    `json:"provider_reason,omitempty"`
	SandboxDependencyChecked     bool      `json:"sandbox_dependency_checked"`
	SandboxDependencyAvailable   bool      `json:"sandbox_dependency_available"`
	SandboxDependencyReason      string    `json:"sandbox_dependency_reason,omitempty"`
	UpdatedAt                    time.Time `json:"updated_at,omitempty"`
}

type SandboxRolloutRuntimeState struct {
	HealthBudgetStatus      string    `json:"health_budget_status,omitempty"`
	HealthBudgetBreachTotal int       `json:"health_budget_breach_total,omitempty"`
	EgressViolationTotal    int       `json:"egress_violation_total,omitempty"`
	EgressViolationBudget   int       `json:"egress_violation_budget,omitempty"`
	EgressBudgetBreached    bool      `json:"egress_budget_breached,omitempty"`
	FreezeState             bool      `json:"freeze_state,omitempty"`
	FreezeReasonCode        string    `json:"freeze_reason_code,omitempty"`
	CapacityQueueDepth      int       `json:"capacity_queue_depth,omitempty"`
	CapacityInflight        int       `json:"capacity_inflight,omitempty"`
	CapacityAction          string    `json:"capacity_action,omitempty"`
	UpdatedAt               time.Time `json:"updated_at,omitempty"`
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

func (m *Manager) SetReactReadinessDependencySnapshot(snapshot ReactReadinessDependencySnapshot) {
	if m == nil {
		return
	}
	m.readinessMu.Lock()
	m.reactReadiness = cloneReactReadinessDependencySnapshot(snapshot)
	m.readinessMu.Unlock()
}

func (m *Manager) ReactReadinessDependencySnapshot() ReactReadinessDependencySnapshot {
	if m == nil {
		return ReactReadinessDependencySnapshot{}
	}
	m.readinessMu.RLock()
	defer m.readinessMu.RUnlock()
	return cloneReactReadinessDependencySnapshot(m.reactReadiness)
}

func (m *Manager) SetSandboxRolloutRuntimeState(state SandboxRolloutRuntimeState) {
	if m == nil {
		return
	}
	m.sandboxRolloutMu.Lock()
	m.sandboxRolloutState = cloneSandboxRolloutRuntimeState(state)
	m.sandboxRolloutMu.Unlock()
}

func (m *Manager) SandboxRolloutRuntimeState() SandboxRolloutRuntimeState {
	if m == nil {
		return SandboxRolloutRuntimeState{}
	}
	m.sandboxRolloutMu.RLock()
	defer m.sandboxRolloutMu.RUnlock()
	return cloneSandboxRolloutRuntimeState(m.sandboxRolloutState)
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
	reactSnapshot := m.ReactReadinessDependencySnapshot()
	findings = append(findings, componentReadinessFindings("scheduler", componentSnapshot.Scheduler)...)
	findings = append(findings, componentReadinessFindings("mailbox", componentSnapshot.Mailbox)...)
	findings = append(findings, componentReadinessFindings("recovery", componentSnapshot.Recovery)...)
	findings = append(findings, reactReadinessFindings(cfg, reactSnapshot)...)
	findings = append(findings, memoryReadinessFindings(cfg)...)
	findings = append(findings, observabilityReadinessFindings(cfg)...)
	findings = append(findings, m.sandboxReadinessFindings(cfg)...)
	findings = append(findings, m.adapterAllowlistReadinessFindings(cfg)...)
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

	effectiveCfg := m.EffectiveConfig()
	runtimeCfg := effectiveCfg.Runtime
	resolvedVersion, _ := ResolveArbitrationRuleVersion(runtimeCfg.Arbitration.Version, requestedRuleVersion)
	cfg := runtimeCfg.Readiness.Admission
	capacityDegradedPolicy := normalizeSandboxCapacityDegradedPolicy(effectiveCfg.Security.Sandbox.Capacity.DegradedPolicy)
	hintCode, hintDomain := mustRemediationHintForPrimaryCode(ReadinessAdmissionCodeBypassDisabled)
	decision := ReadinessAdmissionDecision{
		Enabled:                                  cfg.Enabled,
		Mode:                                     normalizeReadinessAdmissionMode(cfg.Mode),
		BlockOn:                                  normalizeReadinessAdmissionBlockOn(cfg.BlockOn),
		DegradedPolicy:                           normalizeReadinessAdmissionDegradedPolicy(cfg.DegradedPolicy),
		SandboxRolloutPhase:                      strings.ToLower(strings.TrimSpace(effectiveCfg.Security.Sandbox.Rollout.Phase)),
		SandboxCapacityAction:                    SandboxCapacityActionAllow,
		SandboxCapacityDegradedPolicy:            capacityDegradedPolicy,
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
	decision.SandboxCapacityAction = sandboxCapacityActionFromFindings(preflight.Findings)
	if decision.SandboxCapacityAction == "" {
		decision.SandboxCapacityAction = SandboxCapacityActionAllow
	}
	if sandboxRolloutFrozenFromFindings(preflight.Findings) {
		decision.Outcome = ReadinessAdmissionOutcomeDeny
		decision.ReasonCode = ReadinessAdmissionCodeSandboxFrozen
		return decision
	}
	switch decision.SandboxCapacityAction {
	case SandboxCapacityActionDeny:
		decision.Outcome = ReadinessAdmissionOutcomeDeny
		decision.ReasonCode = ReadinessAdmissionCodeSandboxCapacityDeny
		return decision
	case SandboxCapacityActionThrottle:
		if decision.SandboxCapacityDegradedPolicy == SecuritySandboxCapacityDegradedPolicyFailFast {
			decision.Outcome = ReadinessAdmissionOutcomeDeny
			decision.ReasonCode = ReadinessAdmissionCodeSandboxThrottledDeny
		} else {
			decision.Outcome = ReadinessAdmissionOutcomeAllow
			decision.ReasonCode = ReadinessAdmissionCodeSandboxThrottle
		}
		return decision
	}
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
	egressCfg := sandboxCfg.Egress
	severity := ReadinessSeverityWarning
	if sandboxCfg.Required {
		severity = ReadinessSeverityError
	}
	rolloutPhase := strings.ToLower(strings.TrimSpace(sandboxCfg.Rollout.Phase))
	rolloutState := m.SandboxRolloutRuntimeState()
	healthBudgetStatus := normalizeSandboxHealthBudgetStatus(rolloutState.HealthBudgetStatus)
	capacityAction := normalizeSandboxCapacityAction(rolloutState.CapacityAction)
	capacityQueueDepth := rolloutState.CapacityQueueDepth
	capacityInflight := rolloutState.CapacityInflight
	if capacityQueueDepth < 0 {
		capacityQueueDepth = 0
	}
	if capacityInflight < 0 {
		capacityInflight = 0
	}
	if capacityAction == "" {
		capacityAction = evaluateSandboxCapacityAction(sandboxCfg.Capacity, capacityQueueDepth, capacityInflight)
	}
	freezeState := rolloutState.FreezeState || rolloutPhase == SecuritySandboxRolloutPhaseFrozen
	capacityDegradedPolicy := normalizeSandboxCapacityDegradedPolicy(sandboxCfg.Capacity.DegradedPolicy)
	metadataBase := map[string]any{
		"sandbox_enabled":                 true,
		"sandbox_required":                sandboxCfg.Required,
		"sandbox_mode":                    strings.ToLower(strings.TrimSpace(sandboxCfg.Mode)),
		"sandbox_backend":                 strings.ToLower(strings.TrimSpace(sandboxCfg.Executor.Backend)),
		"sandbox_session_mode":            strings.ToLower(strings.TrimSpace(sandboxCfg.Executor.SessionMode)),
		"sandbox_rollout_phase":           rolloutPhase,
		"sandbox_capacity_action":         capacityAction,
		"sandbox_capacity_queue_depth":    capacityQueueDepth,
		"sandbox_capacity_inflight":       capacityInflight,
		"sandbox_capacity_policy":         capacityDegradedPolicy,
		"sandbox_health_budget_status":    healthBudgetStatus,
		"sandbox_health_budget_breaches":  rolloutState.HealthBudgetBreachTotal,
		"sandbox_freeze_state":            freezeState,
		"sandbox_egress_enabled":          egressCfg.Enabled,
		"sandbox_egress_default_action":   strings.ToLower(strings.TrimSpace(egressCfg.DefaultAction)),
		"sandbox_egress_on_violation":     strings.ToLower(strings.TrimSpace(egressCfg.OnViolation)),
		"sandbox_egress_allowlist_total":  len(egressCfg.Allowlist),
		"sandbox_egress_by_tool_total":    len(egressCfg.ByTool),
		"sandbox_egress_violation_total":  rolloutState.EgressViolationTotal,
		"sandbox_egress_violation_budget": rolloutState.EgressViolationBudget,
		"sandbox_egress_budget_breached":  rolloutState.EgressBudgetBreached,
	}
	if reason := strings.TrimSpace(rolloutState.FreezeReasonCode); reason != "" {
		metadataBase["sandbox_freeze_reason_code"] = reason
	}
	selectedProfile := ResolveSandboxProfile(sandboxCfg, "")
	metadataBase["sandbox_profile"] = selectedProfile
	findings := make([]ReadinessFinding, 0, 10)
	findings = append(findings, sandboxEgressReadinessFindings(egressCfg, rolloutState, metadataBase)...)
	configuredBackend := strings.ToLower(strings.TrimSpace(sandboxCfg.Executor.Backend))
	hostOS := strings.ToLower(strings.TrimSpace(runtime.GOOS))
	hostArch := strings.ToLower(strings.TrimSpace(runtime.GOARCH))
	metadataBase["runtime_host_os"] = hostOS
	metadataBase["runtime_host_arch"] = hostArch

	if !containsNormalizedString(sandboxAdapterSupportedBackends(hostOS), configuredBackend) {
		metadata := cloneAnyMap(metadataBase)
		metadata["host_supported_backends"] = sandboxAdapterSupportedBackends(hostOS)
		findings = append(findings, ReadinessFinding{
			Code:     ReadinessCodeSandboxAdapterBackendNotSupported,
			Domain:   ReadinessDomainRuntime,
			Severity: severity,
			Message:  "sandbox backend is not supported on current host",
			Metadata: metadata,
		})
	}
	if expectedHostOS, expectedHostArch, ok := sandboxAdapterHostConstraint(configuredBackend); ok {
		if hostOS != expectedHostOS || hostArch != expectedHostArch {
			metadata := cloneAnyMap(metadataBase)
			metadata["expected_host_os"] = expectedHostOS
			metadata["expected_host_arch"] = expectedHostArch
			findings = append(findings, ReadinessFinding{
				Code:     ReadinessCodeSandboxAdapterHostMismatch,
				Domain:   ReadinessDomainRuntime,
				Severity: severity,
				Message:  "sandbox backend host constraint mismatches runtime host",
				Metadata: metadata,
			})
		}
	}

	if !isSandboxRolloutPhase(rolloutPhase) {
		findings = append(findings, ReadinessFinding{
			Code:     ReadinessCodeSandboxRolloutPhaseInvalid,
			Domain:   ReadinessDomainRuntime,
			Severity: ReadinessSeverityError,
			Message:  "sandbox rollout phase is invalid",
			Metadata: cloneAnyMap(metadataBase),
		})
	}

	if healthBudgetStatus == SandboxHealthBudgetBreached {
		findings = append(findings, ReadinessFinding{
			Code:     ReadinessCodeSandboxRolloutHealthBreached,
			Domain:   ReadinessDomainRuntime,
			Severity: ReadinessSeverityWarning,
			Message:  "sandbox rollout health budget is breached",
			Metadata: cloneAnyMap(metadataBase),
		})
	}

	if freezeState {
		findings = append(findings, ReadinessFinding{
			Code:     ReadinessCodeSandboxRolloutFrozen,
			Domain:   ReadinessDomainRuntime,
			Severity: ReadinessSeverityError,
			Message:  "sandbox rollout is frozen",
			Metadata: cloneAnyMap(metadataBase),
		})
	}

	switch capacityAction {
	case SandboxCapacityActionThrottle:
		findings = append(findings, ReadinessFinding{
			Code:     ReadinessCodeSandboxRolloutCapacityBlocked,
			Domain:   ReadinessDomainRuntime,
			Severity: ReadinessSeverityWarning,
			Message:  "sandbox rollout capacity is throttled",
			Metadata: cloneAnyMap(metadataBase),
		})
	case SandboxCapacityActionDeny:
		findings = append(findings, ReadinessFinding{
			Code:     ReadinessCodeSandboxRolloutCapacityBlocked,
			Domain:   ReadinessDomainRuntime,
			Severity: ReadinessSeverityError,
			Message:  "sandbox rollout capacity is unavailable",
			Metadata: cloneAnyMap(metadataBase),
		})
	}

	profile, ok := sandboxCfg.Profiles[selectedProfile]
	if !ok {
		findings = append(findings, ReadinessFinding{
			Code:     ReadinessCodeSandboxAdapterProfileMissing,
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
	if probeBackend != "" && probeBackend != configuredBackend {
		metadata := cloneAnyMap(metadataBase)
		metadata["probe_backend"] = probeBackend
		findings = append(findings, ReadinessFinding{
			Code:     ReadinessCodeSandboxAdapterHostMismatch,
			Domain:   ReadinessDomainRuntime,
			Severity: severity,
			Message:  "sandbox executor probe backend mismatches configured backend",
			Metadata: metadata,
		})
	}
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
			Code:     ReadinessCodeSandboxAdapterSessionModeUnsupported,
			Domain:   ReadinessDomainRuntime,
			Severity: severity,
			Message:  "sandbox executor does not support configured session_mode",
			Metadata: metadata,
		})
	}

	return findings
}

func sandboxEgressReadinessFindings(cfg SecuritySandboxEgressConfig, state SandboxRolloutRuntimeState, metadataBase map[string]any) []ReadinessFinding {
	if !cfg.Enabled {
		return nil
	}
	findings := make([]ReadinessFinding, 0, 4)
	metadata := cloneAnyMap(metadataBase)
	if metadata == nil {
		metadata = map[string]any{}
	}
	defaultAction := strings.ToLower(strings.TrimSpace(cfg.DefaultAction))
	onViolation := strings.ToLower(strings.TrimSpace(cfg.OnViolation))
	metadata["sandbox_egress_default_action"] = defaultAction
	metadata["sandbox_egress_on_violation"] = onViolation

	if !isSandboxEgressAction(defaultAction) || !isSandboxEgressOnViolation(onViolation) {
		findings = append(findings, ReadinessFinding{
			Code:     ReadinessCodeSandboxEgressPolicyInvalid,
			Domain:   ReadinessDomainRuntime,
			Severity: ReadinessSeverityError,
			Message:  "sandbox egress policy contains unsupported enum values",
			Metadata: cloneAnyMap(metadata),
		})
	}

	byToolKeys := make([]string, 0, len(cfg.ByTool))
	for selector := range cfg.ByTool {
		byToolKeys = append(byToolKeys, selector)
	}
	sort.Strings(byToolKeys)
	for i := range byToolKeys {
		selector := byToolKeys[i]
		action := strings.ToLower(strings.TrimSpace(cfg.ByTool[selector]))
		selectorField := fmt.Sprintf("security.sandbox.egress.by_tool.%s", selector)
		if err := validateNamespaceToolKey(selector, selectorField); err != nil || !isSandboxEgressAction(action) {
			itemMetadata := cloneAnyMap(metadata)
			itemMetadata["sandbox_egress_selector"] = strings.ToLower(strings.TrimSpace(selector))
			itemMetadata["sandbox_egress_selector_action"] = action
			findings = append(findings, ReadinessFinding{
				Code:     ReadinessCodeSandboxEgressPolicyInvalid,
				Domain:   ReadinessDomainRuntime,
				Severity: ReadinessSeverityError,
				Message:  "sandbox egress by_tool policy contains invalid selector or action",
				Metadata: itemMetadata,
			})
		}
	}

	seenAllowlist := map[string]struct{}{}
	for i := range cfg.Allowlist {
		raw := cfg.Allowlist[i]
		entry := strings.ToLower(strings.TrimSpace(raw))
		field := fmt.Sprintf("security.sandbox.egress.allowlist[%d]", i)
		if err := validateSandboxEgressAllowlistPattern(raw, field); err != nil {
			itemMetadata := cloneAnyMap(metadata)
			itemMetadata["sandbox_egress_allowlist_index"] = i
			itemMetadata["sandbox_egress_allowlist_entry"] = entry
			findings = append(findings, ReadinessFinding{
				Code:     ReadinessCodeSandboxEgressAllowlistInvalid,
				Domain:   ReadinessDomainRuntime,
				Severity: ReadinessSeverityError,
				Message:  "sandbox egress allowlist contains malformed host pattern",
				Metadata: itemMetadata,
			})
			continue
		}
		if _, ok := seenAllowlist[entry]; ok {
			itemMetadata := cloneAnyMap(metadata)
			itemMetadata["sandbox_egress_allowlist_entry"] = entry
			findings = append(findings, ReadinessFinding{
				Code:     ReadinessCodeSandboxEgressAllowlistInvalid,
				Domain:   ReadinessDomainRuntime,
				Severity: ReadinessSeverityError,
				Message:  "sandbox egress allowlist contains duplicated host pattern",
				Metadata: itemMetadata,
			})
			continue
		}
		seenAllowlist[entry] = struct{}{}
	}

	if defaultAction == SecuritySandboxEgressActionAllow && len(cfg.Allowlist) > 0 {
		findings = append(findings, ReadinessFinding{
			Code:     ReadinessCodeSandboxEgressRuleConflict,
			Domain:   ReadinessDomainRuntime,
			Severity: ReadinessSeverityWarning,
			Message:  "sandbox egress allowlist conflicts with default_action=allow",
			Metadata: cloneAnyMap(metadata),
		})
	}

	if state.EgressBudgetBreached {
		itemMetadata := cloneAnyMap(metadata)
		itemMetadata["sandbox_egress_violation_total"] = state.EgressViolationTotal
		itemMetadata["sandbox_egress_violation_budget"] = state.EgressViolationBudget
		findings = append(findings, ReadinessFinding{
			Code:     ReadinessCodeSandboxEgressViolationBudgetBreached,
			Domain:   ReadinessDomainRuntime,
			Severity: ReadinessSeverityWarning,
			Message:  "sandbox egress violation budget is breached",
			Metadata: itemMetadata,
		})
	}
	return findings
}

func (m *Manager) adapterAllowlistReadinessFindings(cfg Config) []ReadinessFinding {
	if m == nil || !cfg.Adapter.Allowlist.Enabled {
		return nil
	}
	allowlist := cfg.Adapter.Allowlist
	mode := strings.ToLower(strings.TrimSpace(allowlist.EnforcementMode))
	onUnknown := strings.ToLower(strings.TrimSpace(allowlist.OnUnknownSignature))
	baseMetadata := map[string]any{
		"adapter_allowlist_enabled":              true,
		"adapter_allowlist_enforcement_mode":     mode,
		"adapter_allowlist_on_unknown_signature": onUnknown,
		"adapter_allowlist_entry_total":          len(allowlist.Entries),
	}
	findings := make([]ReadinessFinding, 0, 4)
	if !isAdapterAllowlistEnforcementMode(mode) || !isAdapterAllowlistUnknownSignature(onUnknown) {
		findings = append(findings, ReadinessFinding{
			Code:     ReadinessCodeAdapterAllowlistPolicyConflict,
			Domain:   ReadinessDomainAdapter,
			Severity: ReadinessSeverityError,
			Message:  "adapter allowlist policy contains unsupported enum values",
			Metadata: cloneAnyMap(baseMetadata),
		})
		return findings
	}
	if mode == AdapterAllowlistEnforcementModeObserve && onUnknown == AdapterAllowlistUnknownSignatureDeny {
		findings = append(findings, ReadinessFinding{
			Code:     ReadinessCodeAdapterAllowlistPolicyConflict,
			Domain:   ReadinessDomainAdapter,
			Severity: ReadinessSeverityError,
			Message:  "adapter allowlist on_unknown_signature=deny conflicts with enforcement_mode=observe",
			Metadata: cloneAnyMap(baseMetadata),
		})
		return findings
	}
	if mode == AdapterAllowlistEnforcementModeEnforce && onUnknown == AdapterAllowlistUnknownSignatureAllowAndRecord {
		findings = append(findings, ReadinessFinding{
			Code:     ReadinessCodeAdapterAllowlistPolicyConflict,
			Domain:   ReadinessDomainAdapter,
			Severity: ReadinessSeverityError,
			Message:  "adapter allowlist on_unknown_signature=allow_and_record conflicts with enforcement_mode=enforce",
			Metadata: cloneAnyMap(baseMetadata),
		})
		return findings
	}

	requiredAdapterIDs := make([]string, 0)
	targets := m.AdapterHealthTargets()
	for i := range targets {
		if !targets[i].Required {
			continue
		}
		adapterID := strings.ToLower(strings.TrimSpace(targets[i].Name))
		if adapterID == "" {
			continue
		}
		requiredAdapterIDs = append(requiredAdapterIDs, adapterID)
	}
	sort.Strings(requiredAdapterIDs)

	for i := range requiredAdapterIDs {
		adapterID := requiredAdapterIDs[i]
		matches := matchingAdapterAllowlistEntriesByID(allowlist.Entries, adapterID)
		if len(matches) == 0 {
			severity := ReadinessSeverityWarning
			if mode == AdapterAllowlistEnforcementModeEnforce {
				severity = ReadinessSeverityError
			}
			itemMetadata := cloneAnyMap(baseMetadata)
			itemMetadata["adapter_id"] = adapterID
			itemMetadata["required"] = true
			findings = append(findings, ReadinessFinding{
				Code:     ReadinessCodeAdapterAllowlistMissingEntry,
				Domain:   ReadinessDomainAdapter,
				Severity: severity,
				Message:  "required adapter identity is missing from allowlist entries",
				Metadata: itemMetadata,
			})
			continue
		}
		if mode == AdapterAllowlistEnforcementModeEnforce && adapterAllowlistMatchesSignatureInvalid(matches, onUnknown) {
			itemMetadata := cloneAnyMap(baseMetadata)
			itemMetadata["adapter_id"] = adapterID
			itemMetadata["required"] = true
			itemMetadata["entry_signature_statuses"] = adapterAllowlistSignatureStatuses(matches)
			findings = append(findings, ReadinessFinding{
				Code:     ReadinessCodeAdapterAllowlistSignatureInvalid,
				Domain:   ReadinessDomainAdapter,
				Severity: ReadinessSeverityError,
				Message:  "required adapter allowlist signature status is invalid under enforce mode",
				Metadata: itemMetadata,
			})
		}
	}

	return findings
}

func matchingAdapterAllowlistEntriesByID(entries []AdapterAllowlistEntry, adapterID string) []AdapterAllowlistEntry {
	target := strings.ToLower(strings.TrimSpace(adapterID))
	if target == "" || len(entries) == 0 {
		return nil
	}
	matches := make([]AdapterAllowlistEntry, 0)
	for i := range entries {
		if strings.ToLower(strings.TrimSpace(entries[i].AdapterID)) != target {
			continue
		}
		matches = append(matches, entries[i])
	}
	sort.Slice(matches, func(i, j int) bool {
		left := strings.ToLower(strings.TrimSpace(matches[i].Publisher)) + "|" + strings.ToLower(strings.TrimSpace(matches[i].Version))
		right := strings.ToLower(strings.TrimSpace(matches[j].Publisher)) + "|" + strings.ToLower(strings.TrimSpace(matches[j].Version))
		if left != right {
			return left < right
		}
		return strings.ToLower(strings.TrimSpace(matches[i].SignatureStatus)) < strings.ToLower(strings.TrimSpace(matches[j].SignatureStatus))
	})
	return matches
}

func adapterAllowlistMatchesSignatureInvalid(matches []AdapterAllowlistEntry, onUnknown string) bool {
	if len(matches) == 0 {
		return false
	}
	hasValid := false
	hasUnknown := false
	hasInvalid := false
	for i := range matches {
		switch strings.ToLower(strings.TrimSpace(matches[i].SignatureStatus)) {
		case AdapterAllowlistSignatureStatusValid:
			hasValid = true
		case AdapterAllowlistSignatureStatusUnknown:
			hasUnknown = true
		case AdapterAllowlistSignatureStatusInvalid:
			hasInvalid = true
		}
	}
	if hasValid {
		return false
	}
	if hasInvalid {
		return true
	}
	if hasUnknown && onUnknown == AdapterAllowlistUnknownSignatureDeny {
		return true
	}
	return false
}

func adapterAllowlistSignatureStatuses(entries []AdapterAllowlistEntry) []string {
	if len(entries) == 0 {
		return nil
	}
	out := make([]string, 0, len(entries))
	for i := range entries {
		status := strings.ToLower(strings.TrimSpace(entries[i].SignatureStatus))
		if status == "" {
			continue
		}
		out = append(out, status)
	}
	sort.Strings(out)
	return out
}

func isSandboxEgressAction(action string) bool {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case SecuritySandboxEgressActionDeny, SecuritySandboxEgressActionAllow, SecuritySandboxEgressActionAllowAndRecord:
		return true
	default:
		return false
	}
}

func isSandboxEgressOnViolation(action string) bool {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case SecuritySandboxEgressOnViolationDeny, SecuritySandboxEgressOnViolationAllowAndRecord:
		return true
	default:
		return false
	}
}

func isAdapterAllowlistEnforcementMode(mode string) bool {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case AdapterAllowlistEnforcementModeObserve, AdapterAllowlistEnforcementModeEnforce:
		return true
	default:
		return false
	}
}

func isAdapterAllowlistUnknownSignature(policy string) bool {
	switch strings.ToLower(strings.TrimSpace(policy)) {
	case AdapterAllowlistUnknownSignatureDeny, AdapterAllowlistUnknownSignatureAllowAndRecord:
		return true
	default:
		return false
	}
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

func reactReadinessFindings(cfg Config, snapshot ReactReadinessDependencySnapshot) []ReadinessFinding {
	reactCfg := normalizeRuntimeReactConfig(cfg.Runtime.React)
	base := map[string]any{
		"react_enabled":                     reactCfg.Enabled,
		"react_max_iterations":              reactCfg.MaxIterations,
		"react_tool_call_limit":             reactCfg.ToolCallLimit,
		"react_stream_dispatch_enabled":     reactCfg.StreamToolDispatchEnabled,
		"react_on_budget_exhausted":         reactCfg.OnBudgetExhausted,
		"react_dependency_snapshot_checked": snapshot.ToolRegistryChecked || snapshot.ProviderChecked || snapshot.SandboxDependencyChecked,
	}
	findings := make([]ReadinessFinding, 0, 5)

	if !reactCfg.Enabled {
		findings = append(findings, ReadinessFinding{
			Code:     ReadinessCodeReactLoopDisabled,
			Domain:   ReadinessDomainRuntime,
			Severity: ReadinessSeverityWarning,
			Message:  "react loop is disabled in runtime config",
			Metadata: cloneAnyMap(base),
		})
		return findings
	}

	if !reactCfg.StreamToolDispatchEnabled {
		findings = append(findings, ReadinessFinding{
			Code:     ReadinessCodeReactStreamDispatchUnavailable,
			Domain:   ReadinessDomainRuntime,
			Severity: ReadinessSeverityWarning,
			Message:  "react stream tool dispatch is unavailable",
			Metadata: cloneAnyMap(base),
		})
	}

	if snapshot.ProviderChecked && !snapshot.ProviderToolCallingSupported {
		metadata := cloneAnyMap(base)
		if provider := strings.ToLower(strings.TrimSpace(snapshot.ProviderName)); provider != "" {
			metadata["react_provider"] = provider
		}
		if reason := strings.TrimSpace(snapshot.ProviderReason); reason != "" {
			metadata["react_provider_reason"] = reason
		}
		findings = append(findings, ReadinessFinding{
			Code:     ReadinessCodeReactProviderToolCallingUnsupported,
			Domain:   ReadinessDomainRuntime,
			Severity: ReadinessSeverityError,
			Message:  "react provider does not support required tool-calling capability",
			Metadata: metadata,
		})
	}

	if snapshot.ToolRegistryChecked && !snapshot.ToolRegistryAvailable {
		metadata := cloneAnyMap(base)
		if reason := strings.TrimSpace(snapshot.ToolRegistryReason); reason != "" {
			metadata["react_tool_registry_reason"] = reason
		}
		findings = append(findings, ReadinessFinding{
			Code:     ReadinessCodeReactToolRegistryUnavailable,
			Domain:   ReadinessDomainRuntime,
			Severity: ReadinessSeverityWarning,
			Message:  "react tool registry is unavailable",
			Metadata: metadata,
		})
	}

	if snapshot.SandboxDependencyChecked && !snapshot.SandboxDependencyAvailable {
		metadata := cloneAnyMap(base)
		if reason := strings.TrimSpace(snapshot.SandboxDependencyReason); reason != "" {
			metadata["react_sandbox_dependency_reason"] = reason
		}
		findings = append(findings, ReadinessFinding{
			Code:     ReadinessCodeReactSandboxDependencyUnavailable,
			Domain:   ReadinessDomainRuntime,
			Severity: ReadinessSeverityWarning,
			Message:  "react sandbox dependency is unavailable",
			Metadata: metadata,
		})
	}

	return findings
}

func memoryReadinessFindings(cfg Config) []ReadinessFinding {
	memoryCfg := normalizeRuntimeMemoryConfig(cfg.Runtime.Memory)
	findings := make([]ReadinessFinding, 0, 4)
	switch memoryCfg.Mode {
	case RuntimeMemoryModeBuiltinFilesystem, RuntimeMemoryModeExternalSPI:
	default:
		findings = append(findings, ReadinessFinding{
			Code:     ReadinessCodeMemoryModeInvalid,
			Domain:   ReadinessDomainRuntime,
			Severity: ReadinessSeverityError,
			Message:  "runtime memory mode is invalid",
			Metadata: map[string]any{
				"runtime_memory_mode": strings.TrimSpace(memoryCfg.Mode),
			},
		})
		return findings
	}

	if strings.TrimSpace(memoryCfg.External.ContractVersion) != RuntimeMemoryContractVersionV1 {
		findings = append(findings, ReadinessFinding{
			Code:     ReadinessCodeMemoryContractVersionMismatch,
			Domain:   ReadinessDomainRuntime,
			Severity: ReadinessSeverityError,
			Message:  "runtime memory contract version is unsupported",
			Metadata: map[string]any{
				"runtime_memory_contract_version": strings.TrimSpace(memoryCfg.External.ContractVersion),
			},
		})
	}

	if memoryCfg.Mode == RuntimeMemoryModeExternalSPI {
		if strings.TrimSpace(memoryCfg.External.Profile) == "" {
			findings = append(findings, ReadinessFinding{
				Code:     ReadinessCodeMemoryProfileMissing,
				Domain:   ReadinessDomainRuntime,
				Severity: ReadinessSeverityError,
				Message:  "runtime memory profile is required in external_spi mode",
				Metadata: map[string]any{
					"runtime_memory_mode": strings.TrimSpace(memoryCfg.Mode),
				},
			})
		}
		if !isSupportedMemoryProvider(memoryCfg.External.Provider) {
			findings = append(findings, ReadinessFinding{
				Code:     ReadinessCodeMemoryProviderNotSupported,
				Domain:   ReadinessDomainRuntime,
				Severity: ReadinessSeverityError,
				Message:  "runtime memory provider is not supported",
				Metadata: map[string]any{
					"runtime_memory_provider": strings.TrimSpace(memoryCfg.External.Provider),
				},
			})
		}
		stage2Provider := strings.ToLower(strings.TrimSpace(cfg.ContextAssembler.CA2.Stage2.Provider))
		if stage2Provider == ContextStage2ProviderMemory && strings.TrimSpace(cfg.ContextAssembler.CA2.Stage2.External.Endpoint) == "" {
			findings = append(findings, ReadinessFinding{
				Code:     ReadinessCodeMemorySPIUnavailable,
				Domain:   ReadinessDomainRuntime,
				Severity: ReadinessSeverityWarning,
				Message:  "runtime memory external spi endpoint is unavailable",
				Metadata: map[string]any{
					"runtime_memory_provider": strings.TrimSpace(memoryCfg.External.Provider),
					"context_stage2_provider": stage2Provider,
				},
			})
		}
	}

	if memoryCfg.Mode == RuntimeMemoryModeBuiltinFilesystem {
		if err := validateMemoryFilesystemPathReadiness(memoryCfg.Builtin.RootDir); err != nil {
			findings = append(findings, ReadinessFinding{
				Code:     ReadinessCodeMemoryFilesystemPathInvalid,
				Domain:   ReadinessDomainRuntime,
				Severity: ReadinessSeverityError,
				Message:  "runtime memory filesystem path is invalid",
				Metadata: map[string]any{
					"runtime_memory_root_dir": strings.TrimSpace(memoryCfg.Builtin.RootDir),
					"error":                   strings.TrimSpace(err.Error()),
				},
			})
		}
	}

	if memoryCfg.Mode == RuntimeMemoryModeBuiltinFilesystem && memoryCfg.Fallback.Policy == RuntimeMemoryFallbackPolicyDegradeToBuiltin {
		findings = append(findings, ReadinessFinding{
			Code:     ReadinessCodeMemoryFallbackPolicyConflict,
			Domain:   ReadinessDomainRuntime,
			Severity: ReadinessSeverityWarning,
			Message:  "runtime memory fallback policy conflicts with builtin_filesystem mode",
			Metadata: map[string]any{
				"runtime_memory_mode":            strings.TrimSpace(memoryCfg.Mode),
				"runtime_memory_fallback_policy": strings.TrimSpace(memoryCfg.Fallback.Policy),
			},
		})
	}

	if memoryCfg.Mode == RuntimeMemoryModeExternalSPI && memoryCfg.Fallback.Policy == RuntimeMemoryFallbackPolicyDegradeToBuiltin {
		if err := validateMemoryFilesystemPathReadiness(memoryCfg.Builtin.RootDir); err != nil {
			findings = append(findings, ReadinessFinding{
				Code:     ReadinessCodeMemoryFallbackTargetUnavailable,
				Domain:   ReadinessDomainRuntime,
				Severity: ReadinessSeverityWarning,
				Message:  "runtime memory fallback target builtin filesystem is unavailable",
				Metadata: map[string]any{
					"runtime_memory_root_dir": strings.TrimSpace(memoryCfg.Builtin.RootDir),
					"error":                   strings.TrimSpace(err.Error()),
				},
			})
		}
	}
	return findings
}

func observabilityReadinessFindings(cfg Config) []ReadinessFinding {
	exportCfg := normalizeRuntimeObservabilityConfig(cfg.Runtime.Observability).Export
	bundleCfg := normalizeRuntimeDiagnosticsBundleConfig(cfg.Runtime.Diagnostics.Bundle)
	findings := make([]ReadinessFinding, 0, 4)

	if !isSupportedRuntimeObservabilityExportProfile(exportCfg.Profile) {
		findings = append(findings, ReadinessFinding{
			Code:     ReadinessCodeObservabilityExportProfileInvalid,
			Domain:   ReadinessDomainRuntime,
			Severity: ReadinessSeverityError,
			Message:  "runtime observability export profile is invalid",
			Metadata: map[string]any{
				"runtime_observability_export_profile": strings.TrimSpace(exportCfg.Profile),
			},
		})
	}

	if exportCfg.Enabled &&
		exportCfg.Profile != RuntimeObservabilityExportProfileNone &&
		isObservabilityExportSinkUnavailable(exportCfg.Endpoint) {
		findings = append(findings, ReadinessFinding{
			Code:     ReadinessCodeObservabilityExportSinkUnavailable,
			Domain:   ReadinessDomainRuntime,
			Severity: ReadinessSeverityWarning,
			Message:  "runtime observability export sink is unavailable",
			Metadata: map[string]any{
				"runtime_observability_export_profile":  strings.TrimSpace(exportCfg.Profile),
				"runtime_observability_export_endpoint": strings.TrimSpace(exportCfg.Endpoint),
			},
		})
	}

	if exportCfg.Enabled && exportCfg.Profile == RuntimeObservabilityExportProfileLangfuse {
		endpoint := strings.ToLower(strings.TrimSpace(exportCfg.Endpoint))
		if strings.Contains(endpoint, "auth_invalid") {
			findings = append(findings, ReadinessFinding{
				Code:     ReadinessCodeObservabilityExportAuthInvalid,
				Domain:   ReadinessDomainRuntime,
				Severity: ReadinessSeverityWarning,
				Message:  "runtime observability export auth looks invalid",
				Metadata: map[string]any{
					"runtime_observability_export_profile":  strings.TrimSpace(exportCfg.Profile),
					"runtime_observability_export_endpoint": strings.TrimSpace(exportCfg.Endpoint),
				},
			})
		}
	}

	if err := validateRuntimeDiagnosticsBundlePolicyReadiness(bundleCfg); err != nil {
		findings = append(findings, ReadinessFinding{
			Code:     ReadinessCodeDiagnosticsBundlePolicyInvalid,
			Domain:   ReadinessDomainRuntime,
			Severity: ReadinessSeverityError,
			Message:  "runtime diagnostics bundle policy is invalid",
			Metadata: map[string]any{
				"runtime_diagnostics_bundle_output_dir": strings.TrimSpace(bundleCfg.OutputDir),
				"error":                                 strings.TrimSpace(err.Error()),
			},
		})
		return findings
	}

	if bundleCfg.Enabled {
		if err := validateObservabilityBundleOutputDirReadiness(bundleCfg.OutputDir); err != nil {
			findings = append(findings, ReadinessFinding{
				Code:     ReadinessCodeDiagnosticsBundleOutputUnavailable,
				Domain:   ReadinessDomainRuntime,
				Severity: ReadinessSeverityWarning,
				Message:  "runtime diagnostics bundle output path is unavailable",
				Metadata: map[string]any{
					"runtime_diagnostics_bundle_output_dir": strings.TrimSpace(bundleCfg.OutputDir),
					"error":                                 strings.TrimSpace(err.Error()),
				},
			})
		}
	}
	return findings
}

func validateMemoryFilesystemPathReadiness(root string) error {
	path := strings.TrimSpace(root)
	if path == "" {
		return fmt.Errorf("runtime memory builtin root_dir is required")
	}
	return os.MkdirAll(path, 0o755)
}

func validateRuntimeDiagnosticsBundlePolicyReadiness(cfg RuntimeDiagnosticsBundleConfig) error {
	return ValidateRuntimeDiagnosticsBundleConfig(cfg)
}

func validateObservabilityBundleOutputDirReadiness(path string) error {
	if err := validateRuntimeDiagnosticsBundleOutputDir(path); err != nil {
		return err
	}
	return os.MkdirAll(strings.TrimSpace(path), 0o755)
}

func isObservabilityExportSinkUnavailable(endpoint string) bool {
	raw := strings.TrimSpace(endpoint)
	if raw == "" {
		return true
	}
	lower := strings.ToLower(raw)
	if strings.Contains(lower, "sink_unavailable") {
		return true
	}
	if strings.Contains(lower, "127.0.0.1:9") || strings.Contains(lower, "localhost:9") || strings.Contains(lower, "[::1]:9") {
		return true
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Host)
	if strings.Contains(host, "127.0.0.1:9") || strings.Contains(host, "localhost:9") || strings.Contains(host, "[::1]:9") {
		return true
	}
	return false
}

func isSupportedMemoryProvider(provider string) bool {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "mem0", "zep", "openviking", RuntimeMemoryProviderGeneric:
		return true
	default:
		return false
	}
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

func normalizeSandboxHealthBudgetStatus(in string) string {
	switch strings.ToLower(strings.TrimSpace(in)) {
	case SandboxHealthBudgetNearBudget:
		return SandboxHealthBudgetNearBudget
	case SandboxHealthBudgetBreached:
		return SandboxHealthBudgetBreached
	default:
		return SandboxHealthBudgetWithinBudget
	}
}

func normalizeSandboxCapacityAction(in string) string {
	switch strings.ToLower(strings.TrimSpace(in)) {
	case SandboxCapacityActionThrottle:
		return SandboxCapacityActionThrottle
	case SandboxCapacityActionDeny:
		return SandboxCapacityActionDeny
	case SandboxCapacityActionAllow:
		return SandboxCapacityActionAllow
	default:
		return ""
	}
}

func normalizeSandboxCapacityDegradedPolicy(in string) string {
	switch strings.ToLower(strings.TrimSpace(in)) {
	case SecuritySandboxCapacityDegradedPolicyFailFast:
		return SecuritySandboxCapacityDegradedPolicyFailFast
	case SecuritySandboxCapacityDegradedPolicyAllowAndRecord:
		return SecuritySandboxCapacityDegradedPolicyAllowAndRecord
	default:
		return SecuritySandboxCapacityDegradedPolicyAllowAndRecord
	}
}

func isSandboxRolloutPhase(phase string) bool {
	switch strings.ToLower(strings.TrimSpace(phase)) {
	case SecuritySandboxRolloutPhaseObserve,
		SecuritySandboxRolloutPhaseCanary,
		SecuritySandboxRolloutPhaseBaseline,
		SecuritySandboxRolloutPhaseFull,
		SecuritySandboxRolloutPhaseFrozen:
		return true
	default:
		return false
	}
}

func evaluateSandboxCapacityAction(cfg SecuritySandboxCapacityConfig, queueDepth, inflight int) string {
	if queueDepth < 0 {
		queueDepth = 0
	}
	if inflight < 0 {
		inflight = 0
	}
	if inflight >= cfg.MaxInflight || queueDepth >= cfg.DenyThreshold {
		return SandboxCapacityActionDeny
	}
	if queueDepth >= cfg.ThrottleThreshold {
		return SandboxCapacityActionThrottle
	}
	return SandboxCapacityActionAllow
}

func sandboxRolloutFrozenFromFindings(findings []ReadinessFinding) bool {
	for i := range findings {
		if strings.TrimSpace(findings[i].Code) == ReadinessCodeSandboxRolloutFrozen {
			return true
		}
	}
	return false
}

func sandboxCapacityActionFromFindings(findings []ReadinessFinding) string {
	for i := range findings {
		if strings.TrimSpace(findings[i].Code) != ReadinessCodeSandboxRolloutCapacityBlocked {
			continue
		}
		if findings[i].Metadata == nil {
			continue
		}
		raw, ok := findings[i].Metadata["sandbox_capacity_action"]
		if !ok {
			continue
		}
		action := normalizeSandboxCapacityAction(fmt.Sprint(raw))
		if action != "" {
			return action
		}
	}
	return ""
}

func sandboxAdapterSupportedBackends(hostOS string) []string {
	switch strings.ToLower(strings.TrimSpace(hostOS)) {
	case "windows":
		return []string{SecuritySandboxBackendWindowsJob}
	case "linux":
		return []string{
			SecuritySandboxBackendLinuxNSJail,
			SecuritySandboxBackendLinuxBwrap,
			SecuritySandboxBackendOCIRuntime,
		}
	default:
		return nil
	}
}

func sandboxAdapterHostConstraint(backend string) (string, string, bool) {
	switch strings.ToLower(strings.TrimSpace(backend)) {
	case SecuritySandboxBackendLinuxNSJail, SecuritySandboxBackendLinuxBwrap, SecuritySandboxBackendOCIRuntime:
		return "linux", "amd64", true
	case SecuritySandboxBackendWindowsJob:
		return "windows", "amd64", true
	default:
		return "", "", false
	}
}

func containsNormalizedString(items []string, target string) bool {
	needle := strings.ToLower(strings.TrimSpace(target))
	if needle == "" {
		return false
	}
	for _, item := range items {
		if strings.ToLower(strings.TrimSpace(item)) == needle {
			return true
		}
	}
	return false
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

func cloneReactReadinessDependencySnapshot(in ReactReadinessDependencySnapshot) ReactReadinessDependencySnapshot {
	return ReactReadinessDependencySnapshot{
		ToolRegistryChecked:          in.ToolRegistryChecked,
		ToolRegistryAvailable:        in.ToolRegistryAvailable,
		ToolRegistryReason:           strings.TrimSpace(in.ToolRegistryReason),
		ProviderChecked:              in.ProviderChecked,
		ProviderName:                 strings.ToLower(strings.TrimSpace(in.ProviderName)),
		ProviderToolCallingSupported: in.ProviderToolCallingSupported,
		ProviderReason:               strings.TrimSpace(in.ProviderReason),
		SandboxDependencyChecked:     in.SandboxDependencyChecked,
		SandboxDependencyAvailable:   in.SandboxDependencyAvailable,
		SandboxDependencyReason:      strings.TrimSpace(in.SandboxDependencyReason),
		UpdatedAt:                    in.UpdatedAt.UTC(),
	}
}

func cloneSandboxRolloutRuntimeState(in SandboxRolloutRuntimeState) SandboxRolloutRuntimeState {
	return SandboxRolloutRuntimeState{
		HealthBudgetStatus:      normalizeSandboxHealthBudgetStatus(in.HealthBudgetStatus),
		HealthBudgetBreachTotal: in.HealthBudgetBreachTotal,
		EgressViolationTotal:    in.EgressViolationTotal,
		EgressViolationBudget:   in.EgressViolationBudget,
		EgressBudgetBreached:    in.EgressBudgetBreached,
		FreezeState:             in.FreezeState,
		FreezeReasonCode:        strings.TrimSpace(in.FreezeReasonCode),
		CapacityQueueDepth:      in.CapacityQueueDepth,
		CapacityInflight:        in.CapacityInflight,
		CapacityAction:          normalizeSandboxCapacityAction(in.CapacityAction),
		UpdatedAt:               in.UpdatedAt.UTC(),
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
