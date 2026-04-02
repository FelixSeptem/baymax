package contributioncheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimeBudgetAdmissionGateScriptParity(t *testing.T) {
	root := repoRoot(t)
	shellPath := filepath.Join(root, "scripts", "check-runtime-budget-admission-contract.sh")
	psPath := filepath.Join(root, "scripts", "check-runtime-budget-admission-contract.ps1")

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
		"budget_control_plane_absent",
		"budget_field_reuse_required",
		"policy_decision_path|deny_source|winner_stage|memory_scope_selected|memory_budget_used|memory_hits|memory_rerank_stats|memory_lifecycle_action",
		"A60 预算 admission 同域增量需求（阈值、维度、降级动作、回放、门禁）仅允许在 A60 内以增量任务吸收，不再新开平行提案。",
	}
	for _, token := range requiredTokens {
		if !strings.Contains(shell, token) {
			t.Fatalf("shell runtime budget admission gate missing token %q", token)
		}
		if !strings.Contains(ps, token) {
			t.Fatalf("powershell runtime budget admission gate missing token %q", token)
		}
	}
	if !strings.Contains(shell, "assert_no_parallel_budget_admission_changes") {
		t.Fatalf("shell runtime budget admission gate missing assertion helper for parallel proposal closure")
	}
	if !strings.Contains(ps, "Assert-NoParallelBudgetAdmissionChanges") {
		t.Fatalf("powershell runtime budget admission gate missing assertion helper for parallel proposal closure")
	}

	if !strings.Contains(shell, "set -euo pipefail") {
		t.Fatalf("shell runtime budget admission gate must use set -euo pipefail")
	}
	if !strings.Contains(ps, "lib/native-strict.ps1") || !strings.Contains(ps, "Invoke-NativeStrict") {
		t.Fatalf("powershell runtime budget admission gate must use strict native helper")
	}
	if strings.Contains(ps, "AllowFailure") {
		t.Fatalf("powershell runtime budget admission gate must not add AllowFailure exceptions")
	}
}

func TestQualityGateIncludesRuntimeBudgetAdmissionGate(t *testing.T) {
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
	if !strings.Contains(shell, "check-runtime-budget-admission-contract.sh") {
		t.Fatalf("shell quality gate must invoke runtime budget admission gate")
	}
	if !strings.Contains(shell, "[quality-gate][runtime-budget-admission-contract]") {
		t.Fatalf("shell quality gate must expose blocking runtime budget admission failure label")
	}
	if !strings.Contains(ps, "check-runtime-budget-admission-contract.ps1") {
		t.Fatalf("powershell quality gate must invoke runtime budget admission gate")
	}
	if !strings.Contains(ps, "[quality-gate] runtime budget admission contract suites") {
		t.Fatalf("powershell quality gate must expose runtime budget admission step label")
	}
}

func TestRuntimeBudgetAdmissionRoadmapAndContractIndexClosureMarkers(t *testing.T) {
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
	requiredRoadmap := "A60 预算 admission 同域增量需求（阈值、维度、降级动作、回放、门禁）仅允许在 A60 内以增量任务吸收，不再新开平行提案。"
	requiredIndexRows := []string{
		"Runtime Budget + Admission A60 Gate Assertion",
		"Runtime Budget + Admission A60 Quality Path",
	}

	if !strings.Contains(roadmap, requiredRoadmap) {
		t.Fatalf("roadmap must include A60 same-domain closure marker: %q", requiredRoadmap)
	}
	for _, row := range requiredIndexRows {
		if !strings.Contains(index, row) {
			t.Fatalf("mainline contract index missing A60 row: %q", row)
		}
	}
}
