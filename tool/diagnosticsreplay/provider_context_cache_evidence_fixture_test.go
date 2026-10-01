package diagnosticsreplay

import (
	"os"
	"strings"
	"testing"
)

const providerContextCacheEvidenceFixturePath = "testdata/provider_context_cache_evidence.v1.json"

func TestProviderContextCacheEvidenceFixtureCoversThreeProvidersAndIsBounded(t *testing.T) {
	raw, err := os.ReadFile(providerContextCacheEvidenceFixturePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) > 1<<20 {
		t.Fatalf("fixture exceeds bound: %d", len(raw))
	}
	lower := strings.ToLower(string(raw))
	for _, marker := range []string{"raw_payload", "reasoning", "credential", "password", "bearer token"} {
		if strings.Contains(lower, marker) {
			t.Fatalf("valid fixture contains privacy marker %q", marker)
		}
	}
	first, err := ReplayProviderContextCacheEvidenceJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ReplayProviderContextCacheEvidenceJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Cases) != 3 || len(second.Cases) != 3 {
		t.Fatalf("fixture cases = %d/%d, want 3", len(first.Cases), len(second.Cases))
	}
	providers := map[string]bool{}
	verdicts := map[string]bool{}
	for i, item := range first.Cases {
		providers[item.Provider] = true
		verdicts[item.Verdict] = true
		if item.Digest == "" || item.Digest != item.ReplayDigest || item.Digest != second.Cases[i].Digest {
			t.Fatalf("case %q is not deterministic: %#v", item.CaseID, item)
		}
	}
	for _, provider := range []string{"openai", "anthropic", "gemini"} {
		if !providers[provider] {
			t.Fatalf("fixture does not cover provider %q", provider)
		}
	}
	for _, verdict := range []string{"no-drift", "drift-confirmed", "insufficient-evidence"} {
		if !verdicts[verdict] {
			t.Fatalf("fixture does not cover verdict %q", verdict)
		}
	}
}

func TestProviderContextCacheEvidencePrivacyFixtureFailsFast(t *testing.T) {
	raw, err := os.ReadFile("testdata/provider_context_cache_evidence_privacy.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ReplayProviderContextCacheEvidenceJSON(raw); err == nil || !strings.Contains(err.Error(), ReasonCodeContextCacheEvidencePrivacyDrift) {
		t.Fatalf("privacy fixture error = %v", err)
	}
}
