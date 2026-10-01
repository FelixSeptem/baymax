package contributioncheck

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSandboxLifecycleSuccessCostEvidenceRemainsReferenceOnly(t *testing.T) {
	root := repoRoot(t)
	source := strings.ToLower(mustRead(t, filepath.Join(root, "tool", "diagnosticsreplay", "sandbox_lifecycle_success_cost_evidence.go")))
	for _, forbidden := range []string{
		"net/http", "os/exec", "sandbox-exec", "bubblewrap", "wfp", "proxy", "service account", "credential store",
		"global mutable session", "runtime config", "runtime_recorder", "security.sandbox", "admission state machine",
	} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("evidence evaluator contains forbidden boundary marker %q", forbidden)
		}
	}
	fixture := strings.ToLower(mustRead(t, filepath.Join(root, "tool", "diagnosticsreplay", "testdata", "sandbox_lifecycle_success_cost_evidence.v1.json")))
	for _, forbidden := range []string{"endpoint", "sk-", "token", "raw_response", "raw_payload", "password", "secret", "command"} {
		if strings.Contains(fixture, forbidden) {
			t.Fatalf("fixture contains forbidden material %q", forbidden)
		}
	}
}

func TestSandboxLifecycleSuccessCostEvidenceGateScriptsPreserveParity(t *testing.T) {
	root := repoRoot(t)
	shell := mustRead(t, filepath.Join(root, "scripts", "check-sandbox-lifecycle-success-cost-evidence-contract.sh"))
	powershell := mustRead(t, filepath.Join(root, "scripts", "check-sandbox-lifecycle-success-cost-evidence-contract.ps1"))
	for _, token := range []string{
		"sandbox_lifecycle_success_cost_evidence.v1",
		"SandboxLifecycleSuccessCostEvidence",
		"tool/contributioncheck",
		"offline",
	} {
		if !strings.Contains(shell, token) || !strings.Contains(powershell, token) {
			t.Fatalf("shell/powershell gates must both contain %q", token)
		}
	}
	if !strings.Contains(shell, "set -euo pipefail") {
		t.Fatal("shell gate must use strict mode")
	}
	if !strings.Contains(powershell, "lib/native-strict.ps1") || !strings.Contains(powershell, "Invoke-NativeStrict") {
		t.Fatal("powershell gate must use strict native helper")
	}
}

func TestQualityGateIncludesSandboxLifecycleSuccessCostEvidenceGate(t *testing.T) {
	root := repoRoot(t)
	shell := mustRead(t, filepath.Join(root, "scripts", "check-quality-gate.sh"))
	powershell := mustRead(t, filepath.Join(root, "scripts", "check-quality-gate.ps1"))
	if !strings.Contains(shell, "check-sandbox-lifecycle-success-cost-evidence-contract.sh") || !strings.Contains(shell, "sandbox-lifecycle-success-cost-evidence") {
		t.Fatal("shell quality gate must invoke lifecycle-success-cost gate")
	}
	if !strings.Contains(powershell, "check-sandbox-lifecycle-success-cost-evidence-contract.ps1") || !strings.Contains(powershell, "sandbox lifecycle-success-cost evidence") {
		t.Fatal("powershell quality gate must invoke lifecycle-success-cost gate")
	}
}
