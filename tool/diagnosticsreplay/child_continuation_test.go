package diagnosticsreplay

import (
	"encoding/json"
	"testing"
)

func TestChildContinuationReplayIsReadOnlyAndRejectsDrift(t *testing.T) {
	fixture := ChildContinuationReplayFixture{Version: ChildContinuationReplayVersion, Cases: []ChildContinuationReplayCase{
		{CaseID: "persisted", Form: "child", PauseReason: "input_required", Window: "persisted", SourceState: "persisted", CheckpointID: "cp", GraphID: "graph", GraphDigest: "digest", SourceNamespace: "child", RunID: "run", SessionID: "session", RunOutcome: "paused", StreamOutcome: "paused"},
		{CaseID: "unsupported-team", Form: "team", PauseReason: "approval_required", Window: "unsupported", RunOutcome: "unsupported_capability", StreamOutcome: "unsupported_capability", ExpectedReason: "child_continuation.unsupported_form"},
	}}
	raw, err := json.Marshal(fixture)
	if err != nil {
		t.Fatal(err)
	}
	results, err := ReplayChildContinuationFixtureJSON(raw)
	if err != nil || len(results) != 2 {
		t.Fatalf("replay = %#v, %v", results, err)
	}
	fixture.Cases[0].SourceNamespace = "foreign"
	raw, _ = json.Marshal(fixture)
	if _, err := ReplayChildContinuationFixtureJSON(raw); err == nil {
		t.Fatal("graph drift unexpectedly accepted")
	}
}
