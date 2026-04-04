package contributioncheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRealtimeProtocolGateScriptParity(t *testing.T) {
	root := repoRoot(t)
	shellPath := filepath.Join(root, "scripts", "check-realtime-protocol-contract.sh")
	psPath := filepath.Join(root, "scripts", "check-realtime-protocol-contract.ps1")

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
		"realtime_control_plane_absent",
		"runtime\\.realtime\\.[a-zA-Z0-9_.-]*(control_plane|controlplane|gateway|connection_router|session_router|managed_connection|hosted_realtime|realtime_service)",
		"Test(StoreRunA68|RuntimeRecorderParsesA68RealtimeAdditiveFields|RuntimeRecorderA68ParserCompatibilityAdditiveNullableDefault)",
		"A68 realtime 同域增量需求（事件类型扩展、中断恢复语义、顺序/幂等、回放/门禁）仅允许在 A68 内以增量任务吸收，不再新增平行 realtime 提案。",
	}
	for _, token := range requiredTokens {
		if !strings.Contains(shell, token) {
			t.Fatalf("shell realtime protocol gate missing token %q", token)
		}
		if !strings.Contains(ps, token) {
			t.Fatalf("powershell realtime protocol gate missing token %q", token)
		}
	}
	if !strings.Contains(shell, "assert_no_parallel_a68_changes") {
		t.Fatalf("shell realtime protocol gate missing assertion helper for parallel proposal closure")
	}
	if !strings.Contains(ps, "Assert-NoParallelA68Changes") {
		t.Fatalf("powershell realtime protocol gate missing assertion helper for parallel proposal closure")
	}
	if !strings.Contains(shell, "set -euo pipefail") {
		t.Fatalf("shell realtime protocol gate must use set -euo pipefail")
	}
	if !strings.Contains(ps, "lib/native-strict.ps1") || !strings.Contains(ps, "Invoke-NativeStrict") {
		t.Fatalf("powershell realtime protocol gate must use strict native helper")
	}
	if strings.Contains(ps, "AllowFailure") {
		t.Fatalf("powershell realtime protocol gate must not add AllowFailure exceptions")
	}
}

func TestQualityGateIncludesRealtimeProtocolGate(t *testing.T) {
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
	if !strings.Contains(shell, "check-realtime-protocol-contract.sh") {
		t.Fatalf("shell quality gate must invoke realtime protocol contract gate")
	}
	if !strings.Contains(shell, "[quality-gate][realtime-protocol-contract]") {
		t.Fatalf("shell quality gate must expose blocking realtime protocol gate failure label")
	}
	if !strings.Contains(ps, "check-realtime-protocol-contract.ps1") {
		t.Fatalf("powershell quality gate must invoke realtime protocol contract gate")
	}
	if !strings.Contains(ps, "[quality-gate] realtime protocol contract suites") {
		t.Fatalf("powershell quality gate must expose realtime protocol gate step label")
	}
}

func TestCIIncludesRealtimeProtocolRequiredCheckCandidate(t *testing.T) {
	root := repoRoot(t)
	ciPath := filepath.Join(root, ".github", "workflows", "ci.yml")

	raw, err := os.ReadFile(ciPath)
	if err != nil {
		t.Fatalf("read ci workflow: %v", err)
	}
	ci := string(raw)
	if !strings.Contains(ci, "realtime-protocol-contract-gate:") {
		t.Fatalf("ci workflow must expose realtime-protocol-contract-gate required-check candidate job")
	}
	if !strings.Contains(ci, "Realtime Protocol Contract Gate") {
		t.Fatalf("ci workflow must include human-readable realtime protocol gate step label")
	}
	if !strings.Contains(ci, "bash scripts/check-realtime-protocol-contract.sh") {
		t.Fatalf("ci workflow must execute check-realtime-protocol-contract.sh")
	}
}

func TestRealtimeProtocolRoadmapAndContractIndexClosureMarkers(t *testing.T) {
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
	requiredRoadmap := "A68 realtime 同域增量需求（事件类型扩展、中断恢复语义、顺序/幂等、回放/门禁）仅允许在 A68 内以增量任务吸收，不再新增平行 realtime 提案。"
	requiredIndexRows := []string{
		"A68 Realtime Protocol Replay Fixture (`realtime_event_protocol.v1`)",
		"A68 Realtime Protocol Contract Gate",
		"A68 Realtime Protocol Contract Gate CI Required-Check 候选",
		"A68 Realtime Protocol Contract Gate Quality Path",
	}

	if !strings.Contains(roadmap, requiredRoadmap) {
		t.Fatalf("roadmap must include A68 same-domain closure marker: %q", requiredRoadmap)
	}
	for _, row := range requiredIndexRows {
		if !strings.Contains(index, row) {
			t.Fatalf("mainline contract index missing A68 row: %q", row)
		}
	}
}
