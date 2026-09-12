package host

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

type runtimeInputControl struct {
	called bool
	status types.HostAdmissionStatus
	reason string
}

func (c *runtimeInputControl) ExecuteAction(context.Context, types.HostCommandEnvelope, types.ProtocolAction) (types.HostAdmissionStatus, string, error) {
	return types.HostAdmissionStatusRejected, types.HostReasonUnknownKind, nil
}

func (c *runtimeInputControl) IngestRealtime(context.Context, types.HostCommandEnvelope, types.RealtimeEventEnvelope) (types.HostAdmissionStatus, string, error) {
	return types.HostAdmissionStatusRejected, types.HostReasonUnknownKind, nil
}

func (c *runtimeInputControl) AdmitHostRuntimeInput(_ context.Context, _ types.HostCommandEnvelope, input types.RuntimeInputEnvelope) (types.HostAdmissionStatus, string, error) {
	c.called = true
	if err := input.Validate(); err != nil {
		return types.HostAdmissionStatusRejected, types.RuntimeInputReasonInvalidEnvelope, err
	}
	if c.status != "" || c.reason != "" {
		return c.status, c.reason, nil
	}
	return types.HostAdmissionStatusAccepted, types.RuntimeInputReasonAccepted, nil
}

func TestConnectionAdmitsSteeringAndFollowUpAsSourceOwnedInput(t *testing.T) {
	control := &runtimeInputControl{}
	frames := make(chan any, 2)
	conn := New(nil, WithSourceControl(control)).Connect(func(frame any) error {
		frames <- frame
		return nil
	})
	cmd := types.HostCommandEnvelope{
		Version: types.HostProtocolVersionV1, MessageID: "input-message", Kind: types.HostCommandKindSteering,
		Time: time.Unix(1, 0).UTC(), RequestID: "input-request", SessionID: "session-1", RunID: "run-1",
		Payload: map[string]any{"input_id": "input-1", "payload": "adjust", "time": time.Unix(1, 0).UTC()},
	}
	response, err := conn.HandleCommand(context.Background(), cmd)
	if err != nil || response.Status != types.HostAdmissionStatusAccepted || response.InputKind != types.RuntimeInputKindSteering || response.InputID != "input-1" {
		t.Fatalf("response=%+v err=%v", response, err)
	}
	if !control.called {
		t.Fatal("source input owner was not called")
	}
	if got := (<-frames).(types.HostCommandResponse); got.Status != types.HostAdmissionStatusAccepted {
		t.Fatalf("frame=%+v", got)
	}
}

func TestConnectionRuntimeInputAdmissionFactsRemainCorrelatedAndBounded(t *testing.T) {
	control := &runtimeInputControl{}
	collector := &factCollector{}
	conn := New(nil, WithSourceControl(control), WithEventSink(collector)).Connect(func(any) error { return nil })
	cmd := validHostCommand(types.HostCommandKindFollowUp)
	cmd.MessageID = "follow-up-message"
	cmd.RequestID = "follow-up-request"
	cmd.Payload = map[string]any{"input_id": "follow-up-1", "payload": "next step", "raw_payload": "must-not-escape"}
	response, err := conn.HandleCommand(context.Background(), cmd)
	if err != nil || response.Status != types.HostAdmissionStatusAccepted {
		t.Fatalf("response=%#v err=%v", response, err)
	}
	if response.InputKind != types.RuntimeInputKindFollowUp || response.InputID != "follow-up-1" {
		t.Fatalf("input correlation=%#v", response)
	}
	var found bool
	for _, ev := range collector.snapshot() {
		if ev.Type != types.EventTypeHostAdmission {
			continue
		}
		if ev.Payload["input_kind"] == string(types.RuntimeInputKindFollowUp) && ev.Payload["input_id"] == "follow-up-1" {
			found = true
		}
		if _, leaked := ev.Payload["raw_payload"]; leaked {
			t.Fatalf("raw host input leaked into fact: %#v", ev.Payload)
		}
	}
	if !found {
		t.Fatalf("missing correlated host admission fact: %#v", collector.snapshot())
	}
}

func TestConnectionRuntimeInputAuthorizationAndCapabilityRejectionPrecedeSourceAdmission(t *testing.T) {
	control := &runtimeInputControl{}
	coord := New(nil, WithSourceControl(control), WithAuthorization(contractAuthorization{err: errors.New("policy.denied")}))
	cmd := validHostCommand(types.HostCommandKindSteering)
	cmd.Payload = map[string]any{"input_id": "steer-denied", "payload": "must not mutate"}
	response, err := coord.Connect(func(any) error { return nil }).HandleCommand(context.Background(), cmd)
	if err != nil || response.Status != types.HostAdmissionStatusRejected || response.ReasonCode != "policy.denied" {
		t.Fatalf("authorization response=%#v err=%v", response, err)
	}
	if control.called {
		t.Fatal("source admission ran despite authorization denial")
	}

	unsupported := &runtimeInputControl{status: types.HostAdmissionStatusRejected, reason: types.RuntimeInputReasonUnsupportedKind}
	coord = New(nil, WithSourceControl(unsupported))
	cmd.MessageID = "steer-unsupported-message"
	cmd.RequestID = "steer-unsupported-request"
	response, err = coord.Connect(func(any) error { return nil }).HandleCommand(context.Background(), cmd)
	if err != nil || response.Status != types.HostAdmissionStatusRejected || response.ReasonCode != types.RuntimeInputReasonUnsupportedKind {
		t.Fatalf("capability response=%#v err=%v", response, err)
	}
}
