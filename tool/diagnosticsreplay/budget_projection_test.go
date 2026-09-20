package diagnosticsreplay

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/FelixSeptem/baymax/context/budgetprojection"
)

const budgetProjectionFixturePath = "testdata/budget_projection.v1.json"

func loadBudgetProjectionFixture(t *testing.T) budgetprojection.BudgetProjectionFixture {
	t.Helper()
	raw, err := os.ReadFile(budgetProjectionFixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	fixture, err := ParseBudgetProjectionFixtureJSON(raw)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	return fixture
}

func encodeBudgetProjectionFixture(t *testing.T, fixture budgetprojection.BudgetProjectionFixture) []byte {
	t.Helper()
	encoded, err := json.Marshal(fixture)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	return encoded
}

func TestReplayBudgetProjectionFixtureFile(t *testing.T) {
	raw, err := os.ReadFile(budgetProjectionFixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	first, err := ReplayBudgetProjectionFixtureJSON(raw)
	if err != nil {
		t.Fatalf("replay fixture: %v", err)
	}
	if first.Version != BudgetProjectionFixtureV1 {
		t.Fatalf("version = %q, want %q", first.Version, BudgetProjectionFixtureV1)
	}
	if len(first.Cases) == 0 || len(first.Benchmarks) == 0 {
		t.Fatalf("fixture must contain both cases and benchmarks")
	}
	for _, item := range first.Cases {
		if !item.Idempotent {
			t.Fatalf("case %q is not idempotent", item.CaseID)
		}
		if item.Digest != item.ReplayDigest {
			t.Fatalf("case %q digest mismatch: %s vs %s", item.CaseID, item.Digest, item.ReplayDigest)
		}
	}
	for _, item := range first.Benchmarks {
		if !item.Idempotent {
			t.Fatalf("benchmark %q is not idempotent", item.BenchmarkID)
		}
		if item.Digest != item.ReplayDigest {
			t.Fatalf("benchmark %q recompute mismatch: %s vs %s", item.BenchmarkID, item.Digest, item.ReplayDigest)
		}
	}

	second, err := ReplayBudgetProjectionFixtureJSON(raw)
	if err != nil {
		t.Fatalf("replay fixture twice: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("replay is not idempotent:\n%+v\n%+v", first, second)
	}
}

func TestReplayBudgetProjectionFixtureCompletenessAndGaps(t *testing.T) {
	fixture := loadBudgetProjectionFixture(t)
	byID := map[string]budgetprojection.BudgetProjectionCase{}
	for _, item := range fixture.Cases {
		byID[item.CaseID] = item
	}
	if !byID["full-facts-complete"].Expected.Complete {
		t.Fatalf("full-facts-complete must be complete")
	}
	if byID["all-dimensions-missing"].Expected.Complete {
		t.Fatalf("all-dimensions-missing must not be complete")
	}
	if len(byID["all-dimensions-missing"].DeclaredGaps) != 4 {
		t.Fatalf(
			"all-dimensions-missing declared gaps = %v, want 4",
			byID["all-dimensions-missing"].DeclaredGaps,
		)
	}
	if byID["full-facts-complete"].Expected.Pressure != budgetprojection.PressureNormal {
		t.Fatalf(
			"full-facts-complete pressure = %q, want normal",
			byID["full-facts-complete"].Expected.Pressure,
		)
	}
	if byID["iteration-pressure-critical"].Expected.Pressure != budgetprojection.PressureCritical {
		t.Fatalf(
			"iteration-pressure-critical pressure = %q, want critical",
			byID["iteration-pressure-critical"].Expected.Pressure,
		)
	}
}

func TestReplayBudgetProjectionCoversEveryTaxonomyBranch(t *testing.T) {
	fixture := loadBudgetProjectionFixture(t)
	covered := map[string]bool{}
	for _, item := range fixture.Cases {
		for _, code := range item.DeclaredGaps {
			covered[code] = true
		}
	}
	for _, code := range []string{
		ReasonCodeBudgetMissingIterationLimit,
		ReasonCodeBudgetMissingToolCallLimit,
		ReasonCodeBudgetMissingTimeBudget,
		ReasonCodeBudgetMissingCostThreshold,
	} {
		if !covered[code] {
			t.Fatalf("fixture does not declare gap %q", code)
		}
	}
}

func TestReplayBudgetProjectionRejectsDrift(t *testing.T) {
	base := loadBudgetProjectionFixture(t)

	// Some mutations (for example a changed pressure level) also invalidate the
	// self-computed digest, so a case may legitimately report more than one
	// code. The assertion accepts any of the expected codes.
	mutate := func(name string, apply func(*budgetprojection.BudgetProjectionFixture), want ...string) {
		// Deep-copy through JSON so a mutation never leaks into later cases
		// (struct copy alone would share the case/benchmark slices).
		encoded := encodeBudgetProjectionFixture(t, base)
		fixture, err := ParseBudgetProjectionFixtureJSON(encoded)
		if err != nil {
			t.Fatalf("%s: copy fixture: %v", name, err)
		}
		apply(&fixture)
		_, replayErr := ReplayBudgetProjectionFixtureJSON(encodeBudgetProjectionFixture(t, fixture))
		if replayErr == nil {
			t.Fatalf("%s: expected replay to fail with %s", name, want)
		}
		var validation *ValidationError
		if !errors.As(replayErr, &validation) {
			t.Fatalf("%s: expected *ValidationError, got %T: %v", name, replayErr, replayErr)
		}
		if validation.Code != want[0] && !containsString(want, validation.Code) {
			t.Fatalf("%s: got code %q want one of %v (%s)", name, validation.Code, want, validation.Message)
		}
	}

	mutate("unknown-version", func(f *budgetprojection.BudgetProjectionFixture) {
		f.Version = "budget_projection.v2"
	}, ReasonCodeBudgetUnknownVersion)

	mutate("undeclared-gap", func(f *budgetprojection.BudgetProjectionFixture) {
		f.Cases[0].DeclaredGaps = nil
		f.Cases[0].Facts = budgetprojection.BudgetFacts{}
		f.Cases[0].Expected = budgetprojection.DeriveBudgetProjection(budgetprojection.BudgetFacts{})
		f.Cases[0].ExpectDigest = f.Cases[0].Expected.Digest
	}, ReasonCodeBudgetSchemaDrift)

	mutate("digest-mismatch", func(f *budgetprojection.BudgetProjectionFixture) {
		f.Cases[0].ExpectDigest = "deadbeef"
	}, ReasonCodeBudgetDigestMismatch)

	mutate("pressure-mismatch", func(f *budgetprojection.BudgetProjectionFixture) {
		f.Cases[0].Expected.Pressure = budgetprojection.PressureSaturated
	}, ReasonCodeBudgetPressureLevelMismatch, ReasonCodeBudgetDigestMismatch)

	mutate("notes-overflow", func(f *budgetprojection.BudgetProjectionFixture) {
		notes := make([]string, 0, budgetprojection.MaxNotes+1)
		for index := 0; index <= budgetprojection.MaxNotes; index++ {
			notes = append(notes, "overflow")
		}
		f.Cases[0].Expected.Notes = notes
	}, ReasonCodeBudgetNoteUnbounded, ReasonCodeBudgetDigestMismatch, ReasonCodeBudgetSchemaDrift)

	mutate("note-too-long", func(f *budgetprojection.BudgetProjectionFixture) {
		f.Cases[0].Expected.Notes = []string{strings.Repeat("x", budgetprojection.MaxNoteChars+1)}
	}, ReasonCodeBudgetNoteUnbounded, ReasonCodeBudgetDigestMismatch, ReasonCodeBudgetSchemaDrift)

	mutate("negative-remaining", func(f *budgetprojection.BudgetProjectionFixture) {
		negative := -1.0
		f.Cases[0].Expected.Remaining.Iteration.Remaining = &negative
	}, ReasonCodeBudgetNegativeRemaining, ReasonCodeBudgetDigestMismatch, ReasonCodeBudgetSchemaDrift)

	mutate("benchmark-metric-mismatch", func(f *budgetprojection.BudgetProjectionFixture) {
		f.Benchmarks[0].Expected.Digest = "deadbeef"
	}, ReasonCodeBudgetBenchmarkMetricMismatch)

	mutate("benchmark-unknown-outcome", func(f *budgetprojection.BudgetProjectionFixture) {
		f.Benchmarks[0].Traces[0].Steps[0].Outcome = "explode"
	}, ReasonCodeBudgetBenchmarkSchemaDrift)

	mutate("benchmark-overflow", func(f *budgetprojection.BudgetProjectionFixture) {
		traces := make([]budgetprojection.BudgetBenchmarkTrace, 0, budgetprojection.MaxTraces+1)
		for index := 0; index <= budgetprojection.MaxTraces; index++ {
			traces = append(traces, budgetprojection.BudgetBenchmarkTrace{
				TraceID: "overflow",
				Steps:   []budgetprojection.BudgetBenchmarkStep{{Outcome: budgetprojection.OutcomeProgress}},
			})
		}
		f.Benchmarks[0].Traces = traces
	}, ReasonCodeBudgetBenchmarkOverflowDrift)
}

func TestReplayBudgetProjectionIgnoresUnknownFields(t *testing.T) {
	raw, err := os.ReadFile(budgetProjectionFixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	document["unknown_future_field"] = "ignored"
	cases, ok := document["cases"].([]any)
	if !ok || len(cases) == 0 {
		t.Fatalf("fixture has no cases")
	}
	firstCase, ok := cases[0].(map[string]any)
	if !ok {
		t.Fatalf("fixture case is not an object")
	}
	firstCase["unknown_case_field"] = 42
	encoded, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("marshal patched fixture: %v", err)
	}
	if _, err := ReplayBudgetProjectionFixtureJSON(encoded); err != nil {
		t.Fatalf("unknown fields must be ignored safely: %v", err)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestParseBudgetProjectionFixtureRejectsMalformedPayload(t *testing.T) {
	if _, err := ParseBudgetProjectionFixtureJSON([]byte("{")); err == nil {
		t.Fatalf("expected malformed payload to fail")
	}
}
