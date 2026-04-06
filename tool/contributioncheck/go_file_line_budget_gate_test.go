package contributioncheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGoFileLineBudgetGateScriptParity(t *testing.T) {
	root := repoRoot(t)
	shellPath := filepath.Join(root, "scripts", "check-go-file-line-budget.sh")
	psPath := filepath.Join(root, "scripts", "check-go-file-line-budget.ps1")

	shellRaw, err := os.ReadFile(shellPath)
	if err != nil {
		t.Fatalf("read shell script: %v", err)
	}
	psRaw, err := os.ReadFile(psPath)
	if err != nil {
		t.Fatalf("read powershell script: %v", err)
	}

	shell := string(shellRaw)
	ps := string(psRaw)
	requiredTokens := []string{
		"go-file-line-budget-policy.env",
		"go-file-line-budget-exceptions.csv",
		"BAYMAX_GO_LINE_BUDGET_WARN",
		"BAYMAX_GO_LINE_BUDGET_HARD",
		"git ls-files '*.go'",
		"_test.go",
		"allow_growth",
		"stale exception",
	}
	for _, token := range requiredTokens {
		if !strings.Contains(shell, token) {
			t.Fatalf("shell go-file-line-budget gate missing token %q", token)
		}
		if !strings.Contains(ps, token) {
			t.Fatalf("powershell go-file-line-budget gate missing token %q", token)
		}
	}
	if !strings.Contains(shell, "set -euo pipefail") {
		t.Fatalf("shell go-file-line-budget gate must use set -euo pipefail")
	}
	if !strings.Contains(ps, "lib/native-strict.ps1") {
		t.Fatalf("powershell go-file-line-budget gate must import native strict helper")
	}
}

func TestQualityGateIncludesGoFileLineBudgetGate(t *testing.T) {
	root := repoRoot(t)
	shellPath := filepath.Join(root, "scripts", "check-quality-gate.sh")
	psPath := filepath.Join(root, "scripts", "check-quality-gate.ps1")

	shellRaw, err := os.ReadFile(shellPath)
	if err != nil {
		t.Fatalf("read shell quality gate: %v", err)
	}
	psRaw, err := os.ReadFile(psPath)
	if err != nil {
		t.Fatalf("read powershell quality gate: %v", err)
	}

	shell := string(shellRaw)
	ps := string(psRaw)
	if !strings.Contains(shell, "check-go-file-line-budget.sh") {
		t.Fatalf("shell quality gate must invoke go-file-line-budget gate")
	}
	if !strings.Contains(shell, "[quality-gate][go-file-line-budget]") {
		t.Fatalf("shell quality gate must expose go-file-line-budget failure label")
	}
	if !strings.Contains(ps, "check-go-file-line-budget.ps1") {
		t.Fatalf("powershell quality gate must invoke go-file-line-budget gate")
	}
	if !strings.Contains(ps, "[quality-gate] go file line budget governance") {
		t.Fatalf("powershell quality gate must expose go-file-line-budget step label")
	}
}

func TestGoFileLineBudgetGovernanceArtifactsExist(t *testing.T) {
	root := repoRoot(t)
	requiredFiles := []string{
		filepath.Join(root, "openspec", "governance", "go-file-line-budget-policy.env"),
		filepath.Join(root, "openspec", "governance", "go-file-line-budget-exceptions.csv"),
	}

	for _, path := range requiredFiles {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("required go-file-line-budget artifact missing: %s", path)
		}
	}
}
