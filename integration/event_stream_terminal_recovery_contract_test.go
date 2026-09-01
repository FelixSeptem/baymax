package integration

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
	obsevent "github.com/FelixSeptem/baymax/observability/event"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
	"github.com/FelixSeptem/baymax/tool/diagnosticsreplay"
)

func TestEventStreamTerminalRecoveryRunStreamParity(t *testing.T) {
	subscription := types.EventStreamSubscription{Version: types.DurableEventStreamBindingVersionV1, SubscriptionID: "integration-recovery", Source: types.ProtocolSourceRealtime, SessionID: "session-1", RunID: "run-1", StartMode: types.EventStreamStartAfterCursor, Cursor: types.EventStreamCursor{Value: "cursor", Sequence: 1}, DeliveryPolicy: types.EventStreamDeliveryPolicyUnknown, MaxBatchSize: 4}
	outcome := types.EventStreamBindingOutcome{SubscriptionID: subscription.SubscriptionID, Phase: types.EventStreamBindingPhaseLive, ReasonCode: "realtime.binding.live", LastSequence: 2, SourceOutcomeDeclared: true}
	history := []types.RealtimeEventEnvelope{{EventID: "event-2", SessionID: "session-1", RunID: "run-1", Seq: 2, Type: types.RealtimeEventTypeDelta, TS: time.Unix(2, 0).UTC(), Payload: map[string]any{"delta": "partial"}}}
	observation := types.EventStreamTerminalRecoveryObservation{TerminalCandidates: []types.TerminalOutcome{{RunID: "run-1", SessionID: "session-1", State: types.RunStateFailed, FailureFamily: types.FailureFamilyTimedOut, Phase: types.ExecutionPhasePostStart}}, RetainedToolCalls: []types.ToolCallSummary{{CallID: "call-1", Name: "search"}}}

	run, err := runner.ProjectRealtimeEventStreamTerminalRecovery(subscription, outcome, history, nil, observation)
	if err != nil {
		t.Fatalf("run recovery error = %v", err)
	}
	stream, err := runner.ProjectRealtimeEventStreamTerminalRecovery(subscription, outcome, history, nil, observation)
	if err != nil {
		t.Fatalf("stream recovery error = %v", err)
	}
	if run.ObserverState != stream.ObserverState || run.Terminal == nil || stream.Terminal == nil || *run.Terminal != *stream.Terminal || len(run.RetainedEvents) != len(stream.RetainedEvents) || len(run.RetainedToolCalls) != len(stream.RetainedToolCalls) {
		t.Fatalf("run/stream recovery parity drift: run=%#v stream=%#v", run, stream)
	}
}

func TestEventStreamTerminalRecoveryProjectionDiagnosticsAndReplayAgree(t *testing.T) {
	subscription := types.EventStreamSubscription{Version: types.DurableEventStreamBindingVersionV1, SubscriptionID: "integration-recovery-diagnostics", Source: types.ProtocolSourceRealtime, SessionID: "session-1", RunID: "run-1", StartMode: types.EventStreamStartAfterCursor, Cursor: types.EventStreamCursor{Value: "cursor", Sequence: 1}, DeliveryPolicy: types.EventStreamDeliveryPolicyUnknown, MaxBatchSize: 4}
	outcome := types.EventStreamBindingOutcome{SubscriptionID: subscription.SubscriptionID, Phase: types.EventStreamBindingPhaseLive, ReasonCode: "realtime.binding.live", LastSequence: 2, SourceOutcomeDeclared: true}
	history := []types.RealtimeEventEnvelope{{EventID: "event-2", SessionID: subscription.SessionID, RunID: subscription.RunID, Seq: 2, Type: types.RealtimeEventTypeDelta, TS: time.Unix(2, 0).UTC(), Payload: map[string]any{"delta": "partial"}}}
	recovery, err := runner.ProjectRealtimeEventStreamTerminalRecovery(subscription, outcome, history, nil, types.EventStreamTerminalRecoveryObservation{
		TerminalCandidates: []types.TerminalOutcome{{RunID: subscription.RunID, SessionID: subscription.SessionID, State: types.RunStateFailed, FailureFamily: types.FailureFamilyTimedOut, Phase: types.ExecutionPhasePostStart}},
	})
	if err != nil {
		t.Fatalf("ProjectRealtimeEventStreamTerminalRecovery() error = %v", err)
	}
	if recovery.Terminal == nil || recovery.Terminal.State != types.RunStateFailed || recovery.Terminal.FailureFamily != types.FailureFamilyTimedOut {
		t.Fatalf("terminal projection = %#v", recovery)
	}

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX_EVENT_STREAM_RECOVERY_CONTRACT"})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer func() { _ = mgr.Close() }()
	recorder := obsevent.NewRuntimeRecorder(mgr)
	recorder.OnEvent(context.Background(), types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    "run.finished",
		RunID:   subscription.RunID,
		Time:    time.Now().UTC(),
		Payload: map[string]any{
			"status":                                     "failed",
			"stream_recovery_phase":                      string(recovery.ObserverState),
			"stream_recovery_reason":                     outcome.ReasonCode,
			"stream_recovery_terminal_state":             string(recovery.Terminal.State),
			"stream_recovery_terminal_failure_family":    string(recovery.Terminal.FailureFamily),
			"stream_recovery_retained_event_total":       len(recovery.RetainedEvents),
			"stream_recovery_retained_tool_call_total":   len(recovery.RetainedToolCalls),
			"stream_recovery_terminal_conflict_recorded": recovery.TerminalConflictRecorded,
		},
	})
	runs := mgr.RecentRuns(1)
	if len(runs) != 1 {
		t.Fatalf("RecentRuns() = %#v, want one record", runs)
	}
	record := runs[0]
	if record.StreamRecoveryPhase != string(recovery.ObserverState) || record.StreamRecoveryTerminalState != string(recovery.Terminal.State) || record.StreamRecoveryTerminalFailureFamily != string(recovery.Terminal.FailureFamily) || record.StreamRecoveryRetainedEventTotal != len(recovery.RetainedEvents) || record.StreamRecoveryRetainedToolCallTotal != len(recovery.RetainedToolCalls) {
		t.Fatalf("diagnostics recovery drift: %#v", record)
	}

	raw, err := os.ReadFile(filepath.Join("..", "tool", "diagnosticsreplay", "testdata", "event_stream_terminal_recovery.json"))
	if err != nil {
		t.Fatalf("ReadFile(canonical fixture) error = %v", err)
	}
	cases, err := diagnosticsreplay.EvaluateEventStreamTerminalRecoveryFixtureJSON(raw)
	if err != nil {
		t.Fatalf("EvaluateEventStreamTerminalRecoveryFixtureJSON() error = %v", err)
	}
	for _, tc := range cases {
		if tc.Name != "timed_out_terminal" {
			continue
		}
		if tc.Expected.ObserverState != record.StreamRecoveryPhase || tc.Expected.TerminalState != record.StreamRecoveryTerminalState || tc.Expected.TerminalFailureFamily != record.StreamRecoveryTerminalFailureFamily || tc.Expected.RetainedEventTotal != record.StreamRecoveryRetainedEventTotal || tc.Expected.RetainedToolCallTotal != record.StreamRecoveryRetainedToolCallTotal {
			t.Fatalf("replay recovery drift: expected=%#v diagnostics=%#v", tc.Expected, record)
		}
		return
	}
	t.Fatal("canonical fixture missing timed_out_terminal case")
}

func TestEventStreamTerminalRecoveryRejectsMissingRealtimeSourceCapability(t *testing.T) {
	subscription := types.EventStreamSubscription{Version: types.DurableEventStreamBindingVersionV1, SubscriptionID: "integration-recovery-unsupported-source", Source: types.ProtocolSourceRunner, SessionID: "session-1", RunID: "run-1", StartMode: types.EventStreamStartLatest, DeliveryPolicy: types.EventStreamDeliveryPolicyUnknown, MaxBatchSize: 4}
	outcome := types.EventStreamBindingOutcome{SubscriptionID: subscription.SubscriptionID, Phase: types.EventStreamBindingPhaseLive, ReasonCode: "runner.binding.live", SourceOutcomeDeclared: true}
	_, err := runner.ProjectRealtimeEventStreamTerminalRecovery(subscription, outcome, nil, nil, types.EventStreamTerminalRecoveryObservation{})
	if err == nil || !strings.Contains(err.Error(), types.ProtocolReasonEventStreamInvalidSubscription) {
		t.Fatalf("missing Realtime source capability error = %v", err)
	}
}
