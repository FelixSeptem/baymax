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

func TestFileStoreOptionalGroupCommitAndFlush(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "scheduler-group-commit.json")
	base := time.Now()

	store, err := NewFileStore(path, WithPersistBatchSize(2), WithPersistDebounce(time.Hour))
	if err != nil {
		t.Fatalf("new file store: %v", err)
	}
	if _, err := store.Enqueue(ctx, Task{TaskID: "task-a", RunID: "run-group"}, base); err != nil {
		t.Fatalf("enqueue task-a: %v", err)
	}

	restarted, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("reopen before batch threshold: %v", err)
	}
	if _, exists, err := restarted.Get(ctx, "task-a"); err != nil || exists {
		t.Fatalf("task-a should not be persisted before batch threshold: exists=%v err=%v", exists, err)
	}

	if _, err := store.Enqueue(ctx, Task{TaskID: "task-b", RunID: "run-group"}, base.Add(time.Millisecond)); err != nil {
		t.Fatalf("enqueue task-b: %v", err)
	}
	restartedAfterBatch, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("reopen after batch threshold: %v", err)
	}
	if _, exists, err := restartedAfterBatch.Get(ctx, "task-a"); err != nil || !exists {
		t.Fatalf("task-a should be persisted after group commit: exists=%v err=%v", exists, err)
	}
	if _, exists, err := restartedAfterBatch.Get(ctx, "task-b"); err != nil || !exists {
		t.Fatalf("task-b should be persisted after group commit: exists=%v err=%v", exists, err)
	}

	pathFlush := filepath.Join(t.TempDir(), "scheduler-flush.json")
	flushStore, err := NewFileStore(pathFlush, WithPersistBatchSize(5), WithPersistDebounce(time.Hour))
	if err != nil {
		t.Fatalf("new flush file store: %v", err)
	}
	if _, err := flushStore.Enqueue(ctx, Task{TaskID: "task-flush", RunID: "run-flush"}, base); err != nil {
		t.Fatalf("enqueue task-flush: %v", err)
	}
	if err := flushStore.Flush(); err != nil {
		t.Fatalf("flush pending writes: %v", err)
	}
	restartedAfterFlush, err := NewFileStore(pathFlush)
	if err != nil {
		t.Fatalf("reopen after explicit flush: %v", err)
	}
	if _, exists, err := restartedAfterFlush.Get(ctx, "task-flush"); err != nil || !exists {
		t.Fatalf("task-flush should be persisted after explicit flush: exists=%v err=%v", exists, err)
	}
}

func TestFileStoreOptionalDebounceCommit(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "scheduler-debounce.json")
	base := time.Now()

	store, err := NewFileStore(path, WithPersistBatchSize(10), WithPersistDebounce(25*time.Millisecond))
	if err != nil {
		t.Fatalf("new file store: %v", err)
	}
	if _, err := store.Enqueue(ctx, Task{TaskID: "task-debounce-1", RunID: "run-debounce"}, base); err != nil {
		t.Fatalf("enqueue task-debounce-1: %v", err)
	}
	time.Sleep(35 * time.Millisecond)
	if _, err := store.Enqueue(ctx, Task{TaskID: "task-debounce-2", RunID: "run-debounce"}, base.Add(time.Millisecond)); err != nil {
		t.Fatalf("enqueue task-debounce-2: %v", err)
	}

	restarted, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("reopen after debounce window: %v", err)
	}
	if _, exists, err := restarted.Get(ctx, "task-debounce-1"); err != nil || !exists {
		t.Fatalf("task-debounce-1 should be persisted after debounce-triggered commit: exists=%v err=%v", exists, err)
	}
	if _, exists, err := restarted.Get(ctx, "task-debounce-2"); err != nil || !exists {
		t.Fatalf("task-debounce-2 should be persisted after debounce-triggered commit: exists=%v err=%v", exists, err)
	}
}

func TestFileStoreFlushBoundaryCrashRecoveryConsistency(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "scheduler-flush-boundary.json")
	store, err := NewFileStore(path, WithPersistBatchSize(10), WithPersistDebounce(time.Hour))
	if err != nil {
		t.Fatalf("new file store: %v", err)
	}
	base := time.Now()

	if _, err := store.Enqueue(ctx, Task{TaskID: "task-durable", RunID: "run-flush-boundary"}, base); err != nil {
		t.Fatalf("enqueue durable task: %v", err)
	}
	if err := store.Flush(); err != nil {
		t.Fatalf("flush durable task: %v", err)
	}
	if _, err := store.Enqueue(ctx, Task{TaskID: "task-pending", RunID: "run-flush-boundary"}, base.Add(time.Millisecond)); err != nil {
		t.Fatalf("enqueue pending task: %v", err)
	}

	restarted, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("reopen scheduler file store: %v", err)
	}
	if _, exists, err := restarted.Get(ctx, "task-durable"); err != nil || !exists {
		t.Fatalf("durable task should survive restart: exists=%v err=%v", exists, err)
	}
	if _, exists, err := restarted.Get(ctx, "task-pending"); err != nil || exists {
		t.Fatalf("pending task should not survive restart before next flush: exists=%v err=%v", exists, err)
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
		Task:                  Task{TaskID: "task-budget", RunID: "run-budget"},
		ParentDepth:           1,
		ParentActiveChildren:  1,
		ParentRemainingBudget: 2 * time.Second,
		ChildTimeout:          1200 * time.Millisecond,
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

func TestSchedulerSpawnChildAppliesParentBudgetClampAndStoresTimeoutMetadata(t *testing.T) {
	s, err := New(NewMemoryStore(), WithLeaseTimeout(1*time.Second))
	if err != nil {
		t.Fatalf("new scheduler: %v", err)
	}

	record, err := s.SpawnChild(context.Background(), SpawnRequest{
		Task:                  Task{TaskID: "task-a41-clamp", RunID: "run-a41-clamp"},
		ParentDepth:           0,
		ParentActiveChildren:  0,
		ParentRemainingBudget: 1200 * time.Millisecond,
		ChildTimeout:          2 * time.Second,
		TimeoutResolution: TimeoutResolutionMetadata{
			EffectiveOperationProfile: "interactive",
			Source:                    "request",
			Trace:                     `{"version":"v1","selected_source":"request"}`,
			ResolvedTimeout:           2 * time.Second,
		},
	})
	if err != nil {
		t.Fatalf("spawn child failed: %v", err)
	}
	meta := record.Task.TimeoutResolution
	if !meta.ParentBudgetClamped {
		t.Fatalf("parent budget clamp marker = %v, want true", meta.ParentBudgetClamped)
	}
	if meta.ResolvedTimeout != 1200*time.Millisecond {
		t.Fatalf("resolved timeout = %s, want 1200ms", meta.ResolvedTimeout)
	}
	if meta.EffectiveOperationProfile != "interactive" || meta.Source != "request" {
		t.Fatalf("timeout resolution profile/source mismatch: %#v", meta)
	}
	if strings.TrimSpace(meta.Trace) == "" {
		t.Fatalf("timeout resolution trace should not be empty: %#v", meta)
	}
}

func TestSchedulerSpawnChildRejectsExhaustedParentBudget(t *testing.T) {
	s, err := New(NewMemoryStore(), WithLeaseTimeout(1*time.Second))
	if err != nil {
		t.Fatalf("new scheduler: %v", err)
	}

	_, err = s.SpawnChild(context.Background(), SpawnRequest{
		Task:                  Task{TaskID: "task-a41-parent-exhausted", RunID: "run-a41-parent-exhausted"},
		ParentDepth:           0,
		ParentActiveChildren:  0,
		ParentRemainingBudget: 0,
		ChildTimeout:          500 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("expected parent budget exhausted reject")
	}
	var budgetErr *BudgetError
	if !errors.As(err, &budgetErr) {
		t.Fatalf("budget reject error type mismatch: %v", err)
	}
	if budgetErr.Code != BudgetRejectParentBudgetExhausted {
		t.Fatalf("budget reject code = %q, want %q", budgetErr.Code, BudgetRejectParentBudgetExhausted)
	}
}

func TestSchedulerSpawnChildTimeoutResolutionMetadataSnapshotRestoreStable(t *testing.T) {
	ctx := context.Background()
	s, err := New(NewMemoryStore(), WithLeaseTimeout(1*time.Second))
	if err != nil {
		t.Fatalf("new scheduler: %v", err)
	}

	_, err = s.SpawnChild(ctx, SpawnRequest{
		Task:                  Task{TaskID: "task-a41-restore", RunID: "run-a41-restore"},
		ParentDepth:           0,
		ParentActiveChildren:  0,
		ParentRemainingBudget: 900 * time.Millisecond,
		ChildTimeout:          1500 * time.Millisecond,
		TimeoutResolution: TimeoutResolutionMetadata{
			EffectiveOperationProfile: "batch",
			Source:                    "domain",
			Trace:                     `{"version":"v1","selected_source":"domain"}`,
			ResolvedTimeout:           1500 * time.Millisecond,
		},
	})
	if err != nil {
		t.Fatalf("spawn child failed: %v", err)
	}

	snapshot, err := s.Snapshot(ctx)
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}
	restored, err := New(NewMemoryStore(), WithLeaseTimeout(1*time.Second))
	if err != nil {
		t.Fatalf("new restored scheduler: %v", err)
	}
	if err := restored.Restore(ctx, snapshot); err != nil {
		t.Fatalf("restore failed: %v", err)
	}

	page, err := restored.QueryTasks(ctx, TaskBoardQueryRequest{TaskID: "task-a41-restore"})
	if err != nil {
		t.Fatalf("query tasks failed: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("query result len = %d, want 1", len(page.Items))
	}
	meta := page.Items[0].Task.TimeoutResolution
	if meta.EffectiveOperationProfile != "batch" || meta.Source != "domain" {
		t.Fatalf("restored timeout resolution profile/source mismatch: %#v", meta)
	}
	if !meta.ParentBudgetClamped {
		t.Fatalf("restored parent budget clamp marker = %v, want true", meta.ParentBudgetClamped)
	}
	if meta.ResolvedTimeout != 900*time.Millisecond {
		t.Fatalf("restored resolved timeout = %s, want 900ms", meta.ResolvedTimeout)
	}
	if strings.TrimSpace(meta.Trace) == "" {
		t.Fatalf("restored timeout resolution trace should not be empty: %#v", meta)
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

func TestSchedulerRecoveryBoundarySingleReentryThenFail(t *testing.T) {
	s, err := New(
		NewMemoryStore(),
		WithLeaseTimeout(60*time.Millisecond),
		WithRecoveryBoundary(RecoveryBoundaryConfig{
			Enabled:                  true,
			ResumeBoundary:           RecoveryResumeBoundaryNextAttemptOnly,
			InflightPolicy:           RecoveryInflightPolicyNoRewind,
			TimeoutReentryPolicy:     RecoveryTimeoutReentryPolicySingleReentryFail,
			TimeoutReentryMaxPerTask: 1,
		}),
		WithGovernance(GovernanceConfig{
			QoS: QoSModeFIFO,
			Fairness: FairnessConfig{
				MaxConsecutiveClaimsPerPriority: 3,
			},
			DLQ: DLQConfig{Enabled: true},
			Backoff: RetryBackoffConfig{
				Enabled:     false,
				Initial:     10 * time.Millisecond,
				Max:         20 * time.Millisecond,
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
		TaskID:      "task-recovery-boundary",
		RunID:       "run-recovery-boundary",
		MaxAttempts: 5,
	}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	if _, ok, err := s.Claim(ctx, "worker-rb-a"); err != nil || !ok {
		t.Fatalf("claim #1 failed: ok=%v err=%v", ok, err)
	}
	time.Sleep(90 * time.Millisecond)
	expiredFirst, err := s.ExpireLeases(ctx)
	if err != nil {
		t.Fatalf("expire leases #1: %v", err)
	}
	if len(expiredFirst) != 1 {
		t.Fatalf("expired #1 len = %d, want 1", len(expiredFirst))
	}
	if expiredFirst[0].Record.State != TaskStateQueued {
		t.Fatalf("first timeout should requeue once, state=%q", expiredFirst[0].Record.State)
	}

	if _, ok, err := s.Claim(ctx, "worker-rb-b"); err != nil || !ok {
		t.Fatalf("claim #2 failed: ok=%v err=%v", ok, err)
	}
	time.Sleep(90 * time.Millisecond)
	expiredSecond, err := s.ExpireLeases(ctx)
	if err != nil {
		t.Fatalf("expire leases #2: %v", err)
	}
	if len(expiredSecond) != 1 {
		t.Fatalf("expired #2 len = %d, want 1", len(expiredSecond))
	}
	if expiredSecond[0].Record.State != TaskStateFailed {
		t.Fatalf("second timeout should fail deterministically, state=%q", expiredSecond[0].Record.State)
	}
	if !strings.Contains(strings.ToLower(expiredSecond[0].Record.ErrorMessage), "reentry budget exhausted") {
		t.Fatalf("unexpected boundary exhaustion error message: %q", expiredSecond[0].Record.ErrorMessage)
	}

	if _, ok, err := s.Claim(ctx, "worker-rb-c"); err != nil || ok {
		t.Fatalf("task should not be claimable after reentry exhaustion: ok=%v err=%v", ok, err)
	}
	stats, err := s.Stats(ctx)
	if err != nil {
		t.Fatalf("stats failed: %v", err)
	}
	if stats.RecoveryTimeoutReentryTotal != 1 {
		t.Fatalf("recovery_timeout_reentry_total = %d, want 1", stats.RecoveryTimeoutReentryTotal)
	}
	if stats.RecoveryTimeoutReentryExhaustedTotal != 1 {
		t.Fatalf("recovery_timeout_reentry_exhausted_total = %d, want 1", stats.RecoveryTimeoutReentryExhaustedTotal)
	}
}

func TestSchedulerRestoreNoRewindRejectsTerminalTaskWithRunningAttempt(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	store.SetRecoveryBoundary(RecoveryBoundaryConfig{
		Enabled:                  true,
		ResumeBoundary:           RecoveryResumeBoundaryNextAttemptOnly,
		InflightPolicy:           RecoveryInflightPolicyNoRewind,
		TimeoutReentryPolicy:     RecoveryTimeoutReentryPolicySingleReentryFail,
		TimeoutReentryMaxPerTask: 1,
	})
	now := time.Now()
	err := store.Restore(ctx, StoreSnapshot{
		Backend: "memory",
		Tasks: []TaskRecord{
			{
				Task: Task{
					TaskID: "task-terminal-no-rewind",
					RunID:  "run-no-rewind",
				},
				State: TaskStateSucceeded,
				Attempts: []Attempt{
					{
						AttemptID:      "task-terminal-no-rewind-attempt-1",
						Attempt:        1,
						Status:         AttemptStatusRunning,
						StartedAt:      now.Add(-time.Second),
						HeartbeatAt:    now.Add(-500 * time.Millisecond),
						LeaseExpiresAt: now.Add(time.Second),
					},
				},
				CurrentAttempt: "",
				CreatedAt:      now.Add(-time.Second),
				UpdatedAt:      now,
			},
		},
		Stats: Stats{
			Backend: "memory",
		},
	})
	if !errors.Is(err, ErrSnapshotCorrupt) {
		t.Fatalf("restore error = %v, want ErrSnapshotCorrupt", err)
	}
}

func TestSchedulerNotBeforeSemanticsAndStats(t *testing.T) {
	s, err := New(NewMemoryStore(), WithLeaseTimeout(500*time.Millisecond))
	if err != nil {
		t.Fatalf("new scheduler: %v", err)
	}
	ctx := context.Background()
	now := time.Now()
	if _, err := s.Enqueue(ctx, Task{
		TaskID: "task-not-before-empty",
		RunID:  "run-not-before",
	}); err != nil {
		t.Fatalf("enqueue empty not_before: %v", err)
	}
	if _, err := s.Enqueue(ctx, Task{
		TaskID:    "task-not-before-past",
		RunID:     "run-not-before",
		NotBefore: now.Add(-time.Second),
	}); err != nil {
		t.Fatalf("enqueue past not_before: %v", err)
	}
	futureNotBefore := now.Add(100 * time.Millisecond)
	if _, err := s.Enqueue(ctx, Task{
		TaskID:    "task-not-before-future",
		RunID:     "run-not-before",
		NotBefore: futureNotBefore,
	}); err != nil {
		t.Fatalf("enqueue future not_before: %v", err)
	}

	claimed1, ok, err := s.Claim(ctx, "worker-not-before")
	if err != nil || !ok {
		t.Fatalf("claim #1 failed: ok=%v err=%v", ok, err)
	}
	if claimed1.Record.Task.TaskID != "task-not-before-empty" {
		t.Fatalf("claim #1 task=%q, want task-not-before-empty", claimed1.Record.Task.TaskID)
	}
	claimed2, ok, err := s.Claim(ctx, "worker-not-before")
	if err != nil || !ok {
		t.Fatalf("claim #2 failed: ok=%v err=%v", ok, err)
	}
	if claimed2.Record.Task.TaskID != "task-not-before-past" {
		t.Fatalf("claim #2 task=%q, want task-not-before-past", claimed2.Record.Task.TaskID)
	}
	if _, ok, err := s.Claim(ctx, "worker-not-before"); err != nil || ok {
		t.Fatalf("future task should not be claimable yet: ok=%v err=%v", ok, err)
	}
	time.Sleep(120 * time.Millisecond)
	claimed3, ok, err := s.Claim(ctx, "worker-not-before")
	if err != nil || !ok {
		t.Fatalf("claim #3 failed: ok=%v err=%v", ok, err)
	}
	if claimed3.Record.Task.TaskID != "task-not-before-future" {
		t.Fatalf("claim #3 task=%q, want task-not-before-future", claimed3.Record.Task.TaskID)
	}

	stats, err := s.Stats(ctx)
	if err != nil {
		t.Fatalf("stats failed: %v", err)
	}
	if stats.DelayedTaskTotal != 1 {
		t.Fatalf("delayed_task_total = %d, want 1", stats.DelayedTaskTotal)
	}
	if stats.DelayedClaimTotal != 1 {
		t.Fatalf("delayed_claim_total = %d, want 1", stats.DelayedClaimTotal)
	}
	if stats.DelayedWaitMsP95 <= 0 {
		t.Fatalf("delayed_wait_ms_p95 = %d, want > 0", stats.DelayedWaitMsP95)
	}
}

func TestSchedulerClaimComposesDelayedAndRetryGate(t *testing.T) {
	s, err := New(
		NewMemoryStore(),
		WithLeaseTimeout(500*time.Millisecond),
		WithGovernance(GovernanceConfig{
			QoS: QoSModeFIFO,
			Fairness: FairnessConfig{
				MaxConsecutiveClaimsPerPriority: 3,
			},
			DLQ: DLQConfig{Enabled: false},
			Backoff: RetryBackoffConfig{
				Enabled:     true,
				Initial:     60 * time.Millisecond,
				Max:         60 * time.Millisecond,
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
		TaskID:      "task-delayed-retry",
		RunID:       "run-delayed-retry",
		MaxAttempts: 3,
		NotBefore:   time.Now().Add(100 * time.Millisecond),
	}); err != nil {
		t.Fatalf("enqueue delayed task: %v", err)
	}
	if _, ok, err := s.Claim(ctx, "worker-a"); err != nil || ok {
		t.Fatalf("delayed task should not be claimable before not_before: ok=%v err=%v", ok, err)
	}
	time.Sleep(120 * time.Millisecond)
	claimed, ok, err := s.Claim(ctx, "worker-a")
	if err != nil || !ok {
		t.Fatalf("claim after not_before failed: ok=%v err=%v", ok, err)
	}
	if _, err := s.Requeue(ctx, claimed.Record.Task.TaskID, "retryable"); err != nil {
		t.Fatalf("requeue failed: %v", err)
	}
	if _, ok, err := s.Claim(ctx, "worker-b"); err != nil || ok {
		t.Fatalf("task should be blocked by retry backoff: ok=%v err=%v", ok, err)
	}
	time.Sleep(80 * time.Millisecond)
	if _, ok, err := s.Claim(ctx, "worker-b"); err != nil || !ok {
		t.Fatalf("task should be claimable after both gates pass: ok=%v err=%v", ok, err)
	}
}

func TestSchedulerDelayedTimelineReasons(t *testing.T) {
	collector := &testTimelineCollector{}
	s, err := New(
		NewMemoryStore(),
		WithTimelineEmitter(collector),
		WithLeaseTimeout(500*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("new scheduler: %v", err)
	}
	ctx := context.Background()
	if _, err := s.Enqueue(ctx, Task{
		TaskID:    "task-delayed-timeline",
		RunID:     "run-delayed-timeline",
		NotBefore: time.Now().Add(80 * time.Millisecond),
	}); err != nil {
		t.Fatalf("enqueue delayed task: %v", err)
	}
	if _, ok, err := s.Claim(ctx, "worker-timeline"); err != nil || ok {
		t.Fatalf("delayed task should not be claimable before not_before: ok=%v err=%v", ok, err)
	}
	time.Sleep(100 * time.Millisecond)
	if _, ok, err := s.Claim(ctx, "worker-timeline"); err != nil || !ok {
		t.Fatalf("delayed task should become claimable: ok=%v err=%v", ok, err)
	}

	requiredReasons := map[string]bool{
		ReasonDelayedEnqueue: false,
		ReasonDelayedWait:    false,
		ReasonDelayedReady:   false,
	}
	for _, ev := range collector.events {
		if ev.Type != types.EventTypeActionTimeline {
			continue
		}
		reason, _ := ev.Payload["reason"].(string)
		if _, ok := requiredReasons[reason]; ok {
			requiredReasons[reason] = true
			if ev.Payload["task_id"] == "" {
				t.Fatalf("missing task_id on delayed reason %q: %#v", reason, ev.Payload)
			}
		}
	}
	for reason, seen := range requiredReasons {
		if !seen {
			t.Fatalf("missing delayed timeline reason %q", reason)
		}
	}
}

func TestCanonicalReasonMapper(t *testing.T) {
	required := []string{
		ReasonEnqueue,
		ReasonDelayedEnqueue,
		ReasonDelayedWait,
		ReasonDelayedReady,
		ReasonClaim,
		ReasonHeartbeat,
		ReasonLeaseExpired,
		ReasonManualCancel,
		ReasonManualRetry,
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
