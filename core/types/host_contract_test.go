package types

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestHostCommandRoundTripAndValidation(t *testing.T) {
	in := HostCommandEnvelope{Version: HostProtocolVersionV1, MessageID: "msg-1", Kind: HostCommandKindRunStart, Time: time.Unix(1, 0).UTC(), RequestID: "req-1", SessionID: "sess-1", RunID: "run-1", Payload: map[string]any{"input": "hello"}}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out HostCommandEnvelope
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if err := out.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if out.Kind != HostCommandKindRunStart || out.RequestID != "req-1" {
		t.Fatalf("round trip lost fields: %+v", out)
	}
}

func TestHostCommandValidationRejectsUnknownAndCorrelationMismatch(t *testing.T) {
	base := HostCommandEnvelope{Version: HostProtocolVersionV1, MessageID: "m", Kind: HostCommandKindRunStart, Time: time.Now(), RequestID: "r", SessionID: "s", RunID: "x"}
	cases := []HostCommandEnvelope{
		func() HostCommandEnvelope { c := base; c.Version = "unknown"; return c }(),
		func() HostCommandEnvelope { c := base; c.Kind = HostCommandKind("bad"); return c }(),
		func() HostCommandEnvelope { c := base; c.RequestID = ""; return c }(),
		func() HostCommandEnvelope { c := base; c.RunID = "other id"; return c }(),
	}
	for i, c := range cases {
		if err := c.Validate(); err == nil {
			t.Errorf("case %d expected failure", i)
		}
	}
}

func TestHostCommandAdmissionSeparatesAsyncOutcome(t *testing.T) {
	req := HostCommandEnvelope{Version: HostProtocolVersionV1, MessageID: "m", Kind: HostCommandKindRunStart, Time: time.Now(), RequestID: "r", SessionID: "s", RunID: "run"}
	response, err := NormalizeHostCommandAdmission(req, HostAdmissionStatusAccepted, "")
	if err != nil {
		t.Fatal(err)
	}
	if response.Status != HostAdmissionStatusAccepted || response.Terminal != nil {
		t.Fatalf("unexpected response: %+v", response)
	}
	rejected, err := NormalizeHostCommandAdmission(req, HostAdmissionStatusRejected, HostReasonUnauthorized)
	if err != nil || rejected.ReasonCode != HostReasonUnauthorized {
		t.Fatalf("rejected=%+v err=%v", rejected, err)
	}
}

func TestHostSourceAuthorizationHelpers(t *testing.T) {
	d, _ := ProtocolDescriptorForSource(ProtocolSourceRunner, "rt", "v1", nil, []ProtocolAction{ProtocolActionCancel})
	if err := ValidateHostSourceOwnership(HostSourceOwnership{Descriptor: d, SessionID: "s", RunID: "r", Active: true}); err != nil {
		t.Fatal(err)
	}
	if err := AuthorizeHostCommand(HostAuthorization{Descriptor: d, Action: ProtocolActionCancel, Ready: true, PolicyAllowed: true, SandboxAllowed: true, DurableBindingValid: true, TerminalRecoveryValid: true}); err != nil {
		t.Fatal(err)
	}
	denied := HostAuthorization{Descriptor: d, Action: ProtocolActionCancel, Ready: true}
	if err := AuthorizeHostCommand(denied); err == nil || !strings.Contains(err.Error(), HostReasonUnauthorized) {
		t.Fatalf("expected denied, got %v", err)
	}
}
