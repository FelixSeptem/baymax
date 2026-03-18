package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/orchestration/scheduler"
)

func TestSchedulerQoSPriorityFairnessAndAntiStarvation(t *testing.T) {
	s, err := scheduler.New(
		scheduler.NewMemoryStore(),
		scheduler.WithGovernance(scheduler.GovernanceConfig{
			QoS: scheduler.QoSModePriority,
			Fairness: scheduler.FairnessConfig{
				MaxConsecutiveClaimsPerPriority: 3,
			},
			DLQ: scheduler.DLQConfig{Enabled: false},
		}),
	)
	if err != nil {
		t.Fatalf("new scheduler: %v", err)
	}
	ctx := context.Background()
	batch := []scheduler.Task{
		{TaskID: "high-1", RunID: "run-qos", Priority: scheduler.TaskPriorityHigh},
		{TaskID: "high-2", RunID: "run-qos", Priority: scheduler.TaskPriorityHigh},
		{TaskID: "high-3", RunID: "run-qos", Priority: scheduler.TaskPriorityHigh},
		{TaskID: "high-4", RunID: "run-qos", Priority: scheduler.TaskPriorityHigh},
		{TaskID: "low-1", RunID: "run-qos", Priority: scheduler.TaskPriorityLow},
	}
	for _, task := range batch {
		if _, err := s.Enqueue(ctx, task); err != nil {
			t.Fatalf("enqueue %q: %v", task.TaskID, err)
		}
	}

	order := make([]string, 0, 4)
	for i := 0; i < 4; i++ {
		claimed, ok, err := s.Claim(ctx, "worker-qos")
		if err != nil || !ok {
			t.Fatalf("claim #%d failed: ok=%v err=%v", i+1, ok, err)
		}
		order = append(order, claimed.Record.Task.TaskID)
		if i == 3 && !claimed.FairnessYielded {
			t.Fatal("4th claim should be fairness-yielded")
		}
	}
	want := []string{"high-1", "high-2", "high-3", "low-1"}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("claim order mismatch at %d: got=%q want=%q all=%#v", i, order[i], want[i], order)
		}
	}

	stats, err := s.Stats(ctx)
	if err != nil {
		t.Fatalf("stats failed: %v", err)
	}
	if stats.PriorityClaimTotal < 4 || stats.FairnessYieldTotal < 1 {
		t.Fatalf("qos stats mismatch: %#v", stats)
	}
}

func TestSchedulerQoSRetryBackoffDeadLetterAndReplayIdempotency(t *testing.T) {
	s, err := scheduler.New(
		scheduler.NewMemoryStore(),
		scheduler.WithLeaseTimeout(2*time.Second),
		scheduler.WithGovernance(scheduler.GovernanceConfig{
			QoS: scheduler.QoSModeFIFO,
			Fairness: scheduler.FairnessConfig{
				MaxConsecutiveClaimsPerPriority: 3,
			},
			DLQ: scheduler.DLQConfig{Enabled: true},
			Backoff: scheduler.RetryBackoffConfig{
				Enabled:     true,
				Initial:     40 * time.Millisecond,
				Max:         200 * time.Millisecond,
				Multiplier:  2.0,
				JitterRatio: 0,
			},
		}),
	)
	if err != nil {
		t.Fatalf("new scheduler: %v", err)
	}
	ctx := context.Background()
	if _, err := s.Enqueue(ctx, scheduler.Task{
		TaskID:      "task-qos-dlq",
		RunID:       "run-qos-dlq",
		MaxAttempts: 2,
	}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	claimed1, ok, err := s.Claim(ctx, "worker-a")
	if err != nil || !ok {
		t.Fatalf("claim #1 failed: ok=%v err=%v", ok, err)
	}
	record, err := s.Requeue(ctx, claimed1.Record.Task.TaskID, "retryable")
	if err != nil {
		t.Fatalf("requeue #1 failed: %v", err)
	}
	if record.State != scheduler.TaskStateQueued || record.NextEligibleAt.IsZero() {
		t.Fatalf("unexpected first requeue record: %#v", record)
	}
	if _, ok, err := s.Claim(ctx, "worker-b"); err != nil || ok {
		t.Fatalf("claim during backoff should be blocked: ok=%v err=%v", ok, err)
	}
	time.Sleep(70 * time.Millisecond)
	claimed2, ok, err := s.Claim(ctx, "worker-b")
	if err != nil || !ok {
		t.Fatalf("claim #2 failed: ok=%v err=%v", ok, err)
	}

	dlqRecord, err := s.Requeue(ctx, claimed2.Record.Task.TaskID, "retryable")
	if err != nil {
		t.Fatalf("requeue #2 failed: %v", err)
	}
	if dlqRecord.State != scheduler.TaskStateDeadLetter {
		t.Fatalf("state after retry exhaustion = %q, want dead_letter", dlqRecord.State)
	}
	if _, err := s.Requeue(ctx, claimed2.Record.Task.TaskID, "retryable"); !errors.Is(err, scheduler.ErrTaskNotRunning) {
		t.Fatalf("requeue after dead_letter should fail with ErrTaskNotRunning, got %v", err)
	}

	stats, err := s.Stats(ctx)
	if err != nil {
		t.Fatalf("stats failed: %v", err)
	}
	if stats.RetryBackoffTotal < 1 || stats.DeadLetterTotal != 1 {
		t.Fatalf("qos dlq stats mismatch: %#v", stats)
	}
}

func TestSchedulerQoSRunStreamSemanticEquivalence(t *testing.T) {
	type summary struct {
		state      scheduler.TaskState
		claim      int
		reclaim    int
		backoff    int
		deadLetter int
	}
	exec := func(taskID string) (summary, error) {
		s, err := scheduler.New(
			scheduler.NewMemoryStore(),
			scheduler.WithLeaseTimeout(2*time.Second),
			scheduler.WithGovernance(scheduler.GovernanceConfig{
				QoS: scheduler.QoSModeFIFO,
				Fairness: scheduler.FairnessConfig{
					MaxConsecutiveClaimsPerPriority: 3,
				},
				DLQ: scheduler.DLQConfig{Enabled: true},
				Backoff: scheduler.RetryBackoffConfig{
					Enabled:     true,
					Initial:     30 * time.Millisecond,
					Max:         120 * time.Millisecond,
					Multiplier:  2.0,
					JitterRatio: 0,
				},
			}),
		)
		if err != nil {
			return summary{}, err
		}
		ctx := context.Background()
		if _, err := s.Enqueue(ctx, scheduler.Task{
			TaskID:      taskID,
			RunID:       "run-qos-eq",
			MaxAttempts: 2,
		}); err != nil {
			return summary{}, err
		}
		claimed1, ok, err := s.Claim(ctx, "worker-eq-a")
		if err != nil || !ok {
			return summary{}, errors.New("claim #1 failed")
		}
		if _, err := s.Requeue(ctx, claimed1.Record.Task.TaskID, "retryable"); err != nil {
			return summary{}, err
		}
		time.Sleep(50 * time.Millisecond)
		claimed2, ok, err := s.Claim(ctx, "worker-eq-b")
		if err != nil || !ok {
			return summary{}, errors.New("claim #2 failed")
		}
		if _, err := s.Requeue(ctx, claimed2.Record.Task.TaskID, "retryable"); err != nil {
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
		return summary{
			state:      record.State,
			claim:      stats.ClaimTotal,
			reclaim:    stats.ReclaimTotal,
			backoff:    stats.RetryBackoffTotal,
			deadLetter: stats.DeadLetterTotal,
		}, nil
	}

	runSummary, err := exec("task-qos-run")
	if err != nil {
		t.Fatalf("run path failed: %v", err)
	}
	streamSummary, err := exec("task-qos-stream")
	if err != nil {
		t.Fatalf("stream path failed: %v", err)
	}
	if runSummary != streamSummary {
		t.Fatalf("run/stream qos summary mismatch: run=%#v stream=%#v", runSummary, streamSummary)
	}
	if runSummary.state != scheduler.TaskStateDeadLetter {
		t.Fatalf("terminal state = %q, want dead_letter", runSummary.state)
	}
}
