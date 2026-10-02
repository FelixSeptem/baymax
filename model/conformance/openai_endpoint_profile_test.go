package conformance

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestOpenAIEndpointProfileFixtureReplaysOfficialAndCompatibleProfiles(t *testing.T) {
	raw, err := os.ReadFile("../../tool/diagnosticsreplay/testdata/openai_endpoint_profile_conformance.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := ParseOpenAIEndpointProfileFixtureJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	digests := map[string]string{
		"openai\x00openai_run_stream_parity": "ec53587635884f9f73b49ff53e10b46422466c83d2a601ee1b48620ce1fdd600",
	}
	first, err := ReplayOpenAIEndpointProfileFixture(fixture, digests)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ReplayOpenAIEndpointProfileFixture(fixture, digests)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Cases) != 2 || len(second.Cases) != 2 {
		t.Fatalf("case count = %d/%d, want 2", len(first.Cases), len(second.Cases))
	}
	for i := range first.Cases {
		if first.Cases[i].Verdict != OpenAIEndpointProfileVerdictConformant || first.Cases[i].Digest != first.Cases[i].ReplayDigest || first.Cases[i].Digest != second.Cases[i].Digest {
			t.Fatalf("case %q replay = %#v / %#v", first.Cases[i].Name, first.Cases[i], second.Cases[i])
		}
	}
}

func TestOpenAIEndpointProfileFixtureRejectsReferenceAndPrivacyDrift(t *testing.T) {
	raw, err := os.ReadFile("../../tool/diagnosticsreplay/testdata/openai_endpoint_profile_conformance.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture OpenAIEndpointProfileFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	digests := map[string]string{"openai\x00openai_run_stream_parity": strings.Repeat("f", 64)}
	if _, err := ReplayOpenAIEndpointProfileFixture(fixture, digests); err == nil || !strings.Contains(err.Error(), ReasonOpenAIEndpointProfileReferenceDrift) {
		t.Fatalf("reference error = %v", err)
	}
	fixture = mustOpenAIEndpointProfileFixture(t, raw)
	fixture.Cases[0].Endpoint = "https://user:secret@example.test/v1"
	if _, err := ReplayOpenAIEndpointProfileFixture(fixture, nil); err == nil || !strings.Contains(err.Error(), ReasonOpenAIEndpointProfilePrivacyDrift) {
		t.Fatalf("privacy error = %v", err)
	}
}

func TestOpenAIEndpointProfileFixtureRejectsUndeclaredProjectionDifference(t *testing.T) {
	raw, err := os.ReadFile("../../tool/diagnosticsreplay/testdata/openai_endpoint_profile_conformance.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	fixture := mustOpenAIEndpointProfileFixture(t, raw)
	fixture.Cases[0].ToolResultCorrelated = false
	digests := map[string]string{"openai\x00openai_run_stream_parity": "ec53587635884f9f73b49ff53e10b46422466c83d2a601ee1b48620ce1fdd600"}
	result, err := ReplayOpenAIEndpointProfileFixture(fixture, digests)
	if err != nil {
		t.Fatal(err)
	}
	if result.Cases[0].Verdict != OpenAIEndpointProfileVerdictBlocked || !containsString(result.Cases[0].Reasons, ReasonOpenAIEndpointProfileToolResultDrift) {
		t.Fatalf("result = %#v, want blocked tool-result drift", result.Cases[0])
	}
}

func mustOpenAIEndpointProfileFixture(t *testing.T, raw []byte) OpenAIEndpointProfileFixture {
	t.Helper()
	var fixture OpenAIEndpointProfileFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	return fixture
}
