package composer

import (
	"context"
	"errors"
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
	source := newA66TestComposer(t)
	runID := "run-a66-strict-conflict"
	if _, err := source.DispatchChild(context.Background(), ChildDispatchRequest{
		Task: scheduler.Task{
			TaskID: "task-a66-source",
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

	target := newA66TestComposer(t)
	if _, err := target.Scheduler().Enqueue(context.Background(), scheduler.Task{
		TaskID: "task-a66-target-existing",
		RunID:  runID,
	}); err != nil {
		t.Fatalf("preload conflicting scheduler state failed: %v", err)
	}

	_, err = target.ImportUnifiedSnapshot(context.Background(), UnifiedSnapshotImportRequest{
		Payload:      exported.Payload,
		OperationID:  "op-a66-strict-conflict",
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
	source := newA66TestComposer(t)
	runID := "run-a66-compatible-conflict"
	if _, err := source.DispatchChild(context.Background(), ChildDispatchRequest{
		Task: scheduler.Task{
			TaskID: "task-a66-compatible-source",
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

	target := newA66TestComposer(t)
	existingTaskID := "task-a66-compatible-target-existing"
	if _, err := target.Scheduler().Enqueue(context.Background(), scheduler.Task{
		TaskID: existingTaskID,
		RunID:  runID,
	}); err != nil {
		t.Fatalf("preload conflicting scheduler state failed: %v", err)
	}

	result, err := target.ImportUnifiedSnapshot(context.Background(), UnifiedSnapshotImportRequest{
		Payload:      exported.Payload,
		OperationID:  "op-a66-compatible-conflict",
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

func TestComposerUnifiedSnapshotMemorySegmentAlignsWithA59Lifecycle(t *testing.T) {
	comp := newA66TestComposer(t)
	runID := "run-a66-memory-export"
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
	source := newA66TestComposer(t)
	runID := "run-a66-memory-idempotent"
	exported, err := source.ExportUnifiedSnapshot(context.Background(), UnifiedSnapshotExportRequest{RunID: runID})
	if err != nil {
		t.Fatalf("export unified snapshot failed: %v", err)
	}

	target := newA66TestComposer(t)
	first, err := target.ImportUnifiedSnapshot(context.Background(), UnifiedSnapshotImportRequest{
		Payload:      exported.Payload,
		OperationID:  "op-a66-memory-idempotent",
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
		OperationID:  "op-a66-memory-idempotent",
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
	source := newA66TestComposer(t)
	runID := "run-a66-memory-retrieval-stability"
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

	strictTarget := newA66TestComposer(t)
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
		OperationID:  "op-a66-memory-retrieval-strict",
		RestoreMode:  orchestrationsnapshot.RestoreModeStrict,
		CompatWindow: 1,
	})
	if err == nil {
		t.Fatal("strict restore should fail on retrieval quality drift")
	}

	compatibleTarget := newA66TestComposer(t)
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
		OperationID:  "op-a66-memory-retrieval-compatible",
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

func newA66TestComposer(t *testing.T) *Composer {
	t.Helper()
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX_A66_TEST"})
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
