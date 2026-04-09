package runtimeexample

import (
	"sort"
	"strings"
	"testing"
)

var expectedPatterns = []string{
	"rag-hybrid-retrieval",
	"structured-output-schema-contract",
	"skill-driven-discovery-hybrid",
	"mcp-governed-stdio-http",
	"hitl-governed-checkpoint",
	"context-governed-reference-first",
	"sandbox-governed-toolchain",
	"realtime-interrupt-resume",
	"multi-agents-collab-recovery",
	"workflow-branch-retry-failfast",
	"mapreduce-large-batch",
	"state-session-snapshot-recovery",
	"policy-budget-admission",
	"tracing-eval-smoke",
	"react-plan-notebook-loop",
	"hooks-middleware-extension-pipeline",
	"observability-export-bundle",
	"adapter-onboarding-manifest-capability",
	"security-policy-event-delivery",
	"config-hot-reload-rollback",
	"workflow-routing-strategy-switch",
	"multi-agents-hierarchical-planner-validator",
	"mainline-mailbox-async-delayed-reconcile",
	"mainline-task-board-query-control",
	"mainline-scheduler-qos-backoff-dlq",
	"mainline-readiness-admission-degradation",
	"custom-adapter-mcp-model-tool-memory-pack",
	"custom-adapter-health-readiness-circuit",
}

func TestRequiredPatternsCoverage(t *testing.T) {
	patterns := RequiredPatterns()
	if len(patterns) != len(expectedPatterns) {
		t.Fatalf("expected %d patterns, got %d", len(expectedPatterns), len(patterns))
	}
	for _, expected := range expectedPatterns {
		if _, ok := Lookup(expected); !ok {
			t.Fatalf("pattern not found: %s", expected)
		}
	}
	sorted := append([]string(nil), patterns...)
	sort.Strings(sorted)
	for i := range patterns {
		if patterns[i] != sorted[i] {
			t.Fatalf("required patterns must be sorted")
		}
	}
}

func TestModeSpecSemanticContracts(t *testing.T) {
	anchors := map[string]string{}
	for _, pattern := range RequiredPatterns() {
		spec, ok := Lookup(pattern)
		if !ok {
			t.Fatalf("lookup failed for pattern=%s", pattern)
		}
		if spec.Pattern != pattern {
			t.Fatalf("spec pattern mismatch: got=%s want=%s", spec.Pattern, pattern)
		}
		switch spec.Phase {
		case "P0", "P1", "P2":
		default:
			t.Fatalf("pattern %s uses unsupported phase %q", pattern, spec.Phase)
		}
		if strings.TrimSpace(spec.SemanticAnchor) == "" {
			t.Fatalf("pattern %s missing semantic anchor", pattern)
		}
		if existing, ok := anchors[spec.SemanticAnchor]; ok {
			t.Fatalf("semantic anchor %q duplicated by %s and %s", spec.SemanticAnchor, existing, pattern)
		}
		anchors[spec.SemanticAnchor] = pattern
		if strings.TrimSpace(spec.Classification) == "" {
			t.Fatalf("pattern %s missing classification", pattern)
		}
		if len(spec.RuntimeDomains) == 0 {
			t.Fatalf("pattern %s missing runtime domains", pattern)
		}
		if len(spec.Contracts) == 0 {
			t.Fatalf("pattern %s missing contracts mapping", pattern)
		}
		if len(spec.Gates) == 0 {
			t.Fatalf("pattern %s missing gates mapping", pattern)
		}
		if len(spec.Replay) == 0 {
			t.Fatalf("pattern %s missing replay mapping", pattern)
		}
		if len(spec.MinimalMarkers) < 3 {
			t.Fatalf("pattern %s should have >=3 minimal markers", pattern)
		}
		if len(spec.ProductionMarkers) < 2 {
			t.Fatalf("pattern %s should have >=2 production governance markers", pattern)
		}
	}
}

func TestExpectedMarkerVariantDistinction(t *testing.T) {
	for _, pattern := range RequiredPatterns() {
		spec, _ := Lookup(pattern)
		minimal := spec.ExpectedMarkers("minimal")
		production := spec.ExpectedMarkers("production-ish")
		if len(minimal) != len(spec.MinimalMarkers) {
			t.Fatalf("pattern %s minimal marker count mismatch", pattern)
		}
		if len(production) != len(spec.MinimalMarkers)+len(spec.ProductionMarkers) {
			t.Fatalf("pattern %s production marker count mismatch", pattern)
		}
		if strings.Join(minimal, ",") == strings.Join(production, ",") {
			t.Fatalf("pattern %s production markers must diverge from minimal markers", pattern)
		}
		for _, marker := range spec.ProductionMarkers {
			if !contains(production, marker) {
				t.Fatalf("pattern %s production marker missing: %s", pattern, marker)
			}
		}
	}
}

func TestAllSpecsReturnsIndependentCopies(t *testing.T) {
	specs := AllSpecs()
	if len(specs) != len(expectedPatterns) {
		t.Fatalf("expected %d specs, got %d", len(expectedPatterns), len(specs))
	}
	first := specs[0]
	first.Pattern = "mutated"
	first.RuntimeDomains[0] = "mutated/domain"
	again, ok := Lookup(specs[0].Pattern)
	if !ok {
		t.Fatalf("lookup failed after local mutation")
	}
	if again.Pattern == "mutated" {
		t.Fatalf("lookup result must be isolated copy")
	}
	if again.RuntimeDomains[0] == "mutated/domain" {
		t.Fatalf("runtime domains must be cloned")
	}
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
