package host

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

type factCollector struct {
	mu     sync.Mutex
	events []types.Event
}

func (c *factCollector) OnEvent(_ context.Context, ev types.Event) {
	c.mu.Lock()
	c.events = append(c.events, ev)
	c.mu.Unlock()
}

func (c *factCollector) snapshot() []types.Event {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]types.Event(nil), c.events...)
}

func TestCoordinatorEmitsBoundedAdmissionAndSourceFacts(t *testing.T) {
	collector := &factCollector{}
	control := &contractControl{}
	coord := New(nil, WithEventSink(collector), WithSourceControl(control))
	conn := coord.Connect(func(any) error { return nil })
	cmd := validHostCommand(types.HostCommandKindAction)
	cmd.Payload = map[string]any{"action": "cancel", "cursor": strings.Repeat("secret", 1000), "raw_payload": strings.Repeat("x", 1000)}
	if _, err := conn.HandleCommand(context.Background(), cmd); err != nil {
		t.Fatal(err)
	}
	events := collector.snapshot()
	if !hasAdmission(events, types.EventTypeHostAdmission, "accepted") {
		t.Fatalf("missing admission fact: %#v", events)
	}
	if !hasFact(events, types.EventTypeHostSourceControl, "accepted") {
		t.Fatalf("missing source-control fact: %#v", events)
	}
	for _, ev := range events {
		if strings.Contains(stringifyPayload(ev.Payload), "secret") || strings.Contains(stringifyPayload(ev.Payload), "raw_payload") {
			t.Fatalf("unbounded/raw host payload leaked: %#v", ev.Payload)
		}
	}
}

func TestCoordinatorEmitsCorrelationAndPendingCloseFacts(t *testing.T) {
	collector := &factCollector{}
	conn := New(nil, WithEventSink(collector), WithDeliveryTimeout(20*time.Millisecond)).Connect(func(any) error { return nil })
	if _, err := conn.registerPending("pending-1", "session-1", "run-1"); err != nil {
		t.Fatal(err)
	}
	if err := conn.Respond(types.HostResponseEnvelope{Version: types.HostProtocolVersionV1, MessageID: "wrong", Kind: types.HostEnvelopeKindHostResponse, Time: time.Now().UTC(), RequestID: "pending-1", SessionID: "other-session", RunID: "other-run", Accepted: true}); !errors.Is(err, ErrPendingCorrelation) {
		t.Fatalf("correlation error = %v", err)
	}
	conn.Close()
	events := collector.snapshot()
	if !hasFact(events, types.EventTypeHostCorrelation, "mismatch") {
		t.Fatalf("missing correlation fact: %#v", events)
	}
	if !hasFact(events, types.EventTypeHostPendingClose, "disconnected") {
		t.Fatalf("missing pending-close fact: %#v", events)
	}
}

func TestCoordinatorEmitsDeliveryFailureFactWithoutRawFrame(t *testing.T) {
	collector := &factCollector{}
	conn := New(nil, WithEventSink(collector), WithDeliveryTimeout(10*time.Millisecond)).Connect(func(any) error { return errors.New("writer failed") })
	if _, err := conn.HandleCommand(context.Background(), validHostCommand(types.HostCommandKindRunGet)); err == nil {
		t.Fatal("expected delivery failure")
	}
	events := collector.snapshot()
	if !hasFact(events, types.EventTypeHostDelivery, "failed") {
		t.Fatalf("missing delivery fact: %#v", events)
	}
}

func TestProjectionDropWithRecordRecordsLowPriorityQueueDrop(t *testing.T) {
	collector := &factCollector{}
	writer := &blockingFrameWriter{started: make(chan struct{}), exited: make(chan struct{})}
	conn := New(nil,
		WithEventSink(collector),
		WithDeliveryQueueSize(1),
		WithDeliveryTimeout(250*time.Millisecond),
	).ConnectFrameWriter(writer)
	projection := recordedDropProjection(types.RealtimeEventTypeDelta, types.RealtimeEventTypeDelta, types.RealtimeEventTypeDelta)
	cmd := validHostCommand(types.HostCommandKindEventsSubscribe)
	done := make(chan struct{})
	go func() {
		conn.emitProjection(cmd, projection)
		close(done)
	}()
	select {
	case <-writer.started:
	case <-time.After(time.Second):
		t.Fatal("projection writer did not start")
	}
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("source-authorized low-priority projection blocked on transport")
	}
	if conn.isClosed() {
		t.Fatal("recorded low-priority drop closed the connection")
	}

	events := collector.snapshot()
	var dropped *types.Event
	for i := range events {
		if events[i].Type == types.EventTypeHostDelivery && events[i].Payload["fact"] == "dropped" {
			dropped = &events[i]
			break
		}
	}
	if dropped == nil {
		t.Fatalf("missing recorded low-priority delivery drop: %#v", events)
	}
	if dropped.Payload["delivery_status"] != "dropped" || dropped.Payload["reason_code"] != "host.delivery.source_policy_drop_with_record" {
		t.Fatalf("drop fact = %#v", dropped.Payload)
	}
	conn.Close()
	select {
	case <-writer.exited:
	case <-time.After(time.Second):
		t.Fatal("blocked writer did not exit after connection close")
	}
}

func TestProjectionRecordedDropEligibilityRequiresSourceDeltaAndSink(t *testing.T) {
	conn := &Connection{coord: &Coordinator{sink: &factCollector{}}}
	eligible := recordedDropProjection(types.RealtimeEventTypeDelta)
	if !conn.projectionAllowsRecordedDrop(eligible, 0) {
		t.Fatal("source-declared delta with a standard sink should be eligible")
	}

	notDeclared := recordedDropProjection(types.RealtimeEventTypeDelta)
	notDeclared.Outcome.SourceOutcomeDeclared = false
	if conn.projectionAllowsRecordedDrop(notDeclared, 0) {
		t.Fatal("client-requested policy without source declaration authorized a drop")
	}

	rejectPolicy := recordedDropProjection(types.RealtimeEventTypeDelta)
	rejectPolicy.Subscription.DeliveryPolicy = types.EventStreamDeliveryPolicyReject
	if conn.projectionAllowsRecordedDrop(rejectPolicy, 0) {
		t.Fatal("reject policy authorized a drop")
	}

	for _, eventType := range []types.RealtimeEventType{
		types.RealtimeEventTypeRequest,
		types.RealtimeEventTypeInterrupt,
		types.RealtimeEventTypeResume,
		types.RealtimeEventTypeAck,
		types.RealtimeEventTypeError,
		types.RealtimeEventTypeComplete,
	} {
		projection := recordedDropProjection(eventType)
		if conn.projectionAllowsRecordedDrop(projection, 0) {
			t.Fatalf("non-delta event %q authorized a drop", eventType)
		}
	}

	withoutSink := &Connection{coord: &Coordinator{}}
	if withoutSink.projectionAllowsRecordedDrop(eligible, 0) {
		t.Fatal("drop without a standard event sink would be silent")
	}
	if conn.projectionAllowsRecordedDrop(eligible, len(eligible.Events)) {
		t.Fatal("out-of-range source event authorized a drop")
	}
}

func recordedDropProjection(eventTypes ...types.RealtimeEventType) types.EventStreamBindingProjection {
	when := time.Date(2026, time.September, 12, 12, 0, 0, 0, time.UTC)
	events := make([]types.RealtimeEventEnvelope, 0, len(eventTypes))
	for i, eventType := range eventTypes {
		sequence := int64(i + 1)
		events = append(events, types.RealtimeEventEnvelope{
			EventID: fmt.Sprintf("event-%d", sequence), SessionID: "session-1", RunID: "run-1",
			Seq: sequence, Type: eventType, TS: when.Add(time.Duration(i) * time.Millisecond),
			Payload: map[string]any{"value": sequence},
		})
	}
	maxBatchSize := len(events)
	if maxBatchSize == 0 {
		maxBatchSize = 1
	}
	return types.EventStreamBindingProjection{
		Subscription: types.EventStreamSubscription{
			Version: types.DurableEventStreamBindingVersionV1, SubscriptionID: "subscription-drop-record",
			Source: types.ProtocolSourceRealtime, SessionID: "session-1", RunID: "run-1",
			StartMode: types.EventStreamStartLatest, DeliveryPolicy: types.EventStreamDeliveryPolicyDropWithRecord,
			MaxBatchSize: maxBatchSize,
		},
		Outcome: types.EventStreamBindingOutcome{
			SubscriptionID: "subscription-drop-record", Phase: types.EventStreamBindingPhaseLive,
			ReasonCode: "realtime.binding.live", LastSequence: int64(len(events)), SourceOutcomeDeclared: true,
		},
		Events: events,
	}
}

func hasFact(events []types.Event, eventType, fact string) bool {
	for _, ev := range events {
		if ev.Type == eventType && ev.Payload["fact"] == fact {
			return true
		}
	}
	return false
}

func hasAdmission(events []types.Event, eventType, status string) bool {
	for _, ev := range events {
		if ev.Type == eventType && ev.Payload["admission_status"] == status {
			return true
		}
	}
	return false
}

func stringifyPayload(payload map[string]any) string {
	return fmt.Sprintf("%v", payload)
}
