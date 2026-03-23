package integration

import (
	"context"
	"os"
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
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
	runtimediag "github.com/FelixSeptem/baymax/runtime/diagnostics"
)

func TestTaskBoardControlContractMemoryFileParity(t *testing.T) {
	ctx := context.Background()
	memScheduler := newTaskBoardControlContractScheduler(t, scheduler.NewMemoryStore())
	seedTaskBoardControlContractFixture(t, memScheduler)

	snapshot, err := memScheduler.Snapshot(ctx)
	if err != nil {
		t.Fatalf("memory snapshot failed: %v", err)
	}

	fileStore, err := scheduler.NewFileStore(filepath.Join(t.TempDir(), "scheduler-task-board-control-state.json"))
	if err != nil {
		t.Fatalf("new file store failed: %v", err)
	}
	fileScheduler := newTaskBoardControlContractScheduler(t, fileStore)
	if err := fileScheduler.Restore(ctx, snapshot); err != nil {
		t.Fatalf("restore to file scheduler failed: %v", err)
	}

	requests := []scheduler.TaskBoardControlRequest{
		{
			TaskID:      "task-control-queued",
			Action:      string(scheduler.TaskBoardControlActionCancel),
			OperationID: "op-control-cancel-queued",
		},
		{
			TaskID:      "task-control-awaiting",
			Action:      string(scheduler.TaskBoardControlActionCancel),
			OperationID: "op-control-cancel-awaiting",
		},
		{
			TaskID:      "task-control-failed",
			Action:      string(scheduler.TaskBoardControlActionRetryTerminal),
			OperationID: "op-control-retry-failed",
		},
		{
			TaskID:      "task-control-dead-letter",
			Action:      string(scheduler.TaskBoardControlActionRetryTerminal),
			OperationID: "op-control-retry-dead-letter",
		},
		{
			TaskID:      "task-control-failed",
			Action:      string(scheduler.TaskBoardControlActionRetryTerminal),
			OperationID: "op-control-retry-failed",
		},
	}

	memSummary := applyControlSequenceAndCollectSummary(t, memScheduler, requests)
	fileSummary := applyControlSequenceAndCollectSummary(t, fileScheduler, requests)
	if !reflect.DeepEqual(memSummary, fileSummary) {
		t.Fatalf("memory/file manual control parity mismatch: memory=%#v file=%#v", memSummary, fileSummary)
	}
}

func TestTaskBoardControlContractRunStreamSemanticEquivalence(t *testing.T) {
	runRecord := executeTaskBoardControlRunSummary(t, false, "run-a39-control-run")
	streamRecord := executeTaskBoardControlRunSummary(t, true, "run-a39-control-stream")

	if runRecord.Status != streamRecord.Status {
		t.Fatalf("status mismatch run=%q stream=%q", runRecord.Status, streamRecord.Status)
	}
	if runRecord.TaskBoardManualControlTotal != streamRecord.TaskBoardManualControlTotal ||
		runRecord.TaskBoardManualControlSuccessTotal != streamRecord.TaskBoardManualControlSuccessTotal ||
		runRecord.TaskBoardManualControlRejectedTotal != streamRecord.TaskBoardManualControlRejectedTotal ||
		runRecord.TaskBoardManualControlDedupTotal != streamRecord.TaskBoardManualControlDedupTotal {
		t.Fatalf("manual control aggregate mismatch run=%#v stream=%#v", runRecord, streamRecord)
	}
	if !reflect.DeepEqual(runRecord.TaskBoardManualControlByAction, streamRecord.TaskBoardManualControlByAction) {
		t.Fatalf(
			"manual control action breakdown mismatch run=%#v stream=%#v",
			runRecord.TaskBoardManualControlByAction,
			streamRecord.TaskBoardManualControlByAction,
		)
	}
	if !reflect.DeepEqual(runRecord.TaskBoardManualControlByReason, streamRecord.TaskBoardManualControlByReason) {
		t.Fatalf(
			"manual control reason breakdown mismatch run=%#v stream=%#v",
			runRecord.TaskBoardManualControlByReason,
			streamRecord.TaskBoardManualControlByReason,
		)
	}
}

func TestTaskBoardControlContractDiagnosticsReplayIdempotent(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime-task-board-control-replay.yaml")
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(`
reload:
  enabled: false
`)), 0o600); err != nil {
		t.Fatalf("write config failed: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		FilePath:  cfgPath,
		EnvPrefix: "BAYMAX_A39_REPLAY",
	})
	if err != nil {
		t.Fatalf("new runtime manager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	recorder := event.NewRuntimeRecorder(mgr)
	runFinished := types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    "run.finished",
		RunID:   "run-a39-replay-idempotent",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":                          "success",
			"task_board_manual_control_total": 3,
			"task_board_manual_control_success_total":          2,
			"task_board_manual_control_rejected_total":         1,
			"task_board_manual_control_idempotent_dedup_total": 1,
			"task_board_manual_control_by_action": map[string]any{
				"cancel":         float64(1),
				"retry_terminal": float64(1),
			},
			"task_board_manual_control_by_reason": map[string]any{
				"scheduler.manual_cancel": float64(1),
				"scheduler.manual_retry":  float64(1),
			},
		},
	}
	recorder.OnEvent(context.Background(), runFinished)
	recorder.OnEvent(context.Background(), runFinished)

	items := mgr.RecentRuns(10)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	got := items[0]
	if got.TaskBoardManualControlTotal != 3 ||
		got.TaskBoardManualControlSuccessTotal != 2 ||
		got.TaskBoardManualControlRejectedTotal != 1 ||
		got.TaskBoardManualControlDedupTotal != 1 {
		t.Fatalf("manual control counters should stay stable under replay, got %#v", got)
	}
	if got.TaskBoardManualControlByAction["cancel"] != 1 || got.TaskBoardManualControlByAction["retry_terminal"] != 1 {
		t.Fatalf("manual control action breakdown mismatch under replay: %#v", got.TaskBoardManualControlByAction)
	}
	if got.TaskBoardManualControlByReason["scheduler.manual_cancel"] != 1 ||
		got.TaskBoardManualControlByReason["scheduler.manual_retry"] != 1 {
		t.Fatalf("manual control reason breakdown mismatch under replay: %#v", got.TaskBoardManualControlByReason)
	}
}

func newTaskBoardControlContractScheduler(t *testing.T, store scheduler.QueueStore) *scheduler.Scheduler {
	t.Helper()
	s, err := scheduler.New(
		store,
		scheduler.WithLeaseTimeout(2*time.Second),
		scheduler.WithTaskBoardControl(scheduler.TaskBoardControlConfig{
			Enabled:               true,
			MaxManualRetryPerTask: 3,
		}),
		scheduler.WithGovernance(scheduler.GovernanceConfig{
			QoS: scheduler.QoSModeFIFO,
			Fairness: scheduler.FairnessConfig{
				MaxConsecutiveClaimsPerPriority: 3,
			},
			DLQ: scheduler.DLQConfig{
				Enabled: true,
			},
			Backoff: scheduler.RetryBackoffConfig{
				Enabled:     false,
				Initial:     10 * time.Millisecond,
				Max:         20 * time.Millisecond,
				Multiplier:  2,
				JitterRatio: 0,
			},
		}),
	)
	if err != nil {
		t.Fatalf("new task-board-control scheduler failed: %v", err)
	}
	return s
}

func seedTaskBoardControlContractFixture(t *testing.T, s *scheduler.Scheduler) {
	t.Helper()
	ctx := context.Background()

	if _, err := s.Enqueue(ctx, scheduler.Task{TaskID: "task-control-awaiting", RunID: "run-control-contract"}); err != nil {
		t.Fatalf("enqueue awaiting fixture failed: %v", err)
	}
	claimedAwaiting, ok, err := s.Claim(ctx, "worker-control-awaiting")
	if err != nil || !ok {
		t.Fatalf("claim awaiting fixture failed: ok=%v err=%v", ok, err)
	}
	if _, err := s.MarkAwaitingReport(ctx, claimedAwaiting.Record.Task.TaskID, claimedAwaiting.Attempt.AttemptID, "remote-control-awaiting"); err != nil {
		t.Fatalf("mark awaiting fixture failed: %v", err)
	}

	if _, err := s.Enqueue(ctx, scheduler.Task{TaskID: "task-control-failed", RunID: "run-control-contract"}); err != nil {
		t.Fatalf("enqueue failed fixture failed: %v", err)
	}
	claimedFailed, ok, err := s.Claim(ctx, "worker-control-failed")
	if err != nil || !ok {
		t.Fatalf("claim failed fixture failed: ok=%v err=%v", ok, err)
	}
	if _, err := s.Fail(ctx, scheduler.TerminalCommit{
		TaskID:       claimedFailed.Record.Task.TaskID,
		AttemptID:    claimedFailed.Attempt.AttemptID,
		Status:       scheduler.TaskStateFailed,
		ErrorMessage: "fixture-failed",
		CommittedAt:  time.Now(),
	}); err != nil {
		t.Fatalf("fail failed fixture failed: %v", err)
	}

	if _, err := s.Enqueue(ctx, scheduler.Task{
		TaskID:      "task-control-dead-letter",
		RunID:       "run-control-contract",
		MaxAttempts: 1,
	}); err != nil {
		t.Fatalf("enqueue dead_letter fixture failed: %v", err)
	}
	claimedDead, ok, err := s.Claim(ctx, "worker-control-dead")
	if err != nil || !ok {
		t.Fatalf("claim dead_letter fixture failed: ok=%v err=%v", ok, err)
	}
	if _, err := s.Requeue(ctx, claimedDead.Record.Task.TaskID, "fixture_retry_exhausted"); err != nil {
		t.Fatalf("requeue dead_letter fixture failed: %v", err)
	}

	if _, err := s.Enqueue(ctx, scheduler.Task{TaskID: "task-control-queued", RunID: "run-control-contract"}); err != nil {
		t.Fatalf("enqueue queued fixture failed: %v", err)
	}
}

type taskBoardControlSemanticSummary struct {
	Tasks                      map[string]taskBoardControlTaskSummary
	ManualControlTotal         int
	ManualControlSuccessTotal  int
	ManualControlRejectedTotal int
	ManualControlDedupTotal    int
	ManualControlByAction      map[string]int
	ManualControlByReason      map[string]int
}

type taskBoardControlTaskSummary struct {
	State            scheduler.TaskState
	CurrentAttemptID string
	ManualRetryCount int
}

func applyControlSequenceAndCollectSummary(
	t *testing.T,
	s *scheduler.Scheduler,
	requests []scheduler.TaskBoardControlRequest,
) taskBoardControlSemanticSummary {
	t.Helper()
	ctx := context.Background()
	for _, req := range requests {
		if _, err := s.ControlTask(ctx, req); err != nil {
			t.Fatalf("apply control request failed: req=%#v err=%v", req, err)
		}
	}

	taskIDs := []string{
		"task-control-queued",
		"task-control-awaiting",
		"task-control-failed",
		"task-control-dead-letter",
	}
	tasks := make(map[string]taskBoardControlTaskSummary, len(taskIDs))
	for _, taskID := range taskIDs {
		record, ok, err := s.Get(ctx, taskID)
		if err != nil || !ok {
			t.Fatalf("get task summary failed: task=%s ok=%v err=%v", taskID, ok, err)
		}
		tasks[taskID] = taskBoardControlTaskSummary{
			State:            record.State,
			CurrentAttemptID: strings.TrimSpace(record.CurrentAttempt),
			ManualRetryCount: record.ManualRetryCount,
		}
	}

	stats, err := s.Stats(ctx)
	if err != nil {
		t.Fatalf("stats failed: %v", err)
	}
	return taskBoardControlSemanticSummary{
		Tasks:                      tasks,
		ManualControlTotal:         stats.TaskBoardManualControlTotal,
		ManualControlSuccessTotal:  stats.TaskBoardManualControlSuccessTotal,
		ManualControlRejectedTotal: stats.TaskBoardManualControlRejectedTotal,
		ManualControlDedupTotal:    stats.TaskBoardManualControlDedupTotal,
		ManualControlByAction:      copyStringIntMap(stats.TaskBoardManualControlByAction),
		ManualControlByReason:      copyStringIntMap(stats.TaskBoardManualControlByReason),
	}
}

func executeTaskBoardControlRunSummary(t *testing.T, stream bool, runID string) runtimediag.RunRecord {
	t.Helper()
	cfgPath := filepath.Join(t.TempDir(), "runtime-task-board-control.yaml")
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(`
reload:
  enabled: false
scheduler:
  enabled: true
  task_board:
    control:
      enabled: true
      max_manual_retry_per_task: 3
`)), 0o600); err != nil {
		t.Fatalf("write config failed: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		FilePath:  cfgPath,
		EnvPrefix: "BAYMAX_A39_RUNSTREAM",
	})
	if err != nil {
		t.Fatalf("new runtime manager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	model := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	model.SetStream([]types.ModelEvent{{Type: types.ModelEventTypeFinalAnswer, TextDelta: "ok"}}, nil)

	dispatcher := event.NewDispatcher(event.NewRuntimeRecorder(mgr))
	comp, err := composer.NewBuilder(model).
		WithRuntimeManager(mgr).
		WithEventHandler(dispatcherHandler{dispatcher: dispatcher}).
		Build()
	if err != nil {
		t.Fatalf("build composer failed: %v", err)
	}
	s := comp.Scheduler()

	if _, err := s.Enqueue(context.Background(), scheduler.Task{
		TaskID: "runstream-control-queued",
		RunID:  runID,
	}); err != nil {
		t.Fatalf("enqueue runstream queued failed: %v", err)
	}
	if _, err := s.ControlTask(context.Background(), scheduler.TaskBoardControlRequest{
		TaskID:      "runstream-control-queued",
		Action:      string(scheduler.TaskBoardControlActionCancel),
		OperationID: runID + "-op-cancel",
	}); err != nil {
		t.Fatalf("control cancel failed: %v", err)
	}

	if _, err := s.Enqueue(context.Background(), scheduler.Task{
		TaskID: "runstream-control-failed",
		RunID:  runID,
	}); err != nil {
		t.Fatalf("enqueue runstream failed target failed: %v", err)
	}
	claimed, ok, err := s.Claim(context.Background(), "worker-runstream")
	if err != nil || !ok {
		t.Fatalf("claim runstream failed target failed: ok=%v err=%v", ok, err)
	}
	if _, err := s.Fail(context.Background(), scheduler.TerminalCommit{
		TaskID:       claimed.Record.Task.TaskID,
		AttemptID:    claimed.Attempt.AttemptID,
		Status:       scheduler.TaskStateFailed,
		ErrorMessage: "runstream-failed",
		CommittedAt:  time.Now(),
	}); err != nil {
		t.Fatalf("fail runstream failed target failed: %v", err)
	}
	if _, err := s.ControlTask(context.Background(), scheduler.TaskBoardControlRequest{
		TaskID:      "runstream-control-failed",
		Action:      string(scheduler.TaskBoardControlActionRetryTerminal),
		OperationID: runID + "-op-retry",
	}); err != nil {
		t.Fatalf("control retry failed: %v", err)
	}
	if _, err := s.ControlTask(context.Background(), scheduler.TaskBoardControlRequest{
		TaskID:      "runstream-control-failed",
		Action:      string(scheduler.TaskBoardControlActionRetryTerminal),
		OperationID: runID + "-op-retry",
	}); err != nil {
		t.Fatalf("control retry dedup failed: %v", err)
	}

	req := types.RunRequest{RunID: runID, Input: "emit-run-finished-for-a39"}
	if stream {
		if _, err := comp.Stream(context.Background(), req, nil); err != nil {
			t.Fatalf("composer stream failed: %v", err)
		}
	} else {
		if _, err := comp.Run(context.Background(), req, nil); err != nil {
			t.Fatalf("composer run failed: %v", err)
		}
	}

	return findRunRecord(t, mgr.RecentRuns(10), runID)
}

func copyStringIntMap(in map[string]int) map[string]int {
	if len(in) == 0 {
		return map[string]int{}
	}
	out := make(map[string]int, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
