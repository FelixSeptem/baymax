package diagnosticsreplay

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/FelixSeptem/baymax/runtime/evalcontract"
)

const (
	ContinuityFixtureVersionV1       = evalcontract.ContinuityComparisonVersionV1
	ReasonContinuitySchemaDrift      = evalcontract.ReasonContinuitySchemaDrift
	ReasonContinuityPrivacyViolation = evalcontract.ReasonContinuityPrivacyViolation
)

type ContinuityFixture struct {
	Version   string                            `json:"version"`
	Baseline  evalcontract.ContinuityProjection `json:"baseline"`
	Candidate evalcontract.ContinuityProjection `json:"candidate"`
	Expected  ContinuityExpectation             `json:"expected"`
}

type ContinuityExpectation struct {
	Passed bool     `json:"passed"`
	Drifts []string `json:"drifts,omitempty"`
}

func ParseContinuityFixtureJSON(raw []byte) (ContinuityFixture, error) {
	var f ContinuityFixture
	if err := json.Unmarshal(raw, &f); err != nil {
		return ContinuityFixture{}, &ValidationError{Code: ReasonCodeInvalidJSON, Message: err.Error()}
	}
	f.Version = strings.TrimSpace(strings.ToLower(f.Version))
	if f.Version != ContinuityFixtureVersionV1 {
		return ContinuityFixture{}, fmt.Errorf("%s", ReasonContinuitySchemaDrift)
	}
	if _, _, err := evalcontract.NormalizeContinuityProjection(f.Baseline); err != nil {
		return ContinuityFixture{}, err
	}
	if _, _, err := evalcontract.NormalizeContinuityProjection(f.Candidate); err != nil {
		return ContinuityFixture{}, err
	}
	return f, nil
}

func EvaluateContinuityFixture(f ContinuityFixture) error {
	if f.Version != ContinuityFixtureVersionV1 {
		return fmt.Errorf("%s", ReasonContinuitySchemaDrift)
	}
	result, err := evalcontract.CompareContinuity(f.Baseline, f.Candidate)
	if err != nil {
		return err
	}
	if result.Passed != f.Expected.Passed {
		return fmt.Errorf("%s", ReasonContinuitySchemaDrift)
	}
	if len(f.Expected.Drifts) != len(result.Drifts) {
		return fmt.Errorf("%s", ReasonContinuitySchemaDrift)
	}
	for i, drift := range result.Drifts {
		if drift.Class != f.Expected.Drifts[i] {
			return fmt.Errorf("%s", drift.Class)
		}
	}
	return nil
}
