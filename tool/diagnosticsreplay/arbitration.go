package diagnosticsreplay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

const (
	ArbitrationFixtureVersionA48V1             = "a48.v1"
	ArbitrationFixtureVersionA49V1             = "a49.v1"
	ArbitrationFixtureVersionA50V1             = "a50.v1"
	ArbitrationFixtureVersionA51V1             = "a51.v1"
	ArbitrationFixtureVersionA52V1             = "a52.v1"
	ArbitrationFixtureVersionA57V1             = "sandbox_egress.v1"
	ArbitrationFixtureVersionBudgetAdmissionV1 = "budget_admission.v1"
	ArbitrationFixtureVersionPolicyV1          = "policy_stack.v1"
	ArbitrationFixtureVersionMemoryV1          = "memory.v1"
	ArbitrationFixtureVersionMemoryScopeV1     = "memory_scope.v1"
	ArbitrationFixtureVersionMemorySearchV1    = "memory_search.v1"
	ArbitrationFixtureVersionMemoryLifecycleV1 = "memory_lifecycle.v1"
	ArbitrationFixtureVersionObsV1             = "observability.v1"
	ArbitrationFixtureVersionReactV1           = "react.v1"

	ReasonCodePrecedenceConflict                  = "precedence_conflict"
	ReasonCodePrecedenceDrift                     = "precedence_drift"
	ReasonCodeTieBreakDrift                       = "tie_break_drift"
	ReasonCodeDenySourceMismatch                  = "deny_source_mismatch"
	ReasonCodeTaxonomyDrift                       = "taxonomy_drift"
	ReasonCodeSecondaryOrderDrift                 = "secondary_order_drift"
	ReasonCodeSecondaryCountDrift                 = "secondary_count_drift"
	ReasonCodeHintTaxonomyDrift                   = "hint_taxonomy_drift"
	ReasonCodeRuleVersionDrift                    = "rule_version_drift"
	ReasonCodeVersionMismatch                     = "version_mismatch"
	ReasonCodeUnsupportedVersion                  = "unsupported_version"
	ReasonCodeCrossVersionSemanticDrift           = "cross_version_semantic_drift"
	ReasonCodeSandboxPolicyDrift                  = "sandbox_policy_drift"
	ReasonCodeSandboxFallbackDrift                = "sandbox_fallback_drift"
	ReasonCodeSandboxTimeoutDrift                 = "sandbox_timeout_drift"
	ReasonCodeSandboxCapabilityDrift              = "sandbox_capability_drift"
	ReasonCodeSandboxResourcePolicyDrift          = "sandbox_resource_policy_drift"
	ReasonCodeSandboxSessionLifecycleDrift        = "sandbox_session_lifecycle_drift"
	ReasonCodeSandboxRolloutPhaseDrift            = "sandbox_rollout_phase_drift"
	ReasonCodeSandboxHealthBudgetDrift            = "sandbox_health_budget_drift"
	ReasonCodeSandboxCapacityActionDrift          = "sandbox_capacity_action_drift"
	ReasonCodeSandboxFreezeStateDrift             = "sandbox_freeze_state_drift"
	ReasonCodeMemoryModeDrift                     = "memory_mode_drift"
	ReasonCodeMemoryProfileDrift                  = "memory_profile_drift"
	ReasonCodeMemoryContractVersionDrift          = "memory_contract_version_drift"
	ReasonCodeMemoryFallbackDrift                 = "memory_fallback_drift"
	ReasonCodeMemoryErrorTaxonomyDrift            = "memory_error_taxonomy_drift"
	ReasonCodeMemoryOperationAggregateDrift       = "memory_operation_aggregate_drift"
	ReasonCodeObsExportProfileDrift               = "observability_export_profile_drift"
	ReasonCodeObsExportStatusDrift                = "observability_export_status_drift"
	ReasonCodeObsExportReasonDrift                = "observability_export_reason_drift"
	ReasonCodeBundleSchemaDrift                   = "diagnostics_bundle_schema_drift"
	ReasonCodeBundleRedactionDrift                = "diagnostics_bundle_redaction_drift"
	ReasonCodeBundleFingerprintDrift              = "diagnostics_bundle_fingerprint_drift"
	ReasonCodeReactLoopStepDrift                  = "react_loop_step_drift"
	ReasonCodeReactToolCallBudgetDrift            = "react_tool_call_budget_drift"
	ReasonCodeReactIterationBudgetDrift           = "react_iteration_budget_drift"
	ReasonCodeReactTerminationReasonDrift         = "react_termination_reason_drift"
	ReasonCodeReactStreamDispatchDrift            = "react_stream_dispatch_drift"
	ReasonCodeReactProviderMappingDrift           = "react_provider_mapping_drift"
	ReasonCodeSandboxEgressActionDrift            = "sandbox_egress_action_drift"
	ReasonCodeSandboxEgressPolicySourceDrift      = "sandbox_egress_policy_source_drift"
	ReasonCodeSandboxEgressViolationTaxonomyDrift = "sandbox_egress_violation_taxonomy_drift"
	ReasonCodeAdapterAllowlistDecisionDrift       = "adapter_allowlist_decision_drift"
	ReasonCodeAdapterAllowlistTaxonomyDrift       = "adapter_allowlist_taxonomy_drift"
	ReasonCodeBudgetThresholdDrift                = "budget_threshold_drift"
	ReasonCodeAdmissionDecisionDrift              = "admission_decision_drift"
	ReasonCodeDegradePolicyDrift                  = "degrade_policy_drift"
	ReasonCodeScopeResolutionDrift                = "scope_resolution_drift"
	ReasonCodeRetrievalQualityRegression          = "retrieval_quality_regression"
	ReasonCodeLifecyclePolicyDrift                = "lifecycle_policy_drift"
	ReasonCodeRecoveryConsistencyDrift            = "recovery_consistency_drift"
)

type ArbitrationFixture struct {
	Version string                   `json:"version"`
	Cases   []ArbitrationFixtureCase `json:"cases"`
}

type ArbitrationFixtureCase struct {
	Name        string                 `json:"name"`
	Run         ArbitrationObservation `json:"run"`
	Stream      ArbitrationObservation `json:"stream"`
	Expected    ArbitrationObservation `json:"expected"`
	Idempotency CompositeIdempotency   `json:"idempotency"`
	Additive    map[string]any         `json:"additive,omitempty"`
}

type ArbitrationObservation struct {
	RuntimePrimaryDomain                   string                    `json:"runtime_primary_domain"`
	RuntimePrimaryCode                     string                    `json:"runtime_primary_code"`
	RuntimePrimarySource                   string                    `json:"runtime_primary_source"`
	RuntimePrimaryConflictTotal            int                       `json:"runtime_primary_conflict_total"`
	RuntimeSecondaryReasonCodes            []string                  `json:"runtime_secondary_reason_codes,omitempty"`
	RuntimeSecondaryReasonCount            int                       `json:"runtime_secondary_reason_count,omitempty"`
	RuntimeArbitrationRuleVersion          string                    `json:"runtime_arbitration_rule_version,omitempty"`
	RuntimeArbitrationRuleRequestedVersion string                    `json:"runtime_arbitration_rule_requested_version,omitempty"`
	RuntimeArbitrationRuleEffectiveVersion string                    `json:"runtime_arbitration_rule_effective_version,omitempty"`
	RuntimeArbitrationRuleVersionSource    string                    `json:"runtime_arbitration_rule_version_source,omitempty"`
	RuntimeArbitrationRulePolicyAction     string                    `json:"runtime_arbitration_rule_policy_action,omitempty"`
	RuntimeArbitrationRuleUnsupportedTotal int                       `json:"runtime_arbitration_rule_unsupported_total,omitempty"`
	RuntimeArbitrationRuleMismatchTotal    int                       `json:"runtime_arbitration_rule_mismatch_total,omitempty"`
	RuntimeRemediationHintCode             string                    `json:"runtime_remediation_hint_code,omitempty"`
	RuntimeRemediationHintDomain           string                    `json:"runtime_remediation_hint_domain,omitempty"`
	PolicyPrecedenceVersion                string                    `json:"policy_precedence_version,omitempty"`
	WinnerStage                            string                    `json:"winner_stage,omitempty"`
	DenySource                             string                    `json:"deny_source,omitempty"`
	TieBreakReason                         string                    `json:"tie_break_reason,omitempty"`
	PolicyDecisionPath                     []PolicyDecisionPathEntry `json:"policy_decision_path,omitempty"`
	ModelProvider                          string                    `json:"model_provider,omitempty"`
	ReactEnabled                           bool                      `json:"react_enabled,omitempty"`
	ReactIterationTotal                    int                       `json:"react_iteration_total,omitempty"`
	ReactToolCallTotal                     int                       `json:"react_tool_call_total,omitempty"`
	ReactToolCallBudgetHitTotal            int                       `json:"react_tool_call_budget_hit_total,omitempty"`
	ReactIterationBudgetHitTotal           int                       `json:"react_iteration_budget_hit_total,omitempty"`
	ReactTerminationReason                 string                    `json:"react_termination_reason,omitempty"`
	ReactStreamDispatchEnabled             bool                      `json:"react_stream_dispatch_enabled,omitempty"`
	SandboxMode                            string                    `json:"sandbox_mode,omitempty"`
	SandboxBackend                         string                    `json:"sandbox_backend,omitempty"`
	SandboxProfile                         string                    `json:"sandbox_profile,omitempty"`
	SandboxSessionMode                     string                    `json:"sandbox_session_mode,omitempty"`
	SandboxRequiredCapabilities            []string                  `json:"sandbox_required_capabilities,omitempty"`
	SandboxDecision                        string                    `json:"sandbox_decision,omitempty"`
	SandboxReasonCode                      string                    `json:"sandbox_reason_code,omitempty"`
	SandboxFallbackUsed                    bool                      `json:"sandbox_fallback_used,omitempty"`
	SandboxFallbackReason                  string                    `json:"sandbox_fallback_reason,omitempty"`
	SandboxTimeoutTotal                    int                       `json:"sandbox_timeout_total,omitempty"`
	SandboxLaunchFailedTotal               int                       `json:"sandbox_launch_failed_total,omitempty"`
	SandboxCapabilityMismatchTotal         int                       `json:"sandbox_capability_mismatch_total,omitempty"`
	SandboxQueueWaitMsP95                  int64                     `json:"sandbox_queue_wait_ms_p95,omitempty"`
	SandboxExecLatencyMsP95                int64                     `json:"sandbox_exec_latency_ms_p95,omitempty"`
	SandboxExitCodeLast                    int                       `json:"sandbox_exit_code_last,omitempty"`
	SandboxOOMTotal                        int                       `json:"sandbox_oom_total,omitempty"`
	SandboxResourceCPUMsTotal              int64                     `json:"sandbox_resource_cpu_ms_total,omitempty"`
	SandboxResourceMemoryPeakBytesP95      int64                     `json:"sandbox_resource_memory_peak_bytes_p95,omitempty"`
	SandboxRolloutPhase                    string                    `json:"sandbox_rollout_phase,omitempty"`
	SandboxHealthBudgetStatus              string                    `json:"sandbox_health_budget_status,omitempty"`
	SandboxCapacityAction                  string                    `json:"sandbox_capacity_action,omitempty"`
	SandboxFreezeState                     bool                      `json:"sandbox_freeze_state,omitempty"`
	SandboxFreezeReasonCode                string                    `json:"sandbox_freeze_reason_code,omitempty"`
	SandboxEgressAction                    string                    `json:"sandbox_egress_action,omitempty"`
	SandboxEgressViolationTotal            int                       `json:"sandbox_egress_violation_total,omitempty"`
	SandboxEgressPolicySource              string                    `json:"sandbox_egress_policy_source,omitempty"`
	AdapterAllowlistDecision               string                    `json:"adapter_allowlist_decision,omitempty"`
	AdapterAllowlistBlockTotal             int                       `json:"adapter_allowlist_block_total,omitempty"`
	AdapterAllowlistPrimaryCode            string                    `json:"adapter_allowlist_primary_code,omitempty"`
	BudgetSnapshot                         *BudgetAdmissionSnapshot  `json:"budget_snapshot,omitempty"`
	BudgetDecision                         string                    `json:"budget_decision,omitempty"`
	DegradeAction                          string                    `json:"degrade_action,omitempty"`
	MemoryMode                             string                    `json:"memory_mode,omitempty"`
	MemoryProvider                         string                    `json:"memory_provider,omitempty"`
	MemoryProfile                          string                    `json:"memory_profile,omitempty"`
	MemoryContractVersion                  string                    `json:"memory_contract_version,omitempty"`
	MemoryQueryTotal                       int                       `json:"memory_query_total,omitempty"`
	MemoryUpsertTotal                      int                       `json:"memory_upsert_total,omitempty"`
	MemoryDeleteTotal                      int                       `json:"memory_delete_total,omitempty"`
	MemoryErrorTotal                       int                       `json:"memory_error_total,omitempty"`
	MemoryFallbackTotal                    int                       `json:"memory_fallback_total,omitempty"`
	MemoryFallbackReasonCode               string                    `json:"memory_fallback_reason_code,omitempty"`
	MemoryReasonCode                       string                    `json:"memory_reason_code,omitempty"`
	MemoryScopeSelected                    string                    `json:"memory_scope_selected,omitempty"`
	MemoryBudgetUsed                       int                       `json:"memory_budget_used,omitempty"`
	MemoryHits                             int                       `json:"memory_hits,omitempty"`
	MemoryRerankStats                      map[string]int            `json:"memory_rerank_stats,omitempty"`
	MemoryLifecycleAction                  string                    `json:"memory_lifecycle_action,omitempty"`
	ObservabilityExportProfile             string                    `json:"observability_export_profile,omitempty"`
	ObservabilityExportStatus              string                    `json:"observability_export_status,omitempty"`
	ObservabilityExportReasonCode          string                    `json:"observability_export_reason_code,omitempty"`
	DiagnosticsBundleLastStatus            string                    `json:"diagnostics_bundle_last_status,omitempty"`
	DiagnosticsBundleLastReasonCode        string                    `json:"diagnostics_bundle_last_reason_code,omitempty"`
	DiagnosticsBundleLastSchemaVersion     string                    `json:"diagnostics_bundle_last_schema_version,omitempty"`
	DiagnosticsBundleRedactionStatus       string                    `json:"diagnostics_bundle_redaction_status,omitempty"`
	DiagnosticsBundleGateFingerprint       string                    `json:"diagnostics_bundle_gate_fingerprint,omitempty"`
}

type PolicyDecisionPathEntry struct {
	Stage    string `json:"stage"`
	Code     string `json:"code,omitempty"`
	Source   string `json:"source,omitempty"`
	Decision string `json:"decision,omitempty"`
}

type BudgetAdmissionSnapshot struct {
	Version         string                         `json:"version,omitempty"`
	CostEstimate    BudgetAdmissionCostEstimate    `json:"cost_estimate,omitempty"`
	LatencyEstimate BudgetAdmissionLatencyEstimate `json:"latency_estimate,omitempty"`
}

type BudgetAdmissionCostEstimate struct {
	Token   float64 `json:"token,omitempty"`
	Tool    float64 `json:"tool,omitempty"`
	Sandbox float64 `json:"sandbox,omitempty"`
	Memory  float64 `json:"memory,omitempty"`
	Total   float64 `json:"total,omitempty"`
}

type BudgetAdmissionLatencyEstimate struct {
	TokenMs   int64 `json:"token_ms,omitempty"`
	ToolMs    int64 `json:"tool_ms,omitempty"`
	SandboxMs int64 `json:"sandbox_ms,omitempty"`
	MemoryMs  int64 `json:"memory_ms,omitempty"`
	TotalMs   int64 `json:"total_ms,omitempty"`
}

type ArbitrationReplayOutput struct {
	Version string                        `json:"version"`
	Cases   []ArbitrationNormalizedOutput `json:"cases"`
}

type ArbitrationNormalizedOutput struct {
	Name        string                 `json:"name"`
	Canonical   ArbitrationObservation `json:"canonical"`
	Idempotency CompositeIdempotency   `json:"idempotency"`
}

func ParseArbitrationFixtureJSON(raw []byte) (ArbitrationFixture, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var fixture ArbitrationFixture
	if err := dec.Decode(&fixture); err != nil {
		return ArbitrationFixture{}, &ValidationError{Code: ReasonCodeInvalidJSON, Message: err.Error()}
	}
	version := strings.TrimSpace(fixture.Version)
	if version == "" {
		return ArbitrationFixture{}, &ValidationError{Code: ReasonCodeSchemaMismatch, Message: "version is required"}
	}
	if version != ArbitrationFixtureVersionA48V1 &&
		version != ArbitrationFixtureVersionA49V1 &&
		version != ArbitrationFixtureVersionA50V1 &&
		version != ArbitrationFixtureVersionA51V1 &&
		version != ArbitrationFixtureVersionA52V1 &&
		version != ArbitrationFixtureVersionA57V1 &&
		version != ArbitrationFixtureVersionBudgetAdmissionV1 &&
		version != ArbitrationFixtureVersionPolicyV1 &&
		version != ArbitrationFixtureVersionMemoryV1 &&
		version != ArbitrationFixtureVersionMemoryScopeV1 &&
		version != ArbitrationFixtureVersionMemorySearchV1 &&
		version != ArbitrationFixtureVersionMemoryLifecycleV1 &&
		version != ArbitrationFixtureVersionObsV1 &&
		version != ArbitrationFixtureVersionReactV1 {
		return ArbitrationFixture{}, &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("unsupported fixture version %q", fixture.Version),
		}
	}
	if len(fixture.Cases) == 0 {
		return ArbitrationFixture{}, &ValidationError{Code: ReasonCodeSchemaMismatch, Message: "cases must not be empty"}
	}
	seen := map[string]struct{}{}
	for i := range fixture.Cases {
		name := strings.TrimSpace(fixture.Cases[i].Name)
		if name == "" {
			return ArbitrationFixture{}, &ValidationError{
				Code:    ReasonCodeSchemaMismatch,
				Message: fmt.Sprintf("cases[%d].name is required", i),
			}
		}
		if _, ok := seen[name]; ok {
			return ArbitrationFixture{}, &ValidationError{
				Code:    ReasonCodeSchemaMismatch,
				Message: fmt.Sprintf("duplicate case name %q", name),
			}
		}
		seen[name] = struct{}{}
	}
	fixture.Version = version
	return fixture, nil
}

func EvaluateArbitrationFixtureJSON(raw []byte) (ArbitrationReplayOutput, error) {
	fixture, err := ParseArbitrationFixtureJSON(raw)
	if err != nil {
		return ArbitrationReplayOutput{}, err
	}
	return EvaluateArbitrationFixture(fixture)
}

func EvaluateArbitrationFixture(fixture ArbitrationFixture) (ArbitrationReplayOutput, error) {
	version := strings.TrimSpace(fixture.Version)
	cases := append([]ArbitrationFixtureCase(nil), fixture.Cases...)
	sort.Slice(cases, func(i, j int) bool {
		return strings.TrimSpace(cases[i].Name) < strings.TrimSpace(cases[j].Name)
	})
	out := ArbitrationReplayOutput{
		Version: version,
		Cases:   make([]ArbitrationNormalizedOutput, 0, len(cases)),
	}
	for _, tc := range cases {
		name := strings.TrimSpace(tc.Name)
		expected := canonicalizeArbitrationObservation(tc.Expected)
		run := canonicalizeArbitrationObservation(tc.Run)
		stream := canonicalizeArbitrationObservation(tc.Stream)
		if err := validateArbitrationObservation(version, name, "expected", expected); err != nil {
			return ArbitrationReplayOutput{}, err
		}
		if err := validateArbitrationObservation(version, name, "run", run); err != nil {
			return ArbitrationReplayOutput{}, err
		}
		if err := validateArbitrationObservation(version, name, "stream", stream); err != nil {
			return ArbitrationReplayOutput{}, err
		}
		if err := assertArbitrationEquivalent(version, name, expected, run, "run"); err != nil {
			return ArbitrationReplayOutput{}, err
		}
		if err := assertArbitrationEquivalent(version, name, expected, stream, "stream"); err != nil {
			return ArbitrationReplayOutput{}, err
		}
		if err := assertArbitrationEquivalent(version, name, run, stream, "run/stream"); err != nil {
			return ArbitrationReplayOutput{}, err
		}
		if tc.Idempotency.FirstLogicalIngestTotal <= 0 {
			return ArbitrationReplayOutput{}, &ValidationError{
				Code:    ReasonCodeSchemaMismatch,
				Message: fmt.Sprintf("case %q idempotency.first_logical_ingest_total must be > 0", name),
			}
		}
		if tc.Idempotency.FirstLogicalIngestTotal != tc.Idempotency.ReplayLogicalIngestTotal {
			return ArbitrationReplayOutput{}, &ValidationError{
				Code: ReasonCodeSemanticDrift,
				Message: fmt.Sprintf(
					"case %q replay idempotency drift first=%d replay=%d",
					name,
					tc.Idempotency.FirstLogicalIngestTotal,
					tc.Idempotency.ReplayLogicalIngestTotal,
				),
			}
		}
		out.Cases = append(out.Cases, ArbitrationNormalizedOutput{
			Name:        name,
			Canonical:   expected,
			Idempotency: tc.Idempotency,
		})
	}
	return out, nil
}

func canonicalizeArbitrationObservation(in ArbitrationObservation) ArbitrationObservation {
	out := ArbitrationObservation{
		RuntimePrimaryDomain:                   strings.ToLower(strings.TrimSpace(in.RuntimePrimaryDomain)),
		RuntimePrimaryCode:                     strings.TrimSpace(in.RuntimePrimaryCode),
		RuntimePrimarySource:                   strings.ToLower(strings.TrimSpace(in.RuntimePrimarySource)),
		RuntimePrimaryConflictTotal:            in.RuntimePrimaryConflictTotal,
		RuntimeSecondaryReasonCount:            in.RuntimeSecondaryReasonCount,
		RuntimeArbitrationRuleVersion:          strings.ToLower(strings.TrimSpace(in.RuntimeArbitrationRuleVersion)),
		RuntimeArbitrationRuleRequestedVersion: strings.ToLower(strings.TrimSpace(in.RuntimeArbitrationRuleRequestedVersion)),
		RuntimeArbitrationRuleEffectiveVersion: strings.ToLower(strings.TrimSpace(in.RuntimeArbitrationRuleEffectiveVersion)),
		RuntimeArbitrationRuleVersionSource:    strings.ToLower(strings.TrimSpace(in.RuntimeArbitrationRuleVersionSource)),
		RuntimeArbitrationRulePolicyAction:     strings.ToLower(strings.TrimSpace(in.RuntimeArbitrationRulePolicyAction)),
		RuntimeArbitrationRuleUnsupportedTotal: in.RuntimeArbitrationRuleUnsupportedTotal,
		RuntimeArbitrationRuleMismatchTotal:    in.RuntimeArbitrationRuleMismatchTotal,
		RuntimeRemediationHintCode:             strings.TrimSpace(in.RuntimeRemediationHintCode),
		RuntimeRemediationHintDomain:           strings.ToLower(strings.TrimSpace(in.RuntimeRemediationHintDomain)),
		PolicyPrecedenceVersion:                strings.ToLower(strings.TrimSpace(in.PolicyPrecedenceVersion)),
		WinnerStage:                            strings.ToLower(strings.TrimSpace(in.WinnerStage)),
		DenySource:                             strings.ToLower(strings.TrimSpace(in.DenySource)),
		TieBreakReason:                         strings.ToLower(strings.TrimSpace(in.TieBreakReason)),
		ModelProvider:                          strings.ToLower(strings.TrimSpace(in.ModelProvider)),
		ReactEnabled:                           in.ReactEnabled,
		ReactIterationTotal:                    in.ReactIterationTotal,
		ReactToolCallTotal:                     in.ReactToolCallTotal,
		ReactToolCallBudgetHitTotal:            in.ReactToolCallBudgetHitTotal,
		ReactIterationBudgetHitTotal:           in.ReactIterationBudgetHitTotal,
		ReactTerminationReason:                 strings.ToLower(strings.TrimSpace(in.ReactTerminationReason)),
		ReactStreamDispatchEnabled:             in.ReactStreamDispatchEnabled,
		SandboxMode:                            strings.ToLower(strings.TrimSpace(in.SandboxMode)),
		SandboxBackend:                         strings.ToLower(strings.TrimSpace(in.SandboxBackend)),
		SandboxProfile:                         strings.ToLower(strings.TrimSpace(in.SandboxProfile)),
		SandboxSessionMode:                     strings.ToLower(strings.TrimSpace(in.SandboxSessionMode)),
		SandboxDecision:                        strings.ToLower(strings.TrimSpace(in.SandboxDecision)),
		SandboxReasonCode:                      strings.ToLower(strings.TrimSpace(in.SandboxReasonCode)),
		SandboxFallbackUsed:                    in.SandboxFallbackUsed,
		SandboxFallbackReason:                  strings.ToLower(strings.TrimSpace(in.SandboxFallbackReason)),
		SandboxTimeoutTotal:                    in.SandboxTimeoutTotal,
		SandboxLaunchFailedTotal:               in.SandboxLaunchFailedTotal,
		SandboxCapabilityMismatchTotal:         in.SandboxCapabilityMismatchTotal,
		SandboxQueueWaitMsP95:                  in.SandboxQueueWaitMsP95,
		SandboxExecLatencyMsP95:                in.SandboxExecLatencyMsP95,
		SandboxExitCodeLast:                    in.SandboxExitCodeLast,
		SandboxOOMTotal:                        in.SandboxOOMTotal,
		SandboxResourceCPUMsTotal:              in.SandboxResourceCPUMsTotal,
		SandboxResourceMemoryPeakBytesP95:      in.SandboxResourceMemoryPeakBytesP95,
		SandboxRolloutPhase:                    strings.ToLower(strings.TrimSpace(in.SandboxRolloutPhase)),
		SandboxHealthBudgetStatus:              strings.ToLower(strings.TrimSpace(in.SandboxHealthBudgetStatus)),
		SandboxCapacityAction:                  strings.ToLower(strings.TrimSpace(in.SandboxCapacityAction)),
		SandboxFreezeState:                     in.SandboxFreezeState,
		SandboxFreezeReasonCode:                strings.ToLower(strings.TrimSpace(in.SandboxFreezeReasonCode)),
		SandboxEgressAction:                    strings.ToLower(strings.TrimSpace(in.SandboxEgressAction)),
		SandboxEgressViolationTotal:            in.SandboxEgressViolationTotal,
		SandboxEgressPolicySource:              strings.ToLower(strings.TrimSpace(in.SandboxEgressPolicySource)),
		AdapterAllowlistDecision:               strings.ToLower(strings.TrimSpace(in.AdapterAllowlistDecision)),
		AdapterAllowlistBlockTotal:             in.AdapterAllowlistBlockTotal,
		AdapterAllowlistPrimaryCode:            strings.TrimSpace(in.AdapterAllowlistPrimaryCode),
		BudgetDecision:                         strings.ToLower(strings.TrimSpace(in.BudgetDecision)),
		DegradeAction:                          strings.ToLower(strings.TrimSpace(in.DegradeAction)),
		MemoryMode:                             strings.ToLower(strings.TrimSpace(in.MemoryMode)),
		MemoryProvider:                         strings.ToLower(strings.TrimSpace(in.MemoryProvider)),
		MemoryProfile:                          strings.ToLower(strings.TrimSpace(in.MemoryProfile)),
		MemoryContractVersion:                  strings.ToLower(strings.TrimSpace(in.MemoryContractVersion)),
		MemoryQueryTotal:                       in.MemoryQueryTotal,
		MemoryUpsertTotal:                      in.MemoryUpsertTotal,
		MemoryDeleteTotal:                      in.MemoryDeleteTotal,
		MemoryErrorTotal:                       in.MemoryErrorTotal,
		MemoryFallbackTotal:                    in.MemoryFallbackTotal,
		MemoryFallbackReasonCode:               strings.ToLower(strings.TrimSpace(in.MemoryFallbackReasonCode)),
		MemoryReasonCode:                       strings.ToLower(strings.TrimSpace(in.MemoryReasonCode)),
		MemoryScopeSelected:                    strings.ToLower(strings.TrimSpace(in.MemoryScopeSelected)),
		MemoryBudgetUsed:                       in.MemoryBudgetUsed,
		MemoryHits:                             in.MemoryHits,
		MemoryLifecycleAction:                  strings.ToLower(strings.TrimSpace(in.MemoryLifecycleAction)),
		ObservabilityExportProfile:             strings.ToLower(strings.TrimSpace(in.ObservabilityExportProfile)),
		ObservabilityExportStatus:              strings.ToLower(strings.TrimSpace(in.ObservabilityExportStatus)),
		ObservabilityExportReasonCode:          strings.ToLower(strings.TrimSpace(in.ObservabilityExportReasonCode)),
		DiagnosticsBundleLastStatus:            strings.ToLower(strings.TrimSpace(in.DiagnosticsBundleLastStatus)),
		DiagnosticsBundleLastReasonCode:        strings.ToLower(strings.TrimSpace(in.DiagnosticsBundleLastReasonCode)),
		DiagnosticsBundleLastSchemaVersion:     strings.ToLower(strings.TrimSpace(in.DiagnosticsBundleLastSchemaVersion)),
		DiagnosticsBundleRedactionStatus:       strings.ToLower(strings.TrimSpace(in.DiagnosticsBundleRedactionStatus)),
		DiagnosticsBundleGateFingerprint:       strings.ToLower(strings.TrimSpace(in.DiagnosticsBundleGateFingerprint)),
	}
	if out.RuntimePrimaryConflictTotal < 0 {
		out.RuntimePrimaryConflictTotal = 0
	}
	if out.RuntimeSecondaryReasonCount < 0 {
		out.RuntimeSecondaryReasonCount = 0
	}
	if out.RuntimeArbitrationRuleUnsupportedTotal < 0 {
		out.RuntimeArbitrationRuleUnsupportedTotal = 0
	}
	if out.RuntimeArbitrationRuleMismatchTotal < 0 {
		out.RuntimeArbitrationRuleMismatchTotal = 0
	}
	if out.ReactIterationTotal < 0 {
		out.ReactIterationTotal = 0
	}
	if out.ReactToolCallTotal < 0 {
		out.ReactToolCallTotal = 0
	}
	if out.ReactToolCallBudgetHitTotal < 0 {
		out.ReactToolCallBudgetHitTotal = 0
	}
	if out.ReactIterationBudgetHitTotal < 0 {
		out.ReactIterationBudgetHitTotal = 0
	}
	if out.SandboxTimeoutTotal < 0 {
		out.SandboxTimeoutTotal = 0
	}
	if out.SandboxLaunchFailedTotal < 0 {
		out.SandboxLaunchFailedTotal = 0
	}
	if out.SandboxCapabilityMismatchTotal < 0 {
		out.SandboxCapabilityMismatchTotal = 0
	}
	if out.SandboxQueueWaitMsP95 < 0 {
		out.SandboxQueueWaitMsP95 = 0
	}
	if out.SandboxExecLatencyMsP95 < 0 {
		out.SandboxExecLatencyMsP95 = 0
	}
	if out.SandboxOOMTotal < 0 {
		out.SandboxOOMTotal = 0
	}
	if out.SandboxResourceCPUMsTotal < 0 {
		out.SandboxResourceCPUMsTotal = 0
	}
	if out.SandboxResourceMemoryPeakBytesP95 < 0 {
		out.SandboxResourceMemoryPeakBytesP95 = 0
	}
	if out.SandboxEgressViolationTotal < 0 {
		out.SandboxEgressViolationTotal = 0
	}
	if out.AdapterAllowlistBlockTotal < 0 {
		out.AdapterAllowlistBlockTotal = 0
	}
	if out.MemoryQueryTotal < 0 {
		out.MemoryQueryTotal = 0
	}
	if out.MemoryUpsertTotal < 0 {
		out.MemoryUpsertTotal = 0
	}
	if out.MemoryDeleteTotal < 0 {
		out.MemoryDeleteTotal = 0
	}
	if out.MemoryErrorTotal < 0 {
		out.MemoryErrorTotal = 0
	}
	if out.MemoryFallbackTotal < 0 {
		out.MemoryFallbackTotal = 0
	}
	if out.MemoryBudgetUsed < 0 {
		out.MemoryBudgetUsed = 0
	}
	if out.MemoryHits < 0 {
		out.MemoryHits = 0
	}
	for key, value := range in.MemoryRerankStats {
		normalizedKey := strings.ToLower(strings.TrimSpace(key))
		if normalizedKey == "" {
			continue
		}
		if value < 0 {
			value = 0
		}
		if out.MemoryRerankStats == nil {
			out.MemoryRerankStats = map[string]int{}
		}
		out.MemoryRerankStats[normalizedKey] = value
	}
	if len(out.MemoryRerankStats) == 0 {
		out.MemoryRerankStats = nil
	}
	for i := range in.RuntimeSecondaryReasonCodes {
		code := strings.TrimSpace(in.RuntimeSecondaryReasonCodes[i])
		if code == "" {
			continue
		}
		out.RuntimeSecondaryReasonCodes = append(out.RuntimeSecondaryReasonCodes, code)
	}
	if len(out.RuntimeSecondaryReasonCodes) == 0 {
		out.RuntimeSecondaryReasonCodes = nil
	}
	for i := range in.SandboxRequiredCapabilities {
		item := strings.ToLower(strings.TrimSpace(in.SandboxRequiredCapabilities[i]))
		if item == "" {
			continue
		}
		out.SandboxRequiredCapabilities = append(out.SandboxRequiredCapabilities, item)
	}
	if len(out.SandboxRequiredCapabilities) == 0 {
		out.SandboxRequiredCapabilities = nil
	}
	for i := range in.PolicyDecisionPath {
		item := in.PolicyDecisionPath[i]
		stage := strings.ToLower(strings.TrimSpace(item.Stage))
		if stage == "" {
			continue
		}
		out.PolicyDecisionPath = append(out.PolicyDecisionPath, PolicyDecisionPathEntry{
			Stage:    stage,
			Code:     strings.TrimSpace(item.Code),
			Source:   strings.ToLower(strings.TrimSpace(item.Source)),
			Decision: strings.ToLower(strings.TrimSpace(item.Decision)),
		})
	}
	if len(out.PolicyDecisionPath) == 0 {
		out.PolicyDecisionPath = nil
	}
	out.BudgetSnapshot = canonicalizeBudgetAdmissionSnapshot(in.BudgetSnapshot)
	return out
}

func validateArbitrationObservation(version, caseName, lane string, obs ArbitrationObservation) error {
	if version == ArbitrationFixtureVersionObsV1 {
		return validateObservabilityArbitrationObservation(caseName, lane, obs)
	}
	if version == ArbitrationFixtureVersionMemoryV1 {
		return validateMemoryArbitrationObservation(caseName, lane, obs)
	}
	if version == ArbitrationFixtureVersionMemoryScopeV1 {
		return validateMemoryScopeArbitrationObservation(caseName, lane, obs)
	}
	if version == ArbitrationFixtureVersionMemorySearchV1 {
		return validateMemorySearchArbitrationObservation(caseName, lane, obs)
	}
	if version == ArbitrationFixtureVersionMemoryLifecycleV1 {
		return validateMemoryLifecycleArbitrationObservation(caseName, lane, obs)
	}
	if version == ArbitrationFixtureVersionReactV1 {
		return validateReactArbitrationObservation(caseName, lane, obs)
	}
	if version == ArbitrationFixtureVersionA57V1 {
		return validateSandboxEgressArbitrationObservation(caseName, lane, obs)
	}
	if version == ArbitrationFixtureVersionBudgetAdmissionV1 {
		return validateBudgetAdmissionArbitrationObservation(caseName, lane, obs)
	}
	if version == ArbitrationFixtureVersionPolicyV1 {
		return validatePolicyStackArbitrationObservation(caseName, lane, obs)
	}
	if strings.TrimSpace(obs.RuntimePrimaryDomain) == "" {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s.runtime_primary_domain is required", caseName, lane),
		}
	}
	if strings.TrimSpace(obs.RuntimePrimarySource) == "" {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s.runtime_primary_source is required", caseName, lane),
		}
	}
	if !isCanonicalArbitrationCode(obs.RuntimePrimaryCode) {
		return &ValidationError{
			Code:    ReasonCodeTaxonomyDrift,
			Message: fmt.Sprintf("case %q %s non-canonical primary code %q", caseName, lane, obs.RuntimePrimaryCode),
		}
	}
	if version == ArbitrationFixtureVersionA48V1 {
		return nil
	}
	if version == ArbitrationFixtureVersionA49V1 {
		if strings.TrimSpace(obs.RuntimeArbitrationRuleVersion) == "" {
			return &ValidationError{
				Code:    ReasonCodeSchemaMismatch,
				Message: fmt.Sprintf("case %q %s.runtime_arbitration_rule_version is required", caseName, lane),
			}
		}
		if strings.TrimSpace(obs.RuntimeArbitrationRuleVersion) != runtimeconfig.RuntimeArbitrationRuleVersionA49V1 {
			return &ValidationError{
				Code:    ReasonCodeRuleVersionDrift,
				Message: fmt.Sprintf("case %q %s rule version drift want=%q got=%q", caseName, lane, runtimeconfig.RuntimeArbitrationRuleVersionA49V1, obs.RuntimeArbitrationRuleVersion),
			}
		}
	}
	if len(obs.RuntimeSecondaryReasonCodes) > runtimeconfig.RuntimeArbitrationMaxSecondary {
		return &ValidationError{
			Code:    ReasonCodeSecondaryCountDrift,
			Message: fmt.Sprintf("case %q %s runtime_secondary_reason_codes exceeds max=%d", caseName, lane, runtimeconfig.RuntimeArbitrationMaxSecondary),
		}
	}
	if obs.RuntimeSecondaryReasonCount < len(obs.RuntimeSecondaryReasonCodes) {
		return &ValidationError{
			Code:    ReasonCodeSecondaryCountDrift,
			Message: fmt.Sprintf("case %q %s runtime_secondary_reason_count=%d < len(codes)=%d", caseName, lane, obs.RuntimeSecondaryReasonCount, len(obs.RuntimeSecondaryReasonCodes)),
		}
	}
	seenSecondary := map[string]struct{}{}
	for i := range obs.RuntimeSecondaryReasonCodes {
		secondary := strings.TrimSpace(obs.RuntimeSecondaryReasonCodes[i])
		if !isCanonicalArbitrationCode(secondary) {
			return &ValidationError{
				Code:    ReasonCodeTaxonomyDrift,
				Message: fmt.Sprintf("case %q %s non-canonical secondary code %q", caseName, lane, secondary),
			}
		}
		if secondary == obs.RuntimePrimaryCode {
			return &ValidationError{
				Code:    ReasonCodeSecondaryOrderDrift,
				Message: fmt.Sprintf("case %q %s secondary code repeats primary %q", caseName, lane, secondary),
			}
		}
		if _, ok := seenSecondary[secondary]; ok {
			return &ValidationError{
				Code:    ReasonCodeSecondaryOrderDrift,
				Message: fmt.Sprintf("case %q %s duplicate secondary code %q", caseName, lane, secondary),
			}
		}
		seenSecondary[secondary] = struct{}{}
	}
	hintCode, hintDomain, ok := runtimeconfig.RemediationHintForPrimaryCode(obs.RuntimePrimaryCode)
	if !ok {
		return &ValidationError{
			Code:    ReasonCodeHintTaxonomyDrift,
			Message: fmt.Sprintf("case %q %s unsupported primary for hint taxonomy %q", caseName, lane, obs.RuntimePrimaryCode),
		}
	}
	if strings.TrimSpace(obs.RuntimeRemediationHintCode) == "" || strings.TrimSpace(obs.RuntimeRemediationHintDomain) == "" {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s remediation hint fields are required", caseName, lane),
		}
	}
	if strings.TrimSpace(obs.RuntimeRemediationHintCode) != hintCode || strings.TrimSpace(obs.RuntimeRemediationHintDomain) != hintDomain {
		return &ValidationError{
			Code:    ReasonCodeHintTaxonomyDrift,
			Message: fmt.Sprintf("case %q %s hint taxonomy drift want=%s/%s got=%s/%s", caseName, lane, hintDomain, hintCode, obs.RuntimeRemediationHintDomain, obs.RuntimeRemediationHintCode),
		}
	}
	if version != ArbitrationFixtureVersionA50V1 && version != ArbitrationFixtureVersionA51V1 && version != ArbitrationFixtureVersionA52V1 {
		return nil
	}
	if obs.RuntimeArbitrationRuleVersionSource != runtimeconfig.RuntimeArbitrationVersionSourceDefault &&
		obs.RuntimeArbitrationRuleVersionSource != runtimeconfig.RuntimeArbitrationVersionSourceRequested {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s runtime_arbitration_rule_version_source must be default|requested", caseName, lane),
		}
	}
	switch obs.RuntimeArbitrationRulePolicyAction {
	case runtimeconfig.RuntimeArbitrationPolicyActionNone,
		runtimeconfig.RuntimeArbitrationPolicyActionDisabled,
		runtimeconfig.RuntimeArbitrationPolicyActionFailFastUnsupported,
		runtimeconfig.RuntimeArbitrationPolicyActionFailFastMismatch:
	default:
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s runtime_arbitration_rule_policy_action is invalid", caseName, lane),
		}
	}
	if effective := strings.TrimSpace(obs.RuntimeArbitrationRuleEffectiveVersion); effective != "" && !isSupportedA50Version(effective) {
		return &ValidationError{
			Code:    ReasonCodeRuleVersionDrift,
			Message: fmt.Sprintf("case %q %s runtime_arbitration_rule_effective_version=%q is not registered", caseName, lane, effective),
		}
	}
	if ruleVersion := strings.TrimSpace(obs.RuntimeArbitrationRuleVersion); ruleVersion != "" {
		if !isSupportedA50Version(ruleVersion) {
			return &ValidationError{
				Code:    ReasonCodeRuleVersionDrift,
				Message: fmt.Sprintf("case %q %s runtime_arbitration_rule_version=%q is not registered", caseName, lane, ruleVersion),
			}
		}
		if effective := strings.TrimSpace(obs.RuntimeArbitrationRuleEffectiveVersion); effective != "" && effective != ruleVersion {
			return &ValidationError{
				Code:    ReasonCodeCrossVersionSemanticDrift,
				Message: fmt.Sprintf("case %q %s runtime_arbitration_rule_version=%q mismatches effective=%q", caseName, lane, ruleVersion, effective),
			}
		}
	}
	if requested := strings.TrimSpace(obs.RuntimeArbitrationRuleRequestedVersion); requested != "" &&
		!isSupportedA50Version(requested) &&
		obs.RuntimePrimaryCode != runtimeconfig.ReadinessCodeArbitrationVersionUnsupported {
		return &ValidationError{
			Code:    ReasonCodeUnsupportedVersion,
			Message: fmt.Sprintf("case %q %s requested version %q must produce unsupported classification", caseName, lane, requested),
		}
	}
	switch strings.TrimSpace(obs.RuntimePrimaryCode) {
	case runtimeconfig.ReadinessCodeArbitrationVersionUnsupported:
		if obs.RuntimeArbitrationRulePolicyAction != runtimeconfig.RuntimeArbitrationPolicyActionFailFastUnsupported ||
			obs.RuntimeArbitrationRuleUnsupportedTotal <= 0 {
			return &ValidationError{
				Code:    ReasonCodeUnsupportedVersion,
				Message: fmt.Sprintf("case %q %s unsupported version must set fail_fast_unsupported_version and unsupported_total>0", caseName, lane),
			}
		}
	case runtimeconfig.ReadinessCodeArbitrationVersionMismatch:
		if obs.RuntimeArbitrationRulePolicyAction != runtimeconfig.RuntimeArbitrationPolicyActionFailFastMismatch ||
			obs.RuntimeArbitrationRuleMismatchTotal <= 0 {
			return &ValidationError{
				Code:    ReasonCodeVersionMismatch,
				Message: fmt.Sprintf("case %q %s version mismatch must set fail_fast_version_mismatch and mismatch_total>0", caseName, lane),
			}
		}
	default:
		if obs.RuntimeArbitrationRuleUnsupportedTotal > 0 || obs.RuntimeArbitrationRuleMismatchTotal > 0 {
			return &ValidationError{
				Code:    ReasonCodeCrossVersionSemanticDrift,
				Message: fmt.Sprintf("case %q %s non-version-failure code has unsupported/mismatch counters", caseName, lane),
			}
		}
		if strings.TrimSpace(obs.RuntimeArbitrationRuleEffectiveVersion) == "" {
			return &ValidationError{
				Code:    ReasonCodeCrossVersionSemanticDrift,
				Message: fmt.Sprintf("case %q %s effective version is required for non-version-failure paths", caseName, lane),
			}
		}
	}
	if version == ArbitrationFixtureVersionA51V1 || version == ArbitrationFixtureVersionA52V1 {
		if err := validateSandboxArbitrationObservation(caseName, lane, obs); err != nil {
			return err
		}
	}
	if version == ArbitrationFixtureVersionA52V1 {
		if err := validateSandboxRolloutArbitrationObservation(caseName, lane, obs); err != nil {
			return err
		}
	}
	return nil
}

func assertArbitrationEquivalent(version, caseName string, expected, actual ArbitrationObservation, lane string) error {
	if arbitrationObservationsEqual(version, expected, actual) {
		return nil
	}
	if version == ArbitrationFixtureVersionReactV1 {
		return assertReactArbitrationEquivalent(caseName, lane, expected, actual)
	}
	if version == ArbitrationFixtureVersionObsV1 {
		return assertObservabilityArbitrationEquivalent(caseName, lane, expected, actual)
	}
	if version == ArbitrationFixtureVersionMemoryV1 {
		return assertMemoryArbitrationEquivalent(caseName, lane, expected, actual)
	}
	if version == ArbitrationFixtureVersionMemoryScopeV1 {
		return assertMemoryScopeArbitrationEquivalent(caseName, lane, expected, actual)
	}
	if version == ArbitrationFixtureVersionMemorySearchV1 {
		return assertMemorySearchArbitrationEquivalent(caseName, lane, expected, actual)
	}
	if version == ArbitrationFixtureVersionMemoryLifecycleV1 {
		return assertMemoryLifecycleArbitrationEquivalent(caseName, lane, expected, actual)
	}
	if version == ArbitrationFixtureVersionA57V1 {
		return assertSandboxEgressArbitrationEquivalent(caseName, lane, expected, actual)
	}
	if version == ArbitrationFixtureVersionBudgetAdmissionV1 {
		return assertBudgetAdmissionArbitrationEquivalent(caseName, lane, expected, actual)
	}
	if version == ArbitrationFixtureVersionPolicyV1 {
		return assertPolicyStackArbitrationEquivalent(caseName, lane, expected, actual)
	}
	if version == ArbitrationFixtureVersionA50V1 || version == ArbitrationFixtureVersionA51V1 || version == ArbitrationFixtureVersionA52V1 {
		if expected.RuntimePrimaryCode != actual.RuntimePrimaryCode {
			if expected.RuntimePrimaryCode == runtimeconfig.ReadinessCodeArbitrationVersionUnsupported ||
				actual.RuntimePrimaryCode == runtimeconfig.ReadinessCodeArbitrationVersionUnsupported {
				return &ValidationError{
					Code: ReasonCodeUnsupportedVersion,
					Message: fmt.Sprintf(
						"case %q %s unsupported version classification drift expected=%q actual=%q",
						caseName,
						lane,
						expected.RuntimePrimaryCode,
						actual.RuntimePrimaryCode,
					),
				}
			}
			if expected.RuntimePrimaryCode == runtimeconfig.ReadinessCodeArbitrationVersionMismatch ||
				actual.RuntimePrimaryCode == runtimeconfig.ReadinessCodeArbitrationVersionMismatch {
				return &ValidationError{
					Code: ReasonCodeVersionMismatch,
					Message: fmt.Sprintf(
						"case %q %s version mismatch classification drift expected=%q actual=%q",
						caseName,
						lane,
						expected.RuntimePrimaryCode,
						actual.RuntimePrimaryCode,
					),
				}
			}
		}
		if expected.RuntimeArbitrationRuleVersion != actual.RuntimeArbitrationRuleVersion ||
			expected.RuntimeArbitrationRuleRequestedVersion != actual.RuntimeArbitrationRuleRequestedVersion ||
			expected.RuntimeArbitrationRuleEffectiveVersion != actual.RuntimeArbitrationRuleEffectiveVersion ||
			expected.RuntimeArbitrationRuleVersionSource != actual.RuntimeArbitrationRuleVersionSource ||
			expected.RuntimeArbitrationRulePolicyAction != actual.RuntimeArbitrationRulePolicyAction ||
			expected.RuntimeArbitrationRuleUnsupportedTotal != actual.RuntimeArbitrationRuleUnsupportedTotal ||
			expected.RuntimeArbitrationRuleMismatchTotal != actual.RuntimeArbitrationRuleMismatchTotal {
			if expected.RuntimePrimaryCode == runtimeconfig.ReadinessCodeArbitrationVersionUnsupported ||
				actual.RuntimePrimaryCode == runtimeconfig.ReadinessCodeArbitrationVersionUnsupported {
				return &ValidationError{
					Code: ReasonCodeUnsupportedVersion,
					Message: fmt.Sprintf(
						"case %q %s unsupported version governance drift expected=%#v actual=%#v",
						caseName,
						lane,
						expected,
						actual,
					),
				}
			}
			if expected.RuntimePrimaryCode == runtimeconfig.ReadinessCodeArbitrationVersionMismatch ||
				actual.RuntimePrimaryCode == runtimeconfig.ReadinessCodeArbitrationVersionMismatch {
				return &ValidationError{
					Code: ReasonCodeVersionMismatch,
					Message: fmt.Sprintf(
						"case %q %s version mismatch governance drift expected=%#v actual=%#v",
						caseName,
						lane,
						expected,
						actual,
					),
				}
			}
			return &ValidationError{
				Code: ReasonCodeCrossVersionSemanticDrift,
				Message: fmt.Sprintf(
					"case %q %s cross-version semantic drift expected=%#v actual=%#v",
					caseName,
					lane,
					expected,
					actual,
				),
			}
		}
	}
	if version == ArbitrationFixtureVersionA51V1 || version == ArbitrationFixtureVersionA52V1 {
		if err := assertSandboxArbitrationEquivalent(caseName, lane, expected, actual); err != nil {
			return err
		}
	}
	if version == ArbitrationFixtureVersionA52V1 {
		if err := assertSandboxRolloutArbitrationEquivalent(caseName, lane, expected, actual); err != nil {
			return err
		}
	}
	if precedenceForArbitrationCode(expected.RuntimePrimaryCode) != precedenceForArbitrationCode(actual.RuntimePrimaryCode) {
		return &ValidationError{
			Code: ReasonCodePrecedenceDrift,
			Message: fmt.Sprintf(
				"case %q %s precedence drift expected=%q actual=%q",
				caseName,
				lane,
				expected.RuntimePrimaryCode,
				actual.RuntimePrimaryCode,
			),
		}
	}
	if version == ArbitrationFixtureVersionA49V1 {
		if expected.RuntimeArbitrationRuleVersion != actual.RuntimeArbitrationRuleVersion {
			return &ValidationError{
				Code: ReasonCodeRuleVersionDrift,
				Message: fmt.Sprintf(
					"case %q %s rule version drift expected=%q actual=%q",
					caseName,
					lane,
					expected.RuntimeArbitrationRuleVersion,
					actual.RuntimeArbitrationRuleVersion,
				),
			}
		}
		if expected.RuntimeSecondaryReasonCount != actual.RuntimeSecondaryReasonCount {
			return &ValidationError{
				Code: ReasonCodeSecondaryCountDrift,
				Message: fmt.Sprintf(
					"case %q %s secondary count drift expected=%d actual=%d",
					caseName,
					lane,
					expected.RuntimeSecondaryReasonCount,
					actual.RuntimeSecondaryReasonCount,
				),
			}
		}
		if !equalStringSlice(expected.RuntimeSecondaryReasonCodes, actual.RuntimeSecondaryReasonCodes) {
			return &ValidationError{
				Code: ReasonCodeSecondaryOrderDrift,
				Message: fmt.Sprintf(
					"case %q %s secondary order drift expected=%#v actual=%#v",
					caseName,
					lane,
					expected.RuntimeSecondaryReasonCodes,
					actual.RuntimeSecondaryReasonCodes,
				),
			}
		}
	}
	if version == ArbitrationFixtureVersionA50V1 || version == ArbitrationFixtureVersionA51V1 || version == ArbitrationFixtureVersionA52V1 {
		if expected.RuntimeSecondaryReasonCount != actual.RuntimeSecondaryReasonCount {
			return &ValidationError{
				Code: ReasonCodeSecondaryCountDrift,
				Message: fmt.Sprintf(
					"case %q %s secondary count drift expected=%d actual=%d",
					caseName,
					lane,
					expected.RuntimeSecondaryReasonCount,
					actual.RuntimeSecondaryReasonCount,
				),
			}
		}
		if !equalStringSlice(expected.RuntimeSecondaryReasonCodes, actual.RuntimeSecondaryReasonCodes) {
			return &ValidationError{
				Code: ReasonCodeSecondaryOrderDrift,
				Message: fmt.Sprintf(
					"case %q %s secondary order drift expected=%#v actual=%#v",
					caseName,
					lane,
					expected.RuntimeSecondaryReasonCodes,
					actual.RuntimeSecondaryReasonCodes,
				),
			}
		}
	}
	if expected.RuntimePrimaryCode != actual.RuntimePrimaryCode ||
		expected.RuntimePrimaryConflictTotal != actual.RuntimePrimaryConflictTotal {
		return &ValidationError{
			Code: ReasonCodeTieBreakDrift,
			Message: fmt.Sprintf(
				"case %q %s tie-break drift expected=%#v actual=%#v",
				caseName,
				lane,
				expected,
				actual,
			),
		}
	}
	if (version == ArbitrationFixtureVersionA49V1 || version == ArbitrationFixtureVersionA50V1 || version == ArbitrationFixtureVersionA51V1 || version == ArbitrationFixtureVersionA52V1) &&
		(expected.RuntimeRemediationHintCode != actual.RuntimeRemediationHintCode ||
			expected.RuntimeRemediationHintDomain != actual.RuntimeRemediationHintDomain) {
		return &ValidationError{
			Code: ReasonCodeHintTaxonomyDrift,
			Message: fmt.Sprintf(
				"case %q %s hint taxonomy drift expected=%s/%s actual=%s/%s",
				caseName,
				lane,
				expected.RuntimeRemediationHintDomain,
				expected.RuntimeRemediationHintCode,
				actual.RuntimeRemediationHintDomain,
				actual.RuntimeRemediationHintCode,
			),
		}
	}
	return &ValidationError{
		Code: ReasonCodeTaxonomyDrift,
		Message: fmt.Sprintf(
			"case %q %s taxonomy drift expected=%#v actual=%#v",
			caseName,
			lane,
			expected,
			actual,
		),
	}
}

func validatePolicyStackArbitrationObservation(caseName, lane string, obs ArbitrationObservation) error {
	if strings.TrimSpace(obs.PolicyPrecedenceVersion) != runtimeconfig.RuntimePolicyPrecedenceVersionPolicyStackV1 {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s policy_precedence_version must be %q", caseName, lane, runtimeconfig.RuntimePolicyPrecedenceVersionPolicyStackV1),
		}
	}
	if !isCanonicalPolicyStage(obs.WinnerStage) {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s winner_stage is invalid: %q", caseName, lane, obs.WinnerStage),
		}
	}
	if len(obs.PolicyDecisionPath) == 0 {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s policy_decision_path must not be empty", caseName, lane),
		}
	}
	winnerSeen := false
	denySeen := false
	for i := range obs.PolicyDecisionPath {
		item := obs.PolicyDecisionPath[i]
		if !isCanonicalPolicyStage(item.Stage) {
			return &ValidationError{
				Code:    ReasonCodeSchemaMismatch,
				Message: fmt.Sprintf("case %q %s policy_decision_path[%d].stage is invalid: %q", caseName, lane, i, item.Stage),
			}
		}
		switch strings.TrimSpace(item.Decision) {
		case runtimeconfig.RuntimePolicyDecisionAllow:
		case runtimeconfig.RuntimePolicyDecisionDeny:
			denySeen = true
		default:
			return &ValidationError{
				Code:    ReasonCodeSchemaMismatch,
				Message: fmt.Sprintf("case %q %s policy_decision_path[%d].decision must be allow|deny", caseName, lane, i),
			}
		}
		if item.Stage == obs.WinnerStage {
			winnerSeen = true
		}
	}
	if !winnerSeen {
		return &ValidationError{
			Code:    ReasonCodePrecedenceConflict,
			Message: fmt.Sprintf("case %q %s winner_stage=%q not found in policy_decision_path", caseName, lane, obs.WinnerStage),
		}
	}
	if strings.TrimSpace(obs.TieBreakReason) != "" &&
		strings.TrimSpace(obs.TieBreakReason) != runtimeconfig.RuntimePolicyTieBreakerModeLexicalCodeThenSourceOrder {
		return &ValidationError{
			Code:    ReasonCodeTieBreakDrift,
			Message: fmt.Sprintf("case %q %s unsupported tie_break_reason=%q", caseName, lane, obs.TieBreakReason),
		}
	}
	if strings.TrimSpace(obs.DenySource) != "" && !denySeen {
		return &ValidationError{
			Code:    ReasonCodeDenySourceMismatch,
			Message: fmt.Sprintf("case %q %s deny_source must be empty when no deny candidate exists", caseName, lane),
		}
	}
	if strings.TrimSpace(obs.DenySource) != "" && !isCanonicalPolicySource(obs.DenySource) {
		return &ValidationError{
			Code:    ReasonCodeDenySourceMismatch,
			Message: fmt.Sprintf("case %q %s deny_source is invalid: %q", caseName, lane, obs.DenySource),
		}
	}
	return nil
}

func assertPolicyStackArbitrationEquivalent(caseName, lane string, expected, actual ArbitrationObservation) error {
	if expected.PolicyPrecedenceVersion != actual.PolicyPrecedenceVersion {
		return &ValidationError{
			Code:    ReasonCodePrecedenceConflict,
			Message: fmt.Sprintf("case %q %s policy precedence version drift expected=%q actual=%q", caseName, lane, expected.PolicyPrecedenceVersion, actual.PolicyPrecedenceVersion),
		}
	}
	if expected.WinnerStage != actual.WinnerStage {
		return &ValidationError{
			Code:    ReasonCodePrecedenceConflict,
			Message: fmt.Sprintf("case %q %s precedence conflict expected winner=%q actual=%q", caseName, lane, expected.WinnerStage, actual.WinnerStage),
		}
	}
	if expected.DenySource != actual.DenySource {
		return &ValidationError{
			Code:    ReasonCodeDenySourceMismatch,
			Message: fmt.Sprintf("case %q %s deny source mismatch expected=%q actual=%q", caseName, lane, expected.DenySource, actual.DenySource),
		}
	}
	if expected.TieBreakReason != actual.TieBreakReason {
		return &ValidationError{
			Code:    ReasonCodeTieBreakDrift,
			Message: fmt.Sprintf("case %q %s tie-break drift expected=%q actual=%q", caseName, lane, expected.TieBreakReason, actual.TieBreakReason),
		}
	}
	if !equalPolicyDecisionPath(expected.PolicyDecisionPath, actual.PolicyDecisionPath) {
		return &ValidationError{
			Code:    ReasonCodePrecedenceConflict,
			Message: fmt.Sprintf("case %q %s precedence conflict expected_path=%#v actual_path=%#v", caseName, lane, expected.PolicyDecisionPath, actual.PolicyDecisionPath),
		}
	}
	return nil
}

func validateReactArbitrationObservation(caseName, lane string, obs ArbitrationObservation) error {
	if !obs.ReactEnabled {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s react_enabled must be true", caseName, lane),
		}
	}
	if strings.TrimSpace(obs.ModelProvider) == "" {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s model_provider is required", caseName, lane),
		}
	}
	if obs.ReactIterationTotal <= 0 {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s react_iteration_total must be > 0", caseName, lane),
		}
	}
	if obs.ReactToolCallTotal < 0 ||
		obs.ReactToolCallBudgetHitTotal < 0 ||
		obs.ReactIterationBudgetHitTotal < 0 {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s react budget counters must be >= 0", caseName, lane),
		}
	}
	if !isCanonicalReactTerminationReason(obs.ReactTerminationReason) {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s react_termination_reason is not canonical: %q", caseName, lane, obs.ReactTerminationReason),
		}
	}
	return nil
}

func assertReactArbitrationEquivalent(caseName, lane string, expected, actual ArbitrationObservation) error {
	if expected.ReactEnabled != actual.ReactEnabled || expected.ReactIterationTotal != actual.ReactIterationTotal {
		return &ValidationError{
			Code:    ReasonCodeReactLoopStepDrift,
			Message: fmt.Sprintf("case %q %s react loop step drift expected=%#v actual=%#v", caseName, lane, expected, actual),
		}
	}
	if expected.ReactToolCallTotal != actual.ReactToolCallTotal ||
		expected.ReactToolCallBudgetHitTotal != actual.ReactToolCallBudgetHitTotal {
		return &ValidationError{
			Code:    ReasonCodeReactToolCallBudgetDrift,
			Message: fmt.Sprintf("case %q %s react tool-call budget drift expected=%#v actual=%#v", caseName, lane, expected, actual),
		}
	}
	if expected.ReactIterationBudgetHitTotal != actual.ReactIterationBudgetHitTotal {
		return &ValidationError{
			Code:    ReasonCodeReactIterationBudgetDrift,
			Message: fmt.Sprintf("case %q %s react iteration budget drift expected=%#v actual=%#v", caseName, lane, expected, actual),
		}
	}
	if expected.ReactTerminationReason != actual.ReactTerminationReason {
		return &ValidationError{
			Code:    ReasonCodeReactTerminationReasonDrift,
			Message: fmt.Sprintf("case %q %s react termination reason drift expected=%q actual=%q", caseName, lane, expected.ReactTerminationReason, actual.ReactTerminationReason),
		}
	}
	if expected.ReactStreamDispatchEnabled != actual.ReactStreamDispatchEnabled {
		return &ValidationError{
			Code:    ReasonCodeReactStreamDispatchDrift,
			Message: fmt.Sprintf("case %q %s react stream dispatch drift expected=%t actual=%t", caseName, lane, expected.ReactStreamDispatchEnabled, actual.ReactStreamDispatchEnabled),
		}
	}
	if expected.ModelProvider != actual.ModelProvider {
		return &ValidationError{
			Code:    ReasonCodeReactProviderMappingDrift,
			Message: fmt.Sprintf("case %q %s react provider mapping drift expected=%q actual=%q", caseName, lane, expected.ModelProvider, actual.ModelProvider),
		}
	}
	return nil
}

func validateSandboxArbitrationObservation(caseName, lane string, obs ArbitrationObservation) error {
	switch strings.TrimSpace(obs.SandboxMode) {
	case runtimeconfig.SecuritySandboxModeObserve, runtimeconfig.SecuritySandboxModeEnforce:
	default:
		return &ValidationError{
			Code:    ReasonCodeSandboxPolicyDrift,
			Message: fmt.Sprintf("case %q %s sandbox_mode must be observe|enforce", caseName, lane),
		}
	}
	switch strings.TrimSpace(obs.SandboxDecision) {
	case runtimeconfig.SecuritySandboxActionHost, runtimeconfig.SecuritySandboxActionSandbox, runtimeconfig.SecuritySandboxActionDeny:
	default:
		return &ValidationError{
			Code:    ReasonCodeSandboxPolicyDrift,
			Message: fmt.Sprintf("case %q %s sandbox_decision must be host|sandbox|deny", caseName, lane),
		}
	}
	if !strings.HasPrefix(strings.TrimSpace(obs.SandboxReasonCode), "sandbox.") {
		return &ValidationError{
			Code:    ReasonCodeSandboxPolicyDrift,
			Message: fmt.Sprintf("case %q %s sandbox_reason_code must be sandbox.* canonical code", caseName, lane),
		}
	}
	switch strings.TrimSpace(obs.SandboxSessionMode) {
	case runtimeconfig.SecuritySandboxSessionModePerCall, runtimeconfig.SecuritySandboxSessionModePerSession:
	default:
		return &ValidationError{
			Code:    ReasonCodeSandboxSessionLifecycleDrift,
			Message: fmt.Sprintf("case %q %s sandbox_session_mode must be per_call|per_session", caseName, lane),
		}
	}
	if obs.SandboxFallbackUsed && strings.TrimSpace(obs.SandboxFallbackReason) == "" {
		return &ValidationError{
			Code:    ReasonCodeSandboxFallbackDrift,
			Message: fmt.Sprintf("case %q %s sandbox_fallback_reason is required when fallback_used=true", caseName, lane),
		}
	}
	if !obs.SandboxFallbackUsed && strings.TrimSpace(obs.SandboxFallbackReason) != "" {
		return &ValidationError{
			Code:    ReasonCodeSandboxFallbackDrift,
			Message: fmt.Sprintf("case %q %s sandbox_fallback_reason must be empty when fallback_used=false", caseName, lane),
		}
	}
	return nil
}

func validateSandboxRolloutArbitrationObservation(caseName, lane string, obs ArbitrationObservation) error {
	switch strings.TrimSpace(obs.SandboxRolloutPhase) {
	case runtimeconfig.SecuritySandboxRolloutPhaseObserve,
		runtimeconfig.SecuritySandboxRolloutPhaseCanary,
		runtimeconfig.SecuritySandboxRolloutPhaseBaseline,
		runtimeconfig.SecuritySandboxRolloutPhaseFull,
		runtimeconfig.SecuritySandboxRolloutPhaseFrozen:
	default:
		return &ValidationError{
			Code:    ReasonCodeSandboxRolloutPhaseDrift,
			Message: fmt.Sprintf("case %q %s sandbox_rollout_phase must be observe|canary|baseline|full|frozen", caseName, lane),
		}
	}
	switch strings.TrimSpace(obs.SandboxHealthBudgetStatus) {
	case runtimeconfig.SandboxHealthBudgetWithinBudget,
		runtimeconfig.SandboxHealthBudgetNearBudget,
		runtimeconfig.SandboxHealthBudgetBreached:
	default:
		return &ValidationError{
			Code:    ReasonCodeSandboxHealthBudgetDrift,
			Message: fmt.Sprintf("case %q %s sandbox_health_budget_status must be within_budget|near_budget|breached", caseName, lane),
		}
	}
	switch strings.TrimSpace(obs.SandboxCapacityAction) {
	case runtimeconfig.SandboxCapacityActionAllow,
		runtimeconfig.SandboxCapacityActionThrottle,
		runtimeconfig.SandboxCapacityActionDeny:
	default:
		return &ValidationError{
			Code:    ReasonCodeSandboxCapacityActionDrift,
			Message: fmt.Sprintf("case %q %s sandbox_capacity_action must be allow|throttle|deny", caseName, lane),
		}
	}
	if obs.SandboxFreezeState && strings.TrimSpace(obs.SandboxFreezeReasonCode) == "" {
		return &ValidationError{
			Code:    ReasonCodeSandboxFreezeStateDrift,
			Message: fmt.Sprintf("case %q %s sandbox_freeze_reason_code is required when sandbox_freeze_state=true", caseName, lane),
		}
	}
	if !obs.SandboxFreezeState && strings.TrimSpace(obs.SandboxFreezeReasonCode) != "" {
		return &ValidationError{
			Code:    ReasonCodeSandboxFreezeStateDrift,
			Message: fmt.Sprintf("case %q %s sandbox_freeze_reason_code must be empty when sandbox_freeze_state=false", caseName, lane),
		}
	}
	return nil
}

func assertSandboxArbitrationEquivalent(caseName, lane string, expected, actual ArbitrationObservation) error {
	if expected.SandboxMode != actual.SandboxMode ||
		expected.SandboxBackend != actual.SandboxBackend ||
		expected.SandboxProfile != actual.SandboxProfile ||
		expected.SandboxDecision != actual.SandboxDecision ||
		expected.SandboxReasonCode != actual.SandboxReasonCode {
		return &ValidationError{
			Code:    ReasonCodeSandboxPolicyDrift,
			Message: fmt.Sprintf("case %q %s sandbox policy drift expected=%#v actual=%#v", caseName, lane, expected, actual),
		}
	}
	if expected.SandboxFallbackUsed != actual.SandboxFallbackUsed ||
		expected.SandboxFallbackReason != actual.SandboxFallbackReason ||
		expected.SandboxLaunchFailedTotal != actual.SandboxLaunchFailedTotal {
		return &ValidationError{
			Code:    ReasonCodeSandboxFallbackDrift,
			Message: fmt.Sprintf("case %q %s sandbox fallback drift expected=%#v actual=%#v", caseName, lane, expected, actual),
		}
	}
	if expected.SandboxTimeoutTotal != actual.SandboxTimeoutTotal {
		return &ValidationError{
			Code:    ReasonCodeSandboxTimeoutDrift,
			Message: fmt.Sprintf("case %q %s sandbox timeout drift expected=%d actual=%d", caseName, lane, expected.SandboxTimeoutTotal, actual.SandboxTimeoutTotal),
		}
	}
	if !equalStringSlice(expected.SandboxRequiredCapabilities, actual.SandboxRequiredCapabilities) ||
		expected.SandboxCapabilityMismatchTotal != actual.SandboxCapabilityMismatchTotal {
		return &ValidationError{
			Code:    ReasonCodeSandboxCapabilityDrift,
			Message: fmt.Sprintf("case %q %s sandbox capability drift expected=%#v actual=%#v", caseName, lane, expected, actual),
		}
	}
	if expected.SandboxSessionMode != actual.SandboxSessionMode ||
		expected.SandboxQueueWaitMsP95 != actual.SandboxQueueWaitMsP95 ||
		expected.SandboxExecLatencyMsP95 != actual.SandboxExecLatencyMsP95 {
		return &ValidationError{
			Code:    ReasonCodeSandboxSessionLifecycleDrift,
			Message: fmt.Sprintf("case %q %s sandbox session lifecycle drift expected=%#v actual=%#v", caseName, lane, expected, actual),
		}
	}
	if expected.SandboxExitCodeLast != actual.SandboxExitCodeLast ||
		expected.SandboxOOMTotal != actual.SandboxOOMTotal ||
		expected.SandboxResourceCPUMsTotal != actual.SandboxResourceCPUMsTotal ||
		expected.SandboxResourceMemoryPeakBytesP95 != actual.SandboxResourceMemoryPeakBytesP95 {
		return &ValidationError{
			Code:    ReasonCodeSandboxResourcePolicyDrift,
			Message: fmt.Sprintf("case %q %s sandbox resource policy drift expected=%#v actual=%#v", caseName, lane, expected, actual),
		}
	}
	return nil
}

func assertSandboxRolloutArbitrationEquivalent(caseName, lane string, expected, actual ArbitrationObservation) error {
	if expected.SandboxRolloutPhase != actual.SandboxRolloutPhase {
		return &ValidationError{
			Code:    ReasonCodeSandboxRolloutPhaseDrift,
			Message: fmt.Sprintf("case %q %s sandbox rollout phase drift expected=%q actual=%q", caseName, lane, expected.SandboxRolloutPhase, actual.SandboxRolloutPhase),
		}
	}
	if expected.SandboxHealthBudgetStatus != actual.SandboxHealthBudgetStatus {
		return &ValidationError{
			Code:    ReasonCodeSandboxHealthBudgetDrift,
			Message: fmt.Sprintf("case %q %s sandbox health budget drift expected=%q actual=%q", caseName, lane, expected.SandboxHealthBudgetStatus, actual.SandboxHealthBudgetStatus),
		}
	}
	if expected.SandboxCapacityAction != actual.SandboxCapacityAction {
		return &ValidationError{
			Code:    ReasonCodeSandboxCapacityActionDrift,
			Message: fmt.Sprintf("case %q %s sandbox capacity action drift expected=%q actual=%q", caseName, lane, expected.SandboxCapacityAction, actual.SandboxCapacityAction),
		}
	}
	if expected.SandboxFreezeState != actual.SandboxFreezeState ||
		expected.SandboxFreezeReasonCode != actual.SandboxFreezeReasonCode {
		return &ValidationError{
			Code:    ReasonCodeSandboxFreezeStateDrift,
			Message: fmt.Sprintf("case %q %s sandbox freeze state drift expected=%#v actual=%#v", caseName, lane, expected, actual),
		}
	}
	return nil
}

func validateSandboxEgressArbitrationObservation(caseName, lane string, obs ArbitrationObservation) error {
	if !isSandboxEgressAction(obs.SandboxEgressAction) {
		return &ValidationError{
			Code:    ReasonCodeSandboxEgressActionDrift,
			Message: fmt.Sprintf("case %q %s sandbox_egress_action must be deny|allow|allow_and_record", caseName, lane),
		}
	}
	if !isSandboxEgressPolicySource(obs.SandboxEgressPolicySource) {
		return &ValidationError{
			Code:    ReasonCodeSandboxEgressPolicySourceDrift,
			Message: fmt.Sprintf("case %q %s sandbox_egress_policy_source must be default_action|by_tool|allowlist|on_violation", caseName, lane),
		}
	}
	if obs.SandboxEgressViolationTotal > 0 && obs.SandboxEgressAction == runtimeconfig.SecuritySandboxEgressActionAllow {
		return &ValidationError{
			Code:    ReasonCodeSandboxEgressViolationTaxonomyDrift,
			Message: fmt.Sprintf("case %q %s sandbox_egress_violation_total>0 requires action deny|allow_and_record", caseName, lane),
		}
	}
	if obs.SandboxEgressViolationTotal == 0 &&
		(obs.SandboxEgressAction == runtimeconfig.SecuritySandboxEgressActionDeny ||
			obs.SandboxEgressAction == runtimeconfig.SecuritySandboxEgressActionAllowAndRecord) {
		return &ValidationError{
			Code:    ReasonCodeSandboxEgressViolationTaxonomyDrift,
			Message: fmt.Sprintf("case %q %s deny/allow_and_record requires sandbox_egress_violation_total>0", caseName, lane),
		}
	}
	if !isAdapterAllowlistDecision(obs.AdapterAllowlistDecision) {
		return &ValidationError{
			Code:    ReasonCodeAdapterAllowlistDecisionDrift,
			Message: fmt.Sprintf("case %q %s adapter_allowlist_decision must be allow|deny", caseName, lane),
		}
	}
	if obs.AdapterAllowlistDecision == "deny" {
		if obs.AdapterAllowlistBlockTotal <= 0 {
			return &ValidationError{
				Code:    ReasonCodeAdapterAllowlistTaxonomyDrift,
				Message: fmt.Sprintf("case %q %s deny decision requires adapter_allowlist_block_total>0", caseName, lane),
			}
		}
		if !isCanonicalAdapterAllowlistPrimaryCode(obs.AdapterAllowlistPrimaryCode) {
			return &ValidationError{
				Code:    ReasonCodeAdapterAllowlistTaxonomyDrift,
				Message: fmt.Sprintf("case %q %s adapter_allowlist_primary_code must be canonical adapter.allowlist.*", caseName, lane),
			}
		}
		return nil
	}
	if obs.AdapterAllowlistBlockTotal > 0 {
		return &ValidationError{
			Code:    ReasonCodeAdapterAllowlistTaxonomyDrift,
			Message: fmt.Sprintf("case %q %s allow decision requires adapter_allowlist_block_total=0", caseName, lane),
		}
	}
	if strings.TrimSpace(obs.AdapterAllowlistPrimaryCode) != "" {
		return &ValidationError{
			Code:    ReasonCodeAdapterAllowlistTaxonomyDrift,
			Message: fmt.Sprintf("case %q %s allow decision requires adapter_allowlist_primary_code empty", caseName, lane),
		}
	}
	return nil
}

func assertSandboxEgressArbitrationEquivalent(caseName, lane string, expected, actual ArbitrationObservation) error {
	if expected.SandboxEgressAction != actual.SandboxEgressAction {
		return &ValidationError{
			Code:    ReasonCodeSandboxEgressActionDrift,
			Message: fmt.Sprintf("case %q %s sandbox egress action drift expected=%q actual=%q", caseName, lane, expected.SandboxEgressAction, actual.SandboxEgressAction),
		}
	}
	if expected.SandboxEgressPolicySource != actual.SandboxEgressPolicySource {
		return &ValidationError{
			Code:    ReasonCodeSandboxEgressPolicySourceDrift,
			Message: fmt.Sprintf("case %q %s sandbox egress policy source drift expected=%q actual=%q", caseName, lane, expected.SandboxEgressPolicySource, actual.SandboxEgressPolicySource),
		}
	}
	if expected.SandboxEgressViolationTotal != actual.SandboxEgressViolationTotal {
		return &ValidationError{
			Code:    ReasonCodeSandboxEgressViolationTaxonomyDrift,
			Message: fmt.Sprintf("case %q %s sandbox egress violation taxonomy drift expected=%d actual=%d", caseName, lane, expected.SandboxEgressViolationTotal, actual.SandboxEgressViolationTotal),
		}
	}
	if expected.AdapterAllowlistDecision != actual.AdapterAllowlistDecision {
		return &ValidationError{
			Code:    ReasonCodeAdapterAllowlistDecisionDrift,
			Message: fmt.Sprintf("case %q %s adapter allowlist decision drift expected=%q actual=%q", caseName, lane, expected.AdapterAllowlistDecision, actual.AdapterAllowlistDecision),
		}
	}
	if expected.AdapterAllowlistBlockTotal != actual.AdapterAllowlistBlockTotal ||
		expected.AdapterAllowlistPrimaryCode != actual.AdapterAllowlistPrimaryCode {
		return &ValidationError{
			Code:    ReasonCodeAdapterAllowlistTaxonomyDrift,
			Message: fmt.Sprintf("case %q %s adapter allowlist taxonomy drift expected=%#v actual=%#v", caseName, lane, expected, actual),
		}
	}
	return nil
}

func canonicalizeBudgetAdmissionSnapshot(in *BudgetAdmissionSnapshot) *BudgetAdmissionSnapshot {
	if in == nil {
		return nil
	}
	out := &BudgetAdmissionSnapshot{
		Version: strings.ToLower(strings.TrimSpace(in.Version)),
		CostEstimate: BudgetAdmissionCostEstimate{
			Token:   nonNegativeFloat64(in.CostEstimate.Token),
			Tool:    nonNegativeFloat64(in.CostEstimate.Tool),
			Sandbox: nonNegativeFloat64(in.CostEstimate.Sandbox),
			Memory:  nonNegativeFloat64(in.CostEstimate.Memory),
			Total:   nonNegativeFloat64(in.CostEstimate.Total),
		},
		LatencyEstimate: BudgetAdmissionLatencyEstimate{
			TokenMs:   nonNegativeInt64(in.LatencyEstimate.TokenMs),
			ToolMs:    nonNegativeInt64(in.LatencyEstimate.ToolMs),
			SandboxMs: nonNegativeInt64(in.LatencyEstimate.SandboxMs),
			MemoryMs:  nonNegativeInt64(in.LatencyEstimate.MemoryMs),
			TotalMs:   nonNegativeInt64(in.LatencyEstimate.TotalMs),
		},
	}
	return out
}

func validateBudgetAdmissionArbitrationObservation(caseName, lane string, obs ArbitrationObservation) error {
	if !isCanonicalBudgetDecision(obs.BudgetDecision) {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s budget_decision must be allow|degrade|deny", caseName, lane),
		}
	}
	if obs.BudgetSnapshot == nil {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s budget_snapshot is required", caseName, lane),
		}
	}
	if strings.TrimSpace(obs.BudgetSnapshot.Version) != runtimeconfig.RuntimeAdmissionBudgetSnapshotVersionV1 {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s budget_snapshot.version must be %q", caseName, lane, runtimeconfig.RuntimeAdmissionBudgetSnapshotVersionV1),
		}
	}
	if err := validateBudgetEstimateTotals(caseName, lane, obs.BudgetSnapshot); err != nil {
		return err
	}
	switch strings.TrimSpace(obs.BudgetDecision) {
	case string(runtimeconfig.RuntimeAdmissionBudgetDecisionAllow), string(runtimeconfig.RuntimeAdmissionBudgetDecisionDeny):
		if strings.TrimSpace(obs.DegradeAction) != "" {
			return &ValidationError{
				Code:    ReasonCodeDegradePolicyDrift,
				Message: fmt.Sprintf("case %q %s degrade_action must be empty when budget_decision=%s", caseName, lane, obs.BudgetDecision),
			}
		}
	case string(runtimeconfig.RuntimeAdmissionBudgetDecisionDegrade):
		if !isCanonicalBudgetDegradeAction(obs.DegradeAction) {
			return &ValidationError{
				Code:    ReasonCodeDegradePolicyDrift,
				Message: fmt.Sprintf("case %q %s degrade_action must be canonical when budget_decision=degrade", caseName, lane),
			}
		}
	}
	return nil
}

func validateBudgetEstimateTotals(caseName, lane string, snapshot *BudgetAdmissionSnapshot) error {
	if snapshot == nil {
		return nil
	}
	cost := snapshot.CostEstimate
	latency := snapshot.LatencyEstimate
	if cost.Token < 0 || cost.Tool < 0 || cost.Sandbox < 0 || cost.Memory < 0 || cost.Total < 0 {
		return &ValidationError{
			Code:    ReasonCodeBudgetThresholdDrift,
			Message: fmt.Sprintf("case %q %s budget_snapshot.cost_estimate values must be >= 0", caseName, lane),
		}
	}
	if latency.TokenMs < 0 || latency.ToolMs < 0 || latency.SandboxMs < 0 || latency.MemoryMs < 0 || latency.TotalMs < 0 {
		return &ValidationError{
			Code:    ReasonCodeBudgetThresholdDrift,
			Message: fmt.Sprintf("case %q %s budget_snapshot.latency_estimate values must be >= 0", caseName, lane),
		}
	}
	costSum := nonNegativeFloat64(cost.Token) +
		nonNegativeFloat64(cost.Tool) +
		nonNegativeFloat64(cost.Sandbox) +
		nonNegativeFloat64(cost.Memory)
	if !approxFloat64(cost.Total, costSum) {
		return &ValidationError{
			Code:    ReasonCodeBudgetThresholdDrift,
			Message: fmt.Sprintf("case %q %s budget_snapshot.cost_estimate.total must equal sum(token|tool|sandbox|memory)", caseName, lane),
		}
	}
	latencySum := nonNegativeInt64(latency.TokenMs) +
		nonNegativeInt64(latency.ToolMs) +
		nonNegativeInt64(latency.SandboxMs) +
		nonNegativeInt64(latency.MemoryMs)
	if latency.TotalMs != latencySum {
		return &ValidationError{
			Code:    ReasonCodeBudgetThresholdDrift,
			Message: fmt.Sprintf("case %q %s budget_snapshot.latency_estimate.total_ms must equal sum(token_ms|tool_ms|sandbox_ms|memory_ms)", caseName, lane),
		}
	}
	return nil
}

func assertBudgetAdmissionArbitrationEquivalent(caseName, lane string, expected, actual ArbitrationObservation) error {
	if !equalBudgetAdmissionSnapshot(expected.BudgetSnapshot, actual.BudgetSnapshot) {
		return &ValidationError{
			Code:    ReasonCodeBudgetThresholdDrift,
			Message: fmt.Sprintf("case %q %s budget threshold drift expected_snapshot=%#v actual_snapshot=%#v", caseName, lane, expected.BudgetSnapshot, actual.BudgetSnapshot),
		}
	}
	if expected.BudgetDecision != actual.BudgetDecision {
		return &ValidationError{
			Code:    ReasonCodeAdmissionDecisionDrift,
			Message: fmt.Sprintf("case %q %s admission decision drift expected=%q actual=%q", caseName, lane, expected.BudgetDecision, actual.BudgetDecision),
		}
	}
	if expected.DegradeAction != actual.DegradeAction {
		return &ValidationError{
			Code:    ReasonCodeDegradePolicyDrift,
			Message: fmt.Sprintf("case %q %s degrade policy drift expected=%q actual=%q", caseName, lane, expected.DegradeAction, actual.DegradeAction),
		}
	}
	return &ValidationError{
		Code:    ReasonCodeSemanticDrift,
		Message: fmt.Sprintf("case %q %s budget admission semantic drift expected=%#v actual=%#v", caseName, lane, expected, actual),
	}
}

func equalBudgetAdmissionSnapshot(left, right *BudgetAdmissionSnapshot) bool {
	if left == nil && right == nil {
		return true
	}
	if left == nil || right == nil {
		return false
	}
	return left.Version == right.Version &&
		approxFloat64(left.CostEstimate.Token, right.CostEstimate.Token) &&
		approxFloat64(left.CostEstimate.Tool, right.CostEstimate.Tool) &&
		approxFloat64(left.CostEstimate.Sandbox, right.CostEstimate.Sandbox) &&
		approxFloat64(left.CostEstimate.Memory, right.CostEstimate.Memory) &&
		approxFloat64(left.CostEstimate.Total, right.CostEstimate.Total) &&
		left.LatencyEstimate.TokenMs == right.LatencyEstimate.TokenMs &&
		left.LatencyEstimate.ToolMs == right.LatencyEstimate.ToolMs &&
		left.LatencyEstimate.SandboxMs == right.LatencyEstimate.SandboxMs &&
		left.LatencyEstimate.MemoryMs == right.LatencyEstimate.MemoryMs &&
		left.LatencyEstimate.TotalMs == right.LatencyEstimate.TotalMs
}

func isCanonicalBudgetDecision(decision string) bool {
	switch strings.TrimSpace(decision) {
	case string(runtimeconfig.RuntimeAdmissionBudgetDecisionAllow),
		string(runtimeconfig.RuntimeAdmissionBudgetDecisionDegrade),
		string(runtimeconfig.RuntimeAdmissionBudgetDecisionDeny):
		return true
	default:
		return false
	}
}

func isCanonicalBudgetDegradeAction(action string) bool {
	switch strings.TrimSpace(action) {
	case runtimeconfig.RuntimeAdmissionDegradeActionReduceToolCallLimit,
		runtimeconfig.RuntimeAdmissionDegradeActionTrimMemoryContext,
		runtimeconfig.RuntimeAdmissionDegradeActionSandboxThrottle:
		return true
	default:
		return false
	}
}

func approxFloat64(left, right float64) bool {
	diff := left - right
	if diff < 0 {
		diff = -diff
	}
	return diff <= 1e-9
}

func nonNegativeFloat64(value float64) float64 {
	if value < 0 {
		return 0
	}
	return value
}

func nonNegativeInt64(value int64) int64 {
	if value < 0 {
		return 0
	}
	return value
}

func arbitrationObservationsEqual(version string, left, right ArbitrationObservation) bool {
	if version == ArbitrationFixtureVersionObsV1 {
		return left.ObservabilityExportProfile == right.ObservabilityExportProfile &&
			left.ObservabilityExportStatus == right.ObservabilityExportStatus &&
			left.ObservabilityExportReasonCode == right.ObservabilityExportReasonCode &&
			left.DiagnosticsBundleLastStatus == right.DiagnosticsBundleLastStatus &&
			left.DiagnosticsBundleLastReasonCode == right.DiagnosticsBundleLastReasonCode &&
			left.DiagnosticsBundleLastSchemaVersion == right.DiagnosticsBundleLastSchemaVersion &&
			left.DiagnosticsBundleRedactionStatus == right.DiagnosticsBundleRedactionStatus &&
			left.DiagnosticsBundleGateFingerprint == right.DiagnosticsBundleGateFingerprint
	}
	if version == ArbitrationFixtureVersionMemoryV1 {
		return left.MemoryMode == right.MemoryMode &&
			left.MemoryProvider == right.MemoryProvider &&
			left.MemoryProfile == right.MemoryProfile &&
			left.MemoryContractVersion == right.MemoryContractVersion &&
			left.MemoryQueryTotal == right.MemoryQueryTotal &&
			left.MemoryUpsertTotal == right.MemoryUpsertTotal &&
			left.MemoryDeleteTotal == right.MemoryDeleteTotal &&
			left.MemoryErrorTotal == right.MemoryErrorTotal &&
			left.MemoryFallbackTotal == right.MemoryFallbackTotal &&
			left.MemoryFallbackReasonCode == right.MemoryFallbackReasonCode &&
			left.MemoryReasonCode == right.MemoryReasonCode
	}
	if version == ArbitrationFixtureVersionMemoryScopeV1 {
		return left.MemoryScopeSelected == right.MemoryScopeSelected &&
			left.MemoryBudgetUsed == right.MemoryBudgetUsed &&
			left.MemoryHits == right.MemoryHits
	}
	if version == ArbitrationFixtureVersionMemorySearchV1 {
		return left.MemoryHits == right.MemoryHits &&
			equalIntMap(left.MemoryRerankStats, right.MemoryRerankStats)
	}
	if version == ArbitrationFixtureVersionMemoryLifecycleV1 {
		return left.MemoryLifecycleAction == right.MemoryLifecycleAction
	}
	if version == ArbitrationFixtureVersionReactV1 {
		return left.ModelProvider == right.ModelProvider &&
			left.ReactEnabled == right.ReactEnabled &&
			left.ReactIterationTotal == right.ReactIterationTotal &&
			left.ReactToolCallTotal == right.ReactToolCallTotal &&
			left.ReactToolCallBudgetHitTotal == right.ReactToolCallBudgetHitTotal &&
			left.ReactIterationBudgetHitTotal == right.ReactIterationBudgetHitTotal &&
			left.ReactTerminationReason == right.ReactTerminationReason &&
			left.ReactStreamDispatchEnabled == right.ReactStreamDispatchEnabled
	}
	if version == ArbitrationFixtureVersionA57V1 {
		return left.SandboxEgressAction == right.SandboxEgressAction &&
			left.SandboxEgressPolicySource == right.SandboxEgressPolicySource &&
			left.SandboxEgressViolationTotal == right.SandboxEgressViolationTotal &&
			left.AdapterAllowlistDecision == right.AdapterAllowlistDecision &&
			left.AdapterAllowlistBlockTotal == right.AdapterAllowlistBlockTotal &&
			left.AdapterAllowlistPrimaryCode == right.AdapterAllowlistPrimaryCode
	}
	if version == ArbitrationFixtureVersionBudgetAdmissionV1 {
		return left.BudgetDecision == right.BudgetDecision &&
			left.DegradeAction == right.DegradeAction &&
			equalBudgetAdmissionSnapshot(left.BudgetSnapshot, right.BudgetSnapshot)
	}
	if version == ArbitrationFixtureVersionPolicyV1 {
		return left.PolicyPrecedenceVersion == right.PolicyPrecedenceVersion &&
			left.WinnerStage == right.WinnerStage &&
			left.DenySource == right.DenySource &&
			left.TieBreakReason == right.TieBreakReason &&
			equalPolicyDecisionPath(left.PolicyDecisionPath, right.PolicyDecisionPath)
	}
	if left.RuntimePrimaryDomain != right.RuntimePrimaryDomain ||
		left.RuntimePrimaryCode != right.RuntimePrimaryCode ||
		left.RuntimePrimarySource != right.RuntimePrimarySource ||
		left.RuntimePrimaryConflictTotal != right.RuntimePrimaryConflictTotal {
		return false
	}
	if version == ArbitrationFixtureVersionA48V1 {
		return true
	}
	if left.RuntimeSecondaryReasonCount != right.RuntimeSecondaryReasonCount ||
		left.RuntimeArbitrationRuleVersion != right.RuntimeArbitrationRuleVersion ||
		left.RuntimeRemediationHintCode != right.RuntimeRemediationHintCode ||
		left.RuntimeRemediationHintDomain != right.RuntimeRemediationHintDomain {
		return false
	}
	if !equalStringSlice(left.RuntimeSecondaryReasonCodes, right.RuntimeSecondaryReasonCodes) {
		return false
	}
	if version == ArbitrationFixtureVersionA49V1 {
		return true
	}
	if left.RuntimeArbitrationRuleRequestedVersion != right.RuntimeArbitrationRuleRequestedVersion ||
		left.RuntimeArbitrationRuleEffectiveVersion != right.RuntimeArbitrationRuleEffectiveVersion ||
		left.RuntimeArbitrationRuleVersionSource != right.RuntimeArbitrationRuleVersionSource ||
		left.RuntimeArbitrationRulePolicyAction != right.RuntimeArbitrationRulePolicyAction ||
		left.RuntimeArbitrationRuleUnsupportedTotal != right.RuntimeArbitrationRuleUnsupportedTotal ||
		left.RuntimeArbitrationRuleMismatchTotal != right.RuntimeArbitrationRuleMismatchTotal {
		return false
	}
	if version == ArbitrationFixtureVersionA50V1 {
		return true
	}
	if left.SandboxMode != right.SandboxMode ||
		left.SandboxBackend != right.SandboxBackend ||
		left.SandboxProfile != right.SandboxProfile ||
		left.SandboxSessionMode != right.SandboxSessionMode ||
		!equalStringSlice(left.SandboxRequiredCapabilities, right.SandboxRequiredCapabilities) ||
		left.SandboxDecision != right.SandboxDecision ||
		left.SandboxReasonCode != right.SandboxReasonCode ||
		left.SandboxFallbackUsed != right.SandboxFallbackUsed ||
		left.SandboxFallbackReason != right.SandboxFallbackReason ||
		left.SandboxTimeoutTotal != right.SandboxTimeoutTotal ||
		left.SandboxLaunchFailedTotal != right.SandboxLaunchFailedTotal ||
		left.SandboxCapabilityMismatchTotal != right.SandboxCapabilityMismatchTotal ||
		left.SandboxQueueWaitMsP95 != right.SandboxQueueWaitMsP95 ||
		left.SandboxExecLatencyMsP95 != right.SandboxExecLatencyMsP95 ||
		left.SandboxExitCodeLast != right.SandboxExitCodeLast ||
		left.SandboxOOMTotal != right.SandboxOOMTotal ||
		left.SandboxResourceCPUMsTotal != right.SandboxResourceCPUMsTotal ||
		left.SandboxResourceMemoryPeakBytesP95 != right.SandboxResourceMemoryPeakBytesP95 {
		return false
	}
	if version == ArbitrationFixtureVersionA51V1 {
		return true
	}
	if left.SandboxRolloutPhase != right.SandboxRolloutPhase ||
		left.SandboxHealthBudgetStatus != right.SandboxHealthBudgetStatus ||
		left.SandboxCapacityAction != right.SandboxCapacityAction ||
		left.SandboxFreezeState != right.SandboxFreezeState ||
		left.SandboxFreezeReasonCode != right.SandboxFreezeReasonCode {
		return false
	}
	return true
}

func equalStringSlice(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func equalPolicyDecisionPath(left, right []PolicyDecisionPathEntry) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i].Stage != right[i].Stage ||
			left[i].Code != right[i].Code ||
			left[i].Source != right[i].Source ||
			left[i].Decision != right[i].Decision {
			return false
		}
	}
	return true
}

func equalIntMap(left, right map[string]int) bool {
	if len(left) != len(right) {
		return false
	}
	for key, leftValue := range left {
		if right[key] != leftValue {
			return false
		}
	}
	return true
}

func isCanonicalPolicyStage(stage string) bool {
	needle := strings.ToLower(strings.TrimSpace(stage))
	if needle == "" {
		return false
	}
	for _, item := range runtimeconfig.RuntimePolicyCanonicalStages() {
		if needle == strings.ToLower(strings.TrimSpace(item)) {
			return true
		}
	}
	return false
}

func isCanonicalPolicySource(source string) bool {
	normalized := strings.ToLower(strings.TrimSpace(source))
	if normalized == "" {
		return false
	}
	if isCanonicalPolicyStage(normalized) {
		return true
	}
	normalized = strings.ReplaceAll(normalized, ".", "_")
	normalized = strings.ReplaceAll(normalized, ":", "_")
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.ReplaceAll(normalized, "/", "_")
	return isCanonicalPolicyStage(normalized)
}

func isCanonicalArbitrationCode(code string) bool {
	code = strings.TrimSpace(code)
	if code == "" {
		return false
	}
	if code == runtimeconfig.RuntimePrimaryCodeTimeoutRejected ||
		code == runtimeconfig.RuntimePrimaryCodeTimeoutExhausted ||
		code == runtimeconfig.RuntimePrimaryCodeTimeoutClamped {
		return true
	}
	if _, ok := canonicalReadinessCodes[code]; ok {
		return true
	}
	if _, ok := canonicalAdapterCodes[code]; ok {
		return true
	}
	return false
}

func isCanonicalReactTerminationReason(reason string) bool {
	switch strings.TrimSpace(reason) {
	case runtimeconfig.RuntimeReactTerminationCompleted,
		runtimeconfig.RuntimeReactTerminationMaxIterationsExceeded,
		runtimeconfig.RuntimeReactTerminationToolCallLimitExceeded,
		runtimeconfig.RuntimeReactTerminationToolDispatchFailed,
		runtimeconfig.RuntimeReactTerminationProviderError,
		runtimeconfig.RuntimeReactTerminationContextCanceled:
		return true
	default:
		return false
	}
}

func isSandboxEgressAction(action string) bool {
	switch strings.TrimSpace(action) {
	case runtimeconfig.SecuritySandboxEgressActionDeny,
		runtimeconfig.SecuritySandboxEgressActionAllow,
		runtimeconfig.SecuritySandboxEgressActionAllowAndRecord:
		return true
	default:
		return false
	}
}

func isSandboxEgressPolicySource(source string) bool {
	switch strings.TrimSpace(source) {
	case "default_action", "by_tool", "allowlist", "on_violation":
		return true
	default:
		return false
	}
}

func isAdapterAllowlistDecision(decision string) bool {
	switch strings.TrimSpace(decision) {
	case "allow", "deny":
		return true
	default:
		return false
	}
}

func isCanonicalAdapterAllowlistPrimaryCode(code string) bool {
	switch strings.TrimSpace(code) {
	case runtimeconfig.ReadinessCodeAdapterAllowlistMissingEntry,
		runtimeconfig.ReadinessCodeAdapterAllowlistSignatureInvalid,
		runtimeconfig.ReadinessCodeAdapterAllowlistPolicyConflict:
		return true
	default:
		return false
	}
}

func validateMemoryArbitrationObservation(caseName, lane string, obs ArbitrationObservation) error {
	switch strings.TrimSpace(obs.MemoryMode) {
	case runtimeconfig.RuntimeMemoryModeBuiltinFilesystem, runtimeconfig.RuntimeMemoryModeExternalSPI:
	default:
		return &ValidationError{
			Code:    ReasonCodeMemoryModeDrift,
			Message: fmt.Sprintf("case %q %s memory_mode must be external_spi|builtin_filesystem", caseName, lane),
		}
	}
	if strings.TrimSpace(obs.MemoryProvider) == "" {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s memory_provider is required", caseName, lane),
		}
	}
	if strings.TrimSpace(obs.MemoryProfile) == "" {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s memory_profile is required", caseName, lane),
		}
	}
	if strings.TrimSpace(obs.MemoryContractVersion) != runtimeconfig.RuntimeMemoryContractVersionV1 {
		return &ValidationError{
			Code:    ReasonCodeMemoryContractVersionDrift,
			Message: fmt.Sprintf("case %q %s memory_contract_version must be %q", caseName, lane, runtimeconfig.RuntimeMemoryContractVersionV1),
		}
	}
	if obs.MemoryMode == runtimeconfig.RuntimeMemoryModeBuiltinFilesystem &&
		strings.TrimSpace(obs.MemoryProvider) != runtimeconfig.RuntimeMemoryModeBuiltinFilesystem {
		return &ValidationError{
			Code:    ReasonCodeMemoryProfileDrift,
			Message: fmt.Sprintf("case %q %s builtin_filesystem mode must use memory_provider=builtin_filesystem", caseName, lane),
		}
	}
	if obs.MemoryMode == runtimeconfig.RuntimeMemoryModeExternalSPI &&
		strings.TrimSpace(obs.MemoryProvider) == runtimeconfig.RuntimeMemoryModeBuiltinFilesystem {
		return &ValidationError{
			Code:    ReasonCodeMemoryProfileDrift,
			Message: fmt.Sprintf("case %q %s external_spi mode must not use memory_provider=builtin_filesystem", caseName, lane),
		}
	}
	if obs.MemoryFallbackTotal > 0 && strings.TrimSpace(obs.MemoryFallbackReasonCode) == "" {
		return &ValidationError{
			Code:    ReasonCodeMemoryFallbackDrift,
			Message: fmt.Sprintf("case %q %s memory_fallback_reason_code is required when memory_fallback_total>0", caseName, lane),
		}
	}
	if obs.MemoryFallbackTotal == 0 && strings.TrimSpace(obs.MemoryFallbackReasonCode) != "" {
		return &ValidationError{
			Code:    ReasonCodeMemoryFallbackDrift,
			Message: fmt.Sprintf("case %q %s memory_fallback_reason_code must be empty when memory_fallback_total=0", caseName, lane),
		}
	}
	if strings.TrimSpace(obs.MemoryFallbackReasonCode) != "" &&
		!strings.HasPrefix(strings.TrimSpace(obs.MemoryFallbackReasonCode), "memory.fallback.") {
		return &ValidationError{
			Code:    ReasonCodeMemoryFallbackDrift,
			Message: fmt.Sprintf("case %q %s memory_fallback_reason_code must be memory.fallback.* canonical code", caseName, lane),
		}
	}
	if obs.MemoryErrorTotal > 0 && strings.TrimSpace(obs.MemoryReasonCode) == "" {
		return &ValidationError{
			Code:    ReasonCodeMemoryErrorTaxonomyDrift,
			Message: fmt.Sprintf("case %q %s memory_reason_code is required when memory_error_total>0", caseName, lane),
		}
	}
	if strings.TrimSpace(obs.MemoryReasonCode) != "" &&
		!strings.HasPrefix(strings.TrimSpace(obs.MemoryReasonCode), "memory.") {
		return &ValidationError{
			Code:    ReasonCodeMemoryErrorTaxonomyDrift,
			Message: fmt.Sprintf("case %q %s memory_reason_code must be memory.* canonical code", caseName, lane),
		}
	}
	return nil
}

func assertMemoryArbitrationEquivalent(caseName, lane string, expected, actual ArbitrationObservation) error {
	if expected.MemoryMode != actual.MemoryMode {
		return &ValidationError{
			Code:    ReasonCodeMemoryModeDrift,
			Message: fmt.Sprintf("case %q %s memory mode drift expected=%q actual=%q", caseName, lane, expected.MemoryMode, actual.MemoryMode),
		}
	}
	if expected.MemoryProvider != actual.MemoryProvider || expected.MemoryProfile != actual.MemoryProfile {
		return &ValidationError{
			Code:    ReasonCodeMemoryProfileDrift,
			Message: fmt.Sprintf("case %q %s memory profile drift expected_provider/profile=%q/%q actual_provider/profile=%q/%q", caseName, lane, expected.MemoryProvider, expected.MemoryProfile, actual.MemoryProvider, actual.MemoryProfile),
		}
	}
	if expected.MemoryContractVersion != actual.MemoryContractVersion {
		return &ValidationError{
			Code:    ReasonCodeMemoryContractVersionDrift,
			Message: fmt.Sprintf("case %q %s memory contract version drift expected=%q actual=%q", caseName, lane, expected.MemoryContractVersion, actual.MemoryContractVersion),
		}
	}
	if expected.MemoryFallbackTotal != actual.MemoryFallbackTotal ||
		expected.MemoryFallbackReasonCode != actual.MemoryFallbackReasonCode {
		return &ValidationError{
			Code:    ReasonCodeMemoryFallbackDrift,
			Message: fmt.Sprintf("case %q %s memory fallback drift expected=%#v actual=%#v", caseName, lane, expected, actual),
		}
	}
	if expected.MemoryReasonCode != actual.MemoryReasonCode {
		return &ValidationError{
			Code:    ReasonCodeMemoryErrorTaxonomyDrift,
			Message: fmt.Sprintf("case %q %s memory error taxonomy drift expected_reason=%q actual_reason=%q", caseName, lane, expected.MemoryReasonCode, actual.MemoryReasonCode),
		}
	}
	if expected.MemoryQueryTotal != actual.MemoryQueryTotal ||
		expected.MemoryUpsertTotal != actual.MemoryUpsertTotal ||
		expected.MemoryDeleteTotal != actual.MemoryDeleteTotal ||
		expected.MemoryErrorTotal != actual.MemoryErrorTotal {
		return &ValidationError{
			Code:    ReasonCodeMemoryOperationAggregateDrift,
			Message: fmt.Sprintf("case %q %s memory operation aggregate drift expected=%#v actual=%#v", caseName, lane, expected, actual),
		}
	}
	return nil
}

func validateMemoryScopeArbitrationObservation(caseName, lane string, obs ArbitrationObservation) error {
	switch strings.TrimSpace(obs.MemoryScopeSelected) {
	case runtimeconfig.RuntimeMemoryScopeSession, runtimeconfig.RuntimeMemoryScopeProject, runtimeconfig.RuntimeMemoryScopeGlobal:
	default:
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s memory_scope_selected must be session|project|global", caseName, lane),
		}
	}
	if obs.MemoryBudgetUsed < 0 {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s memory_budget_used must be >= 0", caseName, lane),
		}
	}
	if obs.MemoryHits < 0 {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s memory_hits must be >= 0", caseName, lane),
		}
	}
	return nil
}

func assertMemoryScopeArbitrationEquivalent(caseName, lane string, expected, actual ArbitrationObservation) error {
	if expected.MemoryScopeSelected == actual.MemoryScopeSelected &&
		expected.MemoryBudgetUsed == actual.MemoryBudgetUsed &&
		expected.MemoryHits == actual.MemoryHits {
		return nil
	}
	return &ValidationError{
		Code:    ReasonCodeScopeResolutionDrift,
		Message: fmt.Sprintf("case %q %s scope resolution drift expected=%#v actual=%#v", caseName, lane, expected, actual),
	}
}

func validateMemorySearchArbitrationObservation(caseName, lane string, obs ArbitrationObservation) error {
	if obs.MemoryHits < 0 {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s memory_hits must be >= 0", caseName, lane),
		}
	}
	if len(obs.MemoryRerankStats) == 0 {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s memory_rerank_stats must not be empty", caseName, lane),
		}
	}
	required := []string{"input_total", "reranked_total", "output_total"}
	for i := range required {
		key := required[i]
		value, ok := obs.MemoryRerankStats[key]
		if !ok {
			return &ValidationError{
				Code:    ReasonCodeSchemaMismatch,
				Message: fmt.Sprintf("case %q %s memory_rerank_stats.%s is required", caseName, lane, key),
			}
		}
		if value < 0 {
			return &ValidationError{
				Code:    ReasonCodeSchemaMismatch,
				Message: fmt.Sprintf("case %q %s memory_rerank_stats.%s must be >= 0", caseName, lane, key),
			}
		}
	}
	inputTotal := obs.MemoryRerankStats["input_total"]
	rerankedTotal := obs.MemoryRerankStats["reranked_total"]
	outputTotal := obs.MemoryRerankStats["output_total"]
	if rerankedTotal > inputTotal || outputTotal > rerankedTotal {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s memory_rerank_stats must satisfy output_total<=reranked_total<=input_total", caseName, lane),
		}
	}
	return nil
}

func assertMemorySearchArbitrationEquivalent(caseName, lane string, expected, actual ArbitrationObservation) error {
	if expected.MemoryHits == actual.MemoryHits && equalIntMap(expected.MemoryRerankStats, actual.MemoryRerankStats) {
		return nil
	}
	return &ValidationError{
		Code:    ReasonCodeRetrievalQualityRegression,
		Message: fmt.Sprintf("case %q %s retrieval quality regression expected=%#v actual=%#v", caseName, lane, expected, actual),
	}
}

func validateMemoryLifecycleArbitrationObservation(caseName, lane string, obs ArbitrationObservation) error {
	switch strings.TrimSpace(obs.MemoryLifecycleAction) {
	case "retention_applied", "ttl_expired", "forget_applied", ReasonCodeRecoveryConsistencyDrift:
		return nil
	default:
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s memory_lifecycle_action is invalid", caseName, lane),
		}
	}
}

func assertMemoryLifecycleArbitrationEquivalent(caseName, lane string, expected, actual ArbitrationObservation) error {
	if expected.MemoryLifecycleAction == actual.MemoryLifecycleAction {
		return nil
	}
	if expected.MemoryLifecycleAction == ReasonCodeRecoveryConsistencyDrift ||
		actual.MemoryLifecycleAction == ReasonCodeRecoveryConsistencyDrift {
		return &ValidationError{
			Code:    ReasonCodeRecoveryConsistencyDrift,
			Message: fmt.Sprintf("case %q %s recovery consistency drift expected=%q actual=%q", caseName, lane, expected.MemoryLifecycleAction, actual.MemoryLifecycleAction),
		}
	}
	return &ValidationError{
		Code:    ReasonCodeLifecyclePolicyDrift,
		Message: fmt.Sprintf("case %q %s lifecycle policy drift expected=%q actual=%q", caseName, lane, expected.MemoryLifecycleAction, actual.MemoryLifecycleAction),
	}
}

func validateObservabilityArbitrationObservation(caseName, lane string, obs ArbitrationObservation) error {
	if !isObservabilityExportProfile(obs.ObservabilityExportProfile) {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s observability_export_profile must be none|otlp|langfuse|custom", caseName, lane),
		}
	}
	if !isObservabilityStatus(obs.ObservabilityExportStatus) {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s observability_export_status must be disabled|success|degraded|failed", caseName, lane),
		}
	}
	exportReason := strings.TrimSpace(obs.ObservabilityExportReasonCode)
	if obs.ObservabilityExportStatus == "degraded" || obs.ObservabilityExportStatus == "failed" {
		if exportReason == "" || !strings.HasPrefix(exportReason, "observability.export.") {
			return &ValidationError{
				Code:    ReasonCodeObsExportReasonDrift,
				Message: fmt.Sprintf("case %q %s observability_export_reason_code must be observability.export.* when status is degraded|failed", caseName, lane),
			}
		}
	} else if exportReason != "" {
		return &ValidationError{
			Code:    ReasonCodeObsExportReasonDrift,
			Message: fmt.Sprintf("case %q %s observability_export_reason_code must be empty when status is disabled|success", caseName, lane),
		}
	}
	if strings.TrimSpace(obs.DiagnosticsBundleLastSchemaVersion) == "" {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s diagnostics_bundle_last_schema_version is required", caseName, lane),
		}
	}
	if !isDiagnosticsBundleStatus(obs.DiagnosticsBundleLastStatus) {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s diagnostics_bundle_last_status must be disabled|success|degraded|failed", caseName, lane),
		}
	}
	bundleReason := strings.TrimSpace(obs.DiagnosticsBundleLastReasonCode)
	if obs.DiagnosticsBundleLastStatus == "degraded" || obs.DiagnosticsBundleLastStatus == "failed" {
		if bundleReason == "" || !strings.HasPrefix(bundleReason, "diagnostics.bundle.") {
			return &ValidationError{
				Code:    ReasonCodeBundleSchemaDrift,
				Message: fmt.Sprintf("case %q %s diagnostics_bundle_last_reason_code must be diagnostics.bundle.* when bundle status is degraded|failed", caseName, lane),
			}
		}
	} else if bundleReason != "" {
		return &ValidationError{
			Code:    ReasonCodeBundleSchemaDrift,
			Message: fmt.Sprintf("case %q %s diagnostics_bundle_last_reason_code must be empty when bundle status is disabled|success", caseName, lane),
		}
	}
	if obs.DiagnosticsBundleRedactionStatus != "redacted" && obs.DiagnosticsBundleRedactionStatus != "unredacted" {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s diagnostics_bundle_redaction_status must be redacted|unredacted", caseName, lane),
		}
	}
	if strings.TrimSpace(obs.DiagnosticsBundleGateFingerprint) == "" {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s diagnostics_bundle_gate_fingerprint is required", caseName, lane),
		}
	}
	return nil
}

func assertObservabilityArbitrationEquivalent(caseName, lane string, expected, actual ArbitrationObservation) error {
	if expected.ObservabilityExportProfile != actual.ObservabilityExportProfile {
		return &ValidationError{
			Code:    ReasonCodeObsExportProfileDrift,
			Message: fmt.Sprintf("case %q %s observability export profile drift expected=%q actual=%q", caseName, lane, expected.ObservabilityExportProfile, actual.ObservabilityExportProfile),
		}
	}
	if expected.ObservabilityExportStatus != actual.ObservabilityExportStatus {
		return &ValidationError{
			Code:    ReasonCodeObsExportStatusDrift,
			Message: fmt.Sprintf("case %q %s observability export status drift expected=%q actual=%q", caseName, lane, expected.ObservabilityExportStatus, actual.ObservabilityExportStatus),
		}
	}
	if expected.ObservabilityExportReasonCode != actual.ObservabilityExportReasonCode {
		return &ValidationError{
			Code:    ReasonCodeObsExportReasonDrift,
			Message: fmt.Sprintf("case %q %s observability export reason drift expected=%q actual=%q", caseName, lane, expected.ObservabilityExportReasonCode, actual.ObservabilityExportReasonCode),
		}
	}
	if expected.DiagnosticsBundleLastSchemaVersion != actual.DiagnosticsBundleLastSchemaVersion ||
		expected.DiagnosticsBundleLastStatus != actual.DiagnosticsBundleLastStatus ||
		expected.DiagnosticsBundleLastReasonCode != actual.DiagnosticsBundleLastReasonCode {
		return &ValidationError{
			Code:    ReasonCodeBundleSchemaDrift,
			Message: fmt.Sprintf("case %q %s diagnostics bundle schema drift expected=%#v actual=%#v", caseName, lane, expected, actual),
		}
	}
	if expected.DiagnosticsBundleRedactionStatus != actual.DiagnosticsBundleRedactionStatus {
		return &ValidationError{
			Code:    ReasonCodeBundleRedactionDrift,
			Message: fmt.Sprintf("case %q %s diagnostics bundle redaction drift expected=%q actual=%q", caseName, lane, expected.DiagnosticsBundleRedactionStatus, actual.DiagnosticsBundleRedactionStatus),
		}
	}
	if expected.DiagnosticsBundleGateFingerprint != actual.DiagnosticsBundleGateFingerprint {
		return &ValidationError{
			Code:    ReasonCodeBundleFingerprintDrift,
			Message: fmt.Sprintf("case %q %s diagnostics bundle fingerprint drift expected=%q actual=%q", caseName, lane, expected.DiagnosticsBundleGateFingerprint, actual.DiagnosticsBundleGateFingerprint),
		}
	}
	return &ValidationError{
		Code:    ReasonCodeSemanticDrift,
		Message: fmt.Sprintf("case %q %s observability semantic drift expected=%#v actual=%#v", caseName, lane, expected, actual),
	}
}

func isObservabilityExportProfile(profile string) bool {
	switch strings.TrimSpace(profile) {
	case runtimeconfig.RuntimeObservabilityExportProfileNone,
		runtimeconfig.RuntimeObservabilityExportProfileOTLP,
		runtimeconfig.RuntimeObservabilityExportProfileLangfuse,
		runtimeconfig.RuntimeObservabilityExportProfileCustom:
		return true
	default:
		return false
	}
}

func isObservabilityStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case "disabled", "success", "degraded", "failed":
		return true
	default:
		return false
	}
}

func isDiagnosticsBundleStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case "disabled", "success", "degraded", "failed":
		return true
	default:
		return false
	}
}

func precedenceForArbitrationCode(code string) int {
	switch strings.TrimSpace(code) {
	case runtimeconfig.RuntimePrimaryCodeTimeoutRejected, runtimeconfig.RuntimePrimaryCodeTimeoutExhausted:
		return 1
	case runtimeconfig.ReadinessCodeArbitrationVersionUnsupported, runtimeconfig.ReadinessCodeArbitrationVersionMismatch:
		return 1
	case runtimeconfig.ReadinessCodeConfigInvalid,
		runtimeconfig.ReadinessCodeStrictEscalated,
		runtimeconfig.ReadinessCodeSchedulerActivationError,
		runtimeconfig.ReadinessCodeMailboxActivationError,
		runtimeconfig.ReadinessCodeRecoveryActivationError,
		runtimeconfig.ReadinessCodeRuntimeManagerUnavailable,
		runtimeconfig.ReadinessCodeSandboxRolloutPhaseInvalid,
		runtimeconfig.ReadinessCodeSandboxRolloutFrozen:
		return 2
	case runtimeconfig.ReadinessCodeAdapterRequiredUnavailable, runtimeconfig.ReadinessCodeAdapterRequiredCircuitOpen:
		return 3
	case runtimeconfig.ReadinessCodeMemoryModeInvalid,
		runtimeconfig.ReadinessCodeMemoryProfileMissing,
		runtimeconfig.ReadinessCodeMemoryProviderNotSupported,
		runtimeconfig.ReadinessCodeMemoryFilesystemPathInvalid,
		runtimeconfig.ReadinessCodeMemoryContractVersionMismatch,
		runtimeconfig.ReadinessCodeObservabilityExportProfileInvalid,
		runtimeconfig.ReadinessCodeDiagnosticsBundlePolicyInvalid:
		return 3
	case runtimeconfig.ReadinessCodeSchedulerFallback,
		runtimeconfig.ReadinessCodeMailboxFallback,
		runtimeconfig.ReadinessCodeRecoveryFallback,
		runtimeconfig.ReadinessCodeAdapterOptionalUnavailable,
		runtimeconfig.ReadinessCodeAdapterOptionalCircuitOpen,
		runtimeconfig.ReadinessCodeAdapterDegraded,
		runtimeconfig.ReadinessCodeAdapterHalfOpenDegraded,
		runtimeconfig.ReadinessCodeMemorySPIUnavailable,
		runtimeconfig.ReadinessCodeMemoryFallbackPolicyConflict,
		runtimeconfig.ReadinessCodeMemoryFallbackTargetUnavailable,
		runtimeconfig.ReadinessCodeObservabilityExportSinkUnavailable,
		runtimeconfig.ReadinessCodeObservabilityExportAuthInvalid,
		runtimeconfig.ReadinessCodeDiagnosticsBundleOutputUnavailable,
		runtimeconfig.ReadinessCodeSandboxRolloutHealthBreached,
		runtimeconfig.ReadinessCodeSandboxRolloutCapacityBlocked:
		return 4
	default:
		return 5
	}
}

func isSupportedA50Version(version string) bool {
	normalized := strings.ToLower(strings.TrimSpace(version))
	if normalized == "" {
		return false
	}
	for _, item := range runtimeconfig.RegisteredRuntimeArbitrationRuleVersions() {
		if normalized == strings.ToLower(strings.TrimSpace(item)) {
			return true
		}
	}
	return false
}
