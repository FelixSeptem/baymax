package composer

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/integration/fakes"
	"github.com/FelixSeptem/baymax/observability/event"
	"github.com/FelixSeptem/baymax/orchestration/scheduler"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

type timelineCollector struct {
	events []types.Event
}

func (c *timelineCollector) OnEvent(_ context.Context, ev types.Event) {
	c.events = append(c.events, ev)
}

type dispatcherHandler struct {
	dispatcher *event.Dispatcher
}

func (h dispatcherHandler) OnEvent(ctx context.Context, ev types.Event) {
	if h.dispatcher == nil {
		return
	}
	h.dispatcher.Emit(ctx, ev)
}

func TestComposerSchedulerReloadAppliesOnNextAttemptOnly(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	writeComposerRuntimeConfig(t, cfgPath, 250*time.Millisecond)

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		FilePath:        cfgPath,
		EnvPrefix:       "BAYMAX_A8_TEST",
		EnableHotReload: true,
	})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	model := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	comp, err := NewBuilder(model).WithRuntimeManager(mgr).Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}

	ctx := context.Background()
	taskOne := "task-a8-reload-one"
	if _, err := comp.SpawnChild(ctx, ChildDispatchRequest{
		Task: scheduler.Task{TaskID: taskOne, RunID: "run-a8-reload"},
	}); err != nil {
		t.Fatalf("spawn task one: %v", err)
	}
	claimedOne, ok, err := comp.Scheduler().Claim(ctx, "worker-one")
	if err != nil || !ok {
		t.Fatalf("claim task one failed: ok=%v err=%v", ok, err)
	}
	leaseOne := claimedOne.Attempt.LeaseExpiresAt.Sub(claimedOne.Attempt.StartedAt)
	if leaseOne < 150*time.Millisecond || leaseOne > 450*time.Millisecond {
		t.Fatalf("attempt one lease timeout=%s, want around 250ms", leaseOne)
	}

	recordOneBefore, found, err := comp.Scheduler().Get(ctx, taskOne)
	if err != nil || !found {
		t.Fatalf("get task one before reload failed: found=%v err=%v", found, err)
	}
	attemptOneBefore := mustFindAttempt(t, recordOneBefore, claimedOne.Attempt.AttemptID)

	writeComposerRuntimeConfig(t, cfgPath, 950*time.Millisecond)
	waitFor(t, 4*time.Second, func() bool {
		return mgr.EffectiveConfig().Scheduler.LeaseTimeout >= 900*time.Millisecond
	}, "runtime manager lease_timeout reload to >=900ms")

	taskTwo := "task-a8-reload-two"
	if _, err := comp.SpawnChild(ctx, ChildDispatchRequest{
		Task: scheduler.Task{TaskID: taskTwo, RunID: "run-a8-reload"},
	}); err != nil {
		t.Fatalf("spawn task two: %v", err)
	}
	claimedTwo, ok, err := comp.Scheduler().Claim(ctx, "worker-two")
	if err != nil || !ok {
		t.Fatalf("claim task two failed: ok=%v err=%v", ok, err)
	}
	leaseTwo := claimedTwo.Attempt.LeaseExpiresAt.Sub(claimedTwo.Attempt.StartedAt)
	if leaseTwo < 700*time.Millisecond {
		t.Fatalf("attempt two lease timeout=%s, want >=700ms after reload", leaseTwo)
	}

	recordOneAfter, found, err := comp.Scheduler().Get(ctx, taskOne)
	if err != nil || !found {
		t.Fatalf("get task one after reload failed: found=%v err=%v", found, err)
	}
	attemptOneAfter := mustFindAttempt(t, recordOneAfter, claimedOne.Attempt.AttemptID)
	if !attemptOneBefore.LeaseExpiresAt.Equal(attemptOneAfter.LeaseExpiresAt) {
		t.Fatalf(
			"in-flight attempt lease should not change after reload: before=%s after=%s",
			attemptOneBefore.LeaseExpiresAt,
			attemptOneAfter.LeaseExpiresAt,
		)
	}
}

func TestComposerGuardrailFailFastEmitsBudgetRejectAndSummary(t *testing.T) {
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX_A8_TEST"})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	model := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	collector := &timelineCollector{}
	dispatcher := event.NewDispatcher(event.NewRuntimeRecorder(mgr), collector)
	comp, err := NewBuilder(model).
		WithRuntimeManager(mgr).
		WithEventHandler(dispatcherHandler{dispatcher: dispatcher}).
		Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}

	runID := "run-a8-budget-reject"
	_, err = comp.SpawnChild(context.Background(), ChildDispatchRequest{
		Task: scheduler.Task{
			TaskID: "task-a8-budget-reject",
			RunID:  runID,
		},
		ParentDepth:          4,
		ParentActiveChildren: 0,
		ChildTimeout:         50 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("spawn should fail for budget reject")
	}
	var budgetErr *scheduler.BudgetError
	if !errors.As(err, &budgetErr) {
		t.Fatalf("spawn should return budget error, got %T (%v)", err, err)
	}

	hasBudgetRejectReason := false
	for _, ev := range collector.events {
		if ev.Type != types.EventTypeActionTimeline {
			continue
		}
		reason, _ := ev.Payload["reason"].(string)
		if reason == scheduler.ReasonBudgetReject {
			hasBudgetRejectReason = true
			break
		}
	}
	if !hasBudgetRejectReason {
		t.Fatalf("expected timeline reason %q in %#v", scheduler.ReasonBudgetReject, collector.events)
	}

	if _, err := comp.Run(context.Background(), types.RunRequest{RunID: runID, Input: "emit-finished"}, nil); err != nil {
		t.Fatalf("composer run failed: %v", err)
	}
	runs := mgr.RecentRuns(5)
	found := false
	for _, rec := range runs {
		if strings.TrimSpace(rec.RunID) != runID {
			continue
		}
		found = true
		if rec.SubagentBudgetRejectTotal != 1 {
			t.Fatalf("subagent_budget_reject_total=%d, want 1", rec.SubagentBudgetRejectTotal)
		}
	}
	if !found {
		t.Fatalf("run summary for %q not found in %#v", runID, runs)
	}
}

func TestComposerSpawnChildRejectsUnsupportedOperationProfile(t *testing.T) {
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX_A41_TEST"})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	model := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	comp, err := NewBuilder(model).WithRuntimeManager(mgr).Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}

	taskID := "task-a41-invalid-profile"
	_, err = comp.SpawnChild(context.Background(), ChildDispatchRequest{
		Task: scheduler.Task{
			TaskID: taskID,
			RunID:  "run-a41-invalid-profile",
		},
		OperationProfile: "realtime",
	})
	if err == nil {
		t.Fatal("spawn should fail for unsupported operation profile")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "operation profile must be one of") {
		t.Fatalf("unsupported operation profile error mismatch: %v", err)
	}
	if _, found, getErr := comp.Scheduler().Get(context.Background(), taskID); getErr != nil {
		t.Fatalf("get task failed: %v", getErr)
	} else if found {
		t.Fatalf("unsupported profile should not enqueue task %q", taskID)
	}
}

func TestComposerSpawnChildTimeoutResolutionPrecedenceAndSummary(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime-a41.yaml")
	writeComposerA41RuntimeConfig(t, cfgPath)

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		FilePath:  cfgPath,
		EnvPrefix: "BAYMAX_A41_TEST",
	})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	model := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	dispatcher := event.NewDispatcher(event.NewRuntimeRecorder(mgr))
	comp, err := NewBuilder(model).
		WithRuntimeManager(mgr).
		WithEventHandler(dispatcherHandler{dispatcher: dispatcher}).
		Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}

	runID := "run-a41-timeout-resolution"
	domainRecord, err := comp.SpawnChild(context.Background(), ChildDispatchRequest{
		Task: scheduler.Task{
			TaskID: "task-a41-domain-override",
			RunID:  runID,
		},
		OperationProfile:      runtimeconfig.OperationProfileInteractive,
		ParentRemainingBudget: 8 * time.Second,
	})
	if err != nil {
		t.Fatalf("spawn with domain override failed: %v", err)
	}
	domainMeta := domainRecord.Task.TimeoutResolution
	if domainMeta.Source != runtimeconfig.TimeoutResolutionSourceDomain {
		t.Fatalf("domain override source = %q, want %q", domainMeta.Source, runtimeconfig.TimeoutResolutionSourceDomain)
	}
	if domainMeta.ResolvedTimeout != 6*time.Second {
		t.Fatalf("domain override resolved timeout = %s, want 6s", domainMeta.ResolvedTimeout)
	}
	if domainMeta.ParentBudgetClamped {
		t.Fatalf("domain override should not be clamped: %#v", domainMeta)
	}

	requestRecord, err := comp.SpawnChild(context.Background(), ChildDispatchRequest{
		Task: scheduler.Task{
			TaskID: "task-a41-request-override",
			RunID:  runID,
		},
		OperationProfile:      runtimeconfig.OperationProfileInteractive,
		RequestTimeout:        1500 * time.Millisecond,
		ParentRemainingBudget: 1200 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("spawn with request override failed: %v", err)
	}
	requestMeta := requestRecord.Task.TimeoutResolution
	if requestMeta.EffectiveOperationProfile != runtimeconfig.OperationProfileInteractive {
		t.Fatalf(
			"effective operation profile = %q, want %q",
			requestMeta.EffectiveOperationProfile,
			runtimeconfig.OperationProfileInteractive,
		)
	}
	if requestMeta.Source != runtimeconfig.TimeoutResolutionSourceRequest {
		t.Fatalf("request override source = %q, want %q", requestMeta.Source, runtimeconfig.TimeoutResolutionSourceRequest)
	}
	if requestMeta.ResolvedTimeout != 1200*time.Millisecond {
		t.Fatalf("request override resolved timeout = %s, want 1200ms", requestMeta.ResolvedTimeout)
	}
	if !requestMeta.ParentBudgetClamped {
		t.Fatalf("request override should be parent-budget clamped: %#v", requestMeta)
	}
	if strings.TrimSpace(requestMeta.Trace) == "" {
		t.Fatalf("request override trace should not be empty: %#v", requestMeta)
	}

	page, err := comp.Scheduler().QueryTasks(context.Background(), scheduler.TaskBoardQueryRequest{
		TaskID: "task-a41-request-override",
	})
	if err != nil {
		t.Fatalf("query task board failed: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("task board items len = %d, want 1", len(page.Items))
	}
	if page.Items[0].Task.TimeoutResolution.Source != runtimeconfig.TimeoutResolutionSourceRequest {
		t.Fatalf("task board timeout resolution source mismatch: %#v", page.Items[0].Task.TimeoutResolution)
	}

	if _, err := comp.Run(context.Background(), types.RunRequest{
		RunID: runID,
		Input: "emit-finished",
	}, nil); err != nil {
		t.Fatalf("composer run failed: %v", err)
	}
	runs := mgr.RecentRuns(10)
	found := false
	for _, rec := range runs {
		if strings.TrimSpace(rec.RunID) != runID {
			continue
		}
		found = true
		if rec.EffectiveOperationProfile != runtimeconfig.OperationProfileInteractive {
			t.Fatalf("effective_operation_profile = %q, want %q", rec.EffectiveOperationProfile, runtimeconfig.OperationProfileInteractive)
		}
		if rec.TimeoutResolutionSource != runtimeconfig.TimeoutResolutionSourceRequest {
			t.Fatalf("timeout_resolution_source = %q, want %q", rec.TimeoutResolutionSource, runtimeconfig.TimeoutResolutionSourceRequest)
		}
		if strings.TrimSpace(rec.TimeoutResolutionTrace) == "" {
			t.Fatalf("timeout_resolution_trace should not be empty: %#v", rec)
		}
		if rec.TimeoutParentBudgetClampTotal != 1 {
			t.Fatalf("timeout_parent_budget_clamp_total = %d, want 1", rec.TimeoutParentBudgetClampTotal)
		}
		if rec.TimeoutParentBudgetRejectTotal != 0 {
			t.Fatalf("timeout_parent_budget_reject_total = %d, want 0", rec.TimeoutParentBudgetRejectTotal)
		}
	}
	if !found {
		t.Fatalf("run summary for %q not found in %#v", runID, runs)
	}
}

func TestComposerSpawnChildPassesNotBeforeThrough(t *testing.T) {
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX_A13_TEST"})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	model := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	comp, err := NewBuilder(model).WithRuntimeManager(mgr).Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}

	notBefore := time.Now().Add(100 * time.Millisecond)
	record, err := comp.SpawnChild(context.Background(), ChildDispatchRequest{
		Task: scheduler.Task{
			TaskID:    "task-a13-not-before",
			RunID:     "run-a13-not-before",
			NotBefore: notBefore,
		},
	})
	if err != nil {
		t.Fatalf("spawn child failed: %v", err)
	}
	if record.Task.NotBefore.IsZero() || !record.Task.NotBefore.Equal(notBefore.UTC()) {
		t.Fatalf("spawned task not_before mismatch: got=%s want=%s", record.Task.NotBefore, notBefore.UTC())
	}
	if _, ok, err := comp.Scheduler().Claim(context.Background(), "worker-a13"); err != nil || ok {
		t.Fatalf("child should not be claimable before not_before: ok=%v err=%v", ok, err)
	}
	time.Sleep(120 * time.Millisecond)
	if _, ok, err := comp.Scheduler().Claim(context.Background(), "worker-a13"); err != nil || !ok {
		t.Fatalf("child should be claimable after not_before: ok=%v err=%v", ok, err)
	}
}

func TestComposerReadinessPreflightRequiresRuntimeManager(t *testing.T) {
	model := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	comp, err := NewBuilder(model).Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}
	if _, err := comp.ReadinessPreflight(); err == nil {
		t.Fatal("ReadinessPreflight should fail when runtime manager is not configured")
	}
}

func TestComposerReadinessPreflightPassthroughAndReadOnly(t *testing.T) {
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX_A40_TEST"})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	model := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	comp, err := NewBuilder(model).WithRuntimeManager(mgr).Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}

	mgr.SetReadinessComponentSnapshot(runtimeconfig.RuntimeReadinessComponentSnapshot{
		Scheduler: runtimeconfig.RuntimeReadinessComponentState{
			Enabled:           true,
			ConfiguredBackend: "file",
			EffectiveBackend:  "memory",
			Fallback:          true,
			FallbackReason:    "scheduler.backend.file_init_failed",
		},
	})

	before, err := comp.SchedulerStats(context.Background())
	if err != nil {
		t.Fatalf("SchedulerStats before readiness failed: %v", err)
	}
	runtimeResult := mgr.ReadinessPreflight()
	composerResult, err := comp.ReadinessPreflight()
	if err != nil {
		t.Fatalf("composer readiness failed: %v", err)
	}
	after, err := comp.SchedulerStats(context.Background())
	if err != nil {
		t.Fatalf("SchedulerStats after readiness failed: %v", err)
	}

	if runtimeResult.Status != composerResult.Status {
		t.Fatalf("status mismatch runtime=%q composer=%q", runtimeResult.Status, composerResult.Status)
	}
	if readinessFingerprint(runtimeResult) != readinessFingerprint(composerResult) {
		t.Fatalf("readiness semantics mismatch runtime=%s composer=%s", readinessFingerprint(runtimeResult), readinessFingerprint(composerResult))
	}
	if before.QueueTotal != after.QueueTotal || before.ClaimTotal != after.ClaimTotal || before.ReclaimTotal != after.ReclaimTotal {
		t.Fatalf("readiness query should be read-only, before=%#v after=%#v", before, after)
	}
}

func writeComposerRuntimeConfig(t *testing.T, path string, leaseTimeout time.Duration) {
	t.Helper()
	cfg := strings.Join([]string{
		"reload:",
		"  enabled: true",
		"  debounce: 15ms",
		"scheduler:",
		"  enabled: true",
		"  backend: memory",
		"  lease_timeout: " + leaseTimeout.String(),
		"  heartbeat_interval: 100ms",
		"  queue_limit: 64",
		"  retry_max_attempts: 3",
		"subagent:",
		"  max_depth: 4",
		"  max_active_children: 8",
		"  child_timeout_budget: 3s",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatalf("write runtime config %q: %v", path, err)
	}
}

func writeComposerA41RuntimeConfig(t *testing.T, path string) {
	t.Helper()
	cfg := strings.Join([]string{
		"reload:",
		"  enabled: false",
		"scheduler:",
		"  enabled: true",
		"  backend: memory",
		"  lease_timeout: 300ms",
		"  heartbeat_interval: 100ms",
		"  queue_limit: 64",
		"  retry_max_attempts: 3",
		"subagent:",
		"  max_depth: 4",
		"  max_active_children: 8",
		"  child_timeout_budget: 6s",
		"runtime:",
		"  operation_profiles:",
		"    default_profile: legacy",
		"    legacy:",
		"      timeout: 3s",
		"    interactive:",
		"      timeout: 9s",
		"    background:",
		"      timeout: 30s",
		"    batch:",
		"      timeout: 2m",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatalf("write runtime config %q: %v", path, err)
	}
}

func waitFor(t *testing.T, timeout time.Duration, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %s", msg)
}

func mustFindAttempt(t *testing.T, record scheduler.TaskRecord, attemptID string) scheduler.Attempt {
	t.Helper()
	for _, attempt := range record.Attempts {
		if strings.TrimSpace(attempt.AttemptID) == strings.TrimSpace(attemptID) {
			return attempt
		}
	}
	t.Fatalf("attempt %q not found in %#v", attemptID, record.Attempts)
	return scheduler.Attempt{}
}

func readinessFingerprint(result runtimeconfig.ReadinessResult) string {
	payload := struct {
		Status   runtimeconfig.ReadinessStatus    `json:"status"`
		Findings []runtimeconfig.ReadinessFinding `json:"findings"`
	}{
		Status:   result.Status,
		Findings: result.Findings,
	}
	blob, _ := json.Marshal(payload)
	return string(blob)
}
