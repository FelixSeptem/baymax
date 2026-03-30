package contributioncheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSandboxRolloutGovernanceGateScriptParity(t *testing.T) {
	root := repoRoot(t)
	shellPath := filepath.Join(root, "scripts", "check-sandbox-rollout-governance-contract.sh")
	psPath := filepath.Join(root, "scripts", "check-sandbox-rollout-governance-contract.ps1")

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
		"SandboxRolloutPhaseTransitionValidation",
		"ManagerSandboxRolloutGovernanceRecordRunAutoFreeze",
		"ManagerSandboxRolloutUnfreezeRequiresCooldownAndToken",
		"ManagerSandboxCapacityActionDeterministicFromQueueAndInflight",
		"ManagerReadinessAdmissionSandboxCapacityPolicyMapping",
		"ComposerReadinessAdmissionSandbox(RolloutFrozenRunAndStreamEquivalent|CapacityThrottlePolicyParity|RolloutTimelineReasonParity)",
		"RuntimeRecorder(A52ParserCompatibilityAdditiveNullableDefault|ParsesA52RolloutGovernanceFields)",
		"ReplayContract(PrimaryReasonArbitrationFixtureSuccessAndDeterministicOutput|PrimaryReasonArbitrationFixtureDriftClassification|ArbitrationMixedA51A52Compatibility)",
	}
	for _, token := range requiredTokens {
		if !strings.Contains(shell, token) {
			t.Fatalf("shell rollout governance gate missing token %q", token)
		}
		if !strings.Contains(ps, token) {
			t.Fatalf("powershell rollout governance gate missing token %q", token)
		}
	}

	if !strings.Contains(shell, "set -euo pipefail") {
		t.Fatalf("shell rollout governance gate must use set -euo pipefail")
	}
	if !strings.Contains(ps, "lib/native-strict.ps1") || !strings.Contains(ps, "Invoke-NativeStrict") {
		t.Fatalf("powershell rollout governance gate must use strict native helper")
	}
	if strings.Contains(ps, "AllowFailure") {
		t.Fatalf("powershell rollout governance gate must not add AllowFailure exceptions")
	}
}

func TestQualityGateIncludesSandboxRolloutGovernanceGate(t *testing.T) {
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
	if !strings.Contains(shell, "check-sandbox-rollout-governance-contract.sh") {
		t.Fatalf("shell quality gate must invoke sandbox rollout governance gate")
	}
	if !strings.Contains(shell, "[quality-gate][sandbox-rollout-governance-contract]") {
		t.Fatalf("shell quality gate must expose blocking sandbox rollout governance failure label")
	}
	if !strings.Contains(ps, "check-sandbox-rollout-governance-contract.ps1") {
		t.Fatalf("powershell quality gate must invoke sandbox rollout governance gate")
	}
	if !strings.Contains(ps, "[quality-gate] sandbox rollout governance contract suites") {
		t.Fatalf("powershell quality gate must expose sandbox rollout governance step label")
	}
}
