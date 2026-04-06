package contributioncheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReactPlanNotebookGateScriptParity(t *testing.T) {
	root := repoRoot(t)
	shellPath := filepath.Join(root, "scripts", "check-react-plan-notebook-contract.sh")
	psPath := filepath.Join(root, "scripts", "check-react-plan-notebook-contract.ps1")

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
		"Test(ReactPlan|ReactPlanNotebookDoesNotBypass)",
		"StoreRunReactPlanNotebook",
		"RuntimeRecorderParsesReactPlanNotebookAdditiveFields",
		"RuntimeRecorderReactPlanNotebookParserCompatibilityAdditiveNullableDefault",
		"check-policy-precedence-contract",
		"check-sandbox-egress-allowlist-contract",
		"check-diagnostics-replay-contract",
		"ReactPlanNotebookGateScriptParity",
	}
	for _, token := range requiredTokens {
		if !strings.Contains(shell, token) {
			t.Fatalf("shell react-plan-notebook gate missing token %q", token)
		}
		if !strings.Contains(ps, token) {
			t.Fatalf("powershell react-plan-notebook gate missing token %q", token)
		}
	}
	if !strings.Contains(shell, "set -euo pipefail") {
		t.Fatalf("shell react-plan-notebook gate must use set -euo pipefail")
	}
	if !strings.Contains(ps, "lib/native-strict.ps1") || !strings.Contains(ps, "Invoke-NativeStrict") {
		t.Fatalf("powershell react-plan-notebook gate must use strict native helper")
	}
}

func TestQualityGateIncludesReactPlanNotebookGate(t *testing.T) {
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
	if !strings.Contains(shell, "check-react-plan-notebook-contract.sh") {
		t.Fatalf("shell quality gate must invoke react plan notebook contract gate")
	}
	if !strings.Contains(shell, "[quality-gate][react-plan-notebook-contract]") {
		t.Fatalf("shell quality gate must expose blocking react-plan-notebook gate failure label")
	}
	if !strings.Contains(ps, "check-react-plan-notebook-contract.ps1") {
		t.Fatalf("powershell quality gate must invoke react plan notebook contract gate")
	}
	if !strings.Contains(ps, "[quality-gate] react plan notebook contract suites") {
		t.Fatalf("powershell quality gate must expose react-plan-notebook gate step label")
	}
}

func TestCIIncludesReactPlanNotebookRequiredCheckCandidate(t *testing.T) {
	root := repoRoot(t)
	ciPath := filepath.Join(root, ".github", "workflows", "ci.yml")

	raw, err := os.ReadFile(ciPath)
	if err != nil {
		t.Fatalf("read ci workflow: %v", err)
	}
	ci := string(raw)
	if !strings.Contains(ci, "react-plan-notebook-gate:") {
		t.Fatalf("ci workflow must expose react-plan-notebook-gate required-check candidate job")
	}
	if !strings.Contains(ci, "ReAct Plan Notebook Contract Gate") {
		t.Fatalf("ci workflow must include human-readable react-plan-notebook gate step label")
	}
	if !strings.Contains(ci, "bash scripts/check-react-plan-notebook-contract.sh") {
		t.Fatalf("ci workflow must execute check-react-plan-notebook-contract.sh")
	}
}
