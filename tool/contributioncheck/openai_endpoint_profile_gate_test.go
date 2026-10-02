package contributioncheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenAIEndpointProfileConformanceBoundary(t *testing.T) {
	root := repoRoot(t)
	fixturePath := filepath.Join(root, "tool", "diagnosticsreplay", "testdata", "openai_endpoint_profile_conformance.v1.json")
	info, err := os.Stat(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() > 1<<20 {
		t.Fatalf("fixture exceeds 1 MiB: %d", info.Size())
	}
	raw, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{"raw_prompt", "raw_reasoning", "raw_payload", "credential", "password", "bearer token"} {
		if strings.Contains(strings.ToLower(string(raw)), marker) {
			t.Fatalf("fixture contains privacy marker %q", marker)
		}
	}
	var document struct {
		Version string `json:"version"`
		Cases   []struct {
			ProfileID    string            `json:"profile_id"`
			Capabilities map[string]string `json:"capabilities"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	if document.Version != "openai-compatible-endpoint-profile-conformance.v1" || len(document.Cases) != 2 {
		t.Fatalf("fixture header = %#v", document)
	}
	for _, item := range document.Cases {
		if item.ProfileID == "" {
			t.Fatal("profile id is required")
		}
		for _, capability := range []string{"structured_input", "native_tool_result", "streaming", "cache_usage", "count_tokens"} {
			if _, ok := item.Capabilities[capability]; !ok {
				t.Fatalf("profile %q omits independent capability %q", item.ProfileID, capability)
			}
		}
	}
}

func TestOpenAIEndpointProfileGateScriptsStayPairedAndOffline(t *testing.T) {
	root := repoRoot(t)
	shell, err := os.ReadFile(filepath.Join(root, "scripts", "check-openai-compatible-endpoint-profile.sh"))
	if err != nil {
		t.Fatal(err)
	}
	powershell, err := os.ReadFile(filepath.Join(root, "scripts", "check-openai-compatible-endpoint-profile.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{"openai-compatible-endpoint-profile-conformance.v1", "ReplayOpenAIEndpointProfileFixtureJSON", "review-only", "no network"} {
		if !strings.Contains(string(shell), marker) || !strings.Contains(string(powershell), marker) {
			t.Fatalf("gate pair missing marker %q", marker)
		}
	}
}
