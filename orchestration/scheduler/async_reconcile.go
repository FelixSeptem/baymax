package scheduler

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (s *Scheduler) NextAsyncReconcileDelay() time.Duration {
	if s == nil {
		return 5 * time.Second
	}
	cfg := s.asyncAwait.Reconcile
	seq := s.reconcileSeq.Add(1)
	return reconcileJitteredInterval(cfg.Interval, cfg.JitterRatio, seq)
}

func (s *Scheduler) ReconcileAwaitingReports(ctx context.Context, client A2AReconcilePollClient) (ReconcileCycleStats, error) {
	if s == nil {
		return ReconcileCycleStats{}, fmt.Errorf("scheduler is nil")
	}
	if client == nil {
		return ReconcileCycleStats{}, fmt.Errorf("a2a reconcile poll client is required")
	}
	cfg := s.asyncAwait.Reconcile
	if !cfg.Enabled {
		return ReconcileCycleStats{}, nil
	}

	now := s.nowTime()
	awaiting, err := s.store.ListAwaitingReport(ctx, now, cfg.BatchSize)
	if err != nil {
		return ReconcileCycleStats{}, err
	}

	stats := ReconcileCycleStats{}
	for i := range awaiting {
		record := awaiting[i]
		remoteTaskID := strings.TrimSpace(record.RemoteTaskID)
		if remoteTaskID == "" {
			// Missing correlation key cannot be reconciled; timeout path remains the fallback.
			stats.ErrorTotal++
			continue
		}
		attempt, ok := record.currentAttempt()
		if !ok {
			stats.ErrorTotal++
			continue
		}
		stats.PollTotal++
		classification, terminalRecord, pollErr := ClassifyReconcilePoll(ctx, client, remoteTaskID)
		switch classification {
		case ReconcilePollClassificationTerminal:
			commit, mapErr := ReconcileTerminalCommitFromRecord(
				record.Task.TaskID,
				attempt.AttemptID,
				remoteTaskID,
				terminalRecord,
				now,
			)
			if mapErr != nil {
				stats.ErrorTotal++
				continue
			}
			commitResult, commitErr := s.CommitAsyncReportTerminal(ctx, commit)
			if commitErr != nil {
				stats.ErrorTotal++
				continue
			}
			if commitResult.Conflict {
				stats.ConflictTotalDelta++
			}
			if !commitResult.Duplicate && !commitResult.LateReport {
				stats.TerminalByPoll++
			}
		case ReconcilePollClassificationNotFound:
			// keep_until_timeout: no state mutation before timeout boundary.
		case ReconcilePollClassificationRetryableError, ReconcilePollClassificationNonRetryableErr:
			_ = pollErr
			stats.ErrorTotal++
		default:
			// pending: do nothing
		}
	}

	if err := s.store.RecordAsyncReconcileStats(ctx, stats.PollTotal, stats.ErrorTotal); err != nil {
		return stats, err
	}
	return stats, nil
}

func reconcileJitteredInterval(base time.Duration, ratio float64, seq int64) time.Duration {
	if base <= 0 {
		base = 5 * time.Second
	}
	if ratio <= 0 {
		return base
	}
	if ratio > 1 {
		ratio = 1
	}
	jitterRange := int64(float64(base) * ratio)
	if jitterRange <= 0 {
		return base
	}
	seed := stableRetryJitterSeed("async_reconcile_interval", fmt.Sprintf("%d", seq), int(seq))
	jitter := (seed % (2*jitterRange + 1)) - jitterRange
	withJitter := int64(base) + jitter
	if withJitter <= 0 {
		return time.Millisecond
	}
	return time.Duration(withJitter)
}
