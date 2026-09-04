package snapshot

import (
	"strings"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

func TestImporterValidatesHistoryAndCheckpointBeforeRestore(t *testing.T) {
	manifest, err := ExportManifest(ExportRequest{
		ExportedAt:           time.Unix(1_700_000_000, 0).UTC(),
		Source:               Source{Component: "composer", RunID: "run-1", SessionID: "session-1"},
		RunnerSessionPayload: map[string]any{"run_id": "run-1"}, SchedulerMailboxPayload: map[string]any{"records": []any{}}, ComposerRecoveryPayload: map[string]any{"sequence": 1}, MemoryPayload: map[string]any{"entries": []any{}},
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := MarshalManifest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	history := &types.SessionHistoryBoundary{Version: types.SessionHistoryCheckpointReplayVersionV1, SessionID: "session-1", Root: types.HistoryEntry{ID: "root", Position: 0}, Entries: []types.HistoryEntry{{ID: "root", Position: 0}}, LeafID: "root"}
	checkpoint := &types.CheckpointRef{CheckpointID: manifest.Digest, SchemaVersion: manifest.SchemaVersion, SourceComponent: manifest.Source.Component, Digest: manifest.Digest, SessionID: "session-1"}
	if _, err := NewImporter().Import(ImportRequest{Payload: raw, RestoreMode: RestoreModeStrict, OperationID: "restore-1", History: history, Checkpoint: checkpoint}); err != nil {
		t.Fatalf("valid contextual restore: %v", err)
	}
	history.SessionID = "other-session"
	_, err = NewImporter().Import(ImportRequest{Payload: raw, RestoreMode: RestoreModeStrict, OperationID: "restore-2", History: history, Checkpoint: checkpoint})
	if err == nil || !strings.Contains(err.Error(), types.SessionHistoryReasonCheckpointAssociation) {
		t.Fatalf("mismatch error = %v", err)
	}
}
