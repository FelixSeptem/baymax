package diagnosticsreplay

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestReplayContractCompositeFixtureSuccessAndDeterministicNormalizedOutput(t *testing.T) {
	input := mustReadFixture(t, "readiness_timeout_health_composite_success_input.json")
	expectedRaw := mustReadFixture(t, "readiness_timeout_health_composite_success_expected.json")

	got, err := EvaluateCompositeFixtureJSON(input)
	if err != nil {
		t.Fatalf("EvaluateCompositeFixtureJSON error: %v", err)
	}

	var want CompositeReplayOutput
	if err := json.Unmarshal(expectedRaw, &want); err != nil {
		t.Fatalf("unmarshal expected fixture: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("composite replay output mismatch\n got: %#v\nwant: %#v", got, want)
	}

	run2, err := EvaluateCompositeFixtureJSON(input)
	if err != nil {
		t.Fatalf("EvaluateCompositeFixtureJSON second run error: %v", err)
	}
	if !reflect.DeepEqual(got, run2) {
		t.Fatalf("composite replay output should be deterministic: first=%#v second=%#v", got, run2)
	}
}

func TestReplayContractCompositeFixtureMissingAxisSchemaMismatch(t *testing.T) {
	input := mustReadFixture(t, "readiness_timeout_health_composite_missing_axis_input.json")
	_, err := EvaluateCompositeFixtureJSON(input)
	if err == nil {
		t.Fatal("expected schema mismatch when matrix coverage axes are missing")
	}
	vErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("error type = %T, want *ValidationError", err)
	}
	if vErr.Code != ReasonCodeSchemaMismatch {
		t.Fatalf("code = %q, want %q", vErr.Code, ReasonCodeSchemaMismatch)
	}
}

func TestReplayContractCompositeFixtureSemanticDrift(t *testing.T) {
	input := mustReadFixture(t, "readiness_timeout_health_composite_semantic_drift_input.json")
	_, err := EvaluateCompositeFixtureJSON(input)
	if err == nil {
		t.Fatal("expected semantic drift failure")
	}
	vErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("error type = %T, want *ValidationError", err)
	}
	if vErr.Code != ReasonCodeSemanticDrift {
		t.Fatalf("code = %q, want %q", vErr.Code, ReasonCodeSemanticDrift)
	}
}

func TestReplayContractCompositeFixtureOrderingDrift(t *testing.T) {
	input := mustReadFixture(t, "readiness_timeout_health_composite_ordering_drift_input.json")
	_, err := EvaluateCompositeFixtureJSON(input)
	if err == nil {
		t.Fatal("expected ordering drift failure")
	}
	vErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("error type = %T, want *ValidationError", err)
	}
	if vErr.Code != ReasonCodeOrderingDrift {
		t.Fatalf("code = %q, want %q", vErr.Code, ReasonCodeOrderingDrift)
	}
}

func TestReplayContractCompositeFixtureIdempotencyDrift(t *testing.T) {
	input := mustReadFixture(t, "readiness_timeout_health_composite_idempotency_drift_input.json")
	_, err := EvaluateCompositeFixtureJSON(input)
	if err == nil {
		t.Fatal("expected idempotency drift failure")
	}
	vErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("error type = %T, want *ValidationError", err)
	}
	if vErr.Code != ReasonCodeSemanticDrift {
		t.Fatalf("code = %q, want %q", vErr.Code, ReasonCodeSemanticDrift)
	}
}
