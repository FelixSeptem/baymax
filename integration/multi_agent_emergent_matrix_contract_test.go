package integration

import (
	"context"
	"reflect"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/integration/fakes"
	"github.com/FelixSeptem/baymax/orchestration/composer"
	"github.com/FelixSeptem/baymax/orchestration/mailbox"
	"github.com/FelixSeptem/baymax/orchestration/scheduler"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

type emergentDriftCategory string

const (
	emergentDriftNone                     emergentDriftCategory = "none"
	emergentDriftCascadeAmplification     emergentDriftCategory = "cascade_amplification"
	emergentDriftOrderingInstability      emergentDriftCategory = "ordering_instability"
	emergentDriftTerminalStateInstability emergentDriftCategory = "terminal_state_instability"
)

type emergentMatrixSnapshot struct {
	Component      string
	Mode           string
	TerminalStates map[string]string
	OrderedTrace   []string
	EventTotal     int
	RetryTotal     int
	DuplicateTotal int
}

func TestMultiAgentEmergentDriftTaxonomyGuard(t *testing.T) {
	wantTaxonomy := []emergentDriftCategory{
		emergentDriftNone,
		emergentDriftCascadeAmplification,
		emergentDriftOrderingInstability,
		emergentDriftTerminalStateInstability,
	}
	if got := emergentDriftTaxonomy(); !reflect.DeepEqual(got, wantTaxonomy) {
		t.Fatalf("emergent drift taxonomy mismatch: got=%#v want=%#v", got, wantTaxonomy)
	}

	baseline := emergentMatrixSnapshot{
		Component:      "scheduler",
		Mode:           "parallel",
		TerminalStates: map[string]string{"task-a": "succeeded"},
		OrderedTrace:   []string{"claim:task-a", "complete:task-a"},
		EventTotal:     4,
		RetryTotal:     1,
		DuplicateTotal: 0,
	}
	terminalDrift := baseline
	terminalDrift.TerminalStates = map[string]string{"task-a": "failed"}
	if got := classifyEmergentDrift(baseline, terminalDrift); got != emergentDriftTerminalStateInstability {
		t.Fatalf("terminal drift classification mismatch: got=%q", got)
	}

	cascadeDrift := baseline
	cascadeDrift.EventTotal = baseline.EventTotal + 1
	if got := classifyEmergentDrift(baseline, cascadeDrift); got != emergentDriftCascadeAmplification {
		t.Fatalf("cascade drift classification mismatch: got=%q", got)
	}

	orderingDrift := baseline
	orderingDrift.OrderedTrace = []string{"complete:task-a", "claim:task-a"}
	if got := classifyEmergentDrift(baseline, orderingDrift); got != emergentDriftOrderingInstability {
		t.Fatalf("ordering drift classification mismatch: got=%q", got)
	}

	stable := baseline
	if got := classifyEmergentDrift(baseline, stable); got != emergentDriftNone {
		t.Fatalf("stable classification mismatch: got=%q", got)
	}
}

func TestMultiAgentEmergentMatrixDeterministicBlocking(t *testing.T) {
	matrix := []struct {
		name string
		run  func(*testing.T) emergentMatrixSnapshot
	}{
		{name: "scheduler_parallel", run: runSchedulerParallelSnapshot},
		{name: "mailbox_interleaving", run: runMailboxInterleavingSnapshot},
		{name: "composer_retry", run: runComposerRetrySnapshot},
		{name: "composer_replay", run: runComposerReplaySnapshot},
	}

	for _, tc := range matrix {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			baseline := tc.run(t)
			for i := 0; i < 2; i++ {
				candidate := tc.run(t)
				if drift := classifyEmergentDrift(baseline, candidate); drift != emergentDriftNone {
					t.Fatalf(
						"emergent drift detected in %s: category=%q baseline=%#v candidate=%#v",
						tc.name,
						drift,
						baseline,
						candidate,
					)
				}
			}
		})
	}
}

func emergentDriftTaxonomy() []emergentDriftCategory {
	return []emergentDriftCategory{
		emergentDriftNone,
		emergentDriftCascadeAmplification,
		emergentDriftOrderingInstability,
		emergentDriftTerminalStateInstability,
	}
}

func classifyEmergentDrift(
	baseline emergentMatrixSnapshot,
	candidate emergentMatrixSnapshot,
) emergentDriftCategory {
	if !reflect.DeepEqual(baseline.TerminalStates, candidate.TerminalStates) {
		return emergentDriftTerminalStateInstability
	}
	if baseline.EventTotal != candidate.EventTotal ||
		baseline.RetryTotal != candidate.RetryTotal ||
		baseline.DuplicateTotal != candidate.DuplicateTotal {
		return emergentDriftCascadeAmplification
	}
	if !reflect.DeepEqual(baseline.OrderedTrace, candidate.OrderedTrace) {
		return emergentDriftOrderingInstability
	}
	return emergentDriftNone
}

func runSchedulerParallelSnapshot(t *testing.T) emergentMatrixSnapshot {
	t.Helper()

	ctx := context.Background()
	s, err := scheduler.New(
		scheduler.NewMemoryStore(),
		scheduler.WithLeaseTimeout(500*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("new scheduler: %v", err)
	}

	taskIDs := []string{
		"task-a64-emergent-scheduler-1",
		"task-a64-emergent-scheduler-2",
		"task-a64-emergent-scheduler-3",
		"task-a64-emergent-scheduler-4",
		"task-a64-emergent-scheduler-5",
		"task-a64-emergent-scheduler-6",
	}
	var enqueueWG sync.WaitGroup
	enqueueErrCh := make(chan error, len(taskIDs))
	for _, taskID := range taskIDs {
		taskID := taskID
		enqueueWG.Add(1)
		go func() {
			defer enqueueWG.Done()
			if _, enqueueErr := s.Enqueue(ctx, scheduler.Task{
				TaskID: taskID,
				RunID:  "run-a64-emergent-scheduler-parallel",
			}); enqueueErr != nil {
				enqueueErrCh <- enqueueErr
			}
		}()
	}
	enqueueWG.Wait()
	close(enqueueErrCh)
	for enqueueErr := range enqueueErrCh {
		if enqueueErr != nil {
			t.Fatalf("parallel enqueue failed: %v", enqueueErr)
		}
	}

	var completed atomic.Int32
	claimErrCh := make(chan error, 8)
	var claimWG sync.WaitGroup
	worker := func(workerID string) {
		defer claimWG.Done()
		for completed.Load() < int32(len(taskIDs)) {
			claimed, ok, claimErr := s.Claim(ctx, workerID)
			if claimErr != nil {
				claimErrCh <- claimErr
				return
			}
			if !ok {
				time.Sleep(2 * time.Millisecond)
				continue
			}
			if _, commitErr := s.Complete(ctx, scheduler.TerminalCommit{
				TaskID:      claimed.Record.Task.TaskID,
				AttemptID:   claimed.Attempt.AttemptID,
				Status:      scheduler.TaskStateSucceeded,
				CommittedAt: time.Now().UTC(),
				Result:      map[string]any{"ok": true},
			}); commitErr != nil {
				claimErrCh <- commitErr
				return
			}
			completed.Add(1)
		}
	}
	for _, workerID := range []string{"worker-a", "worker-b", "worker-c"} {
		claimWG.Add(1)
		go worker(workerID)
	}
	claimWG.Wait()
	close(claimErrCh)
	for claimErr := range claimErrCh {
		if claimErr != nil {
			t.Fatalf("parallel claim/complete failed: %v", claimErr)
		}
	}
	if got := int(completed.Load()); got != len(taskIDs) {
		t.Fatalf("parallel completion total=%d, want %d", got, len(taskIDs))
	}

	terminal := make(map[string]string, len(taskIDs))
	for _, taskID := range taskIDs {
		record, ok, getErr := s.Get(ctx, taskID)
		if getErr != nil || !ok {
			t.Fatalf("get task %q failed: found=%v err=%v", taskID, ok, getErr)
		}
		terminal[taskID] = string(record.State)
		if record.State != scheduler.TaskStateSucceeded {
			t.Fatalf("task %q terminal state=%q, want succeeded", taskID, record.State)
		}
	}
	stats, err := s.Stats(ctx)
	if err != nil {
		t.Fatalf("scheduler stats failed: %v", err)
	}
	if stats.ClaimTotal != len(taskIDs) || stats.CompleteTotal != len(taskIDs) {
		t.Fatalf("scheduler stats mismatch under parallel matrix: %#v", stats)
	}

	ordered := append([]string(nil), taskIDs...)
	sort.Strings(ordered)
	return emergentMatrixSnapshot{
		Component:      "scheduler",
		Mode:           "parallel",
		TerminalStates: terminal,
		OrderedTrace:   ordered,
		EventTotal: stats.ClaimTotal +
			stats.CompleteTotal +
			stats.FailTotal +
			stats.ReclaimTotal +
			stats.RetryBackoffTotal +
			stats.DeadLetterTotal,
		RetryTotal:     stats.RetryBackoffTotal,
		DuplicateTotal: stats.DuplicateTerminalCommitTotal,
	}
}

func runMailboxInterleavingSnapshot(t *testing.T) emergentMatrixSnapshot {
	t.Helper()

	ctx := context.Background()
	base := time.Date(2026, time.January, 2, 12, 0, 0, 0, time.UTC)
	var mu sync.Mutex
	clock := base
	now := func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return clock
	}
	advance := func(delta time.Duration) {
		mu.Lock()
		clock = clock.Add(delta)
		mu.Unlock()
	}

	events := make([]mailbox.LifecycleEvent, 0, 16)
	mb, err := mailbox.New(
		mailbox.NewMemoryStore(mailbox.Policy{
			MaxAttempts:    3,
			BackoffInitial: 30 * time.Millisecond,
			BackoffMax:     30 * time.Millisecond,
			JitterRatio:    0,
			DLQEnabled:     false,
		}),
		mailbox.WithClock(now),
		mailbox.WithLifecycleObserver(func(_ context.Context, event mailbox.LifecycleEvent) {
			events = append(events, event)
		}),
	)
	if err != nil {
		t.Fatalf("new mailbox: %v", err)
	}

	const (
		msgImmediate = "msg-a64-emergent-interleaving-immediate"
		msgDelayed   = "msg-a64-emergent-interleaving-delayed"
		runID        = "run-a64-emergent-mailbox-interleaving"
	)
	if _, err := mb.Publish(ctx, mailbox.Envelope{
		MessageID:      msgImmediate,
		IdempotencyKey: msgImmediate,
		Kind:           mailbox.KindCommand,
		RunID:          runID,
		TaskID:         "task-a64-emergent-mailbox-interleaving-immediate",
		WorkflowID:     "wf-a64-emergent-mailbox-interleaving",
		TeamID:         "team-a64-emergent",
	}); err != nil {
		t.Fatalf("publish immediate message failed: %v", err)
	}
	if _, err := mb.Publish(ctx, mailbox.Envelope{
		MessageID:      msgDelayed,
		IdempotencyKey: msgDelayed,
		Kind:           mailbox.KindCommand,
		RunID:          runID,
		TaskID:         "task-a64-emergent-mailbox-interleaving-delayed",
		WorkflowID:     "wf-a64-emergent-mailbox-interleaving",
		TeamID:         "team-a64-emergent",
		NotBefore:      base.Add(120 * time.Millisecond),
	}); err != nil {
		t.Fatalf("publish delayed message failed: %v", err)
	}

	immediateClaim, ok, err := mb.Consume(ctx, "worker-a64-interleaving")
	if err != nil || !ok {
		t.Fatalf("consume immediate message failed: ok=%v err=%v", ok, err)
	}
	if immediateClaim.Envelope.MessageID != msgImmediate {
		t.Fatalf("immediate consume order drift: got=%q want=%q", immediateClaim.Envelope.MessageID, msgImmediate)
	}
	if _, err := mb.Requeue(ctx, msgImmediate, "worker-a64-interleaving", "handler_error"); err != nil {
		t.Fatalf("requeue immediate message failed: %v", err)
	}

	if _, blocked, err := mb.Consume(ctx, "worker-a64-interleaving"); err != nil {
		t.Fatalf("consume during backoff failed: %v", err)
	} else if blocked {
		t.Fatal("consume during backoff should be blocked")
	}

	advance(40 * time.Millisecond)
	retryClaim, ok, err := mb.Consume(ctx, "worker-a64-interleaving")
	if err != nil || !ok {
		t.Fatalf("consume retried immediate message failed: ok=%v err=%v", ok, err)
	}
	if retryClaim.Envelope.MessageID != msgImmediate {
		t.Fatalf("retry consume order drift: got=%q want=%q", retryClaim.Envelope.MessageID, msgImmediate)
	}
	if _, err := mb.Ack(ctx, msgImmediate, "worker-a64-interleaving"); err != nil {
		t.Fatalf("ack immediate message failed: %v", err)
	}

	if _, blocked, err := mb.Consume(ctx, "worker-a64-interleaving"); err != nil {
		t.Fatalf("consume delayed boundary check failed: %v", err)
	} else if blocked {
		t.Fatal("delayed message should remain blocked before not_before")
	}

	advance(120 * time.Millisecond)
	delayedClaim, ok, err := mb.Consume(ctx, "worker-a64-interleaving")
	if err != nil || !ok {
		t.Fatalf("consume delayed message failed: ok=%v err=%v", ok, err)
	}
	if delayedClaim.Envelope.MessageID != msgDelayed {
		t.Fatalf("delayed consume order drift: got=%q want=%q", delayedClaim.Envelope.MessageID, msgDelayed)
	}
	if _, err := mb.Ack(ctx, msgDelayed, "worker-a64-interleaving"); err != nil {
		t.Fatalf("ack delayed message failed: %v", err)
	}

	page, err := mb.Query(ctx, mailbox.QueryRequest{RunID: runID})
	if err != nil {
		t.Fatalf("mailbox query failed: %v", err)
	}
	terminal := make(map[string]string, len(page.Items))
	retryTotal := 0
	for _, item := range page.Items {
		terminal[item.Envelope.MessageID] = string(item.State)
		if item.DeliveryAttempt > 1 {
			retryTotal += item.DeliveryAttempt - 1
		}
	}
	if terminal[msgImmediate] != string(mailbox.StateAcked) || terminal[msgDelayed] != string(mailbox.StateAcked) {
		t.Fatalf("mailbox interleaving terminal states drift: %#v", terminal)
	}
	stats, err := mb.Stats(ctx)
	if err != nil {
		t.Fatalf("mailbox stats failed: %v", err)
	}

	trace := make([]string, 0, len(events))
	for _, event := range events {
		trace = append(trace, string(event.Transition)+":"+event.Record.Envelope.MessageID+":"+event.ReasonCode)
	}
	return emergentMatrixSnapshot{
		Component:      "mailbox",
		Mode:           "interleaving",
		TerminalStates: terminal,
		OrderedTrace:   trace,
		EventTotal: stats.ConsumedTotal +
			stats.AckTotal +
			stats.NackTotal +
			stats.RequeueTotal +
			stats.ExpiredTotal +
			stats.DeadLetterTotal,
		RetryTotal:     retryTotal,
		DuplicateTotal: stats.DuplicatePublishTotal,
	}
}

func runComposerRetrySnapshot(t *testing.T) emergentMatrixSnapshot {
	t.Helper()

	ctx := context.Background()
	comp := newEmergentComposer(t)
	taskID := "task-a64-emergent-composer-retry"
	if _, err := comp.SpawnChild(ctx, composer.ChildDispatchRequest{
		Task: scheduler.Task{
			TaskID:     taskID,
			RunID:      "run-a64-emergent-composer-retry",
			WorkflowID: "wf-a64-emergent-composer-retry",
			TeamID:     "team-a64-emergent",
		},
	}); err != nil {
		t.Fatalf("spawn child for retry matrix failed: %v", err)
	}

	claim1, ok, err := comp.Scheduler().Claim(ctx, "worker-a64-retry-a")
	if err != nil || !ok {
		t.Fatalf("claim #1 failed: ok=%v err=%v", ok, err)
	}
	if _, err := comp.Scheduler().Requeue(ctx, taskID, "retryable"); err != nil {
		t.Fatalf("requeue task failed: %v", err)
	}
	claim2, ok, err := comp.Scheduler().Claim(ctx, "worker-a64-retry-b")
	if err != nil || !ok {
		t.Fatalf("claim #2 failed: ok=%v err=%v", ok, err)
	}
	if _, err := comp.CommitChildTerminal(ctx, scheduler.TerminalCommit{
		TaskID:      taskID,
		AttemptID:   claim2.Attempt.AttemptID,
		Status:      scheduler.TaskStateSucceeded,
		CommittedAt: time.Now().UTC(),
		Result:      map[string]any{"ok": true},
	}); err != nil {
		t.Fatalf("complete retry task failed: %v", err)
	}

	record, ok, err := comp.Scheduler().Get(ctx, taskID)
	if err != nil || !ok {
		t.Fatalf("get retry task failed: found=%v err=%v", ok, err)
	}
	if record.State != scheduler.TaskStateSucceeded || len(record.Attempts) != 2 {
		t.Fatalf("retry task record drift: %#v", record)
	}
	stats, err := comp.SchedulerStats(ctx)
	if err != nil {
		t.Fatalf("scheduler stats failed: %v", err)
	}
	return emergentMatrixSnapshot{
		Component:      "composer",
		Mode:           "retry",
		TerminalStates: map[string]string{taskID: string(record.State)},
		OrderedTrace:   []string{claim1.Attempt.AttemptID, claim2.Attempt.AttemptID},
		EventTotal: stats.ClaimTotal +
			stats.CompleteTotal +
			stats.FailTotal +
			stats.ReclaimTotal +
			stats.RetryBackoffTotal +
			stats.DeadLetterTotal,
		RetryTotal:     len(record.Attempts) - 1,
		DuplicateTotal: stats.DuplicateTerminalCommitTotal,
	}
}

func runComposerReplaySnapshot(t *testing.T) emergentMatrixSnapshot {
	t.Helper()

	ctx := context.Background()
	comp := newEmergentComposer(t)
	out, err := comp.DispatchChild(ctx, composer.ChildDispatchRequest{
		Task: scheduler.Task{
			TaskID:     "task-a64-emergent-composer-replay",
			RunID:      "run-a64-emergent-composer-replay",
			WorkflowID: "wf-a64-emergent-composer-replay",
			TeamID:     "team-a64-emergent",
		},
		Target:               composer.ChildTargetLocal,
		ParentDepth:          0,
		ParentActiveChildren: 0,
		ChildTimeout:         500 * time.Millisecond,
		LocalRunner: composer.LocalChildRunnerFunc(func(context.Context, scheduler.Task) (map[string]any, error) {
			return map[string]any{"ok": true}, nil
		}),
	})
	if err != nil {
		t.Fatalf("dispatch child for replay matrix failed: %v", err)
	}
	if _, err := comp.CommitChildTerminal(ctx, out.Commit); err != nil {
		t.Fatalf("duplicate commit #1 failed: %v", err)
	}
	if _, err := comp.CommitChildTerminal(ctx, out.Commit); err != nil {
		t.Fatalf("duplicate commit #2 failed: %v", err)
	}

	record, ok, err := comp.Scheduler().Get(ctx, out.Record.Task.TaskID)
	if err != nil || !ok {
		t.Fatalf("get replay task failed: found=%v err=%v", ok, err)
	}
	if record.State != scheduler.TaskStateSucceeded {
		t.Fatalf("replay task terminal state=%q, want succeeded", record.State)
	}
	stats, err := comp.SchedulerStats(ctx)
	if err != nil {
		t.Fatalf("scheduler stats failed: %v", err)
	}
	if stats.CompleteTotal != 1 {
		t.Fatalf("replay matrix complete_total drift: %#v", stats)
	}
	if stats.DuplicateTerminalCommitTotal != 2 {
		t.Fatalf("replay matrix duplicate commit total drift: %#v", stats)
	}
	return emergentMatrixSnapshot{
		Component:      "composer",
		Mode:           "replay",
		TerminalStates: map[string]string{out.Record.Task.TaskID: string(record.State)},
		OrderedTrace: []string{
			out.Commit.AttemptID,
			"duplicate_commit_1",
			"duplicate_commit_2",
		},
		EventTotal: stats.ClaimTotal +
			stats.CompleteTotal +
			stats.FailTotal +
			stats.ReclaimTotal +
			stats.RetryBackoffTotal +
			stats.DeadLetterTotal,
		RetryTotal:     len(record.Attempts) - 1,
		DuplicateTotal: stats.DuplicateTerminalCommitTotal,
	}
}

func newEmergentComposer(t *testing.T) *composer.Composer {
	t.Helper()

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		EnvPrefix: "BAYMAX_A64_EMERGENT",
	})
	if err != nil {
		t.Fatalf("new runtime manager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	model := fakes.NewModel([]fakes.ModelStep{
		{Response: types.ModelResponse{FinalAnswer: "ok"}},
	})
	model.SetStream([]types.ModelEvent{
		{Type: types.ModelEventTypeFinalAnswer, TextDelta: "ok"},
	}, nil)
	comp, err := composer.NewBuilder(model).
		WithRuntimeManager(mgr).
		WithSchedulerStore(scheduler.NewMemoryStore()).
		Build()
	if err != nil {
		t.Fatalf("new composer failed: %v", err)
	}
	return comp
}
