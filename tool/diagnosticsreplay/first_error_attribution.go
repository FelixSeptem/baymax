package diagnosticsreplay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/FelixSeptem/baymax/runtime/evalcontract"
)

const FirstErrorAttributionFixtureVersionV1 = evalcontract.FirstErrorAttributionVersionV1

type FirstErrorAttributionFixture struct {
	Version string                      `json:"version"`
	Cases   []FirstErrorAttributionCase `json:"cases"`
}

type FirstErrorAttributionCase struct {
	Name      string                             `json:"name"`
	Baseline  evalcontract.FirstErrorAttribution `json:"baseline"`
	Candidate evalcontract.FirstErrorAttribution `json:"candidate"`
	Expected  FirstErrorAttributionExpected      `json:"expected"`
}

type FirstErrorAttributionExpected struct {
	Passed bool     `json:"passed"`
	Drifts []string `json:"drifts,omitempty"`
	Error  string   `json:"error,omitempty"`
}

type FirstErrorAttributionReplayOutput struct {
	Version string                                  `json:"version"`
	Cases   []FirstErrorAttributionReplayCaseOutput `json:"cases"`
}

type FirstErrorAttributionReplayCaseOutput struct {
	Name              string   `json:"name"`
	Passed            bool     `json:"passed"`
	BaselineIdentity  string   `json:"baseline_identity"`
	CandidateIdentity string   `json:"candidate_identity"`
	NormalizedKind    string   `json:"normalized_kind"`
	NormalizedOwner   string   `json:"normalized_owner"`
	Drifts            []string `json:"drifts,omitempty"`
}

func ParseFirstErrorAttributionFixtureJSON(raw []byte) (FirstErrorAttributionFixture, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var fixture FirstErrorAttributionFixture
	if err := decoder.Decode(&fixture); err != nil {
		return FirstErrorAttributionFixture{}, &ValidationError{Code: ReasonCodeInvalidJSON, Message: err.Error()}
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return FirstErrorAttributionFixture{}, &ValidationError{Code: ReasonCodeInvalidJSON, Message: err.Error()}
	}
	fixture.Version = strings.ToLower(strings.TrimSpace(fixture.Version))
	if fixture.Version != FirstErrorAttributionFixtureVersionV1 || len(fixture.Cases) == 0 {
		return FirstErrorAttributionFixture{}, &ValidationError{Code: evalcontract.ReasonFirstErrorSchemaDrift, Message: "unsupported or empty first-error attribution fixture"}
	}
	seen := make(map[string]struct{}, len(fixture.Cases))
	for i := range fixture.Cases {
		fixture.Cases[i].Name = strings.TrimSpace(fixture.Cases[i].Name)
		if fixture.Cases[i].Name == "" {
			return FirstErrorAttributionFixture{}, &ValidationError{Code: evalcontract.ReasonFirstErrorSchemaDrift, Message: fmt.Sprintf("cases[%d].name is required", i)}
		}
		if _, duplicate := seen[fixture.Cases[i].Name]; duplicate {
			return FirstErrorAttributionFixture{}, &ValidationError{Code: evalcontract.ReasonFirstErrorSchemaDrift, Message: fmt.Sprintf("duplicate case %q", fixture.Cases[i].Name)}
		}
		seen[fixture.Cases[i].Name] = struct{}{}
	}
	return fixture, nil
}

func EvaluateFirstErrorAttributionFixture(fixture FirstErrorAttributionFixture) (FirstErrorAttributionReplayOutput, error) {
	if fixture.Version != FirstErrorAttributionFixtureVersionV1 || len(fixture.Cases) == 0 {
		return FirstErrorAttributionReplayOutput{}, &ValidationError{Code: evalcontract.ReasonFirstErrorSchemaDrift, Message: "unsupported or empty first-error attribution fixture"}
	}
	output := FirstErrorAttributionReplayOutput{
		Version: fixture.Version,
		Cases:   make([]FirstErrorAttributionReplayCaseOutput, 0, len(fixture.Cases)),
	}
	for _, replayCase := range fixture.Cases {
		baseline, baselineIdentity, err := evalcontract.NormalizeFirstErrorAttribution(replayCase.Baseline)
		if err != nil {
			return FirstErrorAttributionReplayOutput{}, attributionValidationError(err, replayCase.Name)
		}
		candidate, candidateIdentity, err := evalcontract.NormalizeFirstErrorAttribution(replayCase.Candidate)
		if err != nil {
			return FirstErrorAttributionReplayOutput{}, attributionValidationError(err, replayCase.Name)
		}
		comparison, err := evalcontract.CompareFirstErrorAttribution(baseline, candidate)
		if err != nil {
			return FirstErrorAttributionReplayOutput{}, attributionValidationError(err, replayCase.Name)
		}
		drifts := make([]string, len(comparison.Drifts))
		for i, drift := range comparison.Drifts {
			drifts[i] = drift.Reason
		}
		sort.Strings(drifts)
		expectedDrifts := append([]string(nil), replayCase.Expected.Drifts...)
		sort.Strings(expectedDrifts)
		if replayCase.Expected.Error != "" || replayCase.Expected.Passed != comparison.Passed || !equalStrings(expectedDrifts, drifts) {
			return FirstErrorAttributionReplayOutput{}, &ValidationError{
				Code:    evalcontract.ReasonFirstErrorSchemaDrift,
				Message: fmt.Sprintf("case %q expected passed=%t drifts=%v, got passed=%t drifts=%v", replayCase.Name, replayCase.Expected.Passed, expectedDrifts, comparison.Passed, drifts),
			}
		}
		output.Cases = append(output.Cases, FirstErrorAttributionReplayCaseOutput{
			Name:              replayCase.Name,
			Passed:            comparison.Passed,
			BaselineIdentity:  baselineIdentity,
			CandidateIdentity: candidateIdentity,
			NormalizedKind:    candidate.FirstError.Kind,
			NormalizedOwner:   candidate.Cause.Owner,
			Drifts:            drifts,
		})
	}
	return output, nil
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func attributionValidationError(err error, caseName string) *ValidationError {
	code := strings.TrimSpace(err.Error())
	if separator := strings.IndexByte(code, ':'); separator >= 0 {
		code = code[:separator]
	}
	if code == "" {
		code = evalcontract.ReasonFirstErrorSchemaDrift
	}
	return &ValidationError{Code: code, Message: fmt.Sprintf("case %q: %v", caseName, err)}
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err == io.EOF {
		return nil
	} else if err != nil {
		return err
	}
	return fmt.Errorf("unexpected trailing JSON value")
}
