package scheduler

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/a2a"
)

type fakeReconcilePollClient struct {
	statusFn func(context.Context, string) (a2a.TaskRecord, error)
	resultFn func(context.Context, string) (a2a.TaskRecord, error)
}

func (c fakeReconcilePollClient) Status(ctx context.Context, taskID string) (a2a.TaskRecord, error) {
	if c.statusFn != nil {
		return c.statusFn(ctx, taskID)
	}
	return a2a.TaskRecord{}, nil
}

func (c fakeReconcilePollClient) Result(ctx context.Context, taskID string) (a2a.TaskRecord, error) {
	if c.resultFn != nil {
		return c.resultFn(ctx, taskID)
	}
	return a2a.TaskRecord{}, nil
}

func TestSchedulerReconcileBoundedBatchAndPollTerminalCommit(t *testing.T) {
	s, err := New(
		NewMemoryStore(),
		WithLeaseTimeout(2*time.Second),
		WithAsyncAwait(AsyncAwaitConfig{
			ReportTimeout:    5 * time.Minute,
			LateReportPolicy: AsyncLateReportPolicyDropAndRecord,
			TimeoutTerminal:  TaskStateFailed,
			Reconcile: AsyncAwaitReconcileConfig{
				Enabled:        true,
				Interval:       5 * time.Second,
				BatchSize:      2,
				JitterRatio:    0,
				NotFoundPolicy: AsyncReconcileNotFoundKeepTimeout,
			},
		}),
	)
	if err != nil {
		t.Fatalf("new scheduler failed: %v", err)
	}
	ctx := context.Background()
	taskIDs := []string{"task-a32-batch-1", "task-a32-batch-2", "task-a32-batch-3"}
	for _, taskID := range taskIDs {
		claimed := mustAwaitingTask(t, s, taskID, "run-"+taskID, "remote-"+taskID)
		if claimed.Record.State == "" {
			t.Fatal("unexpected empty claimed state")
		}
	}
	poller := fakeReconcilePollClient{
		statusFn: func(_ context.Context, remoteTaskID string) (a2a.TaskRecord, error) {
			return a2a.TaskRecord{
				TaskID: remoteTaskID,
				Status: a2a.StatusSucceeded,
			}, nil
		},
		resultFn: func(_ context.Context, remoteTaskID string) (a2a.TaskRecord, error) {
			return a2a.TaskRecord{
				TaskID: remoteTaskID,
				Status: a2a.StatusSucceeded,
				Result: map[string]any{"remote_task_id": remoteTaskID},
			}, nil
		},
	}

	cycle, err := s.ReconcileAwaitingReports(ctx, poller)
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}
	if cycle.PollTotal != 2 || cycle.TerminalByPoll != 2 || cycle.ErrorTotal != 0 {
		t.Fatalf("unexpected reconcile cycle stats: %#v", cycle)
	}

	for _, taskID := range taskIDs[:2] {
		record, found, getErr := s.Get(ctx, taskID)
		if getErr != nil || !found {
			t.Fatalf("get task %q failed: found=%v err=%v", taskID, found, getErr)
		}
		if record.State != TaskStateSucceeded {
			t.Fatalf("task %q state=%q, want succeeded", taskID, record.State)
		}
		if record.ResolutionSource != AsyncResolutionSourceReconcilePoll {
			t.Fatalf("task %q resolution_source=%q, want %q", taskID, record.ResolutionSource, AsyncResolutionSourceReconcilePoll)
		}
	}
	last, found, err := s.Get(ctx, taskIDs[2])
	if err != nil || !found {
		t.Fatalf("get task %q failed: found=%v err=%v", taskIDs[2], found, err)
	}
	if last.State != TaskStateAwaitingReport {
		t.Fatalf("task %q state=%q, want awaiting_report", taskIDs[2], last.State)
	}

	stats, err := s.Stats(ctx)
	if err != nil {
		t.Fatalf("stats failed: %v", err)
	}
	if stats.AsyncReconcilePollTotal != 2 || stats.AsyncReconcileTerminalByPollTotal != 2 {
		t.Fatalf("unexpected reconcile aggregates: %#v", stats)
	}
}

func TestSchedulerReconcileNotFoundKeepsAwaitingUntilTimeout(t *testing.T) {
	s, err := New(
		NewMemoryStore(),
		WithLeaseTimeout(2*time.Second),
		WithAsyncAwait(AsyncAwaitConfig{
			ReportTimeout:    40 * time.Millisecond,
			LateReportPolicy: AsyncLateReportPolicyDropAndRecord,
			TimeoutTerminal:  TaskStateFailed,
			Reconcile: AsyncAwaitReconcileConfig{
				Enabled:        true,
				Interval:       5 * time.Second,
				BatchSize:      8,
				JitterRatio:    0,
				NotFoundPolicy: AsyncReconcileNotFoundKeepTimeout,
			},
		}),
	)
	if err != nil {
		t.Fatalf("new scheduler failed: %v", err)
	}
	ctx := context.Background()
	taskID := "task-a32-not-found"
	mustAwaitingTask(t, s, taskID, "run-a32-not-found", "remote-a32-not-found")

	poller := fakeReconcilePollClient{
		statusFn: func(_ context.Context, remoteTaskID string) (a2a.TaskRecord, error) {
			return a2a.TaskRecord{}, fmt.Errorf("a2a task %q not found", remoteTaskID)
		},
	}
	cycle, err := s.ReconcileAwaitingReports(ctx, poller)
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}
	if cycle.PollTotal != 1 || cycle.ErrorTotal != 0 {
		t.Fatalf("unexpected reconcile cycle stats: %#v", cycle)
	}

	awaiting, found, err := s.Get(ctx, taskID)
	if err != nil || !found {
		t.Fatalf("get awaiting task failed: found=%v err=%v", found, err)
	}
	if awaiting.State != TaskStateAwaitingReport {
		t.Fatalf("state=%q, want awaiting_report", awaiting.State)
	}

	time.Sleep(60 * time.Millisecond)
	expired, err := s.ExpireLeases(ctx)
	if err != nil {
		t.Fatalf("expire leases failed: %v", err)
	}
	if len(expired) != 1 {
		t.Fatalf("expired len=%d, want 1", len(expired))
	}
	if expired[0].Record.State != TaskStateFailed {
		t.Fatalf("timeout terminal state=%q, want failed", expired[0].Record.State)
	}
	if expired[0].Record.ResolutionSource != AsyncResolutionSourceTimeout {
		t.Fatalf("timeout resolution source=%q, want %q", expired[0].Record.ResolutionSource, AsyncResolutionSourceTimeout)
	}
}

func TestSchedulerAsyncTerminalArbitrationFirstTerminalWinsAndConflictOnce(t *testing.T) {
	s, err := New(
		NewMemoryStore(),
		WithLeaseTimeout(2*time.Second),
		WithAsyncAwait(AsyncAwaitConfig{
			ReportTimeout:    15 * time.Minute,
			LateReportPolicy: AsyncLateReportPolicyDropAndRecord,
			TimeoutTerminal:  TaskStateFailed,
			Reconcile: AsyncAwaitReconcileConfig{
				Enabled:        true,
				Interval:       5 * time.Second,
				BatchSize:      8,
				JitterRatio:    0,
				NotFoundPolicy: AsyncReconcileNotFoundKeepTimeout,
			},
		}),
	)
	if err != nil {
		t.Fatalf("new scheduler failed: %v", err)
	}
	ctx := context.Background()
	claimed := mustAwaitingTask(t, s, "task-a32-conflict", "run-a32-conflict", "remote-a32-conflict")

	first, err := s.CommitAsyncReportTerminal(ctx, TerminalCommit{
		TaskID:       claimed.Record.Task.TaskID,
		AttemptID:    claimed.Attempt.AttemptID,
		Status:       TaskStateSucceeded,
		Source:       AsyncResolutionSourceCallback,
		RemoteTaskID: "remote-a32-conflict",
		Result:       map[string]any{"ok": true},
		CommittedAt:  time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("first terminal commit failed: %v", err)
	}
	if first.Duplicate || first.Conflict {
		t.Fatalf("first commit meta mismatch: %#v", first)
	}

	second, err := s.CommitAsyncReportTerminal(ctx, TerminalCommit{
		TaskID:       claimed.Record.Task.TaskID,
		AttemptID:    claimed.Attempt.AttemptID,
		Status:       TaskStateFailed,
		Source:       AsyncResolutionSourceReconcilePoll,
		RemoteTaskID: "remote-a32-conflict",
		ErrorMessage: "remote failure",
		CommittedAt:  time.Now().UTC().Add(time.Second),
	})
	if err != nil {
		t.Fatalf("second terminal commit failed: %v", err)
	}
	if !second.Duplicate || !second.Conflict {
		t.Fatalf("second commit meta mismatch: %#v", second)
	}

	third, err := s.CommitAsyncReportTerminal(ctx, TerminalCommit{
		TaskID:       claimed.Record.Task.TaskID,
		AttemptID:    claimed.Attempt.AttemptID,
		Status:       TaskStateFailed,
		Source:       AsyncResolutionSourceReconcilePoll,
		RemoteTaskID: "remote-a32-conflict",
		ErrorMessage: "remote failure",
		CommittedAt:  time.Now().UTC().Add(2 * time.Second),
	})
	if err != nil {
		t.Fatalf("third terminal commit failed: %v", err)
	}
	if !third.Duplicate || !third.Conflict {
		t.Fatalf("third commit meta mismatch: %#v", third)
	}

	record, found, err := s.Get(ctx, claimed.Record.Task.TaskID)
	if err != nil || !found {
		t.Fatalf("get task failed: found=%v err=%v", found, err)
	}
	if record.State != TaskStateSucceeded {
		t.Fatalf("business state mutated by late conflict, got %q", record.State)
	}
	if !record.TerminalConflict {
		t.Fatalf("terminal_conflict_recorded=false, want true")
	}

	stats, err := s.Stats(ctx)
	if err != nil {
		t.Fatalf("stats failed: %v", err)
	}
	if stats.AsyncTerminalConflictTotal != 1 {
		t.Fatalf("async_terminal_conflict_total=%d, want 1", stats.AsyncTerminalConflictTotal)
	}
}

func TestSchedulerNextAsyncReconcileDelayHonorsJitterRange(t *testing.T) {
	s, err := New(
		NewMemoryStore(),
		WithAsyncAwait(AsyncAwaitConfig{
			ReportTimeout:    15 * time.Minute,
			LateReportPolicy: AsyncLateReportPolicyDropAndRecord,
			TimeoutTerminal:  TaskStateFailed,
			Reconcile: AsyncAwaitReconcileConfig{
				Enabled:        true,
				Interval:       10 * time.Second,
				BatchSize:      32,
				JitterRatio:    0.2,
				NotFoundPolicy: AsyncReconcileNotFoundKeepTimeout,
			},
		}),
	)
	if err != nil {
		t.Fatalf("new scheduler failed: %v", err)
	}
	for i := 0; i < 16; i++ {
		delay := s.NextAsyncReconcileDelay()
		if delay < 8*time.Second || delay > 12*time.Second {
			t.Fatalf("delay=%s out of expected [8s,12s] range", delay)
		}
	}

	steady, err := New(
		NewMemoryStore(),
		WithAsyncAwait(AsyncAwaitConfig{
			ReportTimeout:    15 * time.Minute,
			LateReportPolicy: AsyncLateReportPolicyDropAndRecord,
			TimeoutTerminal:  TaskStateFailed,
			Reconcile: AsyncAwaitReconcileConfig{
				Enabled:        true,
				Interval:       10 * time.Second,
				BatchSize:      32,
				JitterRatio:    0,
				NotFoundPolicy: AsyncReconcileNotFoundKeepTimeout,
			},
		}),
	)
	if err != nil {
		t.Fatalf("new steady scheduler failed: %v", err)
	}
	for i := 0; i < 3; i++ {
		if got := steady.NextAsyncReconcileDelay(); got != 10*time.Second {
			t.Fatalf("steady reconcile delay=%s, want 10s", got)
		}
	}
}

func mustAwaitingTask(t *testing.T, s *Scheduler, taskID, runID, remoteTaskID string) ClaimedTask {
	t.Helper()
	ctx := context.Background()
	if _, err := s.Enqueue(ctx, Task{TaskID: taskID, RunID: runID}); err != nil {
		t.Fatalf("enqueue %q failed: %v", taskID, err)
	}
	claimed, ok, err := s.Claim(ctx, "worker-"+taskID)
	if err != nil || !ok {
		t.Fatalf("claim %q failed: ok=%v err=%v", taskID, ok, err)
	}
	record, err := s.MarkAwaitingReport(ctx, taskID, claimed.Attempt.AttemptID, remoteTaskID)
	if err != nil {
		t.Fatalf("mark awaiting_report %q failed: %v", taskID, err)
	}
	if record.State != TaskStateAwaitingReport {
		t.Fatalf("task %q state=%q, want awaiting_report", taskID, record.State)
	}
	if record.RemoteTaskID != remoteTaskID {
		t.Fatalf("task %q remote_task_id=%q, want %q", taskID, record.RemoteTaskID, remoteTaskID)
	}
	return claimed
}

func TestClassifyReconcilePollClassifiesTransportAndProtocolErrors(t *testing.T) {
	client := fakeReconcilePollClient{
		statusFn: func(_ context.Context, _ string) (a2a.TaskRecord, error) {
			return a2a.TaskRecord{}, context.DeadlineExceeded
		},
	}
	classification, _, err := ClassifyReconcilePoll(context.Background(), client, "remote-timeout")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected timeout error, got %v", err)
	}
	if classification != ReconcilePollClassificationRetryableError {
		t.Fatalf("classification=%q, want %q", classification, ReconcilePollClassificationRetryableError)
	}

	client = fakeReconcilePollClient{
		statusFn: func(_ context.Context, _ string) (a2a.TaskRecord, error) {
			return a2a.TaskRecord{}, errors.New("unsupported method")
		},
	}
	classification, _, err = ClassifyReconcilePoll(context.Background(), client, "remote-protocol")
	if err == nil {
		t.Fatal("expected protocol error")
	}
	if classification != ReconcilePollClassificationNonRetryableErr {
		t.Fatalf("classification=%q, want %q", classification, ReconcilePollClassificationNonRetryableErr)
	}
}
