package contributioncheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAgentRuntimeProtocolGateExcludesRatifiedProtocolSpec(t *testing.T) {
	root := repoRoot(t)
	paths := []string{
		filepath.Join(root, "scripts", "check-agent-runtime-protocol-contract.sh"),
		filepath.Join(root, "scripts", "check-agent-runtime-protocol-contract.ps1"),
	}

	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if !strings.Contains(string(raw), "!openspec/specs/agent-runtime-protocol-contract/**") {
			t.Fatalf("%s must exclude the ratified protocol spec from hosted-dependency scanning", path)
		}
	}
}

func TestAgentRuntimeProtocolGateChecksProjectionContractSurface(t *testing.T) {
	root := repoRoot(t)
	for _, name := range []string{"check-agent-runtime-protocol-contract.sh", "check-agent-runtime-protocol-contract.ps1"} {
		raw, err := os.ReadFile(filepath.Join(root, "scripts", name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		text := string(raw)
		for _, marker := range []string{"capability", "context", "action", "admission", "Run/Stream", "control_plane_absent"} {
			if !strings.Contains(strings.ToLower(text), strings.ToLower(marker)) {
				t.Fatalf("%s must assert %q", name, marker)
			}
		}
	}
}
