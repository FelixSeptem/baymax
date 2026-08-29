package types

import (
	"strings"
	"testing"
	"time"
)

func TestEventStreamSubscriptionValidatesBoundedLatestAndCursorStarts(t *testing.T) {
	latest := EventStreamSubscription{
		Version:        DurableEventStreamBindingVersionV1,
		SubscriptionID: "subscription-latest",
		Source:         ProtocolSourceRealtime,
		SessionID:      "session-1",
		RunID:          "run-1",
		StartMode:      EventStreamStartLatest,
		DeliveryPolicy: EventStreamDeliveryPolicyReject,
		MaxBatchSize:   32,
	}
	if err := latest.Validate(); err != nil {
		t.Fatalf("latest.Validate() error = %v", err)
	}

	afterCursor := latest
	afterCursor.SubscriptionID = "subscription-cursor"
	afterCursor.StartMode = EventStreamStartAfterCursor
	afterCursor.Cursor = EventStreamCursor{Value: "cursor-2", Sequence: 2}
	if err := afterCursor.Validate(); err != nil {
		t.Fatalf("afterCursor.Validate() error = %v", err)
	}

	invalid := afterCursor
	invalid.Cursor = EventStreamCursor{}
	if err := invalid.Validate(); err == nil || !strings.Contains(err.Error(), ProtocolReasonEventStreamInvalidSubscription) {
		t.Fatalf("invalid cursor error = %v, want %q", err, ProtocolReasonEventStreamInvalidSubscription)
	}
}

func TestProjectEventStreamBindingDeduplicatesHandoffOverlap(t *testing.T) {
	subscription := EventStreamSubscription{
		Version:        DurableEventStreamBindingVersionV1,
		SubscriptionID: "subscription-1",
		Source:         ProtocolSourceRealtime,
		SessionID:      "session-1",
		RunID:          "run-1",
		StartMode:      EventStreamStartAfterCursor,
		Cursor:         EventStreamCursor{Value: "cursor-2", Sequence: 2},
		DeliveryPolicy: EventStreamDeliveryPolicyReject,
		MaxBatchSize:   8,
	}
	history := []RealtimeEventEnvelope{eventStreamTestEvent("event-3", 3)}
	live := []RealtimeEventEnvelope{
		eventStreamTestEvent("event-3", 3),
		eventStreamTestEvent("event-4", 4),
	}
	projected, err := ProjectEventStreamBinding(subscription, EventStreamBindingOutcome{
		SubscriptionID:        subscription.SubscriptionID,
		Phase:                 EventStreamBindingPhaseLive,
		ReasonCode:            "realtime.binding.live",
		SourceOutcomeDeclared: true,
	}, history, live)
	if err != nil {
		t.Fatalf("ProjectEventStreamBinding() error = %v", err)
	}
	if got := len(projected.Events); got != 2 {
		t.Fatalf("normalized events = %d, want 2: %#v", got, projected.Events)
	}
	if projected.Events[0].EventID != "event-3" || projected.Events[1].EventID != "event-4" {
		t.Fatalf("normalized event ids = %#v", projected.Events)
	}
	if projected.Outcome.LastSequence != 4 || projected.Outcome.Phase != EventStreamBindingPhaseLive {
		t.Fatalf("outcome = %#v", projected.Outcome)
	}
	if history[0].EventID != "event-3" || live[0].EventID != "event-3" {
		t.Fatalf("projection mutated source event slices: history=%#v live=%#v", history, live)
	}
}

func TestProjectEventStreamBindingRejectsHandoffGap(t *testing.T) {
	subscription := eventStreamTestSubscription(EventStreamDeliveryPolicyReject)
	_, err := ProjectEventStreamBinding(subscription, EventStreamBindingOutcome{
		SubscriptionID:        subscription.SubscriptionID,
		Phase:                 EventStreamBindingPhaseLive,
		ReasonCode:            "realtime.binding.live",
		SourceOutcomeDeclared: true,
	}, []RealtimeEventEnvelope{eventStreamTestEvent("event-3", 3)}, []RealtimeEventEnvelope{eventStreamTestEvent("event-5", 5)})
	if err == nil || !strings.Contains(err.Error(), ProtocolReasonEventStreamSequenceGap) {
		t.Fatalf("handoff gap error = %v, want %q", err, ProtocolReasonEventStreamSequenceGap)
	}
}

func TestEventStreamBindingOutcomePreservesExpiredAndRejectsIncompatibleBackpressure(t *testing.T) {
	subscription := eventStreamTestSubscription(EventStreamDeliveryPolicyReject)
	expired, err := ProjectEventStreamBinding(subscription, EventStreamBindingOutcome{
		SubscriptionID:        subscription.SubscriptionID,
		Phase:                 EventStreamBindingPhaseExpired,
		ReasonCode:            "realtime.cursor.expired",
		SourceOutcomeDeclared: true,
	}, nil, nil)
	if err != nil {
		t.Fatalf("expired projection error = %v", err)
	}
	if expired.Outcome.Phase != EventStreamBindingPhaseExpired || len(expired.Events) != 0 {
		t.Fatalf("expired projection = %#v", expired)
	}

	_, err = ProjectEventStreamBinding(subscription, EventStreamBindingOutcome{
		SubscriptionID:        subscription.SubscriptionID,
		Phase:                 EventStreamBindingPhaseBackpressured,
		ReasonCode:            "realtime.consumer.slow",
		SourceOutcomeDeclared: true,
	}, nil, nil)
	if err == nil || !strings.Contains(err.Error(), ProtocolReasonEventStreamIncompatibleDeliveryOutcome) {
		t.Fatalf("backpressure error = %v, want %q", err, ProtocolReasonEventStreamIncompatibleDeliveryOutcome)
	}
}

func TestMapEventStreamBindingToProtocolPreservesExistingCorrelation(t *testing.T) {
	subscription := eventStreamTestSubscription(EventStreamDeliveryPolicyReject)
	projection, err := ProjectEventStreamBinding(subscription, EventStreamBindingOutcome{
		SubscriptionID:        subscription.SubscriptionID,
		Phase:                 EventStreamBindingPhaseLive,
		ReasonCode:            "realtime.binding.live",
		SourceOutcomeDeclared: true,
	}, []RealtimeEventEnvelope{eventStreamTestEvent("event-3", 3)}, nil)
	if err != nil {
		t.Fatalf("ProjectEventStreamBinding() error = %v", err)
	}

	events, err := MapEventStreamBindingToProtocol(projection)
	if err != nil {
		t.Fatalf("MapEventStreamBindingToProtocol() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("mapped events = %#v, want exactly one", events)
	}
	event := events[0]
	if event.EventID != "event-3" || event.RunID != "run-1" || event.SessionID != "session-1" || event.Sequence != 3 {
		t.Fatalf("event correlation = %#v", event)
	}
	if event.StreamBinding == nil || event.StreamBinding.SubscriptionID != subscription.SubscriptionID || event.StreamBinding.Phase != EventStreamBindingPhaseLive || event.StreamBinding.CursorMode != EventStreamStartAfterCursor || event.StreamBinding.SequenceBoundary != 3 {
		t.Fatalf("binding projection = %#v", event.StreamBinding)
	}
}

func eventStreamTestSubscription(policy EventStreamDeliveryPolicy) EventStreamSubscription {
	return EventStreamSubscription{
		Version:        DurableEventStreamBindingVersionV1,
		SubscriptionID: "subscription-1",
		Source:         ProtocolSourceRealtime,
		SessionID:      "session-1",
		RunID:          "run-1",
		StartMode:      EventStreamStartAfterCursor,
		Cursor:         EventStreamCursor{Value: "cursor-2", Sequence: 2},
		DeliveryPolicy: policy,
		MaxBatchSize:   8,
	}
}

func eventStreamTestEvent(id string, sequence int64) RealtimeEventEnvelope {
	return RealtimeEventEnvelope{
		EventID:   id,
		SessionID: "session-1",
		RunID:     "run-1",
		Seq:       sequence,
		Type:      RealtimeEventTypeDelta,
		TS:        time.Date(2026, time.August, 27, 9, 0, 0, 0, time.UTC),
		Payload:   map[string]any{"delta": id},
	}
}
