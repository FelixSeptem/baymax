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
	ArbitrationFixtureVersionA48V1              = "a48.v1"
	ArbitrationFixtureVersionA49V1              = "a49.v1"
	ArbitrationFixtureVersionA50V1              = "a50.v1"
	ArbitrationFixtureVersionA51V1              = "a51.v1"
	ArbitrationFixtureVersionA52V1              = "a52.v1"
	ArbitrationFixtureVersionHooksMiddlewareV1  = "hooks_middleware.v1"
	ArbitrationFixtureVersionSkillDiscoveryV1   = "skill_discovery_sources.v1"
	ArbitrationFixtureVersionSkillMappingV1     = "skill_preprocess_and_mapping.v1"
	ArbitrationFixtureVersionA57V1              = "sandbox_egress.v1"
	ArbitrationFixtureVersionBudgetAdmissionV1  = "budget_admission.v1"
	ArbitrationFixtureVersionPolicyV1           = "policy_stack.v1"
	ArbitrationFixtureVersionMemoryV1           = "memory.v1"
	ArbitrationFixtureVersionMemoryScopeV1      = "memory_scope.v1"
	ArbitrationFixtureVersionMemorySearchV1     = "memory_search.v1"
	ArbitrationFixtureVersionMemoryLifecycleV1  = "memory_lifecycle.v1"
	ArbitrationFixtureVersionObsV1              = "observability.v1"
	ArbitrationFixtureVersionReactV1            = "react.v1"
	ArbitrationFixtureVersionReactPlanV1        = "react_plan_notebook.v1"
	ArbitrationFixtureVersionRealtimeProtocolV1 = "realtime_event_protocol.v1"
	ArbitrationFixtureVersionOTelSemconvV1      = "otel_semconv.v1"
	ArbitrationFixtureVersionAgentEvalV1        = "agent_eval.v1"
	ArbitrationFixtureVersionAgentEvalDistV1    = "agent_eval_distributed.v1"
	ArbitrationFixtureVersionStateSnapshotV1    = "state_session_snapshot.v1"

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
	ReasonCodeOTelAttrMappingDrift                = "otel_attr_mapping_drift"
	ReasonCodeSpanTopologyDrift                   = "span_topology_drift"
	ReasonCodeEvalMetricDrift                     = "eval_metric_drift"
	ReasonCodeEvalAggregationDrift                = "eval_aggregation_drift"
	ReasonCodeEvalShardResumeDrift                = "eval_shard_resume_drift"
	ReasonCodeReactLoopStepDrift                  = "react_loop_step_drift"
	ReasonCodeReactToolCallBudgetDrift            = "react_tool_call_budget_drift"
	ReasonCodeReactIterationBudgetDrift           = "react_iteration_budget_drift"
	ReasonCodeReactTerminationReasonDrift         = "react_termination_reason_drift"
	ReasonCodeReactStreamDispatchDrift            = "react_stream_dispatch_drift"
	ReasonCodeReactProviderMappingDrift           = "react_provider_mapping_drift"
	ReasonCodeReactPlanVersionDrift               = "react_plan_version_drift"
	ReasonCodeReactPlanChangeReasonDrift          = "react_plan_change_reason_drift"
	ReasonCodeReactPlanHookSemanticDrift          = "react_plan_hook_semantic_drift"
	ReasonCodeReactPlanRecoverDrift               = "react_plan_recover_drift"
	ReasonCodeRealtimeEventOrderDrift             = "realtime_event_order_drift"
	ReasonCodeRealtimeInterruptSemanticDrift      = "realtime_interrupt_semantic_drift"
	ReasonCodeRealtimeResumeSemanticDrift         = "realtime_resume_semantic_drift"
	ReasonCodeRealtimeIdempotencyDrift            = "realtime_idempotency_drift"
	ReasonCodeRealtimeSequenceGapDrift            = "realtime_sequence_gap_drift"
	ReasonCodeSandboxEgressActionDrift            = "sandbox_egress_action_drift"
	ReasonCodeSandboxEgressPolicySourceDrift      = "sandbox_egress_policy_source_drift"
	ReasonCodeSandboxEgressViolationTaxonomyDrift = "sandbox_egress_violation_taxonomy_drift"
	ReasonCodeAdapterAllowlistDecisionDrift       = "adapter_allowlist_decision_drift"
	ReasonCodeAdapterAllowlistTaxonomyDrift       = "adapter_allowlist_taxonomy_drift"
	ReasonCodeBudgetThresholdDrift                = "budget_threshold_drift"
	ReasonCodeAdmissionDecisionDrift              = "admission_decision_drift"
	ReasonCodeDegradePolicyDrift                  = "degrade_policy_drift"
	ReasonCodeHooksOrderDrift                     = "hooks_order_drift"
	ReasonCodeSkillDiscoverySourceDrift           = "skill_discovery_source_drift"
	ReasonCodeSkillBundleMappingDrift             = "skill_bundle_mapping_drift"
	ReasonCodeScopeResolutionDrift                = "scope_resolution_drift"
	ReasonCodeRetrievalQualityRegression          = "retrieval_quality_regression"
	ReasonCodeLifecyclePolicyDrift                = "lifecycle_policy_drift"
	ReasonCodeRecoveryConsistencyDrift            = "recovery_consistency_drift"
	ReasonCodeSnapshotSchemaDrift                 = "snapshot_schema_drift"
	ReasonCodeStateRestoreSemanticDrift           = "state_restore_semantic_drift"
	ReasonCodeSnapshotCompatWindowDrift           = "snapshot_compat_window_drift"
	ReasonCodePartialRestorePolicyDrift           = "partial_restore_policy_drift"
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
	HooksEnabled                           bool                      `json:"hooks_enabled,omitempty"`
	HooksFailMode                          string                    `json:"hooks_fail_mode,omitempty"`
	HooksPhases                            []string                  `json:"hooks_phases,omitempty"`
	ToolMiddlewareEnabled                  bool                      `json:"tool_middleware_enabled,omitempty"`
	ToolMiddlewareFailMode                 string                    `json:"tool_middleware_fail_mode,omitempty"`
	SkillDiscoveryMode                     string                    `json:"skill_discovery_mode,omitempty"`
	SkillDiscoveryRoots                    []string                  `json:"skill_discovery_roots,omitempty"`
	SkillPreprocessEnabled                 bool                      `json:"skill_preprocess_enabled,omitempty"`
	SkillPreprocessPhase                   string                    `json:"skill_preprocess_phase,omitempty"`
	SkillPreprocessFailMode                string                    `json:"skill_preprocess_fail_mode,omitempty"`
	SkillPreprocessStatus                  string                    `json:"skill_preprocess_status,omitempty"`
	SkillPreprocessReasonCode              string                    `json:"skill_preprocess_reason_code,omitempty"`
	SkillPreprocessSpecCount               int                       `json:"skill_preprocess_spec_count,omitempty"`
	SkillBundlePromptMode                  string                    `json:"skill_bundle_prompt_mode,omitempty"`
	SkillBundleWhitelistMode               string                    `json:"skill_bundle_whitelist_mode,omitempty"`
	SkillBundleConflictPolicy              string                    `json:"skill_bundle_conflict_policy,omitempty"`
	SkillBundlePromptTotal                 int                       `json:"skill_bundle_prompt_total,omitempty"`
	SkillBundleWhitelistTotal              int                       `json:"skill_bundle_whitelist_total,omitempty"`
	SkillBundleWhitelistRejectedTotal      int                       `json:"skill_bundle_whitelist_rejected_total,omitempty"`
	ModelProvider                          string                    `json:"model_provider,omitempty"`
	ReactEnabled                           bool                      `json:"react_enabled,omitempty"`
	ReactIterationTotal                    int                       `json:"react_iteration_total,omitempty"`
	ReactToolCallTotal                     int                       `json:"react_tool_call_total,omitempty"`
	ReactToolCallBudgetHitTotal            int                       `json:"react_tool_call_budget_hit_total,omitempty"`
	ReactIterationBudgetHitTotal           int                       `json:"react_iteration_budget_hit_total,omitempty"`
	ReactTerminationReason                 string                    `json:"react_termination_reason,omitempty"`
	ReactStreamDispatchEnabled             bool                      `json:"react_stream_dispatch_enabled,omitempty"`
	ReactPlanID                            string                    `json:"react_plan_id,omitempty"`
	ReactPlanVersion                       int                       `json:"react_plan_version,omitempty"`
	ReactPlanChangeTotal                   int                       `json:"react_plan_change_total,omitempty"`
	ReactPlanLastAction                    string                    `json:"react_plan_last_action,omitempty"`
	ReactPlanChangeReason                  string                    `json:"react_plan_change_reason,omitempty"`
	ReactPlanRecoverCount                  int                       `json:"react_plan_recover_count,omitempty"`
	ReactPlanHookStatus                    string                    `json:"react_plan_hook_status,omitempty"`
	RealtimeProtocolVersion                string                    `json:"realtime_protocol_version,omitempty"`
	RealtimeEventSeqMax                    int64                     `json:"realtime_event_seq_max,omitempty"`
	RealtimeInterruptTotal                 int                       `json:"realtime_interrupt_total,omitempty"`
	RealtimeResumeTotal                    int                       `json:"realtime_resume_total,omitempty"`
	RealtimeResumeSource                   string                    `json:"realtime_resume_source,omitempty"`
	RealtimeIdempotencyDedupTotal          int                       `json:"realtime_idempotency_dedup_total,omitempty"`
	RealtimeLastErrorCode                  string                    `json:"realtime_last_error_code,omitempty"`
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
	TraceExportStatus                      string                    `json:"trace_export_status,omitempty"`
	TraceSchemaVersion                     string                    `json:"trace_schema_version,omitempty"`
	TraceTopologyClass                     string                    `json:"trace_topology_class,omitempty"`
	TraceCanonicalAttrKeys                 []string                  `json:"trace_canonical_attr_keys,omitempty"`
	EvalSuiteID                            string                    `json:"eval_suite_id,omitempty"`
	EvalSummary                            map[string]any            `json:"eval_summary,omitempty"`
	EvalExecutionMode                      string                    `json:"eval_execution_mode,omitempty"`
	EvalJobID                              string                    `json:"eval_job_id,omitempty"`
	EvalShardTotal                         int                       `json:"eval_shard_total,omitempty"`
	EvalResumeCount                        int                       `json:"eval_resume_count,omitempty"`
	StateSnapshotVersion                   string                    `json:"state_snapshot_version,omitempty"`
	StateRestoreAction                     string                    `json:"state_restore_action,omitempty"`
	StateRestoreConflictCode               string                    `json:"state_restore_conflict_code,omitempty"`
	StateRestoreSource                     string                    `json:"state_restore_source,omitempty"`
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
		version != ArbitrationFixtureVersionHooksMiddlewareV1 &&
		version != ArbitrationFixtureVersionSkillDiscoveryV1 &&
		version != ArbitrationFixtureVersionSkillMappingV1 &&
		version != ArbitrationFixtureVersionA57V1 &&
		version != ArbitrationFixtureVersionBudgetAdmissionV1 &&
		version != ArbitrationFixtureVersionPolicyV1 &&
		version != ArbitrationFixtureVersionMemoryV1 &&
		version != ArbitrationFixtureVersionMemoryScopeV1 &&
		version != ArbitrationFixtureVersionMemorySearchV1 &&
		version != ArbitrationFixtureVersionMemoryLifecycleV1 &&
		version != ArbitrationFixtureVersionObsV1 &&
		version != ArbitrationFixtureVersionReactV1 &&
		version != ArbitrationFixtureVersionReactPlanV1 &&
		version != ArbitrationFixtureVersionRealtimeProtocolV1 &&
		version != ArbitrationFixtureVersionOTelSemconvV1 &&
		version != ArbitrationFixtureVersionAgentEvalV1 &&
		version != ArbitrationFixtureVersionAgentEvalDistV1 &&
		version != ArbitrationFixtureVersionStateSnapshotV1 {
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
		HooksEnabled:                           in.HooksEnabled,
		HooksFailMode:                          strings.ToLower(strings.TrimSpace(in.HooksFailMode)),
		ToolMiddlewareEnabled:                  in.ToolMiddlewareEnabled,
		ToolMiddlewareFailMode:                 strings.ToLower(strings.TrimSpace(in.ToolMiddlewareFailMode)),
		SkillDiscoveryMode:                     strings.ToLower(strings.TrimSpace(in.SkillDiscoveryMode)),
		SkillPreprocessEnabled:                 in.SkillPreprocessEnabled,
		SkillPreprocessPhase:                   strings.ToLower(strings.TrimSpace(in.SkillPreprocessPhase)),
		SkillPreprocessFailMode:                strings.ToLower(strings.TrimSpace(in.SkillPreprocessFailMode)),
		SkillPreprocessStatus:                  strings.ToLower(strings.TrimSpace(in.SkillPreprocessStatus)),
		SkillPreprocessReasonCode:              strings.ToLower(strings.TrimSpace(in.SkillPreprocessReasonCode)),
		SkillPreprocessSpecCount:               in.SkillPreprocessSpecCount,
		SkillBundlePromptMode:                  strings.ToLower(strings.TrimSpace(in.SkillBundlePromptMode)),
		SkillBundleWhitelistMode:               strings.ToLower(strings.TrimSpace(in.SkillBundleWhitelistMode)),
		SkillBundleConflictPolicy:              strings.ToLower(strings.TrimSpace(in.SkillBundleConflictPolicy)),
		SkillBundlePromptTotal:                 in.SkillBundlePromptTotal,
		SkillBundleWhitelistTotal:              in.SkillBundleWhitelistTotal,
		SkillBundleWhitelistRejectedTotal:      in.SkillBundleWhitelistRejectedTotal,
		ModelProvider:                          strings.ToLower(strings.TrimSpace(in.ModelProvider)),
		ReactEnabled:                           in.ReactEnabled,
		ReactIterationTotal:                    in.ReactIterationTotal,
		ReactToolCallTotal:                     in.ReactToolCallTotal,
		ReactToolCallBudgetHitTotal:            in.ReactToolCallBudgetHitTotal,
		ReactIterationBudgetHitTotal:           in.ReactIterationBudgetHitTotal,
		ReactTerminationReason:                 strings.ToLower(strings.TrimSpace(in.ReactTerminationReason)),
		ReactStreamDispatchEnabled:             in.ReactStreamDispatchEnabled,
		ReactPlanID:                            strings.ToLower(strings.TrimSpace(in.ReactPlanID)),
		ReactPlanVersion:                       in.ReactPlanVersion,
		ReactPlanChangeTotal:                   in.ReactPlanChangeTotal,
		ReactPlanLastAction:                    strings.ToLower(strings.TrimSpace(in.ReactPlanLastAction)),
		ReactPlanChangeReason:                  strings.ToLower(strings.TrimSpace(in.ReactPlanChangeReason)),
		ReactPlanRecoverCount:                  in.ReactPlanRecoverCount,
		ReactPlanHookStatus:                    strings.ToLower(strings.TrimSpace(in.ReactPlanHookStatus)),
		RealtimeProtocolVersion:                strings.ToLower(strings.TrimSpace(in.RealtimeProtocolVersion)),
		RealtimeEventSeqMax:                    in.RealtimeEventSeqMax,
		RealtimeInterruptTotal:                 in.RealtimeInterruptTotal,
		RealtimeResumeTotal:                    in.RealtimeResumeTotal,
		RealtimeResumeSource:                   strings.ToLower(strings.TrimSpace(in.RealtimeResumeSource)),
		RealtimeIdempotencyDedupTotal:          in.RealtimeIdempotencyDedupTotal,
		RealtimeLastErrorCode:                  strings.ToLower(strings.TrimSpace(in.RealtimeLastErrorCode)),
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
		TraceExportStatus:                      strings.ToLower(strings.TrimSpace(in.TraceExportStatus)),
		TraceSchemaVersion:                     strings.ToLower(strings.TrimSpace(in.TraceSchemaVersion)),
		TraceTopologyClass:                     strings.ToLower(strings.TrimSpace(in.TraceTopologyClass)),
		EvalSuiteID:                            strings.ToLower(strings.TrimSpace(in.EvalSuiteID)),
		EvalSummary:                            canonicalizeAnyMap(in.EvalSummary),
		EvalExecutionMode:                      strings.ToLower(strings.TrimSpace(in.EvalExecutionMode)),
		EvalJobID:                              strings.ToLower(strings.TrimSpace(in.EvalJobID)),
		EvalShardTotal:                         in.EvalShardTotal,
		EvalResumeCount:                        in.EvalResumeCount,
		StateSnapshotVersion:                   strings.ToLower(strings.TrimSpace(in.StateSnapshotVersion)),
		StateRestoreAction:                     strings.ToLower(strings.TrimSpace(in.StateRestoreAction)),
		StateRestoreConflictCode:               strings.ToLower(strings.TrimSpace(in.StateRestoreConflictCode)),
		StateRestoreSource:                     strings.ToLower(strings.TrimSpace(in.StateRestoreSource)),
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
	if out.ReactPlanVersion < 0 {
		out.ReactPlanVersion = 0
	}
	if out.ReactPlanChangeTotal < 0 {
		out.ReactPlanChangeTotal = 0
	}
	if out.ReactPlanRecoverCount < 0 {
		out.ReactPlanRecoverCount = 0
	}
	if out.RealtimeEventSeqMax < 0 {
		out.RealtimeEventSeqMax = 0
	}
	if out.RealtimeInterruptTotal < 0 {
		out.RealtimeInterruptTotal = 0
	}
	if out.RealtimeResumeTotal < 0 {
		out.RealtimeResumeTotal = 0
	}
	if out.RealtimeIdempotencyDedupTotal < 0 {
		out.RealtimeIdempotencyDedupTotal = 0
	}
	if out.SkillPreprocessSpecCount < 0 {
		out.SkillPreprocessSpecCount = 0
	}
	if out.SkillBundlePromptTotal < 0 {
		out.SkillBundlePromptTotal = 0
	}
	if out.SkillBundleWhitelistTotal < 0 {
		out.SkillBundleWhitelistTotal = 0
	}
	if out.SkillBundleWhitelistRejectedTotal < 0 {
		out.SkillBundleWhitelistRejectedTotal = 0
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
	if out.EvalShardTotal < 0 {
		out.EvalShardTotal = 0
	}
	if out.EvalResumeCount < 0 {
		out.EvalResumeCount = 0
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
	for i := range in.HooksPhases {
		item := strings.ToLower(strings.TrimSpace(in.HooksPhases[i]))
		if item == "" {
			continue
		}
		out.HooksPhases = append(out.HooksPhases, item)
	}
	if len(out.HooksPhases) == 0 {
		out.HooksPhases = nil
	}
	for i := range in.SkillDiscoveryRoots {
		item := strings.TrimSpace(in.SkillDiscoveryRoots[i])
		if item == "" {
			continue
		}
		out.SkillDiscoveryRoots = append(out.SkillDiscoveryRoots, item)
	}
	if len(out.SkillDiscoveryRoots) == 0 {
		out.SkillDiscoveryRoots = nil
	}
	traceAttrSet := map[string]struct{}{}
	for i := range in.TraceCanonicalAttrKeys {
		item := strings.ToLower(strings.TrimSpace(in.TraceCanonicalAttrKeys[i]))
		if item == "" {
			continue
		}
		traceAttrSet[item] = struct{}{}
	}
	if len(traceAttrSet) > 0 {
		out.TraceCanonicalAttrKeys = make([]string, 0, len(traceAttrSet))
		for item := range traceAttrSet {
			out.TraceCanonicalAttrKeys = append(out.TraceCanonicalAttrKeys, item)
		}
		sort.Strings(out.TraceCanonicalAttrKeys)
	}
	out.BudgetSnapshot = canonicalizeBudgetAdmissionSnapshot(in.BudgetSnapshot)
	return out
}

func validateArbitrationObservation(version, caseName, lane string, obs ArbitrationObservation) error {
	if version == ArbitrationFixtureVersionHooksMiddlewareV1 {
		return validateHooksMiddlewareArbitrationObservation(caseName, lane, obs)
	}
	if version == ArbitrationFixtureVersionSkillDiscoveryV1 {
		return validateSkillDiscoveryArbitrationObservation(caseName, lane, obs)
	}
	if version == ArbitrationFixtureVersionSkillMappingV1 {
		return validateSkillMappingArbitrationObservation(caseName, lane, obs)
	}
	if version == ArbitrationFixtureVersionOTelSemconvV1 {
		return validateOTelSemconvArbitrationObservation(caseName, lane, obs)
	}
	if version == ArbitrationFixtureVersionAgentEvalV1 {
		return validateAgentEvalArbitrationObservation(caseName, lane, obs)
	}
	if version == ArbitrationFixtureVersionAgentEvalDistV1 {
		return validateAgentEvalDistributedArbitrationObservation(caseName, lane, obs)
	}
	if version == ArbitrationFixtureVersionStateSnapshotV1 {
		return validateStateSessionSnapshotArbitrationObservation(caseName, lane, obs)
	}
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
	if version == ArbitrationFixtureVersionReactPlanV1 {
		return validateReactPlanNotebookArbitrationObservation(caseName, lane, obs)
	}
	if version == ArbitrationFixtureVersionRealtimeProtocolV1 {
		return validateRealtimeProtocolArbitrationObservation(caseName, lane, obs)
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
	if version == ArbitrationFixtureVersionHooksMiddlewareV1 {
		return assertHooksMiddlewareArbitrationEquivalent(caseName, lane, expected, actual)
	}
	if version == ArbitrationFixtureVersionSkillDiscoveryV1 {
		return assertSkillDiscoveryArbitrationEquivalent(caseName, lane, expected, actual)
	}
	if version == ArbitrationFixtureVersionSkillMappingV1 {
		return assertSkillMappingArbitrationEquivalent(caseName, lane, expected, actual)
	}
	if version == ArbitrationFixtureVersionOTelSemconvV1 {
		return assertOTelSemconvArbitrationEquivalent(caseName, lane, expected, actual)
	}
	if version == ArbitrationFixtureVersionAgentEvalV1 {
		return assertAgentEvalArbitrationEquivalent(caseName, lane, expected, actual)
	}
	if version == ArbitrationFixtureVersionAgentEvalDistV1 {
		return assertAgentEvalDistributedArbitrationEquivalent(caseName, lane, expected, actual)
	}
	if version == ArbitrationFixtureVersionStateSnapshotV1 {
		return assertStateSessionSnapshotArbitrationEquivalent(caseName, lane, expected, actual)
	}
	if version == ArbitrationFixtureVersionReactV1 {
		return assertReactArbitrationEquivalent(caseName, lane, expected, actual)
	}
	if version == ArbitrationFixtureVersionReactPlanV1 {
		return assertReactPlanNotebookArbitrationEquivalent(caseName, lane, expected, actual)
	}
	if version == ArbitrationFixtureVersionRealtimeProtocolV1 {
		return assertRealtimeProtocolArbitrationEquivalent(caseName, lane, expected, actual)
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

func validateStateSessionSnapshotArbitrationObservation(caseName, lane string, obs ArbitrationObservation) error {
	if strings.TrimSpace(obs.StateSnapshotVersion) != ArbitrationFixtureVersionStateSnapshotV1 {
		return &ValidationError{
			Code:    ReasonCodeSnapshotSchemaDrift,
			Message: fmt.Sprintf("case %q %s state_snapshot_version must be %q", caseName, lane, ArbitrationFixtureVersionStateSnapshotV1),
		}
	}
	if !isCanonicalStateRestoreAction(obs.StateRestoreAction) {
		return &ValidationError{
			Code:    ReasonCodeSnapshotSchemaDrift,
			Message: fmt.Sprintf("case %q %s state_restore_action is not canonical: %q", caseName, lane, obs.StateRestoreAction),
		}
	}
	if strings.TrimSpace(obs.StateRestoreSource) == "" {
		return &ValidationError{
			Code:    ReasonCodeSnapshotSchemaDrift,
			Message: fmt.Sprintf("case %q %s state_restore_source is required", caseName, lane),
		}
	}
	if strings.TrimSpace(obs.StateRestoreConflictCode) != "" &&
		!isCanonicalStateRestoreConflictCode(obs.StateRestoreConflictCode) {
		return &ValidationError{
			Code:    ReasonCodeSnapshotSchemaDrift,
			Message: fmt.Sprintf("case %q %s state_restore_conflict_code is not canonical: %q", caseName, lane, obs.StateRestoreConflictCode),
		}
	}
	if obs.StateRestoreAction == "compatible_bounded_restore" && strings.TrimSpace(obs.StateRestoreConflictCode) == "" {
		return &ValidationError{
			Code:    ReasonCodePartialRestorePolicyDrift,
			Message: fmt.Sprintf("case %q %s compatible_bounded_restore requires state_restore_conflict_code", caseName, lane),
		}
	}
	if obs.StateRestoreAction != "compatible_bounded_restore" && strings.TrimSpace(obs.StateRestoreConflictCode) != "" {
		return &ValidationError{
			Code:    ReasonCodeStateRestoreSemanticDrift,
			Message: fmt.Sprintf("case %q %s state_restore_conflict_code must be empty for action=%q", caseName, lane, obs.StateRestoreAction),
		}
	}
	return nil
}

func assertStateSessionSnapshotArbitrationEquivalent(caseName, lane string, expected, actual ArbitrationObservation) error {
	if expected.StateSnapshotVersion != actual.StateSnapshotVersion {
		return &ValidationError{
			Code:    ReasonCodeSnapshotSchemaDrift,
			Message: fmt.Sprintf("case %q %s snapshot schema drift expected=%q actual=%q", caseName, lane, expected.StateSnapshotVersion, actual.StateSnapshotVersion),
		}
	}
	if isStrictRestoreAction(expected.StateRestoreAction) != isStrictRestoreAction(actual.StateRestoreAction) {
		return &ValidationError{
			Code:    ReasonCodeSnapshotCompatWindowDrift,
			Message: fmt.Sprintf("case %q %s compat window drift expected_action=%q actual_action=%q", caseName, lane, expected.StateRestoreAction, actual.StateRestoreAction),
		}
	}
	if expected.StateRestoreAction == "compatible_bounded_restore" || actual.StateRestoreAction == "compatible_bounded_restore" {
		if expected.StateRestoreAction != actual.StateRestoreAction ||
			expected.StateRestoreConflictCode != actual.StateRestoreConflictCode ||
			expected.StateRestoreSource != actual.StateRestoreSource {
			return &ValidationError{
				Code:    ReasonCodePartialRestorePolicyDrift,
				Message: fmt.Sprintf("case %q %s partial restore policy drift expected=%#v actual=%#v", caseName, lane, expected, actual),
			}
		}
		return nil
	}
	if expected.StateRestoreAction != actual.StateRestoreAction ||
		expected.StateRestoreConflictCode != actual.StateRestoreConflictCode ||
		expected.StateRestoreSource != actual.StateRestoreSource {
		return &ValidationError{
			Code:    ReasonCodeStateRestoreSemanticDrift,
			Message: fmt.Sprintf("case %q %s state restore semantic drift expected=%#v actual=%#v", caseName, lane, expected, actual),
		}
	}
	return nil
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

func validateReactPlanNotebookArbitrationObservation(caseName, lane string, obs ArbitrationObservation) error {
	if err := validateReactArbitrationObservation(caseName, lane, obs); err != nil {
		return err
	}
	if strings.TrimSpace(obs.ReactPlanID) == "" {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s react_plan_id is required", caseName, lane),
		}
	}
	if obs.ReactPlanVersion <= 0 {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s react_plan_version must be > 0", caseName, lane),
		}
	}
	if obs.ReactPlanChangeTotal <= 0 {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s react_plan_change_total must be > 0", caseName, lane),
		}
	}
	if !isCanonicalReactPlanAction(obs.ReactPlanLastAction) {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s react_plan_last_action is invalid: %q", caseName, lane, obs.ReactPlanLastAction),
		}
	}
	if strings.TrimSpace(obs.ReactPlanChangeReason) == "" {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s react_plan_change_reason is required", caseName, lane),
		}
	}
	if obs.ReactPlanRecoverCount < 0 || obs.ReactPlanRecoverCount > obs.ReactPlanChangeTotal {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s react_plan_recover_count must be in [0, react_plan_change_total]", caseName, lane),
		}
	}
	if !isCanonicalReactPlanHookStatus(obs.ReactPlanHookStatus) {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s react_plan_hook_status is invalid: %q", caseName, lane, obs.ReactPlanHookStatus),
		}
	}
	return nil
}

func assertReactPlanNotebookArbitrationEquivalent(caseName, lane string, expected, actual ArbitrationObservation) error {
	if err := assertReactArbitrationEquivalent(caseName, lane, expected, actual); err != nil {
		return err
	}
	if expected.ReactPlanVersion != actual.ReactPlanVersion {
		return &ValidationError{
			Code:    ReasonCodeReactPlanVersionDrift,
			Message: fmt.Sprintf("case %q %s react plan version drift expected=%d actual=%d", caseName, lane, expected.ReactPlanVersion, actual.ReactPlanVersion),
		}
	}
	if expected.ReactPlanChangeReason != actual.ReactPlanChangeReason {
		return &ValidationError{
			Code:    ReasonCodeReactPlanChangeReasonDrift,
			Message: fmt.Sprintf("case %q %s react plan change reason drift expected=%q actual=%q", caseName, lane, expected.ReactPlanChangeReason, actual.ReactPlanChangeReason),
		}
	}
	if expected.ReactPlanID != actual.ReactPlanID ||
		expected.ReactPlanChangeTotal != actual.ReactPlanChangeTotal ||
		expected.ReactPlanLastAction != actual.ReactPlanLastAction ||
		expected.ReactPlanHookStatus != actual.ReactPlanHookStatus {
		return &ValidationError{
			Code:    ReasonCodeReactPlanHookSemanticDrift,
			Message: fmt.Sprintf("case %q %s react plan hook semantic drift expected=%#v actual=%#v", caseName, lane, expected, actual),
		}
	}
	if expected.ReactPlanRecoverCount != actual.ReactPlanRecoverCount {
		return &ValidationError{
			Code:    ReasonCodeReactPlanRecoverDrift,
			Message: fmt.Sprintf("case %q %s react plan recover drift expected=%d actual=%d", caseName, lane, expected.ReactPlanRecoverCount, actual.ReactPlanRecoverCount),
		}
	}
	return nil
}

func validateRealtimeProtocolArbitrationObservation(caseName, lane string, obs ArbitrationObservation) error {
	if err := validateReactArbitrationObservation(caseName, lane, obs); err != nil {
		return err
	}
	if strings.TrimSpace(obs.RealtimeProtocolVersion) != ArbitrationFixtureVersionRealtimeProtocolV1 {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s realtime_protocol_version must be %q", caseName, lane, ArbitrationFixtureVersionRealtimeProtocolV1),
		}
	}
	if obs.RealtimeEventSeqMax <= 0 {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s realtime_event_seq_max must be > 0", caseName, lane),
		}
	}
	if obs.RealtimeInterruptTotal < 0 ||
		obs.RealtimeResumeTotal < 0 ||
		obs.RealtimeIdempotencyDedupTotal < 0 {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s realtime counters must be >= 0", caseName, lane),
		}
	}
	if obs.RealtimeResumeTotal > obs.RealtimeInterruptTotal {
		return &ValidationError{
			Code:    ReasonCodeRealtimeResumeSemanticDrift,
			Message: fmt.Sprintf("case %q %s realtime_resume_total must be <= realtime_interrupt_total", caseName, lane),
		}
	}
	if !isCanonicalRealtimeResumeSource(obs.RealtimeResumeSource) {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s realtime_resume_source must be cursor or empty", caseName, lane),
		}
	}
	if obs.RealtimeResumeTotal > 0 && strings.TrimSpace(obs.RealtimeResumeSource) == "" {
		return &ValidationError{
			Code:    ReasonCodeRealtimeResumeSemanticDrift,
			Message: fmt.Sprintf("case %q %s realtime_resume_source is required when realtime_resume_total > 0", caseName, lane),
		}
	}
	if obs.RealtimeResumeTotal == 0 && strings.TrimSpace(obs.RealtimeResumeSource) != "" {
		return &ValidationError{
			Code:    ReasonCodeRealtimeResumeSemanticDrift,
			Message: fmt.Sprintf("case %q %s realtime_resume_source must be empty when realtime_resume_total == 0", caseName, lane),
		}
	}
	if !isCanonicalRealtimeReasonCode(obs.RealtimeLastErrorCode) {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s realtime_last_error_code is not canonical: %q", caseName, lane, obs.RealtimeLastErrorCode),
		}
	}
	return nil
}

func assertRealtimeProtocolArbitrationEquivalent(caseName, lane string, expected, actual ArbitrationObservation) error {
	if err := assertReactArbitrationEquivalent(caseName, lane, expected, actual); err != nil {
		return err
	}
	if expected.RealtimeProtocolVersion != actual.RealtimeProtocolVersion {
		return &ValidationError{
			Code:    ReasonCodeRealtimeEventOrderDrift,
			Message: fmt.Sprintf("case %q %s realtime protocol version drift expected=%q actual=%q", caseName, lane, expected.RealtimeProtocolVersion, actual.RealtimeProtocolVersion),
		}
	}
	if expected.RealtimeEventSeqMax != actual.RealtimeEventSeqMax {
		return &ValidationError{
			Code:    ReasonCodeRealtimeSequenceGapDrift,
			Message: fmt.Sprintf("case %q %s realtime sequence gap drift expected=%d actual=%d", caseName, lane, expected.RealtimeEventSeqMax, actual.RealtimeEventSeqMax),
		}
	}
	if expected.RealtimeInterruptTotal != actual.RealtimeInterruptTotal {
		return &ValidationError{
			Code:    ReasonCodeRealtimeInterruptSemanticDrift,
			Message: fmt.Sprintf("case %q %s realtime interrupt semantic drift expected=%d actual=%d", caseName, lane, expected.RealtimeInterruptTotal, actual.RealtimeInterruptTotal),
		}
	}
	if expected.RealtimeResumeTotal != actual.RealtimeResumeTotal ||
		expected.RealtimeResumeSource != actual.RealtimeResumeSource {
		return &ValidationError{
			Code:    ReasonCodeRealtimeResumeSemanticDrift,
			Message: fmt.Sprintf("case %q %s realtime resume semantic drift expected=%#v actual=%#v", caseName, lane, expected, actual),
		}
	}
	if expected.RealtimeIdempotencyDedupTotal != actual.RealtimeIdempotencyDedupTotal {
		return &ValidationError{
			Code:    ReasonCodeRealtimeIdempotencyDrift,
			Message: fmt.Sprintf("case %q %s realtime idempotency drift expected=%d actual=%d", caseName, lane, expected.RealtimeIdempotencyDedupTotal, actual.RealtimeIdempotencyDedupTotal),
		}
	}
	if expected.RealtimeLastErrorCode != actual.RealtimeLastErrorCode {
		switch {
		case isRealtimeOrderReasonCode(expected.RealtimeLastErrorCode) || isRealtimeOrderReasonCode(actual.RealtimeLastErrorCode):
			return &ValidationError{
				Code:    ReasonCodeRealtimeEventOrderDrift,
				Message: fmt.Sprintf("case %q %s realtime event order drift expected=%q actual=%q", caseName, lane, expected.RealtimeLastErrorCode, actual.RealtimeLastErrorCode),
			}
		case strings.Contains(expected.RealtimeLastErrorCode, "resume.") || strings.Contains(actual.RealtimeLastErrorCode, "resume."):
			return &ValidationError{
				Code:    ReasonCodeRealtimeResumeSemanticDrift,
				Message: fmt.Sprintf("case %q %s realtime resume semantic drift expected=%q actual=%q", caseName, lane, expected.RealtimeLastErrorCode, actual.RealtimeLastErrorCode),
			}
		case strings.Contains(expected.RealtimeLastErrorCode, "interrupt.") || strings.Contains(actual.RealtimeLastErrorCode, "interrupt."):
			return &ValidationError{
				Code:    ReasonCodeRealtimeInterruptSemanticDrift,
				Message: fmt.Sprintf("case %q %s realtime interrupt semantic drift expected=%q actual=%q", caseName, lane, expected.RealtimeLastErrorCode, actual.RealtimeLastErrorCode),
			}
		default:
			return &ValidationError{
				Code:    ReasonCodeRealtimeEventOrderDrift,
				Message: fmt.Sprintf("case %q %s realtime error code drift expected=%q actual=%q", caseName, lane, expected.RealtimeLastErrorCode, actual.RealtimeLastErrorCode),
			}
		}
	}
	return &ValidationError{
		Code:    ReasonCodeSemanticDrift,
		Message: fmt.Sprintf("case %q %s realtime semantic drift expected=%#v actual=%#v", caseName, lane, expected, actual),
	}
}

func validateHooksMiddlewareArbitrationObservation(caseName, lane string, obs ArbitrationObservation) error {
	if !isHooksFailMode(obs.HooksFailMode) {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s hooks_fail_mode must be fail_fast|degrade", caseName, lane),
		}
	}
	if !isToolMiddlewareFailMode(obs.ToolMiddlewareFailMode) {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s tool_middleware_fail_mode must be fail_fast|degrade", caseName, lane),
		}
	}
	if len(obs.HooksPhases) == 0 {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s hooks_phases must not be empty", caseName, lane),
		}
	}
	for i := range obs.HooksPhases {
		if !isHookPhase(obs.HooksPhases[i]) {
			return &ValidationError{
				Code:    ReasonCodeSchemaMismatch,
				Message: fmt.Sprintf("case %q %s hooks_phases[%d] is invalid: %q", caseName, lane, i, obs.HooksPhases[i]),
			}
		}
	}
	if !equalStringSlice(obs.HooksPhases, canonicalHookPhases()) {
		return &ValidationError{
			Code:    ReasonCodeHooksOrderDrift,
			Message: fmt.Sprintf("case %q %s hooks order drift expected=%v actual=%v", caseName, lane, canonicalHookPhases(), obs.HooksPhases),
		}
	}
	return nil
}

func assertHooksMiddlewareArbitrationEquivalent(caseName, lane string, expected, actual ArbitrationObservation) error {
	if !equalStringSlice(expected.HooksPhases, actual.HooksPhases) {
		return &ValidationError{
			Code:    ReasonCodeHooksOrderDrift,
			Message: fmt.Sprintf("case %q %s hooks order drift expected=%v actual=%v", caseName, lane, expected.HooksPhases, actual.HooksPhases),
		}
	}
	if expected.HooksEnabled != actual.HooksEnabled ||
		expected.HooksFailMode != actual.HooksFailMode ||
		expected.ToolMiddlewareEnabled != actual.ToolMiddlewareEnabled ||
		expected.ToolMiddlewareFailMode != actual.ToolMiddlewareFailMode {
		return &ValidationError{
			Code:    ReasonCodeSemanticDrift,
			Message: fmt.Sprintf("case %q %s hooks/middleware semantic drift expected=%#v actual=%#v", caseName, lane, expected, actual),
		}
	}
	return nil
}

func validateSkillDiscoveryArbitrationObservation(caseName, lane string, obs ArbitrationObservation) error {
	if !isSkillDiscoveryMode(obs.SkillDiscoveryMode) {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s skill_discovery_mode must be agents_md|folder|hybrid", caseName, lane),
		}
	}
	if (obs.SkillDiscoveryMode == runtimeconfig.RuntimeSkillDiscoveryModeFolder || obs.SkillDiscoveryMode == runtimeconfig.RuntimeSkillDiscoveryModeHybrid) &&
		len(obs.SkillDiscoveryRoots) == 0 {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s skill_discovery_roots must not be empty when discovery mode is folder|hybrid", caseName, lane),
		}
	}
	for i := range obs.SkillDiscoveryRoots {
		if strings.TrimSpace(obs.SkillDiscoveryRoots[i]) == "" {
			return &ValidationError{
				Code:    ReasonCodeSchemaMismatch,
				Message: fmt.Sprintf("case %q %s skill_discovery_roots[%d] must not be empty", caseName, lane, i),
			}
		}
	}
	if obs.SkillPreprocessSpecCount <= 0 {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s skill_preprocess_spec_count must be > 0", caseName, lane),
		}
	}
	return nil
}

func assertSkillDiscoveryArbitrationEquivalent(caseName, lane string, expected, actual ArbitrationObservation) error {
	if expected.SkillDiscoveryMode != actual.SkillDiscoveryMode ||
		!equalStringSlice(expected.SkillDiscoveryRoots, actual.SkillDiscoveryRoots) ||
		expected.SkillPreprocessSpecCount != actual.SkillPreprocessSpecCount {
		return &ValidationError{
			Code:    ReasonCodeSkillDiscoverySourceDrift,
			Message: fmt.Sprintf("case %q %s skill discovery source drift expected=%#v actual=%#v", caseName, lane, expected, actual),
		}
	}
	return nil
}

func validateSkillMappingArbitrationObservation(caseName, lane string, obs ArbitrationObservation) error {
	if !isSkillPreprocessPhase(obs.SkillPreprocessPhase) {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s skill_preprocess_phase must be before_run_stream", caseName, lane),
		}
	}
	if !isSkillPreprocessFailMode(obs.SkillPreprocessFailMode) {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s skill_preprocess_fail_mode must be fail_fast|degrade", caseName, lane),
		}
	}
	if !isSkillPreprocessStatus(obs.SkillPreprocessStatus) {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s skill_preprocess_status must be success|failed|degraded|skipped", caseName, lane),
		}
	}
	if !isSkillBundlePromptMode(obs.SkillBundlePromptMode) {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s skill_bundle_prompt_mode must be disabled|append", caseName, lane),
		}
	}
	if !isSkillBundleWhitelistMode(obs.SkillBundleWhitelistMode) {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s skill_bundle_whitelist_mode must be disabled|merge", caseName, lane),
		}
	}
	if !isSkillBundleConflictPolicy(obs.SkillBundleConflictPolicy) {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s skill_bundle_conflict_policy must be fail_fast|first_win", caseName, lane),
		}
	}
	if obs.SkillBundlePromptTotal < 0 || obs.SkillBundleWhitelistTotal < 0 || obs.SkillBundleWhitelistRejectedTotal < 0 {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s skill bundle counters must be >= 0", caseName, lane),
		}
	}
	if (obs.SkillPreprocessStatus == "failed" || obs.SkillPreprocessStatus == "degraded") && !isSkillPreprocessReasonCode(obs.SkillPreprocessReasonCode) {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s skill_preprocess_reason_code must be canonical when status is failed|degraded", caseName, lane),
		}
	}
	if (obs.SkillPreprocessStatus == "success" || obs.SkillPreprocessStatus == "skipped") && strings.TrimSpace(obs.SkillPreprocessReasonCode) != "" {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s skill_preprocess_reason_code must be empty when status is success|skipped", caseName, lane),
		}
	}
	return nil
}

func assertSkillMappingArbitrationEquivalent(caseName, lane string, expected, actual ArbitrationObservation) error {
	if expected.SkillPreprocessEnabled != actual.SkillPreprocessEnabled ||
		expected.SkillPreprocessPhase != actual.SkillPreprocessPhase ||
		expected.SkillPreprocessFailMode != actual.SkillPreprocessFailMode ||
		expected.SkillPreprocessStatus != actual.SkillPreprocessStatus ||
		expected.SkillPreprocessReasonCode != actual.SkillPreprocessReasonCode ||
		expected.SkillBundlePromptMode != actual.SkillBundlePromptMode ||
		expected.SkillBundleWhitelistMode != actual.SkillBundleWhitelistMode ||
		expected.SkillBundleConflictPolicy != actual.SkillBundleConflictPolicy ||
		expected.SkillBundlePromptTotal != actual.SkillBundlePromptTotal ||
		expected.SkillBundleWhitelistTotal != actual.SkillBundleWhitelistTotal ||
		expected.SkillBundleWhitelistRejectedTotal != actual.SkillBundleWhitelistRejectedTotal {
		return &ValidationError{
			Code:    ReasonCodeSkillBundleMappingDrift,
			Message: fmt.Sprintf("case %q %s skill preprocess/mapping drift expected=%#v actual=%#v", caseName, lane, expected, actual),
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
	if version == ArbitrationFixtureVersionHooksMiddlewareV1 {
		return left.HooksEnabled == right.HooksEnabled &&
			left.HooksFailMode == right.HooksFailMode &&
			equalStringSlice(left.HooksPhases, right.HooksPhases) &&
			left.ToolMiddlewareEnabled == right.ToolMiddlewareEnabled &&
			left.ToolMiddlewareFailMode == right.ToolMiddlewareFailMode
	}
	if version == ArbitrationFixtureVersionSkillDiscoveryV1 {
		return left.SkillDiscoveryMode == right.SkillDiscoveryMode &&
			equalStringSlice(left.SkillDiscoveryRoots, right.SkillDiscoveryRoots) &&
			left.SkillPreprocessSpecCount == right.SkillPreprocessSpecCount
	}
	if version == ArbitrationFixtureVersionSkillMappingV1 {
		return left.SkillPreprocessEnabled == right.SkillPreprocessEnabled &&
			left.SkillPreprocessPhase == right.SkillPreprocessPhase &&
			left.SkillPreprocessFailMode == right.SkillPreprocessFailMode &&
			left.SkillPreprocessStatus == right.SkillPreprocessStatus &&
			left.SkillPreprocessReasonCode == right.SkillPreprocessReasonCode &&
			left.SkillBundlePromptMode == right.SkillBundlePromptMode &&
			left.SkillBundleWhitelistMode == right.SkillBundleWhitelistMode &&
			left.SkillBundleConflictPolicy == right.SkillBundleConflictPolicy &&
			left.SkillBundlePromptTotal == right.SkillBundlePromptTotal &&
			left.SkillBundleWhitelistTotal == right.SkillBundleWhitelistTotal &&
			left.SkillBundleWhitelistRejectedTotal == right.SkillBundleWhitelistRejectedTotal
	}
	if version == ArbitrationFixtureVersionOTelSemconvV1 {
		return left.TraceExportStatus == right.TraceExportStatus &&
			left.TraceSchemaVersion == right.TraceSchemaVersion &&
			left.TraceTopologyClass == right.TraceTopologyClass &&
			equalStringSlice(left.TraceCanonicalAttrKeys, right.TraceCanonicalAttrKeys)
	}
	if version == ArbitrationFixtureVersionAgentEvalV1 {
		return left.EvalSuiteID == right.EvalSuiteID &&
			left.EvalExecutionMode == right.EvalExecutionMode &&
			equalAnyMap(left.EvalSummary, right.EvalSummary)
	}
	if version == ArbitrationFixtureVersionAgentEvalDistV1 {
		return left.EvalSuiteID == right.EvalSuiteID &&
			left.EvalExecutionMode == right.EvalExecutionMode &&
			left.EvalJobID == right.EvalJobID &&
			left.EvalShardTotal == right.EvalShardTotal &&
			left.EvalResumeCount == right.EvalResumeCount &&
			equalAnyMap(left.EvalSummary, right.EvalSummary)
	}
	if version == ArbitrationFixtureVersionStateSnapshotV1 {
		return left.StateSnapshotVersion == right.StateSnapshotVersion &&
			left.StateRestoreAction == right.StateRestoreAction &&
			left.StateRestoreConflictCode == right.StateRestoreConflictCode &&
			left.StateRestoreSource == right.StateRestoreSource
	}
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
	if version == ArbitrationFixtureVersionReactPlanV1 {
		return left.ModelProvider == right.ModelProvider &&
			left.ReactEnabled == right.ReactEnabled &&
			left.ReactIterationTotal == right.ReactIterationTotal &&
			left.ReactToolCallTotal == right.ReactToolCallTotal &&
			left.ReactToolCallBudgetHitTotal == right.ReactToolCallBudgetHitTotal &&
			left.ReactIterationBudgetHitTotal == right.ReactIterationBudgetHitTotal &&
			left.ReactTerminationReason == right.ReactTerminationReason &&
			left.ReactStreamDispatchEnabled == right.ReactStreamDispatchEnabled &&
			left.ReactPlanID == right.ReactPlanID &&
			left.ReactPlanVersion == right.ReactPlanVersion &&
			left.ReactPlanChangeTotal == right.ReactPlanChangeTotal &&
			left.ReactPlanLastAction == right.ReactPlanLastAction &&
			left.ReactPlanChangeReason == right.ReactPlanChangeReason &&
			left.ReactPlanRecoverCount == right.ReactPlanRecoverCount &&
			left.ReactPlanHookStatus == right.ReactPlanHookStatus
	}
	if version == ArbitrationFixtureVersionRealtimeProtocolV1 {
		return left.ModelProvider == right.ModelProvider &&
			left.ReactEnabled == right.ReactEnabled &&
			left.ReactIterationTotal == right.ReactIterationTotal &&
			left.ReactToolCallTotal == right.ReactToolCallTotal &&
			left.ReactToolCallBudgetHitTotal == right.ReactToolCallBudgetHitTotal &&
			left.ReactIterationBudgetHitTotal == right.ReactIterationBudgetHitTotal &&
			left.ReactTerminationReason == right.ReactTerminationReason &&
			left.ReactStreamDispatchEnabled == right.ReactStreamDispatchEnabled &&
			left.RealtimeProtocolVersion == right.RealtimeProtocolVersion &&
			left.RealtimeEventSeqMax == right.RealtimeEventSeqMax &&
			left.RealtimeInterruptTotal == right.RealtimeInterruptTotal &&
			left.RealtimeResumeTotal == right.RealtimeResumeTotal &&
			left.RealtimeResumeSource == right.RealtimeResumeSource &&
			left.RealtimeIdempotencyDedupTotal == right.RealtimeIdempotencyDedupTotal &&
			left.RealtimeLastErrorCode == right.RealtimeLastErrorCode
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

func anyMapString(m map[string]any, key string) string {
	if len(m) == 0 {
		return ""
	}
	raw, ok := m[strings.ToLower(strings.TrimSpace(key))]
	if !ok || raw == nil {
		return ""
	}
	text, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func equalAnyMap(left, right map[string]any) bool {
	if len(left) == 0 && len(right) == 0 {
		return true
	}
	lhs, err := json.Marshal(canonicalizeAnyMap(left))
	if err != nil {
		return false
	}
	rhs, err := json.Marshal(canonicalizeAnyMap(right))
	if err != nil {
		return false
	}
	return bytes.Equal(lhs, rhs)
}

func canonicalizeAnyMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		normalizedKey := strings.ToLower(strings.TrimSpace(key))
		if normalizedKey == "" {
			continue
		}
		out[normalizedKey] = canonicalizeAnyValue(value)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func canonicalizeAnyValue(in any) any {
	switch typed := in.(type) {
	case map[string]any:
		return canonicalizeAnyMap(typed)
	case []any:
		out := make([]any, 0, len(typed))
		for i := range typed {
			out = append(out, canonicalizeAnyValue(typed[i]))
		}
		return out
	case []string:
		out := make([]any, 0, len(typed))
		for i := range typed {
			out = append(out, strings.TrimSpace(typed[i]))
		}
		return out
	case string:
		return strings.TrimSpace(typed)
	default:
		return in
	}
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

func isCanonicalStateRestoreAction(action string) bool {
	switch strings.TrimSpace(action) {
	case "strict_exact_restore", "compatible_exact_restore", "compatible_bounded_restore", "idempotent_noop":
		return true
	default:
		return false
	}
}

func isStrictRestoreAction(action string) bool {
	return strings.TrimSpace(action) == "strict_exact_restore"
}

func isCanonicalStateRestoreConflictCode(code string) bool {
	switch strings.TrimSpace(code) {
	case "state_snapshot_invalid_payload",
		"state_snapshot_restore_mode_invalid",
		"state_snapshot_strict_incompatible",
		"state_snapshot_compat_window_exceeded",
		"state_snapshot_operation_conflict",
		"state_snapshot_digest_mismatch",
		"snapshot_recovery_boundary_violation",
		"snapshot_scheduler_conflict",
		"snapshot_mailbox_conflict",
		"snapshot_memory_contract_mismatch",
		"snapshot_memory_lifecycle_mismatch",
		"snapshot_memory_search_policy_mismatch",
		"snapshot_memory_retrieval_quality_drift":
		return true
	default:
		return false
	}
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

func isCanonicalReactPlanAction(action string) bool {
	switch strings.TrimSpace(action) {
	case "create", "revise", "complete", "recover":
		return true
	default:
		return false
	}
}

func isCanonicalReactPlanHookStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case "ok", "degraded", "failed", "disabled":
		return true
	default:
		return false
	}
}

func isCanonicalRealtimeResumeSource(source string) bool {
	switch strings.TrimSpace(source) {
	case "", "cursor":
		return true
	default:
		return false
	}
}

func isCanonicalRealtimeReasonCode(code string) bool {
	switch strings.TrimSpace(code) {
	case "",
		"realtime.event_order_drift",
		"realtime.sequence_gap",
		"realtime.resume.invalid_cursor",
		"realtime.interrupt.freeze",
		"realtime.unsupported_event_type",
		"realtime.schema_invalid",
		"realtime.buffer_overflow":
		return true
	default:
		return false
	}
}

func isRealtimeOrderReasonCode(code string) bool {
	switch strings.TrimSpace(code) {
	case "realtime.event_order_drift", "realtime.sequence_gap":
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

func canonicalHookPhases() []string {
	return []string{
		runtimeconfig.RuntimeHookPhaseBeforeReasoning,
		runtimeconfig.RuntimeHookPhaseAfterReasoning,
		runtimeconfig.RuntimeHookPhaseBeforeActing,
		runtimeconfig.RuntimeHookPhaseAfterActing,
		runtimeconfig.RuntimeHookPhaseBeforeReply,
		runtimeconfig.RuntimeHookPhaseAfterReply,
	}
}

func isHookPhase(phase string) bool {
	needle := strings.ToLower(strings.TrimSpace(phase))
	for _, item := range canonicalHookPhases() {
		if needle == item {
			return true
		}
	}
	return false
}

func isHooksFailMode(mode string) bool {
	switch strings.TrimSpace(mode) {
	case runtimeconfig.RuntimeHooksFailModeFailFast, runtimeconfig.RuntimeHooksFailModeDegrade:
		return true
	default:
		return false
	}
}

func isToolMiddlewareFailMode(mode string) bool {
	switch strings.TrimSpace(mode) {
	case runtimeconfig.RuntimeToolMiddlewareFailModeFailFast, runtimeconfig.RuntimeToolMiddlewareFailModeDegrade:
		return true
	default:
		return false
	}
}

func isSkillDiscoveryMode(mode string) bool {
	switch strings.TrimSpace(mode) {
	case runtimeconfig.RuntimeSkillDiscoveryModeAgentsMD, runtimeconfig.RuntimeSkillDiscoveryModeFolder, runtimeconfig.RuntimeSkillDiscoveryModeHybrid:
		return true
	default:
		return false
	}
}

func isSkillPreprocessPhase(phase string) bool {
	return strings.TrimSpace(phase) == runtimeconfig.RuntimeSkillPreprocessPhaseBeforeRunStream
}

func isSkillPreprocessFailMode(mode string) bool {
	switch strings.TrimSpace(mode) {
	case runtimeconfig.RuntimeSkillPreprocessFailModeFailFast, runtimeconfig.RuntimeSkillPreprocessFailModeDegrade:
		return true
	default:
		return false
	}
}

func isSkillPreprocessStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case "success", "failed", "degraded", "skipped":
		return true
	default:
		return false
	}
}

func isSkillBundlePromptMode(mode string) bool {
	switch strings.TrimSpace(mode) {
	case runtimeconfig.RuntimeSkillBundleMappingPromptModeDisabled, runtimeconfig.RuntimeSkillBundleMappingPromptModeAppend:
		return true
	default:
		return false
	}
}

func isSkillBundleWhitelistMode(mode string) bool {
	switch strings.TrimSpace(mode) {
	case runtimeconfig.RuntimeSkillBundleMappingWhitelistModeDisabled, runtimeconfig.RuntimeSkillBundleMappingWhitelistModeMerge:
		return true
	default:
		return false
	}
}

func isSkillBundleConflictPolicy(policy string) bool {
	switch strings.TrimSpace(policy) {
	case runtimeconfig.RuntimeSkillBundleMappingConflictPolicyFailFast, runtimeconfig.RuntimeSkillBundleMappingConflictPolicyFirstWin:
		return true
	default:
		return false
	}
}

func isSkillPreprocessReasonCode(code string) bool {
	switch strings.TrimSpace(code) {
	case "skill_preprocess_failed",
		"skill_bundle_prompt_conflict",
		"skill_bundle_whitelist_conflict",
		"skill_bundle_whitelist_exceeds_sandbox",
		"skill_bundle_whitelist_exceeds_adapter_allowlist",
		"skill_bundle_whitelist_invalid_tool",
		"skill_bundle_whitelist_violation":
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

func validateOTelSemconvArbitrationObservation(caseName, lane string, obs ArbitrationObservation) error {
	if strings.TrimSpace(obs.TraceSchemaVersion) != ArbitrationFixtureVersionOTelSemconvV1 {
		return &ValidationError{
			Code:    ReasonCodeOTelAttrMappingDrift,
			Message: fmt.Sprintf("case %q %s trace_schema_version must be %q", caseName, lane, ArbitrationFixtureVersionOTelSemconvV1),
		}
	}
	switch strings.TrimSpace(obs.TraceExportStatus) {
	case "disabled", "success", "degraded", "failed":
	default:
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s trace_export_status must be disabled|success|degraded|failed", caseName, lane),
		}
	}
	if strings.TrimSpace(obs.TraceTopologyClass) == "" {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s trace_topology_class is required", caseName, lane),
		}
	}
	if len(obs.TraceCanonicalAttrKeys) == 0 {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s trace_canonical_attr_keys must not be empty", caseName, lane),
		}
	}
	requiredKeys := map[string]struct{}{
		"trace.schema_version": {},
		"trace.domain":         {},
	}
	for i := range obs.TraceCanonicalAttrKeys {
		item := strings.ToLower(strings.TrimSpace(obs.TraceCanonicalAttrKeys[i]))
		if item == "" {
			return &ValidationError{
				Code:    ReasonCodeSchemaMismatch,
				Message: fmt.Sprintf("case %q %s trace_canonical_attr_keys contains empty key", caseName, lane),
			}
		}
		delete(requiredKeys, item)
	}
	if len(requiredKeys) > 0 {
		return &ValidationError{
			Code:    ReasonCodeOTelAttrMappingDrift,
			Message: fmt.Sprintf("case %q %s trace_canonical_attr_keys missing required canonical keys", caseName, lane),
		}
	}
	return nil
}

func assertOTelSemconvArbitrationEquivalent(caseName, lane string, expected, actual ArbitrationObservation) error {
	if expected.TraceTopologyClass != actual.TraceTopologyClass {
		return &ValidationError{
			Code:    ReasonCodeSpanTopologyDrift,
			Message: fmt.Sprintf("case %q %s span topology drift expected=%q actual=%q", caseName, lane, expected.TraceTopologyClass, actual.TraceTopologyClass),
		}
	}
	if expected.TraceSchemaVersion != actual.TraceSchemaVersion ||
		expected.TraceExportStatus != actual.TraceExportStatus ||
		!equalStringSlice(expected.TraceCanonicalAttrKeys, actual.TraceCanonicalAttrKeys) {
		return &ValidationError{
			Code:    ReasonCodeOTelAttrMappingDrift,
			Message: fmt.Sprintf("case %q %s otel attr mapping drift expected=%#v actual=%#v", caseName, lane, expected, actual),
		}
	}
	return &ValidationError{
		Code:    ReasonCodeSemanticDrift,
		Message: fmt.Sprintf("case %q %s otel semantic drift expected=%#v actual=%#v", caseName, lane, expected, actual),
	}
}

func validateAgentEvalArbitrationObservation(caseName, lane string, obs ArbitrationObservation) error {
	if strings.TrimSpace(obs.EvalSuiteID) == "" {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s eval_suite_id is required", caseName, lane),
		}
	}
	if strings.TrimSpace(obs.EvalExecutionMode) != runtimeconfig.RuntimeEvalExecutionModeLocal {
		return &ValidationError{
			Code:    ReasonCodeEvalAggregationDrift,
			Message: fmt.Sprintf("case %q %s eval_execution_mode must be local for agent_eval.v1 fixtures", caseName, lane),
		}
	}
	if len(obs.EvalSummary) == 0 {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s eval_summary is required", caseName, lane),
		}
	}
	version := strings.ToLower(strings.TrimSpace(anyMapString(obs.EvalSummary, "version")))
	if version != "" && version != ArbitrationFixtureVersionAgentEvalV1 {
		return &ValidationError{
			Code:    ReasonCodeEvalMetricDrift,
			Message: fmt.Sprintf("case %q %s eval_summary.version must be %q", caseName, lane, ArbitrationFixtureVersionAgentEvalV1),
		}
	}
	requiredSummary := []string{"task_success", "tool_correctness", "deny_intercept", "cost_latency", "all_constraints_pass"}
	for i := range requiredSummary {
		if _, ok := obs.EvalSummary[requiredSummary[i]]; !ok {
			return &ValidationError{
				Code:    ReasonCodeSchemaMismatch,
				Message: fmt.Sprintf("case %q %s eval_summary.%s is required", caseName, lane, requiredSummary[i]),
			}
		}
	}
	return nil
}

func assertAgentEvalArbitrationEquivalent(caseName, lane string, expected, actual ArbitrationObservation) error {
	if expected.EvalExecutionMode != actual.EvalExecutionMode {
		return &ValidationError{
			Code:    ReasonCodeEvalAggregationDrift,
			Message: fmt.Sprintf("case %q %s eval execution mode drift expected=%q actual=%q", caseName, lane, expected.EvalExecutionMode, actual.EvalExecutionMode),
		}
	}
	if expected.EvalSuiteID != actual.EvalSuiteID {
		return &ValidationError{
			Code:    ReasonCodeEvalMetricDrift,
			Message: fmt.Sprintf("case %q %s eval suite drift expected=%q actual=%q", caseName, lane, expected.EvalSuiteID, actual.EvalSuiteID),
		}
	}
	if !equalAnyMap(expected.EvalSummary, actual.EvalSummary) {
		return &ValidationError{
			Code:    ReasonCodeEvalMetricDrift,
			Message: fmt.Sprintf("case %q %s eval metric drift expected=%#v actual=%#v", caseName, lane, expected.EvalSummary, actual.EvalSummary),
		}
	}
	return &ValidationError{
		Code:    ReasonCodeSemanticDrift,
		Message: fmt.Sprintf("case %q %s eval semantic drift expected=%#v actual=%#v", caseName, lane, expected, actual),
	}
}

func validateAgentEvalDistributedArbitrationObservation(caseName, lane string, obs ArbitrationObservation) error {
	if strings.TrimSpace(obs.EvalSuiteID) == "" {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s eval_suite_id is required", caseName, lane),
		}
	}
	if strings.TrimSpace(obs.EvalExecutionMode) != runtimeconfig.RuntimeEvalExecutionModeDistributed {
		return &ValidationError{
			Code:    ReasonCodeEvalAggregationDrift,
			Message: fmt.Sprintf("case %q %s eval_execution_mode must be distributed for agent_eval_distributed.v1 fixtures", caseName, lane),
		}
	}
	if strings.TrimSpace(obs.EvalJobID) == "" {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s eval_job_id is required", caseName, lane),
		}
	}
	if obs.EvalShardTotal <= 0 {
		return &ValidationError{
			Code:    ReasonCodeEvalShardResumeDrift,
			Message: fmt.Sprintf("case %q %s eval_shard_total must be > 0", caseName, lane),
		}
	}
	if obs.EvalResumeCount < 0 {
		return &ValidationError{
			Code:    ReasonCodeEvalShardResumeDrift,
			Message: fmt.Sprintf("case %q %s eval_resume_count must be >= 0", caseName, lane),
		}
	}
	if len(obs.EvalSummary) == 0 {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s eval_summary is required", caseName, lane),
		}
	}
	version := strings.ToLower(strings.TrimSpace(anyMapString(obs.EvalSummary, "version")))
	if version != "" && version != ArbitrationFixtureVersionAgentEvalDistV1 {
		return &ValidationError{
			Code:    ReasonCodeEvalMetricDrift,
			Message: fmt.Sprintf("case %q %s eval_summary.version must be %q", caseName, lane, ArbitrationFixtureVersionAgentEvalDistV1),
		}
	}
	aggregation := strings.ToLower(strings.TrimSpace(anyMapString(obs.EvalSummary, "aggregation")))
	if aggregation == "" {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s eval_summary.aggregation is required", caseName, lane),
		}
	}
	switch aggregation {
	case runtimeconfig.RuntimeEvalExecutionAggregationWeightedMean, runtimeconfig.RuntimeEvalExecutionAggregationWorstCase:
	default:
		return &ValidationError{
			Code:    ReasonCodeEvalAggregationDrift,
			Message: fmt.Sprintf("case %q %s eval_summary.aggregation must be weighted_mean|worst_case", caseName, lane),
		}
	}
	requiredSummary := []string{"task_success_rate", "tool_correctness_rate", "deny_intercept_rate", "cost_estimate", "latency_estimate"}
	for i := range requiredSummary {
		if _, ok := obs.EvalSummary[requiredSummary[i]]; !ok {
			return &ValidationError{
				Code:    ReasonCodeSchemaMismatch,
				Message: fmt.Sprintf("case %q %s eval_summary.%s is required", caseName, lane, requiredSummary[i]),
			}
		}
	}
	return nil
}

func assertAgentEvalDistributedArbitrationEquivalent(caseName, lane string, expected, actual ArbitrationObservation) error {
	if expected.EvalExecutionMode != actual.EvalExecutionMode {
		return &ValidationError{
			Code:    ReasonCodeEvalAggregationDrift,
			Message: fmt.Sprintf("case %q %s eval execution mode drift expected=%q actual=%q", caseName, lane, expected.EvalExecutionMode, actual.EvalExecutionMode),
		}
	}
	if expected.EvalShardTotal != actual.EvalShardTotal ||
		expected.EvalResumeCount != actual.EvalResumeCount ||
		expected.EvalJobID != actual.EvalJobID {
		return &ValidationError{
			Code:    ReasonCodeEvalShardResumeDrift,
			Message: fmt.Sprintf("case %q %s eval shard/resume drift expected=%#v actual=%#v", caseName, lane, expected, actual),
		}
	}
	if !strings.EqualFold(strings.TrimSpace(anyMapString(expected.EvalSummary, "aggregation")), strings.TrimSpace(anyMapString(actual.EvalSummary, "aggregation"))) {
		return &ValidationError{
			Code:    ReasonCodeEvalAggregationDrift,
			Message: fmt.Sprintf("case %q %s eval aggregation drift expected=%q actual=%q", caseName, lane, anyMapString(expected.EvalSummary, "aggregation"), anyMapString(actual.EvalSummary, "aggregation")),
		}
	}
	if expected.EvalSuiteID != actual.EvalSuiteID || !equalAnyMap(expected.EvalSummary, actual.EvalSummary) {
		return &ValidationError{
			Code:    ReasonCodeEvalMetricDrift,
			Message: fmt.Sprintf("case %q %s eval metric drift expected=%#v actual=%#v", caseName, lane, expected, actual),
		}
	}
	return &ValidationError{
		Code:    ReasonCodeSemanticDrift,
		Message: fmt.Sprintf("case %q %s eval distributed semantic drift expected=%#v actual=%#v", caseName, lane, expected, actual),
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
