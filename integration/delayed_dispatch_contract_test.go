package integration

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/a2a"
	"github.com/FelixSeptem/baymax/orchestration/scheduler"
)

func TestDelayedDispatchContractEarlyClaimBlockedThenReady(t *testing.T) {
	s, err := scheduler.New(
		scheduler.NewMemoryStore(),
		scheduler.WithLeaseTimeout(500*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("new scheduler: %v", err)
	}
	ctx := context.Background()
	if _, err := s.Enqueue(ctx, scheduler.Task{
		TaskID:    "task-delayed-a13",
		RunID:     "run-delayed-a13",
		NotBefore: time.Now().Add(120 * time.Millisecond),
	}); err != nil {
		t.Fatalf("enqueue delayed task: %v", err)
	}
	if _, err := s.Enqueue(ctx, scheduler.Task{
		TaskID: "task-ready-a13",
		RunID:  "run-delayed-a13",
	}); err != nil {
		t.Fatalf("enqueue ready task: %v", err)
	}

	claimedReady, ok, err := s.Claim(ctx, "worker-a13")
	if err != nil || !ok {
		t.Fatalf("first claim failed: ok=%v err=%v", ok, err)
	}
	if claimedReady.Record.Task.TaskID != "task-ready-a13" {
		t.Fatalf("first claim task=%q, want task-ready-a13", claimedReady.Record.Task.TaskID)
	}
	if _, ok, err := s.Claim(ctx, "worker-a13"); err != nil || ok {
		t.Fatalf("delayed task should still be blocked before not_before: ok=%v err=%v", ok, err)
	}

	time.Sleep(140 * time.Millisecond)
	claimedDelayed, ok, err := s.Claim(ctx, "worker-a13")
	if err != nil || !ok {
		t.Fatalf("delayed task should be claimable after boundary: ok=%v err=%v", ok, err)
	}
	if claimedDelayed.Record.Task.TaskID != "task-delayed-a13" {
		t.Fatalf("delayed claim task=%q, want task-delayed-a13", claimedDelayed.Record.Task.TaskID)
	}
}

func TestDelayedDispatchContractRecoveryNoEarlyClaim(t *testing.T) {
	ctx := context.Background()
	statePath := filepath.Join(t.TempDir(), "scheduler-state.json")
	store1, err := scheduler.NewFileStore(statePath)
	if err != nil {
		t.Fatalf("new file store #1: %v", err)
	}
	s1, err := scheduler.New(store1, scheduler.WithLeaseTimeout(500*time.Millisecond))
	if err != nil {
		t.Fatalf("new scheduler #1: %v", err)
	}
	if _, err := s1.Enqueue(ctx, scheduler.Task{
		TaskID:    "task-delayed-recovery-a13",
		RunID:     "run-delayed-recovery-a13",
		NotBefore: time.Now().Add(120 * time.Millisecond),
	}); err != nil {
		t.Fatalf("enqueue delayed task: %v", err)
	}

	store2, err := scheduler.NewFileStore(statePath)
	if err != nil {
		t.Fatalf("new file store #2: %v", err)
	}
	s2, err := scheduler.New(store2, scheduler.WithLeaseTimeout(500*time.Millisecond))
	if err != nil {
		t.Fatalf("new scheduler #2: %v", err)
	}
	if _, ok, err := s2.Claim(ctx, "worker-recovery-a13"); err != nil || ok {
		t.Fatalf("restored delayed task should remain blocked before boundary: ok=%v err=%v", ok, err)
	}
	time.Sleep(140 * time.Millisecond)
	if _, ok, err := s2.Claim(ctx, "worker-recovery-a13"); err != nil || !ok {
		t.Fatalf("restored delayed task should be claimable after boundary: ok=%v err=%v", ok, err)
	}
}

func TestDelayedDispatchContractRunStreamSemanticEquivalence(t *testing.T) {
	type summary struct {
		state           scheduler.TaskState
		delayedTasks    int
		delayedClaims   int
		delayedWaitP95  int64
		claimTotal      int
		terminalSuccess int
	}
	exec := func(taskID string) (summary, error) {
		s, err := scheduler.New(
			scheduler.NewMemoryStore(),
			scheduler.WithLeaseTimeout(500*time.Millisecond),
		)
		if err != nil {
			return summary{}, err
		}
		ctx := context.Background()
		if _, err := s.Enqueue(ctx, scheduler.Task{
			TaskID:    taskID,
			RunID:     "run-delayed-equivalence-a13",
			NotBefore: time.Now().Add(90 * time.Millisecond),
		}); err != nil {
			return summary{}, err
		}
		if _, ok, err := s.Claim(ctx, "worker-eq-a13"); err != nil {
			return summary{}, err
		} else if ok {
			return summary{}, errors.New("task claimed before not_before boundary")
		}
		time.Sleep(110 * time.Millisecond)
		claimed, ok, err := s.Claim(ctx, "worker-eq-a13")
		if err != nil || !ok {
			if err != nil {
				return summary{}, err
			}
			return summary{}, errors.New("task not claimable after not_before boundary")
		}
		if _, err := s.Complete(ctx, scheduler.TerminalCommit{
			TaskID:      claimed.Record.Task.TaskID,
			AttemptID:   claimed.Attempt.AttemptID,
			Status:      scheduler.TaskStateSucceeded,
			Result:      map[string]any{"ok": true},
			CommittedAt: time.Now(),
		}); err != nil {
			return summary{}, err
		}
		record, ok, err := s.Get(ctx, taskID)
		if err != nil || !ok {
			return summary{}, err
		}
		stats, err := s.Stats(ctx)
		if err != nil {
			return summary{}, err
		}
		return summary{
			state:           record.State,
			delayedTasks:    stats.DelayedTaskTotal,
			delayedClaims:   stats.DelayedClaimTotal,
			delayedWaitP95:  stats.DelayedWaitMsP95,
			claimTotal:      stats.ClaimTotal,
			terminalSuccess: stats.CompleteTotal,
		}, nil
	}

	runSummary, err := exec("task-delayed-run-a13")
	if err != nil {
		t.Fatalf("run path failed: %v", err)
	}
	streamSummary, err := exec("task-delayed-stream-a13")
	if err != nil {
		t.Fatalf("stream path failed: %v", err)
	}
	if runSummary != streamSummary {
		t.Fatalf("run/stream delayed summary mismatch: run=%#v stream=%#v", runSummary, streamSummary)
	}
	if runSummary.state != scheduler.TaskStateSucceeded {
		t.Fatalf("terminal state = %q, want succeeded", runSummary.state)
	}
}

func TestDelayedDispatchContractAsyncReportingCompatibility(t *testing.T) {
	s, err := scheduler.New(
		scheduler.NewMemoryStore(),
		scheduler.WithLeaseTimeout(500*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("new scheduler: %v", err)
	}
	ctx := context.Background()
	if _, err := s.Enqueue(ctx, scheduler.Task{
		TaskID:    "task-delayed-async-a13",
		RunID:     "run-delayed-async-a13",
		NotBefore: time.Now().Add(90 * time.Millisecond),
	}); err != nil {
		t.Fatalf("enqueue delayed task: %v", err)
	}
	if _, ok, err := s.Claim(ctx, "worker-async-a13"); err != nil || ok {
		t.Fatalf("delayed task should be blocked before not_before: ok=%v err=%v", ok, err)
	}
	time.Sleep(110 * time.Millisecond)
	claimed, ok, err := s.Claim(ctx, "worker-async-a13")
	if err != nil || !ok {
		t.Fatalf("delayed task should be claimable after boundary: ok=%v err=%v", ok, err)
	}

	execution, err := scheduler.ExecutionFromAsyncReport(claimed, a2a.AsyncReport{
		ReportKey:  "delayed-async-a13-report",
		OutcomeKey: "succeeded|ok",
		TaskID:     claimed.Record.Task.TaskID,
		AttemptID:  claimed.Attempt.AttemptID,
		Status:     a2a.StatusSucceeded,
		Result:     map[string]any{"ok": true},
	})
	if err != nil {
		t.Fatalf("ExecutionFromAsyncReport failed: %v", err)
	}
	if _, err := s.Complete(ctx, execution.Commit); err != nil {
		t.Fatalf("complete from async report failed: %v", err)
	}
	stats, err := s.Stats(ctx)
	if err != nil {
		t.Fatalf("stats failed: %v", err)
	}
	if stats.DelayedTaskTotal != 1 || stats.DelayedClaimTotal != 1 || stats.CompleteTotal != 1 {
		t.Fatalf("delayed+async summary mismatch: %#v", stats)
	}
}

func TestDelayedDispatchContractAsyncDelayedReplayNoInflation(t *testing.T) {
	s, err := scheduler.New(
		scheduler.NewMemoryStore(),
		scheduler.WithLeaseTimeout(500*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("new scheduler: %v", err)
	}
	ctx := context.Background()
	if _, err := s.Enqueue(ctx, scheduler.Task{
		TaskID:    "task-delayed-async-replay-a14",
		RunID:     "run-delayed-async-replay-a14",
		NotBefore: time.Now().Add(90 * time.Millisecond),
	}); err != nil {
		t.Fatalf("enqueue delayed task: %v", err)
	}
	if _, ok, err := s.Claim(ctx, "worker-async-replay-a14"); err != nil || ok {
		t.Fatalf("delayed task should be blocked before not_before: ok=%v err=%v", ok, err)
	}
	time.Sleep(110 * time.Millisecond)
	claimed, ok, err := s.Claim(ctx, "worker-async-replay-a14")
	if err != nil || !ok {
		t.Fatalf("delayed task should be claimable after boundary: ok=%v err=%v", ok, err)
	}

	report := a2a.AsyncReport{
		ReportKey:  "delayed-async-replay-a14-report",
		OutcomeKey: "succeeded|ok",
		TaskID:     claimed.Record.Task.TaskID,
		AttemptID:  claimed.Attempt.AttemptID,
		Status:     a2a.StatusSucceeded,
		Result:     map[string]any{"ok": true},
	}
	firstExec, err := scheduler.ExecutionFromAsyncReport(claimed, report)
	if err != nil {
		t.Fatalf("ExecutionFromAsyncReport failed: %v", err)
	}
	if _, err := s.Complete(ctx, firstExec.Commit); err != nil {
		t.Fatalf("first complete failed: %v", err)
	}
	replayExec, err := scheduler.ExecutionFromAsyncReport(claimed, report)
	if err != nil {
		t.Fatalf("ExecutionFromAsyncReport replay failed: %v", err)
	}
	if _, err := s.Complete(ctx, replayExec.Commit); err != nil {
		t.Fatalf("replay complete failed: %v", err)
	}
	stats, err := s.Stats(ctx)
	if err != nil {
		t.Fatalf("stats failed: %v", err)
	}
	if stats.DelayedTaskTotal != 1 ||
		stats.DelayedClaimTotal != 1 ||
		stats.CompleteTotal != 1 ||
		stats.DuplicateTerminalCommitTotal != 1 {
		t.Fatalf("delayed+async replay should not inflate logical aggregates: %#v", stats)
	}
}
