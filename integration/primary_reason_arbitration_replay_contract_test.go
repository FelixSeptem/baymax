package integration

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/FelixSeptem/baymax/tool/diagnosticsreplay"
)

var semanticFixtureLegacyAliases = map[string]string{
	"context_reference_first_success_input.json":    "a67_ctx_reference_first_success_input.json",
	"context_isolate_handoff_success_input.json":    "a67_ctx_isolate_handoff_success_input.json",
	"context_edit_gate_success_input.json":          "a67_ctx_edit_gate_success_input.json",
	"context_relevance_swapback_success_input.json": "a67_ctx_swapback_success_input.json",
	"context_lifecycle_tiering_success_input.json":  "a67_ctx_lifecycle_tiering_success_input.json",
	"context_reference_resolution_drift_input.json": "a67_ctx_reference_resolution_drift_input.json",
	"context_isolate_handoff_drift_input.json":      "a67_ctx_isolate_handoff_drift_input.json",
	"context_edit_gate_threshold_drift_input.json":  "a67_ctx_edit_gate_threshold_drift_input.json",
	"context_relevance_swapback_drift_input.json":   "a67_ctx_swapback_relevance_drift_input.json",
	"context_lifecycle_tiering_drift_input.json":    "a67_ctx_lifecycle_tiering_drift_input.json",
	"context_recap_semantic_drift_input.json":       "a67_ctx_recap_semantic_drift_input.json",
	"realtime_event_protocol_success_input.json":    "a68_realtime_event_protocol_success_input.json",
	"realtime_event_order_drift_input.json":         "a68_realtime_event_order_drift_input.json",
	"realtime_interrupt_semantic_drift_input.json":  "a68_realtime_interrupt_semantic_drift_input.json",
	"realtime_resume_semantic_drift_input.json":     "a68_realtime_resume_semantic_drift_input.json",
	"realtime_idempotency_drift_input.json":         "a68_realtime_idempotency_drift_input.json",
	"realtime_sequence_gap_drift_input.json":        "a68_realtime_sequence_gap_drift_input.json",
}

func TestPrimaryReasonArbitrationReplayContractFixtureSuite(t *testing.T) {
	tests := []struct {
		name          string
		versionFolder string
		fixture       string
		expected      string
	}{
		{
			name:          "a49",
			versionFolder: "a49",
			fixture:       "success.json",
			expected:      diagnosticsreplay.ArbitrationFixtureVersionExplainabilityV1,
		},
		{
			name:          "a50",
			versionFolder: "a50",
			fixture:       "success.json",
			expected:      diagnosticsreplay.ArbitrationFixtureVersionVersionGovernanceV1,
		},
		{
			name:          "a51",
			versionFolder: "tool",
			fixture:       "a51_sandbox_success_input.json",
			expected:      diagnosticsreplay.ArbitrationFixtureVersionSandboxExecutionV1,
		},
		{
			name:          "a57-sandbox-egress",
			versionFolder: "tool",
			fixture:       "a57_sandbox_egress_success_input.json",
			expected:      diagnosticsreplay.ArbitrationFixtureVersionSandboxEgressV1,
		},
		{
			name:          "a58-policy-stack",
			versionFolder: "tool",
			fixture:       "a58_policy_stack_success_input.json",
			expected:      diagnosticsreplay.ArbitrationFixtureVersionPolicyV1,
		},
		{
			name:          "a59-memory-scope",
			versionFolder: "tool",
			fixture:       "a59_memory_scope_success_input.json",
			expected:      diagnosticsreplay.ArbitrationFixtureVersionMemoryScopeV1,
		},
		{
			name:          "a59-memory-search",
			versionFolder: "tool",
			fixture:       "a59_memory_search_success_input.json",
			expected:      diagnosticsreplay.ArbitrationFixtureVersionMemorySearchV1,
		},
		{
			name:          "a59-memory-lifecycle",
			versionFolder: "tool",
			fixture:       "a59_memory_lifecycle_success_input.json",
			expected:      diagnosticsreplay.ArbitrationFixtureVersionMemoryLifecycleV1,
		},
		{
			name:          "a60-budget-admission",
			versionFolder: "tool",
			fixture:       "a60_budget_admission_success_input.json",
			expected:      diagnosticsreplay.ArbitrationFixtureVersionBudgetAdmissionV1,
		},
		{
			name:          "a61-otel-semconv",
			versionFolder: "tool",
			fixture:       "a61_otel_semconv_success_input.json",
			expected:      diagnosticsreplay.ArbitrationFixtureVersionOTelSemconvV1,
		},
		{
			name:          "a61-agent-eval",
			versionFolder: "tool",
			fixture:       "a61_agent_eval_success_input.json",
			expected:      diagnosticsreplay.ArbitrationFixtureVersionAgentEvalV1,
		},
		{
			name:          "a61-agent-eval-distributed",
			versionFolder: "tool",
			fixture:       "a61_agent_eval_distributed_success_input.json",
			expected:      diagnosticsreplay.ArbitrationFixtureVersionAgentEvalDistV1,
		},
		{
			name:          "a65-hooks-middleware",
			versionFolder: "tool",
			fixture:       "a65_hooks_middleware_success_input.json",
			expected:      diagnosticsreplay.ArbitrationFixtureVersionHooksMiddlewareV1,
		},
		{
			name:          "a65-skill-discovery-sources",
			versionFolder: "tool",
			fixture:       "a65_skill_discovery_sources_success_input.json",
			expected:      diagnosticsreplay.ArbitrationFixtureVersionSkillDiscoveryV1,
		},
		{
			name:          "a65-skill-preprocess-mapping",
			versionFolder: "tool",
			fixture:       "a65_skill_preprocess_and_mapping_success_input.json",
			expected:      diagnosticsreplay.ArbitrationFixtureVersionSkillMappingV1,
		},
		{
			name:          "realtime-protocol",
			versionFolder: "tool",
			fixture:       "realtime_event_protocol_success_input.json",
			expected:      diagnosticsreplay.ArbitrationFixtureVersionRealtimeProtocolV1,
		},
		{
			name:          "context-reference-first",
			versionFolder: "tool",
			fixture:       "context_reference_first_success_input.json",
			expected:      diagnosticsreplay.ArbitrationFixtureVersionContextRefFirstV1,
		},
		{
			name:          "context-isolate-handoff",
			versionFolder: "tool",
			fixture:       "context_isolate_handoff_success_input.json",
			expected:      diagnosticsreplay.ArbitrationFixtureVersionContextHandoffV1,
		},
		{
			name:          "context-edit-gate",
			versionFolder: "tool",
			fixture:       "context_edit_gate_success_input.json",
			expected:      diagnosticsreplay.ArbitrationFixtureVersionContextEditGateV1,
		},
		{
			name:          "context-swapback",
			versionFolder: "tool",
			fixture:       "context_relevance_swapback_success_input.json",
			expected:      diagnosticsreplay.ArbitrationFixtureVersionContextSwapBackV1,
		},
		{
			name:          "context-lifecycle-tiering",
			versionFolder: "tool",
			fixture:       "context_lifecycle_tiering_success_input.json",
			expected:      diagnosticsreplay.ArbitrationFixtureVersionContextTieringV1,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			raw := mustReadArbitrationReplayFixture(t, tc.versionFolder, tc.fixture)
			out, err := diagnosticsreplay.EvaluateArbitrationFixtureJSON(raw)
			if err != nil {
				t.Fatalf("EvaluateArbitrationFixtureJSON success fixture failed: %v", err)
			}
			if strings.TrimSpace(out.Version) != tc.expected {
				t.Fatalf("fixture version=%q, want %q", out.Version, tc.expected)
			}
			if len(out.Cases) < 1 {
				t.Fatalf("normalized cases len=%d, want >= 1", len(out.Cases))
			}
			replayOut, err := diagnosticsreplay.EvaluateArbitrationFixtureJSON(raw)
			if err != nil {
				t.Fatalf("EvaluateArbitrationFixtureJSON replay failed: %v", err)
			}
			if !reflect.DeepEqual(out, replayOut) {
				t.Fatalf("replay output drift first=%#v replay=%#v", out, replayOut)
			}
		})
	}
}

func TestReplayContractBudgetAdmissionFixtureCompatibility(t *testing.T) {
	raw := mustReadArbitrationReplayFixture(t, "tool", "a60_budget_admission_success_input.json")
	out, err := diagnosticsreplay.EvaluateArbitrationFixtureJSON(raw)
	if err != nil {
		t.Fatalf("EvaluateArbitrationFixtureJSON budget success fixture failed: %v", err)
	}
	if strings.TrimSpace(out.Version) != diagnosticsreplay.ArbitrationFixtureVersionBudgetAdmissionV1 {
		t.Fatalf("fixture version=%q, want %q", out.Version, diagnosticsreplay.ArbitrationFixtureVersionBudgetAdmissionV1)
	}
}

func TestPrimaryReasonArbitrationReplayContractDriftGuardFailFast(t *testing.T) {
	tests := []struct {
		name       string
		versionDir string
		fixture    string
		wantCode   string
		messageHas string
	}{
		{
			name:       "precedence",
			versionDir: "a49",
			fixture:    "drift-precedence.json",
			wantCode:   diagnosticsreplay.ReasonCodePrecedenceDrift,
			messageHas: "precedence drift",
		},
		{
			name:       "tie-break",
			versionDir: "a49",
			fixture:    "drift-tie-break.json",
			wantCode:   diagnosticsreplay.ReasonCodeTieBreakDrift,
			messageHas: "tie-break drift",
		},
		{
			name:       "taxonomy",
			versionDir: "a49",
			fixture:    "drift-taxonomy.json",
			wantCode:   diagnosticsreplay.ReasonCodeTaxonomyDrift,
			messageHas: "non-canonical primary code",
		},
		{
			name:       "secondary-order",
			versionDir: "a49",
			fixture:    "drift-secondary-order.json",
			wantCode:   diagnosticsreplay.ReasonCodeSecondaryOrderDrift,
			messageHas: "secondary order drift",
		},
		{
			name:       "secondary-count",
			versionDir: "a49",
			fixture:    "drift-secondary-count.json",
			wantCode:   diagnosticsreplay.ReasonCodeSecondaryCountDrift,
			messageHas: "secondary count drift",
		},
		{
			name:       "hint-taxonomy",
			versionDir: "a49",
			fixture:    "drift-hint-taxonomy.json",
			wantCode:   diagnosticsreplay.ReasonCodeHintTaxonomyDrift,
			messageHas: "hint taxonomy drift",
		},
		{
			name:       "rule-version",
			versionDir: "a49",
			fixture:    "drift-rule-version.json",
			wantCode:   diagnosticsreplay.ReasonCodeRuleVersionDrift,
			messageHas: "rule version drift",
		},
		{
			name:       "a50-version-mismatch",
			versionDir: "a50",
			fixture:    "drift-version-mismatch.json",
			wantCode:   diagnosticsreplay.ReasonCodeVersionMismatch,
			messageHas: "version mismatch",
		},
		{
			name:       "a50-unsupported-version",
			versionDir: "a50",
			fixture:    "drift-unsupported-version.json",
			wantCode:   diagnosticsreplay.ReasonCodeUnsupportedVersion,
			messageHas: "unsupported version",
		},
		{
			name:       "a50-cross-version-semantic-drift",
			versionDir: "a50",
			fixture:    "drift-cross-version-semantic-drift.json",
			wantCode:   diagnosticsreplay.ReasonCodeCrossVersionSemanticDrift,
			messageHas: "cross-version semantic drift",
		},
		{
			name:       "a51-sandbox-policy-drift",
			versionDir: "tool",
			fixture:    "a51_sandbox_policy_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeSandboxPolicyDrift,
			messageHas: "sandbox policy drift",
		},
		{
			name:       "a51-sandbox-fallback-drift",
			versionDir: "tool",
			fixture:    "a51_sandbox_fallback_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeSandboxFallbackDrift,
			messageHas: "sandbox fallback drift",
		},
		{
			name:       "a51-sandbox-timeout-drift",
			versionDir: "tool",
			fixture:    "a51_sandbox_timeout_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeSandboxTimeoutDrift,
			messageHas: "sandbox timeout drift",
		},
		{
			name:       "a51-sandbox-capability-drift",
			versionDir: "tool",
			fixture:    "a51_sandbox_capability_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeSandboxCapabilityDrift,
			messageHas: "sandbox capability drift",
		},
		{
			name:       "a51-sandbox-resource-policy-drift",
			versionDir: "tool",
			fixture:    "a51_sandbox_resource_policy_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeSandboxResourcePolicyDrift,
			messageHas: "sandbox resource policy drift",
		},
		{
			name:       "a51-sandbox-session-lifecycle-drift",
			versionDir: "tool",
			fixture:    "a51_sandbox_session_lifecycle_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeSandboxSessionLifecycleDrift,
			messageHas: "sandbox session lifecycle drift",
		},
		{
			name:       "a57-sandbox-egress-action-drift",
			versionDir: "tool",
			fixture:    "a57_sandbox_egress_action_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeSandboxEgressActionDrift,
			messageHas: "sandbox egress action drift",
		},
		{
			name:       "a57-sandbox-egress-policy-source-drift",
			versionDir: "tool",
			fixture:    "a57_sandbox_egress_policy_source_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeSandboxEgressPolicySourceDrift,
			messageHas: "policy source drift",
		},
		{
			name:       "a57-sandbox-egress-violation-taxonomy-drift",
			versionDir: "tool",
			fixture:    "a57_sandbox_egress_violation_taxonomy_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeSandboxEgressViolationTaxonomyDrift,
			messageHas: "violation taxonomy drift",
		},
		{
			name:       "a57-adapter-allowlist-decision-drift",
			versionDir: "tool",
			fixture:    "a57_adapter_allowlist_decision_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeAdapterAllowlistDecisionDrift,
			messageHas: "allowlist decision drift",
		},
		{
			name:       "a57-adapter-allowlist-taxonomy-drift",
			versionDir: "tool",
			fixture:    "a57_adapter_allowlist_taxonomy_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeAdapterAllowlistTaxonomyDrift,
			messageHas: "allowlist taxonomy drift",
		},
		{
			name:       "a58-policy-precedence-conflict",
			versionDir: "tool",
			fixture:    "a58_policy_stack_precedence_conflict_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodePrecedenceConflict,
			messageHas: "precedence conflict",
		},
		{
			name:       "a58-policy-tie-break-drift",
			versionDir: "tool",
			fixture:    "a58_policy_stack_tie_break_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeTieBreakDrift,
			messageHas: "tie-break drift",
		},
		{
			name:       "a58-policy-deny-source-mismatch",
			versionDir: "tool",
			fixture:    "a58_policy_stack_deny_source_mismatch_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeDenySourceMismatch,
			messageHas: "deny source mismatch",
		},
		{
			name:       "a59-scope-resolution-drift",
			versionDir: "tool",
			fixture:    "a59_memory_scope_resolution_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeScopeResolutionDrift,
			messageHas: "scope resolution drift",
		},
		{
			name:       "a59-retrieval-quality-regression",
			versionDir: "tool",
			fixture:    "a59_memory_retrieval_quality_regression_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeRetrievalQualityRegression,
			messageHas: "retrieval quality regression",
		},
		{
			name:       "a59-lifecycle-policy-drift",
			versionDir: "tool",
			fixture:    "a59_memory_lifecycle_policy_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeLifecyclePolicyDrift,
			messageHas: "lifecycle policy drift",
		},
		{
			name:       "a59-recovery-consistency-drift",
			versionDir: "tool",
			fixture:    "a59_memory_recovery_consistency_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeRecoveryConsistencyDrift,
			messageHas: "recovery consistency drift",
		},
		{
			name:       "a60-budget-threshold-drift",
			versionDir: "tool",
			fixture:    "a60_budget_threshold_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeBudgetThresholdDrift,
			messageHas: "budget threshold drift",
		},
		{
			name:       "a60-admission-decision-drift",
			versionDir: "tool",
			fixture:    "a60_budget_decision_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeAdmissionDecisionDrift,
			messageHas: "admission decision drift",
		},
		{
			name:       "a60-degrade-policy-drift",
			versionDir: "tool",
			fixture:    "a60_degrade_policy_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeDegradePolicyDrift,
			messageHas: "degrade policy drift",
		},
		{
			name:       "a61-otel-attr-mapping-drift",
			versionDir: "tool",
			fixture:    "a61_otel_attr_mapping_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeOTelAttrMappingDrift,
			messageHas: "otel attr mapping drift",
		},
		{
			name:       "a61-span-topology-drift",
			versionDir: "tool",
			fixture:    "a61_span_topology_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeSpanTopologyDrift,
			messageHas: "span topology drift",
		},
		{
			name:       "a61-eval-metric-drift",
			versionDir: "tool",
			fixture:    "a61_eval_metric_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeEvalMetricDrift,
			messageHas: "eval metric drift",
		},
		{
			name:       "a61-eval-aggregation-drift",
			versionDir: "tool",
			fixture:    "a61_eval_aggregation_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeEvalAggregationDrift,
			messageHas: "eval aggregation drift",
		},
		{
			name:       "a61-eval-shard-resume-drift",
			versionDir: "tool",
			fixture:    "a61_eval_shard_resume_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeEvalShardResumeDrift,
			messageHas: "eval shard/resume drift",
		},
		{
			name:       "a65-hooks-order-drift",
			versionDir: "tool",
			fixture:    "a65_hooks_order_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeHooksOrderDrift,
			messageHas: "hooks order drift",
		},
		{
			name:       "a65-skill-discovery-source-drift",
			versionDir: "tool",
			fixture:    "a65_skill_discovery_source_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeSkillDiscoverySourceDrift,
			messageHas: "skill discovery source drift",
		},
		{
			name:       "a65-skill-bundle-mapping-drift",
			versionDir: "tool",
			fixture:    "a65_skill_bundle_mapping_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeSkillBundleMappingDrift,
			messageHas: "skill preprocess/mapping drift",
		},
		{
			name:       "realtime-event-order-drift",
			versionDir: "tool",
			fixture:    "realtime_event_order_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeRealtimeEventOrderDrift,
			messageHas: "realtime event order drift",
		},
		{
			name:       "realtime-interrupt-semantic-drift",
			versionDir: "tool",
			fixture:    "realtime_interrupt_semantic_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeRealtimeInterruptSemanticDrift,
			messageHas: "realtime interrupt semantic drift",
		},
		{
			name:       "realtime-resume-semantic-drift",
			versionDir: "tool",
			fixture:    "realtime_resume_semantic_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeRealtimeResumeSemanticDrift,
			messageHas: "realtime resume semantic drift",
		},
		{
			name:       "realtime-idempotency-drift",
			versionDir: "tool",
			fixture:    "realtime_idempotency_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeRealtimeIdempotencyDrift,
			messageHas: "realtime idempotency drift",
		},
		{
			name:       "realtime-sequence-gap-drift",
			versionDir: "tool",
			fixture:    "realtime_sequence_gap_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeRealtimeSequenceGapDrift,
			messageHas: "realtime sequence gap drift",
		},
		{
			name:       "context-reference-resolution-drift",
			versionDir: "tool",
			fixture:    "context_reference_resolution_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeReferenceResolutionDrift,
			messageHas: "reference resolution drift",
		},
		{
			name:       "context-isolate-handoff-drift",
			versionDir: "tool",
			fixture:    "context_isolate_handoff_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeIsolateHandoffDrift,
			messageHas: "isolate handoff drift",
		},
		{
			name:       "context-edit-gate-threshold-drift",
			versionDir: "tool",
			fixture:    "context_edit_gate_threshold_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeEditGateThresholdDrift,
			messageHas: "edit gate threshold drift",
		},
		{
			name:       "context-swapback-relevance-drift",
			versionDir: "tool",
			fixture:    "context_relevance_swapback_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeSwapbackRelevanceDrift,
			messageHas: "swapback relevance drift",
		},
		{
			name:       "context-lifecycle-tiering-drift",
			versionDir: "tool",
			fixture:    "context_lifecycle_tiering_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeLifecycleTieringDrift,
			messageHas: "lifecycle tiering drift",
		},
		{
			name:       "context-recap-semantic-drift",
			versionDir: "tool",
			fixture:    "context_recap_semantic_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeRecapSemanticDrift,
			messageHas: "recap semantic drift",
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := diagnosticsreplay.EvaluateArbitrationFixtureJSON(
				mustReadArbitrationReplayFixture(t, tc.versionDir, tc.fixture),
			)
			if err == nil {
				t.Fatalf("fixture %q should fail", tc.fixture)
			}
			vErr, ok := err.(*diagnosticsreplay.ValidationError)
			if !ok {
				t.Fatalf("error type=%T, want *ValidationError", err)
			}
			if vErr.Code != tc.wantCode {
				t.Fatalf("error code=%q, want %q", vErr.Code, tc.wantCode)
			}
			if !strings.Contains(strings.ToLower(vErr.Message), strings.ToLower(tc.messageHas)) {
				t.Fatalf("error message=%q, want contains %q", vErr.Message, tc.messageHas)
			}
		})
	}
}

func TestReplayContractSandboxEgressAllowlistFixture(t *testing.T) {
	raw := mustReadArbitrationReplayFixture(t, "tool", "a57_sandbox_egress_success_input.json")
	out, err := diagnosticsreplay.EvaluateArbitrationFixtureJSON(raw)
	if err != nil {
		t.Fatalf("EvaluateArbitrationFixtureJSON success fixture failed: %v", err)
	}
	if strings.TrimSpace(out.Version) != diagnosticsreplay.ArbitrationFixtureVersionSandboxEgressV1 {
		t.Fatalf("fixture version=%q, want %q", out.Version, diagnosticsreplay.ArbitrationFixtureVersionSandboxEgressV1)
	}
	if len(out.Cases) == 0 {
		t.Fatal("normalized output cases should not be empty")
	}
	replayOut, err := diagnosticsreplay.EvaluateArbitrationFixtureJSON(raw)
	if err != nil {
		t.Fatalf("EvaluateArbitrationFixtureJSON replay failed: %v", err)
	}
	if !reflect.DeepEqual(out, replayOut) {
		t.Fatalf("replay output drift first=%#v replay=%#v", out, replayOut)
	}

	_, err = diagnosticsreplay.EvaluateArbitrationFixtureJSON(
		mustReadArbitrationReplayFixture(t, "tool", "a57_adapter_allowlist_taxonomy_drift_input.json"),
	)
	if err == nil {
		t.Fatal("taxonomy drift fixture should fail")
	}
	vErr, ok := err.(*diagnosticsreplay.ValidationError)
	if !ok {
		t.Fatalf("error type=%T, want *ValidationError", err)
	}
	if vErr.Code != diagnosticsreplay.ReasonCodeAdapterAllowlistTaxonomyDrift {
		t.Fatalf("error code=%q, want %q", vErr.Code, diagnosticsreplay.ReasonCodeAdapterAllowlistTaxonomyDrift)
	}
}

func TestReplayContractPolicyPrecedenceFixture(t *testing.T) {
	raw := mustReadArbitrationReplayFixture(t, "tool", "a58_policy_stack_success_input.json")
	out, err := diagnosticsreplay.EvaluateArbitrationFixtureJSON(raw)
	if err != nil {
		t.Fatalf("EvaluateArbitrationFixtureJSON success fixture failed: %v", err)
	}
	if strings.TrimSpace(out.Version) != diagnosticsreplay.ArbitrationFixtureVersionPolicyV1 {
		t.Fatalf("fixture version=%q, want %q", out.Version, diagnosticsreplay.ArbitrationFixtureVersionPolicyV1)
	}
	if len(out.Cases) == 0 {
		t.Fatal("normalized output cases should not be empty")
	}
	replayOut, err := diagnosticsreplay.EvaluateArbitrationFixtureJSON(raw)
	if err != nil {
		t.Fatalf("EvaluateArbitrationFixtureJSON replay failed: %v", err)
	}
	if !reflect.DeepEqual(out, replayOut) {
		t.Fatalf("replay output drift first=%#v replay=%#v", out, replayOut)
	}

	_, err = diagnosticsreplay.EvaluateArbitrationFixtureJSON(
		mustReadArbitrationReplayFixture(t, "tool", "a58_policy_stack_precedence_conflict_drift_input.json"),
	)
	if err == nil {
		t.Fatal("precedence conflict drift fixture should fail")
	}
	vErr, ok := err.(*diagnosticsreplay.ValidationError)
	if !ok {
		t.Fatalf("error type=%T, want *ValidationError", err)
	}
	if vErr.Code != diagnosticsreplay.ReasonCodePrecedenceConflict {
		t.Fatalf("error code=%q, want %q", vErr.Code, diagnosticsreplay.ReasonCodePrecedenceConflict)
	}
}

func TestReplayContractMixedPolicyPrecedenceReactSandboxEgressCompatibility(t *testing.T) {
	fixtures := []string{
		"a50_arbitration_success_input.json",
		"a56_react_success_input.json",
		"a57_sandbox_egress_success_input.json",
		"a58_policy_stack_success_input.json",
		"a59_memory_scope_success_input.json",
		"a59_memory_search_success_input.json",
		"a59_memory_lifecycle_success_input.json",
		"a60_budget_admission_success_input.json",
		"a61_otel_semconv_success_input.json",
		"a61_agent_eval_success_input.json",
		"a61_agent_eval_distributed_success_input.json",
		"a65_hooks_middleware_success_input.json",
		"a65_skill_discovery_sources_success_input.json",
		"a65_skill_preprocess_and_mapping_success_input.json",
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
			if _, err := diagnosticsreplay.EvaluateArbitrationFixtureJSON(
				mustReadArbitrationReplayFixture(t, "tool", name),
			); err != nil {
				t.Fatalf("fixture %q should parse and evaluate without regression: %v", name, err)
			}
		})
	}
}

func TestReplayContractTracingEvalFixtureSuite(t *testing.T) {
	fixtures := []string{
		"a61_otel_semconv_success_input.json",
		"a61_agent_eval_success_input.json",
		"a61_agent_eval_distributed_success_input.json",
	}
	for _, fixture := range fixtures {
		fixture := fixture
		t.Run(fixture, func(t *testing.T) {
			if _, err := diagnosticsreplay.EvaluateArbitrationFixtureJSON(
				mustReadArbitrationReplayFixture(t, "tool", fixture),
			); err != nil {
				t.Fatalf("tracing/eval fixture %q should pass: %v", fixture, err)
			}
		})
	}
}

func TestReplayContractTracingEvalDriftGuardFailFast(t *testing.T) {
	tests := []struct {
		fixture  string
		wantCode string
	}{
		{
			fixture:  "a61_otel_attr_mapping_drift_input.json",
			wantCode: diagnosticsreplay.ReasonCodeOTelAttrMappingDrift,
		},
		{
			fixture:  "a61_span_topology_drift_input.json",
			wantCode: diagnosticsreplay.ReasonCodeSpanTopologyDrift,
		},
		{
			fixture:  "a61_eval_metric_drift_input.json",
			wantCode: diagnosticsreplay.ReasonCodeEvalMetricDrift,
		},
		{
			fixture:  "a61_eval_aggregation_drift_input.json",
			wantCode: diagnosticsreplay.ReasonCodeEvalAggregationDrift,
		},
		{
			fixture:  "a61_eval_shard_resume_drift_input.json",
			wantCode: diagnosticsreplay.ReasonCodeEvalShardResumeDrift,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.fixture, func(t *testing.T) {
			_, err := diagnosticsreplay.EvaluateArbitrationFixtureJSON(
				mustReadArbitrationReplayFixture(t, "tool", tc.fixture),
			)
			if err == nil {
				t.Fatalf("fixture %q should fail", tc.fixture)
			}
			vErr, ok := err.(*diagnosticsreplay.ValidationError)
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
			if _, err := diagnosticsreplay.EvaluateArbitrationFixtureJSON(
				mustReadArbitrationReplayFixture(t, "tool", fixture),
			); err != nil {
				t.Fatalf("hooks/middleware fixture %q should pass: %v", fixture, err)
			}
		})
	}
}

func TestReplayContractHooksMiddlewareDriftGuardFailFast(t *testing.T) {
	tests := []struct {
		fixture  string
		wantCode string
	}{
		{
			fixture:  "a65_hooks_order_drift_input.json",
			wantCode: diagnosticsreplay.ReasonCodeHooksOrderDrift,
		},
		{
			fixture:  "a65_skill_discovery_source_drift_input.json",
			wantCode: diagnosticsreplay.ReasonCodeSkillDiscoverySourceDrift,
		},
		{
			fixture:  "a65_skill_bundle_mapping_drift_input.json",
			wantCode: diagnosticsreplay.ReasonCodeSkillBundleMappingDrift,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.fixture, func(t *testing.T) {
			_, err := diagnosticsreplay.EvaluateArbitrationFixtureJSON(
				mustReadArbitrationReplayFixture(t, "tool", tc.fixture),
			)
			if err == nil {
				t.Fatalf("fixture %q should fail", tc.fixture)
			}
			vErr, ok := err.(*diagnosticsreplay.ValidationError)
			if !ok {
				t.Fatalf("error type=%T, want *ValidationError", err)
			}
			if vErr.Code != tc.wantCode {
				t.Fatalf("error code=%q, want %q", vErr.Code, tc.wantCode)
			}
		})
	}
}

func mustReadArbitrationReplayFixture(t *testing.T, versionDir, name string) []byte {
	t.Helper()
	root := repoRootForArbitrationReplay(t)
	if strings.EqualFold(strings.TrimSpace(versionDir), "tool") {
		path := filepath.Join(
			root,
			"tool",
			"diagnosticsreplay",
			"testdata",
			name,
		)
		raw, err := os.ReadFile(path)
		if err == nil {
			return raw
		}
		if alias, ok := semanticFixtureLegacyAliases[name]; ok {
			aliasPath := filepath.Join(
				root,
				"tool",
				"diagnosticsreplay",
				"testdata",
				alias,
			)
			raw, aliasErr := os.ReadFile(aliasPath)
			if aliasErr == nil {
				return raw
			}
			t.Fatalf("read fixture %s via alias %s: %v", path, aliasPath, aliasErr)
		}
		t.Fatalf("read fixture %s: %v", path, err)
	}

	path := filepath.Join(
		root,
		"integration",
		"testdata",
		"diagnostics-replay",
		versionDir,
		"v1",
		name,
	)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	return raw
}

func repoRootForArbitrationReplay(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve runtime caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), ".."))
}
