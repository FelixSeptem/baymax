package diagnosticsreplay

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	runtimediag "github.com/FelixSeptem/baymax/runtime/diagnostics"
)

func TestParseToolLifecycleReplayJSONNormalizesStagesAndOrder(t *testing.T) {
	raw := []byte(`{"version":"tool_lifecycle_failure_isolation.v1","calls":[{"call_id":"b","tool_name":"local.echo","input_index":1,"stages":[{"stage":"prepare","outcome":"succeeded"},{"stage":"validate","outcome":"succeeded"},{"stage":"authorize","outcome":"not_applicable"},{"stage":"execute","outcome":"succeeded"},{"stage":"finalize","outcome":"succeeded"}]},{"call_id":"a","tool_name":"local.echo","input_index":0,"stages":[{"stage":"prepare","outcome":"succeeded"},{"stage":"validate","outcome":"failed"},{"stage":"authorize","outcome":"skipped"},{"stage":"execute","outcome":"skipped"},{"stage":"finalize","outcome":"succeeded"}],"failure_origin":"validation"}]}`)
	out, err := ParseToolLifecycleReplayJSON(raw)
	if err != nil {
		t.Fatalf("ParseToolLifecycleReplayJSON() error = %v", err)
	}
	if len(out.Calls) != 2 || out.Calls[0].CallID != "a" || out.Calls[1].CallID != "b" {
		t.Fatalf("calls = %#v", out.Calls)
	}
}

func TestParseToolLifecycleReplayJSONRejectsStageDrift(t *testing.T) {
	raw := []byte(`{"version":"tool_lifecycle_failure_isolation.v1","calls":[{"call_id":"a","tool_name":"local.echo","stages":[{"stage":"validate","outcome":"succeeded"}]}]}`)
	_, err := ParseToolLifecycleReplayJSON(raw)
	var validationErr *ValidationError
	if err == nil || !errors.As(err, &validationErr) || validationErr.Code != "lifecycle_stage_order_drift" {
		t.Fatalf("error = %v, want lifecycle_stage_order_drift", err)
	}
}

func TestParseToolLifecycleReplayJSONRejectsUnsupportedVersionAndDuplicateCallID(t *testing.T) {
	_, err := ParseToolLifecycleReplayJSON([]byte(`{"version":"old","calls":[]}`))
	var validationErr *ValidationError
	if err == nil || !errors.As(err, &validationErr) || validationErr.Code != "unsupported_version" {
		t.Fatalf("error = %v, want unsupported_version", err)
	}
	duplicate := `{"version":"tool_lifecycle_failure_isolation.v1","calls":[{"call_id":"a","tool_name":"local.echo","stages":[{"stage":"prepare","outcome":"succeeded"},{"stage":"validate","outcome":"succeeded"},{"stage":"authorize","outcome":"not_applicable"},{"stage":"execute","outcome":"succeeded"},{"stage":"finalize","outcome":"succeeded"}]},{"call_id":"a","tool_name":"local.echo","stages":[{"stage":"prepare","outcome":"succeeded"},{"stage":"validate","outcome":"succeeded"},{"stage":"authorize","outcome":"not_applicable"},{"stage":"execute","outcome":"succeeded"},{"stage":"finalize","outcome":"succeeded"}]}]}`
	_, err = ParseToolLifecycleReplayJSON([]byte(duplicate))
	if err == nil || !errors.As(err, &validationErr) || validationErr.Code != "duplicate_call_id" {
		t.Fatalf("error = %v, want duplicate_call_id", err)
	}
}

func TestParseToolLifecycleReplayJSONRejectsFinalizeDrift(t *testing.T) {
	raw := []byte(`{"version":"tool_lifecycle_failure_isolation.v1","calls":[{"call_id":"a","tool_name":"local.echo","finalized":true,"stages":[{"stage":"prepare","outcome":"succeeded"},{"stage":"validate","outcome":"succeeded"},{"stage":"authorize","outcome":"not_applicable"},{"stage":"execute","outcome":"failed"},{"stage":"finalize","outcome":"failed"}]}]}`)
	_, err := ParseToolLifecycleReplayJSON(raw)
	var validationErr *ValidationError
	if err == nil || !errors.As(err, &validationErr) || validationErr.Code != "duplicate_finalize" {
		t.Fatalf("error = %v, want duplicate_finalize", err)
	}
}

func TestParseToolLifecycleReplayJSONRejectsFailureOriginFinalizeAndHostedOwnershipDrift(t *testing.T) {
	base := `{"version":"tool_lifecycle_failure_isolation.v1","calls":[{"call_id":"a","tool_name":"local.echo",%s,"stages":[{"stage":"prepare","outcome":"succeeded"},{"stage":"validate","outcome":"succeeded"},{"stage":"authorize","outcome":"not_applicable"},{"stage":"execute","outcome":"succeeded"},{"stage":"finalize","outcome":"succeeded"}]}]}`
	for _, test := range []struct{ field, code string }{
		{`"failure_origin":"unbounded"`, "failure_origin_drift"},
		{`"finalize_count":2`, "duplicate_finalize"},
		{`"owner":"hosted_executor"`, "hosted_ownership"},
	} {
		_, err := ParseToolLifecycleReplayJSON([]byte(fmt.Sprintf(base, test.field)))
		var validationErr *ValidationError
		if err == nil || !errors.As(err, &validationErr) || validationErr.Code != test.code {
			t.Fatalf("field %q error = %v, want %s", test.field, err, test.code)
		}
	}
}

func TestCanonicalToolLifecycleFixtureIsDeterministicAndComplete(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "tool_lifecycle_failure_isolation.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	first, err := ParseToolLifecycleReplayJSON(raw)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	second, err := ParseToolLifecycleReplayJSON(raw)
	if err != nil {
		t.Fatalf("parse fixture second time: %v", err)
	}
	one, _ := json.Marshal(first)
	two, _ := json.Marshal(second)
	if string(one) != string(two) {
		t.Fatalf("fixture normalization is not deterministic")
	}
	if len(first.Calls) < 10 || first.Calls[0].InputIndex != 0 || first.Calls[len(first.Calls)-1].InputIndex != 10 {
		t.Fatalf("fixture calls = %#v", first.Calls)
	}
}

func TestLegacyDiagnosticsRecordRemainsCompatibleWithLifecycleReplay(t *testing.T) {
	legacy := []byte(`{"time":"2026-09-03T00:00:00Z","component":"tool","call_id":"legacy","name":"local.echo","latency_ms":4,"retry_count":0}`)
	var record runtimediag.CallRecord
	if err := json.Unmarshal(legacy, &record); err != nil {
		t.Fatalf("legacy diagnostics record must remain readable: %v", err)
	}
	if record.CallID != "legacy" || record.LifecycleStage != "" || record.AttemptCount != 0 {
		t.Fatalf("legacy record compatibility = %#v", record)
	}
	raw, err := os.ReadFile(filepath.Join("testdata", "tool_lifecycle_failure_isolation.json"))
	if err != nil {
		t.Fatalf("read lifecycle fixture: %v", err)
	}
	if _, err := ParseToolLifecycleReplayJSON(raw); err != nil {
		t.Fatalf("lifecycle replay should remain readable alongside legacy records: %v", err)
	}
}
