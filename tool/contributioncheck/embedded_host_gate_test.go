package contributioncheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmbeddedHostContractGateShellPowerShellParity(t *testing.T) {
	root := repoRoot(t)
	shell, err := os.ReadFile(filepath.Join(root, "scripts", "check-embedded-host-contract.sh"))
	if err != nil {
		t.Fatal(err)
	}
	ps, err := os.ReadFile(filepath.Join(root, "scripts", "check-embedded-host-contract.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	sh, pw := string(shell), string(ps)
	for _, marker := range []string{"source_owner", "non_goals", "bounded_state", "embedded_host_protocol.v1", "go test ./core/types ./core/runner ./host/..."} {
		if !strings.Contains(sh, marker) || !strings.Contains(pw, marker) {
			t.Fatalf("host contract gates must both contain %q", marker)
		}
	}
	if !strings.Contains(sh, "set -euo pipefail") || !strings.Contains(pw, "Set-StrictMode") || !strings.Contains(pw, "Invoke-NativeStrict") {
		t.Fatal("host contract gates must use strict fail-fast execution")
	}
}

func TestQualityGateIncludesEmbeddedHostContractGate(t *testing.T) {
	root := repoRoot(t)
	for _, tc := range []struct {
		name, marker string
	}{
		{"scripts/check-quality-gate.sh", "check-embedded-host-contract.sh"},
		{"scripts/check-quality-gate.ps1", "check-embedded-host-contract.ps1"},
	} {
		raw, err := os.ReadFile(filepath.Join(root, tc.name))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(raw), tc.marker) {
			t.Fatalf("%s must invoke embedded host contract gate", tc.name)
		}
	}
}

func TestEmbeddedHostExampleGovernanceMarkersStaySynchronized(t *testing.T) {
	root := repoRoot(t)
	paths := []string{
		filepath.Join(root, "examples", "agent-modes", "MATRIX.md"),
		filepath.Join(root, "examples", "agent-modes", "realtime-interrupt-resume", "README.md"),
		filepath.Join(root, "examples", "agent-modes", "realtime-interrupt-resume", "minimal", "README.md"),
		filepath.Join(root, "examples", "agent-modes", "realtime-interrupt-resume", "production-ish", "README.md"),
	}
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text := string(raw)
		for _, marker := range []string{"embedded_host.command_response_event_correlation", "host_negotiation_completed", "host_command_correlated", "host_terminal_authoritative"} {
			if !strings.Contains(text, marker) {
				t.Fatalf("%s missing governed host marker %q", path, marker)
			}
		}
	}
	roadmap, err := os.ReadFile(filepath.Join(root, "docs", "development-roadmap.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(roadmap), "establish-embedded-host-command-response-and-event-correlation-contract") {
		t.Fatal("roadmap must retain active embedded-host proposal reference")
	}
}
