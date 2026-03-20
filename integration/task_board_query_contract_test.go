package integration

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/orchestration/scheduler"
)

func TestTaskBoardQueryContractMemoryFileParity(t *testing.T) {
	ctx := context.Background()
	memScheduler := newTaskBoardContractScheduler(t, scheduler.NewMemoryStore())
	seedTaskBoardContractFixture(t, memScheduler)

	snapshot, err := memScheduler.Snapshot(ctx)
	if err != nil {
		t.Fatalf("memory snapshot failed: %v", err)
	}

	fileStore, err := scheduler.NewFileStore(filepath.Join(t.TempDir(), "scheduler-state.json"))
	if err != nil {
		t.Fatalf("new file store failed: %v", err)
	}
	fileScheduler := newTaskBoardContractScheduler(t, fileStore)
	if err := fileScheduler.Restore(ctx, snapshot); err != nil {
		t.Fatalf("restore snapshot to file backend failed: %v", err)
	}

	pageSize := 2
	req := scheduler.TaskBoardQueryRequest{
		TeamID:   "team-a",
		PageSize: &pageSize,
		Sort: scheduler.TaskBoardQuerySort{
			Field: "updated_at",
			Order: "desc",
		},
	}
	memPages := collectTaskBoardQueryPages(t, memScheduler, req)
	filePages := collectTaskBoardQueryPages(t, fileScheduler, req)
	if !reflect.DeepEqual(memPages, filePages) {
		t.Fatalf("memory/file query parity mismatch: memory=%#v file=%#v", memPages, filePages)
	}
}

func TestTaskBoardQueryContractSnapshotRestoreStability(t *testing.T) {
	ctx := context.Background()
	beforeScheduler := newTaskBoardContractScheduler(t, scheduler.NewMemoryStore())
	seedTaskBoardContractFixture(t, beforeScheduler)

	pageSize := 3
	req := scheduler.TaskBoardQueryRequest{
		TeamID:   "team-a",
		PageSize: &pageSize,
		Sort: scheduler.TaskBoardQuerySort{
			Field: "created_at",
			Order: "asc",
		},
	}
	beforePages := collectTaskBoardQueryPages(t, beforeScheduler, req)

	snapshot, err := beforeScheduler.Snapshot(ctx)
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}
	restoredScheduler := newTaskBoardContractScheduler(t, scheduler.NewMemoryStore())
	if err := restoredScheduler.Restore(ctx, snapshot); err != nil {
		t.Fatalf("restore failed: %v", err)
	}

	afterPages := collectTaskBoardQueryPages(t, restoredScheduler, req)
	if !reflect.DeepEqual(beforePages, afterPages) {
		t.Fatalf("snapshot restore query stability mismatch: before=%#v after=%#v", beforePages, afterPages)
	}
}

func TestTaskBoardQueryContractAwaitingReportStateFilter(t *testing.T) {
	ctx := context.Background()
	s := newTaskBoardContractScheduler(t, scheduler.NewMemoryStore())
	seedTaskBoardContractFixture(t, s)

	result, err := s.QueryTasks(ctx, scheduler.TaskBoardQueryRequest{
		TeamID: "team-a",
		State:  string(scheduler.TaskStateAwaitingReport),
	})
	if err != nil {
		t.Fatalf("query awaiting_report failed: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("awaiting_report query items = %d, want 1", len(result.Items))
	}
	if result.Items[0].Task.TaskID != "task-int-running" || result.Items[0].State != scheduler.TaskStateAwaitingReport {
		t.Fatalf("awaiting_report query mismatch: %#v", result.Items[0])
	}
}

func newTaskBoardContractScheduler(t *testing.T, store scheduler.QueueStore) *scheduler.Scheduler {
	t.Helper()
	s, err := scheduler.New(
		store,
		scheduler.WithLeaseTimeout(2*time.Second),
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
		t.Fatalf("new scheduler failed: %v", err)
	}
	return s
}

func seedTaskBoardContractFixture(t *testing.T, s *scheduler.Scheduler) {
	t.Helper()
	ctx := context.Background()

	if _, err := s.Enqueue(ctx, scheduler.Task{
		TaskID:      "task-int-running",
		RunID:       "run-int-a-1",
		WorkflowID:  "wf-int-a",
		TeamID:      "team-a",
		AgentID:     "agent-a",
		PeerID:      "peer-a",
		ParentRunID: "parent-a",
		Priority:    scheduler.TaskPriorityNormal,
	}); err != nil {
		t.Fatalf("enqueue running task: %v", err)
	}
	claimedRunning, ok, err := s.Claim(ctx, "worker-int-running")
	if err != nil || !ok {
		t.Fatalf("claim running task failed: ok=%v err=%v", ok, err)
	}
	if _, err := s.MarkAwaitingReport(ctx, claimedRunning.Record.Task.TaskID, claimedRunning.Attempt.AttemptID); err != nil {
		t.Fatalf("mark awaiting_report task failed: %v", err)
	}

	if _, err := s.Enqueue(ctx, scheduler.Task{
		TaskID:      "task-int-dead-letter",
		RunID:       "run-int-a-2",
		WorkflowID:  "wf-int-a",
		TeamID:      "team-a",
		AgentID:     "agent-a",
		PeerID:      "peer-a",
		ParentRunID: "parent-a",
		Priority:    scheduler.TaskPriorityLow,
		MaxAttempts: 1,
	}); err != nil {
		t.Fatalf("enqueue dead-letter task: %v", err)
	}
	claimedDead, ok, err := s.Claim(ctx, "worker-int-dead")
	if err != nil || !ok {
		t.Fatalf("claim dead-letter task failed: ok=%v err=%v", ok, err)
	}
	if _, err := s.Requeue(ctx, claimedDead.Record.Task.TaskID, "retryable_error"); err != nil {
		t.Fatalf("requeue dead-letter task failed: %v", err)
	}

	if _, err := s.Enqueue(ctx, scheduler.Task{
		TaskID:      "task-int-succeeded",
		RunID:       "run-int-b-1",
		WorkflowID:  "wf-int-b",
		TeamID:      "team-b",
		AgentID:     "agent-b",
		PeerID:      "peer-b",
		ParentRunID: "parent-b",
		Priority:    scheduler.TaskPriorityHigh,
	}); err != nil {
		t.Fatalf("enqueue succeeded task: %v", err)
	}
	claimedSucceeded, ok, err := s.Claim(ctx, "worker-int-succeeded")
	if err != nil || !ok {
		t.Fatalf("claim succeeded task failed: ok=%v err=%v", ok, err)
	}
	if _, err := s.Complete(ctx, scheduler.TerminalCommit{
		TaskID:      claimedSucceeded.Record.Task.TaskID,
		AttemptID:   claimedSucceeded.Attempt.AttemptID,
		Status:      scheduler.TaskStateSucceeded,
		Result:      map[string]any{"ok": true},
		CommittedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("complete succeeded task failed: %v", err)
	}

	for _, task := range []scheduler.Task{
		{
			TaskID:      "task-int-queued-high",
			RunID:       "run-int-a-3",
			WorkflowID:  "wf-int-a",
			TeamID:      "team-a",
			AgentID:     "agent-a",
			PeerID:      "peer-a",
			ParentRunID: "parent-a",
			Priority:    scheduler.TaskPriorityHigh,
		},
		{
			TaskID:      "task-int-queued-low",
			RunID:       "run-int-a-4",
			WorkflowID:  "wf-int-a",
			TeamID:      "team-a",
			AgentID:     "agent-a",
			PeerID:      "peer-a",
			ParentRunID: "parent-a",
			Priority:    scheduler.TaskPriorityLow,
		},
	} {
		if _, err := s.Enqueue(ctx, task); err != nil {
			t.Fatalf("enqueue queued task %q: %v", task.TaskID, err)
		}
	}
}

func collectTaskBoardQueryPages(t *testing.T, s *scheduler.Scheduler, req scheduler.TaskBoardQueryRequest) []scheduler.TaskBoardQueryResult {
	t.Helper()
	ctx := context.Background()
	pages := make([]scheduler.TaskBoardQueryResult, 0)
	seenCursor := map[string]struct{}{}
	current := req
	for {
		page, err := s.QueryTasks(ctx, current)
		if err != nil {
			t.Fatalf("query page failed: %v", err)
		}
		pages = append(pages, page)
		next := page.NextCursor
		if next == "" {
			break
		}
		if _, exists := seenCursor[next]; exists {
			t.Fatalf("cursor must advance deterministically without loops: %q", next)
		}
		seenCursor[next] = struct{}{}
		current.Cursor = next
	}
	return pages
}
