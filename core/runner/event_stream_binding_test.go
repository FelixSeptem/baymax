package runner

import (
	"runtime"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

func TestProjectRealtimeEventStreamBindingPreservesRealtimeOwnership(t *testing.T) {
	subscription := types.EventStreamSubscription{
		Version:        types.DurableEventStreamBindingVersionV1,
		SubscriptionID: "subscription-1",
		Source:         types.ProtocolSourceRealtime,
		SessionID:      "session-1",
		RunID:          "run-1",
		StartMode:      types.EventStreamStartAfterCursor,
		Cursor:         types.EventStreamCursor{Value: "cursor-1", Sequence: 1},
		DeliveryPolicy: types.EventStreamDeliveryPolicyReject,
		MaxBatchSize:   8,
	}
	event := types.RealtimeEventEnvelope{
		EventID:   "event-2",
		SessionID: "session-1",
		RunID:     "run-1",
		Seq:       2,
		Type:      types.RealtimeEventTypeDelta,
		TS:        time.Date(2026, time.August, 27, 10, 0, 0, 0, time.UTC),
		Payload:   map[string]any{"delta": "next"},
	}

	projected, err := ProjectRealtimeEventStreamBinding(subscription, types.EventStreamBindingOutcome{
		SubscriptionID:        subscription.SubscriptionID,
		Phase:                 types.EventStreamBindingPhaseLive,
		ReasonCode:            "realtime.binding.live",
		SourceOutcomeDeclared: true,
	}, []types.RealtimeEventEnvelope{event}, nil)
	if err != nil {
		t.Fatalf("ProjectRealtimeEventStreamBinding() error = %v", err)
	}
	if len(projected.Events) != 1 || projected.Events[0].EventID != "event-2" || projected.Outcome.LastSequence != 2 {
		t.Fatalf("projected = %#v", projected)
	}
}

func TestRealtimeEventStreamBindingIsPureAndDoesNotOwnControlPlaneState(t *testing.T) {
	subscription := types.EventStreamSubscription{
		Version: types.DurableEventStreamBindingVersionV1, SubscriptionID: "sub-pure",
		Source: types.ProtocolSourceRealtime, SessionID: "session-pure", RunID: "run-pure",
		StartMode: types.EventStreamStartLatest, DeliveryPolicy: types.EventStreamDeliveryPolicyUnknown, MaxBatchSize: 4,
	}
	event := streamBindingEvent("event-pure", "session-pure", "run-pure", 1)
	before := runtime.NumGoroutine()
	projected, err := ProjectRealtimeEventStreamBinding(subscription, types.EventStreamBindingOutcome{
		SubscriptionID: "sub-pure", Phase: types.EventStreamBindingPhaseLive,
		ReasonCode: "realtime.binding.live", SourceOutcomeDeclared: true,
	}, nil, []types.RealtimeEventEnvelope{event})
	if err != nil {
		t.Fatalf("projection error = %v", err)
	}
	if runtime.NumGoroutine() != before {
		t.Fatalf("binding projection must not create listener/connection goroutines")
	}
	if len(projected.Events) != 1 {
		t.Fatalf("projected events = %#v", projected.Events)
	}
	projected.Events[0].Payload["delta"] = "mutated"
	if event.Payload["delta"] == "mutated" {
		t.Fatalf("projection must clone source payload and avoid source mutation")
	}
}

func TestRealtimeRunAndStreamBindingProjectionParity(t *testing.T) {
	subscription := types.EventStreamSubscription{
		Version:        types.DurableEventStreamBindingVersionV1,
		SubscriptionID: "subscription-parity",
		Source:         types.ProtocolSourceRealtime,
		SessionID:      "session-parity",
		RunID:          "run-parity",
		StartMode:      types.EventStreamStartAfterCursor,
		Cursor:         types.EventStreamCursor{Value: "cursor-2", Sequence: 2},
		DeliveryPolicy: types.EventStreamDeliveryPolicyDropWithRecord,
		MaxBatchSize:   8,
	}
	history := []types.RealtimeEventEnvelope{
		streamBindingEvent("event-3", "session-parity", "run-parity", 3),
	}
	live := []types.RealtimeEventEnvelope{
		streamBindingEvent("event-3", "session-parity", "run-parity", 3),
		streamBindingEvent("event-4", "session-parity", "run-parity", 4),
	}
	outcome := types.EventStreamBindingOutcome{
		SubscriptionID:        subscription.SubscriptionID,
		Phase:                 types.EventStreamBindingPhaseLive,
		ReasonCode:            "realtime.binding.live",
		SourceOutcomeDeclared: true,
	}
	runProjection, err := ProjectRealtimeEventStreamBinding(subscription, outcome, history, live)
	if err != nil {
		t.Fatalf("Run projection error = %v", err)
	}
	streamProjection, err := ProjectRealtimeEventStreamBinding(subscription, outcome, history, live)
	if err != nil {
		t.Fatalf("Stream projection error = %v", err)
	}
	runEvents, err := types.MapEventStreamBindingToProtocol(runProjection)
	if err != nil {
		t.Fatalf("Run protocol mapping error = %v", err)
	}
	streamEvents, err := types.MapEventStreamBindingToProtocol(streamProjection)
	if err != nil {
		t.Fatalf("Stream protocol mapping error = %v", err)
	}
	if len(runEvents) != len(streamEvents) || len(runEvents) != 2 {
		t.Fatalf("Run/Stream event count mismatch run=%d stream=%d", len(runEvents), len(streamEvents))
	}
	for i := range runEvents {
		if runEvents[i].EventID != streamEvents[i].EventID || runEvents[i].Sequence != streamEvents[i].Sequence || runEvents[i].StreamBinding == nil || streamEvents[i].StreamBinding == nil || *runEvents[i].StreamBinding != *streamEvents[i].StreamBinding {
			t.Fatalf("Run/Stream event mismatch at %d: run=%#v stream=%#v", i, runEvents[i], streamEvents[i])
		}
	}
}

func streamBindingEvent(id, sessionID, runID string, sequence int64) types.RealtimeEventEnvelope {
	return types.RealtimeEventEnvelope{
		EventID:   id,
		SessionID: sessionID,
		RunID:     runID,
		Seq:       sequence,
		Type:      types.RealtimeEventTypeDelta,
		TS:        time.Date(2026, time.August, 28, 10, 0, 0, 0, time.UTC),
		Payload:   map[string]any{"delta": id},
	}
}
