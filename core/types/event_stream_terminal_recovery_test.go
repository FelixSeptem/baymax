package types

import (
	"strings"
	"testing"
	"time"
)

func TestProjectEventStreamTerminalRecoveryPreservesFactsAndFirstTerminal(t *testing.T) {
	subscription := EventStreamSubscription{
		Version: DurableEventStreamBindingVersionV1, SubscriptionID: "recovery-1",
		Source: ProtocolSourceRealtime, SessionID: "session-1", RunID: "run-1",
		StartMode: EventStreamStartAfterCursor, Cursor: EventStreamCursor{Value: "cursor-1", Sequence: 1},
		DeliveryPolicy: EventStreamDeliveryPolicyUnknown, MaxBatchSize: 8,
	}
	binding, err := ProjectEventStreamBinding(subscription, EventStreamBindingOutcome{
		SubscriptionID: subscription.SubscriptionID, Phase: EventStreamBindingPhaseLive,
		ReasonCode: "realtime.binding.live", LastSequence: 2, SourceOutcomeDeclared: true,
	}, []RealtimeEventEnvelope{recoveryEvent("event-2", 2)}, nil)
	if err != nil {
		t.Fatalf("ProjectEventStreamBinding() error = %v", err)
	}
	completed := TerminalOutcome{RunID: "run-1", SessionID: "session-1", State: RunStateCompleted, FailureFamily: FailureFamilyNone, Phase: ExecutionPhasePostStart}
	conflict := TerminalOutcome{RunID: "run-1", SessionID: "session-1", State: RunStateFailed, FailureFamily: FailureFamilyRuntimeFailed, Phase: ExecutionPhasePostStart}

	recovery, err := ProjectEventStreamTerminalRecovery(binding, EventStreamTerminalRecoveryObservation{
		ObserverState:      EventStreamRecoveryObserverDisconnected,
		TerminalCandidates: []TerminalOutcome{completed, completed, conflict},
		RetainedToolCalls:  []ToolCallSummary{{CallID: "call-1", Name: "search"}},
	})
	if err != nil {
		t.Fatalf("ProjectEventStreamTerminalRecovery() error = %v", err)
	}
	if recovery.ObserverState != EventStreamRecoveryObserverTerminalAvailable || recovery.Terminal == nil || *recovery.Terminal != completed {
		t.Fatalf("recovery terminal = %#v, want completed terminal_available", recovery)
	}
	if !recovery.TerminalConflictRecorded || len(recovery.RetainedEvents) != 1 || len(recovery.RetainedToolCalls) != 1 {
		t.Fatalf("recovery facts = %#v", recovery)
	}
}

func TestProjectEventStreamTerminalRecoveryRejectsCrossRunTerminal(t *testing.T) {
	subscription := EventStreamSubscription{
		Version: DurableEventStreamBindingVersionV1, SubscriptionID: "recovery-2",
		Source: ProtocolSourceRealtime, SessionID: "session-1", RunID: "run-1",
		StartMode: EventStreamStartLatest, DeliveryPolicy: EventStreamDeliveryPolicyUnknown, MaxBatchSize: 8,
	}
	binding, err := ProjectEventStreamBinding(subscription, EventStreamBindingOutcome{SubscriptionID: subscription.SubscriptionID, Phase: EventStreamBindingPhaseDisconnected, ReasonCode: "realtime.binding.disconnected", SourceOutcomeDeclared: true}, nil, nil)
	if err != nil {
		t.Fatalf("ProjectEventStreamBinding() error = %v", err)
	}
	_, err = ProjectEventStreamTerminalRecovery(binding, EventStreamTerminalRecoveryObservation{TerminalCandidates: []TerminalOutcome{{RunID: "other-run", State: RunStateCompleted, FailureFamily: FailureFamilyNone, Phase: ExecutionPhasePostStart}}})
	if err == nil || !strings.Contains(err.Error(), "terminal run_id") {
		t.Fatalf("error = %v, want cross-run terminal validation", err)
	}
}

func TestProjectEventStreamTerminalRecoveryObserverDisconnectAndStopAreIdempotent(t *testing.T) {
	subscription := EventStreamSubscription{
		Version: DurableEventStreamBindingVersionV1, SubscriptionID: "recovery-observer-idempotent",
		Source: ProtocolSourceRealtime, SessionID: "session-1", RunID: "run-1",
		StartMode: EventStreamStartLatest, DeliveryPolicy: EventStreamDeliveryPolicyUnknown, MaxBatchSize: 8,
	}
	for _, tc := range []struct {
		name   string
		phase  EventStreamBindingPhase
		state  EventStreamRecoveryObserverState
		reason string
	}{
		{name: "disconnect", phase: EventStreamBindingPhaseDisconnected, state: EventStreamRecoveryObserverDisconnected, reason: "realtime.binding.disconnected"},
		{name: "stop", phase: EventStreamBindingPhaseClosed, state: EventStreamRecoveryObserverStopped, reason: "realtime.binding.closed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			binding, err := ProjectEventStreamBinding(subscription, EventStreamBindingOutcome{
				SubscriptionID: subscription.SubscriptionID, Phase: tc.phase, ReasonCode: tc.reason, SourceOutcomeDeclared: true,
			}, nil, nil)
			if err != nil {
				t.Fatalf("ProjectEventStreamBinding() error = %v", err)
			}
			first, err := ProjectEventStreamTerminalRecovery(binding, EventStreamTerminalRecoveryObservation{})
			if err != nil {
				t.Fatalf("first ProjectEventStreamTerminalRecovery() error = %v", err)
			}
			second, err := ProjectEventStreamTerminalRecovery(binding, EventStreamTerminalRecoveryObservation{})
			if err != nil {
				t.Fatalf("second ProjectEventStreamTerminalRecovery() error = %v", err)
			}
			if first.ObserverState != tc.state || second.ObserverState != tc.state || first.Terminal != nil || second.Terminal != nil {
				t.Fatalf("observer recovery must be idempotent without a terminal: first=%#v second=%#v", first, second)
			}
		})
	}
}

func TestProjectEventStreamTerminalRecoveryNeverRestoresTerminalRunToWorking(t *testing.T) {
	subscription := EventStreamSubscription{
		Version: DurableEventStreamBindingVersionV1, SubscriptionID: "recovery-terminal-stable",
		Source: ProtocolSourceRealtime, SessionID: "session-1", RunID: "run-1",
		StartMode: EventStreamStartLatest, DeliveryPolicy: EventStreamDeliveryPolicyUnknown, MaxBatchSize: 8,
	}
	binding, err := ProjectEventStreamBinding(subscription, EventStreamBindingOutcome{
		SubscriptionID: subscription.SubscriptionID, Phase: EventStreamBindingPhaseClosed, ReasonCode: "realtime.binding.closed", SourceOutcomeDeclared: true,
	}, nil, nil)
	if err != nil {
		t.Fatalf("ProjectEventStreamBinding() error = %v", err)
	}
	completed := TerminalOutcome{RunID: subscription.RunID, SessionID: subscription.SessionID, State: RunStateCompleted, FailureFamily: FailureFamilyNone, Phase: ExecutionPhasePostStart}
	for _, state := range []EventStreamRecoveryObserverState{EventStreamRecoveryObserverDisconnected, EventStreamRecoveryObserverStopped} {
		recovery, err := ProjectEventStreamTerminalRecovery(binding, EventStreamTerminalRecoveryObservation{ObserverState: state, TerminalCandidates: []TerminalOutcome{completed}})
		if err != nil {
			t.Fatalf("ProjectEventStreamTerminalRecovery(%q) error = %v", state, err)
		}
		if recovery.ObserverState != EventStreamRecoveryObserverTerminalAvailable || recovery.Terminal == nil || recovery.Terminal.State != RunStateCompleted || recovery.Terminal.State == RunStateWorking {
			t.Fatalf("terminal Run must remain completed after observer %q: %#v", state, recovery)
		}
	}
}

func recoveryEvent(id string, seq int64) RealtimeEventEnvelope {
	return RealtimeEventEnvelope{EventID: id, SessionID: "session-1", RunID: "run-1", Seq: seq, Type: RealtimeEventTypeDelta, TS: time.Unix(seq, 0).UTC(), Payload: map[string]any{"delta": id}}
}
