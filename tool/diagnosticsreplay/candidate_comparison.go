package diagnosticsreplay

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/FelixSeptem/baymax/runtime/evalcontract"
)

const CandidateComparisonFixtureVersionV1 = evalcontract.CandidateComparisonV1

type CandidateComparisonFixture struct {
	Version  string                                `json:"version"`
	Input    evalcontract.CandidateComparisonInput `json:"input"`
	Expected CandidateComparisonExpectation        `json:"expected"`
}

type CandidateComparisonExpectation struct {
	Status      string                     `json:"status"`
	Reason      string                     `json:"reason,omitempty"`
	SelectedID  string                     `json:"selected_id,omitempty"`
	ScoreStatus string                     `json:"score_status,omitempty"`
	ReviewOnly  bool                       `json:"review_only"`
	Axes        *CandidateComparisonAxes   `json:"axes,omitempty"`
	Totals      *CandidateComparisonTotals `json:"totals,omitempty"`
}

type CandidateComparisonAxes struct {
	Quality     evalcontract.CandidateEvidence `json:"quality"`
	Reliability evalcontract.CandidateEvidence `json:"reliability"`
	Contract    evalcontract.CandidateEvidence `json:"contract"`
	Latency     evalcontract.CandidateEvidence `json:"latency"`
	Cost        evalcontract.CandidateEvidence `json:"cost"`
}

type CandidateComparisonTotals struct {
	Attempts   int   `json:"attempts"`
	Retries    int   `json:"retries"`
	CacheHits  int   `json:"cache_hits"`
	Incomplete int   `json:"incomplete"`
	ModelCalls int   `json:"model_calls"`
	CostMilli  int64 `json:"cost_milli"`
	Runs       int   `json:"runs"`
}

func ParseCandidateComparisonFixtureJSON(raw []byte) (CandidateComparisonFixture, error) {
	var fixture CandidateComparisonFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		return CandidateComparisonFixture{}, &ValidationError{Code: ReasonCodeInvalidJSON, Message: err.Error()}
	}
	fixture.Version = strings.ToLower(strings.TrimSpace(fixture.Version))
	if fixture.Version != CandidateComparisonFixtureVersionV1 {
		return CandidateComparisonFixture{}, fmt.Errorf("%s", evalcontract.ReasonCandidateComparisonVersionDrift)
	}
	return fixture, nil
}

// EvaluateCandidateComparisonFixture is replay-only: it invokes the pure
// evalcontract reducer and never enters a runtime, provider, tool, or network
// execution path.
func EvaluateCandidateComparisonFixture(fixture CandidateComparisonFixture) error {
	if fixture.Version != CandidateComparisonFixtureVersionV1 {
		return fmt.Errorf("%s", evalcontract.ReasonCandidateComparisonVersionDrift)
	}
	result, err := evalcontract.BuildCandidateComparison(fixture.Input)
	if fixture.Expected.Status == "error" {
		if fixture.Expected.Reason == "" || err == nil || err.Error() != fixture.Expected.Reason {
			return fmt.Errorf("%s", evalcontract.ReasonCandidateComparisonNonActionable)
		}
		return nil
	}
	if err != nil {
		return err
	}
	if fixture.Expected.Status != "success" || result.Selected.CandidateID != fixture.Expected.SelectedID || result.Score.Status != fixture.Expected.ScoreStatus || result.ReviewOnly != fixture.Expected.ReviewOnly {
		return fmt.Errorf("%s", evalcontract.ReasonCandidateComparisonNonActionable)
	}
	if fixture.Expected.Axes != nil {
		axes := CandidateComparisonAxes{Quality: result.Quality, Reliability: result.Reliability, Contract: result.Contract, Latency: result.Latency, Cost: result.Cost}
		if !reflect.DeepEqual(*fixture.Expected.Axes, axes) {
			return fmt.Errorf("%s", evalcontract.ReasonCandidateEvidenceUnavailable)
		}
	}
	if fixture.Expected.Totals != nil {
		totals := CandidateComparisonTotals{Attempts: result.AttemptTotal, Retries: result.RetryTotal, CacheHits: result.CacheHitTotal, Incomplete: result.IncompleteTotal, ModelCalls: result.ModelCallTotal, CostMilli: result.CostTotalMilli, Runs: result.RunCount}
		if *fixture.Expected.Totals != totals {
			return fmt.Errorf("%s", evalcontract.ReasonCandidateCostLedgerDrift)
		}
	}
	return nil
}
