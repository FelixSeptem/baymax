package diagnosticsreplay

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	EvalFixtureVersionV1        = "evaluation_contract.v1"
	ReasonEvalCorpusDrift       = "corpus_version_drift"
	ReasonEvalBadcaseDrift      = "badcase_replay_drift"
	ReasonEvalMetricRubricDrift = "metric_rubric_drift"
	ReasonEvalAggregateConflict = "experiment_aggregate_conflict"
	ReasonEvalApprovalMissing   = "approval_missing"
)

type EvalContractFixture struct {
	Version string             `json:"version"`
	Cases   []EvalContractCase `json:"cases"`
}

type EvalContractCase struct {
	Name     string                  `json:"name"`
	Expected EvalContractObservation `json:"expected"`
	Observed EvalContractObservation `json:"observed"`
}

type EvalContractObservation struct {
	CorpusVersion         string `json:"corpus_version,omitempty"`
	RubricDigest          string `json:"rubric_digest,omitempty"`
	BadcaseExpectedDigest string `json:"badcase_expected_digest,omitempty"`
	BadcaseObservedDigest string `json:"badcase_observed_digest,omitempty"`
	AggregateDigest       string `json:"aggregate_digest,omitempty"`
	ApprovalStatus        string `json:"approval_status,omitempty"`
	ReviewerID            string `json:"reviewer_id,omitempty"`
}

func ParseEvalContractFixtureJSON(raw []byte) (EvalContractFixture, error) {
	var f EvalContractFixture
	if err := json.Unmarshal(raw, &f); err != nil {
		return EvalContractFixture{}, &ValidationError{Code: ReasonCodeInvalidJSON, Message: err.Error()}
	}
	f.Version = strings.ToLower(strings.TrimSpace(f.Version))
	if f.Version != EvalFixtureVersionV1 {
		return EvalContractFixture{}, &ValidationError{Code: ReasonEvalCorpusDrift, Message: fmt.Sprintf("unsupported fixture version %q", f.Version)}
	}
	if len(f.Cases) == 0 {
		return EvalContractFixture{}, &ValidationError{Code: ReasonCodeMissingRequiredField, Message: "cases must not be empty"}
	}
	for i := range f.Cases {
		f.Cases[i].Name = strings.TrimSpace(f.Cases[i].Name)
		if f.Cases[i].Name == "" {
			return EvalContractFixture{}, &ValidationError{Code: ReasonCodeMissingRequiredField, Message: fmt.Sprintf("cases[%d].name is required", i)}
		}
	}
	return f, nil
}

func EvaluateEvalContractFixture(f EvalContractFixture) error {
	if f.Version != EvalFixtureVersionV1 {
		return &ValidationError{Code: ReasonEvalCorpusDrift, Message: "unsupported fixture version"}
	}
	for _, c := range f.Cases {
		if c.Expected.CorpusVersion != "" && c.Expected.CorpusVersion != c.Observed.CorpusVersion {
			return &ValidationError{Code: ReasonEvalCorpusDrift, Message: c.Name}
		}
		if c.Expected.RubricDigest != "" && c.Expected.RubricDigest != c.Observed.RubricDigest {
			return &ValidationError{Code: ReasonEvalMetricRubricDrift, Message: c.Name}
		}
		if c.Expected.BadcaseObservedDigest != "" && c.Expected.BadcaseObservedDigest != c.Observed.BadcaseObservedDigest {
			return &ValidationError{Code: ReasonEvalBadcaseDrift, Message: c.Name}
		}
		if c.Expected.AggregateDigest != "" && c.Expected.AggregateDigest != c.Observed.AggregateDigest {
			return &ValidationError{Code: ReasonEvalAggregateConflict, Message: c.Name}
		}
		if c.Expected.ApprovalStatus == "approved" && (c.Observed.ApprovalStatus != "approved" || strings.TrimSpace(c.Observed.ReviewerID) == "") {
			return &ValidationError{Code: ReasonEvalApprovalMissing, Message: c.Name}
		}
	}
	return nil
}
