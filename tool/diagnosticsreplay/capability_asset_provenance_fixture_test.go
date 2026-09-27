package diagnosticsreplay

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

const capabilityAssetProvenanceFixturePath = "testdata/capability_asset_provenance.v1.json"

func TestCapabilityAssetProvenanceFixtureIsBoundedDeterministicAndCovered(t *testing.T) {
	raw, err := os.ReadFile(capabilityAssetProvenanceFixturePath)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.ToLower(string(raw))
	for _, marker := range []string{"credential", "password", "reasoning", "raw_payload", "workspace_content"} {
		if strings.Contains(text, marker) {
			t.Fatalf("fixture contains sensitive marker %q", marker)
		}
	}
	first, err := ReplayCapabilityAssetProvenanceJSON(raw)
	if err != nil {
		t.Fatalf("first replay: %v", err)
	}
	second, err := ReplayCapabilityAssetProvenanceJSON(raw)
	if err != nil {
		t.Fatalf("second replay: %v", err)
	}
	if len(first.Cases) < 8 {
		t.Fatalf("fixture coverage = %d, want at least 8 cases", len(first.Cases))
	}
	if !reflectCapabilityResultEqual(first, second) {
		t.Fatalf("fixture replay is not idempotent: first=%#v second=%#v", first, second)
	}
	for _, item := range first.Cases {
		if !item.Idempotent || item.Digest != item.ReplayDigest {
			t.Fatalf("case %q is not idempotent: %#v", item.CaseID, item)
		}
	}
}

func TestCapabilityAssetProvenanceRejectsNegativeFixtures(t *testing.T) {
	for _, name := range []string{"capability_asset_provenance_scope_violation.json", "capability_asset_provenance_privacy_violation.json"} {
		raw, err := os.ReadFile("testdata/" + name)
		if err != nil {
			t.Fatal(err)
		}
		result, err := ReplayCapabilityAssetProvenanceJSON(raw)
		if name == "capability_asset_provenance_scope_violation.json" {
			if err != nil || len(result.Cases) != 1 || !containsCapabilityString(result.Cases[0].Findings, ReasonCodeCapabilityAssetScopeViolation) {
				t.Fatalf("fixture %s result=%#v error=%v", name, result, err)
			}
			continue
		}
		if err == nil {
			t.Fatalf("fixture %s unexpectedly passed", name)
		}
		if name == "capability_asset_provenance_privacy_violation.json" && !strings.Contains(err.Error(), ReasonCodeCapabilityAssetPrivacyOrBoundViolation) {
			t.Fatalf("fixture %s error = %v", name, err)
		}
	}
}

func reflectCapabilityResultEqual(left, right CapabilityAssetProvenanceResult) bool {
	a, _ := json.Marshal(left)
	b, _ := json.Marshal(right)
	return string(a) == string(b)
}
