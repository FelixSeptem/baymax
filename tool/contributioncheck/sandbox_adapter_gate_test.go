package contributioncheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSandboxAdapterConformanceGateScriptParity(t *testing.T) {
	root := repoRoot(t)
	shellPath := filepath.Join(root, "scripts", "check-sandbox-adapter-conformance-contract.sh")
	psPath := filepath.Join(root, "scripts", "check-sandbox-adapter-conformance-contract.ps1")

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
		"Test(ParseSandboxManifest|ActivateSandboxManifest|SandboxProfilePack)",
		"TestSandboxAdapterConformance",
		"TestManagerReadinessPreflightSandboxAdapter",
		"TestReplayContract(SandboxProfilePackTrack|MixedTracksBackwardCompatible|ProfileVersionValidation)",
	}
	for _, token := range requiredTokens {
		if !strings.Contains(shell, token) {
			t.Fatalf("shell sandbox adapter gate missing token %q", token)
		}
		if !strings.Contains(ps, token) {
			t.Fatalf("powershell sandbox adapter gate missing token %q", token)
		}
	}

	if !strings.Contains(shell, "set -euo pipefail") {
		t.Fatalf("shell sandbox adapter gate must use set -euo pipefail")
	}
	if !strings.Contains(ps, "lib/native-strict.ps1") || !strings.Contains(ps, "Invoke-NativeStrict") {
		t.Fatalf("powershell sandbox adapter gate must use strict native helper")
	}
	if strings.Contains(ps, "AllowFailure") {
		t.Fatalf("powershell sandbox adapter gate must not add AllowFailure exceptions")
	}
}

func TestQualityGateIncludesSandboxAdapterConformanceGate(t *testing.T) {
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
	if !strings.Contains(shell, "check-sandbox-adapter-conformance-contract.sh") {
		t.Fatalf("shell quality gate must invoke sandbox adapter conformance gate")
	}
	if !strings.Contains(shell, "[quality-gate][sandbox-adapter-conformance-contract]") {
		t.Fatalf("shell quality gate must expose blocking sandbox adapter failure label")
	}
	if !strings.Contains(ps, "check-sandbox-adapter-conformance-contract.ps1") {
		t.Fatalf("powershell quality gate must invoke sandbox adapter conformance gate")
	}
	if !strings.Contains(ps, "[quality-gate] sandbox adapter conformance contract") {
		t.Fatalf("powershell quality gate must expose sandbox adapter step label")
	}
}
