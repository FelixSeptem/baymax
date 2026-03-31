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

func TestReplayContractArbitrationMixedA48A52MemoryCompatibility(t *testing.T) {
	fixtures := []string{
		"a48_arbitration_success_input.json",
		"a51_sandbox_success_input.json",
		"a52_sandbox_rollout_success_input.json",
		"a54_memory_success_input.json",
		"a55_observability_success_input.json",
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
