package extension

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDecodeAuthoringCaseCanonicalizesCandidatesAndCapabilities(t *testing.T) {
	raw := []byte(`{
  "profile":"external_extension_authoring.v1",
  "case":"valid-activation",
  "unknown_additive":{"safe":true},
  "candidates":[
    {"name":"zeta","kind":"hook","version":"1.0.0","compat":">=0.1.0 <1.0.0","digest":"sha256:z","source":"user-auto","required":["tool.write","tool.read"]},
    {"name":"alpha","kind":"tool","version":"1.0.0","compat":">=0.1.0 <1.0.0","digest":"sha256:a","source":"project-explicit"}
  ],
  "admission":{"runtime_version":"0.2.0","available_capabilities":["tool.write","tool.read","tool.read"],"best_effort":true},
  "lifecycle":{"action":"success","failure_policy":"deny"},
  "expected":{"status":"passed","phase":"complete","actual":"activated"}
}`)

	first, err := DecodeAuthoringCase(raw)
	if err != nil {
		t.Fatalf("DecodeAuthoringCase() error = %v", err)
	}
	second, err := DecodeAuthoringCase(raw)
	if err != nil {
		t.Fatalf("DecodeAuthoringCase() second error = %v", err)
	}
	if first.Candidates[0].Name != "alpha" || first.Candidates[1].Name != "zeta" {
		t.Fatalf("candidate order = %#v", first.Candidates)
	}
	wantCapabilities := []string{"tool.read", "tool.write"}
	if got := first.Admission.AvailableCapabilities; strings.Join(got, ",") != strings.Join(wantCapabilities, ",") {
		t.Fatalf("capabilities = %#v, want %#v", got, wantCapabilities)
	}
	firstJSON, _ := json.Marshal(first)
	secondJSON, _ := json.Marshal(second)
	if string(firstJSON) != string(secondJSON) {
		t.Fatalf("canonical decode is not stable:\n%s\n%s", firstJSON, secondJSON)
	}
}

func TestDecodeAuthoringCaseRejectsMalformedInputBeforeEvaluation(t *testing.T) {
	cases := []struct {
		name   string
		raw    string
		reason string
		field  string
	}{
		{name: "unsupported profile", raw: `{"profile":"external_extension_authoring.v2","case":"x","candidates":[{}],"expected":{"status":"failed","phase":"schema"}}`, reason: ReasonAuthoringUnsupportedProfile, field: "profile"},
		{name: "missing case", raw: `{"profile":"external_extension_authoring.v1","candidates":[{}],"expected":{"status":"failed","phase":"schema"}}`, reason: ReasonAuthoringMissingField, field: "case"},
		{name: "unknown action", raw: `{"profile":"external_extension_authoring.v1","case":"x","candidates":[{"name":"lint","kind":"hook","version":"1.0.0","compat":">=0.1.0 <1.0.0","digest":"sha256:a","source":"project-explicit"}],"lifecycle":{"action":"download"},"expected":{"status":"failed","phase":"schema"}}`, reason: ReasonAuthoringInvalidField, field: "lifecycle.action"},
		{name: "missing expected status", raw: `{"profile":"external_extension_authoring.v1","case":"x","candidates":[{"name":"lint","kind":"hook","version":"1.0.0","compat":">=0.1.0 <1.0.0","digest":"sha256:a","source":"project-explicit"}],"expected":{"phase":"complete"}}`, reason: ReasonAuthoringMissingField, field: "expected.status"},
		{name: "unknown expected phase", raw: `{"profile":"external_extension_authoring.v1","case":"x","candidates":[{"name":"lint","kind":"hook","version":"1.0.0","compat":">=0.1.0 <1.0.0","digest":"sha256:a","source":"project-explicit"}],"expected":{"status":"passed","phase":"downloaded"}}`, reason: ReasonAuthoringInvalidField, field: "expected.phase"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DecodeAuthoringCase([]byte(tc.raw))
			if err == nil {
				t.Fatal("expected validation error")
			}
			if got := AuthoringErrorReason(err); got != tc.reason {
				t.Fatalf("reason = %q, want %q", got, tc.reason)
			}
			if got := AuthoringErrorField(err); got != tc.field {
				t.Fatalf("field = %q, want %q", got, tc.field)
			}
		})
	}
}

func TestAuthoringRemediationCoversCanonicalReasonsAndIsBounded(t *testing.T) {
	reasons := []string{
		ReasonAuthoringMissingField,
		ReasonAuthoringInvalidField,
		ReasonAuthoringUnsupportedProfile,
		ReasonMissingField,
		ReasonInvalidField,
		ReasonAmbiguousConflict,
		ReasonCompatibilityMismatch,
		"adapter.capability.missing_required",
		"adapter.capability.optional_downgraded",
		ReasonTimeout,
		ReasonPanic,
		ReasonInvalidResult,
		ReasonExecutionFailed,
		ReasonAuthoringFinalizeFailed,
		ReasonAuthoringReloadFailed,
		ReasonAuthoringStaleGeneration,
		ReasonAuthoringPolicyDenied,
		ReasonAuthoringSandboxDenied,
		ReasonAuthoringAllowlistDenied,
		ReasonAuthoringEgressDenied,
		ReasonAuthoringReplayDrift,
	}
	for _, reason := range reasons {
		guidance := AuthoringRemediation(reason)
		if guidance == "" {
			t.Fatalf("missing remediation for %q", reason)
		}
		if len(guidance) > MaxAuthoringTextBytes {
			t.Fatalf("remediation for %q exceeds bound: %d", reason, len(guidance))
		}
	}
	if got := AuthoringRemediation("extension.unknown"); got != "review the conformance reason and owning contract" {
		t.Fatalf("unknown remediation = %q", got)
	}
}

func TestAuthoringResultIsBoundedAndDoesNotExposePayloads(t *testing.T) {
	long := strings.Repeat("secret", 100)
	result := NewAuthoringResult(AuthoringStatusFailed, AuthoringPhaseManifest, ReasonMissingField, long, long, long)
	if len(result.Field) > MaxAuthoringFieldBytes || len(result.Expected) > MaxAuthoringTextBytes || len(result.Actual) > MaxAuthoringTextBytes || len(result.Remediation) > MaxAuthoringTextBytes {
		t.Fatalf("result is not bounded: %#v", result)
	}
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"source_code", "credentials", "reasoning", "workspace", "payload"} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatalf("result leaked forbidden field %q: %s", forbidden, raw)
		}
	}
}

func TestEvaluateAuthoringCaseUsesGovernanceAndReportsSuccess(t *testing.T) {
	raw := []byte(`{"profile":"external_extension_authoring.v1","case":"activate","candidates":[{"name":"lint","kind":"hook","version":"1.0.0","compat":">=0.1.0 <1.0.0","digest":"sha256:a","source":"project-explicit","required":["tool.read"]}],"admission":{"runtime_version":"0.2.0","available_capabilities":["tool.read"]},"lifecycle":{"action":"success","failure_policy":"deny"},"expected":{"status":"passed","phase":"complete","actual":"activated"}}`)
	result, err := EvaluateAuthoringCase(raw)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != AuthoringStatusPassed || result.Phase != AuthoringPhaseComplete || result.Actual != "activated" {
		t.Fatalf("result=%#v", result)
	}
}

func TestEvaluateAuthoringCaseRejectsRequiredCapabilityWithoutActivation(t *testing.T) {
	raw := []byte(`{"profile":"external_extension_authoring.v1","case":"missing-capability","candidates":[{"name":"lint","kind":"hook","version":"1.0.0","compat":">=0.1.0 <1.0.0","digest":"sha256:a","source":"project-explicit","required":["tool.write"]}],"admission":{"runtime_version":"0.2.0","available_capabilities":["tool.read"]},"lifecycle":{"action":"success","failure_policy":"deny"},"expected":{"status":"failed","phase":"admission"}}`)
	result, err := EvaluateAuthoringCase(raw)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != AuthoringStatusFailed || result.Phase != AuthoringPhaseAdmission || result.ReasonCode != "adapter.capability.missing_required" || result.Actual != "inactive" {
		t.Fatalf("result=%#v", result)
	}
}

func TestEvaluateAuthoringCaseRejectsEqualPrecedenceConflict(t *testing.T) {
	raw := []byte(`{"profile":"external_extension_authoring.v1","case":"conflict","candidates":[{"name":"lint","kind":"hook","version":"1.0.0","compat":">=0.1.0 <1.0.0","digest":"sha256:a","source":"project-explicit"},{"name":"lint","kind":"hook","version":"1.0.0","compat":">=0.1.0 <1.0.0","digest":"sha256:b","source":"project-explicit"}],"admission":{"runtime_version":"0.2.0"},"lifecycle":{"action":"success"},"expected":{"status":"failed","phase":"admission"}}`)
	result, err := EvaluateAuthoringCase(raw)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != AuthoringStatusFailed || result.ReasonCode != ReasonAmbiguousConflict || result.Actual != "inactive" {
		t.Fatalf("result=%#v", result)
	}
}

func TestEvaluateAuthoringCaseReportsManifestOwnerReason(t *testing.T) {
	raw := []byte(`{"profile":"external_extension_authoring.v1","case":"missing-digest","candidates":[{"name":"lint","kind":"hook","version":"1.0.0","compat":">=0.1.0 <1.0.0","source":"project-explicit"}],"admission":{"runtime_version":"0.2.0"},"expected":{"status":"failed","phase":"manifest"}}`)
	result, err := EvaluateAuthoringCase(raw)
	if err != nil {
		t.Fatal(err)
	}
	if result.Phase != AuthoringPhaseManifest || result.ReasonCode != ReasonMissingField || result.Field != "digest" || result.Actual != "inactive" {
		t.Fatalf("result=%#v", result)
	}
}

func TestEvaluateAuthoringCaseCannotBypassSecurityOwners(t *testing.T) {
	for _, tc := range []struct{ name, boundary, reason string }{
		{"policy", "policy", ReasonAuthoringPolicyDenied},
		{"sandbox", "sandbox", ReasonAuthoringSandboxDenied},
		{"allowlist", "allowlist", ReasonAuthoringAllowlistDenied},
		{"egress", "egress", ReasonAuthoringEgressDenied},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := []byte(`{"profile":"external_extension_authoring.v1","case":"blocked","candidates":[{"name":"lint","kind":"hook","version":"1.0.0","compat":">=0.1.0 <1.0.0","digest":"sha256:a","source":"project-explicit"}],"admission":{"runtime_version":"0.2.0"},"boundaries":{"` + tc.boundary + `":"deny"},"expected":{"status":"failed","phase":"admission"}}`)
			result, err := EvaluateAuthoringCase(raw)
			if err != nil {
				t.Fatal(err)
			}
			if result.ReasonCode != tc.reason || result.Actual != "inactive" {
				t.Fatalf("result=%#v", result)
			}
		})
	}
}

func TestAuthoringBoundaryValidationAndDenialUseCanonicalOrder(t *testing.T) {
	base := `"profile":"external_extension_authoring.v1","case":"blocked","candidates":[{"name":"lint","kind":"hook","version":"1.0.0","compat":">=0.1.0 <1.0.0","digest":"sha256:a","source":"project-explicit"}],"admission":{"runtime_version":"0.2.0"},"expected":{"status":"failed","phase":"admission"}`
	raw := []byte(`{` + base + `,"boundaries":{"policy":"deny","sandbox":"deny"}}`)
	result, err := EvaluateAuthoringCase(raw)
	if err != nil {
		t.Fatal(err)
	}
	if result.ReasonCode != ReasonAuthoringPolicyDenied || result.Field != "boundaries.policy" {
		t.Fatalf("result=%#v", result)
	}
	invalid := []byte(`{` + base + `,"boundaries":{"policy":"invalid","sandbox":"also-invalid"}}`)
	decoded, err := DecodeAuthoringCase(invalid)
	if err == nil || decoded.Profile != "" || len(decoded.Candidates) != 0 || AuthoringErrorField(err) != "boundaries.policy" {
		t.Fatalf("invalid boundary result=%#v err=%v", decoded, err)
	}
}

func TestEvaluateAuthoringCaseReloadFailureAndStaleGenerationAreIsolated(t *testing.T) {
	for _, tc := range []struct{ action, reason, actual string }{
		{"reload_failure", ReasonAuthoringReloadFailed, "previous_generation_active"},
		{"stale_generation", ReasonAuthoringStaleGeneration, "stale_event_suppressed"},
	} {
		raw := []byte(`{"profile":"external_extension_authoring.v1","case":"reload","candidates":[{"name":"lint","kind":"hook","version":"1.0.0","compat":">=0.1.0 <1.0.0","digest":"sha256:a","source":"project-explicit"}],"admission":{"runtime_version":"0.2.0"},"lifecycle":{"action":"` + tc.action + `","failure_policy":"deny"},"expected":{"status":"failed","phase":"reload"}}`)
		result, err := EvaluateAuthoringCase(raw)
		if err != nil {
			t.Fatal(err)
		}
		if result.Phase != AuthoringPhaseReload || result.ReasonCode != tc.reason || result.Actual != tc.actual {
			t.Fatalf("%s result=%#v", tc.action, result)
		}
	}
}

func TestEvaluateAuthoringProjectionValidatesModeAndPreservesParity(t *testing.T) {
	raw := []byte(`{"profile":"external_extension_authoring.v1","case":"activate","candidates":[{"name":"lint","kind":"hook","version":"1.0.0","compat":">=0.1.0 <1.0.0","digest":"sha256:a","source":"project-explicit"}],"admission":{"runtime_version":"0.2.0"},"expected":{"status":"passed","phase":"complete","actual":"activated"}}`)
	run, err := EvaluateAuthoringProjection(raw, AuthoringProjectionRun)
	if err != nil {
		t.Fatal(err)
	}
	stream, err := EvaluateAuthoringProjection(raw, AuthoringProjectionStream)
	if err != nil {
		t.Fatal(err)
	}
	if run != stream {
		t.Fatalf("run=%#v stream=%#v", run, stream)
	}
	if _, err := EvaluateAuthoringProjection(raw, "parallel-terminal"); err == nil || AuthoringErrorField(err) != "projection" {
		t.Fatalf("invalid projection err=%v", err)
	}
}
