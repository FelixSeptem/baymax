package scheduler

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

type testTimelineCollector struct {
	events []types.Event
}

func (c *testTimelineCollector) OnEvent(_ context.Context, ev types.Event) {
	c.events = append(c.events, ev)
}

func TestMemoryAndFileStoreParity(t *testing.T) {
	type factory struct {
		name  string
		build func(t *testing.T) QueueStore
	}
	factories := []factory{
		{
			name: "memory",
			build: func(t *testing.T) QueueStore {
				t.Helper()
				return NewMemoryStore()
			},
		},
		{
			name: "file",
			build: func(t *testing.T) QueueStore {
				t.Helper()
				store, err := NewFileStore(filepath.Join(t.TempDir(), "scheduler-state.json"))
				if err != nil {
					t.Fatalf("new file store: %v", err)
				}
				return store
			},
		},
	}
	for _, item := range factories {
		item := item
		t.Run(item.name, func(t *testing.T) {
			runStoreParitySuite(t, item.build(t))
		})
	}
}

func TestStoreSnapshotRestoreRoundTrip(t *testing.T) {
	type factory struct {
		name  string
		build func(t *testing.T) QueueStore
	}
	factories := []factory{
		{
			name: "memory",
			build: func(t *testing.T) QueueStore {
				t.Helper()
				return NewMemoryStore()
			},
		},
		{
			name: "file",
			build: func(t *testing.T) QueueStore {
				t.Helper()
				store, err := NewFileStore(filepath.Join(t.TempDir(), "scheduler-state.json"))
				if err != nil {
					t.Fatalf("new file store: %v", err)
				}
				return store
			},
		},
	}
	for _, item := range factories {
		item := item
		t.Run(item.name, func(t *testing.T) {
			ctx := context.Background()
			store := item.build(t)
			snapStore, ok := store.(interface {
				Snapshot(context.Context) (StoreSnapshot, error)
				Restore(context.Context, StoreSnapshot) error
			})
			if !ok {
				t.Fatalf("store %T does not implement snapshot/restore", store)
			}

			now := time.Now()
			if _, err := store.Enqueue(ctx, Task{TaskID: "task-snap-1", RunID: "run-snap"}, now); err != nil {
				t.Fatalf("enqueue #1: %v", err)
			}
			if _, err := store.Enqueue(ctx, Task{TaskID: "task-snap-2", RunID: "run-snap"}, now); err != nil {
				t.Fatalf("enqueue #2: %v", err)
			}
			claimed, ok, err := store.Claim(ctx, "worker-snap", now.Add(10*time.Millisecond), 1*time.Second)
			if err != nil || !ok {
				t.Fatalf("claim failed: ok=%v err=%v", ok, err)
			}
			if _, err := store.CommitTerminal(ctx, TerminalCommit{
				TaskID:      claimed.Record.Task.TaskID,
				AttemptID:   claimed.Attempt.AttemptID,
				Status:      TaskStateSucceeded,
				Result:      map[string]any{"ok": true},
				CommittedAt: now.Add(100 * time.Millisecond),
			}); err != nil {
				t.Fatalf("commit failed: %v", err)
			}

			snapshot, err := snapStore.Snapshot(ctx)
			if err != nil {
				t.Fatalf("snapshot failed: %v", err)
			}
			restored := item.build(t)
			restorer, ok := restored.(interface {
				Snapshot(context.Context) (StoreSnapshot, error)
				Restore(context.Context, StoreSnapshot) error
			})
			if !ok {
				t.Fatalf("restored store %T does not implement snapshot/restore", restored)
			}
			if err := restorer.Restore(ctx, snapshot); err != nil {
				t.Fatalf("restore failed: %v", err)
			}
			snapAfter, err := restorer.Snapshot(ctx)
			if err != nil {
				t.Fatalf("snapshot after restore failed: %v", err)
			}
			if len(snapAfter.Tasks) != len(snapshot.Tasks) || len(snapAfter.Queue) != len(snapshot.Queue) {
				t.Fatalf("snapshot roundtrip mismatch: before=%#v after=%#v", snapshot, snapAfter)
			}
		})
	}
}

func TestFileStoreCorruptSnapshotFailsFast(t *testing.T) {
	path := filepath.Join(t.TempDir(), "corrupt-state.json")
	content := `{
  "tasks": {
    "task-corrupt": {
      "task": {"task_id":"task-corrupt"},
      "state":"running",
      "current_attempt_id":"missing",
      "attempts":[]
    }
  },
  "queue": [],
  "terminal_commits": {},
  "stats": {"backend":"file"}
}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write corrupt snapshot: %v", err)
	}
	_, err := NewFileStore(path)
	if err == nil {
		t.Fatal("expected file store load to fail for corrupt snapshot")
	}
	if !errors.Is(err, ErrSnapshotCorrupt) {
		t.Fatalf("expected ErrSnapshotCorrupt, got %v", err)
	}
}

func runStoreParitySuite(t *testing.T, store QueueStore) {
	t.Helper()
	ctx := context.Background()
	base := time.Now()

	record, err := store.Enqueue(ctx, Task{
		TaskID:      "task-1",
		RunID:       "run-1",
		WorkflowID:  "wf-1",
		TeamID:      "team-1",
		StepID:      "step-1",
		AgentID:     "agent-1",
		PeerID:      "peer-1",
		MaxAttempts: 3,
		Payload: map[string]any{
			"query": "hello",
		},
	}, base)
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if record.State != TaskStateQueued {
		t.Fatalf("state = %q, want queued", record.State)
	}

	claimed1, ok, err := store.Claim(ctx, "worker-a", base.Add(10*time.Millisecond), 1*time.Second)
	if err != nil {
		t.Fatalf("claim #1: %v", err)
	}
	if !ok {
		t.Fatal("claim #1 returned no task")
	}
	if claimed1.Record.State != TaskStateRunning || claimed1.Attempt.Attempt != 1 {
		t.Fatalf("unexpected first claim: %#v", claimed1)
	}

	renewed, err := store.Heartbeat(
		ctx,
		claimed1.Record.Task.TaskID,
		claimed1.Attempt.AttemptID,
		claimed1.Attempt.LeaseToken,
		base.Add(300*time.Millisecond),
		1*time.Second,
	)
	if err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
	if !renewed.Attempt.LeaseExpiresAt.After(base.Add(1 * time.Second)) {
		t.Fatalf("lease should be extended, got %v", renewed.Attempt.LeaseExpiresAt)
	}

	expired, err := store.ExpireLeases(ctx, base.Add(1600*time.Millisecond))
	if err != nil {
		t.Fatalf("expire leases: %v", err)
	}
	if len(expired) != 1 {
		t.Fatalf("expired len = %d, want 1", len(expired))
	}
	if expired[0].Attempt.AttemptID != claimed1.Attempt.AttemptID {
		t.Fatalf("expired attempt mismatch: %#v", expired[0].Attempt)
	}

	claimed2, ok, err := store.Claim(ctx, "worker-b", base.Add(1700*time.Millisecond), 1*time.Second)
	if err != nil {
		t.Fatalf("claim #2: %v", err)
	}
	if !ok {
		t.Fatal("claim #2 returned no task")
	}
	if claimed2.Attempt.Attempt != 2 {
		t.Fatalf("claim #2 attempt = %d, want 2", claimed2.Attempt.Attempt)
	}

	if _, err := store.CommitTerminal(ctx, TerminalCommit{
		TaskID:      claimed2.Record.Task.TaskID,
		AttemptID:   claimed1.Attempt.AttemptID,
		Status:      TaskStateFailed,
		CommittedAt: base.Add(1800 * time.Millisecond),
	}); !errors.Is(err, ErrStaleAttempt) {
		t.Fatalf("stale commit error = %v, want ErrStaleAttempt", err)
	}

	committed, err := store.CommitTerminal(ctx, TerminalCommit{
		TaskID:      claimed2.Record.Task.TaskID,
		AttemptID:   claimed2.Attempt.AttemptID,
		Status:      TaskStateSucceeded,
		Result:      map[string]any{"ok": true},
		CommittedAt: base.Add(2 * time.Second),
	})
	if err != nil {
		t.Fatalf("commit terminal: %v", err)
	}
	if committed.Duplicate {
		t.Fatal("first commit should not be duplicate")
	}
	if committed.Record.State != TaskStateSucceeded {
		t.Fatalf("terminal state = %q, want succeeded", committed.Record.State)
	}

	dup, err := store.CommitTerminal(ctx, TerminalCommit{
		TaskID:      claimed2.Record.Task.TaskID,
		AttemptID:   claimed2.Attempt.AttemptID,
		Status:      TaskStateSucceeded,
		Result:      map[string]any{"ok": true},
		CommittedAt: base.Add(2100 * time.Millisecond),
	})
	if err != nil {
		t.Fatalf("duplicate commit: %v", err)
	}
	if !dup.Duplicate {
		t.Fatal("duplicate commit should be marked as duplicate")
	}

	stats, err := store.Stats(ctx)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.QueueTotal != 1 || stats.ClaimTotal != 2 || stats.ReclaimTotal < 1 || stats.CompleteTotal != 1 || stats.DuplicateTerminalCommitTotal != 1 {
		t.Fatalf("stats mismatch: %#v", stats)
	}
}

func TestFileStoreCrashRecoveryAndTakeover(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "scheduler-state.json")
	base := time.Now()

	store1, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("new file store #1: %v", err)
	}
	if _, err := store1.Enqueue(ctx, Task{TaskID: "task-crash", RunID: "run-crash"}, base); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	claimed, ok, err := store1.Claim(ctx, "worker-a", base.Add(50*time.Millisecond), 1*time.Second)
	if err != nil || !ok {
		t.Fatalf("claim #1 failed: ok=%v err=%v", ok, err)
	}
	if claimed.Attempt.Attempt != 1 {
		t.Fatalf("attempt = %d, want 1", claimed.Attempt.Attempt)
	}

	store2, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("new file store #2: %v", err)
	}
	record, exists, err := store2.Get(ctx, "task-crash")
	if err != nil {
		t.Fatalf("get after restart: %v", err)
	}
	if !exists {
		t.Fatal("task should survive restart")
	}
	if record.State != TaskStateRunning || strings.TrimSpace(record.CurrentAttempt) == "" {
		t.Fatalf("unexpected recovered record: %#v", record)
	}

	expired, err := store2.ExpireLeases(ctx, base.Add(2*time.Second))
	if err != nil {
		t.Fatalf("expire after restart: %v", err)
	}
	if len(expired) != 1 {
		t.Fatalf("expired len = %d, want 1", len(expired))
	}
	claimed2, ok, err := store2.Claim(ctx, "worker-b", base.Add(2100*time.Millisecond), 1*time.Second)
	if err != nil || !ok {
		t.Fatalf("claim #2 failed: ok=%v err=%v", ok, err)
	}
	if claimed2.Attempt.Attempt != 2 {
		t.Fatalf("attempt #2 = %d, want 2", claimed2.Attempt.Attempt)
	}
}

func TestSchedulerGuardrailBudgetRejectAndTimelineReasons(t *testing.T) {
	collector := &testTimelineCollector{}
	s, err := New(
		NewMemoryStore(),
		WithTimelineEmitter(collector),
		WithLeaseTimeout(1*time.Second),
		WithGuardrails(Guardrails{
			MaxDepth:           1,
			MaxActiveChildren:  1,
			ChildTimeoutBudget: 500 * time.Millisecond,
		}),
	)
	if err != nil {
		t.Fatalf("new scheduler: %v", err)
	}

	_, err = s.SpawnChild(context.Background(), SpawnRequest{
		Task:                 Task{TaskID: "task-budget", RunID: "run-budget"},
		ParentDepth:          1,
		ParentActiveChildren: 1,
		ChildTimeout:         1200 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("expected budget reject")
	}
	var budgetErr *BudgetError
	if !errors.As(err, &budgetErr) {
		t.Fatalf("budget reject error type mismatch: %v", err)
	}

	reasons := map[string]bool{}
	for _, ev := range collector.events {
		if ev.Type != types.EventTypeActionTimeline {
			continue
		}
		reason, _ := ev.Payload["reason"].(string)
		reasons[reason] = true
	}
	if !reasons[ReasonBudgetReject] {
		t.Fatalf("missing reason %q in timeline %#v", ReasonBudgetReject, reasons)
	}
}

func TestSchedulerPriorityClaimWithFairnessWindow(t *testing.T) {
	s, err := New(
		NewMemoryStore(),
		WithGovernance(GovernanceConfig{
			QoS: QoSModePriority,
			Fairness: FairnessConfig{
				MaxConsecutiveClaimsPerPriority: 3,
			},
			DLQ: DLQConfig{Enabled: false},
			Backoff: RetryBackoffConfig{
				Enabled:     true,
				Initial:     10 * time.Millisecond,
				Max:         50 * time.Millisecond,
				Multiplier:  2.0,
				JitterRatio: 0,
			},
		}),
	)
	if err != nil {
		t.Fatalf("new scheduler: %v", err)
	}

	ctx := context.Background()
	tasks := []Task{
		{TaskID: "high-1", RunID: "run-priority", Priority: TaskPriorityHigh},
		{TaskID: "high-2", RunID: "run-priority", Priority: TaskPriorityHigh},
		{TaskID: "high-3", RunID: "run-priority", Priority: TaskPriorityHigh},
		{TaskID: "high-4", RunID: "run-priority", Priority: TaskPriorityHigh},
		{TaskID: "low-1", RunID: "run-priority", Priority: TaskPriorityLow},
	}
	for _, task := range tasks {
		if _, err := s.Enqueue(ctx, task); err != nil {
			t.Fatalf("enqueue %q: %v", task.TaskID, err)
		}
	}

	claimIDs := make([]string, 0, 4)
	for i := 0; i < 4; i++ {
		claimed, ok, err := s.Claim(ctx, "worker-priority")
		if err != nil || !ok {
			t.Fatalf("claim #%d failed: ok=%v err=%v", i+1, ok, err)
		}
		claimIDs = append(claimIDs, claimed.Record.Task.TaskID)
		if i == 3 && !claimed.FairnessYielded {
			t.Fatal("fourth claim should be fairness-yielded")
		}
	}
	want := []string{"high-1", "high-2", "high-3", "low-1"}
	for i := range want {
		if claimIDs[i] != want[i] {
			t.Fatalf("claim order mismatch at %d: got %q want %q (all=%#v)", i, claimIDs[i], want[i], claimIDs)
		}
	}

	stats, err := s.Stats(ctx)
	if err != nil {
		t.Fatalf("stats failed: %v", err)
	}
	if stats.QoSMode != string(QoSModePriority) {
		t.Fatalf("stats.qos_mode = %q, want %q", stats.QoSMode, QoSModePriority)
	}
	if stats.PriorityClaimTotal < 4 {
		t.Fatalf("priority_claim_total = %d, want >= 4", stats.PriorityClaimTotal)
	}
	if stats.FairnessYieldTotal < 1 {
		t.Fatalf("fairness_yield_total = %d, want >= 1", stats.FairnessYieldTotal)
	}
}

func TestSchedulerRetryBackoffAndDeadLetter(t *testing.T) {
	s, err := New(
		NewMemoryStore(),
		WithLeaseTimeout(2*time.Second),
		WithGovernance(GovernanceConfig{
			QoS: QoSModeFIFO,
			Fairness: FairnessConfig{
				MaxConsecutiveClaimsPerPriority: 3,
			},
			DLQ: DLQConfig{Enabled: true},
			Backoff: RetryBackoffConfig{
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
	if _, err := s.Enqueue(ctx, Task{
		TaskID:      "task-dlq",
		RunID:       "run-dlq",
		MaxAttempts: 2,
	}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	claimed1, ok, err := s.Claim(ctx, "worker-a")
	if err != nil || !ok {
		t.Fatalf("claim #1 failed: ok=%v err=%v", ok, err)
	}
	requeued, err := s.Requeue(ctx, claimed1.Record.Task.TaskID, "retryable_error")
	if err != nil {
		t.Fatalf("requeue #1 failed: %v", err)
	}
	if requeued.State != TaskStateQueued {
		t.Fatalf("state after first requeue = %q, want queued", requeued.State)
	}
	if requeued.NextEligibleAt.IsZero() {
		t.Fatal("next_eligible_at should be set after backoff requeue")
	}

	if _, ok, err := s.Claim(ctx, "worker-b"); err != nil || ok {
		t.Fatalf("claim during backoff should be blocked: ok=%v err=%v", ok, err)
	}
	time.Sleep(70 * time.Millisecond)
	claimed2, ok, err := s.Claim(ctx, "worker-b")
	if err != nil || !ok {
		t.Fatalf("claim #2 failed: ok=%v err=%v", ok, err)
	}

	dlqRecord, err := s.Requeue(ctx, claimed2.Record.Task.TaskID, "retryable_error")
	if err != nil {
		t.Fatalf("requeue #2 failed: %v", err)
	}
	if dlqRecord.State != TaskStateDeadLetter {
		t.Fatalf("state after retry exhaustion = %q, want dead_letter", dlqRecord.State)
	}
	if strings.TrimSpace(dlqRecord.DeadLetterCode) == "" {
		t.Fatal("dead_letter_code should be set")
	}

	stats, err := s.Stats(ctx)
	if err != nil {
		t.Fatalf("stats failed: %v", err)
	}
	if stats.RetryBackoffTotal < 1 {
		t.Fatalf("retry_backoff_total = %d, want >= 1", stats.RetryBackoffTotal)
	}
	if stats.DeadLetterTotal != 1 {
		t.Fatalf("dead_letter_total = %d, want 1", stats.DeadLetterTotal)
	}
}

func TestCanonicalReasonMapper(t *testing.T) {
	required := []string{
		ReasonEnqueue,
		ReasonClaim,
		ReasonHeartbeat,
		ReasonLeaseExpired,
		ReasonRequeue,
		ReasonQoSClaim,
		ReasonFairnessYield,
		ReasonRetryBackoff,
		ReasonDeadLetter,
		ReasonSpawn,
		ReasonJoin,
		ReasonBudgetReject,
	}
	for _, reason := range required {
		mapped, ok := CanonicalReason(reason)
		if !ok {
			t.Fatalf("expected canonical reason %q to be accepted", reason)
		}
		if mapped != reason {
			t.Fatalf("mapped reason = %q, want %q", mapped, reason)
		}
	}
	if mapped, ok := CanonicalReason("enqueue"); ok || mapped != "" {
		t.Fatalf("expected non-canonical reason to be rejected, got mapped=%q ok=%v", mapped, ok)
	}
}

func TestSchedulerLifecycleTimelineCorrelation(t *testing.T) {
	collector := &testTimelineCollector{}
	s, err := New(
		NewMemoryStore(),
		WithTimelineEmitter(collector),
		WithLeaseTimeout(80*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("new scheduler: %v", err)
	}

	ctx := context.Background()
	task, err := s.Enqueue(ctx, Task{
		TaskID:     "task-timeline",
		RunID:      "run-timeline",
		WorkflowID: "wf-timeline",
		TeamID:     "team-timeline",
		StepID:     "step-timeline",
		AgentID:    "agent-timeline",
		PeerID:     "peer-timeline",
	})
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if task.State != TaskStateQueued {
		t.Fatalf("state = %q, want queued", task.State)
	}
	claimed, ok, err := s.Claim(ctx, "worker-1")
	if err != nil || !ok {
		t.Fatalf("claim failed: ok=%v err=%v", ok, err)
	}
	if _, err := s.Heartbeat(ctx, claimed.Record.Task.TaskID, claimed.Attempt.AttemptID, claimed.Attempt.LeaseToken); err != nil {
		t.Fatalf("heartbeat failed: %v", err)
	}
	time.Sleep(120 * time.Millisecond)
	expired, err := s.ExpireLeases(ctx)
	if err != nil {
		t.Fatalf("expire leases failed: %v", err)
	}
	if len(expired) != 1 {
		t.Fatalf("expired len = %d, want 1", len(expired))
	}
	time.Sleep(70 * time.Millisecond)
	reclaimed, ok, err := s.Claim(ctx, "worker-2")
	if err != nil || !ok {
		t.Fatalf("reclaim failed: ok=%v err=%v", ok, err)
	}
	if _, err := s.Complete(ctx, TerminalCommit{
		TaskID:      reclaimed.Record.Task.TaskID,
		AttemptID:   reclaimed.Attempt.AttemptID,
		Status:      TaskStateSucceeded,
		Result:      map[string]any{"done": true},
		CommittedAt: time.Now(),
	}); err != nil {
		t.Fatalf("complete failed: %v", err)
	}

	requiredReasons := map[string]bool{
		ReasonEnqueue:   false,
		ReasonClaim:     false,
		ReasonHeartbeat: false,
		ReasonRequeue:   false,
		ReasonJoin:      false,
	}
	for _, ev := range collector.events {
		if ev.Type != types.EventTypeActionTimeline {
			continue
		}
		reason, _ := ev.Payload["reason"].(string)
		if _, ok := requiredReasons[reason]; ok {
			requiredReasons[reason] = true
		}
		if reason == ReasonClaim || reason == ReasonHeartbeat || reason == ReasonRequeue || reason == ReasonJoin {
			if ev.Payload["task_id"] == "" || ev.Payload["attempt_id"] == "" {
				t.Fatalf("missing task/attempt correlation in payload: %#v", ev.Payload)
			}
		}
		if reason == ReasonEnqueue && ev.Payload["task_id"] == "" {
			t.Fatalf("missing task correlation in enqueue payload: %#v", ev.Payload)
		}
		if ev.Payload["workflow_id"] != "wf-timeline" || ev.Payload["team_id"] != "team-timeline" || ev.Payload["step_id"] != "step-timeline" {
			t.Fatalf("run linkage metadata mismatch: %#v", ev.Payload)
		}
	}
	for reason, seen := range requiredReasons {
		if !seen {
			t.Fatalf("missing reason %q in timeline events", reason)
		}
	}
}
