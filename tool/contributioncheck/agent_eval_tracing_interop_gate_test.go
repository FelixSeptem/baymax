package contributioncheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAgentEvalTracingInteropGateScriptParity(t *testing.T) {
	root := repoRoot(t)
	shellPath := filepath.Join(root, "scripts", "check-agent-eval-and-tracing-interop-contract.sh")
	psPath := filepath.Join(root, "scripts", "check-agent-eval-and-tracing-interop-contract.ps1")

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
		"tracing_eval_field_reuse_required",
		"runtime\\.eval\\.execution\\.[a-zA-Z0-9_.-]*(control_plane|controlplane|scheduler_service|orchestrator_endpoint|controller_endpoint|hosted_scheduler|remote_scheduler)",
		"runtime\\.eval\\.[a-zA-Z0-9_.-]*(policy_decision_path|deny_source|winner_stage|memory_scope_selected|memory_budget_used|memory_hits|memory_rerank_stats|memory_lifecycle_action|budget_snapshot|budget_decision|degrade_action)",
		"RuntimeRecorderParsesTracingEvalAdditiveFields",
		"RuntimeRecorderTracingEvalParserCompatibilityAdditiveNullableDefault",
		"StoreRunTracingEvalAdditiveFieldsPersistAndReplayIdempotent",
		"StoreRunTracingEvalAdditiveFieldsBoundedCardinality",
		"Tracing+eval 同域增量需求（语义映射、指标汇总、执行治理、回放、门禁）仅允许在本提案内以增量任务吸收，不再新开平行提案。",
	}
	for _, token := range requiredTokens {
		if !strings.Contains(shell, token) {
			t.Fatalf("shell tracing+eval interop gate missing token %q", token)
		}
		if !strings.Contains(ps, token) {
			t.Fatalf("powershell tracing+eval interop gate missing token %q", token)
		}
	}
	if !strings.Contains(shell, "assert_no_parallel_tracing_eval_changes") {
		t.Fatalf("shell tracing+eval interop gate missing assertion helper for parallel proposal closure")
	}
	if !strings.Contains(ps, "Assert-NoParallelTracingEvalChanges") {
		t.Fatalf("powershell tracing+eval interop gate missing assertion helper for parallel proposal closure")
	}

	if !strings.Contains(shell, "set -euo pipefail") {
		t.Fatalf("shell tracing+eval interop gate must use set -euo pipefail")
	}
	if !strings.Contains(ps, "lib/native-strict.ps1") || !strings.Contains(ps, "Invoke-NativeStrict") {
		t.Fatalf("powershell tracing+eval interop gate must use strict native helper")
	}
	if strings.Contains(ps, "AllowFailure") {
		t.Fatalf("powershell tracing+eval interop gate must not add AllowFailure exceptions")
	}
}

func TestQualityGateIncludesAgentEvalTracingInteropGate(t *testing.T) {
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
	if !strings.Contains(shell, "check-agent-eval-and-tracing-interop-contract.sh") {
		t.Fatalf("shell quality gate must invoke tracing+eval interop gate")
	}
	if !strings.Contains(shell, "[quality-gate][agent-eval-tracing-interop-contract]") {
		t.Fatalf("shell quality gate must expose blocking tracing+eval interop gate failure label")
	}
	if !strings.Contains(ps, "check-agent-eval-and-tracing-interop-contract.ps1") {
		t.Fatalf("powershell quality gate must invoke tracing+eval interop gate")
	}
	if !strings.Contains(ps, "[quality-gate] agent eval and tracing interop contract suites") {
		t.Fatalf("powershell quality gate must expose tracing+eval interop step label")
	}
}

func TestAgentEvalTracingInteropRoadmapAndContractIndexClosureMarkers(t *testing.T) {
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
	requiredRoadmap := "Tracing+eval 同域增量需求（语义映射、指标汇总、执行治理、回放、门禁）仅允许在本提案内以增量任务吸收，不再新开平行提案。"
	requiredIndexRows := []string{
		"Tracing + Eval Replay Fixture (`otel_semconv.v1`/`agent_eval.v1`/`agent_eval_distributed.v1`)",
		"Tracing + Eval Interop Gate",
		"Tracing + Eval Interop Gate CI Required-Check 候选",
		"Tracing + Eval Interop Gate Quality Path",
	}

	if !strings.Contains(roadmap, requiredRoadmap) {
		t.Fatalf("roadmap must include tracing+eval same-domain closure marker: %q", requiredRoadmap)
	}
	for _, row := range requiredIndexRows {
		if !strings.Contains(index, row) {
			t.Fatalf("mainline contract index missing tracing+eval row: %q", row)
		}
	}
}
