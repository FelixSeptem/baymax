package eventadapter

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/orchestration/mailbox"
)

func TestCallbackAdapterNeverOwnsHistoryOrMintsJournalIdentity(t *testing.T) {
	ctx := context.Background()
	var events []types.NativeExecutionEvent
	o, err := New(options(SinkFunc(func(_ context.Context, event types.NativeExecutionEvent) error {
		events = append(events, event)
		return nil
	})))
	if err != nil {
		t.Fatal(err)
	}
	defer o.Close()
	if _, ownsHistory := any(o).(types.NativeEventHistoryReader); ownsHistory {
		t.Fatal("callback adapter became a source history owner")
	}
	o.OnEvent(ctx, fact())
	o.OnEvent(ctx, fact())
	if len(events) != 2 || events[0].Event.EventID == "" || events[0].Event.EventID != events[1].Event.EventID {
		t.Fatalf("callback delivery or mapping identity changed: %+v", events)
	}
	for _, event := range events {
		if event.History != nil || event.Capabilities.Identity != "mapping_reference" || event.Capabilities.Ordering != "callback_order" || event.Capabilities.Cursor || event.Capabilities.DurableHistory || event.Capabilities.Deduplication || event.Capabilities.GapVerification {
			t.Fatalf("callback was advertised as journal evidence: %+v", event)
		}
	}

	owner, err := mailbox.New(mailbox.NewMemoryStore(mailbox.Policy{}))
	if err != nil {
		t.Fatal(err)
	}
	source, err := owner.HistorySource(ctx)
	if err != nil {
		t.Fatal(err)
	}
	before, err := owner.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, callbackReference := range []string{events[0].Event.EventID, events[0].TraceID} {
		if _, err := owner.ReadAfter(ctx, source, &types.OpaqueCursor{Value: callbackReference}, 1); !errors.Is(err, types.ErrCursorMalformed) {
			t.Fatalf("callback reference accepted as mailbox cursor: %q %v", callbackReference, err)
		}
	}
	after, err := owner.Snapshot(ctx)
	if err != nil || !reflect.DeepEqual(before, after) || o.Status().Delivered != 2 {
		t.Fatalf("cursor rejection activated mailbox or adapter: err=%v status=%+v", err, o.Status())
	}
}

func TestRetainedFactCanReachSinkWithoutCallbackIdentityRewrite(t *testing.T) {
	ctx := context.Background()
	owner, err := mailbox.New(mailbox.NewMemoryStore(mailbox.Policy{}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Publish(ctx, mailbox.Envelope{MessageID: "retained", IdempotencyKey: "retained", Kind: mailbox.KindCommand, RunID: "source-run"}); err != nil {
		t.Fatal(err)
	}
	source, err := owner.HistorySource(ctx)
	if err != nil {
		t.Fatal(err)
	}
	page, err := owner.ReadAfter(ctx, source, nil, 1)
	if err != nil || len(page.Events) != 1 {
		t.Fatalf("source page: %+v err=%v", page, err)
	}
	before, err := owner.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var projected types.NativeExecutionEvent
	sink := SinkFunc(func(_ context.Context, event types.NativeExecutionEvent) error {
		projected = event
		return nil
	})
	if err := sink.Consume(ctx, page.Events[0]); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(projected, page.Events[0]) || projected.History == nil || projected.History.Source != source || projected.History.Sequence != 1 || projected.Capabilities.Ordering != "source_sequence" {
		t.Fatalf("retained source identity was rewritten: %+v", projected)
	}
	after, err := owner.Snapshot(ctx)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("host fact projection mutated source: %v", err)
	}
}
