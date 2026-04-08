package diagnosticsreplay

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestReplayContractPrimaryReasonArbitrationFixtureSuccessAndDeterministicOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "a49",
			input:    "a49_arbitration_success_input.json",
			expected: "a49_arbitration_success_expected.json",
		},
		{
			name:     "a50",
			input:    "a50_arbitration_success_input.json",
			expected: "a50_arbitration_success_expected.json",
		},
		{
			name:     "a51",
			input:    "a51_sandbox_success_input.json",
			expected: "a51_sandbox_success_expected.json",
		},
		{
			name:     "a52",
			input:    "a52_sandbox_rollout_success_input.json",
			expected: "a52_sandbox_rollout_success_expected.json",
		},
		{
			name:     "a54-memory",
			input:    "a54_memory_success_input.json",
			expected: "a54_memory_success_expected.json",
		},
		{
			name:     "a55-observability",
			input:    "a55_observability_success_input.json",
			expected: "a55_observability_success_expected.json",
		},
		{
			name:     "a56-react",
			input:    "a56_react_success_input.json",
			expected: "a56_react_success_expected.json",
		},
		{
			name:     "react-plan-notebook",
			input:    "react_plan_notebook_success_input.json",
			expected: "react_plan_notebook_success_expected.json",
		},
		{
			name:     "realtime-protocol",
			input:    "realtime_event_protocol_success_input.json",
			expected: "realtime_event_protocol_success_expected.json",
		},
		{
			name:     "context-reference-first",
			input:    "context_reference_first_success_input.json",
			expected: "context_reference_first_success_expected.json",
		},
		{
			name:     "context-isolate-handoff",
			input:    "context_isolate_handoff_success_input.json",
			expected: "context_isolate_handoff_success_expected.json",
		},
		{
			name:     "context-edit-gate",
			input:    "context_edit_gate_success_input.json",
			expected: "context_edit_gate_success_expected.json",
		},
		{
			name:     "context-swapback",
			input:    "context_relevance_swapback_success_input.json",
			expected: "context_relevance_swapback_success_expected.json",
		},
		{
			name:     "context-lifecycle-tiering",
			input:    "context_lifecycle_tiering_success_input.json",
			expected: "context_lifecycle_tiering_success_expected.json",
		},
		{
			name:     "a57-sandbox-egress-allowlist",
			input:    "a57_sandbox_egress_success_input.json",
			expected: "a57_sandbox_egress_success_expected.json",
		},
		{
			name:     "a58-policy-stack",
			input:    "a58_policy_stack_success_input.json",
			expected: "a58_policy_stack_success_expected.json",
		},
		{
			name:     "a59-memory-scope",
			input:    "a59_memory_scope_success_input.json",
			expected: "a59_memory_scope_success_expected.json",
		},
		{
			name:     "a59-memory-search",
			input:    "a59_memory_search_success_input.json",
			expected: "a59_memory_search_success_expected.json",
		},
		{
			name:     "a59-memory-lifecycle",
			input:    "a59_memory_lifecycle_success_input.json",
			expected: "a59_memory_lifecycle_success_expected.json",
		},
		{
			name:     "a60-budget-admission",
			input:    "a60_budget_admission_success_input.json",
			expected: "a60_budget_admission_success_expected.json",
		},
		{
			name:     "a61-otel-semconv",
			input:    "a61_otel_semconv_success_input.json",
			expected: "a61_otel_semconv_success_expected.json",
		},
		{
			name:     "a61-agent-eval",
			input:    "a61_agent_eval_success_input.json",
			expected: "a61_agent_eval_success_expected.json",
		},
		{
			name:     "a61-agent-eval-distributed",
			input:    "a61_agent_eval_distributed_success_input.json",
			expected: "a61_agent_eval_distributed_success_expected.json",
		},
		{
			name:     "a61-inferential-advisory",
			input:    "a61_inferential_advisory_success_input.json",
			expected: "a61_inferential_advisory_success_expected.json",
		},
		{
			name:     "a61-inferential-advisory-distributed",
			input:    "a61_inferential_advisory_distributed_success_input.json",
			expected: "a61_inferential_advisory_distributed_success_expected.json",
		},
		{
			name:     "a65-hooks-middleware",
			input:    "a65_hooks_middleware_success_input.json",
			expected: "a65_hooks_middleware_success_expected.json",
		},
		{
			name:     "a65-skill-discovery",
			input:    "a65_skill_discovery_sources_success_input.json",
			expected: "a65_skill_discovery_sources_success_expected.json",
		},
		{
			name:     "a65-skill-preprocess-mapping",
			input:    "a65_skill_preprocess_and_mapping_success_input.json",
			expected: "a65_skill_preprocess_and_mapping_success_expected.json",
		},
		{
			name:     "a66-state-session-snapshot",
			input:    "a66_state_session_snapshot_success_input.json",
			expected: "a66_state_session_snapshot_success_expected.json",
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			input := mustReadFixture(t, tc.input)
			expectedRaw := mustReadFixture(t, tc.expected)
			got, err := EvaluateArbitrationFixtureJSON(input)
			if err != nil {
				t.Fatalf("EvaluateArbitrationFixtureJSON error: %v", err)
			}
			var want ArbitrationReplayOutput
			if err := json.Unmarshal(expectedRaw, &want); err != nil {
				t.Fatalf("unmarshal expected fixture: %v", err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("arbitration replay output mismatch\ngot=%#v\nwant=%#v", got, want)
			}
			replay, err := EvaluateArbitrationFixtureJSON(input)
			if err != nil {
				t.Fatalf("EvaluateArbitrationFixtureJSON replay error: %v", err)
			}
			if !reflect.DeepEqual(got, replay) {
				t.Fatalf("arbitration replay output should be deterministic: first=%#v replay=%#v", got, replay)
			}
		})
	}
}

func TestReplayContractPrimaryReasonArbitrationFixtureDriftClassification(t *testing.T) {
	tests := []struct {
		name       string
		fixture    string
		wantCode   string
		wantInText string
	}{
		{
			name:       "precedence",
			fixture:    "a49_arbitration_precedence_drift_input.json",
			wantCode:   ReasonCodePrecedenceDrift,
			wantInText: "precedence drift",
		},
		{
			name:       "tie-break",
			fixture:    "a49_arbitration_tie_break_drift_input.json",
			wantCode:   ReasonCodeTieBreakDrift,
			wantInText: "tie-break drift",
		},
		{
			name:       "taxonomy",
			fixture:    "a49_arbitration_taxonomy_drift_input.json",
			wantCode:   ReasonCodeTaxonomyDrift,
			wantInText: "non-canonical primary code",
		},
		{
			name:       "secondary-order",
			fixture:    "a49_arbitration_secondary_order_drift_input.json",
			wantCode:   ReasonCodeSecondaryOrderDrift,
			wantInText: "secondary order drift",
		},
		{
			name:       "secondary-count",
			fixture:    "a49_arbitration_secondary_count_drift_input.json",
			wantCode:   ReasonCodeSecondaryCountDrift,
			wantInText: "secondary count drift",
		},
		{
			name:       "hint-taxonomy",
			fixture:    "a49_arbitration_hint_taxonomy_drift_input.json",
			wantCode:   ReasonCodeHintTaxonomyDrift,
			wantInText: "hint taxonomy drift",
		},
		{
			name:       "rule-version",
			fixture:    "a49_arbitration_rule_version_drift_input.json",
			wantCode:   ReasonCodeRuleVersionDrift,
			wantInText: "rule version drift",
		},
		{
			name:       "a50-version-mismatch",
			fixture:    "a50_arbitration_version_mismatch_drift_input.json",
			wantCode:   ReasonCodeVersionMismatch,
			wantInText: "version mismatch",
		},
		{
			name:       "a50-unsupported-version",
			fixture:    "a50_arbitration_unsupported_version_drift_input.json",
			wantCode:   ReasonCodeUnsupportedVersion,
			wantInText: "unsupported version",
		},
		{
			name:       "a50-cross-version-semantic-drift",
			fixture:    "a50_arbitration_cross_version_semantic_drift_input.json",
			wantCode:   ReasonCodeCrossVersionSemanticDrift,
			wantInText: "cross-version semantic drift",
		},
		{
			name:       "a51-sandbox-policy-drift",
			fixture:    "a51_sandbox_policy_drift_input.json",
			wantCode:   ReasonCodeSandboxPolicyDrift,
			wantInText: "sandbox policy drift",
		},
		{
			name:       "a51-sandbox-fallback-drift",
			fixture:    "a51_sandbox_fallback_drift_input.json",
			wantCode:   ReasonCodeSandboxFallbackDrift,
			wantInText: "sandbox fallback drift",
		},
		{
			name:       "a51-sandbox-timeout-drift",
			fixture:    "a51_sandbox_timeout_drift_input.json",
			wantCode:   ReasonCodeSandboxTimeoutDrift,
			wantInText: "sandbox timeout drift",
		},
		{
			name:       "a51-sandbox-capability-drift",
			fixture:    "a51_sandbox_capability_drift_input.json",
			wantCode:   ReasonCodeSandboxCapabilityDrift,
			wantInText: "sandbox capability drift",
		},
		{
			name:       "a51-sandbox-resource-policy-drift",
			fixture:    "a51_sandbox_resource_policy_drift_input.json",
			wantCode:   ReasonCodeSandboxResourcePolicyDrift,
			wantInText: "sandbox resource policy drift",
		},
		{
			name:       "a51-sandbox-session-lifecycle-drift",
			fixture:    "a51_sandbox_session_lifecycle_drift_input.json",
			wantCode:   ReasonCodeSandboxSessionLifecycleDrift,
			wantInText: "sandbox session lifecycle drift",
		},
		{
			name:       "a52-sandbox-rollout-phase-drift",
			fixture:    "a52_sandbox_rollout_phase_drift_input.json",
			wantCode:   ReasonCodeSandboxRolloutPhaseDrift,
			wantInText: "sandbox rollout phase drift",
		},
		{
			name:       "a52-sandbox-health-budget-drift",
			fixture:    "a52_sandbox_health_budget_drift_input.json",
			wantCode:   ReasonCodeSandboxHealthBudgetDrift,
			wantInText: "sandbox health budget drift",
		},
		{
			name:       "a52-sandbox-capacity-action-drift",
			fixture:    "a52_sandbox_capacity_action_drift_input.json",
			wantCode:   ReasonCodeSandboxCapacityActionDrift,
			wantInText: "sandbox capacity action drift",
		},
		{
			name:       "a52-sandbox-freeze-state-drift",
			fixture:    "a52_sandbox_freeze_state_drift_input.json",
			wantCode:   ReasonCodeSandboxFreezeStateDrift,
			wantInText: "sandbox freeze state drift",
		},
		{
			name:       "a54-memory-mode-drift",
			fixture:    "a54_memory_mode_drift_input.json",
			wantCode:   ReasonCodeMemoryModeDrift,
			wantInText: "memory mode drift",
		},
		{
			name:       "a54-memory-profile-drift",
			fixture:    "a54_memory_profile_drift_input.json",
			wantCode:   ReasonCodeMemoryProfileDrift,
			wantInText: "memory profile drift",
		},
		{
			name:       "a54-memory-contract-version-drift",
			fixture:    "a54_memory_contract_version_drift_input.json",
			wantCode:   ReasonCodeMemoryContractVersionDrift,
			wantInText: "memory_contract_version",
		},
		{
			name:       "a54-memory-fallback-drift",
			fixture:    "a54_memory_fallback_drift_input.json",
			wantCode:   ReasonCodeMemoryFallbackDrift,
			wantInText: "memory fallback drift",
		},
		{
			name:       "a54-memory-error-taxonomy-drift",
			fixture:    "a54_memory_error_taxonomy_drift_input.json",
			wantCode:   ReasonCodeMemoryErrorTaxonomyDrift,
			wantInText: "memory error taxonomy drift",
		},
		{
			name:       "a54-memory-operation-aggregate-drift",
			fixture:    "a54_memory_operation_aggregate_drift_input.json",
			wantCode:   ReasonCodeMemoryOperationAggregateDrift,
			wantInText: "memory operation aggregate drift",
		},
		{
			name:       "a54-memory-unsupported-version",
			fixture:    "a54_memory_unsupported_version_input.json",
			wantCode:   ReasonCodeSchemaMismatch,
			wantInText: "unsupported fixture version",
		},
		{
			name:       "a55-observability-export-profile-drift",
			fixture:    "a55_observability_export_profile_drift_input.json",
			wantCode:   ReasonCodeObsExportProfileDrift,
			wantInText: "export profile drift",
		},
		{
			name:       "a55-observability-export-status-drift",
			fixture:    "a55_observability_export_status_drift_input.json",
			wantCode:   ReasonCodeObsExportStatusDrift,
			wantInText: "export status drift",
		},
		{
			name:       "a55-observability-export-reason-drift",
			fixture:    "a55_observability_export_reason_drift_input.json",
			wantCode:   ReasonCodeObsExportReasonDrift,
			wantInText: "export reason drift",
		},
		{
			name:       "a55-observability-bundle-schema-drift",
			fixture:    "a55_observability_bundle_schema_drift_input.json",
			wantCode:   ReasonCodeBundleSchemaDrift,
			wantInText: "bundle schema drift",
		},
		{
			name:       "a55-observability-bundle-redaction-drift",
			fixture:    "a55_observability_bundle_redaction_drift_input.json",
			wantCode:   ReasonCodeBundleRedactionDrift,
			wantInText: "bundle redaction drift",
		},
		{
			name:       "a55-observability-bundle-fingerprint-drift",
			fixture:    "a55_observability_bundle_fingerprint_drift_input.json",
			wantCode:   ReasonCodeBundleFingerprintDrift,
			wantInText: "bundle fingerprint drift",
		},
		{
			name:       "a56-react-loop-step-drift",
			fixture:    "a56_react_loop_step_drift_input.json",
			wantCode:   ReasonCodeReactLoopStepDrift,
			wantInText: "react loop step drift",
		},
		{
			name:       "a56-react-tool-call-budget-drift",
			fixture:    "a56_react_tool_call_budget_drift_input.json",
			wantCode:   ReasonCodeReactToolCallBudgetDrift,
			wantInText: "react tool-call budget drift",
		},
		{
			name:       "a56-react-iteration-budget-drift",
			fixture:    "a56_react_iteration_budget_drift_input.json",
			wantCode:   ReasonCodeReactIterationBudgetDrift,
			wantInText: "react iteration budget drift",
		},
		{
			name:       "a56-react-termination-reason-drift",
			fixture:    "a56_react_termination_reason_drift_input.json",
			wantCode:   ReasonCodeReactTerminationReasonDrift,
			wantInText: "react termination reason drift",
		},
		{
			name:       "a56-react-stream-dispatch-drift",
			fixture:    "a56_react_stream_dispatch_drift_input.json",
			wantCode:   ReasonCodeReactStreamDispatchDrift,
			wantInText: "react stream dispatch drift",
		},
		{
			name:       "a56-react-provider-mapping-drift",
			fixture:    "a56_react_provider_mapping_drift_input.json",
			wantCode:   ReasonCodeReactProviderMappingDrift,
			wantInText: "react provider mapping drift",
		},
		{
			name:       "react-plan-version-drift",
			fixture:    "react_plan_version_drift_input.json",
			wantCode:   ReasonCodeReactPlanVersionDrift,
			wantInText: "react plan version drift",
		},
		{
			name:       "react-plan-change-reason-drift",
			fixture:    "react_plan_change_reason_drift_input.json",
			wantCode:   ReasonCodeReactPlanChangeReasonDrift,
			wantInText: "react plan change reason drift",
		},
		{
			name:       "react-plan-hook-semantic-drift",
			fixture:    "react_plan_hook_semantic_drift_input.json",
			wantCode:   ReasonCodeReactPlanHookSemanticDrift,
			wantInText: "react plan hook semantic drift",
		},
		{
			name:       "react-plan-recover-drift",
			fixture:    "react_plan_recover_drift_input.json",
			wantCode:   ReasonCodeReactPlanRecoverDrift,
			wantInText: "react plan recover drift",
		},
		{
			name:       "realtime-event-order-drift",
			fixture:    "realtime_event_order_drift_input.json",
			wantCode:   ReasonCodeRealtimeEventOrderDrift,
			wantInText: "realtime event order drift",
		},
		{
			name:       "realtime-interrupt-semantic-drift",
			fixture:    "realtime_interrupt_semantic_drift_input.json",
			wantCode:   ReasonCodeRealtimeInterruptSemanticDrift,
			wantInText: "realtime interrupt semantic drift",
		},
		{
			name:       "realtime-resume-semantic-drift",
			fixture:    "realtime_resume_semantic_drift_input.json",
			wantCode:   ReasonCodeRealtimeResumeSemanticDrift,
			wantInText: "realtime resume semantic drift",
		},
		{
			name:       "realtime-idempotency-drift",
			fixture:    "realtime_idempotency_drift_input.json",
			wantCode:   ReasonCodeRealtimeIdempotencyDrift,
			wantInText: "realtime idempotency drift",
		},
		{
			name:       "realtime-sequence-gap-drift",
			fixture:    "realtime_sequence_gap_drift_input.json",
			wantCode:   ReasonCodeRealtimeSequenceGapDrift,
			wantInText: "realtime sequence gap drift",
		},
		{
			name:       "context-reference-resolution-drift",
			fixture:    "context_reference_resolution_drift_input.json",
			wantCode:   ReasonCodeReferenceResolutionDrift,
			wantInText: "reference resolution drift",
		},
		{
			name:       "context-isolate-handoff-drift",
			fixture:    "context_isolate_handoff_drift_input.json",
			wantCode:   ReasonCodeIsolateHandoffDrift,
			wantInText: "isolate handoff drift",
		},
		{
			name:       "context-edit-gate-threshold-drift",
			fixture:    "context_edit_gate_threshold_drift_input.json",
			wantCode:   ReasonCodeEditGateThresholdDrift,
			wantInText: "edit gate threshold drift",
		},
		{
			name:       "context-swapback-relevance-drift",
			fixture:    "context_relevance_swapback_drift_input.json",
			wantCode:   ReasonCodeSwapbackRelevanceDrift,
			wantInText: "swapback relevance drift",
		},
		{
			name:       "context-lifecycle-tiering-drift",
			fixture:    "context_lifecycle_tiering_drift_input.json",
			wantCode:   ReasonCodeLifecycleTieringDrift,
			wantInText: "lifecycle tiering drift",
		},
		{
			name:       "context-recap-semantic-drift",
			fixture:    "context_recap_semantic_drift_input.json",
			wantCode:   ReasonCodeRecapSemanticDrift,
			wantInText: "recap semantic drift",
		},
		{
			name:       "a56-react-schema-malformed",
			fixture:    "a56_react_schema_malformed_input.json",
			wantCode:   ReasonCodeSchemaMismatch,
			wantInText: "model_provider is required",
		},
		{
			name:       "a57-sandbox-egress-action-drift",
			fixture:    "a57_sandbox_egress_action_drift_input.json",
			wantCode:   ReasonCodeSandboxEgressActionDrift,
			wantInText: "sandbox egress action drift",
		},
		{
			name:       "a57-sandbox-egress-policy-source-drift",
			fixture:    "a57_sandbox_egress_policy_source_drift_input.json",
			wantCode:   ReasonCodeSandboxEgressPolicySourceDrift,
			wantInText: "policy source drift",
		},
		{
			name:       "a57-sandbox-egress-violation-taxonomy-drift",
			fixture:    "a57_sandbox_egress_violation_taxonomy_drift_input.json",
			wantCode:   ReasonCodeSandboxEgressViolationTaxonomyDrift,
			wantInText: "violation taxonomy drift",
		},
		{
			name:       "a57-adapter-allowlist-decision-drift",
			fixture:    "a57_adapter_allowlist_decision_drift_input.json",
			wantCode:   ReasonCodeAdapterAllowlistDecisionDrift,
			wantInText: "allowlist decision drift",
		},
		{
			name:       "a57-adapter-allowlist-taxonomy-drift",
			fixture:    "a57_adapter_allowlist_taxonomy_drift_input.json",
			wantCode:   ReasonCodeAdapterAllowlistTaxonomyDrift,
			wantInText: "allowlist taxonomy drift",
		},
		{
			name:       "a58-policy-precedence-conflict",
			fixture:    "a58_policy_stack_precedence_conflict_drift_input.json",
			wantCode:   ReasonCodePrecedenceConflict,
			wantInText: "precedence conflict",
		},
		{
			name:       "a58-policy-tie-break-drift",
			fixture:    "a58_policy_stack_tie_break_drift_input.json",
			wantCode:   ReasonCodeTieBreakDrift,
			wantInText: "tie-break drift",
		},
		{
			name:       "a58-policy-deny-source-mismatch",
			fixture:    "a58_policy_stack_deny_source_mismatch_drift_input.json",
			wantCode:   ReasonCodeDenySourceMismatch,
			wantInText: "deny source mismatch",
		},
		{
			name:       "a59-scope-resolution-drift",
			fixture:    "a59_memory_scope_resolution_drift_input.json",
			wantCode:   ReasonCodeScopeResolutionDrift,
			wantInText: "scope resolution drift",
		},
		{
			name:       "a59-retrieval-quality-regression",
			fixture:    "a59_memory_retrieval_quality_regression_drift_input.json",
			wantCode:   ReasonCodeRetrievalQualityRegression,
			wantInText: "retrieval quality regression",
		},
		{
			name:       "a59-lifecycle-policy-drift",
			fixture:    "a59_memory_lifecycle_policy_drift_input.json",
			wantCode:   ReasonCodeLifecyclePolicyDrift,
			wantInText: "lifecycle policy drift",
		},
		{
			name:       "a59-recovery-consistency-drift",
			fixture:    "a59_memory_recovery_consistency_drift_input.json",
			wantCode:   ReasonCodeRecoveryConsistencyDrift,
			wantInText: "recovery consistency drift",
		},
		{
			name:       "a60-budget-threshold-drift",
			fixture:    "a60_budget_threshold_drift_input.json",
			wantCode:   ReasonCodeBudgetThresholdDrift,
			wantInText: "budget threshold drift",
		},
		{
			name:       "a60-admission-decision-drift",
			fixture:    "a60_budget_decision_drift_input.json",
			wantCode:   ReasonCodeAdmissionDecisionDrift,
			wantInText: "admission decision drift",
		},
		{
			name:       "a60-degrade-policy-drift",
			fixture:    "a60_degrade_policy_drift_input.json",
			wantCode:   ReasonCodeDegradePolicyDrift,
			wantInText: "degrade policy drift",
		},
		{
			name:       "a61-otel-attr-mapping-drift",
			fixture:    "a61_otel_attr_mapping_drift_input.json",
			wantCode:   ReasonCodeOTelAttrMappingDrift,
			wantInText: "otel attr mapping drift",
		},
		{
			name:       "a61-span-topology-drift",
			fixture:    "a61_span_topology_drift_input.json",
			wantCode:   ReasonCodeSpanTopologyDrift,
			wantInText: "span topology drift",
		},
		{
			name:       "a61-eval-metric-drift",
			fixture:    "a61_eval_metric_drift_input.json",
			wantCode:   ReasonCodeEvalMetricDrift,
			wantInText: "eval metric drift",
		},
		{
			name:       "a61-eval-aggregation-drift",
			fixture:    "a61_eval_aggregation_drift_input.json",
			wantCode:   ReasonCodeEvalAggregationDrift,
			wantInText: "eval aggregation drift",
		},
		{
			name:       "a61-eval-shard-resume-drift",
			fixture:    "a61_eval_shard_resume_drift_input.json",
			wantCode:   ReasonCodeEvalShardResumeDrift,
			wantInText: "eval shard/resume drift",
		},
		{
			name:       "a61-inferential-advisory-drift",
			fixture:    "a61_inferential_advisory_drift_input.json",
			wantCode:   ReasonCodeEvalInferentialAdvisoryDrift,
			wantInText: "inferential advisory status drift",
		},
		{
			name:       "a65-hooks-order-drift",
			fixture:    "a65_hooks_order_drift_input.json",
			wantCode:   ReasonCodeHooksOrderDrift,
			wantInText: "hooks order drift",
		},
		{
			name:       "a65-skill-discovery-source-drift",
			fixture:    "a65_skill_discovery_source_drift_input.json",
			wantCode:   ReasonCodeSkillDiscoverySourceDrift,
			wantInText: "skill discovery source drift",
		},
		{
			name:       "a65-skill-bundle-mapping-drift",
			fixture:    "a65_skill_bundle_mapping_drift_input.json",
			wantCode:   ReasonCodeSkillBundleMappingDrift,
			wantInText: "skill preprocess/mapping drift",
		},
		{
			name:       "a65-hooks-schema-mismatch",
			fixture:    "a65_hooks_middleware_schema_mismatch_input.json",
			wantCode:   ReasonCodeSchemaMismatch,
			wantInText: "hooks_phases must not be empty",
		},
		{
			name:       "a66-snapshot-schema-drift",
			fixture:    "a66_snapshot_schema_drift_input.json",
			wantCode:   ReasonCodeSnapshotSchemaDrift,
			wantInText: "state_snapshot_version",
		},
		{
			name:       "a66-state-restore-semantic-drift",
			fixture:    "a66_state_restore_semantic_drift_input.json",
			wantCode:   ReasonCodeStateRestoreSemanticDrift,
			wantInText: "semantic drift",
		},
		{
			name:       "a66-snapshot-compat-window-drift",
			fixture:    "a66_snapshot_compat_window_drift_input.json",
			wantCode:   ReasonCodeSnapshotCompatWindowDrift,
			wantInText: "compat window drift",
		},
		{
			name:       "a66-partial-restore-policy-drift",
			fixture:    "a66_partial_restore_policy_drift_input.json",
			wantCode:   ReasonCodePartialRestorePolicyDrift,
			wantInText: "partial restore policy drift",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := EvaluateArbitrationFixtureJSON(mustReadFixture(t, tc.fixture))
			if err == nil {
				t.Fatalf("fixture %q should fail", tc.fixture)
			}
			vErr, ok := err.(*ValidationError)
			if !ok {
				t.Fatalf("error type=%T, want *ValidationError", err)
			}
			if vErr.Code != tc.wantCode {
				t.Fatalf("error code=%q, want %q", vErr.Code, tc.wantCode)
			}
			if tc.wantInText != "" && !strings.Contains(strings.ToLower(vErr.Message), strings.ToLower(tc.wantInText)) {
				t.Fatalf("error message=%q, want contains %q", vErr.Message, tc.wantInText)
			}
		})
	}
}

func TestReplayContractBudgetAdmissionFixtureAndDriftClassification(t *testing.T) {
	if _, err := EvaluateArbitrationFixtureJSON(mustReadFixture(t, "a60_budget_admission_success_input.json")); err != nil {
		t.Fatalf("budget admission success fixture should pass: %v", err)
	}
	tests := []struct {
		fixture  string
		wantCode string
	}{
		{
			fixture:  "a60_budget_threshold_drift_input.json",
			wantCode: ReasonCodeBudgetThresholdDrift,
		},
		{
			fixture:  "a60_budget_decision_drift_input.json",
			wantCode: ReasonCodeAdmissionDecisionDrift,
		},
		{
			fixture:  "a60_degrade_policy_drift_input.json",
			wantCode: ReasonCodeDegradePolicyDrift,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.fixture, func(t *testing.T) {
			_, err := EvaluateArbitrationFixtureJSON(mustReadFixture(t, tc.fixture))
			if err == nil {
				t.Fatalf("fixture %q should fail", tc.fixture)
			}
			vErr, ok := err.(*ValidationError)
			if !ok {
				t.Fatalf("error type=%T, want *ValidationError", err)
			}
			if vErr.Code != tc.wantCode {
				t.Fatalf("error code=%q, want %q", vErr.Code, tc.wantCode)
			}
		})
	}
}

func TestReplayContractArbitrationMixedSandboxRolloutMemoryReactSandboxEgressCompatibility(t *testing.T) {
	fixtures := []string{
		"a52_sandbox_rollout_success_input.json",
		"a54_memory_success_input.json",
		"a59_memory_scope_success_input.json",
		"a59_memory_search_success_input.json",
		"a59_memory_lifecycle_success_input.json",
		"a60_budget_admission_success_input.json",
		"a61_otel_semconv_success_input.json",
		"a61_agent_eval_success_input.json",
		"a61_agent_eval_distributed_success_input.json",
		"a61_inferential_advisory_success_input.json",
		"a61_inferential_advisory_distributed_success_input.json",
		"a65_hooks_middleware_success_input.json",
		"a65_skill_discovery_sources_success_input.json",
		"a65_skill_preprocess_and_mapping_success_input.json",
		"a66_state_session_snapshot_success_input.json",
		"realtime_event_protocol_success_input.json",
		"context_reference_first_success_input.json",
		"context_isolate_handoff_success_input.json",
		"context_edit_gate_success_input.json",
		"context_relevance_swapback_success_input.json",
		"context_lifecycle_tiering_success_input.json",
		"a56_react_success_input.json",
		"a57_sandbox_egress_success_input.json",
	}
	for _, name := range fixtures {
		name := name
		t.Run(name, func(t *testing.T) {
			if _, err := EvaluateArbitrationFixtureJSON(mustReadFixture(t, name)); err != nil {
				t.Fatalf("fixture %q should parse and evaluate without regression: %v", name, err)
			}
		})
	}
}

func TestReplayContractArbitrationMixedPolicyPrecedenceReactSandboxEgressCompatibility(t *testing.T) {
	fixtures := []string{
		"a50_arbitration_success_input.json",
		"a56_react_success_input.json",
		"a57_sandbox_egress_success_input.json",
		"a58_policy_stack_success_input.json",
		"a60_budget_admission_success_input.json",
		"a61_otel_semconv_success_input.json",
		"a61_agent_eval_success_input.json",
		"a61_agent_eval_distributed_success_input.json",
		"a61_inferential_advisory_success_input.json",
		"a61_inferential_advisory_distributed_success_input.json",
		"a65_hooks_middleware_success_input.json",
		"a65_skill_discovery_sources_success_input.json",
		"a65_skill_preprocess_and_mapping_success_input.json",
		"a66_state_session_snapshot_success_input.json",
		"realtime_event_protocol_success_input.json",
		"context_reference_first_success_input.json",
		"context_isolate_handoff_success_input.json",
		"context_edit_gate_success_input.json",
		"context_relevance_swapback_success_input.json",
		"context_lifecycle_tiering_success_input.json",
	}
	for _, name := range fixtures {
		name := name
		t.Run(name, func(t *testing.T) {
			if _, err := EvaluateArbitrationFixtureJSON(mustReadFixture(t, name)); err != nil {
				t.Fatalf("fixture %q should parse and evaluate without regression: %v", name, err)
			}
		})
	}
}

func TestReplayContractContextSemanticFixtureAliasCompatibility(t *testing.T) {
	fixtures := []string{
		"context_reference_first_success_input.json",
		"context_isolate_handoff_success_input.json",
		"context_edit_gate_success_input.json",
		"context_relevance_swapback_success_input.json",
		"context_lifecycle_tiering_success_input.json",
	}
	for _, name := range fixtures {
		name := name
		t.Run(name, func(t *testing.T) {
			if _, err := EvaluateArbitrationFixtureJSON(mustReadFixture(t, name)); err != nil {
				t.Fatalf("semantic fixture alias %q should parse and evaluate without regression: %v", name, err)
			}
		})
	}
}

func TestReplayContractNamingMigrationDriftTaxonomyStability(t *testing.T) {
	cases := []struct {
		legacyFixture   string
		semanticFixture string
	}{
		{
			legacyFixture:   "a" + "67_react_plan_version_drift_input.json",
			semanticFixture: "react_plan_version_drift_input.json",
		},
		{
			legacyFixture:   "a" + "67_react_plan_change_reason_drift_input.json",
			semanticFixture: "react_plan_change_reason_drift_input.json",
		},
		{
			legacyFixture:   "a" + "67_react_plan_hook_semantic_drift_input.json",
			semanticFixture: "react_plan_hook_semantic_drift_input.json",
		},
		{
			legacyFixture:   "a" + "67_react_plan_recover_drift_input.json",
			semanticFixture: "react_plan_recover_drift_input.json",
		},
		{
			legacyFixture:   "a" + "67_ctx_reference_resolution_drift_input.json",
			semanticFixture: "context_reference_resolution_drift_input.json",
		},
		{
			legacyFixture:   "a" + "67_ctx_isolate_handoff_drift_input.json",
			semanticFixture: "context_isolate_handoff_drift_input.json",
		},
		{
			legacyFixture:   "a" + "67_ctx_edit_gate_threshold_drift_input.json",
			semanticFixture: "context_edit_gate_threshold_drift_input.json",
		},
		{
			legacyFixture:   "a" + "67_ctx_swapback_relevance_drift_input.json",
			semanticFixture: "context_relevance_swapback_drift_input.json",
		},
		{
			legacyFixture:   "a" + "67_ctx_lifecycle_tiering_drift_input.json",
			semanticFixture: "context_lifecycle_tiering_drift_input.json",
		},
		{
			legacyFixture:   "a" + "67_ctx_recap_semantic_drift_input.json",
			semanticFixture: "context_recap_semantic_drift_input.json",
		},
		{
			legacyFixture:   "a" + "68_realtime_event_order_drift_input.json",
			semanticFixture: "realtime_event_order_drift_input.json",
		},
		{
			legacyFixture:   "a" + "68_realtime_interrupt_semantic_drift_input.json",
			semanticFixture: "realtime_interrupt_semantic_drift_input.json",
		},
		{
			legacyFixture:   "a" + "68_realtime_resume_semantic_drift_input.json",
			semanticFixture: "realtime_resume_semantic_drift_input.json",
		},
		{
			legacyFixture:   "a" + "68_realtime_idempotency_drift_input.json",
			semanticFixture: "realtime_idempotency_drift_input.json",
		},
		{
			legacyFixture:   "a" + "68_realtime_sequence_gap_drift_input.json",
			semanticFixture: "realtime_sequence_gap_drift_input.json",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.semanticFixture, func(t *testing.T) {
			_, legacyErr := EvaluateArbitrationFixtureJSON(mustReadFixture(t, tc.legacyFixture))
			if legacyErr == nil {
				t.Fatalf("legacy drift fixture %q should fail", tc.legacyFixture)
			}
			legacyValidationErr, ok := legacyErr.(*ValidationError)
			if !ok {
				t.Fatalf("legacy drift error type=%T, want *ValidationError", legacyErr)
			}
			_, semanticErr := EvaluateArbitrationFixtureJSON(mustReadFixture(t, tc.semanticFixture))
			if semanticErr == nil {
				t.Fatalf("semantic drift fixture %q should fail", tc.semanticFixture)
			}
			semanticValidationErr, ok := semanticErr.(*ValidationError)
			if !ok {
				t.Fatalf("semantic drift error type=%T, want *ValidationError", semanticErr)
			}
			if semanticValidationErr.Code != legacyValidationErr.Code {
				t.Fatalf(
					"drift taxonomy changed for %q: legacy=%q semantic=%q",
					tc.semanticFixture,
					legacyValidationErr.Code,
					semanticValidationErr.Code,
				)
			}
		})
	}
}

func TestReplayContractTracingEvalFixtureSuite(t *testing.T) {
	fixtures := []string{
		"a61_otel_semconv_success_input.json",
		"a61_agent_eval_success_input.json",
		"a61_agent_eval_distributed_success_input.json",
		"a61_inferential_advisory_success_input.json",
		"a61_inferential_advisory_distributed_success_input.json",
	}
	for _, fixture := range fixtures {
		fixture := fixture
		t.Run(fixture, func(t *testing.T) {
			if _, err := EvaluateArbitrationFixtureJSON(mustReadFixture(t, fixture)); err != nil {
				t.Fatalf("tracing/eval fixture %q should pass: %v", fixture, err)
			}
		})
	}
}

func TestReplayContractTracingEvalDriftClassification(t *testing.T) {
	tests := []struct {
		fixture  string
		wantCode string
	}{
		{
			fixture:  "a61_otel_attr_mapping_drift_input.json",
			wantCode: ReasonCodeOTelAttrMappingDrift,
		},
		{
			fixture:  "a61_span_topology_drift_input.json",
			wantCode: ReasonCodeSpanTopologyDrift,
		},
		{
			fixture:  "a61_eval_metric_drift_input.json",
			wantCode: ReasonCodeEvalMetricDrift,
		},
		{
			fixture:  "a61_eval_aggregation_drift_input.json",
			wantCode: ReasonCodeEvalAggregationDrift,
		},
		{
			fixture:  "a61_eval_shard_resume_drift_input.json",
			wantCode: ReasonCodeEvalShardResumeDrift,
		},
		{
			fixture:  "a61_inferential_advisory_drift_input.json",
			wantCode: ReasonCodeEvalInferentialAdvisoryDrift,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.fixture, func(t *testing.T) {
			_, err := EvaluateArbitrationFixtureJSON(mustReadFixture(t, tc.fixture))
			if err == nil {
				t.Fatalf("fixture %q should fail", tc.fixture)
			}
			vErr, ok := err.(*ValidationError)
			if !ok {
				t.Fatalf("error type=%T, want *ValidationError", err)
			}
			if vErr.Code != tc.wantCode {
				t.Fatalf("error code=%q, want %q", vErr.Code, tc.wantCode)
			}
		})
	}
}

func TestReplayContractHooksMiddlewareFixtureSuite(t *testing.T) {
	fixtures := []string{
		"a65_hooks_middleware_success_input.json",
		"a65_skill_discovery_sources_success_input.json",
		"a65_skill_preprocess_and_mapping_success_input.json",
	}
	for _, fixture := range fixtures {
		fixture := fixture
		t.Run(fixture, func(t *testing.T) {
			if _, err := EvaluateArbitrationFixtureJSON(mustReadFixture(t, fixture)); err != nil {
				t.Fatalf("hooks/middleware fixture %q should pass: %v", fixture, err)
			}
		})
	}
}

func TestReplayContractHooksMiddlewareDriftClassification(t *testing.T) {
	tests := []struct {
		fixture  string
		wantCode string
	}{
		{
			fixture:  "a65_hooks_order_drift_input.json",
			wantCode: ReasonCodeHooksOrderDrift,
		},
		{
			fixture:  "a65_skill_discovery_source_drift_input.json",
			wantCode: ReasonCodeSkillDiscoverySourceDrift,
		},
		{
			fixture:  "a65_skill_bundle_mapping_drift_input.json",
			wantCode: ReasonCodeSkillBundleMappingDrift,
		},
		{
			fixture:  "a65_hooks_middleware_schema_mismatch_input.json",
			wantCode: ReasonCodeSchemaMismatch,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.fixture, func(t *testing.T) {
			_, err := EvaluateArbitrationFixtureJSON(mustReadFixture(t, tc.fixture))
			if err == nil {
				t.Fatalf("fixture %q should fail", tc.fixture)
			}
			vErr, ok := err.(*ValidationError)
			if !ok {
				t.Fatalf("error type=%T, want *ValidationError", err)
			}
			if vErr.Code != tc.wantCode {
				t.Fatalf("error code=%q, want %q", vErr.Code, tc.wantCode)
			}
		})
	}
}

func TestReplayContractTracingEvalMixedFixtureBackwardCompatibility(t *testing.T) {
	fixtures := []string{
		"a50_arbitration_success_input.json",
		"a55_observability_success_input.json",
		"a59_memory_scope_success_input.json",
		"a60_budget_admission_success_input.json",
		"a61_otel_semconv_success_input.json",
		"a61_agent_eval_success_input.json",
		"a61_agent_eval_distributed_success_input.json",
		"a61_inferential_advisory_success_input.json",
		"a61_inferential_advisory_distributed_success_input.json",
		"a65_hooks_middleware_success_input.json",
		"a65_skill_discovery_sources_success_input.json",
		"a65_skill_preprocess_and_mapping_success_input.json",
		"a66_state_session_snapshot_success_input.json",
		"realtime_event_protocol_success_input.json",
		"context_reference_first_success_input.json",
		"context_isolate_handoff_success_input.json",
		"context_edit_gate_success_input.json",
		"context_relevance_swapback_success_input.json",
		"context_lifecycle_tiering_success_input.json",
	}
	for _, fixture := range fixtures {
		fixture := fixture
		t.Run(fixture, func(t *testing.T) {
			if _, err := EvaluateArbitrationFixtureJSON(mustReadFixture(t, fixture)); err != nil {
				t.Fatalf("fixture %q should parse and evaluate without regression: %v", fixture, err)
			}
		})
	}
}
