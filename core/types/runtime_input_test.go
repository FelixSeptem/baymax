package types

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func validRuntimeInput() RuntimeInputEnvelope {
	return RuntimeInputEnvelope{
		Version:           RuntimeInputProtocolVersionV1,
		InputID:           "input-1",
		Kind:              RuntimeInputKindSteering,
		Time:              time.Unix(1, 0).UTC(),
		SessionID:         "session-1",
		RunID:             "run-1",
		CausationID:       "cause-1",
		SourceCorrelation: "source-1",
		Payload:           "continue with the safer option",
	}
}

func TestRuntimeInputEnvelopeRoundTripAndNormalizedIdentity(t *testing.T) {
	in := validRuntimeInput()
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out RuntimeInputEnvelope
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if err := out.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if got := out.NormalizedIdentity(); got != "session-1/run-1/input-1" {
		t.Fatalf("identity=%q", got)
	}
	if out.ApplyBoundary != RuntimeInputApplyBoundaryUnspecified {
		t.Fatalf("omitted apply boundary should default to unspecified, got %q", out.ApplyBoundary)
	}
}

func TestRuntimeInputEnvelopeValidationRejectsMalformedPayloads(t *testing.T) {
	base := validRuntimeInput()
	cases := []RuntimeInputEnvelope{
		func() RuntimeInputEnvelope { v := base; v.Version = "runtime_input.v0"; return v }(),
		func() RuntimeInputEnvelope { v := base; v.Kind = RuntimeInputKind("unknown"); return v }(),
		func() RuntimeInputEnvelope { v := base; v.InputID = ""; return v }(),
		func() RuntimeInputEnvelope { v := base; v.SessionID = ""; return v }(),
		func() RuntimeInputEnvelope { v := base; v.RunID = ""; return v }(),
		func() RuntimeInputEnvelope {
			v := base
			v.Payload = strings.Repeat("x", RuntimeInputMaxPayloadBytes+1)
			return v
		}(),
		func() RuntimeInputEnvelope { v := base; v.Payload = "line\nfeed"; return v }(),
	}
	for i, tc := range cases {
		if err := tc.Validate(); err == nil {
			t.Errorf("case %d expected validation failure", i)
		}
	}
}

func TestRuntimeInputAdmissionReasonNormalization(t *testing.T) {
	input := validRuntimeInput()
	input.Kind = RuntimeInputKindFollowUp
	admission, err := NormalizeRuntimeInputAdmission(input, RuntimeInputAdmissionStatusRejected, "")
	if err != nil {
		t.Fatal(err)
	}
	if admission.Status != RuntimeInputAdmissionStatusRejected || admission.ReasonCode != RuntimeInputReasonInvalidEnvelope {
		t.Fatalf("admission=%+v", admission)
	}
	duplicate, err := NormalizeRuntimeInputAdmission(input, RuntimeInputAdmissionStatusDuplicate, "")
	if err != nil {
		t.Fatal(err)
	}
	if duplicate.ReasonCode != RuntimeInputReasonDuplicate {
		t.Fatalf("duplicate=%+v", duplicate)
	}
}

func TestRuntimeInputOutcomeVocabularyIsStable(t *testing.T) {
	want := []RuntimeInputAdmissionStatus{
		RuntimeInputAdmissionStatusAccepted,
		RuntimeInputAdmissionStatusRejected,
		RuntimeInputAdmissionStatusDuplicate,
	}
	for _, status := range want {
		if !IsValidRuntimeInputAdmissionStatus(status) {
			t.Fatalf("status %q should be valid", status)
		}
	}
	for _, reason := range []string{
		RuntimeInputReasonAccepted,
		RuntimeInputReasonRejected,
		RuntimeInputReasonDuplicate,
		RuntimeInputReasonStale,
		RuntimeInputReasonTerminal,
		RuntimeInputReasonBackpressure,
		RuntimeInputReasonDisconnected,
		RuntimeInputReasonNotApplied,
	} {
		if !IsValidRuntimeInputReason(reason) {
			t.Fatalf("reason %q should be valid", reason)
		}
	}
}
