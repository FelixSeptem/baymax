package composer

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/orchestration/scheduler"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

type completionRecoveryRunner struct {
	Runner
	snapshot runner.CompletionReferenceSnapshot
	captured string
	restored int
}

func (r *completionRecoveryRunner) SnapshotCompletionReferences(runID string) (runner.CompletionReferenceSnapshot, bool) {
	r.captured = runID
	return r.snapshot, true
}

func (r *completionRecoveryRunner) RestoreCompletionReferences(snapshot runner.CompletionReferenceSnapshot) error {
	r.restored += len(snapshot.Pending)
	return nil
}

func TestRecoverySnapshotPersistsAndRestoresCompletionReferences(t *testing.T) {
	base := &completionRecoveryRunner{snapshot: runner.CompletionReferenceSnapshot{Pending: []types.CompletionReference{{MessageID: "msg-1", IdempotencyKey: "idem-1", SessionID: "session-1", RunID: "run-completion-recovery"}}}}
	store := scheduler.NewMemoryStore()
	comp := &Composer{runner: base, now: time.Now, recoveryConflictPolicy: runtimeconfig.RecoveryConflictPolicyFailFast, scheduler: mustScheduler(t, store)}
	if _, err := comp.scheduler.Enqueue(context.Background(), scheduler.Task{TaskID: "task-completion-recovery", RunID: "run-completion-recovery"}); err != nil {
		t.Fatal(err)
	}
	snapshot, err := comp.CaptureRecoverySnapshot(context.Background(), "run-completion-recovery", "")
	if err != nil {
		t.Fatal(err)
	}
	if base.captured != "run-completion-recovery" || len(snapshot.CompletionReferences.Pending) != 1 {
		t.Fatalf("completion refs were not captured: captured=%q snapshot=%#v", base.captured, snapshot.CompletionReferences)
	}
	if err := comp.restoreCompletionReferences(snapshot); err != nil {
		t.Fatal(err)
	}
	if base.restored != 1 {
		t.Fatalf("restored=%d, want 1", base.restored)
	}
}

func TestNormalizeRecoverySnapshotRejectsCompletionReferenceDrift(t *testing.T) {
	snapshot := testRecoverySnapshot("run-completion-reference-drift")
	snapshot.CompletionReferences.Pending = []types.CompletionReference{{MessageID: "msg-1", IdempotencyKey: "idem-1", SessionID: "session-1", RunID: "other-run"}}
	if _, err := normalizeRecoverySnapshot(snapshot, snapshot.Run.RunID); err == nil {
		t.Fatal("expected completion reference run association mismatch")
	}
}

func mustScheduler(t *testing.T, store scheduler.QueueStore) *scheduler.Scheduler {
	t.Helper()
	s, err := scheduler.New(store)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

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

func TestRecoveryErrorProjectsToRecoveryConflictWithSourcePhase(t *testing.T) {
	pre, err := RecoveryTerminalOutcome("run-recovery", newRecoveryError(RecoveryErrorConflict, "snapshot mismatch", nil), types.ExecutionPhasePreExecution)
	if err != nil {
		t.Fatal(err)
	}
	if pre.FailureFamily != types.FailureFamilyRecoveryConflict || pre.Phase != types.ExecutionPhasePreExecution || pre.State != types.RunStateFailed {
		t.Fatalf("pre outcome = %#v", pre)
	}
	post, err := RecoveryTerminalOutcome("run-recovery", newRecoveryError(RecoveryErrorConflict, "late report", nil), types.ExecutionPhasePostStart)
	if err != nil {
		t.Fatal(err)
	}
	if post.FailureFamily != types.FailureFamilyRecoveryConflict || post.Phase != types.ExecutionPhasePostStart {
		t.Fatalf("post outcome = %#v", post)
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

func TestNormalizeRecoverySnapshotRejectsWorkspaceAssociationMismatch(t *testing.T) {
	snapshot := testRecoverySnapshot("run-recovery-workspace-checkpoint")
	snapshot.Scheduler.Tasks[0].Task.WorkspaceProvenance = &types.WorkspaceProvenance{
		WorkspaceID: "ws", ChangeSetID: "cs", BeforeIntegrity: "before", AfterIntegrity: "after",
		ProducedByRunID: "run-other", ProducedByStepID: "step",
	}
	snapshot.Scheduler.Tasks[0].Attempts[0].WorkspaceProvenance = snapshot.Scheduler.Tasks[0].Task.WorkspaceProvenance
	if _, err := normalizeRecoverySnapshot(snapshot, snapshot.Run.RunID); err == nil {
		t.Fatal("expected workspace association mismatch")
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
