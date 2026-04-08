package composer

import (
	"context"
	"errors"
	"os"
	"path/filepath"
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

func TestFileRecoveryStoreOptionalGroupCommitFlushAndDebounce(t *testing.T) {
	root := t.TempDir()
	store, err := NewFileRecoveryStore(root, WithRecoveryPersistBatchSize(2), WithRecoveryPersistDebounce(time.Hour))
	if err != nil {
		t.Fatalf("new file recovery store: %v", err)
	}

	snapA := testRecoverySnapshot("run-recovery-group-a")
	snapB := testRecoverySnapshot("run-recovery-group-b")
	if err := store.Save(context.Background(), snapA); err != nil {
		t.Fatalf("save snapshot A: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "run-recovery-group-a.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("snapshot A should not be persisted before batch threshold: err=%v", err)
	}
	if pending, found, err := store.Load(context.Background(), snapA.Run.RunID); err != nil || !found {
		t.Fatalf("pending load for snapshot A failed: found=%v err=%v", found, err)
	} else if pending.Run.RunID != snapA.Run.RunID {
		t.Fatalf("pending snapshot run_id mismatch: got=%q want=%q", pending.Run.RunID, snapA.Run.RunID)
	}
	if err := store.Save(context.Background(), snapB); err != nil {
		t.Fatalf("save snapshot B: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "run-recovery-group-a.json")); err != nil {
		t.Fatalf("snapshot A should be persisted after group commit: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "run-recovery-group-b.json")); err != nil {
		t.Fatalf("snapshot B should be persisted after group commit: %v", err)
	}

	flushRoot := t.TempDir()
	flushStore, err := NewFileRecoveryStore(flushRoot, WithRecoveryPersistBatchSize(5), WithRecoveryPersistDebounce(time.Hour))
	if err != nil {
		t.Fatalf("new flush recovery store: %v", err)
	}
	flushSnapshot := testRecoverySnapshot("run-recovery-flush")
	if err := flushStore.Save(context.Background(), flushSnapshot); err != nil {
		t.Fatalf("save flush snapshot: %v", err)
	}
	flushFile := filepath.Join(flushRoot, "run-recovery-flush.json")
	if _, err := os.Stat(flushFile); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("flush snapshot should be pending before explicit flush: err=%v", err)
	}
	if err := flushStore.Flush(); err != nil {
		t.Fatalf("flush recovery store: %v", err)
	}
	if _, err := os.Stat(flushFile); err != nil {
		t.Fatalf("flush snapshot should be persisted after explicit flush: %v", err)
	}

	debounceRoot := t.TempDir()
	debounceStore, err := NewFileRecoveryStore(
		debounceRoot,
		WithRecoveryPersistBatchSize(10),
		WithRecoveryPersistDebounce(25*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("new debounce recovery store: %v", err)
	}
	snapDebounceA := testRecoverySnapshot("run-recovery-debounce-a")
	snapDebounceB := testRecoverySnapshot("run-recovery-debounce-b")
	if err := debounceStore.Save(context.Background(), snapDebounceA); err != nil {
		t.Fatalf("save debounce snapshot A: %v", err)
	}
	time.Sleep(35 * time.Millisecond)
	if err := debounceStore.Save(context.Background(), snapDebounceB); err != nil {
		t.Fatalf("save debounce snapshot B: %v", err)
	}
	if _, err := os.Stat(filepath.Join(debounceRoot, "run-recovery-debounce-a.json")); err != nil {
		t.Fatalf("debounce snapshot A should be persisted after debounce-triggered save: %v", err)
	}
	if _, err := os.Stat(filepath.Join(debounceRoot, "run-recovery-debounce-b.json")); err != nil {
		t.Fatalf("debounce snapshot B should be persisted after debounce-triggered save: %v", err)
	}
}

func TestFileRecoveryStoreFlushBoundaryCrashRecoveryConsistency(t *testing.T) {
	root := t.TempDir()
	store, err := NewFileRecoveryStore(root, WithRecoveryPersistBatchSize(10), WithRecoveryPersistDebounce(time.Hour))
	if err != nil {
		t.Fatalf("new file recovery store: %v", err)
	}
	snapDurable := testRecoverySnapshot("run-recovery-durable")
	snapPending := testRecoverySnapshot("run-recovery-pending")

	if err := store.Save(context.Background(), snapDurable); err != nil {
		t.Fatalf("save durable snapshot: %v", err)
	}
	if err := store.Flush(); err != nil {
		t.Fatalf("flush durable snapshot: %v", err)
	}
	if err := store.Save(context.Background(), snapPending); err != nil {
		t.Fatalf("save pending snapshot: %v", err)
	}

	restarted, err := NewFileRecoveryStore(root)
	if err != nil {
		t.Fatalf("reopen recovery store: %v", err)
	}
	if _, found, err := restarted.Load(context.Background(), snapDurable.Run.RunID); err != nil || !found {
		t.Fatalf("durable snapshot should survive restart: found=%v err=%v", found, err)
	}
	if _, found, err := restarted.Load(context.Background(), snapPending.Run.RunID); err != nil || found {
		t.Fatalf("pending snapshot should not survive restart before next flush: found=%v err=%v", found, err)
	}
}

func TestNormalizeRecoverySnapshotInteractionState(t *testing.T) {
	snapshot := testRecoverySnapshot("run-recovery-interaction-normalize")
	snapshot.Interaction = RecoveryInteractionState{
		Realtime: RecoveryRealtimeInteractionState{
			SessionID:      " session-interaction ",
			ResumeCursor:   " cursor-interaction ",
			EventSeqMax:    -1,
			InterruptTotal: -2,
			ResumeTotal:    -3,
			ResumeSource:   " CURSOR ",
		},
		IsolateHandoff: RecoveryIsolateHandoffState{
			Detected:         false,
			Stage2ReasonCode: " isolate_handoff_rejected ",
			Stage2Reason:     " isolate_handoff_rejected ",
			Stage2SkipReason: " stage2.isolate_handoff.empty ",
		},
	}
	normalized, err := normalizeRecoverySnapshot(snapshot, snapshot.Run.RunID)
	if err != nil {
		t.Fatalf("normalizeRecoverySnapshot failed: %v", err)
	}
	if normalized.Interaction.Realtime.SessionID != "session-interaction" ||
		normalized.Interaction.Realtime.ResumeCursor != "cursor-interaction" ||
		normalized.Interaction.Realtime.EventSeqMax != 0 ||
		normalized.Interaction.Realtime.InterruptTotal != 0 ||
		normalized.Interaction.Realtime.ResumeTotal != 0 ||
		normalized.Interaction.Realtime.ResumeSource != "cursor" {
		t.Fatalf("normalized realtime interaction mismatch: %#v", normalized.Interaction.Realtime)
	}
	if !normalized.Interaction.IsolateHandoff.Detected ||
		normalized.Interaction.IsolateHandoff.Stage2ReasonCode != "isolate_handoff_rejected" ||
		normalized.Interaction.IsolateHandoff.Stage2SkipReason != "stage2.isolate_handoff.empty" {
		t.Fatalf("normalized isolate-handoff interaction mismatch: %#v", normalized.Interaction.IsolateHandoff)
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
