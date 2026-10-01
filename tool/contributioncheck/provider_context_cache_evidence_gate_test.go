package contributioncheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProviderContextCacheEvidenceGateScriptsStayPairedAndOffline(t *testing.T) {
	root := repoRoot(t)
	shellPath := filepath.Join(root, "scripts", "check-provider-context-cache-evidence-contract.sh")
	psPath := filepath.Join(root, "scripts", "check-provider-context-cache-evidence-contract.ps1")
	shell, err := os.ReadFile(shellPath)
	if err != nil {
		t.Fatal(err)
	}
	ps, err := os.ReadFile(psPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{"provider_context_cache_evidence.v1", "go test ./model/conformance", "provider_context_cache_evidence.go", "review-only"} {
		if !strings.Contains(string(shell), marker) || !strings.Contains(string(ps), marker) {
			t.Fatalf("shell/PowerShell gate pair missing marker %q", marker)
		}
	}
}
