package integration

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/a2a"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/orchestration/scheduler"
)

func TestAsyncReportingContractDeliveryMatrix(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		collector := &a2aTimelineCollector{}
		server := a2a.NewInMemoryServer(a2a.HandlerFunc(func(ctx context.Context, req a2a.TaskRequest) (map[string]any, error) {
			return map[string]any{"ok": true}, nil
		}), collector)
		client := a2a.NewClient(server, nil, nil, a2a.ClientPolicy{
			Timeout:            300 * time.Millisecond,
			RequestMaxAttempts: 1,
			AsyncReporting: a2a.AsyncReportingPolicy{
				Enabled: true,
				Retry: a2a.AsyncReportingRetryPolicy{
					MaxAttempts:    2,
					BackoffInitial: time.Millisecond,
					BackoffMax:     2 * time.Millisecond,
				},
				JitterRatio: 0,
			},
		}, collector)
		sink := a2a.NewChannelReportSink(4)
		ack, err := client.SubmitAsync(context.Background(), a2a.TaskRequest{
			TaskID:  "async-success-task",
			AgentID: "agent-success",
			PeerID:  "peer-success",
			Method:  "async.success",
		}, sink)
		if err != nil {
			t.Fatalf("SubmitAsync failed: %v", err)
		}
		select {
		case report := <-sink.Channel():
			if report.TaskID != ack.TaskID || report.Status != a2a.StatusSucceeded {
				t.Fatalf("unexpected report: %#v", report)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for async report")
		}
	})

	t.Run("retry_then_success", func(t *testing.T) {
		collector := &a2aTimelineCollector{}
		server := a2a.NewInMemoryServer(a2a.HandlerFunc(func(ctx context.Context, req a2a.TaskRequest) (map[string]any, error) {
			return map[string]any{"ok": true}, nil
		}), collector)
		var attempts atomic.Int32
		sink := a2a.NewCallbackReportSink(func(ctx context.Context, report a2a.AsyncReport) error {
			if attempts.Add(1) < 3 {
				return &a2a.AsyncReportDeliveryError{Cause: errors.New("sink temporary error"), Retryable: true}
			}
			return nil
		})
		client := a2a.NewClient(server, nil, nil, a2a.ClientPolicy{
			Timeout:            300 * time.Millisecond,
			RequestMaxAttempts: 1,
			AsyncReporting: a2a.AsyncReportingPolicy{
				Enabled: true,
				Retry: a2a.AsyncReportingRetryPolicy{
					MaxAttempts:    3,
					BackoffInitial: time.Millisecond,
					BackoffMax:     3 * time.Millisecond,
				},
				JitterRatio: 0,
			},
		}, collector)
		if _, err := client.SubmitAsync(context.Background(), a2a.TaskRequest{
			TaskID:  "async-retry-task",
			AgentID: "agent-retry",
			PeerID:  "peer-retry",
			Method:  "async.retry",
		}, sink); err != nil {
			t.Fatalf("SubmitAsync failed: %v", err)
		}
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) && attempts.Load() < 3 {
			time.Sleep(10 * time.Millisecond)
		}
		if got := attempts.Load(); got != 3 {
			t.Fatalf("sink attempts = %d, want 3", got)
		}
		retryEvents := 0
		deliverEvents := 0
		for _, ev := range collector.snapshot() {
			reason, _ := ev.Payload["reason"].(string)
			switch reason {
			case a2a.ReasonAsyncReportRetry:
				retryEvents++
			case a2a.ReasonAsyncReportDeliver:
				deliverEvents++
			}
		}
		if retryEvents != 2 || deliverEvents == 0 {
			t.Fatalf("async retry/deliver timeline mismatch retry=%d deliver=%d", retryEvents, deliverEvents)
		}
	})

	t.Run("final_drop_does_not_mutate_business_terminal", func(t *testing.T) {
		collector := &a2aTimelineCollector{}
		server := a2a.NewInMemoryServer(a2a.HandlerFunc(func(ctx context.Context, req a2a.TaskRequest) (map[string]any, error) {
			return map[string]any{"ok": true}, nil
		}), collector)
		client := a2a.NewClient(server, nil, nil, a2a.ClientPolicy{
			Timeout:            300 * time.Millisecond,
			RequestMaxAttempts: 1,
			AsyncReporting: a2a.AsyncReportingPolicy{
				Enabled: true,
				Retry: a2a.AsyncReportingRetryPolicy{
					MaxAttempts:    1,
					BackoffInitial: 0,
					BackoffMax:     0,
				},
				JitterRatio: 0,
			},
		}, collector)
		ack, err := client.SubmitAsync(context.Background(), a2a.TaskRequest{
			TaskID:  "async-drop-task",
			AgentID: "agent-drop",
			PeerID:  "peer-drop",
			Method:  "async.drop",
		}, a2a.NewCallbackReportSink(func(context.Context, a2a.AsyncReport) error {
			return &a2a.AsyncReportDeliveryError{Cause: errors.New("sink hard fail"), Retryable: false}
		}))
		if err != nil {
			t.Fatalf("SubmitAsync failed: %v", err)
		}
		terminal := waitForA2ATerminal(t, server, ack.TaskID)
		if terminal.Status != a2a.StatusSucceeded {
			t.Fatalf("business terminal status = %q, want succeeded", terminal.Status)
		}
		deadline := time.Now().Add(2 * time.Second)
		foundDrop := false
		for time.Now().Before(deadline) {
			for _, ev := range collector.snapshot() {
				reason, _ := ev.Payload["reason"].(string)
				if reason == a2a.ReasonAsyncReportDrop {
					foundDrop = true
					break
				}
			}
			if foundDrop {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		if !foundDrop {
			t.Fatalf("missing timeline reason %q", a2a.ReasonAsyncReportDrop)
		}
	})
}

func TestAsyncReportingContractDedupAndReplayIdempotency(t *testing.T) {
	s, err := scheduler.New(scheduler.NewMemoryStore(), scheduler.WithLeaseTimeout(500*time.Millisecond))
	if err != nil {
		t.Fatalf("new scheduler: %v", err)
	}
	ctx := context.Background()
	if _, err := s.Enqueue(ctx, scheduler.Task{TaskID: "async-dedup-task", RunID: "async-dedup-run"}); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	claimed, ok, err := s.Claim(ctx, "worker-dedup")
	if err != nil || !ok {
		t.Fatalf("claim failed: ok=%v err=%v", ok, err)
	}
	if _, err := s.MarkAwaitingReport(ctx, claimed.Record.Task.TaskID, claimed.Attempt.AttemptID); err != nil {
		t.Fatalf("mark awaiting_report failed: %v", err)
	}
	report := a2a.AsyncReport{
		ReportKey:  "async-dedup-key",
		OutcomeKey: "succeeded|ok",
		TaskID:     claimed.Record.Task.TaskID,
		AttemptID:  claimed.Attempt.AttemptID,
		Status:     a2a.StatusSucceeded,
		Result:     map[string]any{"ok": true},
	}
	first, err := scheduler.ExecutionFromAsyncReport(claimed, report)
	if err != nil {
		t.Fatalf("ExecutionFromAsyncReport failed: %v", err)
	}
	result1, err := s.CommitAsyncReportTerminal(ctx, first.Commit)
	if err != nil {
		t.Fatalf("first complete failed: %v", err)
	}
	if result1.Duplicate {
		t.Fatal("first complete should not be duplicate")
	}

	second, err := scheduler.ExecutionFromAsyncReport(claimed, report)
	if err != nil {
		t.Fatalf("ExecutionFromAsyncReport replay failed: %v", err)
	}
	result2, err := s.CommitAsyncReportTerminal(ctx, second.Commit)
	if err != nil {
		t.Fatalf("second complete failed: %v", err)
	}
	if !result2.Duplicate {
		t.Fatal("replayed complete should be duplicate")
	}
	stats, err := s.Stats(ctx)
	if err != nil {
		t.Fatalf("stats failed: %v", err)
	}
	if stats.CompleteTotal != 1 || stats.DuplicateTerminalCommitTotal != 1 {
		t.Fatalf("dedup/replay stats mismatch: %#v", stats)
	}
}

func TestAsyncReportingContractRunStreamEquivalence(t *testing.T) {
	collector := &a2aTimelineCollector{}
	server := a2a.NewInMemoryServer(a2a.HandlerFunc(func(ctx context.Context, req a2a.TaskRequest) (map[string]any, error) {
		return map[string]any{"ok": true}, nil
	}), collector)
	client := a2a.NewClient(server, nil, nil, a2a.ClientPolicy{
		Timeout:            300 * time.Millisecond,
		RequestMaxAttempts: 1,
		AsyncReporting: a2a.AsyncReportingPolicy{
			Enabled: true,
			Retry: a2a.AsyncReportingRetryPolicy{
				MaxAttempts:    2,
				BackoffInitial: time.Millisecond,
				BackoffMax:     2 * time.Millisecond,
			},
			JitterRatio: 0,
		},
	}, collector)

	runSink := a2a.NewChannelReportSink(2)
	streamSink := a2a.NewChannelReportSink(2)
	runAck, err := client.SubmitAsync(context.Background(), a2a.TaskRequest{
		TaskID:  "async-run-task",
		AgentID: "agent-run",
		PeerID:  "peer-eq",
		Method:  "async.run",
	}, runSink)
	if err != nil {
		t.Fatalf("run SubmitAsync failed: %v", err)
	}
	streamAck, err := client.SubmitAsync(context.Background(), a2a.TaskRequest{
		TaskID:  "async-stream-task",
		AgentID: "agent-stream",
		PeerID:  "peer-eq",
		Method:  "async.stream",
	}, streamSink)
	if err != nil {
		t.Fatalf("stream SubmitAsync failed: %v", err)
	}
	runReport := <-runSink.Channel()
	streamReport := <-streamSink.Channel()
	if runReport.Status != streamReport.Status {
		t.Fatalf("run/stream report status mismatch run=%q stream=%q", runReport.Status, streamReport.Status)
	}

	var runReasons map[string]int
	var streamReasons map[string]int
	deadline := time.Now().Add(2 * time.Second)
	for {
		runReasons = asyncReasonDistributionByRun(collector.snapshot(), runAck.TaskID)
		streamReasons = asyncReasonDistributionByRun(collector.snapshot(), streamAck.TaskID)
		if len(runReasons) > 0 && len(streamReasons) > 0 && equalReasonDistribution(runReasons, streamReasons) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("async reason distributions did not converge run=%#v stream=%#v", runReasons, streamReasons)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestAsyncReportingContractRecoveryReplayNoInflation(t *testing.T) {
	s1, err := scheduler.New(scheduler.NewMemoryStore(), scheduler.WithLeaseTimeout(500*time.Millisecond))
	if err != nil {
		t.Fatalf("new scheduler #1: %v", err)
	}
	ctx := context.Background()
	if _, err := s1.Enqueue(ctx, scheduler.Task{TaskID: "async-recovery-task", RunID: "async-recovery-run"}); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	claimed, ok, err := s1.Claim(ctx, "worker-recovery")
	if err != nil || !ok {
		t.Fatalf("claim failed: ok=%v err=%v", ok, err)
	}
	if _, err := s1.MarkAwaitingReport(ctx, claimed.Record.Task.TaskID, claimed.Attempt.AttemptID); err != nil {
		t.Fatalf("mark awaiting_report failed: %v", err)
	}
	report := a2a.AsyncReport{
		ReportKey:  "async-recovery-key",
		OutcomeKey: "succeeded|ok",
		TaskID:     claimed.Record.Task.TaskID,
		AttemptID:  claimed.Attempt.AttemptID,
		Status:     a2a.StatusSucceeded,
		Result:     map[string]any{"ok": true},
	}
	execution, err := scheduler.ExecutionFromAsyncReport(claimed, report)
	if err != nil {
		t.Fatalf("ExecutionFromAsyncReport failed: %v", err)
	}
	if _, err := s1.CommitAsyncReportTerminal(ctx, execution.Commit); err != nil {
		t.Fatalf("complete failed: %v", err)
	}
	snapshot, err := s1.Snapshot(ctx)
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}

	s2, err := scheduler.New(scheduler.NewMemoryStore(), scheduler.WithLeaseTimeout(500*time.Millisecond))
	if err != nil {
		t.Fatalf("new scheduler #2: %v", err)
	}
	if err := s2.Restore(ctx, snapshot); err != nil {
		t.Fatalf("restore failed: %v", err)
	}
	before, err := s2.Stats(ctx)
	if err != nil {
		t.Fatalf("stats before replay failed: %v", err)
	}
	replayExec, err := scheduler.ExecutionFromAsyncReport(claimed, report)
	if err != nil {
		t.Fatalf("ExecutionFromAsyncReport replay failed: %v", err)
	}
	if _, err := s2.CommitAsyncReportTerminal(ctx, replayExec.Commit); err != nil {
		t.Fatalf("replay complete failed: %v", err)
	}
	after, err := s2.Stats(ctx)
	if err != nil {
		t.Fatalf("stats after replay failed: %v", err)
	}
	if before.CompleteTotal != after.CompleteTotal {
		t.Fatalf("recovery replay should not inflate complete_total before=%d after=%d", before.CompleteTotal, after.CompleteTotal)
	}
}

type asyncAwaitLifecycleSummary struct {
	awaitingObserved  bool
	terminalState     scheduler.TaskState
	lateReport        bool
	duplicate         bool
	asyncAwaitTotal   int
	asyncTimeoutTotal int
}

func TestAsyncReportingContractAwaitingLifecycleRunStreamEquivalence(t *testing.T) {
	exec := func(taskID string) (asyncAwaitLifecycleSummary, error) {
		s, err := scheduler.New(
			scheduler.NewMemoryStore(),
			scheduler.WithLeaseTimeout(500*time.Millisecond),
			scheduler.WithAsyncAwait(scheduler.AsyncAwaitConfig{
				ReportTimeout:    20 * time.Millisecond,
				LateReportPolicy: scheduler.AsyncLateReportPolicyDropAndRecord,
				TimeoutTerminal:  scheduler.TaskStateFailed,
			}),
		)
		if err != nil {
			return asyncAwaitLifecycleSummary{}, err
		}
		return executeAsyncAwaitTimeoutFlow(s, taskID)
	}

	runSummary, err := exec("async-await-run")
	if err != nil {
		t.Fatalf("run flow failed: %v", err)
	}
	streamSummary, err := exec("async-await-stream")
	if err != nil {
		t.Fatalf("stream flow failed: %v", err)
	}
	if runSummary != streamSummary {
		t.Fatalf("run/stream async-await summary mismatch: run=%#v stream=%#v", runSummary, streamSummary)
	}
}

func TestAsyncReportingContractAwaitingLifecycleMemoryFileParity(t *testing.T) {
	memoryScheduler, err := scheduler.New(
		scheduler.NewMemoryStore(),
		scheduler.WithLeaseTimeout(500*time.Millisecond),
		scheduler.WithAsyncAwait(scheduler.AsyncAwaitConfig{
			ReportTimeout:    20 * time.Millisecond,
			LateReportPolicy: scheduler.AsyncLateReportPolicyDropAndRecord,
			TimeoutTerminal:  scheduler.TaskStateFailed,
		}),
	)
	if err != nil {
		t.Fatalf("new memory scheduler failed: %v", err)
	}
	memorySummary, err := executeAsyncAwaitTimeoutFlow(memoryScheduler, "async-await-memory")
	if err != nil {
		t.Fatalf("memory flow failed: %v", err)
	}

	fileStore, err := scheduler.NewFileStore(filepath.Join(t.TempDir(), "scheduler-a31-state.json"))
	if err != nil {
		t.Fatalf("new file store failed: %v", err)
	}
	fileScheduler, err := scheduler.New(
		fileStore,
		scheduler.WithLeaseTimeout(500*time.Millisecond),
		scheduler.WithAsyncAwait(scheduler.AsyncAwaitConfig{
			ReportTimeout:    20 * time.Millisecond,
			LateReportPolicy: scheduler.AsyncLateReportPolicyDropAndRecord,
			TimeoutTerminal:  scheduler.TaskStateFailed,
		}),
	)
	if err != nil {
		t.Fatalf("new file scheduler failed: %v", err)
	}
	fileSummary, err := executeAsyncAwaitTimeoutFlow(fileScheduler, "async-await-file")
	if err != nil {
		t.Fatalf("file flow failed: %v", err)
	}

	if memorySummary != fileSummary {
		t.Fatalf("memory/file async-await summary mismatch: memory=%#v file=%#v", memorySummary, fileSummary)
	}
}

func executeAsyncAwaitTimeoutFlow(s *scheduler.Scheduler, taskID string) (asyncAwaitLifecycleSummary, error) {
	ctx := context.Background()
	if _, err := s.Enqueue(ctx, scheduler.Task{
		TaskID: taskID,
		RunID:  "run-" + taskID,
	}); err != nil {
		return asyncAwaitLifecycleSummary{}, err
	}
	claimed, ok, err := s.Claim(ctx, "worker-"+taskID)
	if err != nil || !ok {
		if err != nil {
			return asyncAwaitLifecycleSummary{}, err
		}
		return asyncAwaitLifecycleSummary{}, errors.New("claim failed")
	}
	awaiting, err := s.MarkAwaitingReport(ctx, claimed.Record.Task.TaskID, claimed.Attempt.AttemptID)
	if err != nil {
		return asyncAwaitLifecycleSummary{}, err
	}
	time.Sleep(40 * time.Millisecond)
	if _, err := s.ExpireLeases(ctx); err != nil {
		return asyncAwaitLifecycleSummary{}, err
	}
	lateCommit, err := s.CommitAsyncReportTerminal(ctx, scheduler.TerminalCommit{
		TaskID:      claimed.Record.Task.TaskID,
		AttemptID:   claimed.Attempt.AttemptID,
		Status:      scheduler.TaskStateSucceeded,
		Result:      map[string]any{"ok": true},
		CommittedAt: time.Now().UTC(),
	})
	if err != nil {
		return asyncAwaitLifecycleSummary{}, err
	}
	record, found, err := s.Get(ctx, claimed.Record.Task.TaskID)
	if err != nil || !found {
		if err != nil {
			return asyncAwaitLifecycleSummary{}, err
		}
		return asyncAwaitLifecycleSummary{}, errors.New("get failed")
	}
	stats, err := s.Stats(ctx)
	if err != nil {
		return asyncAwaitLifecycleSummary{}, err
	}
	return asyncAwaitLifecycleSummary{
		awaitingObserved:  awaiting.State == scheduler.TaskStateAwaitingReport,
		terminalState:     record.State,
		lateReport:        lateCommit.LateReport,
		duplicate:         lateCommit.Duplicate,
		asyncAwaitTotal:   stats.AsyncAwaitTotal,
		asyncTimeoutTotal: stats.AsyncTimeoutTotal,
	}, nil
}

func waitForA2ATerminal(t *testing.T, server a2a.Server, taskID string) a2a.TaskRecord {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		rec, err := server.Status(context.Background(), taskID)
		if err != nil {
			t.Fatalf("status(%s) failed: %v", taskID, err)
		}
		if rec.Status == a2a.StatusSucceeded || rec.Status == a2a.StatusFailed || rec.Status == a2a.StatusCanceled {
			return rec
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("task %s did not reach terminal status", taskID)
	return a2a.TaskRecord{}
}

func asyncReasonDistributionByRun(events []types.Event, runID string) map[string]int {
	out := map[string]int{}
	for _, ev := range events {
		if strings.TrimSpace(ev.RunID) != strings.TrimSpace(runID) {
			continue
		}
		reason, _ := ev.Payload["reason"].(string)
		if strings.HasPrefix(reason, "a2a.async_") {
			out[reason]++
		}
	}
	return out
}

func equalReasonDistribution(left map[string]int, right map[string]int) bool {
	if len(left) != len(right) {
		return false
	}
	for reason, count := range left {
		if right[reason] != count {
			return false
		}
	}
	return true
}

func TestAsyncReportingContractLegacyDirectAsyncEntrypointNotSupportedPublicly(t *testing.T) {
	root := integrationRepoRoot(t)
	asyncSource := mustReadIntegrationFile(t, filepath.Join(root, "orchestration", "invoke", "async.go"))
	if strings.Contains(asyncSource, "func InvokeAsync(") {
		t.Fatal("legacy direct public invoke.InvokeAsync entrypoint must not be reintroduced")
	}
	legacyUsages := findLegacyInvokeQualifiedUsages(t, root)
	if len(legacyUsages) > 0 {
		t.Fatalf("legacy direct async invoke usage detected outside canonical mailbox path: %#v", legacyUsages)
	}
}
