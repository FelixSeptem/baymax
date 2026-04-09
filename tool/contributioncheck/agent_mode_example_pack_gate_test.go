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
		"pattern -> phase -> a71_scope -> a71_status -> semantic_anchor -> runtime_path_evidence -> expected_verification_markers -> minimal -> production-ish -> contracts -> gates -> replay",
		"context-governed-reference-first",
		"custom-adapter-health-readiness-circuit",
		"missing matrix rows",
		"rows missing semantic/runtime evidence columns",
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
		"agent-mode-smoke-semantic-evidence-missing",
		"unsupported variant",
		"go run",
		"verification.mainline_runtime_path=ok",
		"verification.semantic.anchor=",
		"verification.semantic.governance=",
		"verification.semantic.expected_markers=",
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
		"## Failure/Rollback Notes",
		"Production Migration Checklist",
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

func TestAgentModeRealRuntimeSemanticContractGateScriptParity(t *testing.T) {
	root := repoRoot(t)
	shellPath := filepath.Join(root, "scripts", "check-agent-mode-real-runtime-semantic-contract.sh")
	psPath := filepath.Join(root, "scripts", "check-agent-mode-real-runtime-semantic-contract.ps1")

	shellRaw, err := os.ReadFile(shellPath)
	if err != nil {
		t.Fatalf("read shell real runtime semantic gate: %v", err)
	}
	psRaw, err := os.ReadFile(psPath)
	if err != nil {
		t.Fatalf("read powershell real runtime semantic gate: %v", err)
	}

	shell := string(shellRaw)
	ps := string(psRaw)
	requiredTokens := []string{
		"agent-mode-shared-semantic-engine-detected",
		"agent-mode-semantic-ownership-missing",
		"agent-mode-missing-runtime-path-evidence",
		"semantic_example.go",
		"modeimpl.RunMinimal()",
		"modeimpl.RunProduction()",
		"verification.semantic.runtime_path=",
		"verification.semantic.expected_markers=",
		"runtimeexample.MustRun",
	}
	for _, token := range requiredTokens {
		if !strings.Contains(shell, token) {
			t.Fatalf("shell real runtime semantic gate missing token %q", token)
		}
		if !strings.Contains(ps, token) {
			t.Fatalf("powershell real runtime semantic gate missing token %q", token)
		}
	}
	if !strings.Contains(shell, "set -euo pipefail") {
		t.Fatalf("shell real runtime semantic gate must use set -euo pipefail")
	}
	if !strings.Contains(ps, "Set-StrictMode -Version Latest") {
		t.Fatalf("powershell real runtime semantic gate must use strict mode")
	}
}

func TestAgentModeReadmeRuntimeSyncContractGateScriptParity(t *testing.T) {
	root := repoRoot(t)
	shellPath := filepath.Join(root, "scripts", "check-agent-mode-readme-runtime-sync-contract.sh")
	psPath := filepath.Join(root, "scripts", "check-agent-mode-readme-runtime-sync-contract.ps1")

	shellRaw, err := os.ReadFile(shellPath)
	if err != nil {
		t.Fatalf("read shell readme runtime sync gate: %v", err)
	}
	psRaw, err := os.ReadFile(psPath)
	if err != nil {
		t.Fatalf("read powershell readme runtime sync gate: %v", err)
	}

	shell := string(shellRaw)
	ps := string(psRaw)
	requiredTokens := []string{
		"agent-mode-readme-runtime-desync",
		"agent-mode-readme-required-sections-missing",
		"## Run",
		"## Prerequisites",
		"## Real Runtime Path",
		"## Expected Output/Verification",
		"## Failure/Rollback Notes",
	}
	for _, token := range requiredTokens {
		if !strings.Contains(shell, token) {
			t.Fatalf("shell readme runtime sync gate missing token %q", token)
		}
		if !strings.Contains(ps, token) {
			t.Fatalf("powershell readme runtime sync gate missing token %q", token)
		}
	}
	if !strings.Contains(shell, "set -euo pipefail") {
		t.Fatalf("shell readme runtime sync gate must use set -euo pipefail")
	}
	if !strings.Contains(ps, "Set-StrictMode -Version Latest") {
		t.Fatalf("powershell readme runtime sync gate must use strict mode")
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
	if !strings.Contains(shell, "check-agent-mode-real-runtime-semantic-contract.sh") {
		t.Fatalf("shell quality gate must invoke agent mode real runtime semantic contract")
	}
	if !strings.Contains(shell, "[quality-gate][agent-mode-real-runtime-semantic-contract]") {
		t.Fatalf("shell quality gate must expose real runtime semantic contract blocking label")
	}
	if !strings.Contains(shell, "check-agent-mode-readme-runtime-sync-contract.sh") {
		t.Fatalf("shell quality gate must invoke agent mode readme runtime sync contract")
	}
	if !strings.Contains(shell, "[quality-gate][agent-mode-readme-runtime-sync-contract]") {
		t.Fatalf("shell quality gate must expose readme runtime sync contract blocking label")
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
	if !strings.Contains(ps, "check-agent-mode-real-runtime-semantic-contract.ps1") {
		t.Fatalf("powershell quality gate must invoke agent mode real runtime semantic contract")
	}
	if !strings.Contains(ps, "[quality-gate] agent mode real runtime semantic contract") {
		t.Fatalf("powershell quality gate must expose real runtime semantic contract step label")
	}
	if !strings.Contains(ps, "check-agent-mode-readme-runtime-sync-contract.ps1") {
		t.Fatalf("powershell quality gate must invoke agent mode readme runtime sync contract")
	}
	if !strings.Contains(ps, "[quality-gate] agent mode readme runtime sync contract") {
		t.Fatalf("powershell quality gate must expose readme runtime sync step label")
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
		"pattern -> phase -> a71_scope -> a71_status -> semantic_anchor -> runtime_path_evidence -> expected_verification_markers -> minimal -> production-ish -> contracts -> gates -> replay",
		"| `rag-hybrid-retrieval` |",
		"| `context-governed-reference-first` |",
		"| `custom-adapter-health-readiness-circuit` |",
		"minimal:",
		"production-ish:",
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
	if !strings.Contains(playbook, "## Variant Distinction Rules") {
		t.Fatalf("playbook must include variant distinction section")
	}
	if !strings.Contains(playbook, "## Semantic Evidence Fields") {
		t.Fatalf("playbook must include semantic evidence fields section")
	}
	if !strings.Contains(playbook, "## Production Migration Checklist") {
		t.Fatalf("playbook must include production migration checklist")
	}
	if !strings.Contains(stability, "\"max_p95_ms\"") {
		t.Fatalf("stability baseline must include max_p95_ms threshold")
	}
	if !strings.Contains(stability, "\"max_flaky_rate\"") {
		t.Fatalf("stability baseline must include max_flaky_rate threshold")
	}
	if !strings.Contains(repoReadme, "check-agent-mode-real-runtime-semantic-contract") {
		t.Fatalf("repository README must include agent mode real runtime semantic gate")
	}
	if !strings.Contains(repoReadme, "check-agent-mode-readme-runtime-sync-contract") {
		t.Fatalf("repository README must include agent mode readme runtime sync gate")
	}
	if !strings.Contains(index, "Agent Mode Example Pack Mapping") {
		t.Fatalf("mainline contract index must include agent mode mapping section")
	}
	if !strings.Contains(index, "check-agent-mode-real-runtime-semantic-contract") {
		t.Fatalf("mainline contract index must include real runtime semantic gate mapping")
	}
	if !strings.Contains(index, "check-agent-mode-readme-runtime-sync-contract") {
		t.Fatalf("mainline contract index must include readme runtime sync gate mapping")
	}
}

func TestAgentModeMatrixRowCoverageAndReplayGateMapping(t *testing.T) {
	root := repoRoot(t)
	matrixPath := filepath.Join(root, "examples", "agent-modes", "MATRIX.md")
	raw, err := os.ReadFile(matrixPath)
	if err != nil {
		t.Fatalf("read matrix: %v", err)
	}

	lines := strings.Split(string(raw), "\n")
	rowCount := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "| `") {
			continue
		}
		parts := strings.Split(trimmed, "|")
		if len(parts) < 13 {
			t.Fatalf("matrix row has insufficient columns: %q", trimmed)
		}
		rowCount++

		pattern := strings.TrimSpace(parts[1])
		runtimePathEvidence := strings.TrimSpace(parts[6])
		expectedMarkers := strings.TrimSpace(parts[7])
		contracts := strings.TrimSpace(parts[10])
		gates := strings.TrimSpace(parts[11])
		replay := strings.TrimSpace(parts[12])

		if runtimePathEvidence == "" || runtimePathEvidence == "-" {
			t.Fatalf("pattern %s missing runtime path evidence", pattern)
		}
		if expectedMarkers == "" || expectedMarkers == "-" {
			t.Fatalf("pattern %s missing expected verification markers", pattern)
		}
		if !strings.Contains(expectedMarkers, "minimal:") || !strings.Contains(expectedMarkers, "production-ish:") {
			t.Fatalf("pattern %s expected markers must include minimal and production-ish markers", pattern)
		}
		if contracts == "" || contracts == "-" {
			t.Fatalf("pattern %s missing contracts mapping", pattern)
		}
		if gates == "" || gates == "-" {
			t.Fatalf("pattern %s missing gates mapping", pattern)
		}
		if replay == "" || replay == "-" {
			t.Fatalf("pattern %s missing replay mapping", pattern)
		}

		gateTokens := strings.Split(gates, ";")
		for _, gateToken := range gateTokens {
			gateToken = strings.TrimSpace(gateToken)
			gateToken = strings.Trim(gateToken, "`")
			if gateToken == "" || gateToken == "-" {
				continue
			}
			if !strings.HasSuffix(gateToken, ".*") {
				t.Fatalf("pattern %s gate token must use wildcard suffix: %q", pattern, gateToken)
			}
			base := strings.TrimSuffix(gateToken, ".*")
			shellPath := filepath.Join(root, "scripts", base+".sh")
			psPath := filepath.Join(root, "scripts", base+".ps1")
			if _, err := os.Stat(shellPath); err != nil {
				t.Fatalf("pattern %s gate shell script missing for token %q: %v", pattern, gateToken, err)
			}
			if _, err := os.Stat(psPath); err != nil {
				t.Fatalf("pattern %s gate powershell script missing for token %q: %v", pattern, gateToken, err)
			}
		}
	}
	if rowCount != 28 {
		t.Fatalf("expected 28 mode rows in matrix, got %d", rowCount)
	}
}
