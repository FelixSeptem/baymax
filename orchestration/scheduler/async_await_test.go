package scheduler

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestSchedulerAsyncAwaitTransitionAndCommit(t *testing.T) {
	s, err := New(
		NewMemoryStore(),
		WithLeaseTimeout(2*time.Second),
		WithAsyncAwait(AsyncAwaitConfig{
			ReportTimeout:    15 * time.Minute,
			LateReportPolicy: AsyncLateReportPolicyDropAndRecord,
			TimeoutTerminal:  TaskStateFailed,
		}),
	)
	if err != nil {
		t.Fatalf("new scheduler failed: %v", err)
	}

	now := time.Unix(1_700_000_000, 0).UTC()
	s.now = func() time.Time { return now }

	ctx := context.Background()
	if _, err := s.Enqueue(ctx, Task{TaskID: "task-await-ok", RunID: "run-await-ok"}); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	claimed, ok, err := s.Claim(ctx, "worker-await-ok")
	if err != nil || !ok {
		t.Fatalf("claim failed: ok=%v err=%v", ok, err)
	}

	record, err := s.MarkAwaitingReport(ctx, claimed.Record.Task.TaskID, claimed.Attempt.AttemptID)
	if err != nil {
		t.Fatalf("mark awaiting_report failed: %v", err)
	}
	if record.State != TaskStateAwaitingReport {
		t.Fatalf("state=%q, want awaiting_report", record.State)
	}
	if record.ReportTimeoutAt.Sub(record.AwaitingReportSince) != 15*time.Minute {
		t.Fatalf("report timeout delta=%s, want 15m", record.ReportTimeoutAt.Sub(record.AwaitingReportSince))
	}

	committedAt := now.Add(10 * time.Second)
	result, err := s.CommitAsyncReportTerminal(ctx, TerminalCommit{
		TaskID:      claimed.Record.Task.TaskID,
		AttemptID:   claimed.Attempt.AttemptID,
		Status:      TaskStateSucceeded,
		Result:      map[string]any{"ok": true},
		CommittedAt: committedAt,
	})
	if err != nil {
		t.Fatalf("commit async report failed: %v", err)
	}
	if result.Duplicate || result.LateReport {
		t.Fatalf("unexpected commit meta: %#v", result)
	}
	if result.Record.State != TaskStateSucceeded {
		t.Fatalf("terminal state=%q, want succeeded", result.Record.State)
	}

	stats, err := s.Stats(ctx)
	if err != nil {
		t.Fatalf("stats failed: %v", err)
	}
	if stats.AsyncAwaitTotal != 1 {
		t.Fatalf("async_await_total=%d, want 1", stats.AsyncAwaitTotal)
	}
}

func TestSchedulerCommitAsyncReportRequiresAwaitingState(t *testing.T) {
	s, err := New(NewMemoryStore(), WithLeaseTimeout(2*time.Second))
	if err != nil {
		t.Fatalf("new scheduler failed: %v", err)
	}
	ctx := context.Background()
	if _, err := s.Enqueue(ctx, Task{TaskID: "task-await-required", RunID: "run-await-required"}); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	claimed, ok, err := s.Claim(ctx, "worker-await-required")
	if err != nil || !ok {
		t.Fatalf("claim failed: ok=%v err=%v", ok, err)
	}
	_, err = s.CommitAsyncReportTerminal(ctx, TerminalCommit{
		TaskID:      claimed.Record.Task.TaskID,
		AttemptID:   claimed.Attempt.AttemptID,
		Status:      TaskStateSucceeded,
		Result:      map[string]any{"ok": true},
		CommittedAt: time.Now().UTC(),
	})
	if !errors.Is(err, ErrTaskNotAwaitingReport) {
		t.Fatalf("commit async report error=%v, want %v", err, ErrTaskNotAwaitingReport)
	}
}

func TestSchedulerAsyncAwaitTimeoutTerminalizationAndLateReportDrop(t *testing.T) {
	s, err := New(
		NewMemoryStore(),
		WithLeaseTimeout(2*time.Second),
		WithAsyncAwait(AsyncAwaitConfig{
			ReportTimeout:    1 * time.Minute,
			LateReportPolicy: AsyncLateReportPolicyDropAndRecord,
			TimeoutTerminal:  TaskStateFailed,
		}),
	)
	if err != nil {
		t.Fatalf("new scheduler failed: %v", err)
	}

	now := time.Unix(1_700_000_100, 0).UTC()
	s.now = func() time.Time { return now }

	ctx := context.Background()
	if _, err := s.Enqueue(ctx, Task{TaskID: "task-await-timeout", RunID: "run-await-timeout"}); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	claimed, ok, err := s.Claim(ctx, "worker-await-timeout")
	if err != nil || !ok {
		t.Fatalf("claim failed: ok=%v err=%v", ok, err)
	}
	if _, err := s.MarkAwaitingReport(ctx, claimed.Record.Task.TaskID, claimed.Attempt.AttemptID); err != nil {
		t.Fatalf("mark awaiting_report failed: %v", err)
	}

	now = now.Add(2 * time.Minute)
	expired, err := s.ExpireLeases(ctx)
	if err != nil {
		t.Fatalf("expire leases failed: %v", err)
	}
	if len(expired) != 1 {
		t.Fatalf("expired count=%d, want 1", len(expired))
	}
	if expired[0].Record.State != TaskStateFailed {
		t.Fatalf("timeout terminal state=%q, want failed", expired[0].Record.State)
	}

	stats, err := s.Stats(ctx)
	if err != nil {
		t.Fatalf("stats failed: %v", err)
	}
	if stats.AsyncTimeoutTotal != 1 {
		t.Fatalf("async_timeout_total=%d, want 1", stats.AsyncTimeoutTotal)
	}

	late, err := s.CommitAsyncReportTerminal(ctx, TerminalCommit{
		TaskID:      claimed.Record.Task.TaskID,
		AttemptID:   claimed.Attempt.AttemptID,
		Status:      TaskStateSucceeded,
		Result:      map[string]any{"ok": true},
		CommittedAt: now.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("late report commit should be handled by drop_and_record, got err=%v", err)
	}
	if !late.LateReport || !late.Duplicate {
		t.Fatalf("late report meta mismatch: %#v", late)
	}
	after, found, err := s.Get(ctx, claimed.Record.Task.TaskID)
	if err != nil || !found {
		t.Fatalf("get after late report failed: found=%v err=%v", found, err)
	}
	if after.State != TaskStateFailed {
		t.Fatalf("late report must not mutate terminal state, got=%q", after.State)
	}
}

func TestSchedulerAsyncAwaitTimeoutDeadLetterWhenConfigured(t *testing.T) {
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
				Enabled: false,
			},
		}),
		WithAsyncAwait(AsyncAwaitConfig{
			ReportTimeout:    1 * time.Minute,
			LateReportPolicy: AsyncLateReportPolicyDropAndRecord,
			TimeoutTerminal:  TaskStateDeadLetter,
		}),
	)
	if err != nil {
		t.Fatalf("new scheduler failed: %v", err)
	}

	now := time.Unix(1_700_000_200, 0).UTC()
	s.now = func() time.Time { return now }

	ctx := context.Background()
	if _, err := s.Enqueue(ctx, Task{TaskID: "task-await-dlq", RunID: "run-await-dlq"}); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	claimed, ok, err := s.Claim(ctx, "worker-await-dlq")
	if err != nil || !ok {
		t.Fatalf("claim failed: ok=%v err=%v", ok, err)
	}
	if _, err := s.MarkAwaitingReport(ctx, claimed.Record.Task.TaskID, claimed.Attempt.AttemptID); err != nil {
		t.Fatalf("mark awaiting_report failed: %v", err)
	}

	now = now.Add(2 * time.Minute)
	expired, err := s.ExpireLeases(ctx)
	if err != nil {
		t.Fatalf("expire leases failed: %v", err)
	}
	if len(expired) != 1 {
		t.Fatalf("expired count=%d, want 1", len(expired))
	}
	if expired[0].Record.State != TaskStateDeadLetter {
		t.Fatalf("timeout terminal state=%q, want dead_letter", expired[0].Record.State)
	}
	if expired[0].Record.DeadLetterCode != "async_report_timeout" {
		t.Fatalf("dead_letter_code=%q, want async_report_timeout", expired[0].Record.DeadLetterCode)
	}
}

func TestSchedulerAsyncAwaitSnapshotRestoreKeepsAwaitingSemantics(t *testing.T) {
	original, err := New(
		NewMemoryStore(),
		WithLeaseTimeout(2*time.Second),
		WithAsyncAwait(AsyncAwaitConfig{
			ReportTimeout:    15 * time.Minute,
			LateReportPolicy: AsyncLateReportPolicyDropAndRecord,
			TimeoutTerminal:  TaskStateFailed,
		}),
	)
	if err != nil {
		t.Fatalf("new scheduler failed: %v", err)
	}
	now := time.Unix(1_700_000_300, 0).UTC()
	original.now = func() time.Time { return now }

	ctx := context.Background()
	if _, err := original.Enqueue(ctx, Task{TaskID: "task-await-restore", RunID: "run-await-restore"}); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	claimed, ok, err := original.Claim(ctx, "worker-await-restore")
	if err != nil || !ok {
		t.Fatalf("claim failed: ok=%v err=%v", ok, err)
	}
	if _, err := original.MarkAwaitingReport(ctx, claimed.Record.Task.TaskID, claimed.Attempt.AttemptID); err != nil {
		t.Fatalf("mark awaiting_report failed: %v", err)
	}

	snapshot, err := original.Snapshot(ctx)
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}
	restored, err := New(
		NewMemoryStore(),
		WithLeaseTimeout(2*time.Second),
		WithAsyncAwait(AsyncAwaitConfig{
			ReportTimeout:    15 * time.Minute,
			LateReportPolicy: AsyncLateReportPolicyDropAndRecord,
			TimeoutTerminal:  TaskStateFailed,
		}),
	)
	if err != nil {
		t.Fatalf("new restored scheduler failed: %v", err)
	}
	restored.now = func() time.Time { return now }
	if err := restored.Restore(ctx, snapshot); err != nil {
		t.Fatalf("restore failed: %v", err)
	}

	record, found, err := restored.Get(ctx, claimed.Record.Task.TaskID)
	if err != nil || !found {
		t.Fatalf("get restored task failed: found=%v err=%v", found, err)
	}
	if record.State != TaskStateAwaitingReport {
		t.Fatalf("restored state=%q, want awaiting_report", record.State)
	}
	if record.ReportTimeoutAt.IsZero() {
		t.Fatal("restored report_timeout_at should not be zero")
	}

	result, err := restored.CommitAsyncReportTerminal(ctx, TerminalCommit{
		TaskID:      claimed.Record.Task.TaskID,
		AttemptID:   claimed.Attempt.AttemptID,
		Status:      TaskStateSucceeded,
		Result:      map[string]any{"ok": true},
		CommittedAt: now.Add(2 * time.Second),
	})
	if err != nil {
		t.Fatalf("commit async report after restore failed: %v", err)
	}
	if result.Record.State != TaskStateSucceeded {
		t.Fatalf("terminal state after restore=%q, want succeeded", result.Record.State)
	}
}
