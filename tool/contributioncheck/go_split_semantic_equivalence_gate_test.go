package contributioncheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGoSplitSemanticEquivalenceGateScriptParity(t *testing.T) {
	root := repoRoot(t)
	shellPath := filepath.Join(root, "scripts", "check-go-split-semantic-equivalence.sh")
	psPath := filepath.Join(root, "scripts", "check-go-split-semantic-equivalence.ps1")

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
		"go-split-strong-check",
		"TimeoutResolutionContractRunStreamAndMemoryFileParity",
		"TimeoutResolutionContractReplayIdempotency",
		"AssemblerContextStage2MemoryGovernanceDiagnosticsFields",
		"StoreRunMemoryGovernanceAdditiveFieldsPersistAndReplayIdempotent",
		"RuntimeRecorderParsesMemoryGovernanceAdditiveFields",
		"RuntimeRecorderMemoryGovernanceParserCompatibilityAdditiveNullableDefault",
		"ReplayContractPrimaryReasonArbitrationFixture",
		"PrimaryReasonArbitrationReplayContract",
	}
	for _, token := range requiredTokens {
		if !strings.Contains(shell, token) {
			t.Fatalf("shell go-split strong check gate missing token %q", token)
		}
		if !strings.Contains(ps, token) {
			t.Fatalf("powershell go-split strong check gate missing token %q", token)
		}
	}
	if !strings.Contains(shell, "set -euo pipefail") {
		t.Fatalf("shell go-split strong check gate must use set -euo pipefail")
	}
	if !strings.Contains(ps, "lib/native-strict.ps1") || !strings.Contains(ps, "Invoke-NativeStrict") {
		t.Fatalf("powershell go-split strong check gate must use strict native helper")
	}
}

func TestQualityGateIncludesGoSplitSemanticEquivalenceGate(t *testing.T) {
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
	if !strings.Contains(shell, "check-go-split-semantic-equivalence.sh") {
		t.Fatalf("shell quality gate must invoke go-split semantic equivalence strong checks")
	}
	if !strings.Contains(shell, "[quality-gate][go-split-strong-check]") {
		t.Fatalf("shell quality gate must expose go-split strong check failure label")
	}
	if !strings.Contains(ps, "check-go-split-semantic-equivalence.ps1") {
		t.Fatalf("powershell quality gate must invoke go-split semantic equivalence strong checks")
	}
	if !strings.Contains(ps, "[quality-gate] go split semantic equivalence strong checks") {
		t.Fatalf("powershell quality gate must expose go-split strong check step label")
	}
}
