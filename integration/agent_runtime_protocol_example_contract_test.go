package integration

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestAgentRuntimeProtocolExampleVariantsEmitRealMarkers(t *testing.T) {
	variants := []struct {
		name   string
		path   string
		marker string
	}{
		{name: "minimal", path: "./examples/agent-modes/agent-runtime-protocol-projection/minimal", marker: "protocol_checkpoint_mapped"},
		{name: "production-ish", path: "./examples/agent-modes/agent-runtime-protocol-projection/production-ish", marker: "protocol_invalid_transition_rejected"},
	}
	for _, variant := range variants {
		t.Run(variant.name, func(t *testing.T) {
			cmd := exec.Command("go", "run", variant.path)
			_, file, _, _ := runtime.Caller(0)
			cmd.Dir = filepath.Dir(filepath.Dir(file))
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("go run failed: %v\n%s", err, output)
			}
			text := string(output)
			required := []string{
				"verification.mainline_runtime_path=ok",
				"verification.semantic.phase=",
				"verification.semantic.anchor=agent_runtime_protocol.capability_context_admission",
				"verification.semantic.classification=agent_runtime_protocol.projection",
				"verification.semantic.runtime_path=",
				"verification.semantic.expected_markers=",
				"verification.semantic.governance=",
				"verification.semantic.marker_count=",
				"verification.semantic.marker." + variant.marker + "=ok",
				"result.final_answer=",
				"result.signature=",
			}
			for _, token := range required {
				if !strings.Contains(text, token) {
					t.Fatalf("output missing %q:\n%s", token, output)
				}
			}
			if variant.name == "minimal" && !strings.Contains(text, "verification.semantic.governance=baseline") {
				t.Fatalf("minimal example must report baseline governance:\n%s", output)
			}
			if variant.name == "production-ish" && !strings.Contains(text, "verification.semantic.governance=enforced") {
				t.Fatalf("production-ish example must report enforced governance:\n%s", output)
			}
			if !strings.Contains(text, "verification.mainline_runtime_path=ok") || !strings.Contains(text, "verification.semantic.marker."+variant.marker+"=ok") {
				t.Fatalf("output missing real markers:\n%s", output)
			}
			for _, marker := range []string{"protocol_descriptor_validated", "protocol_context_validated", "protocol_admission_projected"} {
				if !strings.Contains(text, "verification.semantic.marker."+marker+"=ok") {
					t.Fatalf("output missing projection marker %q:\n%s", marker, output)
				}
			}
		})
	}
}
