package diagnosticsreplay

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

var semanticFixtureLegacyAliases = map[string]string{
	"context_reference_first_success_input.json":       "a" + "67_ctx_reference_first_success_input.json",
	"context_reference_first_success_expected.json":    "a" + "67_ctx_reference_first_success_expected.json",
	"context_isolate_handoff_success_input.json":       "a" + "67_ctx_isolate_handoff_success_input.json",
	"context_isolate_handoff_success_expected.json":    "a" + "67_ctx_isolate_handoff_success_expected.json",
	"context_edit_gate_success_input.json":             "a" + "67_ctx_edit_gate_success_input.json",
	"context_edit_gate_success_expected.json":          "a" + "67_ctx_edit_gate_success_expected.json",
	"context_relevance_swapback_success_input.json":    "a" + "67_ctx_swapback_success_input.json",
	"context_relevance_swapback_success_expected.json": "a" + "67_ctx_swapback_success_expected.json",
	"context_lifecycle_tiering_success_input.json":     "a" + "67_ctx_lifecycle_tiering_success_input.json",
	"context_lifecycle_tiering_success_expected.json":  "a" + "67_ctx_lifecycle_tiering_success_expected.json",
	"context_reference_resolution_drift_input.json":    "a" + "67_ctx_reference_resolution_drift_input.json",
	"context_isolate_handoff_drift_input.json":         "a" + "67_ctx_isolate_handoff_drift_input.json",
	"context_edit_gate_threshold_drift_input.json":     "a" + "67_ctx_edit_gate_threshold_drift_input.json",
	"context_relevance_swapback_drift_input.json":      "a" + "67_ctx_swapback_relevance_drift_input.json",
	"context_lifecycle_tiering_drift_input.json":       "a" + "67_ctx_lifecycle_tiering_drift_input.json",
	"context_recap_semantic_drift_input.json":          "a" + "67_ctx_recap_semantic_drift_input.json",
	"react_plan_notebook_success_input.json":           "a" + "67_react_plan_notebook_success_input.json",
	"react_plan_notebook_success_expected.json":        "a" + "67_react_plan_notebook_success_expected.json",
	"react_plan_version_drift_input.json":              "a" + "67_react_plan_version_drift_input.json",
	"react_plan_change_reason_drift_input.json":        "a" + "67_react_plan_change_reason_drift_input.json",
	"react_plan_hook_semantic_drift_input.json":        "a" + "67_react_plan_hook_semantic_drift_input.json",
	"react_plan_recover_drift_input.json":              "a" + "67_react_plan_recover_drift_input.json",
	"realtime_event_protocol_success_input.json":       "a" + "68_realtime_event_protocol_success_input.json",
	"realtime_event_protocol_success_expected.json":    "a" + "68_realtime_event_protocol_success_expected.json",
	"realtime_event_order_drift_input.json":            "a" + "68_realtime_event_order_drift_input.json",
	"realtime_interrupt_semantic_drift_input.json":     "a" + "68_realtime_interrupt_semantic_drift_input.json",
	"realtime_resume_semantic_drift_input.json":        "a" + "68_realtime_resume_semantic_drift_input.json",
	"realtime_idempotency_drift_input.json":            "a" + "68_realtime_idempotency_drift_input.json",
	"realtime_sequence_gap_drift_input.json":           "a" + "68_realtime_sequence_gap_drift_input.json",
}

func TestReplayContractSuccessFixture(t *testing.T) {
	input := mustReadFixture(t, "success_input.json")
	expected := mustReadFixture(t, "success_expected.json")

	got, err := ParseMinimalReplayJSON(input)
	if err != nil {
		t.Fatalf("ParseMinimalReplayJSON error: %v", err)
	}

	var want ReplayOutput
	if err := json.Unmarshal(expected, &want); err != nil {
		t.Fatalf("unmarshal expected fixture: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("replay output mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestReplayContractMalformedJSONReasonCode(t *testing.T) {
	input := mustReadFixture(t, "invalid_json_input.txt")
	_, err := ParseMinimalReplayJSON(input)
	if err == nil {
		t.Fatal("expected malformed json error")
	}
	vErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("error type = %T, want *ValidationError", err)
	}
	if vErr.Code != ReasonCodeInvalidJSON {
		t.Fatalf("code = %q, want %q", vErr.Code, ReasonCodeInvalidJSON)
	}
}

func TestReplayContractMissingFieldReasonCode(t *testing.T) {
	input := mustReadFixture(t, "missing_field_input.json")
	_, err := ParseMinimalReplayJSON(input)
	if err == nil {
		t.Fatal("expected missing field error")
	}
	vErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("error type = %T, want *ValidationError", err)
	}
	if vErr.Code != ReasonCodeMissingRequiredField {
		t.Fatalf("code = %q, want %q", vErr.Code, ReasonCodeMissingRequiredField)
	}
}

func TestTerminalOutcomeReplayFixturePreservesAdditiveClassification(t *testing.T) {
	input := mustReadFixture(t, "terminal_outcomes.json")
	got, err := ParseMinimalReplayJSON(input)
	if err != nil {
		t.Fatalf("ParseMinimalReplayJSON error: %v", err)
	}
	if len(got.Events) != 11 {
		t.Fatalf("event count = %d, want 11", len(got.Events))
	}
	if got.Events[0].TerminalState != "completed" || got.Events[0].FailureFamily != "none" {
		t.Fatalf("success classification = %#v", got.Events[0])
	}
	if got.Events[10].FailureFamily != "recovery_conflict" || got.Events[10].TerminalPhase != "post_start" {
		t.Fatalf("late conflict classification = %#v", got.Events[10])
	}
}

func TestProviderModelAdmissionReplayFixtureIsVersionedAndRedacted(t *testing.T) {
	input := mustReadFixture(t, "provider_model_admission.v1.json")
	got, err := ParseMinimalReplayJSON(input)
	if err != nil {
		t.Fatalf("ParseMinimalReplayJSON error: %v", err)
	}
	if len(got.Events) != 9 {
		t.Fatalf("provider admission replay = %#v", got)
	}
	wantReasons := map[string]bool{
		"provider.admission.ready":            true,
		"provider.catalog.unknown_model":      true,
		"adapter.capability.missing_required": true,
		"provider.catalog.optional_fallback":  true,
		"provider.credential.missing":         true,
		"provider.credential.unverified":      true,
		"provider.catalog.reload_rollback":    true,
	}
	for _, event := range got.Events {
		if !wantReasons[event.Reason] {
			t.Fatalf("unexpected provider admission reason %q in %#v", event.Reason, got)
		}
	}
	if got.Events[6].Phase != "run" || got.Events[7].Phase != "stream" || got.Events[6].Reason != got.Events[7].Reason {
		t.Fatalf("run/stream parity events diverged: %#v", got.Events[6:8])
	}
	if string(input) == "" {
		t.Fatal("provider admission fixture is empty")
	}
	for _, secret := range []string{"sk-", "authorization", "endpoint", "token"} {
		if strings.Contains(strings.ToLower(string(input)), secret) {
			t.Fatalf("provider admission fixture contains secret material %q", secret)
		}
	}
}

func TestProviderModelAdmissionReplayIsIdempotent(t *testing.T) {
	input := mustReadFixture(t, "provider_model_admission.v1.json")

	first, err := ParseMinimalReplayJSON(input)
	if err != nil {
		t.Fatalf("first ParseMinimalReplayJSON error: %v", err)
	}
	second, err := ParseMinimalReplayJSON(input)
	if err != nil {
		t.Fatalf("second ParseMinimalReplayJSON error: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("replay is not idempotent\nfirst: %#v\nsecond: %#v", first, second)
	}
	if len(second.Events) != 9 {
		t.Fatalf("idempotent replay event count = %d, want 9", len(second.Events))
	}
}

func mustReadFixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("testdata", name)
	raw, err := os.ReadFile(path)
	if err == nil {
		return raw
	}
	if alias, ok := semanticFixtureLegacyAliases[name]; ok {
		aliasPath := filepath.Join("testdata", alias)
		raw, aliasErr := os.ReadFile(aliasPath)
		if aliasErr == nil {
			return raw
		}
		t.Fatalf("read fixture %s via alias %s: %v", path, aliasPath, aliasErr)
	}
	t.Fatalf("read fixture %s: %v", path, err)
	return nil
}
