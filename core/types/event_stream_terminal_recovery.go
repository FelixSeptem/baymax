package types

import (
	"fmt"
	"strings"
)

const (
	EventStreamTerminalRecoveryVersionV1          = "runtime_event_stream_terminal_recovery.v1"
	ProtocolReasonEventStreamTerminalDrift        = "stream_recovery_terminal_drift"
	ProtocolReasonEventStreamRetainedFactsDrift   = "stream_recovery_retained_facts_drift"
	ProtocolReasonEventStreamRunStreamParityDrift = "stream_recovery_run_stream_parity_drift"
)

// EventStreamRecoveryObserverState describes only a host observer lifecycle.
// It never changes the source-owned Run lifecycle or invokes retry/resume.
type EventStreamRecoveryObserverState string

const (
	EventStreamRecoveryObserverCatchingUp        EventStreamRecoveryObserverState = "catching_up"
	EventStreamRecoveryObserverLive              EventStreamRecoveryObserverState = "live"
	EventStreamRecoveryObserverDisconnected      EventStreamRecoveryObserverState = "disconnected"
	EventStreamRecoveryObserverStopped           EventStreamRecoveryObserverState = "stopped"
	EventStreamRecoveryObserverTerminalAvailable EventStreamRecoveryObserverState = "terminal_available"
	EventStreamRecoveryObserverExpired           EventStreamRecoveryObserverState = "expired"
	EventStreamRecoveryObserverGap               EventStreamRecoveryObserverState = "gap"
	EventStreamRecoveryObserverBackpressure      EventStreamRecoveryObserverState = "backpressure"
)

// EventStreamTerminalRecoveryObservation contains source-owned facts for a
// single bounded subscription. Terminal candidates are passed through the
// existing arbiter so recovery cannot replace a business terminal outcome.
type EventStreamTerminalRecoveryObservation struct {
	ObserverState      EventStreamRecoveryObserverState `json:"observer_state,omitempty"`
	TerminalCandidates []TerminalOutcome                `json:"terminal_candidates,omitempty"`
	RetainedToolCalls  []ToolCallSummary                `json:"retained_tool_calls,omitempty"`
}

// EventStreamTerminalRecovery is a pure normalized recovery projection. It
// does not store history, allocate queues, or mutate source runtime state.
type EventStreamTerminalRecovery struct {
	Version                  string                           `json:"version"`
	Subscription             EventStreamSubscription          `json:"subscription"`
	Binding                  EventStreamBindingOutcome        `json:"binding"`
	ObserverState            EventStreamRecoveryObserverState `json:"observer_state"`
	RetainedEvents           []RealtimeEventEnvelope          `json:"retained_events,omitempty"`
	RetainedToolCalls        []ToolCallSummary                `json:"retained_tool_calls,omitempty"`
	Terminal                 *TerminalOutcome                 `json:"terminal,omitempty"`
	TerminalConflictRecorded bool                             `json:"terminal_conflict_recorded,omitempty"`
}

// ProjectEventStreamTerminalRecovery combines an already validated binding
// projection with source-owned terminal candidates and retained facts.
func ProjectEventStreamTerminalRecovery(binding EventStreamBindingProjection, observation EventStreamTerminalRecoveryObservation) (EventStreamTerminalRecovery, error) {
	if err := binding.Subscription.Validate(); err != nil {
		return EventStreamTerminalRecovery{}, err
	}
	if err := binding.Outcome.Validate(binding.Subscription); err != nil {
		return EventStreamTerminalRecovery{}, err
	}
	state := observation.ObserverState
	if state == "" {
		state = eventStreamRecoveryStateFromBinding(binding.Outcome.Phase)
	}
	if !isValidEventStreamRecoveryObserverState(state) {
		return EventStreamTerminalRecovery{}, eventStreamTerminalRecoveryError("unsupported observer_state %q", state)
	}

	arbiter := NewTerminalOutcomeArbiter()
	conflict := false
	for _, candidate := range observation.TerminalCandidates {
		if strings.TrimSpace(candidate.RunID) != strings.TrimSpace(binding.Subscription.RunID) {
			return EventStreamTerminalRecovery{}, eventStreamTerminalRecoveryError("terminal run_id does not match subscription")
		}
		if candidate.SessionID != "" && strings.TrimSpace(candidate.SessionID) != strings.TrimSpace(binding.Subscription.SessionID) {
			return EventStreamTerminalRecovery{}, eventStreamTerminalRecoveryError("terminal session_id does not match subscription")
		}
		published, err := arbiter.Publish(candidate)
		if err != nil {
			return EventStreamTerminalRecovery{}, err
		}
		if published == TerminalPublishConflict {
			conflict = true
		}
	}

	recovery := EventStreamTerminalRecovery{
		Version:                  EventStreamTerminalRecoveryVersionV1,
		Subscription:             binding.Subscription,
		Binding:                  binding.Outcome,
		ObserverState:            state,
		RetainedEvents:           cloneRecoveryEvents(binding.Events),
		RetainedToolCalls:        append([]ToolCallSummary(nil), observation.RetainedToolCalls...),
		TerminalConflictRecorded: conflict,
	}
	if terminal, ok := arbiter.Terminal(); ok {
		cloned := terminal
		recovery.Terminal = &cloned
		recovery.ObserverState = EventStreamRecoveryObserverTerminalAvailable
	}
	return recovery, nil
}

func eventStreamRecoveryStateFromBinding(phase EventStreamBindingPhase) EventStreamRecoveryObserverState {
	switch phase {
	case EventStreamBindingPhaseCatchingUp:
		return EventStreamRecoveryObserverCatchingUp
	case EventStreamBindingPhaseLive, EventStreamBindingPhaseAccepted:
		return EventStreamRecoveryObserverLive
	case EventStreamBindingPhaseDisconnected:
		return EventStreamRecoveryObserverDisconnected
	case EventStreamBindingPhaseClosed:
		return EventStreamRecoveryObserverStopped
	case EventStreamBindingPhaseExpired:
		return EventStreamRecoveryObserverExpired
	case EventStreamBindingPhaseGap:
		return EventStreamRecoveryObserverGap
	case EventStreamBindingPhaseBackpressured:
		return EventStreamRecoveryObserverBackpressure
	default:
		return EventStreamRecoveryObserverDisconnected
	}
}

func isValidEventStreamRecoveryObserverState(state EventStreamRecoveryObserverState) bool {
	switch state {
	case EventStreamRecoveryObserverCatchingUp, EventStreamRecoveryObserverLive,
		EventStreamRecoveryObserverDisconnected, EventStreamRecoveryObserverStopped,
		EventStreamRecoveryObserverTerminalAvailable, EventStreamRecoveryObserverExpired,
		EventStreamRecoveryObserverGap, EventStreamRecoveryObserverBackpressure:
		return true
	default:
		return false
	}
}

func cloneRecoveryEvents(events []RealtimeEventEnvelope) []RealtimeEventEnvelope {
	cloned := make([]RealtimeEventEnvelope, len(events))
	for i := range events {
		cloned[i] = cloneEventStreamEnvelope(events[i])
	}
	return cloned
}

func eventStreamTerminalRecoveryError(format string, args ...any) error {
	return fmt.Errorf("%s: %s", ProtocolReasonEventStreamTerminalDrift, fmt.Sprintf(format, args...))
}
