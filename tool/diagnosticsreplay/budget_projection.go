package diagnosticsreplay

import (
	"errors"

	"github.com/FelixSeptem/baymax/context/budgetprojection"
)

// BudgetProjectionFixtureV1 is the versioned namespace for the derived budget
// projection (budget facts -> bounded read-only remaining projection). It is
// owned by context/budgetprojection; replay re-exports it so tooling and gate
// scripts reference one constant.
const BudgetProjectionFixtureV1 = budgetprojection.FixtureVersionBudgetV1

// Derived budget projection drift and gap classifications. These are aliases,
// not copies: the replay tooling, the contract layer, the spec, and the gate
// scripts must share exactly one vocabulary. A divergence here is a taxonomy
// drift, not a new code.
const (
	ReasonCodeBudgetSchemaDrift                = budgetprojection.CodeSchemaDrift
	ReasonCodeBudgetUnknownVersion             = budgetprojection.CodeUnknownVersion
	ReasonCodeBudgetMissingIterationLimit      = budgetprojection.CodeMissingIterationLimit
	ReasonCodeBudgetMissingToolCallLimit       = budgetprojection.CodeMissingToolCallLimit
	ReasonCodeBudgetMissingTimeBudget          = budgetprojection.CodeMissingTimeBudget
	ReasonCodeBudgetMissingCostThreshold       = budgetprojection.CodeMissingCostThreshold
	ReasonCodeBudgetNegativeRemaining          = budgetprojection.CodeNegativeRemaining
	ReasonCodeBudgetRatioOutOfRange            = budgetprojection.CodeRatioOutOfRange
	ReasonCodeBudgetPressureLevelMismatch      = budgetprojection.CodePressureLevelMismatch
	ReasonCodeBudgetNoteUnbounded              = budgetprojection.CodeNoteUnbounded
	ReasonCodeBudgetDigestMismatch             = budgetprojection.CodeDigestMismatch
	ReasonCodeBudgetOverflowDrift              = budgetprojection.CodeOverflowDrift
	ReasonCodeBudgetWritebackShape             = budgetprojection.CodeWritebackShapeDetected
	ReasonCodeBudgetBenchmarkSchemaDrift       = budgetprojection.CodeBenchmarkSchemaDrift
	ReasonCodeBudgetBenchmarkMetricMismatch    = budgetprojection.CodeBenchmarkMetricMismatch
	ReasonCodeBudgetBenchmarkRecoveryRecompute = budgetprojection.CodeBenchmarkRecoveryRecompute
	ReasonCodeBudgetBenchmarkOverflowDrift     = budgetprojection.CodeBenchmarkOverflowDrift
	ReasonCodeBudgetReplayNotIdempotent        = budgetprojection.CodeReplayNotIdempotent
)

// BudgetProjectionReplayCase is the normalized, deterministic per-case replay
// record. It carries classifications, digests, and flags only: no raw
// reasoning, transcript or unbounded payload may ever appear here.
//
// Codes is the remaining, non-pinned classification set. Declared gaps that
// still reproduce are pinned evidence and are not treated as failures.
type BudgetProjectionReplayCase struct {
	CaseID       string   `json:"case_id"`
	Codes        []string `json:"codes,omitempty"`
	Digest       string   `json:"digest"`
	ReplayDigest string   `json:"replay_digest"`
	Idempotent   bool     `json:"idempotent"`
	Complete     bool     `json:"complete"`
}

// BudgetProjectionBenchmarkReplayCase is the normalized benchmark replay record.
type BudgetProjectionBenchmarkReplayCase struct {
	BenchmarkID  string   `json:"benchmark_id"`
	Codes        []string `json:"codes,omitempty"`
	Digest       string   `json:"digest"`
	ReplayDigest string   `json:"replay_digest"`
	Idempotent   bool     `json:"idempotent"`
	TraceCount   int      `json:"trace_count"`
	StepCount    int      `json:"step_count"`
}

// BudgetProjectionReplayResult is the normalized replay output. Replaying the
// same fixture twice yields identical results: replay reads no clock, no
// network, and no runtime state.
type BudgetProjectionReplayResult struct {
	Version    string                                `json:"version"`
	Cases      []BudgetProjectionReplayCase          `json:"cases"`
	Benchmarks []BudgetProjectionBenchmarkReplayCase `json:"benchmarks"`
}

// ParseBudgetProjectionFixtureJSON parses and validates a versioned derived
// budget projection fixture without executing it.
func ParseBudgetProjectionFixtureJSON(raw []byte) (budgetprojection.BudgetProjectionFixture, error) {
	return budgetprojection.ParseBudgetProjectionFixtureJSON(raw)
}

// ReplayBudgetProjectionFixtureJSON performs the offline, read-only replay of a
// budget_projection.v1 fixture.
//
// It calls no model, executes no tool, touches no runtime state, and mutates no
// counter. Validation failures are returned as *ValidationError with the
// contract's stable code so callers never string-match error text.
func ReplayBudgetProjectionFixtureJSON(raw []byte) (BudgetProjectionReplayResult, error) {
	fixture, err := budgetprojection.ParseBudgetProjectionFixtureJSON(raw)
	if err != nil {
		return BudgetProjectionReplayResult{}, classifyBudgetProjectionError(err, "")
	}

	result := BudgetProjectionReplayResult{
		Version:    fixture.Version,
		Cases:      make([]BudgetProjectionReplayCase, 0, len(fixture.Cases)),
		Benchmarks: make([]BudgetProjectionBenchmarkReplayCase, 0, len(fixture.Benchmarks)),
	}
	for index := range fixture.Cases {
		record, err := replayBudgetProjectionCase(fixture.Cases[index])
		if err != nil {
			return BudgetProjectionReplayResult{}, classifyBudgetProjectionError(err, fixture.Cases[index].CaseID)
		}
		result.Cases = append(result.Cases, record)
	}
	for index := range fixture.Benchmarks {
		record, err := replayBudgetBenchmarkCase(fixture.Benchmarks[index])
		if err != nil {
			return BudgetProjectionReplayResult{}, classifyBudgetProjectionError(err, fixture.Benchmarks[index].BenchmarkID)
		}
		result.Benchmarks = append(result.Benchmarks, record)
	}
	return result, nil
}

// replayBudgetProjectionCase normalizes one already-validated case and proves
// digest idempotency by deriving the projection twice. A mismatch means the
// canonical form is not deterministic, which is a contract failure rather than
// a data failure.
func replayBudgetProjectionCase(projectionCase budgetprojection.BudgetProjectionCase) (BudgetProjectionReplayCase, error) {
	first := budgetprojection.DeriveBudgetProjection(projectionCase.Facts)
	replay := budgetprojection.DeriveBudgetProjection(projectionCase.Facts)
	if first.Digest != replay.Digest {
		return BudgetProjectionReplayCase{}, &ValidationError{
			Code:    ReasonCodeBudgetReplayNotIdempotent,
			Message: "budget projection digest is not deterministic",
		}
	}
	codes := budgetprojection.ValidateBudgetProjectionFixtureCase(projectionCase)
	if len(codes) > 0 {
		return BudgetProjectionReplayCase{}, &ValidationError{
			Code:    codes[0],
			Message: "budget projection case drift: " + joinCodes(codes),
		}
	}
	return BudgetProjectionReplayCase{
		CaseID:       projectionCase.CaseID,
		Codes:        codes,
		Digest:       first.Digest,
		ReplayDigest: replay.Digest,
		Idempotent:   first.Digest == replay.Digest,
		Complete:     first.Complete,
	}, nil
}

// replayBudgetBenchmarkCase normalizes one benchmark case, recomputes metrics
// twice and proves that a snapshot/recovery style recompute is deterministic.
func replayBudgetBenchmarkCase(benchmark budgetprojection.BudgetBenchmarkCase) (BudgetProjectionBenchmarkReplayCase, error) {
	first, err := budgetprojection.EvaluateBudgetBenchmark(benchmark.Traces)
	if err != nil {
		return BudgetProjectionBenchmarkReplayCase{}, err
	}
	replay, err := budgetprojection.EvaluateBudgetBenchmark(benchmark.Traces)
	if err != nil {
		return BudgetProjectionBenchmarkReplayCase{}, err
	}
	if first.Digest != replay.Digest {
		return BudgetProjectionBenchmarkReplayCase{}, &ValidationError{
			Code:    ReasonCodeBudgetBenchmarkRecoveryRecompute,
			Message: "budget benchmark recompute is not deterministic",
		}
	}
	codes := budgetprojection.ValidateBudgetBenchmarkCase(benchmark)
	if len(codes) > 0 {
		return BudgetProjectionBenchmarkReplayCase{}, &ValidationError{
			Code:    codes[0],
			Message: "budget benchmark case drift: " + joinCodes(codes),
		}
	}
	return BudgetProjectionBenchmarkReplayCase{
		BenchmarkID:  benchmark.BenchmarkID,
		Codes:        codes,
		Digest:       first.Digest,
		ReplayDigest: replay.Digest,
		Idempotent:   first.Digest == replay.Digest,
		TraceCount:   first.TraceCount,
		StepCount:    first.StepCount,
	}, nil
}

// classifyBudgetProjectionError converts a contract-layer classification into a
// replay ValidationError, preserving the stable code verbatim.
func classifyBudgetProjectionError(err error, caseID string) error {
	var classified *budgetprojection.Error
	if errors.As(err, &classified) {
		return &ValidationError{Code: classified.Code, Message: classified.Message}
	}
	var validation *ValidationError
	if errors.As(err, &validation) {
		return validation
	}
	return &ValidationError{Code: ReasonCodeBudgetSchemaDrift, Message: err.Error()}
}

func joinCodes(codes []string) string {
	if len(codes) == 0 {
		return ""
	}
	joined := codes[0]
	for _, code := range codes[1:] {
		joined += "," + code
	}
	return joined
}
