package diagnosticsreplay

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strings"

	"github.com/FelixSeptem/baymax/model/catalog"
)

const (
	ModelRouteIntentAdmissionFixtureV1 = catalog.RouteIntentAdmissionVersionV1

	ReasonCodeModelRouteIntentSchemaDrift         = "model.route_intent.schema_drift"
	ReasonCodeModelRouteIntentVerdictDrift        = "model.route_intent.verdict_drift"
	ReasonCodeModelRouteIntentGenerationDrift     = "model.route_intent.generation_drift"
	ReasonCodeModelRouteIntentReasonOrderDrift    = "model.route_intent.reason_order_drift"
	ReasonCodeModelRouteIntentParityDrift         = "model.route_intent.parity_drift"
	ReasonCodeModelRouteIntentDigestDrift         = "model.route_intent.digest_drift"
	ReasonCodeModelRouteIntentPrivacyViolation    = "model.route_intent.privacy_violation"
	ReasonCodeModelRouteIntentReplayNotIdempotent = "model.route_intent.replay_not_idempotent"
)

type ModelRouteIntentAdmissionFixture struct {
	Version string                                 `json:"version"`
	Cases   []ModelRouteIntentAdmissionFixtureCase `json:"cases"`
}

type ModelRouteIntentAdmissionFixtureCase struct {
	CaseID   string                            `json:"case_id"`
	Input    catalog.RouteIntentAdmissionInput `json:"input"`
	Expected ModelRouteIntentAdmissionExpected `json:"expected"`
}

type ModelRouteIntentAdmissionExpected struct {
	Verdict           string            `json:"verdict"`
	CatalogGeneration string            `json:"catalog_generation,omitempty"`
	Selected          *catalog.Identity `json:"selected,omitempty"`
	Fallback          *catalog.Identity `json:"fallback,omitempty"`
	Reasons           []string          `json:"reasons,omitempty"`
	RunStreamParity   bool              `json:"run_stream_parity,omitempty"`
	CanonicalDigest   string            `json:"canonical_digest,omitempty"`
}

type ModelRouteIntentAdmissionReplayResult struct {
	Version string                                `json:"version"`
	Cases   []ModelRouteIntentAdmissionReplayCase `json:"cases"`
}

type ModelRouteIntentAdmissionReplayCase struct {
	CaseID            string            `json:"case_id"`
	Verdict           string            `json:"verdict"`
	CatalogGeneration string            `json:"catalog_generation,omitempty"`
	Selected          *catalog.Identity `json:"selected,omitempty"`
	Fallback          *catalog.Identity `json:"fallback,omitempty"`
	Reasons           []string          `json:"reasons,omitempty"`
	RunStreamParity   bool              `json:"run_stream_parity,omitempty"`
	Digest            string            `json:"digest"`
	ReplayDigest      string            `json:"replay_digest"`
	Idempotent        bool              `json:"idempotent"`
}

// ReplayModelRouteIntentAdmissionFixtureJSON validates and replays an offline
// route-intent evidence fixture. Unknown JSON fields are intentionally ignored
// for additive compatibility.
func ReplayModelRouteIntentAdmissionFixtureJSON(raw []byte) (ModelRouteIntentAdmissionReplayResult, error) {
	var fixture ModelRouteIntentAdmissionFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		return ModelRouteIntentAdmissionReplayResult{}, &ValidationError{Code: ReasonCodeInvalidJSON, Message: err.Error()}
	}
	if fixture.Version != ModelRouteIntentAdmissionFixtureV1 {
		return ModelRouteIntentAdmissionReplayResult{}, &ValidationError{Code: ReasonCodeModelRouteIntentSchemaDrift, Message: "unsupported route-intent fixture version"}
	}
	if len(fixture.Cases) == 0 {
		return ModelRouteIntentAdmissionReplayResult{}, &ValidationError{Code: ReasonCodeModelRouteIntentSchemaDrift, Message: "fixture cases are required"}
	}
	result := ModelRouteIntentAdmissionReplayResult{Version: fixture.Version, Cases: make([]ModelRouteIntentAdmissionReplayCase, 0, len(fixture.Cases))}
	for _, item := range fixture.Cases {
		first, err := replayModelRouteIntentAdmissionCase(item)
		if err != nil {
			return ModelRouteIntentAdmissionReplayResult{}, err
		}
		second, err := replayModelRouteIntentAdmissionCase(item)
		if err != nil {
			return ModelRouteIntentAdmissionReplayResult{}, err
		}
		if first.Digest != second.Digest {
			return ModelRouteIntentAdmissionReplayResult{}, &ValidationError{Code: ReasonCodeModelRouteIntentReplayNotIdempotent, Message: item.CaseID}
		}
		first.ReplayDigest = second.Digest
		first.Idempotent = true
		result.Cases = append(result.Cases, first)
	}
	return result, nil
}

func replayModelRouteIntentAdmissionCase(input ModelRouteIntentAdmissionFixtureCase) (ModelRouteIntentAdmissionReplayCase, error) {
	caseID := strings.TrimSpace(input.CaseID)
	if caseID == "" {
		return ModelRouteIntentAdmissionReplayCase{}, &ValidationError{Code: ReasonCodeModelRouteIntentSchemaDrift, Message: "case_id is required"}
	}
	result, err := catalog.CompareRouteIntentAdmission(input.Input)
	if err != nil {
		code := catalog.ErrorCode(err)
		if code == catalog.ReasonRouteIntentPrivacyViolation {
			code = ReasonCodeModelRouteIntentPrivacyViolation
		}
		if code == "" {
			code = ReasonCodeModelRouteIntentSchemaDrift
		}
		return ModelRouteIntentAdmissionReplayCase{}, &ValidationError{Code: code, Message: err.Error()}
	}
	if err := compareModelRouteIntentExpected(result, input.Expected); err != nil {
		return ModelRouteIntentAdmissionReplayCase{}, err
	}
	output := ModelRouteIntentAdmissionReplayCase{CaseID: caseID, Verdict: result.Verdict, CatalogGeneration: result.CatalogGeneration, Selected: result.Selected, Fallback: result.Fallback, Reasons: append([]string(nil), result.Reasons...), RunStreamParity: result.RunStreamParity}
	encoded, err := json.Marshal(output)
	if err != nil {
		return ModelRouteIntentAdmissionReplayCase{}, &ValidationError{Code: ReasonCodeModelRouteIntentSchemaDrift, Message: err.Error()}
	}
	digest := sha256.Sum256(encoded)
	output.Digest = hex.EncodeToString(digest[:])
	return output, nil
}

func compareModelRouteIntentExpected(result catalog.RouteIntentAdmissionResult, expected ModelRouteIntentAdmissionExpected) error {
	if expected.Verdict != "" && result.Verdict != expected.Verdict {
		return &ValidationError{Code: ReasonCodeModelRouteIntentVerdictDrift, Message: "verdict mismatch"}
	}
	if expected.CatalogGeneration != "" && result.CatalogGeneration != expected.CatalogGeneration {
		return &ValidationError{Code: ReasonCodeModelRouteIntentGenerationDrift, Message: "catalog generation mismatch"}
	}
	if expected.Selected != nil && !reflect.DeepEqual(result.Selected, expected.Selected) {
		return &ValidationError{Code: ReasonCodeModelRouteIntentVerdictDrift, Message: "selected identity mismatch"}
	}
	if expected.Fallback != nil && !reflect.DeepEqual(result.Fallback, expected.Fallback) {
		return &ValidationError{Code: ReasonCodeModelRouteIntentVerdictDrift, Message: "fallback identity mismatch"}
	}
	if expected.Reasons != nil && !reflect.DeepEqual(result.Reasons, expected.Reasons) {
		return &ValidationError{Code: ReasonCodeModelRouteIntentReasonOrderDrift, Message: "reason order mismatch"}
	}
	if expected.RunStreamParity != result.RunStreamParity {
		return &ValidationError{Code: ReasonCodeModelRouteIntentParityDrift, Message: "run/stream parity mismatch"}
	}
	if expected.CanonicalDigest != "" && expected.CanonicalDigest != result.CanonicalDigest {
		return &ValidationError{Code: ReasonCodeModelRouteIntentDigestDrift, Message: "canonical digest mismatch"}
	}
	return nil
}
