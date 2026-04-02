package contributioncheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHooksMiddlewareGateScriptParity(t *testing.T) {
	root := repoRoot(t)
	shellPath := filepath.Join(root, "scripts", "check-hooks-middleware-contract.sh")
	psPath := filepath.Join(root, "scripts", "check-hooks-middleware-contract.ps1")

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
		"control_plane_absent",
		"control_plane|controlplane|orchestrator|controller|service_endpoint|remote_hook|hosted_hook|managed_middleware",
		"TestReplayContractA65HooksMiddleware",
		"A65 hooks/middleware 同域增量需求（lifecycle、middleware、discovery、preprocess、mapping、回放、门禁）仅允许在 A65 内以增量任务吸收，不再新开平行提案。",
	}
	for _, token := range requiredTokens {
		if !strings.Contains(shell, token) {
			t.Fatalf("shell hooks/middleware gate missing token %q", token)
		}
		if !strings.Contains(ps, token) {
			t.Fatalf("powershell hooks/middleware gate missing token %q", token)
		}
	}
	if !strings.Contains(shell, "assert_no_parallel_a65_changes") {
		t.Fatalf("shell hooks/middleware gate missing assertion helper for parallel proposal closure")
	}
	if !strings.Contains(ps, "Assert-NoParallelA65Changes") {
		t.Fatalf("powershell hooks/middleware gate missing assertion helper for parallel proposal closure")
	}
	if !strings.Contains(shell, "set -euo pipefail") {
		t.Fatalf("shell hooks/middleware gate must use set -euo pipefail")
	}
	if !strings.Contains(ps, "lib/native-strict.ps1") || !strings.Contains(ps, "Invoke-NativeStrict") {
		t.Fatalf("powershell hooks/middleware gate must use strict native helper")
	}
	if strings.Contains(ps, "AllowFailure") {
		t.Fatalf("powershell hooks/middleware gate must not add AllowFailure exceptions")
	}
}

func TestQualityGateIncludesHooksMiddlewareGate(t *testing.T) {
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
	if !strings.Contains(shell, "check-hooks-middleware-contract.sh") {
		t.Fatalf("shell quality gate must invoke hooks/middleware contract gate")
	}
	if !strings.Contains(shell, "[quality-gate][hooks-middleware-contract]") {
		t.Fatalf("shell quality gate must expose blocking hooks/middleware gate failure label")
	}
	if !strings.Contains(ps, "check-hooks-middleware-contract.ps1") {
		t.Fatalf("powershell quality gate must invoke hooks/middleware contract gate")
	}
	if !strings.Contains(ps, "[quality-gate] hooks + middleware contract suites") {
		t.Fatalf("powershell quality gate must expose hooks/middleware gate step label")
	}
}

func TestHooksMiddlewareRoadmapAndContractIndexClosureMarkers(t *testing.T) {
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
	requiredRoadmap := "A65 hooks/middleware 同域增量需求（lifecycle、middleware、discovery、preprocess、mapping、回放、门禁）仅允许在 A65 内以增量任务吸收，不再新开平行提案。"
	requiredIndexRows := []string{
		"A65 Hooks + Middleware/Skill Replay Fixtures",
		"A65 Hooks + Middleware Contract Gate",
		"A65 Hooks + Middleware Contract Gate CI Required-Check 候选",
		"A65 Hooks + Middleware Contract Gate Quality Path",
	}

	if !strings.Contains(roadmap, requiredRoadmap) {
		t.Fatalf("roadmap must include A65 same-domain closure marker: %q", requiredRoadmap)
	}
	for _, row := range requiredIndexRows {
		if !strings.Contains(index, row) {
			t.Fatalf("mainline contract index missing A65 row: %q", row)
		}
	}
}
