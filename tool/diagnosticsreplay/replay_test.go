package diagnosticsreplay

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

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
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	return raw
}
