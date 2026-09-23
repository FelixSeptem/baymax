package schemaaudit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAuditProducesDeterministicPressureQualityAndProjectionCandidate(t *testing.T) {
	alpha, err := CanonicalSchemaFacts(map[string]any{"type": "object", "properties": map[string]any{"q": map[string]any{"type": "string"}}})
	if err != nil {
		t.Fatalf("canonical alpha: %v", err)
	}
	noise, err := CanonicalSchemaFacts(map[string]any{"type": "object", "properties": map[string]any{"n": map[string]any{"type": "string"}}})
	if err != nil {
		t.Fatalf("canonical noise: %v", err)
	}
	input := Input{
		Version:    AuditVersion,
		SnapshotID: "snapshot-1",
		Tools: []AdmittedTool{
			{Identity: "alpha", Source: "local", Capabilities: []string{"search"}, Priority: 100, Admitted: true, Schema: alpha},
			{Identity: "noise", Source: "local", Capabilities: []string{"admin"}, Priority: 1, Admitted: true, Schema: noise},
		},
		Policy:     Policy{MaxTools: 8, MaxStrategies: 5, SchemaBytesBudget: alpha.Bytes + 1, TokenBudget: alpha.TokenEstimate + 1, QualityMinF1: 0.75},
		Cases:      []TaskCase{{ID: "find", Intent: "find data", Expected: []string{"alpha"}, Allowed: []string{"alpha"}, Forbidden: []string{"noise"}}},
		Strategies: []StrategySpec{{Kind: StrategyFullAdmitted}, {Kind: StrategyCapabilityFiltered, Capability: "search"}},
	}

	first, err := Audit(input)
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	second, err := Audit(input)
	if err != nil {
		t.Fatalf("replay audit: %v", err)
	}
	if first.Digest == "" || first.Digest != second.Digest {
		t.Fatalf("digest = %q/%q, want stable non-empty digest", first.Digest, second.Digest)
	}
	if first.Pressure.Level != PressureExceedsBudget {
		t.Fatalf("pressure level = %q, want %q", first.Pressure.Level, PressureExceedsBudget)
	}
	if first.Conclusion != ConclusionProjectionCandidate {
		t.Fatalf("conclusion = %q, want %q", first.Conclusion, ConclusionProjectionCandidate)
	}
	if len(first.Strategies) != 2 || first.Strategies[1].Quality.F1 <= first.Strategies[0].Quality.F1 {
		t.Fatalf("strategy quality = %#v, want filtered strategy improvement", first.Strategies)
	}
}

func TestCanonicalSchemaFactsNormalizesObjectOrderAndRejectsPrivacyMaterial(t *testing.T) {
	left, err := CanonicalSchemaFacts(map[string]any{"type": "object", "properties": map[string]any{"a": 1, "b": 2}})
	if err != nil {
		t.Fatal(err)
	}
	right, err := CanonicalSchemaFacts(map[string]any{"properties": map[string]any{"b": 2, "a": 1}, "type": "object"})
	if err != nil {
		t.Fatal(err)
	}
	if left != right {
		t.Fatalf("equivalent schemas differ: %#v != %#v", left, right)
	}
	if _, err := CanonicalSchemaFacts(map[string]any{"endpoint": "https://example.invalid"}); err == nil {
		t.Fatal("privacy material unexpectedly accepted")
	}
}

func TestAuditRejectsMissingAdmissionAndGoldConflictWithoutPartialResult(t *testing.T) {
	input := Input{Version: AuditVersion, SnapshotID: "bad", Tools: []AdmittedTool{{Identity: "x", Admitted: false}}, Cases: []TaskCase{{ID: "c", Expected: []string{"x"}, Forbidden: []string{"x"}}}}
	result, err := Audit(input)
	if err == nil {
		t.Fatal("invalid input unexpectedly accepted")
	}
	if result.Version != "" || result.Digest != "" || len(result.Strategies) != 0 {
		t.Fatalf("invalid audit returned partial result: %#v", result)
	}
	if CodeOf(err) != CodeMissingAdmission {
		t.Fatalf("error code = %q, want %q", CodeOf(err), CodeMissingAdmission)
	}
}

func TestStrategiesAndReplayAreBoundedAndDeterministic(t *testing.T) {
	alpha, _ := CanonicalSchemaFacts(map[string]any{"type": "object"})
	beta, _ := CanonicalSchemaFacts(map[string]any{"type": "object", "properties": map[string]any{"x": "long"}})
	in := Input{Version: AuditVersion, SnapshotID: "s", Tools: []AdmittedTool{
		{Identity: "z", Source: "mcp", Capabilities: []string{"search"}, Priority: 2, Admitted: true, Schema: beta},
		{Identity: "a", Source: "local", Capabilities: []string{"search"}, Priority: 10, Admitted: true, Schema: alpha},
	}, Cases: []TaskCase{{ID: "c", Expected: []string{"a"}, Allowed: []string{"a", "z"}, Forbidden: []string{"z"}}}, Strategies: []StrategySpec{
		{Kind: StrategyPriorityTopK, K: 1}, {Kind: StrategySourcePartitioned, Source: "local"}, {Kind: StrategyFixtureDeclared, Declared: []string{"a"}},
	}}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ReplayJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Strategies[0].Selected[0] != "a" || got.Strategies[1].Selected[0] != "a" || got.Strategies[2].Selected[0] != "a" {
		t.Fatalf("unexpected deterministic selections: %#v", got.Strategies)
	}
	if err := CompareParity(got, got); err != nil {
		t.Fatalf("same result parity failed: %v", err)
	}
}

func TestSyntheticFixtureAndHistoricalDefaultsReplay(t *testing.T) {
	path := filepath.Join("testdata", "synthetic_pressure_quality.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	result, err := ReplayJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if result.Conclusion != ConclusionProjectionCandidate {
		t.Fatalf("fixture conclusion = %q", result.Conclusion)
	}
	legacy := append([]byte(`{"unknown_future_field":true,`), raw[1:]...)
	if _, err := ReplayJSON(legacy); err != nil {
		t.Fatalf("unknown fields should be tolerated: %v", err)
	}
}

func TestConclusionMatrixAndCorpusAdvisoryIsolation(t *testing.T) {
	search, err := CanonicalSchemaFacts(map[string]any{"type": "object"})
	if err != nil {
		t.Fatal(err)
	}
	admin, err := CanonicalSchemaFacts(map[string]any{"type": "object", "properties": map[string]any{"admin": map[string]any{"type": "boolean"}}})
	if err != nil {
		t.Fatal(err)
	}
	base := Input{Version: AuditVersion, SnapshotID: "matrix", Tools: []AdmittedTool{
		{Identity: "search", Source: "local", Capabilities: []string{"search"}, Priority: 10, Admitted: true, Schema: search},
		{Identity: "admin", Source: "local", Capabilities: []string{"admin"}, Priority: 1, Admitted: true, Schema: admin},
	}, Cases: []TaskCase{{ID: "search", Expected: []string{"search"}, Allowed: []string{"search"}, Forbidden: []string{"admin"}}}}
	tests := []struct {
		name, want string
		mutate     func(*Input)
	}{
		{"quality only", ConclusionQualityOnly, func(in *Input) {
			in.Policy.QualityMinF1 = 0.9
			in.Policy.SchemaBytesBudget = 1 << 20
			in.Policy.TokenBudget = 1 << 20
			in.Strategies = []StrategySpec{{Kind: StrategyFullAdmitted}}
		}},
		{"pressure and quality gap", ConclusionPressureQualityGap, func(in *Input) {
			in.Policy.QualityMinF1 = 0.9
			in.Policy.SchemaBytesBudget = search.Bytes
			in.Policy.TokenBudget = search.TokenEstimate
			in.Strategies = []StrategySpec{{Kind: StrategyFullAdmitted}}
		}},
		{"projection candidate", ConclusionProjectionCandidate, func(in *Input) {
			in.Policy.QualityMinF1 = 0.9
			in.Policy.SchemaBytesBudget = search.Bytes
			in.Policy.TokenBudget = search.TokenEstimate
			in.Strategies = []StrategySpec{{Kind: StrategyFullAdmitted}, {Kind: StrategyCapabilityFiltered, Capability: "search"}}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := base
			tt.mutate(&in)
			got, err := Audit(in)
			if err != nil {
				t.Fatal(err)
			}
			if got.Conclusion != tt.want {
				t.Fatalf("conclusion = %q, want %q", got.Conclusion, tt.want)
			}
		})
	}
	within := base
	within.Tools = within.Tools[:1]
	within.Cases[0].Forbidden = nil
	within.Policy = Policy{SchemaBytesBudget: 1 << 20, TokenBudget: 1 << 20, QualityMinF1: 0.9}
	within.Strategies = []StrategySpec{{Kind: StrategyFullAdmitted}}
	withoutCorpus, err := Audit(within)
	if err != nil {
		t.Fatal(err)
	}
	within.Corpus = &CorpusAdvisory{Version: "eval_corpus_advisory.v1", LabeledCases: 0, Coverage: 0, Trend: "insufficient"}
	withCorpus, err := Audit(within)
	if err != nil {
		t.Fatal(err)
	}
	if withoutCorpus.Conclusion != ConclusionBaselineSufficient || withCorpus.Conclusion != withoutCorpus.Conclusion {
		t.Fatalf("advisory changed synthetic conclusion: %q -> %q", withoutCorpus.Conclusion, withCorpus.Conclusion)
	}
}

func TestPressureOnlyAndParityDrift(t *testing.T) {
	facts, err := CanonicalSchemaFacts(map[string]any{"type": "object"})
	if err != nil {
		t.Fatal(err)
	}
	in := Input{Version: AuditVersion, SnapshotID: "pressure", Tools: []AdmittedTool{{Identity: "search", Source: "local", Admitted: true, Schema: facts}}, Policy: Policy{SchemaBytesBudget: facts.Bytes - 1, TokenBudget: facts.TokenEstimate - 1, QualityMinF1: 0.9}, Cases: []TaskCase{{ID: "search", Expected: []string{"search"}, Allowed: []string{"search"}}}, Strategies: []StrategySpec{{Kind: StrategyFullAdmitted}}}
	got, err := Audit(in)
	if err != nil {
		t.Fatal(err)
	}
	if got.Conclusion != ConclusionPressureOnly {
		t.Fatalf("conclusion = %q, want %q", got.Conclusion, ConclusionPressureOnly)
	}
	drift := got
	drift.Conclusion = ConclusionQualityOnly
	if CodeOf(CompareParity(got, drift)) != CodeRunStreamParityDrift {
		t.Fatalf("parity drift code = %q", CodeOf(CompareParity(got, drift)))
	}
}

func TestCompareParityRejectsCaseReasonAndCorpusDrift(t *testing.T) {
	facts, err := CanonicalSchemaFacts(map[string]any{"type": "object"})
	if err != nil {
		t.Fatal(err)
	}
	in := Input{
		Version:    AuditVersion,
		SnapshotID: "parity-semantics",
		Tools:      []AdmittedTool{{Identity: "search", Source: "local", Admitted: true, Schema: facts}},
		Cases:      []TaskCase{{ID: "search", Expected: []string{"search"}, Allowed: []string{"search"}}},
		Corpus:     &CorpusAdvisory{Version: "eval_corpus_advisory.v1", LabeledCases: 1, Coverage: 1, Trend: "improving"},
	}
	result, err := Audit(in)
	if err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*Result){
		func(got *Result) { got.Cases[0].Baseline.F1 = 0 },
		func(got *Result) { got.Reasons = []string{"measurement_drift"} },
		func(got *Result) { got.Corpus.Trend = "declining" },
	} {
		drift := result
		drift.Cases = append([]CaseResult(nil), result.Cases...)
		if result.Corpus != nil {
			corpus := *result.Corpus
			drift.Corpus = &corpus
		}
		mutate(&drift)
		if code := CodeOf(CompareParity(result, drift)); code != CodeRunStreamParityDrift {
			t.Fatalf("parity drift code = %q, want %q", code, CodeRunStreamParityDrift)
		}
	}
}

func TestStrategyWithoutMatchingToolReportsBoundedSkipReason(t *testing.T) {
	facts, err := CanonicalSchemaFacts(map[string]any{"type": "object"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := Audit(Input{
		Version:    AuditVersion,
		SnapshotID: "bounded-skip-reason",
		Tools:      []AdmittedTool{{Identity: "search", Source: "local", Capabilities: []string{"search"}, Admitted: true, Schema: facts}},
		Cases:      []TaskCase{{ID: "search", Expected: []string{"search"}, Allowed: []string{"search"}}},
		Strategies: []StrategySpec{{Kind: StrategyCapabilityFiltered, Capability: "admin"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := result.Strategies[0].SkipReason, "no_matching_tools"; got != want {
		t.Fatalf("skip reason = %q, want %q", got, want)
	}
}

func TestAuditValidationMatrixReturnsStableCodeWithoutPartialResult(t *testing.T) {
	facts, err := CanonicalSchemaFacts(map[string]any{"type": "object"})
	if err != nil {
		t.Fatal(err)
	}
	valid := func() Input {
		return Input{
			Version: AuditVersion, SnapshotID: "validation-matrix",
			Tools: []AdmittedTool{{Identity: "search", Source: "local", Capabilities: []string{"search"}, Admitted: true, Schema: facts}},
			Cases: []TaskCase{{ID: "find", Expected: []string{"search"}, Allowed: []string{"search"}}},
		}
	}
	tests := []struct {
		name   string
		mutate func(*Input)
		want   string
	}{
		{"duplicate identity", func(in *Input) { in.Tools = append(in.Tools, in.Tools[0]) }, CodeDuplicateIdentity},
		{"missing admission", func(in *Input) { in.Tools[0].Admitted = false }, CodeMissingAdmission},
		{"schema overflow", func(in *Input) { in.Policy.MaxPerToolSchemaBytes = facts.Bytes - 1 }, CodeOverflow},
		{"identity overflow", func(in *Input) { in.Policy.MaxIdentityBytes = 1 }, CodeOverflow},
		{"label overflow", func(in *Input) { in.Policy.MaxLabelCount = 1; in.Tools[0].Capabilities = []string{"a", "b"} }, CodeOverflow},
		{"gold conflict", func(in *Input) { in.Cases[0].Forbidden = []string{"search"} }, CodeGoldSetConflict},
		{"unsupported strategy", func(in *Input) { in.Strategies = []StrategySpec{{Kind: "arbitrary_callback"}} }, CodeUnsupportedStrategy},
		{"forbidden source", func(in *Input) { in.Tools[0].Source = "network" }, CodeUnsupportedSource},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := valid()
			tt.mutate(&in)
			got, err := Audit(in)
			if code := CodeOf(err); code != tt.want {
				t.Fatalf("error code = %q, want %q", code, tt.want)
			}
			if got.Version != "" || got.Digest != "" || got.Conclusion != "" || len(got.Strategies) != 0 || len(got.Cases) != 0 {
				t.Fatalf("invalid audit returned partial result: %#v", got)
			}
		})
	}
	if _, err := CanonicalSchemaFacts(map[string]any{"credential": "not-allowed"}); CodeOf(err) != CodePrivacyMaterial {
		t.Fatalf("privacy error code = %q, want %q", CodeOf(err), CodePrivacyMaterial)
	}
}

func TestSyntheticFixtureMatrixAndCorpusAdvisoryIsolation(t *testing.T) {
	tests := []struct {
		file       string
		conclusion string
		code       string
	}{
		{"synthetic_within_budget.json", ConclusionBaselineSufficient, ""},
		{"synthetic_pressure_only.json", ConclusionPressureOnly, ""},
		{"synthetic_quality_only.json", ConclusionQualityOnly, ""},
		{"synthetic_pressure_quality.json", ConclusionProjectionCandidate, ""},
		{"synthetic_gold_conflict.json", "", CodeGoldSetConflict},
		{"synthetic_overflow.json", "", CodeOverflow},
		{"synthetic_strategy_drift.json", "", CodeUnsupportedStrategy},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("testdata", tt.file))
			if err != nil {
				t.Fatal(err)
			}
			result, err := ReplayJSON(raw)
			if code := CodeOf(err); code != tt.code {
				t.Fatalf("error code = %q, want %q", code, tt.code)
			}
			if tt.code == "" && result.Conclusion != tt.conclusion {
				t.Fatalf("conclusion = %q, want %q", result.Conclusion, tt.conclusion)
			}
		})
	}
	baseRaw, err := os.ReadFile(filepath.Join("testdata", "synthetic_within_budget.json"))
	if err != nil {
		t.Fatal(err)
	}
	base, err := ReplayJSON(baseRaw)
	if err != nil {
		t.Fatal(err)
	}
	advisoryRaw, err := os.ReadFile(filepath.Join("testdata", "corpus_advisory_optional.json"))
	if err != nil {
		t.Fatal(err)
	}
	advisory, err := ReplayJSON(advisoryRaw)
	if err != nil {
		t.Fatal(err)
	}
	if base.Conclusion != advisory.Conclusion || advisory.Corpus == nil || advisory.Corpus.LabeledCases != 0 || advisory.Corpus.Trend != "insufficient" {
		t.Fatalf("unexpected advisory isolation: base=%q advisory=%#v", base.Conclusion, advisory)
	}
	in, err := ParseFixtureJSON(baseRaw)
	if err != nil {
		t.Fatal(err)
	}
	in.Stream = true
	stream, err := Audit(in)
	if err != nil {
		t.Fatal(err)
	}
	if err := CompareParity(base, stream); err != nil {
		t.Fatalf("run/stream parity: %v", err)
	}
}

func TestPriorityTieAndEquivalentInputOrderingAreStable(t *testing.T) {
	facts, err := CanonicalSchemaFacts(map[string]any{"type": "object"})
	if err != nil {
		t.Fatal(err)
	}
	base := Input{
		Version: AuditVersion, SnapshotID: "stable-order",
		Tools: []AdmittedTool{
			{Identity: "zeta", Source: "local", Admitted: true, Priority: 10, Schema: facts},
			{Identity: "alpha", Source: "local", Admitted: true, Priority: 10, Schema: facts},
		},
		Cases:      []TaskCase{{ID: "all", Expected: []string{"alpha"}, Allowed: []string{"alpha", "zeta"}}},
		Strategies: []StrategySpec{{Kind: StrategyPriorityTopK, K: 1}},
	}
	reordered := base
	reordered.Tools = append([]AdmittedTool(nil), base.Tools[1], base.Tools[0])
	first, err := Audit(base)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Audit(reordered)
	if err != nil {
		t.Fatal(err)
	}
	if first.Digest != second.Digest || len(first.Strategies) != 1 || len(first.Strategies[0].Selected) != 1 || first.Strategies[0].Selected[0] != "alpha" {
		t.Fatalf("unstable priority tie: first=%#v second=%#v", first, second)
	}
}
