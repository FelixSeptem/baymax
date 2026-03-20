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
		"Workflow Graph Composability Determinism A15",
		"Workflow Graph Composability Compile Fail-Fast A15",
		"Workflow Graph Composability Run/Stream+Resume A15",
		"Workflow Graph Composability Composer Path A15",
		"Full-Chain Reference Example A20 Smoke Gate",
		"Full-Chain Reference Example A20 Quality Path",
		"External Adapter Template A21 Docs Consistency",
		"External Adapter Template A21 Contribution Traceability",
		"External Adapter Conformance A22 Matrix",
		"External Adapter Conformance A22 Gate Path",
		"Adapter Scaffold A23 Determinism + Conflict",
		"Adapter Scaffold A23 Bootstrap Mapping + Offline Executable",
		"Adapter Scaffold A23 Drift Gate Path",
		"Adapter Manifest Contract A26 Core Validation",
		"Adapter Manifest Runtime Activation A26",
		"Adapter Manifest Scaffold/Conformance Alignment A26",
		"Adapter Manifest Contract A26 Gate Path",
		"Adapter Capability Negotiation A27 Core Semantics",
		"Adapter Capability Negotiation A27 Run/Stream Equivalence",
		"Adapter Capability Negotiation A27 Scaffold + Conformance Alignment",
		"Adapter Capability Negotiation A27 Gate Path",
		"Adapter Contract Profile Versioning A28 Core Validation",
		"Adapter Contract Profile Versioning A28 Manifest Integration",
		"Adapter Contract Replay A28 Fixtures",
		"Adapter Contract Replay A28 Gate Path",
		"Task Board Query A29 Filters + Defaults",
		"Task Board Query A29 Backend Parity + Restore",
		"Task Board Query A29 Gate Path",
		"Mailbox Contract A30 Sync/Async/Delayed Convergence",
		"Mailbox Contract A30 Query + Correlation Mapping",
		"Mailbox Contract A30 Backend Parity + Restore Determinism",
		"Mailbox Contract A30 Gate Path",
		"Async-Await Lifecycle A31 Scheduler Transition + Timeout",
		"Async-Await Lifecycle A31 Run/Stream + Backend Parity",
		"Async-Await Lifecycle A31 Gate Path",
		"Pre-1 Governance A24 Docs Consistency",
		"Pre-1 Governance A24 Gate Path",
		"Pre-1 Governance A24 Quality Path",
		"Status Parity Governance A25",
		"Core Module README Richness A25",
		"Status Parity + README Richness A25 Gate Path",
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
