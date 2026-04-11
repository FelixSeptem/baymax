package contributioncheck

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
		"pattern -> phase -> a71_scope -> a71_status -> doc-baseline-ready -> impl-ready -> semantic_anchor -> runtime_path_evidence -> expected_verification_markers -> failure_rollback_ref -> minimal -> production-ish -> contracts -> gates -> replay",
		"| yes | yes |",
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

func TestAgentModeAntiTemplateContractGateScriptParity(t *testing.T) {
	root := repoRoot(t)
	shellPath := filepath.Join(root, "scripts", "check-agent-mode-anti-template-contract.sh")
	psPath := filepath.Join(root, "scripts", "check-agent-mode-anti-template-contract.ps1")

	shellRaw, err := os.ReadFile(shellPath)
	if err != nil {
		t.Fatalf("read shell anti-template gate: %v", err)
	}
	psRaw, err := os.ReadFile(psPath)
	if err != nil {
		t.Fatalf("read powershell anti-template gate: %v", err)
	}

	shell := string(shellRaw)
	ps := string(psRaw)
	requiredTokens := []string{
		"agent-mode-template-skeleton-detected",
		"agent-mode-semantic-ownership-missing",
		"agent-mode-variant-behavior-not-diverged",
		"BAYMAX_AGENT_MODE_TEMPLATE_HOMOGENEITY_THRESHOLD",
		"semantic_example.go",
		"runtimeexample.MustRun(",
	}
	for _, token := range requiredTokens {
		if !strings.Contains(shell, token) {
			t.Fatalf("shell anti-template gate missing token %q", token)
		}
		if !strings.Contains(ps, token) {
			t.Fatalf("powershell anti-template gate missing token %q", token)
		}
	}
	if !strings.Contains(shell, "set -euo pipefail") {
		t.Fatalf("shell anti-template gate must use set -euo pipefail")
	}
	if !strings.Contains(ps, "Set-StrictMode -Version Latest") {
		t.Fatalf("powershell anti-template gate must use strict mode")
	}
}

func TestAgentModeDocFirstDeliveryContractGateScriptParity(t *testing.T) {
	root := repoRoot(t)
	shellPath := filepath.Join(root, "scripts", "check-agent-mode-doc-first-delivery-contract.sh")
	psPath := filepath.Join(root, "scripts", "check-agent-mode-doc-first-delivery-contract.ps1")

	shellRaw, err := os.ReadFile(shellPath)
	if err != nil {
		t.Fatalf("read shell doc-first gate: %v", err)
	}
	psRaw, err := os.ReadFile(psPath)
	if err != nil {
		t.Fatalf("read powershell doc-first gate: %v", err)
	}

	shell := string(shellRaw)
	ps := string(psRaw)
	requiredTokens := []string{
		"agent-mode-doc-first-baseline-missing",
		"agent-mode-doc-required-sections-missing",
		"examples/agent-modes/doc-baseline-freeze.md",
		"doc-baseline-ready",
		"impl-ready",
		"failure_rollback_ref",
		"## Variant Delta (vs minimal)",
	}
	for _, token := range requiredTokens {
		if !strings.Contains(shell, token) {
			t.Fatalf("shell doc-first gate missing token %q", token)
		}
		if !strings.Contains(ps, token) {
			t.Fatalf("powershell doc-first gate missing token %q", token)
		}
	}
	if !strings.Contains(shell, "set -euo pipefail") {
		t.Fatalf("shell doc-first gate must use set -euo pipefail")
	}
	if !strings.Contains(ps, "Set-StrictMode -Version Latest") {
		t.Fatalf("powershell doc-first gate must use strict mode")
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
	if !strings.Contains(shell, "check-agent-mode-anti-template-contract.sh") {
		t.Fatalf("shell quality gate must invoke agent mode anti-template contract")
	}
	if !strings.Contains(shell, "[quality-gate][agent-mode-anti-template-contract]") {
		t.Fatalf("shell quality gate must expose anti-template contract blocking label")
	}
	if !strings.Contains(shell, "check-agent-mode-doc-first-delivery-contract.sh") {
		t.Fatalf("shell quality gate must invoke agent mode doc-first delivery contract")
	}
	if !strings.Contains(shell, "[quality-gate][agent-mode-doc-first-delivery-contract]") {
		t.Fatalf("shell quality gate must expose doc-first delivery contract blocking label")
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
	if !strings.Contains(ps, "check-agent-mode-anti-template-contract.ps1") {
		t.Fatalf("powershell quality gate must invoke agent mode anti-template contract")
	}
	if !strings.Contains(ps, "[quality-gate] agent mode anti-template contract") {
		t.Fatalf("powershell quality gate must expose anti-template contract step label")
	}
	if !strings.Contains(ps, "check-agent-mode-doc-first-delivery-contract.ps1") {
		t.Fatalf("powershell quality gate must invoke agent mode doc-first delivery contract")
	}
	if !strings.Contains(ps, "[quality-gate] agent mode doc-first delivery contract") {
		t.Fatalf("powershell quality gate must expose doc-first delivery contract step label")
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
		"pattern -> phase -> a71_scope -> a71_status -> doc-baseline-ready -> impl-ready -> semantic_anchor -> runtime_path_evidence -> expected_verification_markers -> failure_rollback_ref -> minimal -> production-ish -> contracts -> gates -> replay",
		"doc-baseline-ready",
		"failure rollback ref",
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
	if !strings.Contains(repoReadme, "check-agent-mode-anti-template-contract") {
		t.Fatalf("repository README must include agent mode anti-template gate")
	}
	if !strings.Contains(repoReadme, "check-agent-mode-doc-first-delivery-contract") {
		t.Fatalf("repository README must include agent mode doc-first delivery gate")
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
	if !strings.Contains(index, "check-agent-mode-anti-template-contract") {
		t.Fatalf("mainline contract index must include anti-template gate mapping")
	}
	if !strings.Contains(index, "check-agent-mode-doc-first-delivery-contract") {
		t.Fatalf("mainline contract index must include doc-first gate mapping")
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
		if len(parts) < 16 {
			t.Fatalf("matrix row has insufficient columns: %q", trimmed)
		}
		rowCount++

		pattern := strings.TrimSpace(parts[1])
		docBaselineReady := strings.TrimSpace(parts[5])
		implReady := strings.TrimSpace(parts[6])
		semanticAnchor := strings.TrimSpace(parts[7])
		runtimePathEvidence := strings.TrimSpace(parts[8])
		expectedMarkers := strings.TrimSpace(parts[9])
		failureRollbackRef := strings.TrimSpace(parts[10])
		minimalPath := strings.TrimSpace(parts[11])
		productionPath := strings.TrimSpace(parts[12])
		contracts := strings.TrimSpace(parts[13])
		gates := strings.TrimSpace(parts[14])
		replay := strings.TrimSpace(parts[15])

		if docBaselineReady != "yes" || implReady != "yes" {
			t.Fatalf("pattern %s must have doc-baseline-ready and impl-ready set to yes", pattern)
		}
		if semanticAnchor == "" || semanticAnchor == "-" {
			t.Fatalf("pattern %s missing semantic anchor", pattern)
		}

		if runtimePathEvidence == "" || runtimePathEvidence == "-" {
			t.Fatalf("pattern %s missing runtime path evidence", pattern)
		}
		if expectedMarkers == "" || expectedMarkers == "-" {
			t.Fatalf("pattern %s missing expected verification markers", pattern)
		}
		if !strings.Contains(expectedMarkers, "minimal:") || !strings.Contains(expectedMarkers, "production-ish:") {
			t.Fatalf("pattern %s expected markers must include minimal and production-ish markers", pattern)
		}
		if failureRollbackRef == "" || failureRollbackRef == "-" || !strings.Contains(failureRollbackRef, "README.md") {
			t.Fatalf("pattern %s missing failure rollback references", pattern)
		}
		if minimalPath == "" || minimalPath == "-" || productionPath == "" || productionPath == "-" {
			t.Fatalf("pattern %s missing variant path mapping", pattern)
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

func TestAgentModeAntiTemplateContractExecutionPassFailAndExceptionalBranches(t *testing.T) {
	root := repoRoot(t)
	psScript := filepath.Join(root, "scripts", "check-agent-mode-anti-template-contract.ps1")

	output, code := runPowerShellScript(t, root, psScript, nil)
	if code != 0 {
		t.Fatalf("anti-template ps script should pass baseline, exit=%d output=%s", code, output)
	}
	if !strings.Contains(output, "[agent-mode-anti-template-contract] passed") {
		t.Fatalf("anti-template ps pass output missing success marker: %s", output)
	}

	mainPath := filepath.Join(root, "examples", "agent-modes", "rag-hybrid-retrieval", "minimal", "main.go")
	original := readFileStrict(t, mainPath)
	mutated := original + "\n// regression sentinel: runtimeexample.MustRun(\n"
	writeFileStrict(t, mainPath, mutated)
	defer writeFileStrict(t, mainPath, original)

	output, code = runPowerShellScript(t, root, psScript, nil)
	if code == 0 {
		t.Fatalf("anti-template ps script should fail when wrapper sentinel exists")
	}
	if !strings.Contains(output, "[agent-mode-anti-template-contract][agent-mode-template-skeleton-detected]") {
		t.Fatalf("anti-template ps fail output missing classification code: %s", output)
	}

	output, code = runPowerShellScript(t, root, psScript, map[string]string{
		"BAYMAX_AGENT_MODE_TEMPLATE_HOMOGENEITY_THRESHOLD": "1",
	})
	if code == 0 {
		t.Fatalf("anti-template ps script should fail on invalid threshold")
	}
	if !strings.Contains(output, "BAYMAX_AGENT_MODE_TEMPLATE_HOMOGENEITY_THRESHOLD must be integer >= 2") {
		t.Fatalf("anti-template ps exceptional branch output mismatch: %s", output)
	}
}

func TestAgentModeDocFirstDeliveryContractExecutionPassFailAndExceptionalBranches(t *testing.T) {
	root := repoRoot(t)
	psScript := filepath.Join(root, "scripts", "check-agent-mode-doc-first-delivery-contract.ps1")

	output, code := runPowerShellScript(t, root, psScript, nil)
	if code != 0 {
		t.Fatalf("doc-first ps script should pass baseline, exit=%d output=%s", code, output)
	}
	if !strings.Contains(output, "[agent-mode-doc-first-delivery-contract] passed") {
		t.Fatalf("doc-first ps pass output missing success marker: %s", output)
	}

	matrixPath := filepath.Join(root, "examples", "agent-modes", "MATRIX.md")
	originalMatrix := readFileStrict(t, matrixPath)
	mutatedMatrix := strings.ReplaceAll(originalMatrix, "doc-baseline-ready", "doc_ready_state")
	writeFileStrict(t, matrixPath, mutatedMatrix)
	output, code = runPowerShellScript(t, root, psScript, nil)
	writeFileStrict(t, matrixPath, originalMatrix)
	if code == 0 {
		t.Fatalf("doc-first ps script should fail when doc-first baseline columns are missing")
	}
	if !strings.Contains(output, "[agent-mode-doc-first-delivery-contract][agent-mode-doc-first-baseline-missing]") {
		t.Fatalf("doc-first ps missing baseline branch output mismatch: %s", output)
	}

	readmePath := filepath.Join(root, "examples", "agent-modes", "rag-hybrid-retrieval", "minimal", "README.md")
	original := readFileStrict(t, readmePath)
	if !strings.Contains(original, "## Run") {
		t.Fatalf("target README missing expected section token ## Run")
	}
	mutated := strings.Replace(original, "## Run", "## Entry", 1)
	writeFileStrict(t, readmePath, mutated)
	defer writeFileStrict(t, readmePath, original)

	output, code = runPowerShellScript(t, root, psScript, nil)
	if code == 0 {
		t.Fatalf("doc-first ps script should fail when required README section is missing")
	}
	if !strings.Contains(output, "[agent-mode-doc-first-delivery-contract][agent-mode-doc-required-sections-missing]") {
		t.Fatalf("doc-first ps missing section branch output mismatch: %s", output)
	}
}

func TestAgentModeA72ShellPowerShellParityOnClassifications(t *testing.T) {
	root := repoRoot(t)
	bashPath, ok := findBashExecutable()
	if !ok {
		t.Skip("bash executable not found; skip shell/powershell parity runtime check")
	}

	antiPS := filepath.Join(root, "scripts", "check-agent-mode-anti-template-contract.ps1")
	antiSH := filepath.Join(root, "scripts", "check-agent-mode-anti-template-contract.sh")
	docPS := filepath.Join(root, "scripts", "check-agent-mode-doc-first-delivery-contract.ps1")
	docSH := filepath.Join(root, "scripts", "check-agent-mode-doc-first-delivery-contract.sh")

	psOutput, psCode := runPowerShellScript(t, root, antiPS, nil)
	shOutput, shCode := runBashScript(t, root, bashPath, antiSH, nil)
	assertSameOutcomeAndClassification(t, psCode, psOutput, shCode, shOutput, "[agent-mode-anti-template-contract]", "")

	mainPath := filepath.Join(root, "examples", "agent-modes", "rag-hybrid-retrieval", "minimal", "main.go")
	originalMain := readFileStrict(t, mainPath)
	writeFileStrict(t, mainPath, originalMain+"\n// parity sentinel: runtimeexample.MustRun(\n")
	defer writeFileStrict(t, mainPath, originalMain)

	psOutput, psCode = runPowerShellScript(t, root, antiPS, nil)
	shOutput, shCode = runBashScript(t, root, bashPath, antiSH, nil)
	assertSameOutcomeAndClassification(
		t,
		psCode,
		psOutput,
		shCode,
		shOutput,
		"[agent-mode-anti-template-contract]",
		"[agent-mode-template-skeleton-detected]",
	)

	matrixPath := filepath.Join(root, "examples", "agent-modes", "MATRIX.md")
	originalMatrix := readFileStrict(t, matrixPath)
	mutatedMatrix := strings.ReplaceAll(originalMatrix, "doc-baseline-ready", "doc_ready_state")
	writeFileStrict(t, matrixPath, mutatedMatrix)
	defer writeFileStrict(t, matrixPath, originalMatrix)

	psOutput, psCode = runPowerShellScript(t, root, docPS, nil)
	shOutput, shCode = runBashScript(t, root, bashPath, docSH, nil)
	writeFileStrict(t, matrixPath, originalMatrix)
	assertSameOutcomeAndClassification(
		t,
		psCode,
		psOutput,
		shCode,
		shOutput,
		"[agent-mode-doc-first-delivery-contract]",
		"[agent-mode-doc-first-baseline-missing]",
	)
}

func assertSameOutcomeAndClassification(
	t *testing.T,
	psCode int,
	psOutput string,
	shCode int,
	shOutput string,
	prefix string,
	classification string,
) {
	t.Helper()
	psPass := psCode == 0
	shPass := shCode == 0
	if psPass != shPass {
		t.Fatalf("shell/powershell result diverged: ps(exit=%d) sh(exit=%d)\nps=%s\nsh=%s", psCode, shCode, psOutput, shOutput)
	}
	if !strings.Contains(psOutput, prefix) || !strings.Contains(shOutput, prefix) {
		t.Fatalf("shell/powershell outputs missing expected prefix %q\nps=%s\nsh=%s", prefix, psOutput, shOutput)
	}
	if classification != "" {
		if !strings.Contains(psOutput, classification) || !strings.Contains(shOutput, classification) {
			t.Fatalf(
				"shell/powershell outputs missing classification %q\nps(exit=%d)=%s\nsh(exit=%d)=%s",
				classification,
				psCode,
				psOutput,
				shCode,
				shOutput,
			)
		}
	}
}

func runPowerShellScript(t *testing.T, root string, scriptPath string, extraEnv map[string]string) (string, int) {
	t.Helper()
	cmd := exec.Command("pwsh", "-NoLogo", "-NoProfile", "-File", scriptPath)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), formatEnvMap(extraEnv)...)
	out, err := cmd.CombinedOutput()
	exitCode := commandExitCode(err)
	return string(out), exitCode
}

func runBashScript(t *testing.T, root string, bashPath string, scriptPath string, extraEnv map[string]string) (string, int) {
	t.Helper()
	cmd := exec.Command(bashPath, scriptPath)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), formatEnvMap(extraEnv)...)
	out, err := cmd.CombinedOutput()
	raw := string(out)
	if isBashInfrastructureFailure(raw) {
		t.Skipf("bash infrastructure failure during parity execution: %s", raw)
	}
	exitCode := commandExitCode(err)
	return raw, exitCode
}

func findBashExecutable() (string, bool) {
	if override := strings.TrimSpace(os.Getenv("BAYMAX_TEST_BASH_PATH")); override != "" {
		if isUsableBash(override) {
			return override, true
		}
	}

	candidates := []string{
		`D:\git\Git\bin\bash.exe`,
		`D:\git\Git\usr\bin\bash.exe`,
		`C:\Program Files\Git\bin\bash.exe`,
		`C:\Program Files\Git\usr\bin\bash.exe`,
	}
	for _, candidate := range candidates {
		if isUsableBash(candidate) {
			return candidate, true
		}
	}

	if path, err := exec.LookPath("bash"); err == nil {
		if isUsableBash(path) {
			return path, true
		}
	}

	return "", false
}

func isUsableBash(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	if _, err := os.Stat(path); err != nil {
		return false
	}
	cmd := exec.Command(path, "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(out)), "gnu bash")
}

func formatEnvMap(extraEnv map[string]string) []string {
	if len(extraEnv) == 0 {
		return nil
	}
	formatted := make([]string, 0, len(extraEnv))
	for key, value := range extraEnv {
		formatted = append(formatted, fmt.Sprintf("%s=%s", key, value))
	}
	return formatted
}

func readFileStrict(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %s: %v", path, err)
	}
	return string(raw)
}

func writeFileStrict(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}

func commandExitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}

func isBashInfrastructureFailure(output string) bool {
	sanitized := strings.ReplaceAll(output, "\x00", "")
	lower := strings.ToLower(strings.TrimSpace(sanitized))
	if lower == "" {
		return false
	}
	fragments := []string{
		"fatal error - couldn't create signal pipe",
		"win32 error 5",
		"e_accessdenied",
	}
	for _, frag := range fragments {
		if strings.Contains(lower, frag) {
			return runtime.GOOS == "windows"
		}
	}
	return false
}
