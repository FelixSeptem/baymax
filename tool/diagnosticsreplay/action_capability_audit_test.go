package diagnosticsreplay

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestReplayActionCapabilityAuditNormalizesCompleteReadAction(t *testing.T) {
	readOnly := true
	idempotent := true
	retryable := true
	fixture := ActionCapabilityAuditFixture{
		Version: ActionCapabilityAuditVersion,
		Cases: []ActionCapabilityAuditCase{{
			CaseID: "read-action",
			Action: ActionCapabilityDescriptor{
				Identity:   "catalog.lookup",
				Source:     "host-admitted",
				Namespace:  "catalog",
				Tool:       "lookup",
				Version:    "2026-09-24",
				Owner:      "catalog-owner",
				Scope:      "tenant:demo",
				Effect:     "read",
				SideEffect: "none",
				Risk:       "low",
				Reversible: &readOnly,
				Idempotent: &idempotent,
				Retryable:  &retryable,
				TimeoutMS:  1000,
			},
			Expected: ActionCapabilityAuditExpected{Verdict: ActionCapabilityVerdictCompliant},
		}},
	}
	raw, err := json.Marshal(fixture)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	result, err := ReplayActionCapabilityAuditJSON(raw)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if len(result.Cases) != 1 || result.Cases[0].Verdict != ActionCapabilityVerdictCompliant {
		t.Fatalf("unexpected result: %#v", result)
	}
	if result.Cases[0].Digest == "" || !result.Cases[0].Idempotent {
		t.Fatalf("expected deterministic digest: %#v", result.Cases[0])
	}
}

func TestReplayActionCapabilityAuditRejectsMissingVersionAndSensitiveEvidence(t *testing.T) {
	raw := []byte(`{"version":"action_capability_audit.v1","cases":[{"case_id":"bad","action":{"identity":"write","source":"host","tool":"write","digest":"sha256:1"},"evidence":[{"id":"e1","stage":"issued","status":"observed","summary":"Authorization: Bearer secret"}]}]}`)
	if _, err := ReplayActionCapabilityAuditJSON(raw); err == nil {
		t.Fatal("expected privacy/bound validation error")
	} else if !strings.Contains(err.Error(), ReasonCodeActionCapabilityPrivacyOrBoundViolation) {
		t.Fatalf("error = %v, want privacy classification", err)
	}
}

func TestReplayActionCapabilityAuditDistinguishesIssuedWithoutConfirmation(t *testing.T) {
	idempotent := false
	retryable := false
	action := ActionCapabilityDescriptor{
		Identity:   "billing.charge",
		Source:     "host-admitted",
		Tool:       "charge",
		Digest:     "sha256:charge",
		Owner:      "billing",
		Scope:      "tenant:demo",
		Effect:     "write",
		SideEffect: "external",
		Risk:       "high",
		Reversible: ptrBool(false),
		Idempotent: &idempotent,
		Retryable:  &retryable,
		TimeoutMS:  1000,
		Stages:     []string{"preview", "approve", "commit", "verify"},
	}
	fixture := ActionCapabilityAuditFixture{
		Version: ActionCapabilityAuditVersion,
		Cases: []ActionCapabilityAuditCase{{
			CaseID:   "issued-unknown",
			Action:   action,
			Evidence: []ActionCapabilityEvidence{{ID: "i1", Stage: "issued", Status: "observed", ActionIdentity: action.Identity, Version: action.Digest, Scope: action.Scope}},
			Expected: ActionCapabilityAuditExpected{Verdict: ActionCapabilityVerdictInsufficientEvidence},
		}},
	}
	raw, err := json.Marshal(fixture)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	result, err := ReplayActionCapabilityAuditJSON(raw)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if got := result.Cases[0].Verdict; got != ActionCapabilityVerdictInsufficientEvidence {
		t.Fatalf("verdict = %q, want %q", got, ActionCapabilityVerdictInsufficientEvidence)
	}
}

func TestReplayActionCapabilityAuditIssuedWithoutConfirmationIsNeverCompliant(t *testing.T) {
	action := ActionCapabilityDescriptor{
		Identity: "catalog.lookup", Source: "host-admitted", Version: "v1", Owner: "catalog-owner", Scope: "tenant:demo",
		Effect: "read", SideEffect: "none", Risk: "low", Reversible: ptrBool(true), Idempotent: ptrBool(true), Retryable: ptrBool(true), TimeoutMS: 1000,
	}
	raw, err := json.Marshal(ActionCapabilityAuditFixture{Version: ActionCapabilityAuditVersion, Cases: []ActionCapabilityAuditCase{{
		CaseID: "read-issued-unconfirmed", Action: action, Evidence: []ActionCapabilityEvidence{{ID: "issued", Stage: "issued", Status: "interrupted", ActionIdentity: action.Identity, Version: action.Version, Scope: action.Scope, AttemptID: "attempt-1", CorrelationID: "correlation-1"}},
		Expected: ActionCapabilityAuditExpected{Verdict: ActionCapabilityVerdictInsufficientEvidence, DriftCodes: []string{ReasonCodeActionCapabilityEvidenceInsufficient}},
	}}})
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	if _, err := ReplayActionCapabilityAuditJSON(raw); err != nil {
		t.Fatalf("replay issued-unconfirmed read: %v", err)
	}
}

func TestReplayActionCapabilityAuditRejectsDiscoveryAndApprovalScopeDrift(t *testing.T) {
	base := ActionCapabilityDescriptor{
		Identity: "billing.charge", Source: "host-admitted", Tool: "charge", Digest: "sha256:charge", Owner: "billing", Scope: "tenant:demo",
		Effect: "write", SideEffect: "external", Risk: "high", Reversible: ptrBool(false), Idempotent: ptrBool(true), Retryable: ptrBool(false), TimeoutMS: 1000,
		Stages: []string{"preview", "approve", "commit", "verify"},
	}
	fixture := ActionCapabilityAuditFixture{Version: ActionCapabilityAuditVersion, Cases: []ActionCapabilityAuditCase{{
		CaseID: "scope-drift", Action: base,
		Evidence: []ActionCapabilityEvidence{
			{ID: "preview", Stage: "preview", Status: "observed", ActionIdentity: base.Identity, Version: base.Digest, Scope: "tenant:demo"},
			{ID: "approve", Stage: "approve", Status: "confirmed", ActionIdentity: base.Identity, Version: base.Digest, Scope: "tenant:demo"},
			{ID: "commit", Stage: "commit", Status: "observed", ActionIdentity: base.Identity, Version: base.Digest, Scope: "tenant:other"},
			{ID: "verify", Stage: "verify", Status: "confirmed", ActionIdentity: base.Identity, Version: base.Digest, Scope: "tenant:other"},
		},
		Expected: ActionCapabilityAuditExpected{Verdict: ActionCapabilityVerdictGap, DriftCodes: []string{ReasonCodeActionCapabilityApprovalScopeDrift, ReasonCodeActionCapabilityEvidenceCorrelationDrift}},
	}}}
	raw, err := json.Marshal(fixture)
	if err != nil {
		t.Fatalf("marshal scope fixture: %v", err)
	}
	if _, err := ReplayActionCapabilityAuditJSON(raw); err != nil {
		t.Fatalf("scope replay: %v", err)
	}

	fixture.Cases[0].Action.Source = "runtime-discovery"
	raw, err = json.Marshal(fixture)
	if err != nil {
		t.Fatalf("marshal discovery fixture: %v", err)
	}
	if _, err := ReplayActionCapabilityAuditJSON(raw); err == nil || !strings.Contains(err.Error(), ReasonCodeActionCapabilityExecutionOrDiscovery) {
		t.Fatalf("discovery error = %v", err)
	}
}

func TestReplayActionCapabilityAuditBoundsRawFixtureBytes(t *testing.T) {
	raw := []byte(`{"version":"action_capability_audit.v1","cases":[{"case_id":"bounded","action":{}}],"ignored":"` + strings.Repeat("x", 1<<20) + `"}`)
	if _, err := ReplayActionCapabilityAuditJSON(raw); err == nil || !strings.Contains(err.Error(), ReasonCodeActionCapabilityPrivacyOrBoundViolation) {
		t.Fatalf("oversized fixture error = %v, want privacy/bound classification", err)
	}
}

func TestReplayActionCapabilityAuditCorrelatesConfirmationWithIssuedAttempt(t *testing.T) {
	action := ActionCapabilityDescriptor{
		Identity: "billing.charge", Source: "host-admitted", Digest: "sha256:charge", Owner: "billing", Scope: "tenant:demo",
		Effect: "write", SideEffect: "external", Risk: "high", Reversible: ptrBool(false), Idempotent: ptrBool(true), Retryable: ptrBool(false),
		TimeoutMS: 1000, Stages: []string{"preview", "approve", "commit", "verify"},
	}
	evidence := []ActionCapabilityEvidence{
		{ID: "preview", Stage: "preview", Status: "confirmed", ActionIdentity: action.Identity, Version: action.Digest, Scope: action.Scope},
		{ID: "approve", Stage: "approve", Status: "confirmed", ActionIdentity: action.Identity, Version: action.Digest, Scope: action.Scope},
		{ID: "commit", Stage: "commit", Status: "observed", ActionIdentity: action.Identity, Version: action.Digest, Scope: action.Scope},
		{ID: "verify", Stage: "verify", Status: "confirmed", ActionIdentity: action.Identity, Version: action.Digest, Scope: action.Scope},
		{ID: "issued", Stage: "issued", Status: "observed", ActionIdentity: action.Identity, Version: action.Digest, Scope: action.Scope, AttemptID: "attempt-1", CorrelationID: "correlation-1"},
		{ID: "confirmed", Stage: "confirmed", Status: "confirmed", ActionIdentity: action.Identity, Version: action.Digest, Scope: action.Scope, AttemptID: "attempt-2", CorrelationID: "correlation-2"},
	}
	raw, err := json.Marshal(ActionCapabilityAuditFixture{Version: ActionCapabilityAuditVersion, Cases: []ActionCapabilityAuditCase{{CaseID: "mismatched-confirmation", Action: action, Evidence: evidence}}})
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	result, err := ReplayActionCapabilityAuditJSON(raw)
	if err != nil {
		t.Fatalf("replay mismatched confirmation: %v", err)
	}
	got := result.Cases[0]
	if got.Verdict != ActionCapabilityVerdictGap || !containsActionCapabilityString(got.DriftCodes, "action_capability_evidence_correlation_drift") {
		t.Fatalf("mismatched attempt/correlation was accepted: %#v", got)
	}
	if got.EvidenceState == "confirmed" {
		t.Fatalf("uncorrelated confirmation was counted as confirmed: %#v", got)
	}
}

func TestReplayActionCapabilityAuditDigestNormalizesObservedFacts(t *testing.T) {
	action := ActionCapabilityDescriptor{
		Identity: "catalog.lookup", Source: "host-admitted", Version: "v1", Owner: "catalog-owner", Scope: "tenant:demo",
		Effect: "read", SideEffect: "none", Risk: "low", Reversible: ptrBool(true), Idempotent: ptrBool(true), Retryable: ptrBool(true), TimeoutMS: 1000,
	}
	first := ActionCapabilityAuditFixture{Version: ActionCapabilityAuditVersion, Cases: []ActionCapabilityAuditCase{{CaseID: "normalized", Action: action, Observed: &ActionCapabilityObservedFacts{Effect: " READ ", SideEffect: " NONE ", Risk: " LOW "}}}}
	second := first
	second.Cases = append([]ActionCapabilityAuditCase(nil), first.Cases...)
	second.Cases[0].Observed = &ActionCapabilityObservedFacts{Effect: "read", SideEffect: "none", Risk: "low"}
	rawFirst, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("marshal first fixture: %v", err)
	}
	rawSecond, err := json.Marshal(second)
	if err != nil {
		t.Fatalf("marshal second fixture: %v", err)
	}
	firstResult, err := ReplayActionCapabilityAuditJSON(rawFirst)
	if err != nil {
		t.Fatalf("replay first fixture: %v", err)
	}
	secondResult, err := ReplayActionCapabilityAuditJSON(rawSecond)
	if err != nil {
		t.Fatalf("replay second fixture: %v", err)
	}
	if firstResult.Cases[0].Digest != secondResult.Cases[0].Digest {
		t.Fatalf("equivalent observed facts produced different digests: %q != %q", firstResult.Cases[0].Digest, secondResult.Cases[0].Digest)
	}
}

func TestReplayActionCapabilityAuditRejectsSensitiveMetadataAndUnboundedRetry(t *testing.T) {
	baseAction := ActionCapabilityDescriptor{
		Identity: "catalog.lookup", Source: "host-admitted", Version: "v1", Owner: "catalog-owner", Scope: "tenant:demo",
		Effect: "read", SideEffect: "none", Risk: "low", Reversible: ptrBool(true), Idempotent: ptrBool(true), Retryable: ptrBool(true), TimeoutMS: 1000,
	}
	cases := []struct {
		name   string
		mutate func(*ActionCapabilityDescriptor)
	}{
		{name: "credential in precondition", mutate: func(action *ActionCapabilityDescriptor) { action.Preconditions = []string{"Bearer secret-token"} }},
		{name: "retry budget exceeds limit", mutate: func(action *ActionCapabilityDescriptor) { action.RetryMax = 11 }},
		{name: "timeout exceeds limit", mutate: func(action *ActionCapabilityDescriptor) { action.TimeoutMS = 24*60*60*1000 + 1 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			action := baseAction
			tc.mutate(&action)
			raw, err := json.Marshal(ActionCapabilityAuditFixture{Version: ActionCapabilityAuditVersion, Cases: []ActionCapabilityAuditCase{{CaseID: "bounded-metadata", Action: action}}})
			if err != nil {
				t.Fatalf("marshal fixture: %v", err)
			}
			if _, err := ReplayActionCapabilityAuditJSON(raw); err == nil || !strings.Contains(err.Error(), ReasonCodeActionCapabilityPrivacyOrBoundViolation) {
				t.Fatalf("invalid metadata error = %v, want privacy/bound classification", err)
			}
		})
	}
}

func TestReplayActionCapabilityAuditCorrelatesHighRiskStageScopes(t *testing.T) {
	action := ActionCapabilityDescriptor{
		Identity: "billing.charge", Source: "host-admitted", Digest: "sha256:charge", Owner: "billing", Scope: "tenant:demo",
		Effect: "write", SideEffect: "external", Risk: "high", Reversible: ptrBool(false), Idempotent: ptrBool(true), Retryable: ptrBool(false),
		TimeoutMS: 1000, Stages: []string{"preview", "approve", "commit", "verify"},
	}
	evidence := make([]ActionCapabilityEvidence, 0, 4)
	for _, stage := range []string{"preview", "approve", "commit", "verify"} {
		evidence = append(evidence, ActionCapabilityEvidence{ID: stage, Stage: stage, Status: "confirmed", ActionIdentity: action.Identity, Version: action.Digest, Scope: "tenant:other"})
	}
	raw, err := json.Marshal(ActionCapabilityAuditFixture{Version: ActionCapabilityAuditVersion, Cases: []ActionCapabilityAuditCase{{CaseID: "stage-scope-drift", Action: action, Evidence: evidence}}})
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	result, err := ReplayActionCapabilityAuditJSON(raw)
	if err != nil {
		t.Fatalf("replay fixture: %v", err)
	}
	got := result.Cases[0]
	if got.Verdict != ActionCapabilityVerdictGap || !containsActionCapabilityString(got.DriftCodes, "action_capability_evidence_correlation_drift") {
		t.Fatalf("stage scope differing from declared scope was accepted: %#v", got)
	}
}

func TestReplayActionCapabilityAuditClassifiesExpectedOutputMismatch(t *testing.T) {
	action := ActionCapabilityDescriptor{
		Identity: "catalog.lookup", Source: "host-admitted", Version: "v1", Owner: "catalog-owner", Scope: "tenant:demo",
		Effect: "read", SideEffect: "none", Risk: "low", Reversible: ptrBool(true), Idempotent: ptrBool(true), Retryable: ptrBool(true), TimeoutMS: 1000,
	}
	raw, err := json.Marshal(ActionCapabilityAuditFixture{Version: ActionCapabilityAuditVersion, Cases: []ActionCapabilityAuditCase{{
		CaseID: "expected-output-drift", Action: action, Expected: ActionCapabilityAuditExpected{Verdict: ActionCapabilityVerdictGap},
	}}})
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	if _, err := ReplayActionCapabilityAuditJSON(raw); err == nil || !strings.Contains(err.Error(), ReasonCodeActionCapabilityVerdictDrift) {
		t.Fatalf("expected output mismatch error = %v, want stable verdict drift code", err)
	}
}

func TestReplayActionCapabilityAuditComparesObservedStartedWithIssuedEvidence(t *testing.T) {
	action := ActionCapabilityDescriptor{
		Identity: "catalog.lookup", Source: "host-admitted", Version: "v1", Owner: "catalog-owner", Scope: "tenant:demo",
		Effect: "read", SideEffect: "none", Risk: "low", Reversible: ptrBool(true), Idempotent: ptrBool(true), Retryable: ptrBool(true), TimeoutMS: 1000,
	}
	cases := []struct {
		name    string
		started bool
		issued  bool
	}{
		{name: "observed started without issued evidence", started: true},
		{name: "issued evidence conflicts with not-started observation", issued: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var evidence []ActionCapabilityEvidence
			if tc.issued {
				evidence = []ActionCapabilityEvidence{{ID: "issued", Stage: "issued", Status: "interrupted"}}
			}
			raw, err := json.Marshal(ActionCapabilityAuditFixture{Version: ActionCapabilityAuditVersion, Cases: []ActionCapabilityAuditCase{{
				CaseID: "started-observation-drift", Action: action, Observed: &ActionCapabilityObservedFacts{Started: ptrBool(tc.started)}, Evidence: evidence,
			}}})
			if err != nil {
				t.Fatalf("marshal fixture: %v", err)
			}
			result, err := ReplayActionCapabilityAuditJSON(raw)
			if err != nil {
				t.Fatalf("replay fixture: %v", err)
			}
			got := result.Cases[0]
			if got.Verdict != ActionCapabilityVerdictGap || !containsActionCapabilityString(got.DriftCodes, ReasonCodeActionCapabilityDeclaredObservedDrift) {
				t.Fatalf("started observation drift was not reported: %#v", got)
			}
		})
	}
}

func TestReplayActionCapabilityAuditRejectsUnknownOrDuplicateJSONFields(t *testing.T) {
	base := `{"version":"action_capability_audit.v1","cases":[{"case_id":"strict-json","action":{}}]}`
	unknown := strings.Replace(base, `"action":{}`, `"action":{},"payload":{"authorization":"Bearer secret"}`, 1)
	unknownNeutral := strings.Replace(base, `"action":{}`, `"action":{},"future_extension":true`, 1)
	duplicate := strings.Replace(base, `"version":"action_capability_audit.v1"`, `"version":"action_capability_audit.v1","version":"action_capability_audit.v1"`, 1)
	for name, raw := range map[string]string{"unknown raw field": unknown, "unknown neutral field": unknownNeutral, "duplicate key": duplicate} {
		t.Run(name, func(t *testing.T) {
			if _, err := ReplayActionCapabilityAuditJSON([]byte(raw)); err == nil {
				t.Fatal("expected strict JSON boundary rejection")
			}
		})
	}
}

func TestReplayActionCapabilityAuditRejectsUnboundedOrSensitiveObservedFacts(t *testing.T) {
	action := ActionCapabilityDescriptor{Identity: "catalog.lookup", Source: "host-admitted", Version: "v1"}
	for name, observed := range map[string]ActionCapabilityObservedFacts{
		"sensitive marker": {Effect: "Bearer secret"},
		"oversized value":  {Risk: strings.Repeat("x", actionCapabilityMaxString+1)},
	} {
		t.Run(name, func(t *testing.T) {
			raw, err := json.Marshal(ActionCapabilityAuditFixture{Version: ActionCapabilityAuditVersion, Cases: []ActionCapabilityAuditCase{{CaseID: "observed-bound", Action: action, Observed: &observed}}})
			if err != nil {
				t.Fatalf("marshal fixture: %v", err)
			}
			if _, err := ReplayActionCapabilityAuditJSON(raw); err == nil || !strings.Contains(err.Error(), ReasonCodeActionCapabilityPrivacyOrBoundViolation) {
				t.Fatalf("observed validation error = %v, want privacy/bound classification", err)
			}
		})
	}
}

func TestReplayActionCapabilityAuditDoesNotTreatInvalidHighRiskStageStatusAsComplete(t *testing.T) {
	action := ActionCapabilityDescriptor{
		Identity: "billing.charge", Source: "host-admitted", Digest: "sha256:charge", Owner: "billing", Scope: "tenant:demo",
		Effect: "write", SideEffect: "external", Risk: "high", Reversible: ptrBool(false), Idempotent: ptrBool(true), Retryable: ptrBool(false),
		TimeoutMS: 1000, Stages: []string{"preview", "approve", "commit", "verify"},
	}
	evidence := []ActionCapabilityEvidence{
		{ID: "preview", Stage: "preview", Status: "timeout", ActionIdentity: action.Identity, Version: action.Digest, Scope: action.Scope},
		{ID: "approve", Stage: "approve", Status: "denied", ActionIdentity: action.Identity, Version: action.Digest, Scope: action.Scope},
		{ID: "commit", Stage: "commit", Status: "failed", ActionIdentity: action.Identity, Version: action.Digest, Scope: action.Scope},
		{ID: "verify", Stage: "verify", Status: "unknown", ActionIdentity: action.Identity, Version: action.Digest, Scope: action.Scope},
	}
	raw, err := json.Marshal(ActionCapabilityAuditFixture{Version: ActionCapabilityAuditVersion, Cases: []ActionCapabilityAuditCase{{CaseID: "invalid-stage-status", Action: action, Evidence: evidence}}})
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	result, err := ReplayActionCapabilityAuditJSON(raw)
	if err != nil {
		t.Fatalf("replay fixture: %v", err)
	}
	got := result.Cases[0]
	if got.Verdict == ActionCapabilityVerdictCompliant || !containsActionCapabilityString(got.DriftCodes, "action_capability_stage_evidence_insufficient") {
		t.Fatalf("negative/unknown high-risk stage statuses were accepted: %#v", got)
	}
}

func TestReplayActionCapabilityAuditRunStreamParityIgnoresReferenceIDs(t *testing.T) {
	action := ActionCapabilityDescriptor{
		Identity: "catalog.lookup", Source: "host-admitted", Version: "v1", Owner: "catalog-owner", Scope: "tenant:demo",
		Effect: "read", SideEffect: "none", Risk: "low", Reversible: ptrBool(true), Idempotent: ptrBool(true), Retryable: ptrBool(true), TimeoutMS: 1000,
	}
	runEvidence := []ActionCapabilityEvidence{{ID: "run-reference", Stage: "intent", Status: "observed", ActionIdentity: action.Identity, Version: action.Version, Scope: action.Scope, AttemptID: "attempt-1", CorrelationID: "correlation-1"}}
	streamEvidence := []ActionCapabilityEvidence{{ID: "stream-reference", Stage: "intent", Status: "observed", ActionIdentity: action.Identity, Version: action.Version, Scope: action.Scope, AttemptID: "attempt-1", CorrelationID: "correlation-1"}}
	fixture := ActionCapabilityAuditFixture{Version: ActionCapabilityAuditVersion, Cases: []ActionCapabilityAuditCase{{
		CaseID: "reference-id-parity", Action: action,
		Run: &ActionCapabilityEvidenceLane{Evidence: runEvidence}, Stream: &ActionCapabilityEvidenceLane{Evidence: streamEvidence},
		Expected: ActionCapabilityAuditExpected{Verdict: ActionCapabilityVerdictInsufficientEvidence, DriftCodes: []string{ReasonCodeActionCapabilityEvidenceInsufficient}},
	}}}
	raw, err := json.Marshal(fixture)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	result, err := ReplayActionCapabilityAuditJSON(raw)
	if err != nil {
		t.Fatalf("replay fixture: %v", err)
	}
	if got := result.Cases[0].RunStreamParity; got != "equivalent" {
		t.Fatalf("reference ID variance caused parity %q, want equivalent", got)
	}
}

func ptrBool(value bool) *bool { return &value }
