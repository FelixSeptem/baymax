package composer

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
