package contributioncheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContextCompressionProductionGateScriptParity(t *testing.T) {
	root := repoRoot(t)
	shellPath := filepath.Join(root, "scripts", "check-context-compression-production-contract.sh")
	psPath := filepath.Join(root, "scripts", "check-context-compression-production-contract.ps1")

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
		"check-context-jit-organization-contract",
		"check-diagnostics-replay-contract",
		"check-context-production-hardening-benchmark-regression",
		"ReplayContractContextCompressionProductionFixtureSuite",
		"ContextCompressionProductionGateScriptParity",
	}
	for _, token := range requiredTokens {
		if !strings.Contains(shell, token) {
			t.Fatalf("shell context-compression-production gate missing token %q", token)
		}
		if !strings.Contains(ps, token) {
			t.Fatalf("powershell context-compression-production gate missing token %q", token)
		}
	}
	if !strings.Contains(shell, "set -euo pipefail") {
		t.Fatalf("shell context-compression-production gate must use set -euo pipefail")
	}
	if !strings.Contains(ps, "lib/native-strict.ps1") || !strings.Contains(ps, "Invoke-NativeStrict") {
		t.Fatalf("powershell context-compression-production gate must use strict native helper")
	}
}

func TestQualityGateIncludesContextCompressionProductionGate(t *testing.T) {
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
	if !strings.Contains(shell, "check-context-compression-production-contract.sh") {
		t.Fatalf("shell quality gate must invoke context compression production contract gate")
	}
	if !strings.Contains(shell, "[quality-gate][context-compression-production-contract]") {
		t.Fatalf("shell quality gate must expose blocking context-compression-production gate failure label")
	}
	if !strings.Contains(ps, "check-context-compression-production-contract.ps1") {
		t.Fatalf("powershell quality gate must invoke context compression production contract gate")
	}
	if !strings.Contains(ps, "[quality-gate] context compression production contract suites") {
		t.Fatalf("powershell quality gate must expose context-compression-production gate step label")
	}
}

func TestCIIncludesContextCompressionProductionRequiredCheckCandidate(t *testing.T) {
	root := repoRoot(t)
	ciPath := filepath.Join(root, ".github", "workflows", "ci.yml")

	raw, err := os.ReadFile(ciPath)
	if err != nil {
		t.Fatalf("read ci workflow: %v", err)
	}
	ci := string(raw)
	if !strings.Contains(ci, "context-compression-production-contract-gate:") {
		t.Fatalf("ci workflow must expose context-compression-production-contract-gate required-check candidate job")
	}
	if !strings.Contains(ci, "Context Compression Production Contract Gate") {
		t.Fatalf("ci workflow must include human-readable context-compression-production gate step label")
	}
	if !strings.Contains(ci, "bash scripts/check-context-compression-production-contract.sh") {
		t.Fatalf("ci workflow must execute check-context-compression-production-contract.sh")
	}
}

func TestContextCompressionProductionRoadmapAndContractIndexClosureMarkers(t *testing.T) {
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
	requiredRoadmap := "Context organization 语义能力同域需求（reference-first、isolate handoff、edit gate、relevance swap-back、lifecycle tiering、task-aware recap）优先在 Context JIT Organization 增量吸收；生产可用治理同域需求（压缩质量门控、冷存检索/清理、一致性回放、强门禁）统一在 a69 吸收，不再新增平行 context 压缩提案。"
	requiredIndexRows := []string{
		"Context Compression Production Replay Fixture (`context_compression_production.v1`)",
		"Context Compression Production Benchmark Regression Gate",
		"Context Compression Production Contract Gate",
		"Context Compression Production Contract Gate CI Required-Check 候选",
		"Context Compression Production Contract Gate Quality Path",
	}

	if !strings.Contains(roadmap, requiredRoadmap) {
		t.Fatalf("roadmap must include context-compression same-domain closure marker: %q", requiredRoadmap)
	}
	for _, row := range requiredIndexRows {
		if !strings.Contains(index, row) {
			t.Fatalf("mainline contract index missing context-compression row: %q", row)
		}
	}
}
