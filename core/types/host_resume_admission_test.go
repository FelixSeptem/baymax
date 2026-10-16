package types

import (
	"encoding/json"
	"strings"
	"testing"
)

func hostResumeFixture() (HostResumeRequest, HostResumeResult) {
	a := DynamicActionReference{Token: "token", Kind: "prepare", Resumable: true, RunID: "run", SessionID: "session", Iteration: 1, CallID: "call", Source: "host", Digest: "digest"}
	cp := RunCheckpoint{Version: DynamicActionResumeProtocolVersion, CheckpointID: "checkpoint", RunID: "run", SessionID: "session", State: RunStateInputRequired, Mode: "run", Iteration: 1, PendingAction: &a, Digest: "digest"}
	d := DynamicActionDecision{Decision: DynamicActionDecisionConfirm, Token: "token", RunID: "run", SessionID: "session", CheckpointID: "checkpoint", CheckpointVersion: DynamicActionResumeProtocolVersion, CheckpointDigest: "digest", IdempotencyKey: "key", Reference: "confirmation"}
	r := HostResumeRequest{Version: HostResumeAdmissionVersion, Checkpoint: cp, Decision: d}
	f := HostResumeFacts{Plan: HostResumePlan{ID: "plan", Version: 1, Digest: "plan-digest"}, BusinessRevision: "revision", AuthorityDigest: "authority", ToolCapabilityDigest: "tools", WriteScopeDigest: "scope", ConfirmationReference: "confirmation", EffectReference: "effect", DecisionAuthorized: true, ExecutionAuthorized: true, CancellationAuthorized: true, ConfirmationValid: true, EffectEligible: true}
	return r, HostResumeResult{Version: HostResumeAdmissionVersion, Status: RunResumeAdmissionAccepted, Binding: HostResumeBinding{Version: HostResumeAdmissionVersion, Checkpoint: cp, Plan: f.Plan, BusinessRevision: f.BusinessRevision, AuthorityDigest: f.AuthorityDigest, ToolCapabilityDigest: f.ToolCapabilityDigest, WriteScopeDigest: f.WriteScopeDigest, ConfirmationReference: f.ConfirmationReference, EffectReference: f.EffectReference}, Current: f}
}

func TestHostResumeAdmissionBounds(t *testing.T) {
	r, result := hostResumeFixture()
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := ValidateHostResumeResult(r, result); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		mutate func(*HostResumeResult)
	}{
		{"empty identity", func(x *HostResumeResult) { x.Binding.Plan.ID = "" }},
		{"zero plan", func(x *HostResumeResult) { x.Binding.Plan.Version = 0 }},
		{"version", func(x *HostResumeResult) { x.Version = "unsupported" }},
		{"invalid utf8", func(x *HostResumeResult) { x.Binding.BusinessRevision = string([]byte{255}) }},
		{"space", func(x *HostResumeResult) { x.Binding.BusinessRevision = "has space" }},
		{"reference", func(x *HostResumeResult) { x.Binding.Plan.ID = strings.Repeat("x", 257) }},
		{"digest", func(x *HostResumeResult) { x.Binding.Plan.Digest = strings.Repeat("x", 129) }},
		{"envelope", func(x *HostResumeResult) {
			x.Binding.Checkpoint.ToolResultRefs = make([]string, 200)
			for i := range x.Binding.Checkpoint.ToolResultRefs {
				x.Binding.Checkpoint.ToolResultRefs[i] = strings.Repeat("x", 256)
			}
		}},
		{"status", func(x *HostResumeResult) { x.Status = RunResumeAdmissionDuplicate }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, x := hostResumeFixture()
			tc.mutate(&x)
			if err := ValidateHostResumeResult(r, x); err == nil {
				t.Fatal("invalid result accepted")
			}
		})
	}
	result.Binding.Plan.ID = strings.Repeat("界", 85) + "x"
	result.Current.Plan = result.Binding.Plan
	result.Binding.Plan.Digest = strings.Repeat("x", 128)
	result.Current.Plan = result.Binding.Plan
	if err := ValidateHostResumeResult(r, result); err != nil {
		t.Fatalf("boundary rejected: %v", err)
	}
}

func TestHostResumeAdmissionDrift(t *testing.T) {
	r, _ := hostResumeFixture()
	for _, tc := range []struct {
		name   string
		mutate func(*HostResumeResult)
		want   HostResumeReason
	}{
		{"authority", func(x *HostResumeResult) { x.Current.DecisionAuthorized = false }, HostResumeAuthorityDenied},
		{"write", func(x *HostResumeResult) { x.Current.ExecutionAuthorized = false }, HostResumeAuthorityDenied},
		{"authority digest", func(x *HostResumeResult) { x.Current.AuthorityDigest = "new" }, HostResumeAuthorityDenied},
		{"revision", func(x *HostResumeResult) { x.Current.BusinessRevision = "new" }, HostResumeRevisionDrift},
		{"plan", func(x *HostResumeResult) { x.Current.Plan.Version++ }, HostResumePlanDrift},
		{"tools", func(x *HostResumeResult) { x.Current.ToolCapabilityDigest = "new" }, HostResumeScopeDrift},
		{"scope", func(x *HostResumeResult) { x.Current.WriteScopeDigest = "new" }, HostResumeScopeDrift},
		{"confirmation", func(x *HostResumeResult) { x.Current.ConfirmationValid = false }, HostResumeConfirmationInvalid},
		{"effect", func(x *HostResumeResult) { x.Current.EffectEligible = false }, HostResumeEffectConflict},
		{"binding", func(x *HostResumeResult) { x.Binding.Checkpoint.CheckpointID = "other" }, HostResumeBindingMismatch},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, x := hostResumeFixture()
			tc.mutate(&x)
			err := ValidateHostResumeResult(r, x)
			if err == nil || err.Error() != string(tc.want) {
				t.Fatalf("err=%v want=%s", err, tc.want)
			}
		})
	}
	for _, kind := range []DynamicActionDecisionKind{DynamicActionDecisionDeny, DynamicActionDecisionTimeout} {
		r.Decision.Decision = kind
		_, x := hostResumeFixture()
		x.Current.ExecutionAuthorized = false
		x.Current.BusinessRevision = "new"
		x.Current.WriteScopeDigest = "new"
		if err := ValidateHostResumeResult(r, x); err != nil {
			t.Fatal(err)
		}
		x.Current.CancellationAuthorized = false
		if err := ValidateHostResumeResult(r, x); err == nil {
			t.Fatal("unauthorized cancellation")
		}
	}
}

func TestHostResumeSnapshotDetached(t *testing.T) {
	r, _ := hostResumeFixture()
	r.Checkpoint.ToolResultRefs = []string{"original"}
	x := r.Detached()
	x.Checkpoint.PendingAction.Token = "mutated"
	x.Checkpoint.ToolResultRefs[0] = "mutated"
	if r.Checkpoint.PendingAction.Token != "token" || r.Checkpoint.ToolResultRefs[0] != "original" {
		t.Fatal("snapshot aliases source")
	}
}

func TestHostResumeDecisionUTF8(t *testing.T) {
	for _, field := range []string{"key", "reference"} {
		r, _ := hostResumeFixture()
		if field == "key" {
			r.Decision.IdempotencyKey = string([]byte{255})
		} else {
			r.Decision.Reference = string([]byte{255})
		}
		if err := r.Validate(); err != HostResumeInvalidBinding {
			t.Fatalf("%s: err=%v, want invalid binding", field, err)
		}
	}
}

func TestHostResumeEnvelopeExactBoundary(t *testing.T) {
	for _, response := range []bool{false, true} {
		r, x := hostResumeFixture()
		refs := &r.Checkpoint.ToolResultRefs
		var envelope any = &r
		validate := func() error {
			if response {
				return ValidateHostResumeResult(r, x)
			}
			return r.Validate()
		}
		if response {
			refs = &x.Binding.Checkpoint.ToolResultRefs
			envelope = &x
		}
		// A first reference makes subsequent JSON additions exactly len(ref)+3.
		*refs = []string{"x"}
		for {
			raw, err := json.Marshal(envelope)
			if err != nil {
				t.Fatal(err)
			}
			remaining := HostResumeMaxEnvelopeBytes - len(raw)
			if remaining <= HostResumeMaxReferenceBytes+3 {
				if remaining <= 3 {
					t.Fatalf("unexpected boundary remainder %d", remaining)
				}
				*refs = append(*refs, strings.Repeat("x", remaining-3))
				break
			}
			*refs = append(*refs, strings.Repeat("x", HostResumeMaxReferenceBytes))
		}
		raw, _ := json.Marshal(envelope)
		if len(raw) != HostResumeMaxEnvelopeBytes || validate() != nil {
			t.Fatalf("response=%v boundary size=%d err=%v", response, len(raw), validate())
		}
		// Lengthen the first reference, staying inside its individual limit.
		(*refs)[0] += "x"
		if err := validate(); err != HostResumeInvalidBinding {
			t.Fatalf("response=%v oversized err=%v", response, err)
		}
	}
}
