package composer

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/orchestration/scheduler"
)

func TestMemoryRecoveryStoreRoundTripAndDuplicateLoad(t *testing.T) {
	store := NewMemoryRecoveryStore()
	snapshot := testRecoverySnapshot("run-recovery-memory")
	if err := store.Save(context.Background(), snapshot); err != nil {
		t.Fatalf("save snapshot: %v", err)
	}
	first, found, err := store.Load(context.Background(), snapshot.Run.RunID)
	if err != nil || !found {
		t.Fatalf("load snapshot #1: found=%v err=%v", found, err)
	}
	second, found, err := store.Load(context.Background(), snapshot.Run.RunID)
	if err != nil || !found {
		t.Fatalf("load snapshot #2: found=%v err=%v", found, err)
	}
	if first.Run.RunID != second.Run.RunID || first.Replay.TerminalCommitCount != second.Replay.TerminalCommitCount {
		t.Fatalf("duplicate load mismatch: first=%#v second=%#v", first, second)
	}
}

func TestFileRecoveryStoreRoundTripAndDuplicateLoad(t *testing.T) {
	store, err := NewFileRecoveryStore(t.TempDir())
	if err != nil {
		t.Fatalf("new file recovery store: %v", err)
	}
	snapshot := testRecoverySnapshot("run-recovery-file")
	if err := store.Save(context.Background(), snapshot); err != nil {
		t.Fatalf("save snapshot: %v", err)
	}
	first, found, err := store.Load(context.Background(), snapshot.Run.RunID)
	if err != nil || !found {
		t.Fatalf("load snapshot #1: found=%v err=%v", found, err)
	}
	second, found, err := store.Load(context.Background(), snapshot.Run.RunID)
	if err != nil || !found {
		t.Fatalf("load snapshot #2: found=%v err=%v", found, err)
	}
	if first.Run.RunID != second.Run.RunID || first.Replay.TerminalCommitCount != second.Replay.TerminalCommitCount {
		t.Fatalf("duplicate load mismatch: first=%#v second=%#v", first, second)
	}
}

func TestFileRecoveryStoreCorruptSnapshotFailsFast(t *testing.T) {
	store, err := NewFileRecoveryStore(t.TempDir())
	if err != nil {
		t.Fatalf("new file recovery store: %v", err)
	}
	runID := "run-recovery-corrupt"
	if err := os.WriteFile(store.filePath(runID), []byte(`{"version":"a9.v1","run":{"run_id":"`+runID+`"}`), 0o600); err != nil {
		t.Fatalf("write corrupt snapshot: %v", err)
	}
	_, _, loadErr := store.Load(context.Background(), runID)
	if loadErr == nil {
		t.Fatal("expected load error for corrupt snapshot")
	}
	var recoveryErr *RecoveryError
	if !errors.As(loadErr, &recoveryErr) {
		t.Fatalf("expected RecoveryError, got %T (%v)", loadErr, loadErr)
	}
	if !IsRecoveryErrorCode(loadErr, RecoveryErrorSnapshotCorrupt) {
		t.Fatalf("expected RecoveryErrorSnapshotCorrupt, got %v", loadErr)
	}
}

func testRecoverySnapshot(runID string) RecoverySnapshot {
	now := time.Now()
	taskID := "task-" + runID
	attemptID := taskID + "-attempt-1"
	return RecoverySnapshot{
		Version:   RecoverySnapshotVersion,
		UpdatedAt: now,
		Run: RecoveryRunSnapshot{
			RunID: runID,
		},
		Scheduler: scheduler.StoreSnapshot{
			Backend: "memory",
			Tasks: []scheduler.TaskRecord{
				{
					Task: scheduler.Task{
						TaskID: taskID,
						RunID:  runID,
					},
					State: scheduler.TaskStateSucceeded,
					Attempts: []scheduler.Attempt{
						{
							AttemptID:  attemptID,
							Attempt:    1,
							Status:     scheduler.AttemptStatusSucceeded,
							StartedAt:  now.Add(-2 * time.Second),
							TerminalAt: now.Add(-1 * time.Second),
						},
					},
					CurrentAttempt: "",
					Result:         map[string]any{"ok": true},
					CreatedAt:      now.Add(-2 * time.Second),
					UpdatedAt:      now,
				},
			},
			TerminalCommits: []scheduler.TerminalCommit{
				{
					TaskID:      taskID,
					AttemptID:   attemptID,
					Status:      scheduler.TaskStateSucceeded,
					Result:      map[string]any{"ok": true},
					CommittedAt: now,
				},
			},
			Stats: scheduler.Stats{
				Backend:       "memory",
				QueueTotal:    1,
				ClaimTotal:    1,
				CompleteTotal: 1,
			},
		},
		A2A: RecoveryA2ASnapshot{
			InFlight: nil,
		},
		Replay: RecoveryReplayCursor{
			Sequence:            now.UnixNano(),
			TerminalCommitCount: 1,
		},
		ConflictPolicy: "fail_fast",
	}
}
