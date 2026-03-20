package integration

import (
	"context"
	"path/filepath"
	"reflect"
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

type mailboxConvergenceSummary struct {
	CommandTotal      int
	ResultTotal       int
	DelayedBlocked    bool
	DelayedReady      bool
	CorrelationMapped bool
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
