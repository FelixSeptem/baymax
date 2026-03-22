package integration

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/a2a"
	"github.com/FelixSeptem/baymax/orchestration/scheduler"
)

func TestAsyncAwaitReconcileContractCallbackLossFallbackConvergence(t *testing.T) {
	s, err := newAsyncAwaitReconcileScheduler(t, scheduler.NewMemoryStore())
	if err != nil {
		t.Fatalf("new scheduler failed: %v", err)
	}
	ctx := context.Background()
	taskID := "a32-fallback-task"
	claimed := seedAwaitingWithRemote(t, s, taskID, "run-a32-fallback", "remote-a32-fallback")

	poller := fakeA32ReconcilePollClient{
		statusFn: func(_ context.Context, remoteTaskID string) (a2a.TaskRecord, error) {
			return a2a.TaskRecord{TaskID: remoteTaskID, Status: a2a.StatusSucceeded}, nil
		},
		resultFn: func(_ context.Context, remoteTaskID string) (a2a.TaskRecord, error) {
			return a2a.TaskRecord{
				TaskID: remoteTaskID,
				Status: a2a.StatusSucceeded,
				Result: map[string]any{"ok": true},
			}, nil
		},
	}
	cycle, err := s.ReconcileAwaitingReports(ctx, poller)
	if err != nil {
		t.Fatalf("reconcile fallback failed: %v", err)
	}
	if cycle.PollTotal != 1 || cycle.TerminalByPoll != 1 || cycle.ErrorTotal != 0 {
		t.Fatalf("unexpected reconcile cycle stats: %#v", cycle)
	}

	record, found, err := s.Get(ctx, taskID)
	if err != nil || !found {
		t.Fatalf("get reconciled task failed: found=%v err=%v", found, err)
	}
	if record.State != scheduler.TaskStateSucceeded {
		t.Fatalf("task state=%q, want succeeded", record.State)
	}
	if record.ResolutionSource != scheduler.AsyncResolutionSourceReconcilePoll {
		t.Fatalf("resolution_source=%q, want %q", record.ResolutionSource, scheduler.AsyncResolutionSourceReconcilePoll)
	}
	if record.RemoteTaskID != "remote-a32-fallback" {
		t.Fatalf("remote_task_id=%q, want remote-a32-fallback", record.RemoteTaskID)
	}
	if claimed.Record.Task.TaskID == "" {
		t.Fatal("expected claimed task to be non-empty")
	}
}

func TestAsyncAwaitReconcileContractRunStreamSemanticEquivalence(t *testing.T) {
	runSummary := executeA32ReconcileFlow(t, scheduler.NewMemoryStore(), "a32-run-equivalent")
	streamSummary := executeA32ReconcileFlow(t, scheduler.NewMemoryStore(), "a32-stream-equivalent")
	if runSummary != streamSummary {
		t.Fatalf("run/stream reconcile summary mismatch: run=%#v stream=%#v", runSummary, streamSummary)
	}
}

func TestAsyncAwaitReconcileContractMemoryFileParity(t *testing.T) {
	memSummary := executeA32ReconcileFlow(t, scheduler.NewMemoryStore(), "a32-memory")

	fileStore, err := scheduler.NewFileStore(filepath.Join(t.TempDir(), "scheduler-a32-state.json"))
	if err != nil {
		t.Fatalf("new file store failed: %v", err)
	}
	fileSummary := executeA32ReconcileFlow(t, fileStore, "a32-file")
	if memSummary != fileSummary {
		t.Fatalf("memory/file reconcile summary mismatch: memory=%#v file=%#v", memSummary, fileSummary)
	}
}

func TestAsyncAwaitReconcileContractReplayIdempotencyForMixedCallbackPollEvents(t *testing.T) {
	s, err := newAsyncAwaitReconcileScheduler(t, scheduler.NewMemoryStore())
	if err != nil {
		t.Fatalf("new scheduler failed: %v", err)
	}
	ctx := context.Background()
	taskID := "a32-replay-mixed"
	claimed := seedAwaitingWithRemote(t, s, taskID, "run-a32-replay-mixed", "remote-a32-replay-mixed")

	if _, err := s.CommitAsyncReportTerminal(ctx, scheduler.TerminalCommit{
		TaskID:       taskID,
		AttemptID:    claimed.Attempt.AttemptID,
		Status:       scheduler.TaskStateSucceeded,
		Source:       scheduler.AsyncResolutionSourceCallback,
		RemoteTaskID: "remote-a32-replay-mixed",
		Result:       map[string]any{"ok": true},
		CommittedAt:  time.Now().UTC(),
	}); err != nil {
		t.Fatalf("callback commit failed: %v", err)
	}

	replayPoll := scheduler.TerminalCommit{
		TaskID:       taskID,
		AttemptID:    claimed.Attempt.AttemptID,
		Status:       scheduler.TaskStateFailed,
		Source:       scheduler.AsyncResolutionSourceReconcilePoll,
		RemoteTaskID: "remote-a32-replay-mixed",
		ErrorMessage: "poll-failed",
		ErrorClass:   "mcp",
		ErrorLayer:   "protocol",
		CommittedAt:  time.Now().UTC().Add(time.Second),
	}
	firstReplay, err := s.CommitAsyncReportTerminal(ctx, replayPoll)
	if err != nil {
		t.Fatalf("first poll replay failed: %v", err)
	}
	if !firstReplay.Duplicate || !firstReplay.Conflict {
		t.Fatalf("first poll replay meta mismatch: %#v", firstReplay)
	}
	secondReplay, err := s.CommitAsyncReportTerminal(ctx, replayPoll)
	if err != nil {
		t.Fatalf("second poll replay failed: %v", err)
	}
	if !secondReplay.Duplicate || !secondReplay.Conflict {
		t.Fatalf("second poll replay meta mismatch: %#v", secondReplay)
	}
	if _, err := s.CommitAsyncReportTerminal(ctx, scheduler.TerminalCommit{
		TaskID:       taskID,
		AttemptID:    claimed.Attempt.AttemptID,
		Status:       scheduler.TaskStateSucceeded,
		Source:       scheduler.AsyncResolutionSourceCallback,
		RemoteTaskID: "remote-a32-replay-mixed",
		Result:       map[string]any{"ok": true},
		CommittedAt:  time.Now().UTC().Add(2 * time.Second),
	}); err != nil {
		t.Fatalf("callback replay failed: %v", err)
	}

	record, found, err := s.Get(ctx, taskID)
	if err != nil || !found {
		t.Fatalf("get task failed: found=%v err=%v", found, err)
	}
	if record.State != scheduler.TaskStateSucceeded {
		t.Fatalf("business terminal state drifted: %q", record.State)
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
	if stats.AsyncReconcileTerminalByPollTotal != 0 {
		t.Fatalf("async_reconcile_terminal_by_poll_total=%d, want 0", stats.AsyncReconcileTerminalByPollTotal)
	}
}

type a32ReconcileSummary struct {
	State            scheduler.TaskState
	ResolutionSource string
	PollTotal        int
	TerminalByPoll   int
	ErrorTotal       int
}

func executeA32ReconcileFlow(t *testing.T, store scheduler.QueueStore, suffix string) a32ReconcileSummary {
	t.Helper()
	s, err := newAsyncAwaitReconcileScheduler(t, store)
	if err != nil {
		t.Fatalf("new scheduler failed: %v", err)
	}
	taskID := "a32-flow-" + suffix
	seedAwaitingWithRemote(t, s, taskID, "run-"+suffix, "remote-"+suffix)

	poller := fakeA32ReconcilePollClient{
		statusFn: func(_ context.Context, remoteTaskID string) (a2a.TaskRecord, error) {
			return a2a.TaskRecord{TaskID: remoteTaskID, Status: a2a.StatusSucceeded}, nil
		},
		resultFn: func(_ context.Context, remoteTaskID string) (a2a.TaskRecord, error) {
			return a2a.TaskRecord{TaskID: remoteTaskID, Status: a2a.StatusSucceeded, Result: map[string]any{"ok": true}}, nil
		},
	}
	cycle, err := s.ReconcileAwaitingReports(context.Background(), poller)
	if err != nil {
		t.Fatalf("reconcile flow failed: %v", err)
	}
	record, found, err := s.Get(context.Background(), taskID)
	if err != nil || !found {
		t.Fatalf("get reconciled task failed: found=%v err=%v", found, err)
	}
	return a32ReconcileSummary{
		State:            record.State,
		ResolutionSource: record.ResolutionSource,
		PollTotal:        cycle.PollTotal,
		TerminalByPoll:   cycle.TerminalByPoll,
		ErrorTotal:       cycle.ErrorTotal,
	}
}

func newAsyncAwaitReconcileScheduler(t *testing.T, store scheduler.QueueStore) (*scheduler.Scheduler, error) {
	t.Helper()
	return scheduler.New(
		store,
		scheduler.WithLeaseTimeout(500*time.Millisecond),
		scheduler.WithAsyncAwait(scheduler.AsyncAwaitConfig{
			ReportTimeout:    200 * time.Millisecond,
			LateReportPolicy: scheduler.AsyncLateReportPolicyDropAndRecord,
			TimeoutTerminal:  scheduler.TaskStateFailed,
			Reconcile: scheduler.AsyncAwaitReconcileConfig{
				Enabled:        true,
				Interval:       20 * time.Millisecond,
				BatchSize:      64,
				JitterRatio:    0,
				NotFoundPolicy: scheduler.AsyncReconcileNotFoundKeepTimeout,
			},
		}),
	)
}

func seedAwaitingWithRemote(
	t *testing.T,
	s *scheduler.Scheduler,
	taskID string,
	runID string,
	remoteTaskID string,
) scheduler.ClaimedTask {
	t.Helper()
	ctx := context.Background()
	if _, err := s.Enqueue(ctx, scheduler.Task{TaskID: taskID, RunID: runID}); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	claimed, ok, err := s.Claim(ctx, "worker-"+taskID)
	if err != nil || !ok {
		t.Fatalf("claim failed: ok=%v err=%v", ok, err)
	}
	record, err := s.MarkAwaitingReport(ctx, taskID, claimed.Attempt.AttemptID, remoteTaskID)
	if err != nil {
		t.Fatalf("mark awaiting_report failed: %v", err)
	}
	if record.State != scheduler.TaskStateAwaitingReport {
		t.Fatalf("state=%q, want awaiting_report", record.State)
	}
	return claimed
}

type fakeA32ReconcilePollClient struct {
	statusFn func(context.Context, string) (a2a.TaskRecord, error)
	resultFn func(context.Context, string) (a2a.TaskRecord, error)
}

func (c fakeA32ReconcilePollClient) Status(ctx context.Context, taskID string) (a2a.TaskRecord, error) {
	if c.statusFn != nil {
		return c.statusFn(ctx, taskID)
	}
	return a2a.TaskRecord{}, nil
}

func (c fakeA32ReconcilePollClient) Result(ctx context.Context, taskID string) (a2a.TaskRecord, error) {
	if c.resultFn != nil {
		return c.resultFn(ctx, taskID)
	}
	return a2a.TaskRecord{}, nil
}
