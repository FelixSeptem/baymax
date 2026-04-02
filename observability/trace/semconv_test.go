package trace

import "testing"

func TestCanonicalSemconvTopologyV1CoversCoreDomains(t *testing.T) {
	topology := CanonicalSemconvTopologyV1()
	required := []string{
		TraceDomainRun,
		TraceDomainModel,
		TraceDomainTool,
		TraceDomainMCP,
		TraceDomainMemory,
		TraceDomainHITL,
	}
	for _, domain := range required {
		spec, ok := topology[domain]
		if !ok {
			t.Fatalf("missing semconv topology domain %q", domain)
		}
		if spec.SpanName == "" {
			t.Fatalf("domain %q must define canonical span name", domain)
		}
		if len(spec.CanonicalAttrKeys) == 0 {
			t.Fatalf("domain %q must define canonical attributes", domain)
		}
	}
}

func TestCanonicalAttributeMapInjectsSchemaAndFiltersUnknownKeys(t *testing.T) {
	attrs := CanonicalAttributeMap(TraceDomainTool, map[string]string{
		AttrRunID:          "run-1",
		AttrStepID:         "step-2",
		AttrToolName:       "search",
		AttrBudgetDecision: "degrade",
		"tool.custom":      "ignore",
	})
	if attrs[AttrTraceSchemaVersion] != OTelSemconvVersionV1 {
		t.Fatalf("schema version = %q, want %q", attrs[AttrTraceSchemaVersion], OTelSemconvVersionV1)
	}
	if attrs[AttrDomain] != TraceDomainTool {
		t.Fatalf("domain attr = %q, want %q", attrs[AttrDomain], TraceDomainTool)
	}
	if attrs["tool.custom"] != "" {
		t.Fatalf("unknown custom key must be filtered, got %#v", attrs)
	}
}

func TestRunStreamSemanticEquivalenceAllowsOrderingDifferences(t *testing.T) {
	runSpans := []SemanticSpan{
		{
			Domain: TraceDomainRun,
			Attributes: map[string]string{
				AttrRunID:          "run-1",
				AttrMode:           "run",
				AttrBudgetDecision: "allow",
			},
		},
		{
			Domain: TraceDomainModel,
			Attributes: map[string]string{
				AttrRunID:  "run-1",
				AttrStepID: "step-model-1",
				AttrMode:   "run",
			},
		},
		{
			Domain: TraceDomainTool,
			Attributes: map[string]string{
				AttrRunID:    "run-1",
				AttrStepID:   "step-tool-1",
				AttrToolName: "search",
			},
		},
	}
	streamSpans := []SemanticSpan{
		{
			Domain: TraceDomainTool,
			Attributes: map[string]string{
				AttrRunID:    "run-1",
				AttrStepID:   "step-tool-1",
				AttrToolName: "search",
			},
		},
		{
			Domain: TraceDomainRun,
			Attributes: map[string]string{
				AttrRunID:          "run-1",
				AttrMode:           "run",
				AttrBudgetDecision: "allow",
			},
		},
		{
			Domain: TraceDomainModel,
			Attributes: map[string]string{
				AttrRunID:  "run-1",
				AttrStepID: "step-model-1",
				AttrMode:   "run",
			},
		},
		{
			Domain: TraceDomainTool,
			Attributes: map[string]string{
				AttrRunID:    "run-1",
				AttrStepID:   "step-tool-1",
				AttrToolName: "search",
			},
		},
	}
	if !SemanticallyEquivalentSpans(runSpans, streamSpans) {
		t.Fatalf("run/stream semantic spans should be equivalent after normalization")
	}
}

func TestRunStreamSemanticEquivalenceDetectsTopologyDrift(t *testing.T) {
	runSpans := []SemanticSpan{
		{
			Domain: TraceDomainRun,
			Attributes: map[string]string{
				AttrRunID: "run-2",
			},
		},
		{
			Domain: TraceDomainTool,
			Attributes: map[string]string{
				AttrRunID:    "run-2",
				AttrStepID:   "step-tool-2",
				AttrToolName: "search",
			},
		},
	}
	streamSpans := []SemanticSpan{
		{
			Domain: TraceDomainRun,
			Attributes: map[string]string{
				AttrRunID: "run-2",
			},
		},
		{
			Domain: TraceDomainMCP,
			Attributes: map[string]string{
				AttrRunID:        "run-2",
				AttrStepID:       "step-tool-2",
				AttrMCPTransport: "http",
			},
		},
	}
	if SemanticallyEquivalentSpans(runSpans, streamSpans) {
		t.Fatalf("topology drift must not be considered equivalent")
	}
}
