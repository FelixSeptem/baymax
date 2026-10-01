package diagnosticsreplay

import (
	"strings"
	"testing"
)

func TestReplaySandboxLifecycleSuccessCostEvidenceValidatesBoundedPrivacySafeEvidence(t *testing.T) {
	valid := []byte(`{
		"version":"sandbox_lifecycle_success_cost_evidence.v1",
		"cases":[{
			"case_id":"cold-launch-success",
			"input":{
				"invocation_ref":"invocation-1",
				"session_ref":"session-1",
				"backend":"host-sandbox",
				"profile":"restricted",
				"session_mode":"per_call",
				"phases":["acquire","launch","execute","release"],
				"duration_bucket":"lt_1s",
				"resource_bucket":"small",
				"retry_ordinal":0,
				"terminal_outcome":"success",
				"success_classification":"successful",
				"baseline":{"duration_bucket":"lt_1s","resource_bucket":"small"},
				"run_stream_parity":true
			},
			"expected":{"verdict":"within-baseline","lifecycle_kind":"cold_launch","has_success_cost":true,"run_stream_parity":true}
		}]
	}`)

	first, err := ReplaySandboxLifecycleSuccessCostEvidenceJSON(valid)
	if err != nil {
		t.Fatalf("replay valid evidence: %v", err)
	}
	second, err := ReplaySandboxLifecycleSuccessCostEvidenceJSON(valid)
	if err != nil {
		t.Fatalf("replay valid evidence again: %v", err)
	}
	if len(first.Cases) != 1 || first.Cases[0].Verdict != SandboxLifecycleSuccessCostVerdictWithinBaseline {
		t.Fatalf("result = %#v, want one within-baseline case", first)
	}
	if first.Cases[0].LifecycleKind != SandboxLifecycleKindColdLaunch || !first.Cases[0].HasSuccessCost || !first.Cases[0].RunStreamParity {
		t.Fatalf("projection = %#v, want cold-launch successful parity projection", first.Cases[0])
	}
	if first.Cases[0].Digest == "" || first.Cases[0].Digest != second.Cases[0].Digest || first.Cases[0].ReplayDigest != second.Cases[0].Digest || !first.Cases[0].Idempotent {
		t.Fatalf("digest/idempotence = %#v / %#v", first.Cases[0], second.Cases[0])
	}

	for name, raw := range map[string][]byte{
		"prohibited-field": []byte(`{"version":"sandbox_lifecycle_success_cost_evidence.v1","cases":[{"case_id":"privacy","input":{"command":"uname"}}]}`),
		"invalid-enum":     []byte(`{"version":"sandbox_lifecycle_success_cost_evidence.v1","cases":[{"case_id":"enum","input":{"invocation_ref":"i","session_ref":"s","backend":"b","profile":"p","session_mode":"invalid"}}]}`),
		"oversized-field":  []byte(`{"version":"sandbox_lifecycle_success_cost_evidence.v1","cases":[{"case_id":"oversized","input":{"invocation_ref":"` + strings.Repeat("x", 257) + `"}}]}`),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := ReplaySandboxLifecycleSuccessCostEvidenceJSON(raw)
			if err == nil || !strings.Contains(err.Error(), SandboxLifecycleSuccessCostReasonPrivacyOrBoundViolation) && !strings.Contains(err.Error(), SandboxLifecycleSuccessCostReasonPrivacyDrift) && !strings.Contains(err.Error(), SandboxLifecycleSuccessCostReasonSchemaDrift) {
				t.Fatalf("error = %v, want deterministic validation failure", err)
			}
		})
	}
}

func TestReplaySandboxLifecycleSuccessCostEvidenceClassifiesLifecycleContinuity(t *testing.T) {
	for name, fixture := range map[string]struct {
		fixture     string
		wantVerdict string
		wantKind    string
		wantCost    bool
	}{
		"per-session-reuse": {
			fixture:     `{"version":"sandbox_lifecycle_success_cost_evidence.v1","cases":[{"case_id":"reuse","input":{"invocation_ref":"i","session_ref":"s","backend":"b","profile":"p","session_mode":"per_session","phases":["execute","release"],"duration_bucket":"lt_1s","resource_bucket":"small","retry_ordinal":0,"terminal_outcome":"success","success_classification":"successful","baseline":{"duration_bucket":"lt_1s","resource_bucket":"small"},"run_stream_parity":true}}]}`,
			wantVerdict: SandboxLifecycleSuccessCostVerdictWithinBaseline,
			wantKind:    SandboxLifecycleKindReuse,
			wantCost:    true,
		},
		"recovery-after-recoverable-failure": {
			fixture:     `{"version":"sandbox_lifecycle_success_cost_evidence.v1","cases":[{"case_id":"recover","input":{"invocation_ref":"i","session_ref":"s","backend":"b","profile":"p","session_mode":"per_session","phases":["acquire","launch","execute","retry","recover","execute","release"],"duration_bucket":"lt_1s","resource_bucket":"small","retry_ordinal":1,"reason_code":"recoverable_failure","terminal_outcome":"success","success_classification":"successful","baseline":{"duration_bucket":"lt_1s","resource_bucket":"small"},"run_stream_parity":true}}]}`,
			wantVerdict: SandboxLifecycleSuccessCostVerdictWithinBaseline,
			wantKind:    SandboxLifecycleKindRecovery,
			wantCost:    true,
		},
		"recovery-without-prior-failure": {
			fixture:     `{"version":"sandbox_lifecycle_success_cost_evidence.v1","cases":[{"case_id":"invalid-recover","input":{"invocation_ref":"i","session_ref":"s","backend":"b","profile":"p","session_mode":"per_session","phases":["acquire","launch","execute","recover","release"],"duration_bucket":"lt_1s","resource_bucket":"small","retry_ordinal":0,"terminal_outcome":"success","success_classification":"successful","baseline":{"duration_bucket":"lt_1s","resource_bucket":"small"},"run_stream_parity":true}}]}`,
			wantVerdict: SandboxLifecycleSuccessCostVerdictGapConfirmed,
			wantKind:    SandboxLifecycleKindRecovery,
			wantCost:    false,
		},
		"duplicate-release": {
			fixture:     `{"version":"sandbox_lifecycle_success_cost_evidence.v1","cases":[{"case_id":"duplicate-release","input":{"invocation_ref":"i","session_ref":"s","backend":"b","profile":"p","session_mode":"per_call","phases":["acquire","launch","execute","release","release"],"duration_bucket":"lt_1s","resource_bucket":"small","retry_ordinal":0,"terminal_outcome":"success","success_classification":"successful","baseline":{"duration_bucket":"lt_1s","resource_bucket":"small"},"run_stream_parity":true}}]}`,
			wantVerdict: SandboxLifecycleSuccessCostVerdictGapConfirmed,
			wantKind:    SandboxLifecycleKindColdLaunch,
			wantCost:    false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			result, err := ReplaySandboxLifecycleSuccessCostEvidenceJSON([]byte(fixture.fixture))
			if err != nil {
				t.Fatalf("replay: %v", err)
			}
			caseResult := result.Cases[0]
			if caseResult.Verdict != fixture.wantVerdict || caseResult.LifecycleKind != fixture.wantKind || caseResult.HasSuccessCost != fixture.wantCost {
				t.Fatalf("result = %#v, want verdict=%q kind=%q success_cost=%t", caseResult, fixture.wantVerdict, fixture.wantKind, fixture.wantCost)
			}
		})
	}
}

func TestReplaySandboxLifecycleSuccessCostEvidenceCountsOnlyTerminalSuccess(t *testing.T) {
	for name, fixture := range map[string]struct {
		terminalOutcome  string
		classification   string
		baselineDuration string
		wantVerdict      string
		wantCost         bool
	}{
		"failed":            {"failure", "failed", "lt_1s", SandboxLifecycleSuccessCostVerdictInsufficient, false},
		"unknown":           {"unknown", "unknown", "lt_1s", SandboxLifecycleSuccessCostVerdictInsufficient, false},
		"contradictory":     {"success", "failed", "lt_1s", SandboxLifecycleSuccessCostVerdictInsufficient, false},
		"baseline-mismatch": {"success", "successful", "gte_1s", SandboxLifecycleSuccessCostVerdictGapConfirmed, true},
	} {
		t.Run(name, func(t *testing.T) {
			raw := []byte(`{"version":"sandbox_lifecycle_success_cost_evidence.v1","cases":[{"case_id":"` + name + `","input":{"invocation_ref":"i","session_ref":"s","backend":"b","profile":"p","session_mode":"per_call","phases":["acquire","launch","execute","release"],"duration_bucket":"lt_1s","resource_bucket":"small","retry_ordinal":0,"terminal_outcome":"` + fixture.terminalOutcome + `","success_classification":"` + fixture.classification + `","baseline":{"duration_bucket":"` + fixture.baselineDuration + `","resource_bucket":"small"},"run_stream_parity":true}}]}`)
			result, err := ReplaySandboxLifecycleSuccessCostEvidenceJSON(raw)
			if err != nil {
				t.Fatalf("replay: %v", err)
			}
			caseResult := result.Cases[0]
			if caseResult.Verdict != fixture.wantVerdict || caseResult.HasSuccessCost != fixture.wantCost {
				t.Fatalf("result = %#v, want verdict=%q success_cost=%t", caseResult, fixture.wantVerdict, fixture.wantCost)
			}
		})
	}
}
