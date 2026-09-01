package runner

import (
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

func TestProjectRealtimeEventStreamTerminalRecoveryPreservesRunStreamProjection(t *testing.T) {
	subscription := types.EventStreamSubscription{Version: types.DurableEventStreamBindingVersionV1, SubscriptionID: "runner-recovery", Source: types.ProtocolSourceRealtime, SessionID: "session-1", RunID: "run-1", StartMode: types.EventStreamStartLatest, DeliveryPolicy: types.EventStreamDeliveryPolicyUnknown, MaxBatchSize: 4}
	outcome := types.EventStreamBindingOutcome{SubscriptionID: "runner-recovery", Phase: types.EventStreamBindingPhaseLive, ReasonCode: "realtime.binding.live", LastSequence: 1, SourceOutcomeDeclared: true}
	events := []types.RealtimeEventEnvelope{{EventID: "event-1", SessionID: "session-1", RunID: "run-1", Seq: 1, Type: types.RealtimeEventTypeDelta, TS: time.Unix(1, 0).UTC(), Payload: map[string]any{"delta": "partial"}}}
	terminal := types.TerminalOutcome{RunID: "run-1", SessionID: "session-1", State: types.RunStateCompleted, FailureFamily: types.FailureFamilyNone, Phase: types.ExecutionPhasePostStart}

	got, err := ProjectRealtimeEventStreamTerminalRecovery(subscription, outcome, events, nil, types.EventStreamTerminalRecoveryObservation{TerminalCandidates: []types.TerminalOutcome{terminal}})
	if err != nil {
		t.Fatalf("ProjectRealtimeEventStreamTerminalRecovery() error = %v", err)
	}
	if got.Terminal == nil || got.Terminal.State != types.RunStateCompleted || len(got.RetainedEvents) != 1 {
		t.Fatalf("recovery = %#v", got)
	}
}

func TestProjectRealtimeEventStreamTerminalRecoveryRunStreamParityAcrossTerminalOutcomes(t *testing.T) {
	subscription := types.EventStreamSubscription{Version: types.DurableEventStreamBindingVersionV1, SubscriptionID: "runner-recovery-parity", Source: types.ProtocolSourceRealtime, SessionID: "session-1", RunID: "run-1", StartMode: types.EventStreamStartAfterCursor, Cursor: types.EventStreamCursor{Value: "cursor-1", Sequence: 1}, DeliveryPolicy: types.EventStreamDeliveryPolicyUnknown, MaxBatchSize: 4}
	partial := []types.RealtimeEventEnvelope{{EventID: "event-2", SessionID: subscription.SessionID, RunID: subscription.RunID, Seq: 2, Type: types.RealtimeEventTypeDelta, TS: time.Unix(2, 0).UTC(), Payload: map[string]any{"delta": "partial"}}}
	for _, tc := range []struct {
		name         string
		outcome      types.EventStreamBindingOutcome
		observation  types.EventStreamTerminalRecoveryObservation
		wantState    types.EventStreamRecoveryObserverState
		wantTerminal types.RunState
	}{
		{name: "reconnect", outcome: types.EventStreamBindingOutcome{SubscriptionID: subscription.SubscriptionID, Phase: types.EventStreamBindingPhaseLive, ReasonCode: "realtime.binding.live", LastSequence: 2, SourceOutcomeDeclared: true}, wantState: types.EventStreamRecoveryObserverLive},
		{name: "cancellation", outcome: types.EventStreamBindingOutcome{SubscriptionID: subscription.SubscriptionID, Phase: types.EventStreamBindingPhaseLive, ReasonCode: "realtime.binding.live", LastSequence: 2, SourceOutcomeDeclared: true}, observation: types.EventStreamTerminalRecoveryObservation{TerminalCandidates: []types.TerminalOutcome{{RunID: subscription.RunID, SessionID: subscription.SessionID, State: types.RunStateCanceled, FailureFamily: types.FailureFamilyCanceled, Phase: types.ExecutionPhasePostStart}}}, wantState: types.EventStreamRecoveryObserverTerminalAvailable, wantTerminal: types.RunStateCanceled},
		{name: "timeout", outcome: types.EventStreamBindingOutcome{SubscriptionID: subscription.SubscriptionID, Phase: types.EventStreamBindingPhaseLive, ReasonCode: "realtime.binding.live", LastSequence: 2, SourceOutcomeDeclared: true}, observation: types.EventStreamTerminalRecoveryObservation{TerminalCandidates: []types.TerminalOutcome{{RunID: subscription.RunID, SessionID: subscription.SessionID, State: types.RunStateFailed, FailureFamily: types.FailureFamilyTimedOut, Phase: types.ExecutionPhasePostStart}}}, wantState: types.EventStreamRecoveryObserverTerminalAvailable, wantTerminal: types.RunStateFailed},
		{name: "provider failure after partial output", outcome: types.EventStreamBindingOutcome{SubscriptionID: subscription.SubscriptionID, Phase: types.EventStreamBindingPhaseLive, ReasonCode: "realtime.binding.live", LastSequence: 2, SourceOutcomeDeclared: true}, observation: types.EventStreamTerminalRecoveryObservation{TerminalCandidates: []types.TerminalOutcome{{RunID: subscription.RunID, SessionID: subscription.SessionID, State: types.RunStateFailed, FailureFamily: types.FailureFamilyRuntimeFailed, Phase: types.ExecutionPhasePostStart}}, RetainedToolCalls: []types.ToolCallSummary{{CallID: "call-1", Name: "search"}}}, wantState: types.EventStreamRecoveryObserverTerminalAvailable, wantTerminal: types.RunStateFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			run, err := ProjectRealtimeEventStreamTerminalRecovery(subscription, tc.outcome, partial, nil, tc.observation)
			if err != nil {
				t.Fatalf("Run projection error = %v", err)
			}
			stream, err := ProjectRealtimeEventStreamTerminalRecovery(subscription, tc.outcome, partial, nil, tc.observation)
			if err != nil {
				t.Fatalf("Stream projection error = %v", err)
			}
			if run.ObserverState != stream.ObserverState || run.ObserverState != tc.wantState || len(run.RetainedEvents) != len(stream.RetainedEvents) || len(run.RetainedToolCalls) != len(stream.RetainedToolCalls) {
				t.Fatalf("Run/Stream recovery parity drift: run=%#v stream=%#v", run, stream)
			}
			if tc.wantTerminal == "" {
				if run.Terminal != nil || stream.Terminal != nil {
					t.Fatalf("unexpected terminal: run=%#v stream=%#v", run, stream)
				}
				return
			}
			if run.Terminal == nil || stream.Terminal == nil || run.Terminal.State != tc.wantTerminal || stream.Terminal.State != tc.wantTerminal {
				t.Fatalf("terminal parity drift: run=%#v stream=%#v", run, stream)
			}
		})
	}
}
