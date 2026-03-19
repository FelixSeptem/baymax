package integration

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/integration/fakes"
	"github.com/FelixSeptem/baymax/observability/event"
	"github.com/FelixSeptem/baymax/orchestration/composer"
	"github.com/FelixSeptem/baymax/orchestration/scheduler"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
	runtimediag "github.com/FelixSeptem/baymax/runtime/diagnostics"
)

type recoveryDispatcherHandler struct {
	dispatcher *event.Dispatcher
}

func (h recoveryDispatcherHandler) OnEvent(ctx context.Context, ev types.Event) {
	if h.dispatcher == nil {
		return
	}
	h.dispatcher.Emit(ctx, ev)
}

func TestComposerRecoveryCrossSessionResumeSuccess(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	recoveryPath := filepath.Join(t.TempDir(), "recovery")
	writeRecoveryRuntimeConfig(t, cfgPath, recoveryPath)

	mgr1, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX_A9_TEST"})
	if err != nil {
		t.Fatalf("new runtime manager #1: %v", err)
	}
	defer func() { _ = mgr1.Close() }()

	model := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	comp1, err := composer.NewBuilder(model).WithRuntimeManager(mgr1).Build()
	if err != nil {
		t.Fatalf("new composer #1: %v", err)
	}
	runID := "run-a9-recovery-success"
	taskID := "task-a9-recovery-success"
	out, err := comp1.DispatchChild(context.Background(), composer.ChildDispatchRequest{
		Task: scheduler.Task{
			TaskID: taskID,
			RunID:  runID,
		},
		Target:               composer.ChildTargetLocal,
		ParentDepth:          0,
		ParentActiveChildren: 0,
		ChildTimeout:         500 * time.Millisecond,
		LocalRunner: composer.LocalChildRunnerFunc(func(ctx context.Context, task scheduler.Task) (map[string]any, error) {
			return map[string]any{"ok": true, "task_id": task.TaskID}, nil
		}),
	})
	if err != nil {
		t.Fatalf("dispatch child failed: %v", err)
	}
	if out.Commit.Status != scheduler.TaskStateSucceeded {
		t.Fatalf("unexpected commit status: %#v", out.Commit)
	}

	mgr2, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX_A9_TEST"})
	if err != nil {
		t.Fatalf("new runtime manager #2: %v", err)
	}
	defer func() { _ = mgr2.Close() }()
	dispatcher := event.NewDispatcher(event.NewRuntimeRecorder(mgr2))
	comp2, err := composer.NewBuilder(model).
		WithRuntimeManager(mgr2).
		WithEventHandler(recoveryDispatcherHandler{dispatcher: dispatcher}).
		WithSchedulerStore(scheduler.NewMemoryStore()).
		Build()
	if err != nil {
		t.Fatalf("new composer #2: %v", err)
	}

	result, err := comp2.Recover(context.Background(), composer.RecoverRequest{RunID: runID})
	if err != nil {
		t.Fatalf("recover failed: %v", err)
	}
	if result.ReplayedTerminalCommits != 1 {
		t.Fatalf("replayed terminal commits = %d, want 1", result.ReplayedTerminalCommits)
	}
	record, found, err := comp2.Scheduler().Get(context.Background(), taskID)
	if err != nil || !found {
		t.Fatalf("get recovered task failed: found=%v err=%v", found, err)
	}
	if record.State != scheduler.TaskStateSucceeded {
		t.Fatalf("recovered task state = %q, want succeeded", record.State)
	}

	if _, err := comp2.Run(context.Background(), types.RunRequest{RunID: runID, Input: "emit-finished"}, nil); err != nil {
		t.Fatalf("composer run failed: %v", err)
	}
	run := findRecoveryRunRecord(t, mgr2.RecentRuns(10), runID)
	if !run.RecoveryEnabled || !run.RecoveryRecovered || run.RecoveryReplayTotal != 1 {
		t.Fatalf("recovery run summary mismatch: %#v", run)
	}
	if run.RecoveryResumeBoundary != runtimeconfig.RecoveryResumeBoundaryNextAttemptOnly {
		t.Fatalf("recovery_resume_boundary=%q, want %q", run.RecoveryResumeBoundary, runtimeconfig.RecoveryResumeBoundaryNextAttemptOnly)
	}
	if run.RecoveryInflightPolicy != runtimeconfig.RecoveryInflightPolicyNoRewind {
		t.Fatalf("recovery_inflight_policy=%q, want %q", run.RecoveryInflightPolicy, runtimeconfig.RecoveryInflightPolicyNoRewind)
	}
	if run.RecoveryTimeoutReentryTotal != 0 || run.RecoveryTimeoutReentryExhaustedTotal != 0 {
		t.Fatalf("unexpected recovery timeout reentry counters: %#v", run)
	}
}

func TestComposerRecoveryReplayIdempotent(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	recoveryPath := filepath.Join(t.TempDir(), "recovery")
	writeRecoveryRuntimeConfig(t, cfgPath, recoveryPath)

	mgr1, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX_A9_TEST"})
	if err != nil {
		t.Fatalf("new runtime manager #1: %v", err)
	}
	defer func() { _ = mgr1.Close() }()

	model := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	comp1, err := composer.NewBuilder(model).WithRuntimeManager(mgr1).Build()
	if err != nil {
		t.Fatalf("new composer #1: %v", err)
	}
	runID := "run-a9-recovery-replay"
	if _, err := comp1.DispatchChild(context.Background(), composer.ChildDispatchRequest{
		Task: scheduler.Task{
			TaskID: "task-a9-recovery-replay",
			RunID:  runID,
			PeerID: "peer-a9",
		},
		Target:               composer.ChildTargetLocal,
		ParentDepth:          0,
		ParentActiveChildren: 0,
		ChildTimeout:         400 * time.Millisecond,
		LocalRunner: composer.LocalChildRunnerFunc(func(ctx context.Context, task scheduler.Task) (map[string]any, error) {
			return map[string]any{"ok": true}, nil
		}),
	}); err != nil {
		t.Fatalf("dispatch child failed: %v", err)
	}

	mgr2, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX_A9_TEST"})
	if err != nil {
		t.Fatalf("new runtime manager #2: %v", err)
	}
	defer func() { _ = mgr2.Close() }()
	comp2, err := composer.NewBuilder(model).
		WithRuntimeManager(mgr2).
		WithSchedulerStore(scheduler.NewMemoryStore()).
		Build()
	if err != nil {
		t.Fatalf("new composer #2: %v", err)
	}
	if _, err := comp2.Recover(context.Background(), composer.RecoverRequest{RunID: runID}); err != nil {
		t.Fatalf("recover #1 failed: %v", err)
	}
	stats1, err := comp2.SchedulerStats(context.Background())
	if err != nil {
		t.Fatalf("scheduler stats #1 failed: %v", err)
	}
	if _, err := comp2.Recover(context.Background(), composer.RecoverRequest{RunID: runID}); err != nil {
		t.Fatalf("recover #2 failed: %v", err)
	}
	stats2, err := comp2.SchedulerStats(context.Background())
	if err != nil {
		t.Fatalf("scheduler stats #2 failed: %v", err)
	}
	if stats1.CompleteTotal != stats2.CompleteTotal || stats1.FailTotal != stats2.FailTotal || stats1.QueueTotal != stats2.QueueTotal {
		t.Fatalf("recovery replay should be idempotent: stats1=%#v stats2=%#v", stats1, stats2)
	}
	if stats1.RecoveryTimeoutReentryTotal != stats2.RecoveryTimeoutReentryTotal ||
		stats1.RecoveryTimeoutReentryExhaustedTotal != stats2.RecoveryTimeoutReentryExhaustedTotal {
		t.Fatalf("recovery timeout reentry stats should be idempotent: stats1=%#v stats2=%#v", stats1, stats2)
	}
}

func TestComposerRecoveryConflictFailFast(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	recoveryPath := filepath.Join(t.TempDir(), "recovery")
	writeRecoveryRuntimeConfig(t, cfgPath, recoveryPath)

	mgr1, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX_A9_TEST"})
	if err != nil {
		t.Fatalf("new runtime manager #1: %v", err)
	}
	defer func() { _ = mgr1.Close() }()

	model := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	comp1, err := composer.NewBuilder(model).WithRuntimeManager(mgr1).Build()
	if err != nil {
		t.Fatalf("new composer #1: %v", err)
	}
	runID := "run-a9-recovery-conflict"
	if _, err := comp1.DispatchChild(context.Background(), composer.ChildDispatchRequest{
		Task: scheduler.Task{
			TaskID: "task-a9-recovery-conflict",
			RunID:  runID,
		},
		Target:               composer.ChildTargetLocal,
		ParentDepth:          0,
		ParentActiveChildren: 0,
		ChildTimeout:         400 * time.Millisecond,
		LocalRunner: composer.LocalChildRunnerFunc(func(ctx context.Context, task scheduler.Task) (map[string]any, error) {
			return map[string]any{"ok": true}, nil
		}),
	}); err != nil {
		t.Fatalf("dispatch child failed: %v", err)
	}

	mgr2, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX_A9_TEST"})
	if err != nil {
		t.Fatalf("new runtime manager #2: %v", err)
	}
	defer func() { _ = mgr2.Close() }()
	comp2, err := composer.NewBuilder(model).
		WithRuntimeManager(mgr2).
		WithSchedulerStore(scheduler.NewMemoryStore()).
		Build()
	if err != nil {
		t.Fatalf("new composer #2: %v", err)
	}
	if _, err := comp2.Scheduler().Enqueue(context.Background(), scheduler.Task{TaskID: "task-existing-conflict", RunID: runID}); err != nil {
		t.Fatalf("preload scheduler state failed: %v", err)
	}
	_, recoverErr := comp2.Recover(context.Background(), composer.RecoverRequest{RunID: runID})
	if recoverErr == nil {
		t.Fatal("expected conflict error for fail-fast recovery")
	}
	var recErr *composer.RecoveryError
	if !errors.As(recoverErr, &recErr) {
		t.Fatalf("expected RecoveryError, got %T (%v)", recoverErr, recoverErr)
	}
	if recErr.Code != composer.RecoveryErrorConflict {
		t.Fatalf("recovery error code = %q, want %q", recErr.Code, composer.RecoveryErrorConflict)
	}
}

func TestComposerRecoveryBoundaryViolationClassifiedAsConflict(t *testing.T) {
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX_A17_TEST"})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	model := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	store := composer.NewMemoryRecoveryStore()
	now := time.Now()
	runID := "run-a17-boundary-conflict"
	taskID := "task-a17-boundary-conflict"
	attemptID := taskID + "-attempt-1"
	if err := store.Save(context.Background(), composer.RecoverySnapshot{
		Version:   composer.RecoverySnapshotVersion,
		UpdatedAt: now,
		Run: composer.RecoveryRunSnapshot{
			RunID: runID,
		},
		Scheduler: scheduler.StoreSnapshot{
			Backend: "memory",
			Tasks: []scheduler.TaskRecord{
				{
					Task:  scheduler.Task{TaskID: taskID, RunID: runID},
					State: scheduler.TaskStateSucceeded,
					Attempts: []scheduler.Attempt{
						{
							AttemptID:  attemptID,
							Attempt:    1,
							Status:     scheduler.AttemptStatusSucceeded,
							StartedAt:  now.Add(-2 * time.Second),
							TerminalAt: now.Add(-time.Second),
						},
					},
					CurrentAttempt: "",
					CreatedAt:      now.Add(-2 * time.Second),
					UpdatedAt:      now,
				},
			},
			Queue: []string{taskID},
			Stats: scheduler.Stats{Backend: "memory"},
		},
		Replay: composer.RecoveryReplayCursor{
			Sequence:            now.UnixNano(),
			TerminalCommitCount: 1,
		},
		ConflictPolicy: runtimeconfig.RecoveryConflictPolicyFailFast,
	}); err != nil {
		t.Fatalf("save boundary-violating snapshot: %v", err)
	}

	comp, err := composer.NewBuilder(model).
		WithRuntimeManager(mgr).
		WithRecoveryStore(store).
		WithSchedulerStore(scheduler.NewMemoryStore()).
		Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}

	_, recoverErr := comp.Recover(context.Background(), composer.RecoverRequest{RunID: runID})
	if recoverErr == nil {
		t.Fatal("expected boundary violation conflict error")
	}
	var recErr *composer.RecoveryError
	if !errors.As(recoverErr, &recErr) {
		t.Fatalf("expected RecoveryError, got %T (%v)", recoverErr, recoverErr)
	}
	if recErr.Code != composer.RecoveryErrorConflict {
		t.Fatalf("recovery error code = %q, want %q", recErr.Code, composer.RecoveryErrorConflict)
	}
}

func writeRecoveryRuntimeConfig(t *testing.T, path, recoveryPath string) {
	t.Helper()
	content := strings.Join([]string{
		"reload:",
		"  enabled: false",
		"scheduler:",
		"  enabled: true",
		"  backend: memory",
		"  lease_timeout: 2s",
		"  heartbeat_interval: 400ms",
		"  queue_limit: 1024",
		"  retry_max_attempts: 3",
		"recovery:",
		"  enabled: true",
		"  backend: file",
		"  path: " + filepath.ToSlash(recoveryPath),
		"  conflict_policy: fail_fast",
		"  resume_boundary: next_attempt_only",
		"  inflight_policy: no_rewind",
		"  timeout_reentry_policy: single_reentry_then_fail",
		"  timeout_reentry_max_per_task: 1",
		"subagent:",
		"  max_depth: 4",
		"  max_active_children: 8",
		"  child_timeout_budget: 5s",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write recovery runtime config: %v", err)
	}
}

func findRecoveryRunRecord(t *testing.T, records []runtimediag.RunRecord, runID string) runtimediag.RunRecord {
	t.Helper()
	target := strings.TrimSpace(runID)
	for _, rec := range records {
		if strings.TrimSpace(rec.RunID) == target {
			return rec
		}
	}
	t.Fatalf("run record %q not found in %#v", runID, records)
	return runtimediag.RunRecord{}
}
