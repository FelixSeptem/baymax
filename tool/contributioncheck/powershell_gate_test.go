package contributioncheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPowerShellGateScriptsUseStrictNativePath(t *testing.T) {
	root := repoRoot(t)
	requiredScripts := []string{
		"scripts/check-docs-consistency.ps1",
		"scripts/check-quality-gate.ps1",
		"scripts/check-multi-agent-shared-contract.ps1",
		"scripts/check-adapter-conformance.ps1",
		"scripts/check-adapter-manifest-contract.ps1",
		"scripts/check-adapter-capability-contract.ps1",
		"scripts/check-adapter-contract-replay.ps1",
		"scripts/check-sandbox-adapter-conformance-contract.ps1",
		"scripts/check-adapter-scaffold-drift.ps1",
		"scripts/check-security-delivery-contract.ps1",
		"scripts/check-security-event-contract.ps1",
		"scripts/check-security-policy-contract.ps1",
		"scripts/check-sandbox-rollout-governance-contract.ps1",
		"scripts/check-state-snapshot-contract.ps1",
		"scripts/check-runtime-budget-admission-contract.ps1",
		"scripts/check-agent-eval-and-tracing-interop-contract.ps1",
	}

	for _, rel := range requiredScripts {
		path := filepath.Join(root, filepath.FromSlash(rel))
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		content := string(raw)
		if !strings.Contains(content, "lib/native-strict.ps1") {
			t.Fatalf("%s must import strict native helper", rel)
		}
		if !strings.Contains(content, "Invoke-NativeStrict") {
			t.Fatalf("%s must use Invoke-NativeStrict for required native commands", rel)
		}
		if strings.Contains(content, "$LASTEXITCODE -ne 0") {
			t.Fatalf("%s must not bypass strict helper via manual LASTEXITCODE checks", rel)
		}
	}
}

func TestPowerShellQualityGateGovernanceWarnExceptionOnly(t *testing.T) {
	root := repoRoot(t)

	qualityPath := filepath.Join(root, "scripts", "check-quality-gate.ps1")
	qualityRaw, err := os.ReadFile(qualityPath)
	if err != nil {
		t.Fatalf("read %s: %v", qualityPath, err)
	}
	quality := string(qualityRaw)

	if !strings.Contains(quality, "only governance exception path") {
		t.Fatalf("%s must document governance exception semantics", filepath.ToSlash(qualityPath))
	}
	if !strings.Contains(quality, "govulncheck") || !strings.Contains(quality, "warn") {
		t.Fatalf("%s must keep govulncheck warn governance exception path", filepath.ToSlash(qualityPath))
	}

	if count := strings.Count(quality, "AllowFailure"); count != 2 {
		t.Fatalf("%s must contain exactly two AllowFailure usages (govulncheck binary/go-run paths), got %d", filepath.ToSlash(qualityPath), count)
	}

	otherScripts := []string{
		"scripts/check-docs-consistency.ps1",
		"scripts/check-multi-agent-shared-contract.ps1",
		"scripts/check-adapter-conformance.ps1",
		"scripts/check-adapter-manifest-contract.ps1",
		"scripts/check-adapter-capability-contract.ps1",
		"scripts/check-adapter-contract-replay.ps1",
		"scripts/check-sandbox-adapter-conformance-contract.ps1",
		"scripts/check-adapter-scaffold-drift.ps1",
		"scripts/check-security-delivery-contract.ps1",
		"scripts/check-security-event-contract.ps1",
		"scripts/check-security-policy-contract.ps1",
		"scripts/check-sandbox-rollout-governance-contract.ps1",
		"scripts/check-state-snapshot-contract.ps1",
		"scripts/check-runtime-budget-admission-contract.ps1",
		"scripts/check-agent-eval-and-tracing-interop-contract.ps1",
	}
	for _, rel := range otherScripts {
		raw, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if readErr != nil {
			t.Fatalf("read %s: %v", rel, readErr)
		}
		if strings.Contains(string(raw), "AllowFailure") {
			t.Fatalf("%s must not introduce non-blocking native command exceptions", rel)
		}
	}
}
