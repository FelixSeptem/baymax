package contributioncheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSemanticLabelingGovernanceGateScriptParity(t *testing.T) {
	root := repoRoot(t)
	shellPath := filepath.Join(root, "scripts", "check-semantic-labeling-governance.sh")
	psPath := filepath.Join(root, "scripts", "check-semantic-labeling-governance.ps1")

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
		"semantic-labeling-governed-path-matrix.yaml",
		"semantic-labeling-legacy-mapping.yaml",
		"semantic-labeling-regression-baseline.csv",
		"legacy-axx-content",
		"legacy-context-stage-wording",
		"legacy_aliases|context_assembler_stage_mapping",
	}
	for _, token := range requiredTokens {
		if !strings.Contains(shell, token) {
			t.Fatalf("shell semantic-labeling governance gate missing token %q", token)
		}
		if !strings.Contains(ps, token) {
			t.Fatalf("powershell semantic-labeling governance gate missing token %q", token)
		}
	}
	if !strings.Contains(shell, "set -euo pipefail") {
		t.Fatalf("shell semantic-labeling governance gate must use set -euo pipefail")
	}
	if !strings.Contains(ps, "lib/native-strict.ps1") {
		t.Fatalf("powershell semantic-labeling governance gate must import native strict helper")
	}
}

func TestQualityGateIncludesSemanticLabelingGovernance(t *testing.T) {
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
	if !strings.Contains(shell, "check-semantic-labeling-governance.sh") {
		t.Fatalf("shell quality gate must invoke semantic-labeling governance gate")
	}
	if !strings.Contains(shell, "[quality-gate][semantic-labeling-governance]") {
		t.Fatalf("shell quality gate must expose semantic-labeling governance failure label")
	}
	if !strings.Contains(ps, "check-semantic-labeling-governance.ps1") {
		t.Fatalf("powershell quality gate must invoke semantic-labeling governance gate")
	}
	if !strings.Contains(ps, "[quality-gate] semantic labeling governance") {
		t.Fatalf("powershell quality gate must expose semantic-labeling governance step label")
	}
}

func TestDocsConsistencyIncludesSemanticLabelingGovernance(t *testing.T) {
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
	if !strings.Contains(shell, "check-semantic-labeling-governance.sh") {
		t.Fatalf("shell docs consistency must invoke semantic-labeling governance gate")
	}
	if !strings.Contains(shell, "[docs-consistency][semantic-labeling-governance]") {
		t.Fatalf("shell docs consistency must expose semantic-labeling governance failure label")
	}
	if !strings.Contains(ps, "check-semantic-labeling-governance.ps1") {
		t.Fatalf("powershell docs consistency must invoke semantic-labeling governance gate")
	}
}

func TestSemanticLabelingGovernanceArtifactsExist(t *testing.T) {
	root := repoRoot(t)
	requiredFiles := []string{
		filepath.Join(root, "openspec", "governance", "semantic-labeling-governed-path-matrix.yaml"),
		filepath.Join(root, "openspec", "governance", "semantic-labeling-legacy-mapping.yaml"),
		filepath.Join(root, "openspec", "governance", "semantic-labeling-regression-baseline.csv"),
	}

	for _, path := range requiredFiles {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("required semantic-labeling governance artifact missing: %s", path)
		}
	}
}
