package diagnosticsreplay

import (
	"bytes"
	"testing"
)

func TestEvaluateExternalExtensionAuthoringFixtureIsCanonicalAndIdempotent(t *testing.T) {
	raw := []byte(`{"profile":"external_extension_authoring.v1","case":"valid","candidates":[{"name":"lint","kind":"hook","version":"1.0.0","compat":">=0.1.0 <1.0.0","digest":"sha256:a","source":"project-explicit"}],"admission":{"runtime_version":"0.2.0"},"expected":{"status":"passed","phase":"complete","actual":"activated"}}`)
	first, err := EvaluateExternalExtensionAuthoringFixture(raw)
	if err != nil {
		t.Fatal(err)
	}
	second, err := EvaluateExternalExtensionAuthoringFixture(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("replay drift:\n%s\n%s", first, second)
	}
	if !bytes.Contains(first, []byte(`"actual":"activated"`)) {
		t.Fatalf("canonical output=%s", first)
	}
}

func TestEvaluateExternalExtensionAuthoringFixtureClassifiesDriftWithoutRewritingExpected(t *testing.T) {
	raw := []byte(`{"profile":"external_extension_authoring.v1","case":"drift","candidates":[{"name":"lint","kind":"hook","version":"1.0.0","compat":">=0.1.0 <1.0.0","digest":"sha256:a","source":"project-explicit"}],"admission":{"runtime_version":"0.2.0"},"expected":{"status":"failed","phase":"admission","actual":"inactive"}}`)
	before := append([]byte(nil), raw...)
	_, err := EvaluateExternalExtensionAuthoringFixture(raw)
	if err == nil || ReplayReasonCode(err) != "authoring_expected_actual_drift" {
		t.Fatalf("err=%v code=%q", err, ReplayReasonCode(err))
	}
	if !bytes.Equal(raw, before) {
		t.Fatal("replay mutated expected fixture")
	}
}
