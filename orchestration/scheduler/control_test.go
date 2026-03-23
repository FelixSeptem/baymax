package scheduler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

func TestTaskBoardControlValidationAndStateMatrixFailFast(t *testing.T) {
	ctx := context.Background()
	s := newTaskBoardControlTestScheduler(t, TaskBoardControlConfig{
		Enabled:               true,
		MaxManualRetryPerTask: 3,
	})

	if _, err := s.Enqueue(ctx, Task{TaskID: "control-running", RunID: "run-control"}); err != nil {
		t.Fatalf("enqueue running failed: %v", err)
	}
	claimed, ok, err := s.Claim(ctx, "worker-control")
	if err != nil || !ok {
		t.Fatalf("claim running failed: ok=%v err=%v", ok, err)
	}
	if claimed.Record.Task.TaskID != "control-running" {
		t.Fatalf("claimed task = %q, want control-running", claimed.Record.Task.TaskID)
	}

	if _, err := s.Enqueue(ctx, Task{TaskID: "control-queued", RunID: "run-control"}); err != nil {
		t.Fatalf("enqueue queued failed: %v", err)
	}
	if _, err := s.ControlTask(ctx, TaskBoardControlRequest{
		TaskID:      "control-queued",
		Action:      string(TaskBoardControlActionCancel),
		OperationID: "",
	}); !errors.Is(err, ErrTaskBoardControlOperationIDRequired) {
		t.Fatalf("missing operation_id error = %v, want %v", err, ErrTaskBoardControlOperationIDRequired)
	}
	if _, err := s.ControlTask(ctx, TaskBoardControlRequest{
		TaskID:      "control-queued",
		Action:      "pause",
		OperationID: "op-invalid-action",
	}); !errors.Is(err, ErrTaskBoardControlActionInvalid) {
		t.Fatalf("invalid action error = %v, want %v", err, ErrTaskBoardControlActionInvalid)
	}

	if _, err := s.ControlTask(ctx, TaskBoardControlRequest{
		TaskID:      "control-running",
		Action:      string(TaskBoardControlActionCancel),
		OperationID: "op-cancel-running",
	}); !errors.Is(err, ErrTaskBoardControlStateConflict) {
		t.Fatalf("cancel running error = %v, want %v", err, ErrTaskBoardControlStateConflict)
	}

	if _, err := s.ControlTask(ctx, TaskBoardControlRequest{
		TaskID:      "control-queued",
		Action:      string(TaskBoardControlActionRetryTerminal),
		OperationID: "op-retry-queued",
	}); !errors.Is(err, ErrTaskBoardControlStateConflict) {
		t.Fatalf("retry queued error = %v, want %v", err, ErrTaskBoardControlStateConflict)
	}
}

func TestTaskBoardControlCancelSemanticsQueuedAndAwaitingReport(t *testing.T) {
	ctx := context.Background()
	s := newTaskBoardControlTestScheduler(t, TaskBoardControlConfig{
		Enabled:               true,
		MaxManualRetryPerTask: 3,
	})

	if _, err := s.Enqueue(ctx, Task{TaskID: "cancel-queued", RunID: "run-cancel"}); err != nil {
		t.Fatalf("enqueue cancel-queued failed: %v", err)
	}
	cancelQueued, err := s.ControlTask(ctx, TaskBoardControlRequest{
		TaskID:      "cancel-queued",
		Action:      string(TaskBoardControlActionCancel),
		OperationID: "op-cancel-queued",
	})
	if err != nil {
		t.Fatalf("cancel queued failed: %v", err)
	}
	if !cancelQueued.Applied || cancelQueued.Action != TaskBoardControlActionCancel || cancelQueued.Reason != ReasonManualCancel {
		t.Fatalf("cancel queued result mismatch: %#v", cancelQueued)
	}
	recordQueued, ok, err := s.Get(ctx, "cancel-queued")
	if err != nil || !ok {
		t.Fatalf("get cancel-queued failed: ok=%v err=%v", ok, err)
	}
	if recordQueued.State != TaskStateFailed || recordQueued.CurrentAttempt != "" {
		t.Fatalf("cancel queued state mismatch: %#v", recordQueued)
	}
	if _, ok, err := s.Claim(ctx, "worker-cancel-queued"); err != nil || ok {
		t.Fatalf("canceled queued task should be non-claimable: ok=%v err=%v", ok, err)
	}

	if _, err := s.Enqueue(ctx, Task{TaskID: "cancel-awaiting", RunID: "run-cancel"}); err != nil {
		t.Fatalf("enqueue cancel-awaiting failed: %v", err)
	}
	claimed, ok, err := s.Claim(ctx, "worker-cancel-awaiting")
	if err != nil || !ok {
		t.Fatalf("claim cancel-awaiting failed: ok=%v err=%v", ok, err)
	}
	if _, err := s.MarkAwaitingReport(ctx, claimed.Record.Task.TaskID, claimed.Attempt.AttemptID, "remote-cancel-awaiting"); err != nil {
		t.Fatalf("mark awaiting_report failed: %v", err)
	}

	cancelAwaiting, err := s.ControlTask(ctx, TaskBoardControlRequest{
		TaskID:      "cancel-awaiting",
		Action:      string(TaskBoardControlActionCancel),
		OperationID: "op-cancel-awaiting",
	})
	if err != nil {
		t.Fatalf("cancel awaiting_report failed: %v", err)
	}
	if !cancelAwaiting.Applied || cancelAwaiting.Action != TaskBoardControlActionCancel || cancelAwaiting.Reason != ReasonManualCancel {
		t.Fatalf("cancel awaiting result mismatch: %#v", cancelAwaiting)
	}
	recordAwaiting, ok, err := s.Get(ctx, "cancel-awaiting")
	if err != nil || !ok {
		t.Fatalf("get cancel-awaiting failed: ok=%v err=%v", ok, err)
	}
	if recordAwaiting.State != TaskStateFailed {
		t.Fatalf("cancel awaiting state = %q, want failed", recordAwaiting.State)
	}
	if !recordAwaiting.AwaitingReportSince.IsZero() || !recordAwaiting.ReportTimeoutAt.IsZero() || recordAwaiting.CurrentAttempt != "" {
		t.Fatalf("cancel awaiting cleanup mismatch: %#v", recordAwaiting)
	}
	lastAttempt := latestAttempt(recordAwaiting)
	if lastAttempt.Status != AttemptStatusFailed || lastAttempt.TerminalAt.IsZero() {
		t.Fatalf("cancel awaiting attempt terminalization mismatch: %#v", lastAttempt)
	}
	if _, ok, err := s.Claim(ctx, "worker-cancel-awaiting-2"); err != nil || ok {
		t.Fatalf("canceled awaiting task should be non-claimable: ok=%v err=%v", ok, err)
	}
}

func TestTaskBoardControlRetryTerminalAndManualRetryBudget(t *testing.T) {
	ctx := context.Background()
	s := newTaskBoardControlTestScheduler(t, TaskBoardControlConfig{
		Enabled:               true,
		MaxManualRetryPerTask: 2,
	})

	if _, err := s.Enqueue(ctx, Task{TaskID: "retry-terminal", RunID: "run-retry"}); err != nil {
		t.Fatalf("enqueue retry-terminal failed: %v", err)
	}
	claim1, ok, err := s.Claim(ctx, "worker-retry-1")
	if err != nil || !ok {
		t.Fatalf("claim retry-terminal #1 failed: ok=%v err=%v", ok, err)
	}
	if _, err := s.Fail(ctx, TerminalCommit{
		TaskID:       claim1.Record.Task.TaskID,
		AttemptID:    claim1.Attempt.AttemptID,
		Status:       TaskStateFailed,
		ErrorMessage: "first fail",
		CommittedAt:  time.Now(),
	}); err != nil {
		t.Fatalf("fail retry-terminal #1 failed: %v", err)
	}

	retry1, err := s.ControlTask(ctx, TaskBoardControlRequest{
		TaskID:      "retry-terminal",
		Action:      string(TaskBoardControlActionRetryTerminal),
		OperationID: "op-retry-1",
	})
	if err != nil {
		t.Fatalf("retry terminal #1 failed: %v", err)
	}
	if !retry1.Applied || retry1.Action != TaskBoardControlActionRetryTerminal || retry1.Reason != ReasonManualRetry {
		t.Fatalf("retry terminal #1 result mismatch: %#v", retry1)
	}
	record, ok, err := s.Get(ctx, "retry-terminal")
	if err != nil || !ok {
		t.Fatalf("get retry-terminal after retry#1 failed: ok=%v err=%v", ok, err)
	}
	if record.State != TaskStateQueued || record.ManualRetryCount != 1 {
		t.Fatalf("retry terminal #1 state mismatch: %#v", record)
	}

	claim2, ok, err := s.Claim(ctx, "worker-retry-2")
	if err != nil || !ok {
		t.Fatalf("claim retry-terminal #2 failed: ok=%v err=%v", ok, err)
	}
	if _, err := s.Fail(ctx, TerminalCommit{
		TaskID:       claim2.Record.Task.TaskID,
		AttemptID:    claim2.Attempt.AttemptID,
		Status:       TaskStateFailed,
		ErrorMessage: "second fail",
		CommittedAt:  time.Now(),
	}); err != nil {
		t.Fatalf("fail retry-terminal #2 failed: %v", err)
	}
	if _, err := s.ControlTask(ctx, TaskBoardControlRequest{
		TaskID:      "retry-terminal",
		Action:      string(TaskBoardControlActionRetryTerminal),
		OperationID: "op-retry-2",
	}); err != nil {
		t.Fatalf("retry terminal #2 failed: %v", err)
	}
	record, ok, err = s.Get(ctx, "retry-terminal")
	if err != nil || !ok {
		t.Fatalf("get retry-terminal after retry#2 failed: ok=%v err=%v", ok, err)
	}
	if record.State != TaskStateQueued || record.ManualRetryCount != 2 {
		t.Fatalf("retry terminal #2 state mismatch: %#v", record)
	}

	claim3, ok, err := s.Claim(ctx, "worker-retry-3")
	if err != nil || !ok {
		t.Fatalf("claim retry-terminal #3 failed: ok=%v err=%v", ok, err)
	}
	if _, err := s.Fail(ctx, TerminalCommit{
		TaskID:       claim3.Record.Task.TaskID,
		AttemptID:    claim3.Attempt.AttemptID,
		Status:       TaskStateFailed,
		ErrorMessage: "third fail",
		CommittedAt:  time.Now(),
	}); err != nil {
		t.Fatalf("fail retry-terminal #3 failed: %v", err)
	}
	if _, err := s.ControlTask(ctx, TaskBoardControlRequest{
		TaskID:      "retry-terminal",
		Action:      string(TaskBoardControlActionRetryTerminal),
		OperationID: "op-retry-3-exhausted",
	}); !errors.Is(err, ErrTaskBoardControlRetryBudgetExceeded) {
		t.Fatalf("retry budget exhausted error = %v, want %v", err, ErrTaskBoardControlRetryBudgetExceeded)
	}
}

func TestTaskBoardControlOperationIDIdempotencyAndReplayStability(t *testing.T) {
	ctx := context.Background()
	s := newTaskBoardControlTestScheduler(t, TaskBoardControlConfig{
		Enabled:               true,
		MaxManualRetryPerTask: 3,
	})
	if _, err := s.Enqueue(ctx, Task{TaskID: "idempotent-cancel", RunID: "run-idempotent-cancel"}); err != nil {
		t.Fatalf("enqueue idempotent-cancel failed: %v", err)
	}

	first, err := s.ControlTask(ctx, TaskBoardControlRequest{
		TaskID:      "idempotent-cancel",
		Action:      string(TaskBoardControlActionCancel),
		OperationID: "op-idempotent-cancel",
	})
	if err != nil {
		t.Fatalf("first control failed: %v", err)
	}
	if !first.Applied || first.Deduplicated {
		t.Fatalf("first control result mismatch: %#v", first)
	}
	second, err := s.ControlTask(ctx, TaskBoardControlRequest{
		TaskID:      "idempotent-cancel",
		Action:      string(TaskBoardControlActionCancel),
		OperationID: "op-idempotent-cancel",
	})
	if err != nil {
		t.Fatalf("second control (dedup) failed: %v", err)
	}
	if !second.Applied || !second.Deduplicated {
		t.Fatalf("second control should be deduplicated: %#v", second)
	}
	stats, err := s.Stats(ctx)
	if err != nil {
		t.Fatalf("stats failed: %v", err)
	}
	if stats.TaskBoardManualControlTotal != 1 ||
		stats.TaskBoardManualControlSuccessTotal != 1 ||
		stats.TaskBoardManualControlRejectedTotal != 0 ||
		stats.TaskBoardManualControlDedupTotal != 1 {
		t.Fatalf("manual control stats mismatch under dedup replay: %#v", stats)
	}

	if _, err := s.ControlTask(ctx, TaskBoardControlRequest{
		TaskID:      "idempotent-cancel",
		Action:      string(TaskBoardControlActionRetryTerminal),
		OperationID: "op-idempotent-cancel",
	}); !errors.Is(err, ErrTaskBoardControlOperationConflict) {
		t.Fatalf("operation conflict error = %v, want %v", err, ErrTaskBoardControlOperationConflict)
	}
	stats, err = s.Stats(ctx)
	if err != nil {
		t.Fatalf("stats after operation conflict failed: %v", err)
	}
	if stats.TaskBoardManualControlRejectedTotal != 1 {
		t.Fatalf("rejected total = %d, want 1", stats.TaskBoardManualControlRejectedTotal)
	}
}

func TestTaskBoardControlTimelineCanonicalReasons(t *testing.T) {
	ctx := context.Background()
	collector := &testTimelineCollector{}
	s := newTaskBoardControlTestScheduler(t, TaskBoardControlConfig{
		Enabled:               true,
		MaxManualRetryPerTask: 3,
	}, WithTimelineEmitter(collector))

	if _, err := s.Enqueue(ctx, Task{TaskID: "timeline-cancel", RunID: "run-timeline"}); err != nil {
		t.Fatalf("enqueue timeline-cancel failed: %v", err)
	}
	if _, err := s.ControlTask(ctx, TaskBoardControlRequest{
		TaskID:      "timeline-cancel",
		Action:      string(TaskBoardControlActionCancel),
		OperationID: "op-timeline-cancel",
	}); err != nil {
		t.Fatalf("manual cancel failed: %v", err)
	}

	if _, err := s.Enqueue(ctx, Task{TaskID: "timeline-retry", RunID: "run-timeline"}); err != nil {
		t.Fatalf("enqueue timeline-retry failed: %v", err)
	}
	claimed, ok, err := s.Claim(ctx, "worker-timeline-retry")
	if err != nil || !ok {
		t.Fatalf("claim timeline-retry failed: ok=%v err=%v", ok, err)
	}
	if _, err := s.Fail(ctx, TerminalCommit{
		TaskID:       claimed.Record.Task.TaskID,
		AttemptID:    claimed.Attempt.AttemptID,
		Status:       TaskStateFailed,
		ErrorMessage: "for retry",
		CommittedAt:  time.Now(),
	}); err != nil {
		t.Fatalf("fail timeline-retry failed: %v", err)
	}
	if _, err := s.ControlTask(ctx, TaskBoardControlRequest{
		TaskID:      "timeline-retry",
		Action:      string(TaskBoardControlActionRetryTerminal),
		OperationID: "op-timeline-retry",
	}); err != nil {
		t.Fatalf("manual retry failed: %v", err)
	}

	if mapped, ok := CanonicalReason(ReasonManualCancel); !ok || mapped != ReasonManualCancel {
		t.Fatalf("canonical reason manual_cancel mismatch: mapped=%q ok=%v", mapped, ok)
	}
	if mapped, ok := CanonicalReason(ReasonManualRetry); !ok || mapped != ReasonManualRetry {
		t.Fatalf("canonical reason manual_retry mismatch: mapped=%q ok=%v", mapped, ok)
	}

	reasons := map[string]int{}
	for _, ev := range collector.events {
		if ev.Type != types.EventTypeActionTimeline {
			continue
		}
		reason, _ := ev.Payload["reason"].(string)
		reasons[reason]++
	}
	if reasons[ReasonManualCancel] != 1 || reasons[ReasonManualRetry] != 1 {
		t.Fatalf("manual control timeline reason counts mismatch: %#v", reasons)
	}
}

func newTaskBoardControlTestScheduler(t *testing.T, control TaskBoardControlConfig, extra ...Option) *Scheduler {
	t.Helper()
	opts := make([]Option, 0, 3+len(extra))
	opts = append(opts,
		WithLeaseTimeout(2*time.Second),
		WithTaskBoardControl(control),
		WithGovernance(GovernanceConfig{
			QoS: QoSModeFIFO,
			Fairness: FairnessConfig{
				MaxConsecutiveClaimsPerPriority: 3,
			},
			DLQ: DLQConfig{
				Enabled: true,
			},
			Backoff: RetryBackoffConfig{
				Enabled:     false,
				Initial:     20 * time.Millisecond,
				Max:         40 * time.Millisecond,
				Multiplier:  2,
				JitterRatio: 0,
			},
		}),
	)
	opts = append(opts, extra...)
	s, err := New(NewMemoryStore(), opts...)
	if err != nil {
		t.Fatalf("new task board control scheduler failed: %v", err)
	}
	return s
}
