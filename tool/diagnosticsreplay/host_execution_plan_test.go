package diagnosticsreplay

import (
	"errors"
	"os"
	"testing"
)

func TestHostExecutionPlanReplayGoldenAndDrift(t *testing.T) {
	path := os.Getenv("BAYMAX_HOST_EXECUTION_PLAN_FIXTURE")
	if path == "" {
		path = "testdata/host_execution_plan.v1.json"
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ReplayHostExecutionPlanFixtureJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Reason != "" || got[1].Reason != ReasonHostExecutionPlanSideEffect || got[0].Digest == "" {
		t.Fatalf("unexpected replay: %#v", got)
	}
	for _, mutation := range [][]byte{
		[]byte(`{"version":"host_execution_plan.v1","cases":[{"case_id":"x","plan_id":"p","plan_version":1,"plan_digest":"sha256:bad","source_outcome":"completed","run_id":"r","budget_nanos":1,"source_evidence":true,"expected_reason":""}]}`),
		[]byte(`{"version":"host_execution_plan.v1","cases":[{"case_id":"x","plan_id":"p","plan_version":1,"plan_digest":"sha256:0123456789012345678901234567890123456789012345678901234567890123","source_outcome":"completed","run_id":"","budget_nanos":1,"source_evidence":true,"expected_reason":""}]}`),
		[]byte(`{"version":"host_execution_plan.v1","cases":[{"case_id":"x","plan_id":"p","plan_version":1,"plan_digest":"sha256:0123456789012345678901234567890123456789012345678901234567890123","source_outcome":"completed","run_id":"r","budget_nanos":1,"source_evidence":true,"expected_reason":"host_execution_plan.outcome_drift","callback":"must-reject"}]}`),
	} {
		_, err := ReplayHostExecutionPlanFixtureJSON(mutation)
		var validation *ValidationError
		if !errors.As(err, &validation) || validation.Code != ReasonHostExecutionPlanReplayDrift {
			t.Fatalf("drift should fail: %v", err)
		}
	}
}

func TestHostExecutionPlanReplayIsReferenceOnly(t *testing.T) {
	// The fixture schema has no executable references, and parsing it must not
	// call a callback even when a JSON string looks like one.
	raw := []byte(`{"version":"host_execution_plan.v1","cases":[{"case_id":"x","plan_id":"p","plan_version":1,"plan_digest":"sha256:0123456789012345678901234567890123456789012345678901234567890123","source_outcome":"completed","run_id":"r","budget_nanos":1,"source_evidence":true,"expected_reason":""}]}`)
	if _, err := ReplayHostExecutionPlanFixtureJSON(raw); err != nil {
		t.Fatal(err)
	}
}
