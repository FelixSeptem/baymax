package types

import (
	"strings"
	"testing"
)

func TestSessionHistoryBoundaryValidatesContinuousChain(t *testing.T) {
	in := SessionHistoryBoundary{
		Version:   "session_history_checkpoint_replay.v1",
		SessionID: "session-1",
		Root:      HistoryEntry{ID: "entry-0", Position: 0, Digest: "d0"},
		Entries: []HistoryEntry{
			{ID: "entry-0", Position: 0, Digest: "d0"},
			{ID: "entry-1", ParentID: "entry-0", Position: 1, Digest: "d1", ProducedByRunID: "run-1", ProducedByStepID: "step-1"},
		},
		LeafID: "entry-1",
	}

	if err := in.Validate(HistoryValidationLimits{MaxEntries: 4, MaxSerializedBytes: 4096}); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if got := in.NormalizedDigest(); got == "" {
		t.Fatal("NormalizedDigest() must not be empty")
	}
}

func TestSessionHistoryBoundaryRejectsGapAndConflicts(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*SessionHistoryBoundary)
		want   string
	}{
		{name: "gap", mutate: func(v *SessionHistoryBoundary) { v.Entries[1].ParentID = "missing" }, want: "session.history_gap"},
		{name: "position", mutate: func(v *SessionHistoryBoundary) { v.Entries[1].Position = 3 }, want: "session.history_gap"},
		{name: "conflict", mutate: func(v *SessionHistoryBoundary) {
			v.Entries = append(v.Entries, HistoryEntry{ID: "entry-1", ParentID: "entry-0", Position: 1})
		}, want: "session.history_conflict"},
	}
	base := SessionHistoryBoundary{
		Version:   "session_history_checkpoint_replay.v1",
		SessionID: "session-1",
		Root:      HistoryEntry{ID: "entry-0", Position: 0, Digest: "d0"},
		Entries: []HistoryEntry{
			{ID: "entry-0", Position: 0, Digest: "d0"},
			{ID: "entry-1", ParentID: "entry-0", Position: 1, Digest: "d1"},
		},
		LeafID: "entry-1",
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := base
			in.Entries = append([]HistoryEntry(nil), base.Entries...)
			tc.mutate(&in)
			err := in.Validate(HistoryValidationLimits{MaxEntries: 8, MaxSerializedBytes: 4096})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Validate() error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestSessionHistoryBranchRequiresParentAndUsesDistinctRun(t *testing.T) {
	valid := BranchProjection{SessionID: "session-1", BranchID: "branch-1", ParentLeafID: "entry-1", ParentRunID: "run-1", RunID: "run-2"}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid branch Validate() error = %v", err)
	}
	invalid := valid
	invalid.ParentLeafID = ""
	if err := invalid.Validate(); err == nil || !strings.Contains(err.Error(), "session.branch_parent_missing") {
		t.Fatalf("missing parent error = %v", err)
	}
	invalid = valid
	invalid.RunID = invalid.ParentRunID
	if err := invalid.Validate(); err == nil || !strings.Contains(err.Error(), "session.branch_run_conflict") {
		t.Fatalf("same run error = %v", err)
	}
}

func TestRestoreAndReplayOperationIdentityValidation(t *testing.T) {
	restore := RestoreOperation{OperationID: "restore-1", Mode: RestoreModeStrict, SessionID: "session-1", HistoryLeafID: "entry-1", CheckpointID: "checkpoint-1", InputDigest: "digest-1"}
	if err := restore.Validate(); err != nil {
		t.Fatalf("restore Validate() error = %v", err)
	}
	replay := ReplayOperation{OperationID: "replay-1", FixtureVersion: "session_history_checkpoint_replay.v1", InputDigest: "digest-1", SideEffectFree: true}
	if err := replay.Validate(); err != nil {
		t.Fatalf("replay Validate() error = %v", err)
	}
	replay.SideEffectFree = false
	if err := replay.Validate(); err == nil || !strings.Contains(err.Error(), "session.replay_side_effect") {
		t.Fatalf("side effect error = %v", err)
	}
}

func TestSessionHistoryBoundaryRejectsOversizedProjection(t *testing.T) {
	in := SessionHistoryBoundary{Version: "session_history_checkpoint_replay.v1", SessionID: "session-1", Root: HistoryEntry{ID: "root"}, Entries: []HistoryEntry{{ID: "root"}}, LeafID: "root", Metadata: map[string]string{"large": strings.Repeat("x", 128)}}
	if err := in.Validate(HistoryValidationLimits{MaxEntries: 4, MaxSerializedBytes: 32}); err == nil || !strings.Contains(err.Error(), "session.history_size_limit") {
		t.Fatalf("size error = %v", err)
	}
}
