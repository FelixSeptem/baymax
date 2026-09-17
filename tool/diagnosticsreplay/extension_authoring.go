package diagnosticsreplay

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/FelixSeptem/baymax/extension"
)

const ReasonCodeAuthoringExpectedActualDrift = "authoring_expected_actual_drift"

// EvaluateExternalExtensionAuthoringFixture evaluates one offline fixture and
// returns canonical JSON without mutating the input or writing diagnostics.
func EvaluateExternalExtensionAuthoringFixture(raw []byte) ([]byte, error) {
	var envelope struct {
		Expected extension.AuthoringResult `json:"expected"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, &ValidationError{Code: ReasonCodeInvalidJSON, Message: err.Error()}
	}
	actual, err := extension.EvaluateAuthoringCase(raw)
	if err != nil {
		return nil, &ValidationError{Code: ReasonCodeInvalidJSONShape, Message: err.Error()}
	}
	canonicalActual, err := json.Marshal(actual)
	if err != nil {
		return nil, err
	}
	canonicalExpected, err := json.Marshal(envelope.Expected)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(canonicalActual, canonicalExpected) {
		return nil, &ValidationError{Code: ReasonCodeAuthoringExpectedActualDrift, Message: fmt.Sprintf("expected=%s actual=%s", canonicalExpected, canonicalActual)}
	}
	return canonicalActual, nil
}

// ReplayReasonCode exposes a deterministic machine-readable replay code.
func ReplayReasonCode(err error) string {
	if validation, ok := err.(*ValidationError); ok {
		return validation.Code
	}
	return ""
}
