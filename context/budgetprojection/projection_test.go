package budgetprojection

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"
)

func fullFacts() BudgetFacts {
	return BudgetFacts{
		Iteration: &BudgetLimitUsage{Limit: 10, Used: 3},
		ToolCall:  &BudgetLimitUsage{Limit: 20, Used: 5},
		Time:      &BudgetTimeFacts{BudgetMS: 60_000, ElapsedMS: 12_000},
		Cost:      &BudgetCostFacts{EstimateTotal: 2.5, HardThreshold: 10, DegradeThreshold: 6},
		Admission: BudgetAdmissionFacts{Decision: "allow"},
		Parent:    &BudgetParentFacts{RemainingMS: 30_000},
		Plan:      &BudgetPlanFacts{ChangeTotal: 4, RecoverCount: 0, Status: "active"},
	}
}

func TestDeriveBudgetProjectionIsDeterministicAndComplete(t *testing.T) {
	facts := fullFacts()
	first := DeriveBudgetProjection(facts)
	second := DeriveBudgetProjection(facts)
	if first.Digest != second.Digest {
		t.Fatalf("digest not deterministic: %s vs %s", first.Digest, second.Digest)
	}
	if !first.Complete {
		t.Fatalf("expected complete projection for full facts")
	}
	if first.Version != FixtureVersionBudgetV1 {
		t.Fatalf("unexpected version %q", first.Version)
	}
	if float64Value(first.Remaining.Iteration.Remaining) != 7 {
		t.Fatalf("iteration remaining = %v, want 7", first.Remaining.Iteration.Remaining)
	}
	if float64Value(first.Remaining.TimeMS.Remaining) != 48_000 {
		t.Fatalf("time remaining = %v, want 48000", first.Remaining.TimeMS.Remaining)
	}
	if float64Value(first.Remaining.CostHeadroom.Remaining) != 7.5 {
		t.Fatalf("cost headroom = %v, want 7.5", first.Remaining.CostHeadroom.Remaining)
	}
	if first.Pressure != PressureNormal {
		t.Fatalf("pressure = %q, want normal", first.Pressure)
	}
}

func TestDeriveBudgetProjectionKeepsAbsentDimensionsNullable(t *testing.T) {
	facts := BudgetFacts{Admission: BudgetAdmissionFacts{Decision: "allow"}}
	projection := DeriveBudgetProjection(facts)
	if projection.Complete {
		t.Fatalf("projection must not be complete when every dimension is absent")
	}
	encoded, err := json.Marshal(projection.Remaining)
	if err != nil {
		t.Fatalf("marshal absent projection dimensions: %v", err)
	}
	var remaining map[string]map[string]any
	if err := json.Unmarshal(encoded, &remaining); err != nil {
		t.Fatalf("decode absent projection dimensions: %v", err)
	}
	for name, dimension := range remaining {
		for _, field := range []string{"remaining", "limit", "used", "ratio"} {
			if _, present := dimension[field]; present {
				t.Fatalf("%s must omit %q when unavailable: %s", name, field, encoded)
			}
		}
	}
	for _, entry := range []struct {
		name      string
		dimension BudgetDimension
	}{
		{"iteration", projection.Remaining.Iteration},
		{"tool_call", projection.Remaining.ToolCall},
		{"time_ms", projection.Remaining.TimeMS},
		{"cost_headroom", projection.Remaining.CostHeadroom},
	} {
		if entry.dimension.Available {
			t.Fatalf("%s must be unavailable", entry.name)
		}
		if entry.dimension.Remaining != nil || entry.dimension.Limit != nil ||
			entry.dimension.Used != nil || entry.dimension.Ratio != nil {
			t.Fatalf("%s must not carry fabricated values: %+v", entry.name, entry.dimension)
		}
	}
	if projection.Pressure != PressureNone {
		t.Fatalf("pressure = %q, want none", projection.Pressure)
	}
	codes := ClassifyBudgetFactsGaps(facts)
	for _, want := range []string{
		CodeMissingIterationLimit,
		CodeMissingToolCallLimit,
		CodeMissingTimeBudget,
		CodeMissingCostThreshold,
	} {
		if !containsCode(codes, want) {
			t.Fatalf("expected gap %q in %v", want, codes)
		}
	}
}

func TestDeriveBudgetProjectionBoundsAndSortsNotes(t *testing.T) {
	facts := BudgetFacts{
		ToolCall:  &BudgetLimitUsage{Limit: 1, Used: 2},
		Time:      &BudgetTimeFacts{BudgetMS: 1, ElapsedMS: 2},
		Cost:      &BudgetCostFacts{HardThreshold: 1, EstimateTotal: 2},
		Admission: BudgetAdmissionFacts{Decision: "deny"},
		Parent:    &BudgetParentFacts{RemainingMS: 0},
		Plan:      &BudgetPlanFacts{RecoverCount: 1},
	}
	projection := DeriveBudgetProjection(facts)
	if len(projection.Notes) > MaxNotes {
		t.Fatalf("notes count = %d, want at most %d: %v", len(projection.Notes), MaxNotes, projection.Notes)
	}
	if !sort.StringsAreSorted(projection.Notes) {
		t.Fatalf("notes must be canonically sorted: %v", projection.Notes)
	}
	for _, note := range projection.Notes {
		if len(note) > MaxNoteChars {
			t.Fatalf("note %q exceeds %d characters", note, MaxNoteChars)
		}
	}
}

func TestDeriveBudgetProjectionKeepsAdmissionNoteBounded(t *testing.T) {
	facts := fullFacts()
	facts.Admission.Decision = strings.Repeat("d", MaxNoteChars+1)
	projection := DeriveBudgetProjection(facts)
	for _, note := range projection.Notes {
		if len(note) > MaxNoteChars {
			t.Fatalf("note %q exceeds %d characters", note, MaxNoteChars)
		}
	}
}

func TestDerivePressureLevelBoundaries(t *testing.T) {
	cases := []struct {
		name  string
		ratio float64
		want  BudgetPressureLevel
	}{
		{"normal", 0, PressureNormal},
		{"below_elevated", 0.4999, PressureNormal},
		{"elevated", 0.5, PressureElevated},
		{"below_critical", 0.7999, PressureElevated},
		{"critical", 0.8, PressureCritical},
		{"below_saturated", 0.9999, PressureCritical},
		{"saturated", 1, PressureSaturated},
		{"over", 1.5, PressureSaturated},
	}
	for _, item := range cases {
		facts := BudgetFacts{
			Iteration: &BudgetLimitUsage{Limit: 1000_000, Used: int(item.ratio * 1000_000)},
			Admission: BudgetAdmissionFacts{Decision: "allow"},
		}
		projection := DeriveBudgetProjection(facts)
		if projection.Pressure != item.want {
			t.Fatalf("%s: pressure = %q, want %q", item.name, projection.Pressure, item.want)
		}
	}
}

func TestClassifyBudgetProjectionRejectsUnboundedNotes(t *testing.T) {
	projection := DeriveBudgetProjection(fullFacts())
	projection.Notes = make([]string, 0, MaxNotes+1)
	for index := 0; index <= MaxNotes; index++ {
		projection.Notes = append(projection.Notes, "note")
	}
	if codes := ClassifyBudgetProjection(projection); !containsCode(codes, CodeNoteUnbounded) {
		t.Fatalf("expected %q in %v", CodeNoteUnbounded, codes)
	}

	projection.Notes = []string{strings.Repeat("x", MaxNoteChars+1)}
	if codes := ClassifyBudgetProjection(projection); !containsCode(codes, CodeNoteUnbounded) {
		t.Fatalf("expected %q for overlong note in %v", CodeNoteUnbounded, codes)
	}
}

func TestClassifyBudgetProjectionRejectsOversizedCanonicalShape(t *testing.T) {
	projection := DeriveBudgetProjection(fullFacts())
	projection.Version = strings.Repeat("v", MaxProjectionBytes)
	if codes := ClassifyBudgetProjection(projection); !containsCode(codes, CodeOverflowDrift) {
		t.Fatalf("expected %q for oversized projection in %v", CodeOverflowDrift, codes)
	}
}

func TestClassifyBudgetProjectionRejectsNegativeAndInvalidRatio(t *testing.T) {
	projection := DeriveBudgetProjection(fullFacts())
	projection.Remaining.Iteration.Remaining = float64Pointer(-1)
	if codes := ClassifyBudgetProjection(projection); !containsCode(codes, CodeNegativeRemaining) {
		t.Fatalf("expected %q in %v", CodeNegativeRemaining, codes)
	}

	projection = DeriveBudgetProjection(fullFacts())
	projection.Remaining.CostHeadroom.Ratio = float64Pointer(4)
	if codes := ClassifyBudgetProjection(projection); !containsCode(codes, CodeRatioOutOfRange) {
		t.Fatalf("expected %q in %v", CodeRatioOutOfRange, codes)
	}
}

func TestValidateBudgetProjectionCaseClassifiesMismatch(t *testing.T) {
	facts := fullFacts()
	expected := DeriveBudgetProjection(facts)
	if codes := ValidateBudgetProjectionCase(facts, expected); len(codes) != 0 {
		t.Fatalf("expected no drift for matching projection, got %v", codes)
	}

	expected.Pressure = PressureSaturated
	if codes := ValidateBudgetProjectionCase(facts, expected); !containsCode(codes, CodePressureLevelMismatch) {
		t.Fatalf("expected %q in %v", CodePressureLevelMismatch, codes)
	}

	expected = DeriveBudgetProjection(facts)
	expected.Digest = "deadbeef"
	if codes := ValidateBudgetProjectionCase(facts, expected); !containsCode(codes, CodeDigestMismatch) {
		t.Fatalf("expected %q in %v", CodeDigestMismatch, codes)
	}

	expected = DeriveBudgetProjection(facts)
	expected.Remaining.Iteration.Remaining = float64Pointer(99)
	if codes := ValidateBudgetProjectionCase(facts, expected); !containsCode(codes, CodeSchemaDrift) {
		t.Fatalf("expected %q in %v", CodeSchemaDrift, codes)
	}
}

func TestClassifyBudgetProjectionRejectsUnknownVersionAndPressure(t *testing.T) {
	projection := DeriveBudgetProjection(fullFacts())
	projection.Version = "budget_projection.v0"
	if codes := ClassifyBudgetProjection(projection); !containsCode(codes, CodeUnknownVersion) {
		t.Fatalf("expected %q in %v", CodeUnknownVersion, codes)
	}

	projection = DeriveBudgetProjection(fullFacts())
	projection.Pressure = "unknown"
	if codes := ClassifyBudgetProjection(projection); !containsCode(codes, CodeSchemaDrift) {
		t.Fatalf("expected %q in %v", CodeSchemaDrift, codes)
	}
}

func TestBudgetProjectionTaxonomyIsStableAndComplete(t *testing.T) {
	if len(BudgetProjectionTaxonomy) != 18 {
		t.Fatalf("taxonomy size = %d, want 18", len(BudgetProjectionTaxonomy))
	}
	seen := map[string]struct{}{}
	for _, code := range BudgetProjectionTaxonomy {
		if _, ok := seen[code]; ok {
			t.Fatalf("duplicate taxonomy code %q", code)
		}
		seen[code] = struct{}{}
	}
}

func TestDetectWritebackShapeIsClean(t *testing.T) {
	if codes := DetectWritebackShape(); len(codes) != 0 {
		t.Fatalf("unexpected writeback shape codes: %v", codes)
	}
}

func TestParseBudgetProjectionFixtureJSONRejectsInvalidPayloads(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{"empty", "", CodeSchemaDrift},
		{"malformed", "{", CodeSchemaDrift},
		{"unknown_version", `{"version":"budget_projection.v2","cases":[]}`, CodeUnknownVersion},
		{"no_cases", `{"version":"budget_projection.v1"}`, CodeSchemaDrift},
	}
	for _, item := range cases {
		if _, err := ParseBudgetProjectionFixtureJSON([]byte(item.raw)); err == nil {
			t.Fatalf("%s: expected error %q", item.name, item.want)
		} else if typed, ok := err.(*Error); !ok || typed.Code != item.want {
			t.Fatalf("%s: got %v, want code %q", item.name, err, item.want)
		}
	}
}

func TestParseBudgetProjectionFixtureJSONBoundsCaseCount(t *testing.T) {
	raw := `{"version":"budget_projection.v1","cases":[` + strings.Repeat(`{"case_id":"c"},`, MaxCases) + `{"case_id":"last"}]}`
	if _, err := ParseBudgetProjectionFixtureJSON([]byte(raw)); err == nil {
		t.Fatalf("expected overflow error for %d cases", MaxCases+1)
	} else if typed, ok := err.(*Error); !ok || typed.Code != CodeOverflowDrift {
		t.Fatalf("got %v, want %q", err, CodeOverflowDrift)
	}
}

func TestValidateBudgetProjectionFixtureCasePinsDeclaredGaps(t *testing.T) {
	facts := BudgetFacts{Admission: BudgetAdmissionFacts{Decision: "allow"}}
	gaps := ClassifyBudgetFactsGaps(facts)
	projectionCase := BudgetProjectionCase{
		CaseID:       "all-missing",
		Facts:        facts,
		Expected:     DeriveBudgetProjection(facts),
		DeclaredGaps: gaps,
	}
	if codes := ValidateBudgetProjectionFixtureCase(projectionCase); len(codes) != 0 {
		t.Fatalf("expected pinned case to validate, got %v", codes)
	}

	projectionCase.DeclaredGaps = nil
	if codes := ValidateBudgetProjectionFixtureCase(projectionCase); !containsCode(codes, CodeSchemaDrift) {
		t.Fatalf("expected undeclared gap to fail, got %v", codes)
	}
}

func containsCode(codes []string, want string) bool {
	for _, code := range codes {
		if code == want {
			return true
		}
	}
	return false
}
