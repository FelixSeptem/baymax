package integration

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/orchestration/scheduler"
)

type schedulerRecoveryFixture struct {
	TaskID         string `json:"task_id"`
	RunID          string `json:"run_id"`
	LeaseTimeoutMs int    `json:"lease_timeout_ms"`
	ExpireWaitMs   int    `json:"expire_wait_ms"`
}

type timelineCollector struct {
	events []types.Event
}

func (c *timelineCollector) OnEvent(_ context.Context, ev types.Event) {
	c.events = append(c.events, ev)
}

func loadSchedulerFixture(t *testing.T, name string) schedulerRecoveryFixture {
	t.Helper()
	path := filepath.Join("testdata", "scheduler", name)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %q: %v", path, err)
	}
	var fixture schedulerRecoveryFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("parse fixture %q: %v", path, err)
	}
	if fixture.TaskID == "" || fixture.RunID == "" {
		t.Fatalf("invalid fixture %q: task_id/run_id are required", path)
	}
	if fixture.LeaseTimeoutMs <= 0 || fixture.ExpireWaitMs <= 0 {
		t.Fatalf("invalid fixture %q: lease_timeout_ms/expire_wait_ms must be > 0", path)
	}
	return fixture
}

func TestSchedulerRecoveryCrashLeaseExpiryTakeover(t *testing.T) {
	fixture := loadSchedulerFixture(t, "crash_takeover.json")
	s, err := scheduler.New(
		scheduler.NewMemoryStore(),
		scheduler.WithLeaseTimeout(time.Duration(fixture.LeaseTimeoutMs)*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("new scheduler: %v", err)
	}
	ctx := context.Background()
	if _, err := s.Enqueue(ctx, scheduler.Task{TaskID: fixture.TaskID, RunID: fixture.RunID}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	claimed1, ok, err := s.Claim(ctx, "worker-a")
	if err != nil || !ok {
		t.Fatalf("claim #1 failed: ok=%v err=%v", ok, err)
	}
	if claimed1.Attempt.Attempt != 1 {
		t.Fatalf("attempt #1 = %d, want 1", claimed1.Attempt.Attempt)
	}

	time.Sleep(time.Duration(fixture.ExpireWaitMs) * time.Millisecond)
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

func TestSchedulerRecoveryDuplicateSubmitCommitIdempotency(t *testing.T) {
	fixture := loadSchedulerFixture(t, "duplicate_commit.json")
	s, err := scheduler.New(
		scheduler.NewMemoryStore(),
		scheduler.WithLeaseTimeout(time.Duration(fixture.LeaseTimeoutMs)*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("new scheduler: %v", err)
	}
	ctx := context.Background()
	if _, err := s.Enqueue(ctx, scheduler.Task{TaskID: fixture.TaskID, RunID: fixture.RunID}); err != nil {
		t.Fatalf("enqueue #1: %v", err)
	}
	if _, err := s.Enqueue(ctx, scheduler.Task{TaskID: fixture.TaskID, RunID: fixture.RunID}); err != nil {
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

func TestSchedulerRecoveryRunStreamSemanticEquivalence(t *testing.T) {
	fixture := loadSchedulerFixture(t, "run_stream_equivalence.json")
	type summary struct {
		status  scheduler.TaskState
		queue   int
		claim   int
		reclaim int
	}
	exec := func(taskID string) (summary, error) {
		s, err := scheduler.New(
			scheduler.NewMemoryStore(),
			scheduler.WithLeaseTimeout(time.Duration(fixture.LeaseTimeoutMs)*time.Millisecond),
		)
		if err != nil {
			return summary{}, err
		}
		ctx := context.Background()
		if _, err := s.Enqueue(ctx, scheduler.Task{TaskID: taskID, RunID: fixture.RunID}); err != nil {
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
		return summary{
			status:  record.State,
			queue:   stats.QueueTotal,
			claim:   stats.ClaimTotal,
			reclaim: stats.ReclaimTotal,
		}, nil
	}

	runSummary, err := exec(fixture.TaskID + "-run")
	if err != nil {
		t.Fatalf("run path failed: %v", err)
	}
	streamSummary, err := exec(fixture.TaskID + "-stream")
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

func TestSchedulerRecoveryTimelineCorrelationRequiredFields(t *testing.T) {
	fixture := loadSchedulerFixture(t, "correlation.json")
	collector := &timelineCollector{}
	s, err := scheduler.New(
		scheduler.NewMemoryStore(),
		scheduler.WithTimelineEmitter(collector),
		scheduler.WithLeaseTimeout(time.Duration(fixture.LeaseTimeoutMs)*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("new scheduler: %v", err)
	}

	ctx := context.Background()
	if _, err := s.Enqueue(ctx, scheduler.Task{
		TaskID:     fixture.TaskID,
		RunID:      fixture.RunID,
		WorkflowID: "wf-correlation",
		TeamID:     "team-correlation",
		StepID:     "step-correlation",
		AgentID:    "agent-correlation",
		PeerID:     "peer-correlation",
	}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	claimed, ok, err := s.Claim(ctx, "worker-a")
	if err != nil || !ok {
		t.Fatalf("claim failed: ok=%v err=%v", ok, err)
	}
	time.Sleep(time.Duration(fixture.ExpireWaitMs) * time.Millisecond)
	expired, err := s.ExpireLeases(ctx)
	if err != nil {
		t.Fatalf("expire leases failed: %v", err)
	}
	if len(expired) != 1 {
		t.Fatalf("expired len = %d, want 1", len(expired))
	}
	reclaimed, ok, err := s.Claim(ctx, "worker-b")
	if err != nil || !ok {
		t.Fatalf("reclaim failed: ok=%v err=%v", ok, err)
	}
	if _, err := s.Complete(ctx, scheduler.TerminalCommit{
		TaskID:      reclaimed.Record.Task.TaskID,
		AttemptID:   reclaimed.Attempt.AttemptID,
		Status:      scheduler.TaskStateSucceeded,
		Result:      map[string]any{"ok": true},
		CommittedAt: time.Now(),
	}); err != nil {
		t.Fatalf("complete failed: %v", err)
	}

	requiredTaskReasons := map[string]bool{
		scheduler.ReasonEnqueue: false,
		scheduler.ReasonClaim:   false,
		scheduler.ReasonRequeue: false,
		scheduler.ReasonJoin:    false,
	}
	requiredAttemptReasons := map[string]bool{
		scheduler.ReasonClaim:   false,
		scheduler.ReasonRequeue: false,
		scheduler.ReasonJoin:    false,
	}
	for _, ev := range collector.events {
		if ev.Type != types.EventTypeActionTimeline {
			continue
		}
		reason, _ := ev.Payload["reason"].(string)
		if _, ok := requiredTaskReasons[reason]; ok {
			requiredTaskReasons[reason] = true
			if ev.Payload["task_id"] == "" {
				t.Fatalf("missing task_id for reason %q: %#v", reason, ev.Payload)
			}
		}
		if _, ok := requiredAttemptReasons[reason]; ok {
			requiredAttemptReasons[reason] = true
			if ev.Payload["attempt_id"] == "" {
				t.Fatalf("missing attempt_id for reason %q: %#v", reason, ev.Payload)
			}
		}
	}
	for reason, seen := range requiredTaskReasons {
		if !seen {
			t.Fatalf("missing timeline reason %q", reason)
		}
	}
	for reason, seen := range requiredAttemptReasons {
		if !seen {
			t.Fatalf("missing timeline attempt-correlation reason %q", reason)
		}
	}
	if claimed.Attempt.AttemptID == "" {
		t.Fatal("initial claim attempt_id should not be empty")
	}
}
