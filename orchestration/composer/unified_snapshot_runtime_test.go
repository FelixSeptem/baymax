package composer

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/integration/fakes"
	"github.com/FelixSeptem/baymax/memory"
	"github.com/FelixSeptem/baymax/orchestration/scheduler"
	orchestrationsnapshot "github.com/FelixSeptem/baymax/orchestration/snapshot"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
	runtimediag "github.com/FelixSeptem/baymax/runtime/diagnostics"
)

func TestComposerUnifiedSnapshotImportStrictConflictFailFast(t *testing.T) {
	source := newUnifiedSnapshotTestComposer(t)
	runID := "run-unified-snapshot-strict-conflict"
	if _, err := source.DispatchChild(context.Background(), ChildDispatchRequest{
		Task: scheduler.Task{
			TaskID: "task-unified-snapshot-source",
			RunID:  runID,
		},
		Target:               ChildTargetLocal,
		ParentDepth:          0,
		ParentActiveChildren: 0,
		ChildTimeout:         500 * time.Millisecond,
		LocalRunner: LocalChildRunnerFunc(func(ctx context.Context, task scheduler.Task) (map[string]any, error) {
			return map[string]any{"ok": true}, nil
		}),
	}); err != nil {
		t.Fatalf("source dispatch child failed: %v", err)
	}
	exported, err := source.ExportUnifiedSnapshot(context.Background(), UnifiedSnapshotExportRequest{
		RunID: runID,
	})
	if err != nil {
		t.Fatalf("export unified snapshot failed: %v", err)
	}

	target := newUnifiedSnapshotTestComposer(t)
	if _, err := target.Scheduler().Enqueue(context.Background(), scheduler.Task{
		TaskID: "task-unified-snapshot-target-existing",
		RunID:  runID,
	}); err != nil {
		t.Fatalf("preload conflicting scheduler state failed: %v", err)
	}

	_, err = target.ImportUnifiedSnapshot(context.Background(), UnifiedSnapshotImportRequest{
		Payload:      exported.Payload,
		OperationID:  "op-unified-snapshot-strict-conflict",
		RestoreMode:  orchestrationsnapshot.RestoreModeStrict,
		CompatWindow: 1,
	})
	if err == nil {
		t.Fatal("strict import should fail on scheduler conflict")
	}
	var recoveryErr *RecoveryError
	if !errors.As(err, &recoveryErr) {
		t.Fatalf("expected RecoveryError, got %T (%v)", err, err)
	}
	if recoveryErr.Code != RecoveryErrorConflict {
		t.Fatalf("recovery error code = %q, want %q", recoveryErr.Code, RecoveryErrorConflict)
	}
}

func TestComposerUnifiedSnapshotImportCompatibleConflictHasBoundedAction(t *testing.T) {
	source := newUnifiedSnapshotTestComposer(t)
	runID := "run-unified-snapshot-compatible-conflict"
	if _, err := source.DispatchChild(context.Background(), ChildDispatchRequest{
		Task: scheduler.Task{
			TaskID: "task-unified-snapshot-compatible-source",
			RunID:  runID,
		},
		Target:               ChildTargetLocal,
		ParentDepth:          0,
		ParentActiveChildren: 0,
		ChildTimeout:         500 * time.Millisecond,
		LocalRunner: LocalChildRunnerFunc(func(ctx context.Context, task scheduler.Task) (map[string]any, error) {
			return map[string]any{"ok": true}, nil
		}),
	}); err != nil {
		t.Fatalf("source dispatch child failed: %v", err)
	}
	exported, err := source.ExportUnifiedSnapshot(context.Background(), UnifiedSnapshotExportRequest{
		RunID: runID,
	})
	if err != nil {
		t.Fatalf("export unified snapshot failed: %v", err)
	}

	target := newUnifiedSnapshotTestComposer(t)
	existingTaskID := "task-unified-snapshot-compatible-target-existing"
	if _, err := target.Scheduler().Enqueue(context.Background(), scheduler.Task{
		TaskID: existingTaskID,
		RunID:  runID,
	}); err != nil {
		t.Fatalf("preload conflicting scheduler state failed: %v", err)
	}

	result, err := target.ImportUnifiedSnapshot(context.Background(), UnifiedSnapshotImportRequest{
		Payload:      exported.Payload,
		OperationID:  "op-unified-snapshot-compatible-conflict",
		RestoreMode:  orchestrationsnapshot.RestoreModeCompatible,
		CompatWindow: 1,
	})
	if err != nil {
		t.Fatalf("compatible import should not fail on bounded conflict, got %v", err)
	}
	if result.RestoreAction != orchestrationsnapshot.RestoreActionCompatibleBounded {
		t.Fatalf("restore action = %q, want %q", result.RestoreAction, orchestrationsnapshot.RestoreActionCompatibleBounded)
	}
	if len(result.SkippedSegments) == 0 {
		t.Fatalf("compatible bounded restore should skip conflicting segments, got %#v", result)
	}

	if _, found, err := target.Scheduler().Get(context.Background(), existingTaskID); err != nil {
		t.Fatalf("query pre-existing scheduler task failed: %v", err)
	} else if !found {
		t.Fatalf("pre-existing task should be retained after bounded restore")
	}
}

func TestComposerUnifiedSnapshotMemorySegmentAlignsWithMemoryLifecycle(t *testing.T) {
	comp := newUnifiedSnapshotTestComposer(t)
	runID := "run-unified-snapshot-memory-export"
	comp.runtimeMgr.RecordRun(runtimediag.RunRecord{
		Time:                time.Now().UTC(),
		RunID:               runID,
		Status:              "success",
		MemoryScopeSelected: runtimeconfig.RuntimeMemoryScopeSession,
		MemoryHits:          4,
		MemoryBudgetUsed:    2,
		MemoryRerankStats: map[string]int{
			"input_total":    8,
			"reranked_total": 4,
			"output_total":   4,
		},
		MemoryLifecycleAction: memory.LifecycleActionRetention,
	})

	exported, err := comp.ExportUnifiedSnapshot(context.Background(), UnifiedSnapshotExportRequest{
		RunID: runID,
	})
	if err != nil {
		t.Fatalf("export unified snapshot failed: %v", err)
	}
	manifest, err := orchestrationsnapshot.UnmarshalManifest(exported.Payload)
	if err != nil {
		t.Fatalf("decode exported manifest failed: %v", err)
	}
	memoryPayload, err := decodeMemoryPayload(manifest.Segments.Memory.Payload)
	if err != nil {
		t.Fatalf("decode memory payload failed: %v", err)
	}

	cfg := comp.runtimeMgr.EffectiveConfig().Runtime.Memory
	if memoryPayload.ContractVersion != memory.ContractVersionMemoryV1 {
		t.Fatalf("memory contract version = %q, want %q", memoryPayload.ContractVersion, memory.ContractVersionMemoryV1)
	}
	if memoryPayload.Lifecycle.RetentionDays != cfg.Lifecycle.RetentionDays {
		t.Fatalf("memory lifecycle retention_days = %d, want %d", memoryPayload.Lifecycle.RetentionDays, cfg.Lifecycle.RetentionDays)
	}
	if memoryPayload.Lifecycle.LastAction != memory.LifecycleActionRetention {
		t.Fatalf("memory lifecycle last_action = %q, want %q", memoryPayload.Lifecycle.LastAction, memory.LifecycleActionRetention)
	}
	if memoryPayload.RetrievalBaseline.ScopeSelected != runtimeconfig.RuntimeMemoryScopeSession ||
		memoryPayload.RetrievalBaseline.Hits != 4 ||
		memoryPayload.RetrievalBaseline.BudgetUsed != 2 {
		t.Fatalf("memory retrieval baseline mismatch: %#v", memoryPayload.RetrievalBaseline)
	}
}

func TestComposerUnifiedSnapshotMemoryRestoreIdempotentNoInflation(t *testing.T) {
	source := newUnifiedSnapshotTestComposer(t)
	runID := "run-unified-snapshot-memory-idempotent"
	exported, err := source.ExportUnifiedSnapshot(context.Background(), UnifiedSnapshotExportRequest{RunID: runID})
	if err != nil {
		t.Fatalf("export unified snapshot failed: %v", err)
	}

	target := newUnifiedSnapshotTestComposer(t)
	first, err := target.ImportUnifiedSnapshot(context.Background(), UnifiedSnapshotImportRequest{
		Payload:      exported.Payload,
		OperationID:  "op-unified-snapshot-memory-idempotent",
		RestoreMode:  orchestrationsnapshot.RestoreModeStrict,
		CompatWindow: 1,
	})
	if err != nil {
		t.Fatalf("first import failed: %v", err)
	}
	statsAfterFirst, err := target.Scheduler().Stats(context.Background())
	if err != nil {
		t.Fatalf("stats after first import failed: %v", err)
	}
	second, err := target.ImportUnifiedSnapshot(context.Background(), UnifiedSnapshotImportRequest{
		Payload:      exported.Payload,
		OperationID:  "op-unified-snapshot-memory-idempotent",
		RestoreMode:  orchestrationsnapshot.RestoreModeStrict,
		CompatWindow: 1,
	})
	if err != nil {
		t.Fatalf("second import failed: %v", err)
	}
	statsAfterSecond, err := target.Scheduler().Stats(context.Background())
	if err != nil {
		t.Fatalf("stats after second import failed: %v", err)
	}

	if !containsString(first.AppliedSegments, "memory") {
		t.Fatalf("first import should apply memory segment, got %#v", first.AppliedSegments)
	}
	if len(second.AppliedSegments) != 0 {
		t.Fatalf("second import should not re-apply segments, got %#v", second.AppliedSegments)
	}
	if second.RestoreAction != orchestrationsnapshot.RestoreActionIdempotentNoop {
		t.Fatalf("second restore action = %q, want %q", second.RestoreAction, orchestrationsnapshot.RestoreActionIdempotentNoop)
	}
	if statsAfterFirst.QueueTotal != statsAfterSecond.QueueTotal ||
		statsAfterFirst.ClaimTotal != statsAfterSecond.ClaimTotal ||
		statsAfterFirst.CompleteTotal != statsAfterSecond.CompleteTotal {
		t.Fatalf("idempotent restore should not inflate scheduler stats: first=%#v second=%#v", statsAfterFirst, statsAfterSecond)
	}
}

func TestComposerUnifiedSnapshotMemoryRetrievalQualityStableAcrossCompatibleRestore(t *testing.T) {
	source := newUnifiedSnapshotTestComposer(t)
	runID := "run-unified-snapshot-memory-retrieval-stability"
	source.runtimeMgr.RecordRun(runtimediag.RunRecord{
		Time:                time.Now().UTC(),
		RunID:               runID,
		Status:              "success",
		MemoryScopeSelected: runtimeconfig.RuntimeMemoryScopeSession,
		MemoryHits:          5,
		MemoryBudgetUsed:    3,
		MemoryRerankStats: map[string]int{
			"output_total": 5,
		},
		MemoryLifecycleAction: memory.LifecycleActionTTL,
	})
	exported, err := source.ExportUnifiedSnapshot(context.Background(), UnifiedSnapshotExportRequest{RunID: runID})
	if err != nil {
		t.Fatalf("export unified snapshot failed: %v", err)
	}

	strictTarget := newUnifiedSnapshotTestComposer(t)
	strictTarget.runtimeMgr.RecordRun(runtimediag.RunRecord{
		Time:                time.Now().UTC(),
		RunID:               runID,
		Status:              "success",
		MemoryScopeSelected: runtimeconfig.RuntimeMemoryScopeProject,
		MemoryHits:          30,
		MemoryBudgetUsed:    20,
		MemoryRerankStats:   map[string]int{"output_total": 30},
	})
	_, err = strictTarget.ImportUnifiedSnapshot(context.Background(), UnifiedSnapshotImportRequest{
		Payload:      exported.Payload,
		OperationID:  "op-unified-snapshot-memory-retrieval-strict",
		RestoreMode:  orchestrationsnapshot.RestoreModeStrict,
		CompatWindow: 1,
	})
	if err == nil {
		t.Fatal("strict restore should fail on retrieval quality drift")
	}

	compatibleTarget := newUnifiedSnapshotTestComposer(t)
	compatibleTarget.runtimeMgr.RecordRun(runtimediag.RunRecord{
		Time:                time.Now().UTC(),
		RunID:               runID,
		Status:              "success",
		MemoryScopeSelected: runtimeconfig.RuntimeMemoryScopeProject,
		MemoryHits:          30,
		MemoryBudgetUsed:    20,
		MemoryRerankStats:   map[string]int{"output_total": 30},
	})
	result, err := compatibleTarget.ImportUnifiedSnapshot(context.Background(), UnifiedSnapshotImportRequest{
		Payload:      exported.Payload,
		OperationID:  "op-unified-snapshot-memory-retrieval-compatible",
		RestoreMode:  orchestrationsnapshot.RestoreModeCompatible,
		CompatWindow: 1,
	})
	if err != nil {
		t.Fatalf("compatible restore should keep bounded semantics, got %v", err)
	}
	if result.ConflictCode != "snapshot_memory_retrieval_quality_drift" {
		t.Fatalf("conflict code = %q, want snapshot_memory_retrieval_quality_drift", result.ConflictCode)
	}
	if result.RestoreAction != orchestrationsnapshot.RestoreActionCompatibleBounded {
		t.Fatalf("restore action = %q, want %q", result.RestoreAction, orchestrationsnapshot.RestoreActionCompatibleBounded)
	}
	if !containsString(result.SkippedSegments, "memory") {
		t.Fatalf("compatible restore should skip memory segment on retrieval drift, got %#v", result.SkippedSegments)
	}
}

func TestComposerUnifiedSnapshotRecoveryInteractionStateAlignsWithRealtimeAndIsolateHandoff(t *testing.T) {
	comp := newUnifiedSnapshotTestComposer(t)
	runID := "run-unified-snapshot-interaction-state"
	comp.runtimeMgr.RecordRun(runtimediag.RunRecord{
		Time:                    time.Now().UTC(),
		RunID:                   runID,
		Status:                  "failed",
		RealtimeProtocolVersion: "realtime_event_protocol.v1",
		RealtimeSessionID:       "session-unified-snapshot-interaction",
		RealtimeEventSeqMax:     7,
		RealtimeInterruptTotal:  1,
		RealtimeResumeTotal:     0,
		RealtimeResumeCursor:    "session-unified-snapshot-interaction:run-unified-snapshot-interaction-state:3",
		Stage2ReasonCode:        "isolate_handoff_rejected",
		Stage2Reason:            "isolate_handoff_rejected",
		Stage2SkipReason:        "stage2.isolate_handoff.empty",
	})

	exported, err := comp.ExportUnifiedSnapshot(context.Background(), UnifiedSnapshotExportRequest{
		RunID: runID,
	})
	if err != nil {
		t.Fatalf("export unified snapshot failed: %v", err)
	}
	manifest, err := orchestrationsnapshot.UnmarshalManifest(exported.Payload)
	if err != nil {
		t.Fatalf("decode exported manifest failed: %v", err)
	}
	recoveryPayload, err := decodeComposerRecoveryPayload(manifest.Segments.ComposerRecovery.Payload)
	if err != nil {
		t.Fatalf("decode recovery payload failed: %v", err)
	}
	interaction := recoveryPayload.Recovery.Interaction
	if interaction.Realtime.SessionID != "session-unified-snapshot-interaction" ||
		interaction.Realtime.ResumeCursor != "session-unified-snapshot-interaction:run-unified-snapshot-interaction-state:3" ||
		interaction.Realtime.EventSeqMax != 7 ||
		interaction.Realtime.InterruptTotal != 1 {
		t.Fatalf("realtime interaction snapshot mismatch: %#v", interaction.Realtime)
	}
	if !interaction.IsolateHandoff.Detected ||
		interaction.IsolateHandoff.Stage2ReasonCode != "isolate_handoff_rejected" ||
		interaction.IsolateHandoff.Stage2SkipReason != "stage2.isolate_handoff.empty" {
		t.Fatalf("isolate-handoff interaction snapshot mismatch: %#v", interaction.IsolateHandoff)
	}
}

func TestComposerUnifiedSnapshotInteractionStateCrashRestartReplayConsistency(t *testing.T) {
	ctx := context.Background()
	runID := "run-unified-snapshot-interaction-crash-restart-replay"
	source := newUnifiedSnapshotTestComposer(t)
	source.runtimeMgr.RecordRun(runtimediag.RunRecord{
		Time:                    time.Now().UTC(),
		RunID:                   runID,
		Status:                  "failed",
		RealtimeProtocolVersion: "realtime_event_protocol.v1",
		RealtimeSessionID:       "session-unified-snapshot-interaction-restart",
		RealtimeEventSeqMax:     9,
		RealtimeInterruptTotal:  1,
		RealtimeResumeTotal:     0,
		RealtimeResumeCursor:    "session-unified-snapshot-interaction-restart:run-unified-snapshot-interaction-crash-restart-replay:4",
		Stage2ReasonCode:        "isolate_handoff_rejected",
		Stage2Reason:            "isolate_handoff_rejected",
		Stage2SkipReason:        "stage2.isolate_handoff.empty",
	})
	exported, err := source.ExportUnifiedSnapshot(ctx, UnifiedSnapshotExportRequest{RunID: runID})
	if err != nil {
		t.Fatalf("export unified snapshot failed: %v", err)
	}

	root := t.TempDir()
	store, err := NewFileRecoveryStore(root)
	if err != nil {
		t.Fatalf("new file recovery store: %v", err)
	}
	target := newUnifiedSnapshotTestComposerWithRecoveryStore(t, store)
	first, err := target.ImportUnifiedSnapshot(ctx, UnifiedSnapshotImportRequest{
		Payload:      exported.Payload,
		OperationID:  "op-unified-snapshot-interaction-crash-restart-replay",
		RestoreMode:  orchestrationsnapshot.RestoreModeStrict,
		CompatWindow: 1,
	})
	if err != nil {
		t.Fatalf("first import failed: %v", err)
	}
	if !containsString(first.AppliedSegments, "composer_recovery") {
		t.Fatalf("first import should apply composer_recovery segment, got %#v", first.AppliedSegments)
	}
	afterFirst, found, err := store.Load(ctx, runID)
	if err != nil || !found {
		t.Fatalf("load recovery snapshot after first import failed: found=%v err=%v", found, err)
	}

	second, err := target.ImportUnifiedSnapshot(ctx, UnifiedSnapshotImportRequest{
		Payload:      exported.Payload,
		OperationID:  "op-unified-snapshot-interaction-crash-restart-replay",
		RestoreMode:  orchestrationsnapshot.RestoreModeStrict,
		CompatWindow: 1,
	})
	if err != nil {
		t.Fatalf("second import failed: %v", err)
	}
	if second.RestoreAction != orchestrationsnapshot.RestoreActionIdempotentNoop {
		t.Fatalf("second restore action = %q, want %q", second.RestoreAction, orchestrationsnapshot.RestoreActionIdempotentNoop)
	}
	afterReplay, found, err := store.Load(ctx, runID)
	if err != nil || !found {
		t.Fatalf("load recovery snapshot after replay import failed: found=%v err=%v", found, err)
	}
	if !reflect.DeepEqual(afterFirst.Interaction, afterReplay.Interaction) {
		t.Fatalf("interaction state drift after replay import: first=%#v replay=%#v", afterFirst.Interaction, afterReplay.Interaction)
	}

	restartedStore, err := NewFileRecoveryStore(root)
	if err != nil {
		t.Fatalf("reopen file recovery store failed: %v", err)
	}
	restarted := newUnifiedSnapshotTestComposerWithRecoveryStore(t, restartedStore)
	persisted, found, err := restartedStore.Load(ctx, runID)
	if err != nil || !found {
		t.Fatalf("load recovery snapshot after restart failed: found=%v err=%v", found, err)
	}
	if !reflect.DeepEqual(afterFirst.Interaction, persisted.Interaction) {
		t.Fatalf("interaction state drift after restart: first=%#v restart=%#v", afterFirst.Interaction, persisted.Interaction)
	}
	if _, err := restarted.Recover(ctx, RecoverRequest{RunID: runID}); err != nil {
		t.Fatalf("recover after restart failed: %v", err)
	}
}

func newUnifiedSnapshotTestComposer(t *testing.T) *Composer {
	t.Helper()
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX_UNIFIED_SNAPSHOT_TEST"})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	model := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	comp, err := NewBuilder(model).WithRuntimeManager(mgr).Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}
	return comp
}

func newUnifiedSnapshotTestComposerWithRecoveryStore(t *testing.T, store RecoveryStore) *Composer {
	t.Helper()
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX_UNIFIED_SNAPSHOT_TEST"})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	model := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	builder := NewBuilder(model).WithRuntimeManager(mgr)
	if store != nil {
		builder = builder.WithRecoveryStore(store)
	}
	comp, err := builder.Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}
	return comp
}
