package diagnosticsreplay

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
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
