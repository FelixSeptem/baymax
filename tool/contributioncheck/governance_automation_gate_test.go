package contributioncheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRoadmapStatusConsistencyGateScriptParity(t *testing.T) {
	root := repoRoot(t)
	shellPath := filepath.Join(root, "scripts", "check-openspec-roadmap-status-consistency.sh")
	psPath := filepath.Join(root, "scripts", "check-openspec-roadmap-status-consistency.ps1")

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
		"roadmap-status-drift",
		"openspec list --json",
		"openspec/changes/archive/INDEX.md",
		"docs/development-roadmap.md",
	}
	for _, token := range requiredTokens {
		if !strings.Contains(shell, token) {
			t.Fatalf("shell roadmap status script missing token %q", token)
		}
		if !strings.Contains(ps, token) {
			t.Fatalf("powershell roadmap status script missing token %q", token)
		}
	}
	if !strings.Contains(shell, "set -euo pipefail") {
		t.Fatalf("shell roadmap status script must use set -euo pipefail")
	}
	if !strings.Contains(ps, "lib/native-strict.ps1") {
		t.Fatalf("powershell roadmap status script must import strict helper")
	}
}

func TestExampleImpactDeclarationGateScriptParity(t *testing.T) {
	root := repoRoot(t)
	shellPath := filepath.Join(root, "scripts", "check-openspec-example-impact-declaration.sh")
	psPath := filepath.Join(root, "scripts", "check-openspec-example-impact-declaration.ps1")

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
		"missing-example-impact-declaration",
		"invalid-example-impact-value",
		"新增示例",
		"修改示例",
		"无需示例变更（附理由）",
	}
	for _, token := range requiredTokens {
		if !strings.Contains(shell, token) {
			t.Fatalf("shell example impact script missing token %q", token)
		}
		if !strings.Contains(ps, token) {
			t.Fatalf("powershell example impact script missing token %q", token)
		}
	}
	if !strings.Contains(shell, "set -euo pipefail") {
		t.Fatalf("shell example impact script must use set -euo pipefail")
	}
	if !strings.Contains(ps, "lib/native-strict.ps1") {
		t.Fatalf("powershell example impact script must import strict helper")
	}
}

func TestDocsConsistencyIncludesRoadmapStatusConsistencyGate(t *testing.T) {
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
	if !strings.Contains(shell, "check-openspec-roadmap-status-consistency.sh") {
		t.Fatalf("shell docs consistency must invoke roadmap status consistency script")
	}
	if !strings.Contains(shell, "[docs-consistency][openspec-roadmap-status-consistency]") {
		t.Fatalf("shell docs consistency must expose roadmap status consistency failure label")
	}
	if !strings.Contains(ps, "check-openspec-roadmap-status-consistency.ps1") {
		t.Fatalf("powershell docs consistency must invoke roadmap status consistency script")
	}
	if !strings.Contains(ps, "[docs-consistency] openspec roadmap status consistency") {
		t.Fatalf("powershell docs consistency must expose roadmap status consistency step label")
	}
}

func TestQualityGateIncludesExampleImpactDeclarationGate(t *testing.T) {
	root := repoRoot(t)
	shellPath := filepath.Join(root, "scripts", "check-quality-gate.sh")
	psPath := filepath.Join(root, "scripts", "check-quality-gate.ps1")

	shellRaw, err := os.ReadFile(shellPath)
	if err != nil {
		t.Fatalf("read shell quality gate script: %v", err)
	}
	psRaw, err := os.ReadFile(psPath)
	if err != nil {
		t.Fatalf("read powershell quality gate script: %v", err)
	}

	shell := string(shellRaw)
	ps := string(psRaw)
	if !strings.Contains(shell, "check-openspec-example-impact-declaration.sh") {
		t.Fatalf("shell quality gate must invoke example impact declaration script")
	}
	if !strings.Contains(shell, "[quality-gate][openspec-example-impact-declaration]") {
		t.Fatalf("shell quality gate must expose example impact declaration failure label")
	}
	if !strings.Contains(ps, "check-openspec-example-impact-declaration.ps1") {
		t.Fatalf("powershell quality gate must invoke example impact declaration script")
	}
	if !strings.Contains(ps, "[quality-gate] openspec example impact declaration") {
		t.Fatalf("powershell quality gate must expose example impact declaration step label")
	}
}

func TestCIGovernanceRequiredCheckCandidates(t *testing.T) {
	root := repoRoot(t)
	ciPath := filepath.Join(root, ".github", "workflows", "ci.yml")
	raw, err := os.ReadFile(ciPath)
	if err != nil {
		t.Fatalf("read ci workflow: %v", err)
	}
	ci := string(raw)

	requiredTokens := []string{
		"openspec-roadmap-status-consistency-gate:",
		"run: bash scripts/check-openspec-roadmap-status-consistency.sh",
		"openspec-example-impact-declaration-gate:",
		"run: bash scripts/check-openspec-example-impact-declaration.sh",
		"if: github.event_name == 'pull_request'",
	}
	for _, token := range requiredTokens {
		if !strings.Contains(ci, token) {
			t.Fatalf("ci workflow missing token %q for governance required-check candidates", token)
		}
	}
}
