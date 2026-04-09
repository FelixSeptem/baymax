package contributioncheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAgentModePatternCoverageGateScriptParity(t *testing.T) {
	root := repoRoot(t)
	shellPath := filepath.Join(root, "scripts", "check-agent-mode-pattern-coverage.sh")
	psPath := filepath.Join(root, "scripts", "check-agent-mode-pattern-coverage.ps1")

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
		"examples/agent-modes/MATRIX.md",
		"context-governed-reference-first",
		"custom-adapter-health-readiness-circuit",
		"missing matrix rows",
		"missing required mode families",
	}
	for _, token := range requiredTokens {
		if !strings.Contains(shell, token) {
			t.Fatalf("shell pattern coverage gate missing token %q", token)
		}
		if !strings.Contains(ps, token) {
			t.Fatalf("powershell pattern coverage gate missing token %q", token)
		}
	}
	if !strings.Contains(shell, "set -euo pipefail") {
		t.Fatalf("shell pattern coverage gate must use set -euo pipefail")
	}
	if !strings.Contains(ps, "Set-StrictMode -Version Latest") {
		t.Fatalf("powershell pattern coverage gate must use strict mode")
	}
}

func TestAgentModeExamplesSmokeGateScriptParity(t *testing.T) {
	root := repoRoot(t)
	shellPath := filepath.Join(root, "scripts", "check-agent-mode-examples-smoke.sh")
	psPath := filepath.Join(root, "scripts", "check-agent-mode-examples-smoke.ps1")

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
		"BAYMAX_AGENT_MODE_SMOKE_PATTERNS",
		"BAYMAX_AGENT_MODE_SMOKE_VARIANTS",
		"no patterns selected",
		"unsupported variant",
		"go run",
		"verification.mainline_runtime_path=ok",
		"result.final_answer=",
		"result.signature=",
		"production-ish",
	}
	for _, token := range requiredTokens {
		if !strings.Contains(shell, token) {
			t.Fatalf("shell examples smoke gate missing token %q", token)
		}
		if !strings.Contains(ps, token) {
			t.Fatalf("powershell examples smoke gate missing token %q", token)
		}
	}
	if !strings.Contains(shell, "set -euo pipefail") {
		t.Fatalf("shell examples smoke gate must use set -euo pipefail")
	}
	if !strings.Contains(ps, "Invoke-NativeCaptureStrict") {
		t.Fatalf("powershell examples smoke gate must use strict native capture helper")
	}
	if !strings.Contains(shell, "GOCACHE") {
		t.Fatalf("shell examples smoke gate must configure go cache for deterministic execution")
	}
	if !strings.Contains(ps, "GOCACHE") {
		t.Fatalf("powershell examples smoke gate must configure go cache for deterministic execution")
	}
}

func TestAgentModeMigrationPlaybookConsistencyGateScriptParity(t *testing.T) {
	root := repoRoot(t)
	shellPath := filepath.Join(root, "scripts", "check-agent-mode-migration-playbook-consistency.sh")
	psPath := filepath.Join(root, "scripts", "check-agent-mode-migration-playbook-consistency.ps1")

	shellRaw, err := os.ReadFile(shellPath)
	if err != nil {
		t.Fatalf("read shell migration playbook gate: %v", err)
	}
	psRaw, err := os.ReadFile(psPath)
	if err != nil {
		t.Fatalf("read powershell migration playbook gate: %v", err)
	}

	shell := string(shellRaw)
	ps := string(psRaw)
	requiredTokens := []string{
		"examples/agent-modes/MATRIX.md",
		"examples/agent-modes/PLAYBOOK.md",
		"missing-checklist",
		"missing-gate",
		"Prod Delta Checklist",
	}
	for _, token := range requiredTokens {
		if !strings.Contains(shell, token) {
			t.Fatalf("shell migration playbook gate missing token %q", token)
		}
		if !strings.Contains(ps, token) {
			t.Fatalf("powershell migration playbook gate missing token %q", token)
		}
	}
	if !strings.Contains(shell, "set -euo pipefail") {
		t.Fatalf("shell migration playbook gate must use set -euo pipefail")
	}
	if !strings.Contains(ps, "Set-StrictMode -Version Latest") {
		t.Fatalf("powershell migration playbook gate must use strict mode")
	}
}

func TestAgentModeSmokeStabilityGovernanceGateScriptParity(t *testing.T) {
	root := repoRoot(t)
	shellPath := filepath.Join(root, "scripts", "check-agent-mode-smoke-stability-governance.sh")
	psPath := filepath.Join(root, "scripts", "check-agent-mode-smoke-stability-governance.ps1")

	shellRaw, err := os.ReadFile(shellPath)
	if err != nil {
		t.Fatalf("read shell smoke stability governance gate: %v", err)
	}
	psRaw, err := os.ReadFile(psPath)
	if err != nil {
		t.Fatalf("read powershell smoke stability governance gate: %v", err)
	}

	shell := string(shellRaw)
	ps := string(psRaw)
	requiredTokens := []string{
		"STABILITY_BASELINE.json",
		"example-smoke-latency-regression",
		"example-smoke-flaky-regression",
		"BAYMAX_AGENT_MODE_STABILITY_TIMEOUT_SEC",
		"BAYMAX_AGENT_MODE_STABILITY_RETRY_MAX",
	}
	for _, token := range requiredTokens {
		if !strings.Contains(shell, token) {
			t.Fatalf("shell smoke stability governance gate missing token %q", token)
		}
		if !strings.Contains(ps, token) {
			t.Fatalf("powershell smoke stability governance gate missing token %q", token)
		}
	}
	if !strings.Contains(shell, "timeout") {
		t.Fatalf("shell smoke stability governance gate must include timeout handling")
	}
	if !strings.Contains(ps, "WaitForExit") {
		t.Fatalf("powershell smoke stability governance gate must include timeout handling")
	}
}

func TestAgentModeLegacyTodoCleanupGateScriptParity(t *testing.T) {
	root := repoRoot(t)
	shellPath := filepath.Join(root, "scripts", "check-agent-mode-legacy-todo-cleanup.sh")
	psPath := filepath.Join(root, "scripts", "check-agent-mode-legacy-todo-cleanup.ps1")

	shellRaw, err := os.ReadFile(shellPath)
	if err != nil {
		t.Fatalf("read shell legacy todo cleanup gate: %v", err)
	}
	psRaw, err := os.ReadFile(psPath)
	if err != nil {
		t.Fatalf("read powershell legacy todo cleanup gate: %v", err)
	}

	shell := string(shellRaw)
	ps := string(psRaw)
	requiredTokens := []string{
		"TODO|TBD|FIXME|待补",
		"LEGACY_TODO_BASELINE",
		"legacy-placeholder",
		"rg",
	}
	for _, token := range requiredTokens {
		if !strings.Contains(shell, token) {
			t.Fatalf("shell legacy todo cleanup gate missing token %q", token)
		}
		if !strings.Contains(ps, token) {
			t.Fatalf("powershell legacy todo cleanup gate missing token %q", token)
		}
	}
	if !strings.Contains(shell, "set -euo pipefail") {
		t.Fatalf("shell legacy todo cleanup gate must use set -euo pipefail")
	}
	if !strings.Contains(ps, "Set-StrictMode -Version Latest") {
		t.Fatalf("powershell legacy todo cleanup gate must use strict mode")
	}
}

func TestAgentModeRealLogicContractGateScriptParity(t *testing.T) {
	root := repoRoot(t)
	shellPath := filepath.Join(root, "scripts", "check-agent-mode-real-logic-contract.sh")
	psPath := filepath.Join(root, "scripts", "check-agent-mode-real-logic-contract.ps1")

	shellRaw, err := os.ReadFile(shellPath)
	if err != nil {
		t.Fatalf("read shell real logic gate: %v", err)
	}
	psRaw, err := os.ReadFile(psPath)
	if err != nil {
		t.Fatalf("read powershell real logic gate: %v", err)
	}

	shell := string(shellRaw)
	ps := string(psRaw)
	requiredTokens := []string{
		"agent-mode-simulated-engine-dependency",
		"agent-mode-placeholder-output-regression",
		"agent-mode-missing-mainline-runtime-path",
		"verification.mainline_runtime_path=",
		"result.signature=",
	}
	for _, token := range requiredTokens {
		if !strings.Contains(shell, token) {
			t.Fatalf("shell real logic gate missing token %q", token)
		}
		if !strings.Contains(ps, token) {
			t.Fatalf("powershell real logic gate missing token %q", token)
		}
	}
	if !strings.Contains(shell, "set -euo pipefail") {
		t.Fatalf("shell real logic gate must use set -euo pipefail")
	}
	if !strings.Contains(ps, "Set-StrictMode -Version Latest") {
		t.Fatalf("powershell real logic gate must use strict mode")
	}
}

func TestAgentModeReadmeSyncContractGateScriptParity(t *testing.T) {
	root := repoRoot(t)
	shellPath := filepath.Join(root, "scripts", "check-agent-mode-readme-sync-contract.sh")
	psPath := filepath.Join(root, "scripts", "check-agent-mode-readme-sync-contract.ps1")

	shellRaw, err := os.ReadFile(shellPath)
	if err != nil {
		t.Fatalf("read shell readme sync gate: %v", err)
	}
	psRaw, err := os.ReadFile(psPath)
	if err != nil {
		t.Fatalf("read powershell readme sync gate: %v", err)
	}

	shell := string(shellRaw)
	ps := string(psRaw)
	requiredTokens := []string{
		"agent-mode-readme-not-updated",
		"agent-mode-readme-missing-required-sections",
		"## Run",
		"## Prerequisites",
		"## Real Runtime Path",
		"## Expected Output/Verification",
	}
	for _, token := range requiredTokens {
		if !strings.Contains(shell, token) {
			t.Fatalf("shell readme sync gate missing token %q", token)
		}
		if !strings.Contains(ps, token) {
			t.Fatalf("powershell readme sync gate missing token %q", token)
		}
	}
	if !strings.Contains(shell, "set -euo pipefail") {
		t.Fatalf("shell readme sync gate must use set -euo pipefail")
	}
	if !strings.Contains(ps, "Set-StrictMode -Version Latest") {
		t.Fatalf("powershell readme sync gate must use strict mode")
	}
}

func TestQualityGateIncludesAgentModeCoverageAndSmoke(t *testing.T) {
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
	if !strings.Contains(shell, "check-agent-mode-pattern-coverage.sh") {
		t.Fatalf("shell quality gate must invoke agent mode pattern coverage")
	}
	if !strings.Contains(shell, "[quality-gate][agent-mode-pattern-coverage]") {
		t.Fatalf("shell quality gate must expose pattern coverage blocking label")
	}
	if !strings.Contains(shell, "check-agent-mode-examples-smoke.sh") {
		t.Fatalf("shell quality gate must invoke agent mode examples smoke")
	}
	if !strings.Contains(shell, "[quality-gate][agent-mode-examples-smoke]") {
		t.Fatalf("shell quality gate must expose examples smoke blocking label")
	}
	if !strings.Contains(shell, "check-agent-mode-smoke-stability-governance.sh") {
		t.Fatalf("shell quality gate must invoke agent mode smoke stability governance")
	}
	if !strings.Contains(shell, "[quality-gate][agent-mode-smoke-stability-governance]") {
		t.Fatalf("shell quality gate must expose smoke stability governance blocking label")
	}
	if !strings.Contains(shell, "check-agent-mode-migration-playbook-consistency.sh") {
		t.Fatalf("shell quality gate must invoke agent mode migration playbook consistency")
	}
	if !strings.Contains(shell, "[quality-gate][agent-mode-migration-playbook-consistency]") {
		t.Fatalf("shell quality gate must expose migration playbook consistency blocking label")
	}
	if !strings.Contains(shell, "check-agent-mode-legacy-todo-cleanup.sh") {
		t.Fatalf("shell quality gate must invoke agent mode legacy todo cleanup")
	}
	if !strings.Contains(shell, "[quality-gate][agent-mode-legacy-todo-cleanup]") {
		t.Fatalf("shell quality gate must expose legacy todo cleanup blocking label")
	}
	if !strings.Contains(shell, "check-agent-mode-real-logic-contract.sh") {
		t.Fatalf("shell quality gate must invoke agent mode real logic contract")
	}
	if !strings.Contains(shell, "[quality-gate][agent-mode-real-logic-contract]") {
		t.Fatalf("shell quality gate must expose real logic contract blocking label")
	}
	if !strings.Contains(shell, "check-agent-mode-readme-sync-contract.sh") {
		t.Fatalf("shell quality gate must invoke agent mode readme sync contract")
	}
	if !strings.Contains(shell, "[quality-gate][agent-mode-readme-sync-contract]") {
		t.Fatalf("shell quality gate must expose readme sync contract blocking label")
	}

	if !strings.Contains(ps, "check-agent-mode-pattern-coverage.ps1") {
		t.Fatalf("powershell quality gate must invoke agent mode pattern coverage")
	}
	if !strings.Contains(ps, "[quality-gate] agent mode pattern coverage") {
		t.Fatalf("powershell quality gate must expose pattern coverage step label")
	}
	if !strings.Contains(ps, "check-agent-mode-examples-smoke.ps1") {
		t.Fatalf("powershell quality gate must invoke agent mode examples smoke")
	}
	if !strings.Contains(ps, "[quality-gate] agent mode examples smoke") {
		t.Fatalf("powershell quality gate must expose examples smoke step label")
	}
	if !strings.Contains(ps, "check-agent-mode-smoke-stability-governance.ps1") {
		t.Fatalf("powershell quality gate must invoke agent mode smoke stability governance")
	}
	if !strings.Contains(ps, "[quality-gate] agent mode smoke stability governance") {
		t.Fatalf("powershell quality gate must expose smoke stability governance step label")
	}
	if !strings.Contains(ps, "check-agent-mode-migration-playbook-consistency.ps1") {
		t.Fatalf("powershell quality gate must invoke agent mode migration playbook consistency")
	}
	if !strings.Contains(ps, "[quality-gate] agent mode migration playbook consistency") {
		t.Fatalf("powershell quality gate must expose migration playbook consistency step label")
	}
	if !strings.Contains(ps, "check-agent-mode-legacy-todo-cleanup.ps1") {
		t.Fatalf("powershell quality gate must invoke agent mode legacy todo cleanup")
	}
	if !strings.Contains(ps, "[quality-gate] agent mode legacy todo cleanup") {
		t.Fatalf("powershell quality gate must expose legacy todo cleanup step label")
	}
	if !strings.Contains(ps, "check-agent-mode-real-logic-contract.ps1") {
		t.Fatalf("powershell quality gate must invoke agent mode real logic contract")
	}
	if !strings.Contains(ps, "[quality-gate] agent mode real logic contract") {
		t.Fatalf("powershell quality gate must expose real logic contract step label")
	}
	if !strings.Contains(ps, "check-agent-mode-readme-sync-contract.ps1") {
		t.Fatalf("powershell quality gate must invoke agent mode readme sync contract")
	}
	if !strings.Contains(ps, "[quality-gate] agent mode readme sync contract") {
		t.Fatalf("powershell quality gate must expose readme sync contract step label")
	}
}

func TestAgentModeMatrixAndDocMappingsExist(t *testing.T) {
	root := repoRoot(t)
	matrixPath := filepath.Join(root, "examples", "agent-modes", "MATRIX.md")
	entryPath := filepath.Join(root, "examples", "agent-modes", "README.md")
	legacyPath := filepath.Join(root, "examples", "agent-modes", "LEGACY_TODO_BASELINE.md")
	playbookPath := filepath.Join(root, "examples", "agent-modes", "PLAYBOOK.md")
	stabilityPath := filepath.Join(root, "examples", "agent-modes", "STABILITY_BASELINE.json")
	readmePath := filepath.Join(root, "README.md")
	indexPath := filepath.Join(root, "docs", "mainline-contract-test-index.md")

	matrixRaw, err := os.ReadFile(matrixPath)
	if err != nil {
		t.Fatalf("read matrix: %v", err)
	}
	entryRaw, err := os.ReadFile(entryPath)
	if err != nil {
		t.Fatalf("read agent mode entry readme: %v", err)
	}
	legacyRaw, err := os.ReadFile(legacyPath)
	if err != nil {
		t.Fatalf("read legacy todo baseline: %v", err)
	}
	playbookRaw, err := os.ReadFile(playbookPath)
	if err != nil {
		t.Fatalf("read playbook: %v", err)
	}
	stabilityRaw, err := os.ReadFile(stabilityPath)
	if err != nil {
		t.Fatalf("read stability baseline: %v", err)
	}
	readmeRaw, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("read repository README: %v", err)
	}
	indexRaw, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read mainline contract index: %v", err)
	}

	matrix := string(matrixRaw)
	entry := string(entryRaw)
	legacy := string(legacyRaw)
	playbook := string(playbookRaw)
	stability := string(stabilityRaw)
	repoReadme := string(readmeRaw)
	index := string(indexRaw)

	requiredMatrixTokens := []string{
		"pattern -> minimal -> production-ish -> contracts -> gates -> replay",
		"| `rag-hybrid-retrieval` |",
		"| `context-governed-reference-first` |",
		"| `custom-adapter-health-readiness-circuit` |",
	}
	for _, token := range requiredMatrixTokens {
		if !strings.Contains(matrix, token) {
			t.Fatalf("matrix missing token %q", token)
		}
	}
	if !strings.Contains(entry, "examples/agent-modes/MATRIX.md") {
		t.Fatalf("agent mode entry readme must reference matrix")
	}
	if !strings.Contains(legacy, "No unresolved `TODO/TBD/FIXME/待补` markers") {
		t.Fatalf("legacy TODO baseline summary missing expected marker")
	}
	if !strings.Contains(playbook, "## Mode Mapping") {
		t.Fatalf("playbook must include mode mapping section")
	}
	if !strings.Contains(playbook, "`context-governed-reference-first`") {
		t.Fatalf("playbook must include context-governed pattern mapping")
	}
	if !strings.Contains(stability, "\"max_p95_ms\"") {
		t.Fatalf("stability baseline must include max_p95_ms threshold")
	}
	if !strings.Contains(stability, "\"max_flaky_rate\"") {
		t.Fatalf("stability baseline must include max_flaky_rate threshold")
	}
	if !strings.Contains(repoReadme, "examples/agent-modes") {
		t.Fatalf("repository README must include agent mode example pack entry")
	}
	if !strings.Contains(index, "Agent Mode Example Pack Mapping") {
		t.Fatalf("mainline contract index must include agent mode mapping section")
	}
}
