package contributioncheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDocsConsistencyRepoHygieneTempArtifacts(t *testing.T) {
	root := repoRoot(t)
	shellPath := filepath.Join(root, "scripts", "check-docs-consistency.sh")
	psPath := filepath.Join(root, "scripts", "check-docs-consistency.ps1")

	shellRaw, err := os.ReadFile(shellPath)
	if err != nil {
		t.Fatalf("read shell docs consistency script: %v", err)
	}
	psRaw, err := os.ReadFile(psPath)
	if err != nil {
		t.Fatalf("read powershell docs consistency script: %v", err)
	}

	shell := string(shellRaw)
	ps := string(psRaw)

	requiredShared := []string{
		"git ls-files --others --exclude-standard",
		"[repo-hygiene]",
		".tmp",
		".bak",
	}
	for _, token := range requiredShared {
		if !strings.Contains(shell, token) {
			t.Fatalf("shell docs consistency missing repo-hygiene token %q", token)
		}
		if !strings.Contains(ps, token) {
			t.Fatalf("powershell docs consistency missing repo-hygiene token %q", token)
		}
	}

	if !strings.Contains(shell, ".go.[0-9]") {
		t.Fatalf("shell docs consistency missing timestamp backup pattern for go files")
	}
	if !strings.Contains(ps, ".go\\.[0-9]+") {
		t.Fatalf("powershell docs consistency missing timestamp backup pattern for go files")
	}
	if !strings.Contains(shell, "*~") {
		t.Fatalf("shell docs consistency missing trailing-tilde temp file pattern")
	}
	if !strings.Contains(ps, "~)$") {
		t.Fatalf("powershell docs consistency missing trailing-tilde temp file pattern")
	}
}
