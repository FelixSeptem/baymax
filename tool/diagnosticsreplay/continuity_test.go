package diagnosticsreplay

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReplayContractContinuityFixtureFile(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "eval_continuity_comparison.v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := ParseContinuityFixtureJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := EvaluateContinuityFixture(f); err != nil {
		t.Fatal(err)
	}
}

func TestContinuityFixtureSuccessAndPrivacyValidation(t *testing.T) {
	raw := []byte(`{"version":"eval_continuity_comparison.v1","baseline":{"version":"eval_continuity_comparison.v1","run_id":"r1","phase":"baseline","facts":[{"kind":"objective","owner":"runner","id":"o","digest":"d"}]},"candidate":{"version":"eval_continuity_comparison.v1","run_id":"r1","phase":"candidate","facts":[{"kind":"objective","owner":"runner","id":"o","digest":"d"}]},"expected":{"passed":true}}`)
	f, err := ParseContinuityFixtureJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := EvaluateContinuityFixture(f); err != nil {
		t.Fatal(err)
	}
	privacy := []byte(`{"version":"eval_continuity_comparison.v1","baseline":{"version":"eval_continuity_comparison.v1","run_id":"r1","phase":"baseline","facts":[{"kind":"objective","owner":"runner","id":"o","body":"secret"}]},"candidate":{"version":"eval_continuity_comparison.v1","run_id":"r1","phase":"candidate","facts":[{"kind":"objective","owner":"runner","id":"o"}]},"expected":{"passed":false}}`)
	if _, err := ParseContinuityFixtureJSON(privacy); err == nil || err.Error()[:len(ReasonContinuityPrivacyViolation)] != ReasonContinuityPrivacyViolation {
		t.Fatalf("got %v", err)
	}
}

func TestContinuityFixtureClassifiesDrift(t *testing.T) {
	raw := []byte(`{"version":"eval_continuity_comparison.v1","baseline":{"version":"eval_continuity_comparison.v1","run_id":"r1","phase":"baseline","facts":[{"kind":"objective","owner":"runner","id":"o","digest":"d1"}]},"candidate":{"version":"eval_continuity_comparison.v1","run_id":"r1","phase":"candidate","facts":[{"kind":"objective","owner":"runner","id":"o","digest":"d2"}]},"expected":{"passed":false,"drifts":["continuity_objective_drift"]}}`)
	f, err := ParseContinuityFixtureJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := EvaluateContinuityFixture(f); err != nil {
		t.Fatal(err)
	}
}
