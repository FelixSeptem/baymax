package contributioncheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHostExecutionPlanOwnershipBoundary(t *testing.T) {
	root := repoRoot(t)
	for _, name := range []string{"orchestration/planbridge", "tool/diagnosticsreplay/host_execution_plan.go"} {
		var body string
		if strings.HasSuffix(name, ".go") {
			raw, err := os.ReadFile(filepath.Join(root, name))
			if err != nil {
				t.Fatal(err)
			}
			body = string(raw)
		} else {
			entries, err := os.ReadDir(filepath.Join(root, name))
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if strings.HasSuffix(entry.Name(), ".go") && !strings.HasSuffix(entry.Name(), "_test.go") {
					raw, err := os.ReadFile(filepath.Join(root, name, entry.Name()))
					if err != nil {
						t.Fatal(err)
					}
					body += string(raw)
				}
			}
		}
		for _, forbidden := range []string{"github.com/FelixSeptem/baymax/observability", "github.com/FelixSeptem/baymax/mcp/", "go func", ".Resume("} {
			if strings.Contains(body, forbidden) {
				t.Fatalf("%s must not own %q", name, forbidden)
			}
		}
	}
	for _, script := range []string{"scripts/check-host-execution-plan-contract.sh", "scripts/check-host-execution-plan-contract.ps1"} {
		raw, err := os.ReadFile(filepath.Join(root, script))
		if err != nil {
			t.Fatal(err)
		}
		for _, required := range []string{"TestHostExecutionPlan", "testdata/host_execution_plan.invalid.json", "examples/host-execution-plan"} {
			if !strings.Contains(string(raw), required) {
				t.Fatalf("%s missing %q", script, required)
			}
		}
	}
}
