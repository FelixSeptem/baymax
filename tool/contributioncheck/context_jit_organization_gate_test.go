package contributioncheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContextJITOrganizationGateScriptParity(t *testing.T) {
	root := repoRoot(t)
	shellPath := filepath.Join(root, "scripts", "check-context-jit-organization-contract.sh")
	psPath := filepath.Join(root, "scripts", "check-context-jit-organization-contract.ps1")

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
		"context_provider_sdk_absent",
		"Test(RuntimeContextJITConfig|ManagerRuntimeContextJIT)",
		"Test(StoreRunContextJIT|RuntimeRecorderParsesContextJITOrganizationAdditiveFields|RuntimeRecorderContextJITParserCompatibilityAdditiveNullableDefault)",
		"Test(ReplayContractPrimaryReasonArbitrationFixtureSuccessAndDeterministicOutput|ReplayContractPrimaryReasonArbitrationFixtureDriftClassification",
		"check-react-plan-notebook-contract",
		"check-realtime-protocol-contract",
		"check-policy-precedence-contract",
		"check-sandbox-egress-allowlist-contract",
		"check-diagnostics-replay-contract",
		"ContextJITOrganizationGateScriptParity",
	}
	for _, token := range requiredTokens {
		if !strings.Contains(shell, token) {
			t.Fatalf("shell context-jit-organization gate missing token %q", token)
		}
		if !strings.Contains(ps, token) {
			t.Fatalf("powershell context-jit-organization gate missing token %q", token)
		}
	}
	if !strings.Contains(shell, "set -euo pipefail") {
		t.Fatalf("shell context-jit-organization gate must use set -euo pipefail")
	}
	if !strings.Contains(ps, "lib/native-strict.ps1") || !strings.Contains(ps, "Invoke-NativeStrict") {
		t.Fatalf("powershell context-jit-organization gate must use strict native helper")
	}
}

func TestQualityGateIncludesContextJITOrganizationGate(t *testing.T) {
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
	if !strings.Contains(shell, "check-context-jit-organization-contract.sh") {
		t.Fatalf("shell quality gate must invoke context jit organization contract gate")
	}
	if !strings.Contains(shell, "[quality-gate][context-jit-organization-contract]") {
		t.Fatalf("shell quality gate must expose blocking context-jit-organization gate failure label")
	}
	if !strings.Contains(ps, "check-context-jit-organization-contract.ps1") {
		t.Fatalf("powershell quality gate must invoke context jit organization contract gate")
	}
	if !strings.Contains(ps, "[quality-gate] context jit organization contract suites") {
		t.Fatalf("powershell quality gate must expose context-jit-organization gate step label")
	}
}

func TestCIIncludesContextJITOrganizationRequiredCheckCandidate(t *testing.T) {
	root := repoRoot(t)
	ciPath := filepath.Join(root, ".github", "workflows", "ci.yml")

	raw, err := os.ReadFile(ciPath)
	if err != nil {
		t.Fatalf("read ci workflow: %v", err)
	}
	ci := string(raw)
	if !strings.Contains(ci, "context-jit-organization-contract-gate:") {
		t.Fatalf("ci workflow must expose context-jit-organization-contract-gate required-check candidate job")
	}
	if !strings.Contains(ci, "Context JIT Organization Contract Gate") {
		t.Fatalf("ci workflow must include human-readable context-jit-organization gate step label")
	}
	if !strings.Contains(ci, "bash scripts/check-context-jit-organization-contract.sh") {
		t.Fatalf("ci workflow must execute check-context-jit-organization-contract.sh")
	}
}

func TestContextJITOrganizationRoadmapAndContractIndexClosureMarkers(t *testing.T) {
	root := repoRoot(t)
	roadmapPath := filepath.Join(root, "docs", "development-roadmap.md")
	indexPath := filepath.Join(root, "docs", "mainline-contract-test-index.md")

	roadmapRaw, err := os.ReadFile(roadmapPath)
	if err != nil {
		t.Fatalf("read roadmap: %v", err)
	}
	indexRaw, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read contract index: %v", err)
	}

	roadmap := string(roadmapRaw)
	index := string(indexRaw)
	requiredRoadmap := "Context organization 同域需求（reference-first、isolate handoff、edit gate、relevance swap-back、lifecycle tiering、task-aware recap）优先在本提案内增量吸收，不再新增平行 context 组织提案。"
	requiredIndexRows := []string{
		"Context JIT Organization Replay Fixture (`context_reference_first.v1`/`context_isolate_handoff.v1`/`context_edit_gate.v1`/`context_relevance_swapback.v1`/`context_lifecycle_tiering.v1`)",
		"Context JIT Organization Contract Gate",
		"Context JIT Organization Contract Gate CI Required-Check 候选",
		"Context JIT Organization Contract Gate Quality Path",
	}

	if !strings.Contains(roadmap, requiredRoadmap) {
		t.Fatalf("roadmap must include context-organization same-domain closure marker: %q", requiredRoadmap)
	}
	for _, row := range requiredIndexRows {
		if !strings.Contains(index, row) {
			t.Fatalf("mainline contract index missing context-jit row: %q", row)
		}
	}
}
