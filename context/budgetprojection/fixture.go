package budgetprojection

import (
	"encoding/json"
	"strings"
)

// BudgetProjectionCase is one fixture case: a facts snapshot, the expected
// projection and the declared gaps that must be pinned.
type BudgetProjectionCase struct {
	CaseID       string           `json:"case_id"`
	Facts        BudgetFacts      `json:"facts"`
	Expected     BudgetProjection `json:"expected"`
	DeclaredGaps []string         `json:"declared_gaps"`
	ExpectDigest string           `json:"expect_digest,omitempty"`
}

// BudgetBenchmarkCase is one benchmark fixture case.
type BudgetBenchmarkCase struct {
	BenchmarkID  string                 `json:"benchmark_id"`
	Traces       []BudgetBenchmarkTrace `json:"traces"`
	Expected     BudgetBenchmarkMetrics `json:"expected"`
	DeclaredGaps []string               `json:"declared_gaps"`
}

// BudgetProjectionFixture is the versioned offline fixture envelope.
type BudgetProjectionFixture struct {
	Version    string                 `json:"version"`
	Cases      []BudgetProjectionCase `json:"cases"`
	Benchmarks []BudgetBenchmarkCase  `json:"benchmarks"`
}

// ParseBudgetProjectionFixtureJSON parses and bounds-checks a fixture. Unknown
// fields are ignored safely; unknown versions are rejected.
func ParseBudgetProjectionFixtureJSON(raw []byte) (BudgetProjectionFixture, error) {
	if len(raw) == 0 {
		return BudgetProjectionFixture{}, newError(CodeSchemaDrift, "fixture payload is empty")
	}
	if len(raw) > MaxFixtureBytes {
		return BudgetProjectionFixture{}, newError(
			CodeOverflowDrift, "fixture payload %d bytes exceeds limit %d", len(raw), MaxFixtureBytes,
		)
	}
	var fixture BudgetProjectionFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		return BudgetProjectionFixture{}, newError(CodeSchemaDrift, "fixture json invalid: %v", err)
	}
	if fixture.Version != FixtureVersionBudgetV1 {
		return BudgetProjectionFixture{}, newError(
			CodeUnknownVersion, "unsupported fixture version %q", fixture.Version,
		)
	}
	if len(fixture.Cases) == 0 && len(fixture.Benchmarks) == 0 {
		return BudgetProjectionFixture{}, newError(
			CodeSchemaDrift, "fixture must contain at least one case or benchmark",
		)
	}
	if len(fixture.Cases) > MaxCases {
		return BudgetProjectionFixture{}, newError(
			CodeOverflowDrift, "case count %d exceeds limit %d", len(fixture.Cases), MaxCases,
		)
	}
	if len(fixture.Benchmarks) > MaxBenchmarks {
		return BudgetProjectionFixture{}, newError(
			CodeBenchmarkOverflowDrift, "benchmark count %d exceeds limit %d", len(fixture.Benchmarks), MaxBenchmarks,
		)
	}
	for _, projectionCase := range fixture.Cases {
		if strings.TrimSpace(projectionCase.CaseID) == "" {
			return BudgetProjectionFixture{}, newError(CodeSchemaDrift, "case id must not be empty")
		}
	}
	for _, benchmark := range fixture.Benchmarks {
		if strings.TrimSpace(benchmark.BenchmarkID) == "" {
			return BudgetProjectionFixture{}, newError(CodeBenchmarkSchemaDrift, "benchmark id must not be empty")
		}
	}
	return fixture, nil
}

// ValidateBudgetProjectionFixtureCase pins a single projection case: shape,
// derived-vs-expected consistency, digest and declared gaps.
func ValidateBudgetProjectionFixtureCase(projectionCase BudgetProjectionCase) []string {
	codes := ValidateBudgetProjectionCase(projectionCase.Facts, projectionCase.Expected)
	if projectionCase.ExpectDigest != "" {
		actual := DeriveBudgetProjection(projectionCase.Facts)
		if projectionCase.ExpectDigest != actual.Digest {
			codes = append(codes, CodeDigestMismatch)
		}
	}
	computed := gapCodesOnly(ClassifyBudgetFactsGaps(projectionCase.Facts))
	if !sameStringSet(computed, projectionCase.DeclaredGaps) {
		// A gap that stopped reproducing and a gap that was never declared are
		// the same contract failure: the fixture must be updated explicitly.
		return []string{CodeSchemaDrift}
	}
	return dedupeStable(removePinnedGaps(codes, computed, projectionCase.DeclaredGaps))
}

// removePinnedGaps drops gap codes that were explicitly declared and are still
// reproducible: a pinned gap is evidence, not a failure.
func removePinnedGaps(codes []string, computed []string, declared []string) []string {
	pinned := map[string]struct{}{}
	for _, code := range computed {
		for _, candidate := range declared {
			if code == candidate {
				pinned[code] = struct{}{}
			}
		}
	}
	output := make([]string, 0, len(codes))
	for _, code := range codes {
		if _, ok := pinned[code]; ok {
			continue
		}
		output = append(output, code)
	}
	return output
}

// ValidateBudgetBenchmarkCase pins a single benchmark case.
func ValidateBudgetBenchmarkCase(benchmark BudgetBenchmarkCase) []string {
	codes := ValidateBudgetBenchmark(benchmark.Traces, benchmark.Expected)
	computed := []string{}
	if _, err := EvaluateBudgetBenchmark(benchmark.Traces); err != nil {
		if typed, ok := err.(*Error); ok {
			computed = append(computed, typed.Code)
		} else {
			computed = append(computed, CodeBenchmarkSchemaDrift)
		}
	}
	if !sameStringSet(computed, benchmark.DeclaredGaps) {
		codes = append(codes, CodeBenchmarkSchemaDrift)
	}
	return dedupeStable(codes)
}

var gapCodeSet = map[string]struct{}{
	CodeMissingIterationLimit: {},
	CodeMissingToolCallLimit:  {},
	CodeMissingTimeBudget:     {},
	CodeMissingCostThreshold:  {},
	CodeNegativeRemaining:     {},
	CodeRatioOutOfRange:       {},
}

func gapCodesOnly(codes []string) []string {
	output := make([]string, 0, len(codes))
	for _, code := range codes {
		if _, ok := gapCodeSet[code]; ok {
			output = append(output, code)
		}
	}
	return output
}

func sameStringSet(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	seen := map[string]struct{}{}
	for _, value := range left {
		seen[value] = struct{}{}
	}
	for _, value := range right {
		if _, ok := seen[value]; !ok {
			return false
		}
	}
	return true
}
