package diagnosticsreplay

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
)

const sandboxLifecycleSuccessCostEvidenceFixturePath = "testdata/sandbox_lifecycle_success_cost_evidence.v1.json"

func TestReplaySandboxLifecycleSuccessCostEvidenceFixtureIsVersionedBoundedAndIdempotent(t *testing.T) {
	raw, err := os.ReadFile(sandboxLifecycleSuccessCostEvidenceFixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if len(raw) > sandboxLifecycleMaxFixtureBytes {
		t.Fatalf("fixture exceeds bound: %d", len(raw))
	}
	for _, forbidden := range []string{"endpoint", "sk-", "token", "raw_response", "raw_payload", "password", "secret", "command"} {
		if strings.Contains(strings.ToLower(string(raw)), forbidden) {
			t.Fatalf("fixture contains forbidden material %q", forbidden)
		}
	}
	first, err := ReplaySandboxLifecycleSuccessCostEvidenceJSON(raw)
	if err != nil {
		t.Fatalf("first replay: %v", err)
	}
	second, err := ReplaySandboxLifecycleSuccessCostEvidenceJSON(raw)
	if err != nil {
		t.Fatalf("second replay: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("replay is not idempotent: first=%#v second=%#v", first, second)
	}
	if len(first.Cases) != 5 {
		t.Fatalf("cases = %d, want 5", len(first.Cases))
	}
}

func TestReplaySandboxLifecycleSuccessCostEvidenceFixtureDetectsDrift(t *testing.T) {
	raw, err := os.ReadFile(sandboxLifecycleSuccessCostEvidenceFixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var fixture SandboxLifecycleSuccessCostEvidenceFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	fixture.Cases[0].Expected.Verdict = SandboxLifecycleSuccessCostVerdictGapConfirmed
	mutated, err := json.Marshal(fixture)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	if _, err := ReplaySandboxLifecycleSuccessCostEvidenceJSON(mutated); err == nil || !strings.Contains(err.Error(), SandboxLifecycleSuccessCostReasonBaselineDrift) {
		t.Fatalf("error = %v, want baseline drift", err)
	}
}

func TestReplaySandboxLifecycleSuccessCostEvidenceFixtureIgnoresUnknownFields(t *testing.T) {
	raw, err := os.ReadFile(sandboxLifecycleSuccessCostEvidenceFixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	document["unknown_future_field"] = map[string]any{"value": "ignored"}
	mutated, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	if _, err := ReplaySandboxLifecycleSuccessCostEvidenceJSON(mutated); err != nil {
		t.Fatalf("unknown additive fields should be ignored: %v", err)
	}
}

func TestReplaySandboxLifecycleSuccessCostEvidenceClassifiesRequiredDriftTaxonomy(t *testing.T) {
	base := `{"version":"sandbox_lifecycle_success_cost_evidence.v1","cases":[{"case_id":"case","input":{"invocation_ref":"i","session_ref":"s","backend":"b","profile":"p","session_mode":"per_call","phases":%s,"duration_bucket":"%s","resource_bucket":"small","retry_ordinal":%d,"terminal_outcome":"%s","success_classification":"%s","baseline":{"duration_bucket":"%s","resource_bucket":"small"},"run_stream_parity":%t}}]}`
	for name, fixture := range map[string]struct {
		fixture    string
		wantReason string
	}{
		"phase": {
			fixture:    `{"version":"sandbox_lifecycle_success_cost_evidence.v1","cases":[{"case_id":"phase","input":{"invocation_ref":"i","session_ref":"s","backend":"b","profile":"p","session_mode":"per_call","phases":["launch","acquire","execute","release"],"duration_bucket":"lt_1s","resource_bucket":"small","retry_ordinal":0,"terminal_outcome":"success","success_classification":"successful","baseline":{"duration_bucket":"lt_1s","resource_bucket":"small"},"run_stream_parity":true}}]}`,
			wantReason: SandboxLifecycleSuccessCostReasonPhaseDrift,
		},
		"continuity": {
			fixture:    `{"version":"sandbox_lifecycle_success_cost_evidence.v1","cases":[{"case_id":"continuity","input":{"invocation_ref":"i","session_ref":"s","backend":"b","profile":"p","session_mode":"per_call","phases":["acquire","launch","execute","release","release"],"duration_bucket":"lt_1s","resource_bucket":"small","retry_ordinal":0,"terminal_outcome":"success","success_classification":"successful","baseline":{"duration_bucket":"lt_1s","resource_bucket":"small"},"run_stream_parity":true}}]}`,
			wantReason: SandboxLifecycleSuccessCostReasonContinuityDrift,
		},
		"success-cost-bucket": {
			fixture:    sprintfSandboxLifecycleFixture(base, `["acquire","launch","execute","release"]`, "lt_1s", 0, "success", "successful", "gte_1s", true),
			wantReason: SandboxLifecycleSuccessCostReasonSuccessCostBucketDrift,
		},
		"baseline": {
			fixture:    sprintfSandboxLifecycleFixture(base, `["acquire","launch","execute","release"]`, "lt_1s", 0, "success", "successful", "gte_1s", true),
			wantReason: SandboxLifecycleSuccessCostReasonLifecycleBaselineDrift,
		},
		"terminal": {
			fixture:    sprintfSandboxLifecycleFixture(base, `["acquire","launch","execute","release"]`, "lt_1s", 0, "failure", "failed", "lt_1s", true),
			wantReason: SandboxLifecycleSuccessCostReasonTerminalOutcomeDrift,
		},
		"parity": {
			fixture:    sprintfSandboxLifecycleFixture(base, `["acquire","launch","execute","release"]`, "lt_1s", 0, "success", "successful", "lt_1s", false),
			wantReason: SandboxLifecycleSuccessCostReasonRunStreamParityDrift,
		},
	} {
		t.Run(name, func(t *testing.T) {
			result, err := ReplaySandboxLifecycleSuccessCostEvidenceJSON([]byte(fixture.fixture))
			if err != nil {
				t.Fatalf("replay: %v", err)
			}
			if !containsSandboxLifecycleReason(result.Cases[0].Reasons, fixture.wantReason) {
				t.Fatalf("reasons = %#v, want %q", result.Cases[0].Reasons, fixture.wantReason)
			}
		})
	}

	_, err := ReplaySandboxLifecycleSuccessCostEvidenceJSON([]byte(`{"version":"sandbox_lifecycle_success_cost_evidence.v1","cases":[{"case_id":"privacy","input":{"command":"forbidden"}}]}`))
	if err == nil || !strings.Contains(err.Error(), SandboxLifecycleSuccessCostReasonPrivacyDrift) {
		t.Fatalf("privacy error = %v, want %q", err, SandboxLifecycleSuccessCostReasonPrivacyDrift)
	}
}

func sprintfSandboxLifecycleFixture(format, phases, duration string, retry int, terminal, classification, baseline string, parity bool) string {
	return fmt.Sprintf(format, phases, duration, retry, terminal, classification, baseline, parity)
}

func containsSandboxLifecycleReason(reasons []string, want string) bool {
	for _, reason := range reasons {
		if reason == want {
			return true
		}
	}
	return false
}
