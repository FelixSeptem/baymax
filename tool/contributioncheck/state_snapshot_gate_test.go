package contributioncheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStateSnapshotGateScriptParity(t *testing.T) {
	root := repoRoot(t)
	shellPath := filepath.Join(root, "scripts", "check-state-snapshot-contract.sh")
	psPath := filepath.Join(root, "scripts", "check-state-snapshot-contract.ps1")

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
		"state_control_plane_absent",
		"state_source_of_truth_reuse_required",
		"runtime\\.(state\\.snapshot|session\\.state)\\.[a-zA-Z0-9_.-]*(control_plane|controlplane|state_service|orchestrator|controller|managed_state|hosted_state|remote_state|migration_center)",
		"runtime\\.state\\.snapshot\\.[a-zA-Z0-9_.-]*(memory_mode|memory_provider|memory_profile|memory_contract_version|memory_scope_selected|memory_budget_used|memory_hits|memory_rerank_stats|memory_lifecycle_action)",
		"A66 必须复用现有 checkpoint/snapshot 语义与 A59 memory lifecycle，不得重写存储层事实源。",
	}
	for _, token := range requiredTokens {
		if !strings.Contains(shell, token) {
			t.Fatalf("shell state snapshot gate missing token %q", token)
		}
		if !strings.Contains(ps, token) {
			t.Fatalf("powershell state snapshot gate missing token %q", token)
		}
	}
	if !strings.Contains(shell, "assert_no_parallel_a66_snapshot_changes") {
		t.Fatalf("shell state snapshot gate missing assertion helper for parallel proposal closure")
	}
	if !strings.Contains(ps, "Assert-NoParallelA66SnapshotChanges") {
		t.Fatalf("powershell state snapshot gate missing assertion helper for parallel proposal closure")
	}
	if !strings.Contains(shell, "set -euo pipefail") {
		t.Fatalf("shell state snapshot gate must use set -euo pipefail")
	}
	if !strings.Contains(ps, "lib/native-strict.ps1") || !strings.Contains(ps, "Invoke-NativeStrict") {
		t.Fatalf("powershell state snapshot gate must use strict native helper")
	}
	if strings.Contains(ps, "AllowFailure") {
		t.Fatalf("powershell state snapshot gate must not add AllowFailure exceptions")
	}
}

func TestQualityGateIncludesStateSnapshotGate(t *testing.T) {
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
	if !strings.Contains(shell, "check-state-snapshot-contract.sh") {
		t.Fatalf("shell quality gate must invoke state snapshot contract gate")
	}
	if !strings.Contains(shell, "[quality-gate][state-snapshot-contract]") {
		t.Fatalf("shell quality gate must expose blocking state snapshot gate failure label")
	}
	if !strings.Contains(ps, "check-state-snapshot-contract.ps1") {
		t.Fatalf("powershell quality gate must invoke state snapshot contract gate")
	}
	if !strings.Contains(ps, "[quality-gate] state snapshot contract suites") {
		t.Fatalf("powershell quality gate must expose state snapshot gate step label")
	}
}

func TestCIIncludesStateSnapshotRequiredCheckCandidate(t *testing.T) {
	root := repoRoot(t)
	ciPath := filepath.Join(root, ".github", "workflows", "ci.yml")

	raw, err := os.ReadFile(ciPath)
	if err != nil {
		t.Fatalf("read ci workflow: %v", err)
	}
	ci := string(raw)
	if !strings.Contains(ci, "state-snapshot-contract-gate:") {
		t.Fatalf("ci workflow must expose state-snapshot-contract-gate required-check candidate job")
	}
	if !strings.Contains(ci, "State Snapshot Contract Gate") {
		t.Fatalf("ci workflow must include human-readable state snapshot gate step label")
	}
	if !strings.Contains(ci, "bash scripts/check-state-snapshot-contract.sh") {
		t.Fatalf("ci workflow must execute check-state-snapshot-contract.sh")
	}
}
