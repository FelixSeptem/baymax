package contributioncheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecutionGraphGateScriptsStayPairedAndOffline(t *testing.T) {
	root := repoRoot(t)
	shell, err := os.ReadFile(filepath.Join(root, "scripts", "check-execution-graph-contract.sh"))
	if err != nil {
		t.Fatalf("read shell gate: %v", err)
	}
	powershell, err := os.ReadFile(filepath.Join(root, "scripts", "check-execution-graph-contract.ps1"))
	if err != nil {
		t.Fatalf("read PowerShell gate: %v", err)
	}
	for _, token := range []string{
		"orchestration/graph",
		"orchestration/workflow",
		"orchestration/teams",
		"orchestration/composer",
		"examples/execution-graph",
		"OfflineExample",
		"owner digest",
	} {
		if !strings.Contains(string(shell), token) || !strings.Contains(string(powershell), token) {
			t.Fatalf("paired execution graph gates must include %q", token)
		}
	}
	if !strings.Contains(string(shell), "set -euo pipefail") {
		t.Fatal("shell gate must fail fast")
	}
	if !strings.Contains(string(powershell), "lib/native-strict.ps1") || !strings.Contains(string(powershell), "Invoke-NativeStrict") {
		t.Fatal("PowerShell gate must use the strict native helper")
	}
	if strings.Contains(string(shell), "go run") || strings.Contains(string(powershell), "go run") {
		t.Fatal("the contract gate must not execute the example as a live orchestration")
	}
}

func TestQualityGateIncludesExecutionGraphContract(t *testing.T) {
	root := repoRoot(t)
	shell, err := os.ReadFile(filepath.Join(root, "scripts", "check-quality-gate.sh"))
	if err != nil {
		t.Fatal(err)
	}
	powershell, err := os.ReadFile(filepath.Join(root, "scripts", "check-quality-gate.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(shell), "check-execution-graph-contract.sh") || !strings.Contains(string(shell), "[quality-gate][execution-graph-contract]") {
		t.Fatal("shell quality gate must include the blocking execution graph contract")
	}
	if !strings.Contains(string(powershell), "check-execution-graph-contract.ps1") || !strings.Contains(string(powershell), "[quality-gate] execution graph contract suites") {
		t.Fatal("PowerShell quality gate must include the blocking execution graph contract")
	}
}
