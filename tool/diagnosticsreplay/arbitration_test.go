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
