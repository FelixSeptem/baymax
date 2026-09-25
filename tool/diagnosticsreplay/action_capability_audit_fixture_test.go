package diagnosticsreplay

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

const actionCapabilityAuditFixturePath = "testdata/action_capability_audit.v1.json"

func TestReplayActionCapabilityAuditFixtureIsBoundedDeterministicAndCovered(t *testing.T) {
	raw, err := os.ReadFile(actionCapabilityAuditFixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if strings.Contains(strings.ToLower(string(raw)), "bearer ") || strings.Contains(strings.ToLower(string(raw)), "password") || strings.Contains(strings.ToLower(string(raw)), "raw_payload") {
		t.Fatal("fixture contains sensitive material")
	}
	first, err := ReplayActionCapabilityAuditJSON(raw)
	if err != nil {
		t.Fatalf("first replay: %v", err)
	}
	second, err := ReplayActionCapabilityAuditJSON(raw)
	if err != nil {
		t.Fatalf("second replay: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("replay is not idempotent: first=%#v second=%#v", first, second)
	}
	if len(first.Cases) < 8 {
		t.Fatalf("fixture coverage = %d, want at least 8 cases", len(first.Cases))
	}
	for _, item := range first.Cases {
		if !item.Idempotent || item.Digest != item.ReplayDigest {
			t.Fatalf("case %q is not idempotent: %#v", item.CaseID, item)
		}
	}
}

func TestReplayActionCapabilityAuditRejectsUnsupportedVersionAndDuplicateConflict(t *testing.T) {
	raw, err := os.ReadFile(actionCapabilityAuditFixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var fixture ActionCapabilityAuditFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	fixture.Version = "action_capability_audit.v0"
	mutated, err := json.Marshal(fixture)
	if err != nil {
		t.Fatalf("marshal unsupported version: %v", err)
	}
	if _, err := ReplayActionCapabilityAuditJSON(mutated); err == nil || !strings.Contains(err.Error(), ReasonCodeActionCapabilityUnknownVersion) {
		t.Fatalf("unsupported version error = %v", err)
	}

	fixture.Version = ActionCapabilityAuditVersion
	fixture.Cases = []ActionCapabilityAuditCase{{
		CaseID:   "duplicate-conflict",
		Action:   ActionCapabilityDescriptor{Identity: "catalog.lookup", Source: "host", Version: "v1", Owner: "owner", Scope: "scope", Effect: "read", SideEffect: "none", Risk: "low", Reversible: ptrBool(true), Idempotent: ptrBool(true), Retryable: ptrBool(true), TimeoutMS: 1000},
		Evidence: []ActionCapabilityEvidence{{ID: "e1", Stage: "intent", Status: "observed"}, {ID: "e1", Stage: "issued", Status: "observed"}},
	}}
	mutated, err = json.Marshal(fixture)
	if err != nil {
		t.Fatalf("marshal duplicate fixture: %v", err)
	}
	if _, err := ReplayActionCapabilityAuditJSON(mutated); err == nil || !strings.Contains(err.Error(), ReasonCodeActionCapabilityDuplicateConflict) {
		t.Fatalf("duplicate conflict error = %v", err)
	}
}

func TestReplayActionCapabilityAuditNegativeFixturesRemainDeterministic(t *testing.T) {
	cases := []struct {
		fixture string
		code    string
	}{
		{fixture: "action_capability_audit_duplicate_conflict.json", code: ReasonCodeActionCapabilityDuplicateConflict},
		{fixture: "action_capability_audit_privacy_violation.json", code: ReasonCodeActionCapabilityPrivacyOrBoundViolation},
	}
	for _, tc := range cases {
		t.Run(tc.fixture, func(t *testing.T) {
			raw, err := os.ReadFile("testdata/" + tc.fixture)
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			_, err = ReplayActionCapabilityAuditJSON(raw)
			if err == nil || !strings.Contains(err.Error(), tc.code) {
				t.Fatalf("error = %v, want %q", err, tc.code)
			}
		})
	}
	raw, err := os.ReadFile("testdata/action_capability_audit_approval_scope_drift.json")
	if err != nil {
		t.Fatalf("read approval scope fixture: %v", err)
	}
	result, err := ReplayActionCapabilityAuditJSON(raw)
	if err != nil {
		t.Fatalf("approval scope replay: %v", err)
	}
	if len(result.Cases) != 1 || !containsActionCapabilityString(result.Cases[0].DriftCodes, ReasonCodeActionCapabilityApprovalScopeDrift) {
		t.Fatalf("approval scope drift was not classified: %#v", result)
	}
}

func TestReplayActionCapabilityAuditRunStreamEdgeFixtureKeepsEquivalentSemantics(t *testing.T) {
	raw, err := os.ReadFile("testdata/action_capability_audit_run_stream_edges.v1.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	result, err := ReplayActionCapabilityAuditJSON(raw)
	if err != nil {
		t.Fatalf("replay fixture: %v", err)
	}
	if len(result.Cases) != 7 {
		t.Fatalf("edge case total = %d, want 7", len(result.Cases))
	}
	for _, item := range result.Cases {
		if item.RunStreamParity != "equivalent" {
			t.Fatalf("case %q parity = %q, want equivalent", item.CaseID, item.RunStreamParity)
		}
	}
}

func TestReplayActionCapabilityAuditDigestIncludesRunStreamEvidence(t *testing.T) {
	raw, err := os.ReadFile("testdata/action_capability_audit_run_stream_edges.v1.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var fixture ActionCapabilityAuditFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	firstRaw, err := json.Marshal(fixture)
	if err != nil {
		t.Fatalf("marshal original fixture: %v", err)
	}
	first, err := ReplayActionCapabilityAuditJSON(firstRaw)
	if err != nil {
		t.Fatalf("replay original fixture: %v", err)
	}
	fixture.Cases[0].Run.Evidence[0].ID = "different-but-stable-reference"
	fixture.Cases[0].Stream.Evidence[0].ID = "different-but-stable-reference"
	secondRaw, err := json.Marshal(fixture)
	if err != nil {
		t.Fatalf("marshal changed fixture: %v", err)
	}
	second, err := ReplayActionCapabilityAuditJSON(secondRaw)
	if err != nil {
		t.Fatalf("replay changed fixture: %v", err)
	}
	if first.Cases[0].Digest == second.Cases[0].Digest {
		t.Fatal("paired lane evidence changed without changing the normalized audit digest")
	}
}

func TestReplayActionCapabilityAuditHistoricalFixturesRemainReadable(t *testing.T) {
	paths := []string{
		"testdata/tool_lifecycle_failure_isolation.json",
		"testdata/terminal_outcomes.json",
		"testdata/success_input.json",
		"testdata/missing_field_input.json",
		"../../integration/testdata/diagnostics-replay/extension-lifecycle-governance/v1/success.json",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read historical fixture: %v", err)
			}
			if !json.Valid(raw) {
				t.Fatal("historical fixture is not valid JSON")
			}
		})
	}
	legacy := ActionCapabilityAuditFixture{Version: ActionCapabilityAuditVersion, Cases: []ActionCapabilityAuditCase{{CaseID: "legacy-without-extension", Action: ActionCapabilityDescriptor{}}}}
	raw, err := json.Marshal(legacy)
	if err != nil {
		t.Fatalf("marshal legacy audit fixture: %v", err)
	}
	result, err := ReplayActionCapabilityAuditJSON(raw)
	if err != nil {
		t.Fatalf("legacy audit fixture replay: %v", err)
	}
	if len(result.Cases) != 1 || result.Cases[0].Verdict != ActionCapabilityVerdictNotApplicable {
		t.Fatalf("legacy audit defaults = %#v", result)
	}
}
