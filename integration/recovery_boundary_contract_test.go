package integration

import (
	"context"
	"os"
	"path/filepath"
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

func TestRecoveryBoundaryCrashRestartReplayTimeoutMatrix(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "scheduler-state.json")
	store1, err := scheduler.NewFileStore(path)
	if err != nil {
		t.Fatalf("new file store #1: %v", err)
	}
	s1, err := scheduler.New(
		store1,
		scheduler.WithLeaseTimeout(70*time.Millisecond),
		scheduler.WithRecoveryBoundary(scheduler.RecoveryBoundaryConfig{
			Enabled:                  true,
			ResumeBoundary:           scheduler.RecoveryResumeBoundaryNextAttemptOnly,
			InflightPolicy:           scheduler.RecoveryInflightPolicyNoRewind,
			TimeoutReentryPolicy:     scheduler.RecoveryTimeoutReentryPolicySingleReentryFail,
			TimeoutReentryMaxPerTask: 1,
		}),
	)
	if err != nil {
		t.Fatalf("new scheduler #1: %v", err)
	}
	if _, err := s1.Enqueue(ctx, scheduler.Task{
		TaskID:      "task-a17-matrix",
		RunID:       "run-a17-matrix",
		MaxAttempts: 5,
	}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if _, ok, err := s1.Claim(ctx, "worker-a17-matrix-a"); err != nil || !ok {
		t.Fatalf("claim #1 failed: ok=%v err=%v", ok, err)
	}

	store2, err := scheduler.NewFileStore(path)
	if err != nil {
		t.Fatalf("new file store #2: %v", err)
	}
	s2, err := scheduler.New(
		store2,
		scheduler.WithLeaseTimeout(70*time.Millisecond),
		scheduler.WithRecoveryBoundary(scheduler.RecoveryBoundaryConfig{
			Enabled:                  true,
			ResumeBoundary:           scheduler.RecoveryResumeBoundaryNextAttemptOnly,
			InflightPolicy:           scheduler.RecoveryInflightPolicyNoRewind,
			TimeoutReentryPolicy:     scheduler.RecoveryTimeoutReentryPolicySingleReentryFail,
			TimeoutReentryMaxPerTask: 1,
		}),
	)
	if err != nil {
		t.Fatalf("new scheduler #2: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	firstExpired, err := s2.ExpireLeases(ctx)
	if err != nil {
		t.Fatalf("expire #1 failed: %v", err)
	}
	if len(firstExpired) != 1 || firstExpired[0].Record.State != scheduler.TaskStateQueued {
		t.Fatalf("first expire should requeue once, got %#v", firstExpired)
	}
	if _, ok, err := s2.Claim(ctx, "worker-a17-matrix-b"); err != nil || !ok {
		t.Fatalf("claim #2 failed: ok=%v err=%v", ok, err)
	}

	time.Sleep(100 * time.Millisecond)
	secondExpired, err := s2.ExpireLeases(ctx)
	if err != nil {
		t.Fatalf("expire #2 failed: %v", err)
	}
	if len(secondExpired) != 1 || secondExpired[0].Record.State != scheduler.TaskStateFailed {
		t.Fatalf("second expire should fail after reentry budget exhaustion, got %#v", secondExpired)
	}
	record, ok, err := s2.Get(ctx, "task-a17-matrix")
	if err != nil || !ok {
		t.Fatalf("get failed: ok=%v err=%v", ok, err)
	}
	if record.State != scheduler.TaskStateFailed {
		t.Fatalf("record state=%q, want failed", record.State)
	}
	stats, err := s2.Stats(ctx)
	if err != nil {
		t.Fatalf("stats failed: %v", err)
	}
	if stats.RecoveryTimeoutReentryTotal != 1 || stats.RecoveryTimeoutReentryExhaustedTotal != 1 {
		t.Fatalf("unexpected reentry stats: %#v", stats)
	}

	snap, err := s2.Snapshot(ctx)
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}
	s3, err := scheduler.New(
		scheduler.NewMemoryStore(),
		scheduler.WithRecoveryBoundary(scheduler.RecoveryBoundaryConfig{
			Enabled:                  true,
			ResumeBoundary:           scheduler.RecoveryResumeBoundaryNextAttemptOnly,
			InflightPolicy:           scheduler.RecoveryInflightPolicyNoRewind,
			TimeoutReentryPolicy:     scheduler.RecoveryTimeoutReentryPolicySingleReentryFail,
			TimeoutReentryMaxPerTask: 1,
		}),
	)
	if err != nil {
		t.Fatalf("new scheduler #3: %v", err)
	}
	if err := s3.Restore(ctx, snap); err != nil {
		t.Fatalf("restore #1 failed: %v", err)
	}
	beforeReplay, err := s3.Stats(ctx)
	if err != nil {
		t.Fatalf("stats before replay failed: %v", err)
	}
	if err := s3.Restore(ctx, snap); err != nil {
		t.Fatalf("restore #2 failed: %v", err)
	}
	afterReplay, err := s3.Stats(ctx)
	if err != nil {
		t.Fatalf("stats after replay failed: %v", err)
	}
	if beforeReplay.RecoveryTimeoutReentryTotal != afterReplay.RecoveryTimeoutReentryTotal ||
		beforeReplay.RecoveryTimeoutReentryExhaustedTotal != afterReplay.RecoveryTimeoutReentryExhaustedTotal ||
		beforeReplay.FailTotal != afterReplay.FailTotal {
		t.Fatalf("replay should remain idempotent: before=%#v after=%#v", beforeReplay, afterReplay)
	}
}

func TestRecoveryBoundaryRunStreamEquivalence(t *testing.T) {
	exec := func(stream bool, runID, taskID string) runtimediag.RunRecord {
		cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
		recoveryPath := filepath.Join(t.TempDir(), "recovery")
		writeRecoveryBoundaryRuntimeConfig(t, cfgPath, recoveryPath)

		mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX_RECOVERY_BOUNDARY_EQ_TEST"})
		if err != nil {
			t.Fatalf("new runtime manager: %v", err)
		}
		t.Cleanup(func() { _ = mgr.Close() })
		model := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
		model.SetStream([]types.ModelEvent{{Type: types.ModelEventTypeFinalAnswer, TextDelta: "ok"}}, nil)

		dispatcher := event.NewDispatcher(event.NewRuntimeRecorder(mgr))
		comp, err := composer.NewBuilder(model).
			WithRuntimeManager(mgr).
			WithEventHandler(dispatcherHandler{dispatcher: dispatcher}).
			Build()
		if err != nil {
			t.Fatalf("new composer: %v", err)
		}

		ctx := context.Background()
		if _, err := comp.Scheduler().Enqueue(ctx, scheduler.Task{
			TaskID:      taskID,
			RunID:       runID,
			MaxAttempts: 5,
		}); err != nil {
			t.Fatalf("enqueue task failed: %v", err)
		}
		if _, ok, err := comp.Scheduler().Claim(ctx, "worker-a17-eq-a"); err != nil || !ok {
			t.Fatalf("claim #1 failed: ok=%v err=%v", ok, err)
		}
		time.Sleep(110 * time.Millisecond)
		if _, err := comp.Scheduler().ExpireLeases(ctx); err != nil {
			t.Fatalf("expire #1 failed: %v", err)
		}
		if _, ok, err := comp.Scheduler().Claim(ctx, "worker-a17-eq-b"); err != nil || !ok {
			t.Fatalf("claim #2 failed: ok=%v err=%v", ok, err)
		}
		time.Sleep(110 * time.Millisecond)
		if _, err := comp.Scheduler().ExpireLeases(ctx); err != nil {
			t.Fatalf("expire #2 failed: %v", err)
		}

		req := types.RunRequest{RunID: runID, Input: "emit-finished"}
		if stream {
			if _, err := comp.Stream(ctx, req, nil); err != nil {
				t.Fatalf("composer stream failed: %v", err)
			}
		} else {
			if _, err := comp.Run(ctx, req, nil); err != nil {
				t.Fatalf("composer run failed: %v", err)
			}
		}
		return findRunRecord(t, mgr.RecentRuns(10), runID)
	}

	runRecord := exec(false, "run-a17-boundary-run", "task-a17-boundary-run")
	streamRecord := exec(true, "run-a17-boundary-stream", "task-a17-boundary-stream")

	if runRecord.Status != streamRecord.Status {
		t.Fatalf("status mismatch run=%q stream=%q", runRecord.Status, streamRecord.Status)
	}
	if runRecord.RecoveryResumeBoundary != streamRecord.RecoveryResumeBoundary ||
		runRecord.RecoveryInflightPolicy != streamRecord.RecoveryInflightPolicy {
		t.Fatalf("recovery boundary semantic mismatch run=%#v stream=%#v", runRecord, streamRecord)
	}
	if runRecord.RecoveryTimeoutReentryTotal != streamRecord.RecoveryTimeoutReentryTotal ||
		runRecord.RecoveryTimeoutReentryExhaustedTotal != streamRecord.RecoveryTimeoutReentryExhaustedTotal {
		t.Fatalf("recovery reentry counters mismatch run=%#v stream=%#v", runRecord, streamRecord)
	}
	if runRecord.RecoveryTimeoutReentryTotal != 1 || runRecord.RecoveryTimeoutReentryExhaustedTotal != 1 {
		t.Fatalf("unexpected reentry counters in run summary: %#v", runRecord)
	}
}

func TestRecoveryBoundaryReplayIdempotency(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	recoveryPath := filepath.Join(t.TempDir(), "recovery")
	writeRecoveryBoundaryRuntimeConfig(t, cfgPath, recoveryPath)

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX_RECOVERY_BOUNDARY_REPLAY_TEST"})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	model := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	dispatcher := event.NewDispatcher(event.NewRuntimeRecorder(mgr))
	comp, err := composer.NewBuilder(model).
		WithRuntimeManager(mgr).
		WithEventHandler(dispatcherHandler{dispatcher: dispatcher}).
		Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}

	ctx := context.Background()
	runID := "run-a17-replay-idempotent"
	taskID := "task-a17-replay-idempotent"
	if _, err := comp.Scheduler().Enqueue(ctx, scheduler.Task{
		TaskID:      taskID,
		RunID:       runID,
		MaxAttempts: 5,
	}); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	if _, ok, err := comp.Scheduler().Claim(ctx, "worker-a17-replay-a"); err != nil || !ok {
		t.Fatalf("claim #1 failed: ok=%v err=%v", ok, err)
	}
	time.Sleep(110 * time.Millisecond)
	if _, err := comp.Scheduler().ExpireLeases(ctx); err != nil {
		t.Fatalf("expire #1 failed: %v", err)
	}
	if _, ok, err := comp.Scheduler().Claim(ctx, "worker-a17-replay-b"); err != nil || !ok {
		t.Fatalf("claim #2 failed: ok=%v err=%v", ok, err)
	}
	time.Sleep(110 * time.Millisecond)
	if _, err := comp.Scheduler().ExpireLeases(ctx); err != nil {
		t.Fatalf("expire #2 failed: %v", err)
	}

	req := types.RunRequest{RunID: runID, Input: "emit-finished"}
	if _, err := comp.Run(ctx, req, nil); err != nil {
		t.Fatalf("first run failed: %v", err)
	}
	first := findRunRecord(t, mgr.RecentRuns(10), runID)
	if _, err := comp.Run(ctx, req, nil); err != nil {
		t.Fatalf("second run failed: %v", err)
	}
	second := findRunRecord(t, mgr.RecentRuns(10), runID)
	if first.RecoveryTimeoutReentryTotal != second.RecoveryTimeoutReentryTotal ||
		first.RecoveryTimeoutReentryExhaustedTotal != second.RecoveryTimeoutReentryExhaustedTotal ||
		first.RecoveryResumeBoundary != second.RecoveryResumeBoundary ||
		first.RecoveryInflightPolicy != second.RecoveryInflightPolicy {
		t.Fatalf("run replay should keep recovery-boundary summary stable: first=%#v second=%#v", first, second)
	}
}

func writeRecoveryBoundaryRuntimeConfig(t *testing.T, path, recoveryPath string) {
	t.Helper()
	content := "" +
		"reload:\n" +
		"  enabled: false\n" +
		"scheduler:\n" +
		"  enabled: true\n" +
		"  backend: memory\n" +
		"  lease_timeout: 80ms\n" +
		"  heartbeat_interval: 20ms\n" +
		"  queue_limit: 1024\n" +
		"  retry_max_attempts: 5\n" +
		"recovery:\n" +
		"  enabled: true\n" +
		"  backend: file\n" +
		"  path: " + filepath.ToSlash(recoveryPath) + "\n" +
		"  conflict_policy: fail_fast\n" +
		"  resume_boundary: next_attempt_only\n" +
		"  inflight_policy: no_rewind\n" +
		"  timeout_reentry_policy: single_reentry_then_fail\n" +
		"  timeout_reentry_max_per_task: 1\n" +
		"subagent:\n" +
		"  max_depth: 4\n" +
		"  max_active_children: 8\n" +
		"  child_timeout_budget: 5s\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write recovery boundary runtime config: %v", err)
	}
}
