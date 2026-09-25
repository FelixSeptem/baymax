package contributioncheck

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/FelixSeptem/baymax/tool/diagnosticsreplay"
)

func TestActionCapabilityAuditRemainsOfflineAndLibraryFirst(t *testing.T) {
	root := repoRoot(t)
	for _, name := range []string{"scripts/check-action-capability-audit-contract.sh", "scripts/check-action-capability-audit-contract.ps1"} {
		body, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		for _, marker := range []string{"ActionCapabilityAudit", "privacy", "run", "stream", "fixture", "library-first"} {
			if !strings.Contains(strings.ToLower(string(body)), strings.ToLower(marker)) {
				t.Fatalf("%s missing marker %q", name, marker)
			}
		}
	}

	implementation := filepath.Join(root, "tool", "diagnosticsreplay", "action_capability_audit.go")
	body, err := os.ReadFile(implementation)
	if err != nil {
		t.Fatalf("read implementation: %v", err)
	}
	for _, forbidden := range []string{
		"github.com/openai/", "anthropics/anthropic-sdk", "google.golang.org/genai", "net/http", "os.Open", "os.WriteFile",
		"RuntimeRecorder", "RunRecord", "ModelRequest", "registry.Register", "dynamicDownload", "marketplaceResolve", "credentialStore", "globalActionQueue", "hostedExecution", "automaticCompensation",
	} {
		if strings.Contains(string(body), forbidden) {
			t.Fatalf("implementation contains forbidden boundary reference %q", forbidden)
		}
	}
}

func TestActionCapabilityAuditCanonicalFixtureReplaysDeterministically(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "tool", "diagnosticsreplay", "testdata", "action_capability_audit.v1.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read canonical fixture: %v", err)
	}
	original := bytes.Clone(raw)
	first, err := diagnosticsreplay.ReplayActionCapabilityAuditJSON(raw)
	if err != nil {
		t.Fatalf("first replay: %v", err)
	}
	second, err := diagnosticsreplay.ReplayActionCapabilityAuditJSON(raw)
	if err != nil {
		t.Fatalf("second replay: %v", err)
	}
	if !bytes.Equal(raw, original) {
		t.Fatal("replay mutated its source fixture")
	}
	if len(first.Cases) != len(second.Cases) {
		t.Fatalf("replay case count changed: %d != %d", len(first.Cases), len(second.Cases))
	}
	for index := range first.Cases {
		if first.Cases[index].Digest != second.Cases[index].Digest || first.Cases[index].Verdict != second.Cases[index].Verdict || !first.Cases[index].Idempotent {
			t.Fatalf("replay output drift at case %d: first=%#v second=%#v", index, first.Cases[index], second.Cases[index])
		}
	}
}

func TestActionCapabilityAuditTaxonomyMatchesBothPlatformGates(t *testing.T) {
	root := repoRoot(t)
	implementation, err := os.ReadFile(filepath.Join(root, "tool", "diagnosticsreplay", "action_capability_audit.go"))
	if err != nil {
		t.Fatalf("read implementation: %v", err)
	}
	actualCodes := regexp.MustCompile(`ReasonCodeActionCapability\w+\s*=\s*"(action_capability_[a-z0-9_]+)"`).FindAllStringSubmatch(string(implementation), -1)
	actual := make([]string, 0, len(actualCodes))
	for _, item := range actualCodes {
		actual = append(actual, item[1])
	}
	sort.Strings(actual)
	for _, name := range []string{"scripts/check-action-capability-audit-contract.ps1", "scripts/check-action-capability-audit-contract.sh"} {
		body, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		gateCodes := actionCapabilityGateTaxonomy(t, string(body), name)
		if len(actual) != len(gateCodes) {
			t.Fatalf("%s taxonomy count = %d, implementation count = %d", name, len(gateCodes), len(actual))
		}
		for index := range actual {
			if actual[index] != gateCodes[index] {
				t.Fatalf("%s taxonomy differs at %d: %q != %q", name, index, gateCodes[index], actual[index])
			}
		}
	}
}

func actionCapabilityGateTaxonomy(t *testing.T, body, name string) []string {
	t.Helper()
	startMarker := "ACTION_CAPABILITY_TAXONOMY_BEGIN"
	endMarker := "ACTION_CAPABILITY_TAXONOMY_END"
	start := strings.Index(body, startMarker)
	end := strings.Index(body, endMarker)
	if start < 0 || end < 0 || end <= start {
		t.Fatalf("%s is missing its bounded taxonomy block", name)
	}
	block := body[start+len(startMarker) : end]
	matches := regexp.MustCompile(`"(action_capability_[a-z0-9_]+)"`).FindAllStringSubmatch(block, -1)
	codes := make([]string, 0, len(matches))
	for _, match := range matches {
		codes = append(codes, match[1])
	}
	sort.Strings(codes)
	return codes
}
