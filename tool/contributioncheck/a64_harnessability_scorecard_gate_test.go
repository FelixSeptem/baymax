package contributioncheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestA64HarnessabilityScorecardScriptParity(t *testing.T) {
	root := repoRoot(t)
	shellPath := filepath.Join(root, "scripts", "check-a64-harnessability-scorecard.sh")
	psPath := filepath.Join(root, "scripts", "check-a64-harnessability-scorecard.ps1")
	baselinePath := filepath.Join(root, "scripts", "a64-harnessability-scorecard-baseline.env")

	shellRaw, err := os.ReadFile(shellPath)
	if err != nil {
		t.Fatalf("read shell script: %v", err)
	}
	psRaw, err := os.ReadFile(psPath)
	if err != nil {
		t.Fatalf("read powershell script: %v", err)
	}
	if _, err := os.Stat(baselinePath); err != nil {
		t.Fatalf("expected baseline file %s: %v", baselinePath, err)
	}

	shell := string(shellRaw)
	ps := string(psRaw)
	requiredTokens := []string{
		"contract_coverage_pct",
		"gate_coverage_pct",
		"docs_consistency",
		"drift",
		"complexity_tier",
		"downgrade_recommendation",
		"computational_first_compliant",
		"inferential_evidence",
		"uncertainty_pct",
		"BAYMAX_A64_INFERENTIAL_BLOCKING_REQUESTED",
	}
	for _, token := range requiredTokens {
		if !strings.Contains(shell, token) {
			t.Fatalf("shell harnessability scorecard missing token %q", token)
		}
		if !strings.Contains(ps, token) {
			t.Fatalf("powershell harnessability scorecard missing token %q", token)
		}
	}
	if !strings.Contains(shell, "set -euo pipefail") {
		t.Fatalf("shell harnessability scorecard must use set -euo pipefail")
	}
	if !strings.Contains(ps, "lib/native-strict.ps1") {
		t.Fatalf("powershell harnessability scorecard must use strict native helper")
	}
}

func TestQualityGateIncludesA64HarnessabilityScorecard(t *testing.T) {
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
	if !strings.Contains(shell, "check-a64-harnessability-scorecard.sh") {
		t.Fatalf("shell quality gate must invoke a64 harnessability scorecard")
	}
	if !strings.Contains(shell, "[quality-gate][a64-harnessability-scorecard]") {
		t.Fatalf("shell quality gate must expose blocking a64 harnessability scorecard failure label")
	}
	if !strings.Contains(ps, "check-a64-harnessability-scorecard.ps1") {
		t.Fatalf("powershell quality gate must invoke a64 harnessability scorecard")
	}
	if !strings.Contains(ps, "[quality-gate] a64 harnessability scorecard") {
		t.Fatalf("powershell quality gate must expose a64 harnessability scorecard step label")
	}
}

func TestMainlineIndexIncludesA64HarnessabilityAndLatencyBaselines(t *testing.T) {
	root := repoRoot(t)
	indexPath := filepath.Join(root, "docs", "mainline-contract-test-index.md")
	raw, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read mainline contract index: %v", err)
	}
	index := string(raw)
	required := []string{
		"check-a64-harnessability-scorecard.sh",
		"check-a64-harnessability-scorecard.ps1",
		"a64-harnessability-scorecard-baseline.env",
		"a64-gate-latency-baseline.env",
	}
	for _, token := range required {
		if !strings.Contains(index, token) {
			t.Fatalf("mainline contract index missing token %q", token)
		}
	}
}
