package schemaaudit

import "testing"

func readinessFacts(t *testing.T) (SchemaFacts, SchemaFacts) {
	t.Helper()
	alpha, err := CanonicalSchemaFacts(map[string]any{"type": "object", "properties": map[string]any{"q": map[string]any{"type": "string"}}})
	if err != nil {
		t.Fatal(err)
	}
	noise, err := CanonicalSchemaFacts(map[string]any{"type": "object", "properties": map[string]any{"n": map[string]any{"type": "string"}}})
	if err != nil {
		t.Fatal(err)
	}
	return alpha, noise
}

func validReadinessInput(t *testing.T) ReadinessInput {
	t.Helper()
	alpha, noise := readinessFacts(t)
	return ReadinessInput{
		Version:    ReadinessVersion,
		SnapshotID: "readiness-test",
		Tools: []AdmittedTool{
			{Identity: "alpha", Source: "local", Capabilities: []string{"search"}, Admitted: true, Schema: alpha},
			{Identity: "noise", Source: "local", Capabilities: []string{"admin"}, Admitted: true, Schema: noise},
		},
		Policy: ReadinessPolicy{
			MinStableSamples:  2,
			SchemaBytesBudget: alpha.Bytes + 1,
			TokenBudget:       alpha.TokenEstimate + 1,
			QualityMinF1:      0.8,
		},
		Cases: []TaskCase{{ID: "find", Expected: []string{"alpha"}, Allowed: []string{"alpha"}, Forbidden: []string{"noise"}}},
		Samples: []ReadinessSample{
			{Ordinal: 1, SchemaBytes: alpha.Bytes + noise.Bytes, TokenEstimate: alpha.TokenEstimate + noise.TokenEstimate, F1: 0.5, ForbiddenHit: 1},
			{Ordinal: 2, SchemaBytes: alpha.Bytes + noise.Bytes, TokenEstimate: alpha.TokenEstimate + noise.TokenEstimate, F1: 0.5, ForbiddenHit: 1},
		},
		Opportunity: ProjectionOpportunity{Selected: []string{"alpha"}, Required: []string{"alpha"}},
	}
}

func TestEvaluateReadinessStableEvidenceProducesDesignReady(t *testing.T) {
	result, err := EvaluateReadiness(validReadinessInput(t))
	if err != nil {
		t.Fatal(err)
	}
	if result.Conclusion != ReadinessReadyForRuntimeDesign {
		t.Fatalf("conclusion = %q, want %q; result=%#v", result.Conclusion, ReadinessReadyForRuntimeDesign, result)
	}
	if result.PressureSignal != ReadinessStable || result.AdmissionIntegrity != ReadinessPass || result.SemanticCompleteness != ReadinessPass {
		t.Fatalf("readiness axes = %#v", result)
	}
	if result.Digest == "" || result.WindowDigest == "" {
		t.Fatal("readiness digests must be populated")
	}
}

func TestEvaluateReadinessSinglePeakIsNotReady(t *testing.T) {
	in := validReadinessInput(t)
	in.Samples[1].SchemaBytes = in.Policy.SchemaBytesBudget
	in.Samples[1].TokenEstimate = in.Policy.TokenBudget
	result, err := EvaluateReadiness(in)
	if err != nil {
		t.Fatal(err)
	}
	if result.PressureSignal != ReadinessTransient || result.Conclusion != ReadinessNotReady {
		t.Fatalf("single peak result = %#v", result)
	}
}

func TestEvaluateReadinessRejectsNonAdmittedAndOversizedInput(t *testing.T) {
	in := validReadinessInput(t)
	in.Tools[0].Admitted = false
	if _, err := EvaluateReadiness(in); CodeOf(err) != CodeReadinessMissingAdmission {
		t.Fatalf("missing admission code = %q", CodeOf(err))
	}
	in = validReadinessInput(t)
	in.Policy.MaxSamples = 1
	if _, err := EvaluateReadiness(in); CodeOf(err) != CodeReadinessOverflow {
		t.Fatalf("sample overflow code = %q", CodeOf(err))
	}
	if _, err := EvaluateReadiness(ReadinessInput{Version: ReadinessVersion}); CodeOf(err) != CodeReadinessOverflow {
		t.Fatalf("empty input code = %q", CodeOf(err))
	}
}

func TestEvaluateReadinessRejectsDuplicateAndUnsupportedTools(t *testing.T) {
	in := validReadinessInput(t)
	in.Tools = append(in.Tools, in.Tools[0])
	if _, err := EvaluateReadiness(in); CodeOf(err) != CodeReadinessOverflow {
		t.Fatalf("duplicate tool code = %q", CodeOf(err))
	}
	in = validReadinessInput(t)
	in.Tools[0].Source = "remote"
	if _, err := EvaluateReadiness(in); CodeOf(err) != CodeReadinessUnsupportedSource {
		t.Fatalf("unsupported source code = %q", CodeOf(err))
	}
}

func TestEvaluateReadinessRejectsOpportunityOutsideAdmittedSet(t *testing.T) {
	in := validReadinessInput(t)
	in.Opportunity.Selected = []string{"remote"}
	result, err := EvaluateReadiness(in)
	if err != nil {
		t.Fatal(err)
	}
	if result.Conclusion != ReadinessBlocked || result.AdmissionIntegrity != ReadinessFail {
		t.Fatalf("blocked opportunity result = %#v", result)
	}
}

func TestEvaluateReadinessEquivalentSampleOrderingHasStableDigest(t *testing.T) {
	first := validReadinessInput(t)
	second := validReadinessInput(t)
	second.Samples[0], second.Samples[1] = second.Samples[1], second.Samples[0]
	a, err := EvaluateReadiness(first)
	if err != nil {
		t.Fatal(err)
	}
	b, err := EvaluateReadiness(second)
	if err != nil {
		t.Fatal(err)
	}
	if a.Digest != b.Digest || a.WindowDigest != b.WindowDigest {
		t.Fatalf("sample ordering changed digest: %q/%q and %q/%q", a.Digest, b.Digest, a.WindowDigest, b.WindowDigest)
	}
}
