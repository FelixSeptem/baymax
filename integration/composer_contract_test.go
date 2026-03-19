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

type composerEventCollector struct {
	events []types.Event
}

func (c *composerEventCollector) OnEvent(_ context.Context, ev types.Event) {
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

func TestComposerContractRunStreamSemanticEquivalence(t *testing.T) {
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX_A8_TEST"})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	model := fakes.NewModel([]fakes.ModelStep{
		{Response: types.ModelResponse{FinalAnswer: "composer-run"}},
	})
	model.SetStream([]types.ModelEvent{
		{Type: types.ModelEventTypeFinalAnswer, TextDelta: "composer-stream"},
	}, nil)

	collector := &composerEventCollector{}
	dispatcher := event.NewDispatcher(
		event.NewRuntimeRecorder(mgr),
		collector,
	)
	comp, err := composer.NewBuilder(model).
		WithRuntimeManager(mgr).
		WithEventHandler(dispatcherHandler{dispatcher: dispatcher}).
		Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}

	runReq := types.RunRequest{RunID: "run-a8-equivalence-run", Input: "ping-run"}
	streamReq := types.RunRequest{RunID: "run-a8-equivalence-stream", Input: "ping-stream"}
	if _, err := comp.Run(context.Background(), runReq, nil); err != nil {
		t.Fatalf("composer run failed: %v", err)
	}
	if _, err := comp.Stream(context.Background(), streamReq, nil); err != nil {
		t.Fatalf("composer stream failed: %v", err)
	}

	runRecord := findRunRecord(t, mgr.RecentRuns(10), runReq.RunID)
	streamRecord := findRunRecord(t, mgr.RecentRuns(10), streamReq.RunID)
	if runRecord.Status != streamRecord.Status {
		t.Fatalf("status mismatch run=%q stream=%q", runRecord.Status, streamRecord.Status)
	}
	if !runRecord.ComposerManaged || !streamRecord.ComposerManaged {
		t.Fatalf("composer marker mismatch run=%v stream=%v", runRecord.ComposerManaged, streamRecord.ComposerManaged)
	}
	if runRecord.SchedulerBackend != streamRecord.SchedulerBackend {
		t.Fatalf("scheduler backend mismatch run=%q stream=%q", runRecord.SchedulerBackend, streamRecord.SchedulerBackend)
	}
	if runRecord.SchedulerQueueTotal != streamRecord.SchedulerQueueTotal ||
		runRecord.SchedulerClaimTotal != streamRecord.SchedulerClaimTotal ||
		runRecord.SchedulerReclaimTotal != streamRecord.SchedulerReclaimTotal {
		t.Fatalf("scheduler aggregate mismatch run=%#v stream=%#v", runRecord, streamRecord)
	}
	if runRecord.SubagentChildTotal != streamRecord.SubagentChildTotal ||
		runRecord.SubagentChildFailed != streamRecord.SubagentChildFailed ||
		runRecord.SubagentBudgetRejectTotal != streamRecord.SubagentBudgetRejectTotal {
		t.Fatalf("subagent aggregate mismatch run=%#v stream=%#v", runRecord, streamRecord)
	}

	runFinishedCount := 0
	for _, ev := range collector.events {
		if ev.Type != "run.finished" {
			continue
		}
		runFinishedCount++
		if _, ok := ev.Payload["composer_managed"]; !ok {
			t.Fatalf("run.finished missing composer marker: %#v", ev.Payload)
		}
	}
	if runFinishedCount < 2 {
		t.Fatalf("run.finished count=%d, want >=2", runFinishedCount)
	}
}

func TestComposerContractDelayedChildRunStreamEquivalence(t *testing.T) {
	exec := func(stream bool, runID, taskID string) (runtimediag.RunRecord, error) {
		mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX_A13_TEST"})
		if err != nil {
			return runtimediag.RunRecord{}, err
		}
		defer func() { _ = mgr.Close() }()

		model := fakes.NewModel([]fakes.ModelStep{
			{Response: types.ModelResponse{FinalAnswer: "ok"}},
		})
		model.SetStream([]types.ModelEvent{
			{Type: types.ModelEventTypeFinalAnswer, TextDelta: "ok"},
		}, nil)

		dispatcher := event.NewDispatcher(event.NewRuntimeRecorder(mgr))
		comp, err := composer.NewBuilder(model).
			WithRuntimeManager(mgr).
			WithEventHandler(dispatcherHandler{dispatcher: dispatcher}).
			Build()
		if err != nil {
			return runtimediag.RunRecord{}, err
		}

		notBefore := time.Now().Add(90 * time.Millisecond)
		if _, err := comp.SpawnChild(context.Background(), composer.ChildDispatchRequest{
			Task: scheduler.Task{
				TaskID:    taskID,
				RunID:     runID,
				NotBefore: notBefore,
			},
		}); err != nil {
			return runtimediag.RunRecord{}, err
		}
		if _, ok, err := comp.Scheduler().Claim(context.Background(), "worker-delayed-a13"); err != nil {
			return runtimediag.RunRecord{}, err
		} else if ok {
			return runtimediag.RunRecord{}, errors.New("task claimed before not_before boundary")
		}
		time.Sleep(110 * time.Millisecond)
		claimed, ok, err := comp.Scheduler().Claim(context.Background(), "worker-delayed-a13")
		if err != nil || !ok {
			if err != nil {
				return runtimediag.RunRecord{}, err
			}
			return runtimediag.RunRecord{}, errors.New("task not claimable after not_before boundary")
		}
		if _, err := comp.CommitChildTerminal(context.Background(), scheduler.TerminalCommit{
			TaskID:      claimed.Record.Task.TaskID,
			AttemptID:   claimed.Attempt.AttemptID,
			Status:      scheduler.TaskStateSucceeded,
			Result:      map[string]any{"ok": true},
			CommittedAt: time.Now(),
		}); err != nil {
			return runtimediag.RunRecord{}, err
		}

		req := types.RunRequest{RunID: runID, Input: "emit-finished"}
		if stream {
			if _, err := comp.Stream(context.Background(), req, nil); err != nil {
				return runtimediag.RunRecord{}, err
			}
		} else {
			if _, err := comp.Run(context.Background(), req, nil); err != nil {
				return runtimediag.RunRecord{}, err
			}
		}
		return findRunRecord(t, mgr.RecentRuns(10), runID), nil
	}

	runRecord, err := exec(false, "run-a13-delayed-run", "task-a13-delayed-run")
	if err != nil {
		t.Fatalf("run path failed: %v", err)
	}
	streamRecord, err := exec(true, "run-a13-delayed-stream", "task-a13-delayed-stream")
	if err != nil {
		t.Fatalf("stream path failed: %v", err)
	}
	if runRecord.Status != streamRecord.Status {
		t.Fatalf("status mismatch run=%q stream=%q", runRecord.Status, streamRecord.Status)
	}
	if runRecord.SchedulerDelayedTaskTotal != streamRecord.SchedulerDelayedTaskTotal ||
		runRecord.SchedulerDelayedClaimTotal != streamRecord.SchedulerDelayedClaimTotal {
		t.Fatalf("delayed totals mismatch run=%#v stream=%#v", runRecord, streamRecord)
	}
	diff := runRecord.SchedulerDelayedWaitMsP95 - streamRecord.SchedulerDelayedWaitMsP95
	if diff < 0 {
		diff = -diff
	}
	if diff > 30 {
		t.Fatalf("delayed wait p95 mismatch beyond tolerance run=%d stream=%d diff=%d", runRecord.SchedulerDelayedWaitMsP95, streamRecord.SchedulerDelayedWaitMsP95, diff)
	}
	if runRecord.SchedulerDelayedTaskTotal != 1 || runRecord.SchedulerDelayedClaimTotal != 1 {
		t.Fatalf("unexpected delayed counters in run summary: %#v", runRecord)
	}
	if runRecord.SchedulerDelayedWaitMsP95 <= 0 {
		t.Fatalf("scheduler_delayed_wait_ms_p95=%d, want > 0", runRecord.SchedulerDelayedWaitMsP95)
	}
}

func TestComposerContractSchedulerFallbackToMemory(t *testing.T) {
	tmpDir := t.TempDir()
	blockedDir := filepath.Join(tmpDir, "blocked-dir")
	if err := os.WriteFile(blockedDir, []byte("x"), 0o600); err != nil {
		t.Fatalf("write blocked dir marker: %v", err)
	}
	cfgPath := filepath.Join(tmpDir, "runtime.yaml")
	cfg := strings.Join([]string{
		"reload:",
		"  enabled: false",
		"scheduler:",
		"  enabled: true",
		"  backend: file",
		"  path: " + filepath.ToSlash(filepath.Join(blockedDir, "scheduler-state.json")),
		"  lease_timeout: 400ms",
		"  heartbeat_interval: 100ms",
		"  queue_limit: 32",
		"  retry_max_attempts: 3",
		"subagent:",
		"  max_depth: 4",
		"  max_active_children: 8",
		"  child_timeout_budget: 3s",
		"",
	}, "\n")
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		FilePath:  cfgPath,
		EnvPrefix: "BAYMAX_A8_TEST",
	})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	model := fakes.NewModel([]fakes.ModelStep{
		{Response: types.ModelResponse{FinalAnswer: "ok"}},
	})
	model.SetStream([]types.ModelEvent{
		{Type: types.ModelEventTypeFinalAnswer, TextDelta: "ok"},
	}, nil)

	dispatcher := event.NewDispatcher(event.NewRuntimeRecorder(mgr))
	comp, err := composer.NewBuilder(model).
		WithRuntimeManager(mgr).
		WithEventHandler(dispatcherHandler{dispatcher: dispatcher}).
		Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}
	runID := "run-a8-fallback"
	if _, err := comp.Run(context.Background(), types.RunRequest{RunID: runID, Input: "fallback"}, nil); err != nil {
		t.Fatalf("composer run failed: %v", err)
	}

	record := findRunRecord(t, mgr.RecentRuns(10), runID)
	if !record.ComposerManaged {
		t.Fatalf("expected composer_managed=true, got %#v", record)
	}
	if record.SchedulerBackend != "memory" {
		t.Fatalf("scheduler backend=%q, want memory", record.SchedulerBackend)
	}
	if !record.SchedulerBackendFallback {
		t.Fatalf("scheduler fallback marker should be true: %#v", record)
	}
	if record.SchedulerBackendFallbackReason == "" {
		t.Fatalf("scheduler fallback reason should not be empty: %#v", record)
	}
}

func TestComposerContractTakeoverReplayIdempotency(t *testing.T) {
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX_A8_TEST"})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	model := fakes.NewModel([]fakes.ModelStep{
		{Response: types.ModelResponse{FinalAnswer: "ok"}},
	})
	model.SetStream([]types.ModelEvent{
		{Type: types.ModelEventTypeFinalAnswer, TextDelta: "ok"},
	}, nil)
	dispatcher := event.NewDispatcher(event.NewRuntimeRecorder(mgr))
	comp, err := composer.NewBuilder(model).
		WithRuntimeManager(mgr).
		WithEventHandler(dispatcherHandler{dispatcher: dispatcher}).
		Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}

	childReq := composer.ChildDispatchRequest{
		Task: scheduler.Task{
			TaskID: "task-a8-idempotent",
			RunID:  "run-a8-idempotent",
		},
		Target:               composer.ChildTargetLocal,
		ParentDepth:          0,
		ParentActiveChildren: 0,
		ChildTimeout:         300 * time.Millisecond,
		LocalRunner: composer.LocalChildRunnerFunc(func(ctx context.Context, task scheduler.Task) (map[string]any, error) {
			return map[string]any{"task_id": task.TaskID, "ok": true}, nil
		}),
	}

	out, err := comp.DispatchChild(context.Background(), childReq)
	if err != nil {
		t.Fatalf("dispatch child failed: %v", err)
	}
	if out.CommitMeta.Duplicate {
		t.Fatalf("first commit should not be duplicate: %#v", out.CommitMeta)
	}

	dup, err := comp.CommitChildTerminal(context.Background(), out.Commit)
	if err != nil {
		t.Fatalf("duplicate terminal commit failed: %v", err)
	}
	if !dup.Duplicate {
		t.Fatalf("second commit should be duplicate: %#v", dup)
	}

	if _, err := comp.Run(context.Background(), types.RunRequest{
		RunID: out.Record.Task.RunID,
		Input: "emit-finished",
	}, nil); err != nil {
		t.Fatalf("composer run failed: %v", err)
	}
	record := findRunRecord(t, mgr.RecentRuns(10), out.Record.Task.RunID)
	if record.SubagentChildTotal != 1 || record.SubagentChildFailed != 0 {
		t.Fatalf("subagent aggregate should not inflate under replay: %#v", record)
	}
}

func findRunRecord(t *testing.T, records []runtimediag.RunRecord, runID string) runtimediag.RunRecord {
	t.Helper()
	for _, rec := range records {
		if strings.TrimSpace(rec.RunID) == strings.TrimSpace(runID) {
			return rec
		}
	}
	t.Fatalf("run record %q not found in %#v", runID, records)
	return runtimediag.RunRecord{}
}
