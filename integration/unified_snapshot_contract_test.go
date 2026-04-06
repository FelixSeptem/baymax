package integration

import (
	"context"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/integration/fakes"
	"github.com/FelixSeptem/baymax/observability/event"
	"github.com/FelixSeptem/baymax/orchestration/composer"
	"github.com/FelixSeptem/baymax/orchestration/scheduler"
	orchestrationsnapshot "github.com/FelixSeptem/baymax/orchestration/snapshot"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
	runtimediag "github.com/FelixSeptem/baymax/runtime/diagnostics"
)

func TestUnifiedSnapshotRecoveryRunStreamEquivalenceAfterRestore(t *testing.T) {
	ctx := context.Background()
	runID := "run-unified-snapshot-restore-run-stream-equivalence"

	sourceMgr := newUnifiedSnapshotManager(t, runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX_UNIFIED_SNAPSHOT_INT_SOURCE_82"})
	source := newUnifiedSnapshotComposer(t, sourceMgr, nil)
	seedUnifiedSnapshotSourceState(t, source, runID)

	exported, err := source.ExportUnifiedSnapshot(ctx, composer.UnifiedSnapshotExportRequest{RunID: runID})
	if err != nil {
		t.Fatalf("export unified snapshot failed: %v", err)
	}

	runMgr := newUnifiedSnapshotManager(t, runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX_UNIFIED_SNAPSHOT_INT_RUN_82"})
	runComposer := newUnifiedSnapshotComposer(t, runMgr, nil)
	if _, err := runComposer.ImportUnifiedSnapshot(ctx, composer.UnifiedSnapshotImportRequest{
		Payload:      exported.Payload,
		OperationID:  "op-unified-snapshot-restore-run",
		RestoreMode:  orchestrationsnapshot.RestoreModeStrict,
		CompatWindow: 1,
	}); err != nil {
		t.Fatalf("run path import failed: %v", err)
	}
	if _, err := runComposer.Run(ctx, types.RunRequest{RunID: runID, Input: "resume-run"}, nil); err != nil {
		t.Fatalf("run path execution failed: %v", err)
	}
	runRecord := findRunRecord(t, runMgr.RecentRuns(10), runID)

	streamMgr := newUnifiedSnapshotManager(t, runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX_UNIFIED_SNAPSHOT_INT_STREAM_82"})
	streamComposer := newUnifiedSnapshotComposer(t, streamMgr, nil)
	if _, err := streamComposer.ImportUnifiedSnapshot(ctx, composer.UnifiedSnapshotImportRequest{
		Payload:      exported.Payload,
		OperationID:  "op-unified-snapshot-restore-stream",
		RestoreMode:  orchestrationsnapshot.RestoreModeStrict,
		CompatWindow: 1,
	}); err != nil {
		t.Fatalf("stream path import failed: %v", err)
	}
	if _, err := streamComposer.Stream(ctx, types.RunRequest{RunID: runID, Input: "resume-stream"}, nil); err != nil {
		t.Fatalf("stream path execution failed: %v", err)
	}
	streamRecord := findRunRecord(t, streamMgr.RecentRuns(10), runID)

	assertRunRecordRestoreEquivalence(t, runRecord, streamRecord)
}

func TestUnifiedSnapshotRestoreMemoryFileBackendParity(t *testing.T) {
	ctx := context.Background()
	runID := "run-unified-snapshot-restore-backend-parity"

	sourceMgr := newUnifiedSnapshotManager(t, runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX_UNIFIED_SNAPSHOT_INT_SOURCE_83"})
	source := newUnifiedSnapshotComposer(t, sourceMgr, nil)
	seedUnifiedSnapshotSourceState(t, source, runID)

	exported, err := source.ExportUnifiedSnapshot(ctx, composer.UnifiedSnapshotExportRequest{RunID: runID})
	if err != nil {
		t.Fatalf("export unified snapshot failed: %v", err)
	}

	memoryMgr := newUnifiedSnapshotManager(t, runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX_UNIFIED_SNAPSHOT_INT_MEMORY_83"})
	memoryComposer := newUnifiedSnapshotComposer(t, memoryMgr, scheduler.NewMemoryStore())
	if _, err := memoryComposer.ImportUnifiedSnapshot(ctx, composer.UnifiedSnapshotImportRequest{
		Payload:      exported.Payload,
		OperationID:  "op-unified-snapshot-memory-backend",
		RestoreMode:  orchestrationsnapshot.RestoreModeStrict,
		CompatWindow: 1,
	}); err != nil {
		t.Fatalf("memory backend import failed: %v", err)
	}

	fileMgr := newUnifiedSnapshotManager(t, runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX_UNIFIED_SNAPSHOT_INT_FILE_83"})
	fileStore, err := scheduler.NewFileStore(filepath.Join(t.TempDir(), "scheduler-state.json"))
	if err != nil {
		t.Fatalf("new file scheduler store failed: %v", err)
	}
	fileComposer := newUnifiedSnapshotComposer(t, fileMgr, fileStore)
	if _, err := fileComposer.ImportUnifiedSnapshot(ctx, composer.UnifiedSnapshotImportRequest{
		Payload:      exported.Payload,
		OperationID:  "op-unified-snapshot-file-backend",
		RestoreMode:  orchestrationsnapshot.RestoreModeStrict,
		CompatWindow: 1,
	}); err != nil {
		t.Fatalf("file backend import failed: %v", err)
	}

	memorySnapshot, err := memoryComposer.Scheduler().Snapshot(ctx)
	if err != nil {
		t.Fatalf("snapshot memory backend failed: %v", err)
	}
	fileSnapshot, err := fileComposer.Scheduler().Snapshot(ctx)
	if err != nil {
		t.Fatalf("snapshot file backend failed: %v", err)
	}
	assertSchedulerSnapshotSemanticParity(t, memorySnapshot, fileSnapshot)

	if _, err := memoryComposer.Run(ctx, types.RunRequest{RunID: runID, Input: "resume-memory"}, nil); err != nil {
		t.Fatalf("memory backend run failed: %v", err)
	}
	if _, err := fileComposer.Run(ctx, types.RunRequest{RunID: runID, Input: "resume-file"}, nil); err != nil {
		t.Fatalf("file backend run failed: %v", err)
	}
	memoryRecord := findRunRecord(t, memoryMgr.RecentRuns(10), runID)
	fileRecord := findRunRecord(t, fileMgr.RecentRuns(10), runID)
	assertRunRecordRestoreEquivalence(t, memoryRecord, fileRecord)
}

func TestUnifiedSnapshotDuplicateImportIdempotentNoSideEffect(t *testing.T) {
	ctx := context.Background()
	runID := "run-unified-snapshot-duplicate-import-idempotent"

	sourceMgr := newUnifiedSnapshotManager(t, runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX_UNIFIED_SNAPSHOT_INT_SOURCE_84"})
	source := newUnifiedSnapshotComposer(t, sourceMgr, nil)
	seedUnifiedSnapshotSourceState(t, source, runID)

	exported, err := source.ExportUnifiedSnapshot(ctx, composer.UnifiedSnapshotExportRequest{RunID: runID})
	if err != nil {
		t.Fatalf("export unified snapshot failed: %v", err)
	}

	targetMgr := newUnifiedSnapshotManager(t, runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX_UNIFIED_SNAPSHOT_INT_TARGET_84"})
	target := newUnifiedSnapshotComposer(t, targetMgr, scheduler.NewMemoryStore())

	first, err := target.ImportUnifiedSnapshot(ctx, composer.UnifiedSnapshotImportRequest{
		Payload:      exported.Payload,
		OperationID:  "op-unified-snapshot-idempotent-import",
		RestoreMode:  orchestrationsnapshot.RestoreModeStrict,
		CompatWindow: 1,
	})
	if err != nil {
		t.Fatalf("first import failed: %v", err)
	}
	if first.RestoreAction != orchestrationsnapshot.RestoreActionStrictExact {
		t.Fatalf("first restore action = %q, want %q", first.RestoreAction, orchestrationsnapshot.RestoreActionStrictExact)
	}
	statsAfterFirst, err := target.Scheduler().Stats(ctx)
	if err != nil {
		t.Fatalf("scheduler stats after first import failed: %v", err)
	}

	second, err := target.ImportUnifiedSnapshot(ctx, composer.UnifiedSnapshotImportRequest{
		Payload:      exported.Payload,
		OperationID:  "op-unified-snapshot-idempotent-import",
		RestoreMode:  orchestrationsnapshot.RestoreModeStrict,
		CompatWindow: 1,
	})
	if err != nil {
		t.Fatalf("second import failed: %v", err)
	}
	if second.RestoreAction != orchestrationsnapshot.RestoreActionIdempotentNoop {
		t.Fatalf("second restore action = %q, want %q", second.RestoreAction, orchestrationsnapshot.RestoreActionIdempotentNoop)
	}
	if len(second.AppliedSegments) != 0 {
		t.Fatalf("second import should not re-apply segments, got %#v", second.AppliedSegments)
	}
	statsAfterSecond, err := target.Scheduler().Stats(ctx)
	if err != nil {
		t.Fatalf("scheduler stats after second import failed: %v", err)
	}
	assertSchedulerStatsNoInflation(t, statsAfterFirst, statsAfterSecond)

	if _, err := target.Run(ctx, types.RunRequest{RunID: runID, Input: "resume-idempotent"}, nil); err != nil {
		t.Fatalf("run after duplicate import failed: %v", err)
	}
	record := findRunRecord(t, targetMgr.RecentRuns(10), runID)
	if !record.RecoveryRecovered {
		t.Fatalf("recovery_recovered = false, want true: %#v", record)
	}
	if record.RecoveryReplayTotal != 1 {
		t.Fatalf("recovery_replay_total = %d, want 1", record.RecoveryReplayTotal)
	}
}

func newUnifiedSnapshotManager(t *testing.T, opts runtimeconfig.ManagerOptions) *runtimeconfig.Manager {
	t.Helper()
	mgr, err := runtimeconfig.NewManager(opts)
	if err != nil {
		t.Fatalf("new runtime manager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	return mgr
}

func newUnifiedSnapshotComposer(t *testing.T, mgr *runtimeconfig.Manager, store scheduler.QueueStore) *composer.Composer {
	t.Helper()
	model := fakes.NewModel([]fakes.ModelStep{
		{Response: types.ModelResponse{FinalAnswer: "ok"}},
	})
	model.SetStream([]types.ModelEvent{
		{Type: types.ModelEventTypeFinalAnswer, TextDelta: "ok"},
	}, nil)
	dispatcher := event.NewDispatcher(event.NewRuntimeRecorder(mgr))

	builder := composer.NewBuilder(model).
		WithRuntimeManager(mgr).
		WithEventHandler(dispatcherHandler{dispatcher: dispatcher})
	if store != nil {
		builder = builder.WithSchedulerStore(store)
	}
	comp, err := builder.Build()
	if err != nil {
		t.Fatalf("new composer failed: %v", err)
	}
	return comp
}

func seedUnifiedSnapshotSourceState(t *testing.T, comp *composer.Composer, runID string) {
	t.Helper()
	ctx := context.Background()
	_, err := comp.DispatchChild(ctx, composer.ChildDispatchRequest{
		Task: scheduler.Task{
			TaskID: runID + "-task-success",
			RunID:  runID,
		},
		Target:               composer.ChildTargetLocal,
		ParentDepth:          0,
		ParentActiveChildren: 0,
		ChildTimeout:         500 * time.Millisecond,
		LocalRunner: composer.LocalChildRunnerFunc(func(context.Context, scheduler.Task) (map[string]any, error) {
			return map[string]any{"ok": true}, nil
		}),
	})
	if err != nil {
		t.Fatalf("seed dispatch child failed: %v", err)
	}
	if _, err := comp.SpawnChild(ctx, composer.ChildDispatchRequest{
		Task: scheduler.Task{
			TaskID: runID + "-task-queued",
			RunID:  runID,
		},
	}); err != nil {
		t.Fatalf("seed queued child failed: %v", err)
	}
}

func assertRunRecordRestoreEquivalence(t *testing.T, left, right runtimediag.RunRecord) {
	t.Helper()
	if left.Status != right.Status {
		t.Fatalf("terminal classification mismatch: left=%q right=%q", left.Status, right.Status)
	}
	if left.RecoveryRecovered != right.RecoveryRecovered ||
		left.RecoveryReplayTotal != right.RecoveryReplayTotal ||
		left.RecoveryConflict != right.RecoveryConflict ||
		strings.TrimSpace(left.RecoveryConflictCode) != strings.TrimSpace(right.RecoveryConflictCode) {
		t.Fatalf("recovery aggregates mismatch: left=%#v right=%#v", left, right)
	}
	if left.SchedulerQueueTotal != right.SchedulerQueueTotal ||
		left.SchedulerClaimTotal != right.SchedulerClaimTotal ||
		left.SchedulerReclaimTotal != right.SchedulerReclaimTotal {
		t.Fatalf("scheduler aggregates mismatch: left=%#v right=%#v", left, right)
	}
}

func assertSchedulerSnapshotSemanticParity(t *testing.T, left, right scheduler.StoreSnapshot) {
	t.Helper()
	leftTasks := make(map[string]schedulerTaskSemantic, len(left.Tasks))
	for i := range left.Tasks {
		item := left.Tasks[i]
		leftTasks[strings.TrimSpace(item.Task.TaskID)] = schedulerTaskSemantic{
			State:          item.State,
			RunID:          strings.TrimSpace(item.Task.RunID),
			CurrentAttempt: strings.TrimSpace(item.CurrentAttempt),
			AttemptTotal:   len(item.Attempts),
		}
	}
	rightTasks := make(map[string]schedulerTaskSemantic, len(right.Tasks))
	for i := range right.Tasks {
		item := right.Tasks[i]
		rightTasks[strings.TrimSpace(item.Task.TaskID)] = schedulerTaskSemantic{
			State:          item.State,
			RunID:          strings.TrimSpace(item.Task.RunID),
			CurrentAttempt: strings.TrimSpace(item.CurrentAttempt),
			AttemptTotal:   len(item.Attempts),
		}
	}
	if !reflect.DeepEqual(leftTasks, rightTasks) {
		t.Fatalf("task semantic mismatch after restore: left=%#v right=%#v", leftTasks, rightTasks)
	}

	leftQueue := make(map[string]struct{}, len(left.Queue))
	for i := range left.Queue {
		leftQueue[strings.TrimSpace(left.Queue[i])] = struct{}{}
	}
	rightQueue := make(map[string]struct{}, len(right.Queue))
	for i := range right.Queue {
		rightQueue[strings.TrimSpace(right.Queue[i])] = struct{}{}
	}
	if !reflect.DeepEqual(leftQueue, rightQueue) {
		t.Fatalf("queue semantic mismatch after restore: left=%#v right=%#v", leftQueue, rightQueue)
	}

	leftCommits := make(map[string]scheduler.TaskState, len(left.TerminalCommits))
	for i := range left.TerminalCommits {
		item := left.TerminalCommits[i]
		key := strings.TrimSpace(item.TaskID) + "|" + strings.TrimSpace(item.AttemptID)
		leftCommits[key] = item.Status
	}
	rightCommits := make(map[string]scheduler.TaskState, len(right.TerminalCommits))
	for i := range right.TerminalCommits {
		item := right.TerminalCommits[i]
		key := strings.TrimSpace(item.TaskID) + "|" + strings.TrimSpace(item.AttemptID)
		rightCommits[key] = item.Status
	}
	if !reflect.DeepEqual(leftCommits, rightCommits) {
		t.Fatalf("terminal commit semantic mismatch after restore: left=%#v right=%#v", leftCommits, rightCommits)
	}
}

func assertSchedulerStatsNoInflation(t *testing.T, first, second scheduler.Stats) {
	t.Helper()
	if first.QueueTotal != second.QueueTotal ||
		first.ClaimTotal != second.ClaimTotal ||
		first.ReclaimTotal != second.ReclaimTotal ||
		first.CompleteTotal != second.CompleteTotal ||
		first.FailTotal != second.FailTotal ||
		first.DuplicateTerminalCommitTotal != second.DuplicateTerminalCommitTotal {
		t.Fatalf("duplicate import should be no-side-effect: first=%#v second=%#v", first, second)
	}
}

type schedulerTaskSemantic struct {
	State          scheduler.TaskState
	RunID          string
	CurrentAttempt string
	AttemptTotal   int
}
