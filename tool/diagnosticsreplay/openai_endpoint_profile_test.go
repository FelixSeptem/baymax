package diagnosticsreplay

import (
	"os"
	"testing"
)

func TestReplayOpenAIEndpointProfileFixtureJSON(t *testing.T) {
	raw, err := os.ReadFile("testdata/openai_endpoint_profile_conformance.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	result, err := ReplayOpenAIEndpointProfileFixtureJSON(raw, map[string]string{
		"openai\x00openai_run_stream_parity": "ec53587635884f9f73b49ff53e10b46422466c83d2a601ee1b48620ce1fdd600",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Cases) != 2 || result.Cases[0].Verdict != OpenAIEndpointProfileVerdictConformant {
		t.Fatalf("result = %#v", result)
	}
}

func TestOpenAIEndpointProfileReplayKeepsHistoricalProjectionFixtureCompatible(t *testing.T) {
	raw, err := os.ReadFile("testdata/model_request_projection.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	result, err := ReplayProviderRequestProjectionFixtureJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Cases) == 0 {
		t.Fatal("historical projection fixture produced no cases")
	}
	for _, item := range result.Cases {
		if item.Digest == "" || item.Digest != item.ReplayDigest {
			t.Fatalf("historical case %q is not idempotent: %#v", item.Name, item)
		}
	}
}
