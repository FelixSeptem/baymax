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
		"Composed Orchestration",
		"Teams Mixed Local+Remote",
		"Workflow A2A Step",
		"Composed Remote Failure",
		"Composed Config/Reload",
		"Composed Diagnostics Replay",
		"Scheduler Crash Takeover",
		"Scheduler Idempotent Replay",
		"Scheduler Run/Stream Equivalence",
		"A2A Scheduler Retry",
		"Scheduler Recovery Takeover",
		"Scheduler Recovery Idempotency",
		"Scheduler Recovery Run/Stream",
		"Composer Run/Stream Equivalence",
		"Composer Scheduler Fallback",
		"Composer Child Replay Idempotency",
		"Composer Recovery Cross-Session",
		"Composer Recovery Replay Idempotency",
		"Composer Recovery Conflict Fail-Fast",
		"Scheduler QoS Fairness",
		"Scheduler QoS Backoff And DLQ",
		"Scheduler QoS Run/Stream Equivalence",
		"Sync Invocation Coverage",
		"Sync Invocation Run/Stream Equivalence",
		"Sync Invocation Scheduler Canceled Mapping",
		"Async Reporting Delivery",
		"Async Reporting Dedup Replay",
		"Async Reporting Run/Stream Equivalence",
		"Async Reporting Recovery Replay",
		"Delayed Dispatch Eligibility",
		"Delayed Dispatch Recovery",
		"Delayed Dispatch Run/Stream Equivalence",
		"Delayed Dispatch Async Compatibility",
		"Tail Governance Cross-Mode Matrix",
		"Tail Governance QoS+Recovery",
		"Tail Governance Parser Compatibility",
		"Tail Governance Async+Delayed Replay",
		"Workflow Graph Composability Determinism",
		"Workflow Graph Composability Compile Fail-Fast",
		"Workflow Graph Composability Run/Stream+Resume",
		"Workflow Graph Composability Composer Path",
		"Full-Chain Reference Example Smoke Gate",
		"Full-Chain Reference Example Quality Path",
		"External Adapter Template Docs Consistency",
		"External Adapter Template Contribution Traceability",
		"External Adapter Conformance Matrix",
		"External Adapter Conformance Gate Path",
		"Adapter Scaffold Determinism + Conflict",
		"Adapter Scaffold Bootstrap Mapping + Offline Executable",
		"Adapter Scaffold Drift Gate Path",
		"Adapter Manifest Contract Core Validation",
		"Adapter Manifest Runtime Activation",
		"Adapter Manifest Scaffold/Conformance Alignment",
		"Adapter Manifest Contract Gate Path",
		"Adapter Capability Negotiation Core Semantics",
		"Adapter Capability Negotiation Run/Stream Equivalence",
		"Adapter Capability Negotiation Scaffold + Conformance Alignment",
		"Adapter Capability Negotiation Gate Path",
		"Adapter Contract Profile Versioning Core Validation",
		"Adapter Contract Profile Versioning Manifest Integration",
		"Adapter Contract Replay Fixtures",
		"Adapter Contract Replay Gate Path",
		"Task Board Query Filters + Defaults",
		"Task Board Query Backend Parity + Restore",
		"Task Board Query Gate Path",
		"Mailbox Contract Sync/Async/Delayed Convergence",
		"Mailbox Contract Query + Correlation Mapping",
		"Mailbox Contract Backend Parity + Restore Determinism",
		"Mailbox Contract Gate Path",
		"Async-Await Lifecycle Scheduler Transition + Timeout",
		"Async-Await Lifecycle Run/Stream + Backend Parity",
		"Async-Await Lifecycle Gate Path",
		"Async-Await Reconcile Callback-Loss Fallback",
		"Async-Await Reconcile Run/Stream + Backend Parity",
		"Async-Await Reconcile Replay Idempotency",
		"Async-Await Reconcile Gate Path",
		"Collaboration Retry Default Disabled + Transport-Only",
		"Collaboration Retry Scheduler Single-Owner",
		"Collaboration Retry Run/Stream + Replay Idempotency",
		"Canonical Invoke Entrypoint Sync Public Surface Guard",
		"Canonical Invoke Entrypoint Async Public Surface Guard",
		"Canonical Invoke Entrypoint Mailbox Convergence Guard",
		"Canonical Invoke Entrypoint Gate Path",
		"Pre-1 Governance Docs Consistency",
		"Pre-1 Governance Gate Path",
		"Pre-1 Governance Quality Path",
		"Status Parity Governance",
		"Core Module README Richness",
		"Status Parity + README Richness Gate Path",
		"PowerShell Gate Fail-Fast Governance Strict Helper Guard",
		"PowerShell Gate Fail-Fast Governance Gate Path",
		"Status Parity Convergence Gate Path",
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
