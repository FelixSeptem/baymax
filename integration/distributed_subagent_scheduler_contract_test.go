package integration

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/a2a"
	"github.com/FelixSeptem/baymax/orchestration/scheduler"
)

type sequencedA2AClient struct {
	mu          sync.Mutex
	submitCount int
}

func (c *sequencedA2AClient) Submit(ctx context.Context, req a2a.TaskRequest) (a2a.TaskRecord, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.submitCount++
	if c.submitCount == 1 {
		return a2a.TaskRecord{}, context.DeadlineExceeded
	}
	return a2a.TaskRecord{TaskID: req.TaskID, Status: a2a.StatusSubmitted}, nil
}

func (c *sequencedA2AClient) WaitResult(
	_ context.Context,
	taskID string,
	_ time.Duration,
	_ func(context.Context, a2a.TaskRecord) error,
) (a2a.TaskRecord, error) {
	return a2a.TaskRecord{
		TaskID: taskID,
		Status: a2a.StatusSucceeded,
		Result: map[string]any{"ok": true},
	}, nil
}

func TestWorkerCrashLeaseExpiryTakeover(t *testing.T) {
	s, err := scheduler.New(
		scheduler.NewMemoryStore(),
		scheduler.WithLeaseTimeout(80*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("new scheduler: %v", err)
	}
	ctx := context.Background()
	if _, err := s.Enqueue(ctx, scheduler.Task{TaskID: "task-crash", RunID: "run-crash"}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	claimed1, ok, err := s.Claim(ctx, "worker-a")
	if err != nil || !ok {
		t.Fatalf("claim #1 failed: ok=%v err=%v", ok, err)
	}
	if claimed1.Attempt.Attempt != 1 {
		t.Fatalf("attempt #1 = %d, want 1", claimed1.Attempt.Attempt)
	}

	time.Sleep(120 * time.Millisecond)
	expired, err := s.ExpireLeases(ctx)
	if err != nil {
		t.Fatalf("expire leases: %v", err)
	}
	if len(expired) != 1 {
		t.Fatalf("expired len = %d, want 1", len(expired))
	}
	claimed2, ok, err := s.Claim(ctx, "worker-b")
	if err != nil || !ok {
		t.Fatalf("claim #2 failed: ok=%v err=%v", ok, err)
	}
	if claimed2.Attempt.Attempt != 2 {
		t.Fatalf("attempt #2 = %d, want 2", claimed2.Attempt.Attempt)
	}
}

func TestSchedulerDuplicateSubmitResultReplayIdempotency(t *testing.T) {
	s, err := scheduler.New(scheduler.NewMemoryStore(), scheduler.WithLeaseTimeout(500*time.Millisecond))
	if err != nil {
		t.Fatalf("new scheduler: %v", err)
	}
	ctx := context.Background()
	if _, err := s.Enqueue(ctx, scheduler.Task{TaskID: "task-idempotent", RunID: "run-idempotent"}); err != nil {
		t.Fatalf("enqueue #1: %v", err)
	}
	// duplicate submit should be idempotent (single logical task)
	if _, err := s.Enqueue(ctx, scheduler.Task{TaskID: "task-idempotent", RunID: "run-idempotent"}); err != nil {
		t.Fatalf("enqueue #2 should be idempotent, got %v", err)
	}

	claimed, ok, err := s.Claim(ctx, "worker-main")
	if err != nil || !ok {
		t.Fatalf("claim failed: ok=%v err=%v", ok, err)
	}
	first, err := s.Complete(ctx, scheduler.TerminalCommit{
		TaskID:      claimed.Record.Task.TaskID,
		AttemptID:   claimed.Attempt.AttemptID,
		Status:      scheduler.TaskStateSucceeded,
		Result:      map[string]any{"ok": true},
		CommittedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("complete #1: %v", err)
	}
	if first.Duplicate {
		t.Fatal("first complete should not be duplicate")
	}
	dup, err := s.Complete(ctx, scheduler.TerminalCommit{
		TaskID:      claimed.Record.Task.TaskID,
		AttemptID:   claimed.Attempt.AttemptID,
		Status:      scheduler.TaskStateSucceeded,
		Result:      map[string]any{"ok": true},
		CommittedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("complete #2: %v", err)
	}
	if !dup.Duplicate {
		t.Fatal("second complete should be duplicate")
	}
	stats, err := s.Stats(ctx)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.QueueTotal != 1 || stats.CompleteTotal != 1 || stats.DuplicateTerminalCommitTotal != 1 {
		t.Fatalf("stats mismatch under duplicate replay: %#v", stats)
	}
}

func TestSchedulerManagedRunStreamSemanticEquivalence(t *testing.T) {
	type summary struct {
		status  scheduler.TaskState
		queue   int
		claim   int
		reclaim int
	}
	exec := func(taskID string) (summary, error) {
		s, err := scheduler.New(scheduler.NewMemoryStore(), scheduler.WithLeaseTimeout(500*time.Millisecond))
		if err != nil {
			return summary{}, err
		}
		ctx := context.Background()
		if _, err := s.Enqueue(ctx, scheduler.Task{TaskID: taskID, RunID: "run-equivalence"}); err != nil {
			return summary{}, err
		}
		claimed, ok, err := s.Claim(ctx, "worker-eq")
		if err != nil || !ok {
			return summary{}, errors.New("claim failed")
		}
		if _, err := s.Complete(ctx, scheduler.TerminalCommit{
			TaskID:      claimed.Record.Task.TaskID,
			AttemptID:   claimed.Attempt.AttemptID,
			Status:      scheduler.TaskStateSucceeded,
			Result:      map[string]any{"mode": "ok"},
			CommittedAt: time.Now(),
		}); err != nil {
			return summary{}, err
		}
		record, ok, err := s.Get(ctx, taskID)
		if err != nil || !ok {
			return summary{}, errors.New("get failed")
		}
		stats, err := s.Stats(ctx)
		if err != nil {
			return summary{}, err
		}
		return summary{status: record.State, queue: stats.QueueTotal, claim: stats.ClaimTotal, reclaim: stats.ReclaimTotal}, nil
	}

	runSummary, err := exec("task-run")
	if err != nil {
		t.Fatalf("run path failed: %v", err)
	}
	streamSummary, err := exec("task-stream")
	if err != nil {
		t.Fatalf("stream path failed: %v", err)
	}
	if runSummary.status != streamSummary.status {
		t.Fatalf("terminal status mismatch: run=%q stream=%q", runSummary.status, streamSummary.status)
	}
	if runSummary.queue != streamSummary.queue || runSummary.claim != streamSummary.claim || runSummary.reclaim != streamSummary.reclaim {
		t.Fatalf("aggregate mismatch: run=%#v stream=%#v", runSummary, streamSummary)
	}
}

func TestA2ASchedulerRetryAndErrorLayerNormalization(t *testing.T) {
	s, err := scheduler.New(scheduler.NewMemoryStore(), scheduler.WithLeaseTimeout(80*time.Millisecond))
	if err != nil {
		t.Fatalf("new scheduler: %v", err)
	}
	ctx := context.Background()
	if _, err := s.Enqueue(ctx, scheduler.Task{
		TaskID:     "task-a2a-retry",
		RunID:      "run-a2a-retry",
		WorkflowID: "wf-a2a-retry",
		TeamID:     "team-a2a-retry",
		StepID:     "step-a2a-retry",
		AgentID:    "agent-main",
		PeerID:     "peer-remote",
		Payload:    map[string]any{"query": "hello"},
	}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	claimed1, ok, err := s.Claim(ctx, "worker-a")
	if err != nil || !ok {
		t.Fatalf("claim #1 failed: ok=%v err=%v", ok, err)
	}
	client := &sequencedA2AClient{}
	exec1, err := scheduler.ExecuteClaimWithA2A(ctx, client, claimed1, 5*time.Millisecond)
	if err == nil {
		t.Fatal("first execution should fail with retryable transport error")
	}
	if !exec1.Retryable {
		t.Fatal("transport failure should be retryable")
	}
	if exec1.Commit.ErrorLayer != string(a2a.ErrorLayerTransport) {
		t.Fatalf("error layer = %q, want transport", exec1.Commit.ErrorLayer)
	}

	time.Sleep(120 * time.Millisecond)
	expired, err := s.ExpireLeases(ctx)
	if err != nil {
		t.Fatalf("expire leases: %v", err)
	}
	if len(expired) != 1 {
		t.Fatalf("expired len = %d, want 1", len(expired))
	}
	claimed2, ok, err := s.Claim(ctx, "worker-b")
	if err != nil || !ok {
		t.Fatalf("claim #2 failed: ok=%v err=%v", ok, err)
	}
	exec2, err := scheduler.ExecuteClaimWithA2A(ctx, client, claimed2, 5*time.Millisecond)
	if err != nil {
		t.Fatalf("second execution should succeed, got %v", err)
	}
	if exec2.Retryable {
		t.Fatal("successful execution should not be retryable")
	}
	if _, err := s.Complete(ctx, exec2.Commit); err != nil {
		t.Fatalf("complete second attempt: %v", err)
	}
	record, ok, err := s.Get(ctx, "task-a2a-retry")
	if err != nil || !ok {
		t.Fatalf("get final task failed: ok=%v err=%v", ok, err)
	}
	if record.State != scheduler.TaskStateSucceeded {
		t.Fatalf("final task state = %q, want succeeded", record.State)
	}
	if len(record.Attempts) < 2 {
		t.Fatalf("attempt history length = %d, want >= 2", len(record.Attempts))
	}
}
