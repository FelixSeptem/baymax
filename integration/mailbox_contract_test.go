package integration

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/a2a"
	"github.com/FelixSeptem/baymax/orchestration/invoke"
	"github.com/FelixSeptem/baymax/orchestration/mailbox"
)

func TestMailboxContractSyncAsyncDelayedConvergenceRunStreamSemanticEquivalence(t *testing.T) {
	runSummary := executeMailboxConvergenceFlow(t, "run")
	streamSummary := executeMailboxConvergenceFlow(t, "stream")

	if runSummary != streamSummary {
		t.Fatalf("mailbox run/stream summary mismatch: run=%#v stream=%#v", runSummary, streamSummary)
	}
	if runSummary.CommandTotal != 3 || runSummary.ResultTotal != 2 {
		t.Fatalf("mailbox convergence count mismatch: %#v", runSummary)
	}
	if !runSummary.DelayedBlocked || !runSummary.DelayedReady || !runSummary.CorrelationMapped {
		t.Fatalf("mailbox convergence semantic mismatch: %#v", runSummary)
	}
}

func TestMailboxContractMemoryFileParityAndRestoreReplayDeterminism(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	clock := now
	policy := mailbox.Policy{
		MaxAttempts:    3,
		BackoffInitial: 10 * time.Millisecond,
		BackoffMax:     10 * time.Millisecond,
		JitterRatio:    0,
		DLQEnabled:     true,
	}

	memMB, err := mailbox.New(
		mailbox.NewMemoryStore(policy),
		mailbox.WithClock(func() time.Time { return clock }),
	)
	if err != nil {
		t.Fatalf("new memory mailbox failed: %v", err)
	}
	if _, err := memMB.Publish(ctx, mailbox.Envelope{
		MessageID:      "msg-parity-command",
		IdempotencyKey: "idem-parity-command",
		Kind:           mailbox.KindCommand,
		RunID:          "run-parity",
		TaskID:         "task-parity",
		WorkflowID:     "wf-parity",
		TeamID:         "team-parity",
	}); err != nil {
		t.Fatalf("publish command failed: %v", err)
	}
	clock = clock.Add(1 * time.Millisecond)
	if _, err := memMB.Publish(ctx, mailbox.Envelope{
		MessageID:      "msg-parity-result",
		IdempotencyKey: "idem-parity-result",
		Kind:           mailbox.KindResult,
		RunID:          "run-parity",
		TaskID:         "task-parity",
		WorkflowID:     "wf-parity",
		TeamID:         "team-parity",
	}); err != nil {
		t.Fatalf("publish result failed: %v", err)
	}
	claimed, ok, err := memMB.Consume(ctx, "worker-parity")
	if err != nil || !ok {
		t.Fatalf("consume parity message failed: ok=%v err=%v", ok, err)
	}
	if _, err := memMB.Ack(ctx, claimed.Envelope.MessageID, "worker-parity"); err != nil {
		t.Fatalf("ack parity message failed: %v", err)
	}
	if _, err := memMB.Publish(ctx, mailbox.Envelope{
		MessageID:      "msg-parity-replay-a",
		IdempotencyKey: "idem-parity-replay",
		Kind:           mailbox.KindCommand,
		RunID:          "run-parity",
		TaskID:         "task-parity",
		WorkflowID:     "wf-parity",
		TeamID:         "team-parity",
	}); err != nil {
		t.Fatalf("publish replay #1 failed: %v", err)
	}
	if _, err := memMB.Publish(ctx, mailbox.Envelope{
		MessageID:      "msg-parity-replay-b",
		IdempotencyKey: "idem-parity-replay",
		Kind:           mailbox.KindCommand,
		RunID:          "run-parity",
		TaskID:         "task-parity",
		WorkflowID:     "wf-parity",
		TeamID:         "team-parity",
	}); err != nil {
		t.Fatalf("publish replay #2 failed: %v", err)
	}

	snapshot, err := memMB.Snapshot(ctx)
	if err != nil {
		t.Fatalf("snapshot memory mailbox failed: %v", err)
	}
	fileStore, err := mailbox.NewFileStore(filepath.Join(t.TempDir(), "mailbox-state.json"), policy)
	if err != nil {
		t.Fatalf("new file store failed: %v", err)
	}
	fileMB, err := mailbox.New(
		fileStore,
		mailbox.WithClock(func() time.Time { return clock }),
	)
	if err != nil {
		t.Fatalf("new file mailbox failed: %v", err)
	}
	if err := fileMB.Restore(ctx, snapshot); err != nil {
		t.Fatalf("restore snapshot to file mailbox failed: %v", err)
	}

	pageSize := 2
	req := mailbox.QueryRequest{
		RunID:    "run-parity",
		PageSize: &pageSize,
		Sort: mailbox.QuerySort{
			Field: "updated_at",
			Order: "desc",
		},
	}
	memPages := collectMailboxPages(t, memMB, req)
	filePages := collectMailboxPages(t, fileMB, req)
	if !reflect.DeepEqual(memPages, filePages) {
		t.Fatalf("mailbox memory/file parity mismatch: memory=%#v file=%#v", memPages, filePages)
	}
}

func TestMailboxContractCanonicalEntrypointConvergenceGuard(t *testing.T) {
	root := integrationRepoRoot(t)
	legacyUsages := findLegacyInvokeQualifiedUsages(t, root)
	if len(legacyUsages) > 0 {
		t.Fatalf("legacy direct invoke usage detected outside canonical mailbox path: %#v", legacyUsages)
	}
}

func TestMailboxContractWorkerLifecycleRunStreamSemanticEquivalence(t *testing.T) {
	runSummary := executeMailboxWorkerLifecycleFlow(t, "run", "memory")
	streamSummary := executeMailboxWorkerLifecycleFlow(t, "stream", "memory")
	if !reflect.DeepEqual(runSummary, streamSummary) {
		t.Fatalf("worker lifecycle run/stream mismatch: run=%#v stream=%#v", runSummary, streamSummary)
	}
	if runSummary.ByTransition[string(mailbox.TransitionConsume)] < 2 ||
		runSummary.ByTransition[string(mailbox.TransitionAck)] != 1 ||
		runSummary.ByTransition[string(mailbox.TransitionRequeue)] != 1 {
		t.Fatalf("worker lifecycle transition coverage mismatch: %#v", runSummary)
	}
}

func TestMailboxContractWorkerLifecycleMemoryFileParity(t *testing.T) {
	memSummary := executeMailboxWorkerLifecycleFlow(t, "parity", "memory")
	fileSummary := executeMailboxWorkerLifecycleFlow(t, "parity", "file")
	if !reflect.DeepEqual(memSummary, fileSummary) {
		t.Fatalf("worker lifecycle memory/file parity mismatch: memory=%#v file=%#v", memSummary, fileSummary)
	}
}

func TestMailboxContractWorkerRecoverReclaimRunStreamSemanticEquivalence(t *testing.T) {
	runSummary := executeMailboxWorkerRecoverFlow(t, "run", "memory", mailbox.WorkerHandlerErrorPolicyRequeue)
	streamSummary := executeMailboxWorkerRecoverFlow(t, "stream", "memory", mailbox.WorkerHandlerErrorPolicyRequeue)
	if !reflect.DeepEqual(runSummary, streamSummary) {
		t.Fatalf("worker recover/reclaim run/stream mismatch: run=%#v stream=%#v", runSummary, streamSummary)
	}
	if runSummary.ReclaimedTotal == 0 ||
		runSummary.PanicRecoveredTotal == 0 ||
		runSummary.ByReason[mailbox.LifecycleReasonLeaseExpired] == 0 ||
		runSummary.ByReason[mailbox.LifecycleReasonHandlerError] == 0 {
		t.Fatalf("worker recover/reclaim coverage mismatch: %#v", runSummary)
	}
}

func TestMailboxContractWorkerRecoverReclaimMemoryFileParity(t *testing.T) {
	memSummary := executeMailboxWorkerRecoverFlow(t, "parity", "memory", mailbox.WorkerHandlerErrorPolicyRequeue)
	fileSummary := executeMailboxWorkerRecoverFlow(t, "parity", "file", mailbox.WorkerHandlerErrorPolicyRequeue)
	if !reflect.DeepEqual(memSummary, fileSummary) {
		t.Fatalf("worker recover/reclaim memory/file parity mismatch: memory=%#v file=%#v", memSummary, fileSummary)
	}
}

func TestMailboxContractWorkerPanicNackPolicyDeterministic(t *testing.T) {
	summary := executeMailboxWorkerRecoverFlow(t, "nack", "memory", mailbox.WorkerHandlerErrorPolicyNack)
	if summary.ByTransition[string(mailbox.TransitionNack)] == 0 {
		t.Fatalf("expected panic path nack transition, got %#v", summary)
	}
	if summary.FinalStateByMessage["msg-worker-panic"] != mailbox.StateNacked {
		t.Fatalf("panic message final state=%q, want %q; summary=%#v", summary.FinalStateByMessage["msg-worker-panic"], mailbox.StateNacked, summary)
	}
}

func TestMailboxContractWorkerHeartbeatNoPrematureReclaim(t *testing.T) {
	ctx := context.Background()
	base := time.Now().UTC()
	var mu sync.Mutex
	clock := base
	events := make([]mailbox.LifecycleEvent, 0, 32)
	mb, err := mailbox.New(
		mailbox.NewMemoryStore(mailbox.Policy{
			MaxAttempts:    3,
			BackoffInitial: 1 * time.Millisecond,
			BackoffMax:     1 * time.Millisecond,
			JitterRatio:    0,
		}),
		mailbox.WithClock(func() time.Time {
			mu.Lock()
			defer mu.Unlock()
			return clock
		}),
		mailbox.WithLifecycleObserver(func(_ context.Context, event mailbox.LifecycleEvent) {
			events = append(events, event)
		}),
	)
	if err != nil {
		t.Fatalf("new mailbox failed: %v", err)
	}
	if _, err := mb.Publish(ctx, mailbox.Envelope{
		MessageID:      "msg-heartbeat-contract",
		IdempotencyKey: "idem-heartbeat-contract",
		Kind:           mailbox.KindCommand,
	}); err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	release := make(chan struct{})
	started := make(chan struct{})
	workerA, err := mailbox.NewWorker(mb, mailbox.WorkerConfig{
		Enabled:           true,
		InflightTimeout:   120 * time.Millisecond,
		HeartbeatInterval: 20 * time.Millisecond,
	}, func(context.Context, mailbox.Record) error {
		close(started)
		<-release
		return nil
	}, "worker-heartbeat-a")
	if err != nil {
		t.Fatalf("new workerA failed: %v", err)
	}
	runDone := make(chan error, 1)
	go func() {
		_, runErr := workerA.RunOnce(ctx)
		runDone <- runErr
	}()
	<-started

	go func() {
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for i := 0; i < 20; i++ {
			<-ticker.C
			mu.Lock()
			clock = clock.Add(10 * time.Millisecond)
			mu.Unlock()
		}
	}()
	time.Sleep(170 * time.Millisecond)

	workerB, err := mailbox.NewWorker(mb, mailbox.WorkerConfig{
		Enabled:           true,
		InflightTimeout:   120 * time.Millisecond,
		HeartbeatInterval: 20 * time.Millisecond,
	}, func(context.Context, mailbox.Record) error {
		return nil
	}, "worker-heartbeat-b")
	if err != nil {
		t.Fatalf("new workerB failed: %v", err)
	}
	processed, err := workerB.RunOnce(ctx)
	if err != nil {
		t.Fatalf("workerB RunOnce failed: %v", err)
	}
	if processed {
		t.Fatalf("workerB should not reclaim active in-flight message, events=%#v", events)
	}
	close(release)
	if runErr := <-runDone; runErr != nil {
		t.Fatalf("workerA RunOnce failed: %v", runErr)
	}
}

func TestMailboxContractWorkerDisabledNoopBaseline(t *testing.T) {
	mb, err := mailbox.New(mailbox.NewMemoryStore(mailbox.Policy{}))
	if err != nil {
		t.Fatalf("new mailbox failed: %v", err)
	}
	ctx := context.Background()
	if _, err := mb.Publish(ctx, mailbox.Envelope{
		MessageID:      "msg-worker-disabled",
		IdempotencyKey: "idem-worker-disabled",
		Kind:           mailbox.KindCommand,
	}); err != nil {
		t.Fatalf("publish failed: %v", err)
	}
	worker, err := mailbox.NewWorker(mb, mailbox.WorkerConfig{Enabled: false}, func(context.Context, mailbox.Record) error {
		return nil
	}, "worker-disabled")
	if err != nil {
		t.Fatalf("new worker failed: %v", err)
	}
	if err := worker.Run(ctx); err != nil {
		t.Fatalf("disabled worker run failed: %v", err)
	}
	rec, ok, err := mb.Consume(ctx, "worker-disabled-check")
	if err != nil || !ok {
		t.Fatalf("message should remain queued when worker disabled: ok=%v err=%v", ok, err)
	}
	if rec.State != mailbox.StateInFlight {
		t.Fatalf("disabled baseline claimed state=%q, want in_flight", rec.State)
	}
}

func TestMailboxContractLifecycleReasonTaxonomyGuard(t *testing.T) {
	required := []string{
		mailbox.LifecycleReasonRetryExhausted,
		mailbox.LifecycleReasonExpired,
		mailbox.LifecycleReasonConsumerMismatch,
		mailbox.LifecycleReasonMessageNotFound,
		mailbox.LifecycleReasonHandlerError,
		mailbox.LifecycleReasonLeaseExpired,
	}
	if got := mailbox.LifecycleCanonicalReasons(); !reflect.DeepEqual(got, required) {
		t.Fatalf("canonical lifecycle reason set drift: got=%#v want=%#v", got, required)
	}

	ctx := context.Background()
	var seen []mailbox.LifecycleEvent
	mb, err := mailbox.New(
		mailbox.NewMemoryStore(mailbox.Policy{
			MaxAttempts:    2,
			BackoffInitial: 1 * time.Millisecond,
			BackoffMax:     1 * time.Millisecond,
			JitterRatio:    0,
		}),
		mailbox.WithLifecycleObserver(func(_ context.Context, event mailbox.LifecycleEvent) {
			seen = append(seen, event)
		}),
	)
	if err != nil {
		t.Fatalf("new mailbox failed: %v", err)
	}
	if _, err := mb.Publish(ctx, mailbox.Envelope{
		MessageID:      "msg-taxonomy-guard",
		IdempotencyKey: "idem-taxonomy-guard",
		Kind:           mailbox.KindCommand,
	}); err != nil {
		t.Fatalf("publish failed: %v", err)
	}
	worker, err := mailbox.NewWorker(mb, mailbox.WorkerConfig{Enabled: true}, func(context.Context, mailbox.Record) error {
		return errors.New("non-canonical-transient-error")
	}, "worker-taxonomy")
	if err != nil {
		t.Fatalf("new worker failed: %v", err)
	}
	processed, err := worker.RunOnce(ctx)
	if err != nil || !processed {
		t.Fatalf("RunOnce failed: processed=%v err=%v", processed, err)
	}
	for _, event := range seen {
		if strings.TrimSpace(event.ReasonCode) == "" {
			continue
		}
		if !mailbox.IsCanonicalLifecycleReason(event.ReasonCode) {
			t.Fatalf("non-canonical lifecycle reason detected: %#v", event)
		}
	}
	foundHandlerError := false
	for _, event := range seen {
		if event.Transition == mailbox.TransitionRequeue && event.ReasonCode == mailbox.LifecycleReasonHandlerError {
			foundHandlerError = true
			break
		}
	}
	if !foundHandlerError {
		t.Fatalf("expected requeue reason mapped to handler_error, events=%#v", seen)
	}
}

type mailboxConvergenceSummary struct {
	CommandTotal      int
	ResultTotal       int
	DelayedBlocked    bool
	DelayedReady      bool
	CorrelationMapped bool
}

type mailboxWorkerLifecycleSummary struct {
	ByTransition map[string]int
	ByReason     map[string]int
	FinalState   mailbox.MessageState
	Attempts     int
}

type mailboxWorkerRecoverSummary struct {
	ByTransition        map[string]int
	ByReason            map[string]int
	FinalStateByMessage map[string]mailbox.MessageState
	ReclaimedTotal      int
	PanicRecoveredTotal int
}

func executeMailboxConvergenceFlow(t *testing.T, label string) mailboxConvergenceSummary {
	t.Helper()

	ctx := context.Background()
	now := time.Now().UTC()
	clock := now
	mb, err := mailbox.New(
		mailbox.NewMemoryStore(mailbox.Policy{}),
		mailbox.WithClock(func() time.Time { return clock }),
	)
	if err != nil {
		t.Fatalf("new mailbox failed: %v", err)
	}
	bridge := invoke.NewMailboxBridge(mb)

	workflowID := "wf-mailbox-" + label
	teamID := "team-mailbox-" + label

	outcome, err := bridge.InvokeSync(ctx, syncMailboxClient{}, invoke.Request{
		TaskID:     "task-sync-" + label,
		WorkflowID: workflowID,
		TeamID:     teamID,
		AgentID:    "agent-main",
		PeerID:     "peer-remote",
		Method:     "workflow.dispatch",
		Payload:    map[string]any{"mode": "sync"},
	})
	if err != nil {
		t.Fatalf("mailbox sync invoke failed: %v", err)
	}
	if outcome.TerminalStatus != a2a.StatusSucceeded {
		t.Fatalf("mailbox sync status mismatch: %#v", outcome)
	}

	if _, err := bridge.InvokeAsync(ctx, asyncMailboxClient{}, invoke.AsyncRequest{
		TaskID:     "task-async-" + label,
		WorkflowID: workflowID,
		TeamID:     teamID,
		AgentID:    "agent-main",
		PeerID:     "peer-remote",
		Method:     "workflow.dispatch",
		Payload:    map[string]any{"mode": "async"},
	}, nil); err != nil {
		t.Fatalf("mailbox async invoke failed: %v", err)
	}

	for {
		claimed, ok, err := mb.Consume(ctx, "worker-drain-"+label)
		if err != nil {
			t.Fatalf("drain consume failed: %v", err)
		}
		if !ok {
			break
		}
		if _, err := mb.Ack(ctx, claimed.Envelope.MessageID, "worker-drain-"+label); err != nil {
			t.Fatalf("drain ack failed: %v", err)
		}
	}

	delayedID := "task-delayed-" + label
	if _, err := bridge.PublishDelayedCommand(ctx, invoke.Request{
		TaskID:     delayedID,
		WorkflowID: workflowID,
		TeamID:     teamID,
		AgentID:    "agent-main",
		PeerID:     "peer-remote",
		Method:     "workflow.dispatch",
		Payload:    map[string]any{"mode": "delayed"},
	}, now.Add(200*time.Millisecond), now.Add(2*time.Second)); err != nil {
		t.Fatalf("publish delayed command failed: %v", err)
	}
	_, delayedBlocked, err := mb.Consume(ctx, "worker-delay-"+label)
	if err != nil {
		t.Fatalf("delayed consume before boundary failed: %v", err)
	}
	clock = clock.Add(250 * time.Millisecond)
	delayedClaimed, delayedReady, err := mb.Consume(ctx, "worker-delay-"+label)
	if err != nil {
		t.Fatalf("delayed consume after boundary failed: %v", err)
	}
	if !delayedReady {
		t.Fatalf("delayed command should be consumable after not_before")
	}
	if delayedClaimed.Envelope.TaskID != delayedID {
		t.Fatalf("delayed command claimed task mismatch: %#v", delayedClaimed)
	}
	if _, err := mb.Ack(ctx, delayedClaimed.Envelope.MessageID, "worker-delay-"+label); err != nil {
		t.Fatalf("ack delayed command failed: %v", err)
	}

	page, err := mb.Query(ctx, mailbox.QueryRequest{
		TeamID: teamID,
	})
	if err != nil {
		t.Fatalf("mailbox query failed: %v", err)
	}
	summary := mailboxConvergenceSummary{
		DelayedBlocked: !delayedBlocked,
		DelayedReady:   delayedReady,
	}
	correlationMapped := true
	for _, item := range page.Items {
		switch item.Envelope.Kind {
		case mailbox.KindCommand:
			summary.CommandTotal++
		case mailbox.KindResult:
			summary.ResultTotal++
			if item.Envelope.CorrelationID == "" {
				correlationMapped = false
			}
		}
		if item.Envelope.RunID == "" || item.Envelope.TaskID == "" || item.Envelope.WorkflowID == "" || item.Envelope.TeamID == "" {
			correlationMapped = false
		}
	}
	summary.CorrelationMapped = correlationMapped
	return summary
}

func executeMailboxWorkerLifecycleFlow(t *testing.T, label, backend string) mailboxWorkerLifecycleSummary {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()
	clock := now
	events := make([]mailbox.LifecycleEvent, 0, 8)
	policy := mailbox.Policy{
		MaxAttempts:    3,
		BackoffInitial: 10 * time.Millisecond,
		BackoffMax:     10 * time.Millisecond,
		JitterRatio:    0,
		DLQEnabled:     true,
	}
	var store mailbox.Store
	switch backend {
	case "file":
		fileStore, err := mailbox.NewFileStore(filepath.Join(t.TempDir(), "worker-mailbox.json"), policy)
		if err != nil {
			t.Fatalf("new file mailbox failed: %v", err)
		}
		store = fileStore
	default:
		store = mailbox.NewMemoryStore(policy)
	}
	mb, err := mailbox.New(
		store,
		mailbox.WithClock(func() time.Time { return clock }),
		mailbox.WithLifecycleObserver(func(_ context.Context, event mailbox.LifecycleEvent) {
			events = append(events, event)
		}),
	)
	if err != nil {
		t.Fatalf("new mailbox failed: %v", err)
	}
	if _, err := mb.Publish(ctx, mailbox.Envelope{
		MessageID:      "msg-worker-" + label,
		IdempotencyKey: "idem-worker-" + label,
		Kind:           mailbox.KindCommand,
		RunID:          "run-worker-" + label,
		TaskID:         "task-worker-" + label,
		WorkflowID:     "wf-worker-" + label,
		TeamID:         "team-worker-" + label,
	}); err != nil {
		t.Fatalf("publish worker command failed: %v", err)
	}
	calls := 0
	worker, err := mailbox.NewWorker(mb, mailbox.WorkerConfig{Enabled: true}, func(context.Context, mailbox.Record) error {
		calls++
		if calls == 1 {
			return errors.New("handler transient")
		}
		return nil
	}, "worker-"+label)
	if err != nil {
		t.Fatalf("new worker failed: %v", err)
	}
	processed, err := worker.RunOnce(ctx)
	if err != nil || !processed {
		t.Fatalf("first RunOnce failed: processed=%v err=%v", processed, err)
	}
	clock = clock.Add(20 * time.Millisecond)
	processed, err = worker.RunOnce(ctx)
	if err != nil || !processed {
		t.Fatalf("second RunOnce failed: processed=%v err=%v", processed, err)
	}

	page, err := mb.Query(ctx, mailbox.QueryRequest{
		RunID: "run-worker-" + label,
	})
	if err != nil {
		t.Fatalf("query worker records failed: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("worker query record count=%d, want 1", len(page.Items))
	}
	summary := mailboxWorkerLifecycleSummary{
		ByTransition: map[string]int{},
		ByReason:     map[string]int{},
		FinalState:   page.Items[0].State,
		Attempts:     page.Items[0].DeliveryAttempt,
	}
	for _, event := range events {
		key := strings.TrimSpace(string(event.Transition))
		if key != "" {
			summary.ByTransition[key]++
		}
		reason := strings.TrimSpace(event.ReasonCode)
		if reason != "" {
			summary.ByReason[reason]++
		}
	}
	return summary
}

func executeMailboxWorkerRecoverFlow(
	t *testing.T,
	label, backend, handlerPolicy string,
) mailboxWorkerRecoverSummary {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()
	clock := now
	events := make([]mailbox.LifecycleEvent, 0, 32)
	policy := mailbox.Policy{
		MaxAttempts:    3,
		BackoffInitial: 10 * time.Millisecond,
		BackoffMax:     10 * time.Millisecond,
		JitterRatio:    0,
		DLQEnabled:     false,
	}
	var store mailbox.Store
	switch backend {
	case "file":
		fileStore, err := mailbox.NewFileStore(filepath.Join(t.TempDir(), "worker-recover-mailbox.json"), policy)
		if err != nil {
			t.Fatalf("new file mailbox failed: %v", err)
		}
		store = fileStore
	default:
		store = mailbox.NewMemoryStore(policy)
	}
	mb, err := mailbox.New(
		store,
		mailbox.WithClock(func() time.Time { return clock }),
		mailbox.WithLifecycleObserver(func(_ context.Context, event mailbox.LifecycleEvent) {
			events = append(events, event)
		}),
	)
	if err != nil {
		t.Fatalf("new mailbox failed: %v", err)
	}

	reclaimID := "msg-worker-reclaim"
	panicID := "msg-worker-panic"
	if _, err := mb.Publish(ctx, mailbox.Envelope{
		MessageID:      reclaimID,
		IdempotencyKey: "idem-worker-reclaim-" + label,
		Kind:           mailbox.KindCommand,
		RunID:          "run-worker-recover-" + label,
		TaskID:         "task-worker-recover-" + label,
		WorkflowID:     "wf-worker-recover-" + label,
		TeamID:         "team-worker-recover-" + label,
	}); err != nil {
		t.Fatalf("publish reclaim command failed: %v", err)
	}
	if _, err := mb.Publish(ctx, mailbox.Envelope{
		MessageID:      panicID,
		IdempotencyKey: "idem-worker-panic-" + label,
		Kind:           mailbox.KindCommand,
		RunID:          "run-worker-recover-" + label,
		TaskID:         "task-worker-recover-" + label,
		WorkflowID:     "wf-worker-recover-" + label,
		TeamID:         "team-worker-recover-" + label,
	}); err != nil {
		t.Fatalf("publish panic command failed: %v", err)
	}
	if _, ok, err := mb.ConsumeWithLease(ctx, "worker-crash-"+label, 30*time.Millisecond, true); err != nil || !ok {
		t.Fatalf("consume for crash simulation failed: ok=%v err=%v", ok, err)
	}

	clock = clock.Add(40 * time.Millisecond)
	panicConsumed := false
	worker, err := mailbox.NewWorker(mb, mailbox.WorkerConfig{
		Enabled:            true,
		HandlerErrorPolicy: handlerPolicy,
		InflightTimeout:    30 * time.Millisecond,
		HeartbeatInterval:  5 * time.Millisecond,
		ReclaimOnConsume:   true,
		PanicPolicy:        mailbox.WorkerPanicPolicyFollowHandler,
	}, func(_ context.Context, rec mailbox.Record) error {
		if rec.Envelope.MessageID == panicID && !panicConsumed {
			panicConsumed = true
			panic("panic contract")
		}
		return nil
	}, "worker-recover-"+label)
	if err != nil {
		t.Fatalf("new worker failed: %v", err)
	}

	for i := 0; i < 8; i++ {
		_, runErr := worker.RunOnce(ctx)
		if runErr != nil {
			t.Fatalf("RunOnce #%d failed: %v", i+1, runErr)
		}
		clock = clock.Add(20 * time.Millisecond)
	}

	page, err := mb.Query(ctx, mailbox.QueryRequest{RunID: "run-worker-recover-" + label})
	if err != nil {
		t.Fatalf("query worker recover records failed: %v", err)
	}
	summary := mailboxWorkerRecoverSummary{
		ByTransition:        map[string]int{},
		ByReason:            map[string]int{},
		FinalStateByMessage: map[string]mailbox.MessageState{},
	}
	for _, item := range page.Items {
		summary.FinalStateByMessage[item.Envelope.MessageID] = item.State
	}
	for _, event := range events {
		key := strings.TrimSpace(string(event.Transition))
		if key != "" {
			summary.ByTransition[key]++
		}
		reason := strings.TrimSpace(event.ReasonCode)
		if reason != "" {
			summary.ByReason[reason]++
		}
		if event.Reclaimed {
			summary.ReclaimedTotal++
		}
		if event.PanicRecovered {
			summary.PanicRecoveredTotal++
		}
	}
	return summary
}

func collectMailboxPages(t *testing.T, mb *mailbox.Mailbox, req mailbox.QueryRequest) []mailbox.QueryResult {
	t.Helper()
	ctx := context.Background()
	pages := make([]mailbox.QueryResult, 0)
	seenCursor := map[string]struct{}{}
	current := req
	for {
		page, err := mb.Query(ctx, current)
		if err != nil {
			t.Fatalf("mailbox query page failed: %v", err)
		}
		pages = append(pages, page)
		next := page.NextCursor
		if next == "" {
			break
		}
		if _, exists := seenCursor[next]; exists {
			t.Fatalf("mailbox cursor must advance deterministically without loops: %q", next)
		}
		seenCursor[next] = struct{}{}
		current.Cursor = next
	}
	return pages
}

type syncMailboxClient struct{}

func (syncMailboxClient) Submit(_ context.Context, req a2a.TaskRequest) (a2a.TaskRecord, error) {
	return a2a.TaskRecord{
		TaskID: req.TaskID,
		Status: a2a.StatusSubmitted,
	}, nil
}

func (syncMailboxClient) WaitResult(
	_ context.Context,
	taskID string,
	_ time.Duration,
	_ func(context.Context, a2a.TaskRecord) error,
) (a2a.TaskRecord, error) {
	return a2a.TaskRecord{
		TaskID: taskID,
		Status: a2a.StatusSucceeded,
		Result: map[string]any{"ok": true},
	}, nil
}

type asyncMailboxClient struct{}

func (asyncMailboxClient) SubmitAsync(ctx context.Context, req a2a.TaskRequest, sink a2a.ReportSink) (a2a.AsyncSubmitAck, error) {
	report := a2a.AsyncReport{
		ReportKey:        req.TaskID + "-report",
		OutcomeKey:       "succeeded|ok",
		WorkflowID:       req.WorkflowID,
		TeamID:           req.TeamID,
		TaskID:           req.TaskID,
		AgentID:          req.AgentID,
		PeerID:           req.PeerID,
		Status:           a2a.StatusSucceeded,
		Result:           map[string]any{"ok": true},
		BusinessTerminal: true,
		UpdatedAt:        time.Now().UTC(),
	}
	if sink != nil {
		if err := sink.Deliver(ctx, report); err != nil {
			return a2a.AsyncSubmitAck{}, err
		}
	}
	return a2a.AsyncSubmitAck{
		TaskID:     req.TaskID,
		WorkflowID: req.WorkflowID,
		TeamID:     req.TeamID,
		AgentID:    req.AgentID,
		PeerID:     req.PeerID,
		AcceptedAt: time.Now().UTC(),
	}, nil
}
