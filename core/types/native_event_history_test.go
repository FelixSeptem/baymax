package types

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestNativeHistoryBoundsAndDetachedPage(t *testing.T) {
	source := SourceRef{Kind: HistorySourceMailbox, Namespace: "mailbox.history.v1", ID: strings.Repeat("s", MaxHistoryReferenceBytes)}
	if err := source.Validate(); err != nil {
		t.Fatal(err)
	}
	source.ID += "s"
	if err := source.Validate(); !errors.Is(err, ErrHistoryInvalid) {
		t.Fatalf("oversized source: %v", err)
	}
	for _, value := range []string{"", " ", string([]byte{0xff}), strings.Repeat("c", MaxHistoryCursorBytes+1)} {
		if err := (OpaqueCursor{Value: value}).Validate(); !errors.Is(err, ErrCursorMalformed) {
			t.Fatalf("cursor %q: %v", value, err)
		}
	}
	if err := (OpaqueCursor{Value: strings.Repeat("c", MaxHistoryCursorBytes)}).Validate(); err != nil {
		t.Fatal(err)
	}
	for _, limit := range []int{-1, 0, MaxHistoryPageSize + 1} {
		if ValidateHistoryLimit(limit) == nil {
			t.Fatalf("accepted %d", limit)
		}
	}
	for _, limit := range []int{1, MaxHistoryPageSize} {
		if err := ValidateHistoryLimit(limit); err != nil {
			t.Fatal(err)
		}
	}
	page := NativeEventPage{Events: []NativeExecutionEvent{{Event: EventEnvelope{Payload: map[string]any{"nested": map[string]any{"value": "original"}}}}}, NextCursor: &OpaqueCursor{Value: "cursor"}}
	copy, err := page.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	copy.Events[0].Event.Payload["nested"].(map[string]any)["value"] = "changed"
	copy.NextCursor.Value = "changed"
	if page.NextCursor.Value != "cursor" || page.Events[0].Event.Payload["nested"].(map[string]any)["value"] != "original" {
		t.Fatal("snapshot aliases caller data")
	}
}

func TestNativeHistoryUnsupportedSourceAudit(t *testing.T) {
	reader := UnsupportedNativeEventHistoryReader{}
	for _, kind := range []HistorySourceKind{HistorySourceRunner, HistorySourceChild, HistorySourceTeam, HistorySourceWorkflow} {
		source := SourceRef{Kind: kind, Namespace: string(kind), ID: "source"}
		caps, err := reader.Capabilities(context.Background(), source)
		if err != nil || caps.Supported || caps.Durable || caps.UnsupportedReason == "" || caps.SequenceScope != "none" {
			t.Fatalf("audit %s: %+v %v", kind, caps, err)
		}
		if _, err = reader.ReadAfter(context.Background(), source, nil, 1); !errors.Is(err, ErrHistoryUnsupported) {
			t.Fatal(err)
		}
	}
	for _, err := range []error{ErrHistoryUnsupported, ErrCursorMalformed, ErrCursorForeign, ErrCursorExpired, ErrHistoryGap, ErrSourceUnavailable} {
		if !errors.Is(fmt.Errorf("wrapped: %w", err), err) {
			t.Fatal("lost typed error")
		}
	}
}

type historyUnsafeMarshaler struct{ calls *int }

func (m historyUnsafeMarshaler) MarshalJSON() ([]byte, error) {
	*m.calls++
	return []byte(`"unsafe"`), nil
}

func TestNativeHistorySnapshotRejectsExecutableSerialization(t *testing.T) {
	calls := 0
	p := NativeEventPage{Events: []NativeExecutionEvent{{Event: EventEnvelope{Payload: map[string]any{"body": historyUnsafeMarshaler{calls: &calls}}}}}}
	if _, err := p.Snapshot(); err == nil || calls != 0 {
		t.Fatalf("snapshot invoked custom serialization: calls=%d err=%v", calls, err)
	}
}

func TestNativeHistorySnapshotBoundsSerializedEventsAndIdentity(t *testing.T) {
	for _, event := range []NativeExecutionEvent{
		{Event: EventEnvelope{Payload: map[string]any{"escaped": strings.Repeat("\x00", 12000)}}},
		{History: &NativeHistoryIdentity{Source: SourceRef{Kind: HistorySourceMailbox, Namespace: "mailbox", ID: "source"}, Sequence: 1, EventID: strings.Repeat("e", MaxHistoryReferenceBytes+1)}},
		{History: &NativeHistoryIdentity{Source: SourceRef{Kind: HistorySourceMailbox, Namespace: "mailbox", ID: ""}, Sequence: 1, EventID: "event"}},
	} {
		if _, err := (NativeEventPage{Events: []NativeExecutionEvent{event}}).Snapshot(); !errors.Is(err, ErrHistoryInvalid) {
			t.Fatalf("accepted unbounded event/identity: %v", err)
		}
	}
}
