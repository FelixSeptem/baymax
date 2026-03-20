package invoke

import (
	"context"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/a2a"
	"github.com/FelixSeptem/baymax/orchestration/mailbox"
)

func TestMailboxBridgeInvokeSyncPublishesCommandAndResult(t *testing.T) {
	ctx := context.Background()
	mb, err := mailbox.New(mailbox.NewMemoryStore(mailbox.Policy{}))
	if err != nil {
		t.Fatalf("new mailbox failed: %v", err)
	}
	bridge := NewMailboxBridge(mb)
	client := &fakeClient{
		waitFn: func(_ context.Context, taskID string, _ time.Duration, _ func(context.Context, a2a.TaskRecord) error) (a2a.TaskRecord, error) {
			return a2a.TaskRecord{
				TaskID: taskID,
				Status: a2a.StatusSucceeded,
				Result: map[string]any{"ok": true},
			}, nil
		},
	}
	outcome, err := bridge.InvokeSync(ctx, client, Request{
		TaskID:     "task-sync-mb",
		WorkflowID: "wf-sync",
		TeamID:     "team-sync",
		AgentID:    "agent-a",
		PeerID:     "peer-b",
		Method:     "workflow.dispatch",
		Payload:    map[string]any{"q": "ping"},
	})
	if err != nil {
		t.Fatalf("InvokeSync via mailbox bridge failed: %v", err)
	}
	if outcome.TerminalStatus != a2a.StatusSucceeded {
		t.Fatalf("terminal status mismatch: %#v", outcome)
	}
	page, err := mb.Query(ctx, mailbox.QueryRequest{
		TaskID: "task-sync-mb",
	})
	if err != nil {
		t.Fatalf("query mailbox failed: %v", err)
	}
	if len(page.Items) != 2 {
		t.Fatalf("expected command+result envelopes, got %#v", page.Items)
	}
	kinds := map[mailbox.EnvelopeKind]int{}
	commandMessageID := ""
	for _, item := range page.Items {
		kinds[item.Envelope.Kind]++
		if item.Envelope.RunID != "wf-sync" ||
			item.Envelope.TaskID != "task-sync-mb" ||
			item.Envelope.WorkflowID != "wf-sync" ||
			item.Envelope.TeamID != "team-sync" {
			t.Fatalf("sync correlation mapping mismatch: %#v", item.Envelope)
		}
		if item.Envelope.Kind == mailbox.KindCommand {
			commandMessageID = item.Envelope.MessageID
		}
	}
	if commandMessageID == "" {
		t.Fatalf("missing command envelope in mailbox page: %#v", page.Items)
	}
	for _, item := range page.Items {
		if item.Envelope.Kind == mailbox.KindResult && item.Envelope.CorrelationID != commandMessageID {
			t.Fatalf("result correlation_id mismatch: got=%q want=%q", item.Envelope.CorrelationID, commandMessageID)
		}
	}
	if kinds[mailbox.KindCommand] != 1 || kinds[mailbox.KindResult] != 1 {
		t.Fatalf("command/result distribution mismatch: %#v", kinds)
	}
}

func TestMailboxBridgeInvokeAsyncPublishesResultFromReport(t *testing.T) {
	ctx := context.Background()
	mb, err := mailbox.New(mailbox.NewMemoryStore(mailbox.Policy{}))
	if err != nil {
		t.Fatalf("new mailbox failed: %v", err)
	}
	bridge := NewMailboxBridge(mb)

	client := fakeAsyncClient{
		submitAsyncFn: func(ctx context.Context, req a2a.TaskRequest, sink a2a.ReportSink) (a2a.AsyncSubmitAck, error) {
			report := a2a.AsyncReport{
				ReportKey:        "report-1",
				OutcomeKey:       "succeeded|ok",
				WorkflowID:       req.WorkflowID,
				TeamID:           req.TeamID,
				TaskID:           req.TaskID,
				AgentID:          req.AgentID,
				PeerID:           req.PeerID,
				Status:           a2a.StatusSucceeded,
				Result:           map[string]any{"ok": true},
				BusinessTerminal: true,
				UpdatedAt:        time.Now(),
			}
			if err := sink.Deliver(ctx, report); err != nil {
				return a2a.AsyncSubmitAck{}, err
			}
			return a2a.AsyncSubmitAck{TaskID: req.TaskID, AcceptedAt: time.Now()}, nil
		},
	}
	if _, err := bridge.InvokeAsync(ctx, client, AsyncRequest{
		TaskID:     "task-async-mb",
		WorkflowID: "wf-async",
		TeamID:     "team-async",
		AgentID:    "agent-a",
		PeerID:     "peer-b",
		Method:     "workflow.dispatch",
	}, nil); err != nil {
		t.Fatalf("InvokeAsync via mailbox bridge failed: %v", err)
	}
	page, err := mb.Query(ctx, mailbox.QueryRequest{
		TaskID: "task-async-mb",
		Kind:   string(mailbox.KindResult),
	})
	if err != nil {
		t.Fatalf("query mailbox failed: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("expected one result envelope, got %#v", page.Items)
	}
	result := page.Items[0]
	if result.Envelope.IdempotencyKey != "report-1" {
		t.Fatalf("async idempotency key mismatch: %#v", result)
	}
	if result.Envelope.RunID != "wf-async" ||
		result.Envelope.TaskID != "task-async-mb" ||
		result.Envelope.WorkflowID != "wf-async" ||
		result.Envelope.TeamID != "team-async" {
		t.Fatalf("async correlation mapping mismatch: %#v", result.Envelope)
	}
}

func TestMailboxBridgePublishDelayedCommand(t *testing.T) {
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
	bridge := NewMailboxBridge(mb)
	if _, err := bridge.PublishDelayedCommand(ctx, Request{
		TaskID:     "task-delayed-mb",
		WorkflowID: "wf-delayed",
		TeamID:     "team-delayed",
		AgentID:    "agent-a",
		PeerID:     "peer-b",
		Payload:    map[string]any{"mode": "delayed"},
	}, now.Add(200*time.Millisecond), now.Add(2*time.Second)); err != nil {
		t.Fatalf("publish delayed command failed: %v", err)
	}
	if _, ok, err := mb.Consume(ctx, "worker-delayed"); err != nil || ok {
		t.Fatalf("delayed command should be blocked before not_before: ok=%v err=%v", ok, err)
	}
	clock = clock.Add(250 * time.Millisecond)
	claimed, ok, err := mb.Consume(ctx, "worker-delayed")
	if err != nil || !ok {
		t.Fatalf("delayed command should be consumable after not_before: ok=%v err=%v", ok, err)
	}
	if claimed.Envelope.RunID != "wf-delayed" ||
		claimed.Envelope.TaskID != "task-delayed-mb" ||
		claimed.Envelope.WorkflowID != "wf-delayed" ||
		claimed.Envelope.TeamID != "team-delayed" {
		t.Fatalf("delayed correlation mapping mismatch: %#v", claimed.Envelope)
	}
}
