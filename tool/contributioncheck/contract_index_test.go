package contributioncheck

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
)

var contractRefPattern = regexp.MustCompile("`([^`]+::Test[A-Za-z0-9_]+)`")

func TestMainlineContractIndexReferencesExistingTests(t *testing.T) {
	root := repoRoot(t)
	indexPath := filepath.Join(root, "docs", "mainline-contract-test-index.md")
	raw, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read contract index: %v", err)
	}
	doc := string(raw)

	requiredRows := []string{
		"Skill Trigger Scoring D1",
		"Skill Trigger Scoring D2",
		"Skill Trigger Scoring D3",
		"Security Policy S2",
		"Security Event S3",
		"Security Delivery S4",
		"Composed Orchestration A5",
		"Teams Mixed Local+Remote A5",
		"Workflow A2A Step A5",
		"Composed Remote Failure A5",
		"Composed Config/Reload A5",
		"Composed Diagnostics Replay A5",
		"Scheduler Crash Takeover A6",
		"Scheduler Idempotent Replay A6",
		"Scheduler Run/Stream Equivalence A6",
		"A2A Scheduler Retry A6",
		"Scheduler Recovery Takeover A7",
		"Scheduler Recovery Idempotency A7",
		"Scheduler Recovery Run/Stream A7",
		"Composer Run/Stream Equivalence A8",
		"Composer Scheduler Fallback A8",
		"Composer Child Replay Idempotency A8",
		"Composer Recovery Cross-Session A9",
		"Composer Recovery Replay Idempotency A9",
		"Composer Recovery Conflict Fail-Fast A9",
		"Scheduler QoS Fairness A10",
		"Scheduler QoS Backoff And DLQ A10",
		"Scheduler QoS Run/Stream Equivalence A10",
		"Sync Invocation Coverage A11",
		"Sync Invocation Run/Stream Equivalence A11",
		"Sync Invocation Scheduler Canceled Mapping A11",
		"Async Reporting Delivery A12",
		"Async Reporting Dedup Replay A12",
		"Async Reporting Run/Stream Equivalence A12",
		"Async Reporting Recovery Replay A12",
		"Delayed Dispatch Eligibility A13",
		"Delayed Dispatch Recovery A13",
		"Delayed Dispatch Run/Stream Equivalence A13",
		"Delayed Dispatch Async Compatibility A13",
		"Tail Governance Cross-Mode Matrix A14",
		"Tail Governance QoS+Recovery A14",
		"Tail Governance Parser Compatibility A14",
		"Tail Governance Async+Delayed Replay A14",
	}
	for _, row := range requiredRows {
		if !strings.Contains(doc, row) {
			t.Fatalf("mainline contract index missing required row: %q", row)
		}
	}

	matches := contractRefPattern.FindAllStringSubmatch(doc, -1)
	if len(matches) == 0 {
		t.Fatal("no contract test references found in mainline contract index")
	}

	seen := map[string]struct{}{}
	missing := make([]string, 0)
	for _, m := range matches {
		ref := strings.TrimSpace(m[1])
		if ref == "" {
			continue
		}
		if _, ok := seen[ref]; ok {
			continue
		}
		seen[ref] = struct{}{}

		parts := strings.Split(ref, "::")
		if len(parts) != 2 {
			missing = append(missing, ref+" (invalid reference format)")
			continue
		}
		filePart := filepath.FromSlash(strings.TrimSpace(parts[0]))
		testName := strings.TrimSpace(parts[1])
		if testName == "" {
			missing = append(missing, ref+" (missing test name)")
			continue
		}

		testFile := filepath.Join(root, filePart)
		content, readErr := os.ReadFile(testFile)
		if readErr != nil {
			missing = append(missing, ref+" (file not found)")
			continue
		}
		pattern := regexp.MustCompile(`(?m)^func\s+` + regexp.QuoteMeta(testName) + `\s*\(`)
		if !pattern.Match(content) {
			missing = append(missing, ref+" (test function not found)")
		}
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		t.Fatalf("mainline contract index has invalid references:\n%s", strings.Join(missing, "\n"))
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve runtime caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
}
