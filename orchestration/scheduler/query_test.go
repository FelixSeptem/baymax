package scheduler

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestTaskBoardQueryFilterDefaultsAndReadOnly(t *testing.T) {
	s := newTaskBoardQueryTestScheduler(t)
	ctx := context.Background()
	seedTaskBoardQueryFixture(t, s)

	before, err := s.Snapshot(ctx)
	if err != nil {
		t.Fatalf("snapshot before query failed: %v", err)
	}

	result, err := s.QueryTasks(ctx, TaskBoardQueryRequest{
		TeamID:   "team-a",
		State:    "QUEUED",
		Priority: "HIGH",
	})
	if err != nil {
		t.Fatalf("query with filters failed: %v", err)
	}
	if result.PageSize != DefaultTaskBoardQueryPageSize {
		t.Fatalf("default page_size = %d, want %d", result.PageSize, DefaultTaskBoardQueryPageSize)
	}
	if result.SortField != "updated_at" || result.SortOrder != "desc" {
		t.Fatalf("default sort mismatch: %#v", result)
	}
	if len(result.Items) != 1 || result.Items[0].Task.TaskID != "task-queued-high" {
		t.Fatalf("AND filter result mismatch: %#v", result.Items)
	}

	awaiting, err := s.QueryTasks(ctx, TaskBoardQueryRequest{
		TeamID: "team-a",
		State:  string(TaskStateAwaitingReport),
	})
	if err != nil {
		t.Fatalf("query awaiting_report failed: %v", err)
	}
	if len(awaiting.Items) != 1 || awaiting.Items[0].Task.TaskID != "task-running" {
		t.Fatalf("awaiting_report filter mismatch: %#v", awaiting.Items)
	}

	empty, err := s.QueryTasks(ctx, TaskBoardQueryRequest{TaskID: "task-missing"})
	if err != nil {
		t.Fatalf("query missing task should not error: %v", err)
	}
	if len(empty.Items) != 0 {
		t.Fatalf("missing task should return empty set, got %#v", empty.Items)
	}

	all, err := s.QueryTasks(ctx, TaskBoardQueryRequest{TeamID: "team-a"})
	if err != nil {
		t.Fatalf("query all failed: %v", err)
	}
	for i := 1; i < len(all.Items); i++ {
		if all.Items[i-1].UpdatedAt.Before(all.Items[i].UpdatedAt) {
			t.Fatalf("updated_at desc order drift: %#v", all.Items)
		}
	}

	after, err := s.Snapshot(ctx)
	if err != nil {
		t.Fatalf("snapshot after query failed: %v", err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("query path must be read-only, snapshot changed before=%#v after=%#v", before, after)
	}
}

func TestTaskBoardQueryValidationAndCursorDeterminism(t *testing.T) {
	s := newTaskBoardQueryTestScheduler(t)
	ctx := context.Background()
	seedTaskBoardQueryFixture(t, s)

	pageSize1 := 1
	baseReq := TaskBoardQueryRequest{
		TeamID:   "team-a",
		PageSize: &pageSize1,
		Sort: TaskBoardQuerySort{
			Field: "created_at",
			Order: "asc",
		},
	}

	first, err := s.QueryTasks(ctx, baseReq)
	if err != nil {
		t.Fatalf("first page query failed: %v", err)
	}
	if len(first.Items) != 1 || first.NextCursor == "" {
		t.Fatalf("first page mismatch: %#v", first)
	}
	if first.NextCursor == "1" {
		t.Fatalf("cursor must be opaque, got %q", first.NextCursor)
	}

	secondReq := baseReq
	secondReq.Cursor = first.NextCursor
	second, err := s.QueryTasks(ctx, secondReq)
	if err != nil {
		t.Fatalf("second page query failed: %v", err)
	}
	if len(second.Items) != 1 {
		t.Fatalf("second page mismatch: %#v", second)
	}

	secondAgain, err := s.QueryTasks(ctx, secondReq)
	if err != nil {
		t.Fatalf("second page replay failed: %v", err)
	}
	if !reflect.DeepEqual(second.Items, secondAgain.Items) || secondAgain.NextCursor != second.NextCursor {
		t.Fatalf("cursor traversal must be deterministic: %#v vs %#v", secondAgain, second)
	}

	thirdReq := baseReq
	thirdReq.Cursor = second.NextCursor
	third, err := s.QueryTasks(ctx, thirdReq)
	if err != nil {
		t.Fatalf("third page query failed: %v", err)
	}
	if len(third.Items) != 1 || third.NextCursor == "" {
		t.Fatalf("third page mismatch: %#v", third)
	}

	fourthReq := baseReq
	fourthReq.Cursor = third.NextCursor
	fourth, err := s.QueryTasks(ctx, fourthReq)
	if err != nil {
		t.Fatalf("fourth page query failed: %v", err)
	}
	if len(fourth.Items) != 1 || fourth.NextCursor != "" {
		t.Fatalf("fourth page mismatch: %#v", fourth)
	}

	if _, err := s.QueryTasks(ctx, TaskBoardQueryRequest{
		TeamID:   "team-a",
		PageSize: &pageSize1,
		Cursor:   "not-a-valid-cursor",
	}); err == nil {
		t.Fatal("expected fail-fast for malformed cursor")
	}
	if _, err := s.QueryTasks(ctx, TaskBoardQueryRequest{
		TeamID:   "team-b",
		PageSize: &pageSize1,
		Sort: TaskBoardQuerySort{
			Field: "created_at",
			Order: "asc",
		},
		Cursor: first.NextCursor,
	}); err == nil {
		t.Fatal("expected fail-fast for cursor query boundary mismatch")
	}

	tooLarge := 201
	if _, err := s.QueryTasks(ctx, TaskBoardQueryRequest{PageSize: &tooLarge}); err == nil {
		t.Fatal("expected fail-fast for page_size > 200")
	}
	zero := 0
	if _, err := s.QueryTasks(ctx, TaskBoardQueryRequest{PageSize: &zero}); err == nil {
		t.Fatal("expected fail-fast for page_size lower bound")
	}
	negative := -1
	if _, err := s.QueryTasks(ctx, TaskBoardQueryRequest{PageSize: &negative}); err == nil {
		t.Fatal("expected fail-fast for page_size negative value")
	}
	if _, err := s.QueryTasks(ctx, TaskBoardQueryRequest{State: "pending"}); err == nil {
		t.Fatal("expected fail-fast for unsupported state")
	}
	if _, err := s.QueryTasks(ctx, TaskBoardQueryRequest{
		Sort: TaskBoardQuerySort{Field: "task_id", Order: "desc"},
	}); err == nil {
		t.Fatal("expected fail-fast for unsupported sort field")
	}
	if _, err := s.QueryTasks(ctx, TaskBoardQueryRequest{
		Sort: TaskBoardQuerySort{Field: "updated_at", Order: "latest"},
	}); err == nil {
		t.Fatal("expected fail-fast for unsupported sort order")
	}
	if _, err := s.QueryTasks(ctx, TaskBoardQueryRequest{
		TimeRange: &TaskBoardQueryTimeRange{
			Start: time.Unix(200, 0),
			End:   time.Unix(100, 0),
		},
	}); err == nil {
		t.Fatal("expected fail-fast for invalid time range")
	}

	normalized, err := normalizeTaskBoardQuery(baseReq)
	if err != nil {
		t.Fatalf("normalize base request failed: %v", err)
	}
	badCursor, err := encodeTaskBoardCursor(taskBoardQueryCursor{
		Offset:    999,
		QueryHash: taskBoardQueryHash(normalized),
	})
	if err != nil {
		t.Fatalf("encode bad cursor failed: %v", err)
	}
	invalidBoundaryReq := baseReq
	invalidBoundaryReq.Cursor = badCursor
	if _, err := s.QueryTasks(ctx, invalidBoundaryReq); err == nil {
		t.Fatal("expected fail-fast for cursor offset out of boundary")
	}
}

func newTaskBoardQueryTestScheduler(t *testing.T) *Scheduler {
	t.Helper()

	s, err := New(
		NewMemoryStore(),
		WithLeaseTimeout(2*time.Second),
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

	base := time.Unix(1_700_000_000, 0).UTC()
	tick := 0
	s.now = func() time.Time {
		ts := base.Add(time.Duration(tick) * time.Second)
		tick++
		return ts
	}
	return s
}

func seedTaskBoardQueryFixture(t *testing.T, s *Scheduler) {
	t.Helper()
	ctx := context.Background()

	if _, err := s.Enqueue(ctx, Task{
		TaskID:      "task-running",
		RunID:       "run-a-1",
		WorkflowID:  "wf-a",
		TeamID:      "team-a",
		AgentID:     "agent-a",
		PeerID:      "peer-a",
		ParentRunID: "parent-a",
		Priority:    TaskPriorityNormal,
	}); err != nil {
		t.Fatalf("enqueue running task: %v", err)
	}
	claimedRunning, ok, err := s.Claim(ctx, "worker-running")
	if err != nil || !ok {
		t.Fatalf("claim running task failed: ok=%v err=%v", ok, err)
	}
	if _, err := s.MarkAwaitingReport(ctx, claimedRunning.Record.Task.TaskID, claimedRunning.Attempt.AttemptID); err != nil {
		t.Fatalf("mark awaiting_report task failed: %v", err)
	}

	if _, err := s.Enqueue(ctx, Task{
		TaskID:      "task-dead-letter",
		RunID:       "run-a-2",
		WorkflowID:  "wf-a",
		TeamID:      "team-a",
		AgentID:     "agent-a",
		PeerID:      "peer-a",
		ParentRunID: "parent-a",
		Priority:    TaskPriorityLow,
		MaxAttempts: 1,
	}); err != nil {
		t.Fatalf("enqueue dead-letter task: %v", err)
	}
	claimedDead, ok, err := s.Claim(ctx, "worker-dead-letter")
	if err != nil || !ok {
		t.Fatalf("claim dead-letter task failed: ok=%v err=%v", ok, err)
	}
	if _, err := s.Requeue(ctx, claimedDead.Record.Task.TaskID, "retryable_error"); err != nil {
		t.Fatalf("requeue dead-letter task failed: %v", err)
	}

	if _, err := s.Enqueue(ctx, Task{
		TaskID:      "task-succeeded",
		RunID:       "run-b-1",
		WorkflowID:  "wf-b",
		TeamID:      "team-b",
		AgentID:     "agent-b",
		PeerID:      "peer-b",
		ParentRunID: "parent-b",
		Priority:    TaskPriorityHigh,
	}); err != nil {
		t.Fatalf("enqueue succeeded task: %v", err)
	}
	claimedSucceeded, ok, err := s.Claim(ctx, "worker-succeeded")
	if err != nil || !ok {
		t.Fatalf("claim succeeded task failed: ok=%v err=%v", ok, err)
	}
	if _, err := s.Complete(ctx, TerminalCommit{
		TaskID:      claimedSucceeded.Record.Task.TaskID,
		AttemptID:   claimedSucceeded.Attempt.AttemptID,
		Status:      TaskStateSucceeded,
		Result:      map[string]any{"ok": true},
		CommittedAt: s.nowTime(),
	}); err != nil {
		t.Fatalf("complete succeeded task failed: %v", err)
	}

	for _, task := range []Task{
		{
			TaskID:      "task-queued-high",
			RunID:       "run-a-3",
			WorkflowID:  "wf-a",
			TeamID:      "team-a",
			AgentID:     "agent-a",
			PeerID:      "peer-a",
			ParentRunID: "parent-a",
			Priority:    TaskPriorityHigh,
		},
		{
			TaskID:      "task-queued-low",
			RunID:       "run-a-4",
			WorkflowID:  "wf-a",
			TeamID:      "team-a",
			AgentID:     "agent-a",
			PeerID:      "peer-a",
			ParentRunID: "parent-a",
			Priority:    TaskPriorityLow,
		},
	} {
		if _, err := s.Enqueue(ctx, task); err != nil {
			t.Fatalf("enqueue queued task %q: %v", task.TaskID, err)
		}
	}
}
