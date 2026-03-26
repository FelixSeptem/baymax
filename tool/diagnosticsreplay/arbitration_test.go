package diagnosticsreplay

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestReplayContractPrimaryReasonArbitrationFixtureSuccessAndDeterministicOutput(t *testing.T) {
	input := mustReadFixture(t, "a48_arbitration_success_input.json")
	expectedRaw := mustReadFixture(t, "a48_arbitration_success_expected.json")

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
			fixture:    "a48_arbitration_precedence_drift_input.json",
			wantCode:   ReasonCodePrecedenceDrift,
			wantInText: "precedence drift",
		},
		{
			name:       "tie-break",
			fixture:    "a48_arbitration_tie_break_drift_input.json",
			wantCode:   ReasonCodeTieBreakDrift,
			wantInText: "tie-break drift",
		},
		{
			name:       "taxonomy",
			fixture:    "a48_arbitration_taxonomy_drift_input.json",
			wantCode:   ReasonCodeTaxonomyDrift,
			wantInText: "non-canonical primary code",
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
