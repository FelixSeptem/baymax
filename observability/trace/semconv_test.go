package trace

import "testing"

func TestProtocolAttributesAreAdditiveAndNullable(t *testing.T) {
	attrs := ProtocolAttributes("run-1", "step-1", "runner", "run-0", "artifact-1", "checkpoint-1")
	if attrs[AttrRunID] != "run-1" || attrs[AttrStepID] != "step-1" || attrs[AttrProtocolSource] != "runner" || attrs[AttrCausationID] != "run-0" || attrs[AttrArtifactID] != "artifact-1" || attrs[AttrCheckpointID] != "checkpoint-1" {
		t.Fatalf("attrs = %#v", attrs)
	}
	if got := ProtocolAttributes("run-1", "", "", "", "", ""); len(got) != 1 || got[AttrRunID] != "run-1" {
		t.Fatalf("nullable attrs = %#v", got)
	}
}

func TestProtocolDecisionAttributesPreserveAdditiveCorrelation(t *testing.T) {
	attrs := ProtocolDecisionAttributes("v1", "accepted", "", "serialize", "queued", "scheduler.session_busy")
	if attrs[AttrProtocolProfileVersion] != "v1" || attrs[AttrProtocolCapabilityDecision] != "accepted" || attrs[AttrProtocolAdmissionPolicy] != "serialize" || attrs[AttrProtocolAdmissionDecision] != "queued" || attrs[AttrProtocolAdmissionReason] != "scheduler.session_busy" {
		t.Fatalf("unexpected decision attributes: %#v", attrs)
	}
	if _, ok := attrs[AttrProtocolCapabilityReason]; ok {
		t.Fatal("empty capability reason must be omitted")
	}
}

func TestEventStreamBindingAttributesAreBoundedAndNullable(t *testing.T) {
	attrs := EventStreamBindingAttributes("sub-1", "live", "accepted", "realtime.binding.live", "after_cursor", 42)
	if attrs[AttrStreamSubscriptionID] != "sub-1" || attrs[AttrStreamBindingPhase] != "live" || attrs[AttrStreamBindingDecision] != "accepted" || attrs[AttrStreamBindingReason] != "realtime.binding.live" || attrs[AttrStreamBindingCursorMode] != "after_cursor" || attrs[AttrStreamBindingSequenceBoundary] != "42" {
		t.Fatalf("unexpected binding attributes: %#v", attrs)
	}
	for _, key := range []string{"stream.cursor", "stream.payload", "stream.subscriber_metadata"} {
		if _, ok := attrs[key]; ok {
			t.Fatalf("unbounded attribute %q must not be emitted", key)
		}
	}
	if got := EventStreamBindingAttributes("", "", "", "", "", 0); len(got) != 0 {
		t.Fatalf("empty binding correlation must be omitted: %#v", got)
	}
}

func TestCheckpointProvenanceAttributesAreBoundedAndNullable(t *testing.T) {
	attrs := CheckpointProvenanceAttributes("derived", "resume", "associated", "")
	if attrs[AttrProtocolCheckpointRelation] != "derived" || attrs[AttrProtocolCheckpointRestoreSource] != "resume" || attrs[AttrProtocolWorkspaceProvenance] != "associated" {
		t.Fatalf("unexpected provenance attrs: %#v", attrs)
	}
	if _, ok := attrs[AttrProtocolWorkspaceDriftReason]; ok {
		t.Fatal("empty drift reason must be omitted")
	}
	for _, key := range []string{"agent.protocol.workspace_id", "agent.protocol.integrity", "workspace.contents"} {
		if _, ok := attrs[key]; ok {
			t.Fatalf("high-cardinality key %q emitted", key)
		}
	}
}

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

func TestTerminalOutcomeAttributesAreBoundedAndAdditive(t *testing.T) {
	attrs := TerminalOutcomeAttributes("completed", "none", "post_start", "react.completed", false, false, 2, 4, "cause-1")
	if attrs[AttrTerminalState] != "completed" || attrs[AttrTerminalFailureFamily] != "none" || attrs[AttrTerminalPhase] != "post_start" || attrs[AttrTerminalAttempt] != "2" {
		t.Fatalf("terminal attrs = %#v", attrs)
	}
	if _, ok := attrs[AttrTerminalCausationID]; ok {
		t.Fatal("causation id must be excluded from bounded OTel terminal attributes")
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
