package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/a2a"
	"github.com/FelixSeptem/baymax/orchestration/scheduler"
)

type tailGovernanceModeSummary struct {
	state          scheduler.TaskState
	queueTotal     int
	claimTotal     int
	completeTotal  int
	delayedTotal   int
	delayedClaim   int
	delayedWaitP95 int64
}

func TestTailGovernanceCrossModeMatrixRunStream(t *testing.T) {
	t.Helper()
	assertEquivalent := func(mode string, run, stream tailGovernanceModeSummary) {
		t.Helper()
		if run.state != stream.state {
			t.Fatalf("%s terminal mismatch run=%q stream=%q", mode, run.state, stream.state)
		}
		if run.queueTotal != stream.queueTotal ||
			run.claimTotal != stream.claimTotal ||
			run.completeTotal != stream.completeTotal ||
			run.delayedTotal != stream.delayedTotal ||
			run.delayedClaim != stream.delayedClaim {
			t.Fatalf("%s aggregate mismatch run=%#v stream=%#v", mode, run, stream)
		}
		if mode == "delayed" {
			diff := run.delayedWaitP95 - stream.delayedWaitP95
			if diff < 0 {
				diff = -diff
			}
			if diff > 40 {
				t.Fatalf("%s delayed wait p95 diff too large run=%d stream=%d", mode, run.delayedWaitP95, stream.delayedWaitP95)
			}
		}
	}

	syncExec := func(taskID string) (tailGovernanceModeSummary, error) {
		ctx := context.Background()
		s, err := scheduler.New(scheduler.NewMemoryStore(), scheduler.WithLeaseTimeout(500*time.Millisecond))
		if err != nil {
			return tailGovernanceModeSummary{}, err
		}
		if _, err := s.Enqueue(ctx, scheduler.Task{TaskID: taskID, RunID: "run-tail-governance-sync"}); err != nil {
			return tailGovernanceModeSummary{}, err
		}
		claimed, ok, err := s.Claim(ctx, "worker-tail-governance-sync")
		if err != nil || !ok {
			return tailGovernanceModeSummary{}, errors.New("sync claim failed")
		}
		if _, err := s.Complete(ctx, scheduler.TerminalCommit{
			TaskID:      claimed.Record.Task.TaskID,
			AttemptID:   claimed.Attempt.AttemptID,
			Status:      scheduler.TaskStateSucceeded,
			Result:      map[string]any{"ok": true},
			CommittedAt: time.Now(),
		}); err != nil {
			return tailGovernanceModeSummary{}, err
		}
		record, ok, err := s.Get(ctx, taskID)
		if err != nil || !ok {
			return tailGovernanceModeSummary{}, errors.New("sync get failed")
		}
		stats, err := s.Stats(ctx)
		if err != nil {
			return tailGovernanceModeSummary{}, err
		}
		return tailGovernanceModeSummary{
			state:         record.State,
			queueTotal:    stats.QueueTotal,
			claimTotal:    stats.ClaimTotal,
			completeTotal: stats.CompleteTotal,
		}, nil
	}

	asyncExec := func(taskID string) (tailGovernanceModeSummary, error) {
		ctx := context.Background()
		s, err := scheduler.New(scheduler.NewMemoryStore(), scheduler.WithLeaseTimeout(500*time.Millisecond))
		if err != nil {
			return tailGovernanceModeSummary{}, err
		}
		if _, err := s.Enqueue(ctx, scheduler.Task{TaskID: taskID, RunID: "run-tail-governance-async"}); err != nil {
			return tailGovernanceModeSummary{}, err
		}
		claimed, ok, err := s.Claim(ctx, "worker-tail-governance-async")
		if err != nil || !ok {
			return tailGovernanceModeSummary{}, errors.New("async claim failed")
		}
		exec, err := scheduler.ExecutionFromAsyncReport(claimed, a2a.AsyncReport{
			ReportKey:  "report-" + taskID,
			OutcomeKey: "succeeded|ok",
			TaskID:     claimed.Record.Task.TaskID,
			AttemptID:  claimed.Attempt.AttemptID,
			Status:     a2a.StatusSucceeded,
			Result:     map[string]any{"ok": true},
		})
		if err != nil {
			return tailGovernanceModeSummary{}, err
		}
		if _, err := s.Complete(ctx, exec.Commit); err != nil {
			return tailGovernanceModeSummary{}, err
		}
		record, ok, err := s.Get(ctx, taskID)
		if err != nil || !ok {
			return tailGovernanceModeSummary{}, errors.New("async get failed")
		}
		stats, err := s.Stats(ctx)
		if err != nil {
			return tailGovernanceModeSummary{}, err
		}
		return tailGovernanceModeSummary{
			state:         record.State,
			queueTotal:    stats.QueueTotal,
			claimTotal:    stats.ClaimTotal,
			completeTotal: stats.CompleteTotal,
		}, nil
	}

	delayedExec := func(taskID string) (tailGovernanceModeSummary, error) {
		ctx := context.Background()
		s, err := scheduler.New(scheduler.NewMemoryStore(), scheduler.WithLeaseTimeout(500*time.Millisecond))
		if err != nil {
			return tailGovernanceModeSummary{}, err
		}
		if _, err := s.Enqueue(ctx, scheduler.Task{
			TaskID:    taskID,
			RunID:     "run-tail-governance-delayed",
			NotBefore: time.Now().Add(90 * time.Millisecond),
		}); err != nil {
			return tailGovernanceModeSummary{}, err
		}
		if _, ok, err := s.Claim(ctx, "worker-tail-governance-delayed"); err != nil {
			return tailGovernanceModeSummary{}, err
		} else if ok {
			return tailGovernanceModeSummary{}, errors.New("delayed task claimed before not_before")
		}
		time.Sleep(120 * time.Millisecond)
		claimed, ok, err := s.Claim(ctx, "worker-tail-governance-delayed")
		if err != nil || !ok {
			return tailGovernanceModeSummary{}, errors.New("delayed claim failed after boundary")
		}
		if _, err := s.Complete(ctx, scheduler.TerminalCommit{
			TaskID:      claimed.Record.Task.TaskID,
			AttemptID:   claimed.Attempt.AttemptID,
			Status:      scheduler.TaskStateSucceeded,
			Result:      map[string]any{"ok": true},
			CommittedAt: time.Now(),
		}); err != nil {
			return tailGovernanceModeSummary{}, err
		}
		record, ok, err := s.Get(ctx, taskID)
		if err != nil || !ok {
			return tailGovernanceModeSummary{}, errors.New("delayed get failed")
		}
		stats, err := s.Stats(ctx)
		if err != nil {
			return tailGovernanceModeSummary{}, err
		}
		return tailGovernanceModeSummary{
			state:          record.State,
			queueTotal:     stats.QueueTotal,
			claimTotal:     stats.ClaimTotal,
			completeTotal:  stats.CompleteTotal,
			delayedTotal:   stats.DelayedTaskTotal,
			delayedClaim:   stats.DelayedClaimTotal,
			delayedWaitP95: stats.DelayedWaitMsP95,
		}, nil
	}

	syncRun, err := syncExec("task-tail-governance-sync-run")
	if err != nil {
		t.Fatalf("sync run failed: %v", err)
	}
	syncStream, err := syncExec("task-tail-governance-sync-stream")
	if err != nil {
		t.Fatalf("sync stream failed: %v", err)
	}
	assertEquivalent("sync", syncRun, syncStream)

	asyncRun, err := asyncExec("task-tail-governance-async-run")
	if err != nil {
		t.Fatalf("async run failed: %v", err)
	}
	asyncStream, err := asyncExec("task-tail-governance-async-stream")
	if err != nil {
		t.Fatalf("async stream failed: %v", err)
	}
	assertEquivalent("async", asyncRun, asyncStream)

	delayedRun, err := delayedExec("task-tail-governance-delayed-run")
	if err != nil {
		t.Fatalf("delayed run failed: %v", err)
	}
	delayedStream, err := delayedExec("task-tail-governance-delayed-stream")
	if err != nil {
		t.Fatalf("delayed stream failed: %v", err)
	}
	assertEquivalent("delayed", delayedRun, delayedStream)
}

type tailGovernanceQoSRecoverySummary struct {
	qosMode                  string
	priorityClaimTotal       int
	fairnessYieldTotal       int
	completeTotal            int
	duplicateTerminalCommits int
}

func TestTailGovernanceQoSRecoveryRunStreamSemanticEquivalence(t *testing.T) {
	exec := func(label string) (tailGovernanceQoSRecoverySummary, error) {
		ctx := context.Background()
		governance := scheduler.GovernanceConfig{
			QoS: scheduler.QoSModePriority,
			Fairness: scheduler.FairnessConfig{
				MaxConsecutiveClaimsPerPriority: 1,
			},
		}
		s, err := scheduler.New(
			scheduler.NewMemoryStore(),
			scheduler.WithLeaseTimeout(500*time.Millisecond),
			scheduler.WithGovernance(governance),
		)
		if err != nil {
			return tailGovernanceQoSRecoverySummary{}, err
		}
		if _, err := s.Enqueue(ctx, scheduler.Task{
			TaskID:   "task-tail-governance-qos-high-" + label,
			RunID:    "run-tail-governance-qos-" + label,
			Priority: scheduler.TaskPriorityHigh,
		}); err != nil {
			return tailGovernanceQoSRecoverySummary{}, err
		}
		if _, err := s.Enqueue(ctx, scheduler.Task{
			TaskID:   "task-tail-governance-qos-low-" + label,
			RunID:    "run-tail-governance-qos-" + label,
			Priority: scheduler.TaskPriorityLow,
		}); err != nil {
			return tailGovernanceQoSRecoverySummary{}, err
		}

		first, ok, err := s.Claim(ctx, "worker-tail-governance-qos-"+label)
		if err != nil || !ok {
			return tailGovernanceQoSRecoverySummary{}, errors.New("qos first claim failed")
		}
		if _, err := s.Complete(ctx, scheduler.TerminalCommit{
			TaskID:      first.Record.Task.TaskID,
			AttemptID:   first.Attempt.AttemptID,
			Status:      scheduler.TaskStateSucceeded,
			Result:      map[string]any{"ok": true},
			CommittedAt: time.Now(),
		}); err != nil {
			return tailGovernanceQoSRecoverySummary{}, err
		}

		snap, err := s.Snapshot(ctx)
		if err != nil {
			return tailGovernanceQoSRecoverySummary{}, err
		}
		restored, err := scheduler.New(
			scheduler.NewMemoryStore(),
			scheduler.WithLeaseTimeout(500*time.Millisecond),
			scheduler.WithGovernance(governance),
		)
		if err != nil {
			return tailGovernanceQoSRecoverySummary{}, err
		}
		if err := restored.Restore(ctx, snap); err != nil {
			return tailGovernanceQoSRecoverySummary{}, err
		}
		second, ok, err := restored.Claim(ctx, "worker-tail-governance-qos-restored-"+label)
		if err != nil || !ok {
			return tailGovernanceQoSRecoverySummary{}, errors.New("qos restored claim failed")
		}
		commit := scheduler.TerminalCommit{
			TaskID:      second.Record.Task.TaskID,
			AttemptID:   second.Attempt.AttemptID,
			Status:      scheduler.TaskStateSucceeded,
			Result:      map[string]any{"ok": true},
			CommittedAt: time.Now(),
		}
		if _, err := restored.Complete(ctx, commit); err != nil {
			return tailGovernanceQoSRecoverySummary{}, err
		}
		if _, err := restored.Complete(ctx, commit); err != nil {
			return tailGovernanceQoSRecoverySummary{}, err
		}
		stats, err := restored.Stats(ctx)
		if err != nil {
			return tailGovernanceQoSRecoverySummary{}, err
		}
		return tailGovernanceQoSRecoverySummary{
			qosMode:                  stats.QoSMode,
			priorityClaimTotal:       stats.PriorityClaimTotal,
			fairnessYieldTotal:       stats.FairnessYieldTotal,
			completeTotal:            stats.CompleteTotal,
			duplicateTerminalCommits: stats.DuplicateTerminalCommitTotal,
		}, nil
	}

	runSummary, err := exec("run")
	if err != nil {
		t.Fatalf("run qos/recovery path failed: %v", err)
	}
	streamSummary, err := exec("stream")
	if err != nil {
		t.Fatalf("stream qos/recovery path failed: %v", err)
	}
	if runSummary != streamSummary {
		t.Fatalf("qos/recovery summary mismatch run=%#v stream=%#v", runSummary, streamSummary)
	}
	if runSummary.qosMode != string(scheduler.QoSModePriority) {
		t.Fatalf("qos mode = %q, want %q", runSummary.qosMode, scheduler.QoSModePriority)
	}
	if runSummary.priorityClaimTotal < 1 {
		t.Fatalf("priority_claim_total = %d, want > 0", runSummary.priorityClaimTotal)
	}
	if runSummary.completeTotal != 2 || runSummary.duplicateTerminalCommits != 1 {
		t.Fatalf("replay idempotency mismatch: %#v", runSummary)
	}
}
