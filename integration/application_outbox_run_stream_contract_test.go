package integration

import (
	"context"
	"testing"

	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/integration/fakes"
	"github.com/FelixSeptem/baymax/orchestration/mailbox"
	"github.com/FelixSeptem/baymax/orchestration/outboxbridge"
)

func TestApplicationOutboxDispatchRunStreamConformance(t *testing.T) {
	run := executeApplicationOutboxSource(t, false)
	stream := executeApplicationOutboxSource(t, true)
	if run.answer != stream.answer || run.runID != stream.runID || run.messageID == "" || stream.messageID == "" {
		t.Fatalf("Run/Stream normalized source drift: run=%+v stream=%+v", run, stream)
	}
	if run.messageID != stream.messageID {
		t.Fatalf("delivery correlation drift: run=%q stream=%q", run.messageID, stream.messageID)
	}
	if run.terminal == nil || stream.terminal == nil {
		t.Fatalf("source terminal outcome missing: run=%+v stream=%+v", run.terminal, stream.terminal)
	}
	if run.dispatchAccepted != stream.dispatchAccepted || !run.dispatchAccepted {
		t.Fatalf("dispatch acceptance drift: run=%v stream=%v", run.dispatchAccepted, stream.dispatchAccepted)
	}
	if run.businessCommitted || stream.businessCommitted {
		t.Fatal("delivery acceptance must not imply business commit")
	}
}

type applicationOutboxSourceResult struct {
	answer            string
	runID             string
	messageID         string
	dispatchAccepted  bool
	businessCommitted bool
	terminal          *types.TerminalOutcome
}

func executeApplicationOutboxSource(t *testing.T, stream bool) applicationOutboxSourceResult {
	t.Helper()
	d := outboxbridge.Descriptor{
		Target:         outboxbridge.TargetMailbox,
		OutboxID:       "source-outbox",
		Revision:       "revision-1",
		AuthorityRef:   "authority-1",
		EffectRef:      "effect-1",
		EffectKey:      "effect-key-1",
		RunID:          "source-run",
		IdempotencyKey: "source-dispatch",
		Mailbox:        &outboxbridge.MailboxBinding{ToAgent: "source-worker"},
	}
	_, digest, err := outboxbridge.Canonical(d)
	if err != nil {
		t.Fatal(err)
	}
	owner, err := mailbox.New(mailbox.NewMemoryStore(mailbox.Policy{}))
	if err != nil {
		t.Fatal(err)
	}
	bridge, err := outboxbridge.New(outboxbridge.Config{
		Mailbox: owner,
		Admission: func(context.Context, outboxbridge.Descriptor, string) (outboxbridge.Decision, error) {
			return outboxbridge.Decision{Eligible: true, Digest: digest}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	dispatched, err := bridge.Dispatch(context.Background(), d)
	if err != nil {
		t.Fatal(err)
	}
	model := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "source-ok"}}})
	model.SetStream([]types.ModelEvent{{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "source-ok"}}, nil)
	eng := runner.New(model)
	request := types.RunRequest{RunID: d.RunID, Input: "source"}
	var result types.RunResult
	if stream {
		result, err = eng.Stream(context.Background(), request, nil)
	} else {
		result, err = eng.Run(context.Background(), request, nil)
	}
	if err != nil {
		t.Fatal(err)
	}
	return applicationOutboxSourceResult{
		answer:            result.FinalAnswer,
		runID:             result.RunID,
		messageID:         dispatched.Facts.MessageID,
		dispatchAccepted:  dispatched.Facts.DispatchAccepted,
		businessCommitted: dispatched.Facts.Effect.EffectCommitted,
		terminal:          result.TerminalOutcome,
	}
}
