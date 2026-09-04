package diagnosticsreplay

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
)

func TestSessionHistoryReplayCanonicalFixture(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "session_history_checkpoint_replay.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if _, err := EvaluateSessionHistoryFixtureJSON(raw); err != nil {
		t.Fatalf("canonical fixture: %v", err)
	}
}

func TestSessionHistoryReplayFixtureSuccessAndDeterminism(t *testing.T) {
	history := types.SessionHistoryBoundary{
		Version: types.SessionHistoryCheckpointReplayVersionV1, SessionID: "session-1",
		Root:    types.HistoryEntry{ID: "entry-0", Position: 0, Digest: "d0"},
		Entries: []types.HistoryEntry{{ID: "entry-0", Position: 0, Digest: "d0"}, {ID: "entry-1", ParentID: "entry-0", Position: 1, Digest: "d1"}}, LeafID: "entry-1",
	}
	branch := &types.BranchProjection{SessionID: "session-1", BranchID: "branch-1", ParentLeafID: "entry-1", ParentRunID: "run-1", RunID: "run-2"}
	fixture := SessionHistoryFixture{Version: SessionHistoryReplayFixtureV1, Cases: []SessionHistoryFixtureCase{{Name: "branch", Run: SessionHistoryObservation{SessionID: "session-1", RunID: "run-2", History: history, Branch: branch}, Stream: SessionHistoryObservation{SessionID: "session-1", RunID: "run-2", History: history, Branch: branch}, Expected: SessionHistoryObservation{SessionID: "session-1", RunID: "run-2", History: history, Branch: branch}, Idempotency: types.ReplayOperation{OperationID: "replay-1", FixtureVersion: SessionHistoryReplayFixtureV1, InputDigest: "digest-1", SideEffectFree: true}}}}
	raw, err := json.Marshal(fixture)
	if err != nil {
		t.Fatal(err)
	}
	first, err := EvaluateSessionHistoryFixtureJSON(raw)
	if err != nil {
		t.Fatalf("first replay: %v", err)
	}
	second, err := EvaluateSessionHistoryFixtureJSON(raw)
	if err != nil {
		t.Fatalf("second replay: %v", err)
	}
	if first.Cases[0].HistoryDigest != second.Cases[0].HistoryDigest || first.Cases[0].BranchID != "branch-1" {
		t.Fatalf("non-deterministic output: %#v %#v", first, second)
	}
}

func TestSessionHistoryReplayFixtureRejectsGapAndSideEffect(t *testing.T) {
	history := types.SessionHistoryBoundary{Version: types.SessionHistoryCheckpointReplayVersionV1, SessionID: "session-1", Root: types.HistoryEntry{ID: "root", Position: 0}, Entries: []types.HistoryEntry{{ID: "root", Position: 0}, {ID: "leaf", ParentID: "missing", Position: 1}}, LeafID: "leaf"}
	obs := SessionHistoryObservation{SessionID: "session-1", RunID: "run-1", History: history}
	fixture := SessionHistoryFixture{Version: SessionHistoryReplayFixtureV1, Cases: []SessionHistoryFixtureCase{{Name: "gap", Run: obs, Stream: obs, Expected: obs, Idempotency: types.ReplayOperation{OperationID: "r", FixtureVersion: SessionHistoryReplayFixtureV1, InputDigest: "d", SideEffectFree: true}}}}
	raw, _ := json.Marshal(fixture)
	if _, err := EvaluateSessionHistoryFixtureJSON(raw); err == nil || !strings.Contains(err.Error(), ReasonCodeSessionHistoryGap) {
		t.Fatalf("gap error = %v", err)
	}
	history.Entries[1].ParentID = "root"
	obs.History = history
	fixture.Cases[0].Run = obs
	fixture.Cases[0].Stream = obs
	fixture.Cases[0].Expected = obs
	fixture.Cases[0].Idempotency.SideEffectFree = false
	raw, _ = json.Marshal(fixture)
	if _, err := EvaluateSessionHistoryFixtureJSON(raw); err == nil || !strings.Contains(err.Error(), ReasonCodeSessionReplaySideEffect) {
		t.Fatalf("side effect error = %v", err)
	}
}
