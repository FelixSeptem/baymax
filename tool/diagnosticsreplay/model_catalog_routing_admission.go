package diagnosticsreplay

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/FelixSeptem/baymax/model/catalog"
)

const (
	ModelCatalogRoutingAdmissionFixtureV1 = catalog.RoutingAdmissionVersionV1

	ReasonCodeModelCatalogSchemaDrift         = "model.catalog.audit.schema_drift"
	ReasonCodeModelCatalogSelectionDrift      = "model.catalog.audit.selection_drift"
	ReasonCodeModelCatalogGenerationDrift     = "model.catalog.audit.generation_drift"
	ReasonCodeModelCatalogReasonOrderDrift    = "model.catalog.audit.reason_order_drift"
	ReasonCodeModelCatalogDigestDrift         = "model.catalog.audit.digest_drift"
	ReasonCodeModelCatalogReplayNotIdempotent = "model.catalog.audit.replay_not_idempotent"
)

// ModelCatalogRoutingAdmissionFixture contains only host-supplied normalized
// facts and bounded expected outcomes. It has no provider payloads or secrets.
type ModelCatalogRoutingAdmissionFixture struct {
	Version string                                    `json:"version"`
	Cases   []ModelCatalogRoutingAdmissionFixtureCase `json:"cases"`
}

type ModelCatalogRoutingAdmissionFixtureCase struct {
	CaseID      string                                `json:"case_id"`
	Mode        string                                `json:"mode,omitempty"`
	Catalog     catalog.Input                         `json:"catalog"`
	Input       catalog.RoutingAdmissionInput         `json:"input"`
	Credentials map[string]catalog.CredentialEvidence `json:"credentials,omitempty"`
	Strict      bool                                  `json:"strict,omitempty"`
	Expected    ModelCatalogRoutingAdmissionExpected  `json:"expected"`
}

type ModelCatalogRoutingAdmissionExpected struct {
	CatalogGeneration string             `json:"catalog_generation,omitempty"`
	Status            string             `json:"status"`
	Selected          *catalog.Identity  `json:"selected,omitempty"`
	Fallback          *catalog.Identity  `json:"fallback,omitempty"`
	Reasons           []string           `json:"reasons,omitempty"`
	CanonicalDigest   string             `json:"canonical_digest,omitempty"`
	ResolverActivated bool               `json:"resolver_activated,omitempty"`
	CandidateOrder    []catalog.Identity `json:"candidate_order,omitempty"`
}

type ModelCatalogRoutingAdmissionReplayResult struct {
	Version string                                   `json:"version"`
	Cases   []ModelCatalogRoutingAdmissionReplayCase `json:"cases"`
}

type ModelCatalogRoutingAdmissionReplayCase struct {
	CaseID            string             `json:"case_id"`
	CatalogGeneration string             `json:"catalog_generation"`
	Status            string             `json:"status"`
	Selected          *catalog.Identity  `json:"selected,omitempty"`
	Fallback          *catalog.Identity  `json:"fallback,omitempty"`
	Reasons           []string           `json:"reasons,omitempty"`
	CandidateOrder    []catalog.Identity `json:"candidate_order,omitempty"`
	ResolverActivated bool               `json:"resolver_activated,omitempty"`
	Digest            string             `json:"digest"`
	ReplayDigest      string             `json:"replay_digest"`
	Idempotent        bool               `json:"idempotent"`
}

// ReplayModelCatalogRoutingAdmissionFixtureJSON validates and replays the
// versioned catalog audit fixture without network, clock, filesystem, provider,
// credential-probe, or runtime side effects.
func ReplayModelCatalogRoutingAdmissionFixtureJSON(raw []byte) (ModelCatalogRoutingAdmissionReplayResult, error) {
	var fixture ModelCatalogRoutingAdmissionFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		return ModelCatalogRoutingAdmissionReplayResult{}, &ValidationError{Code: ReasonCodeInvalidJSON, Message: err.Error()}
	}
	if fixture.Version != ModelCatalogRoutingAdmissionFixtureV1 {
		return ModelCatalogRoutingAdmissionReplayResult{}, &ValidationError{Code: ReasonCodeModelCatalogSchemaDrift, Message: "unsupported model catalog routing admission fixture version"}
	}
	if len(fixture.Cases) == 0 {
		return ModelCatalogRoutingAdmissionReplayResult{}, &ValidationError{Code: ReasonCodeModelCatalogSchemaDrift, Message: "fixture cases are required"}
	}
	result := ModelCatalogRoutingAdmissionReplayResult{Version: fixture.Version, Cases: make([]ModelCatalogRoutingAdmissionReplayCase, 0, len(fixture.Cases))}
	for index := range fixture.Cases {
		first, err := replayModelCatalogRoutingAdmissionCase(fixture.Cases[index])
		if err != nil {
			return ModelCatalogRoutingAdmissionReplayResult{}, err
		}
		second, err := replayModelCatalogRoutingAdmissionCase(fixture.Cases[index])
		if err != nil {
			return ModelCatalogRoutingAdmissionReplayResult{}, err
		}
		if first.Digest != second.Digest {
			return ModelCatalogRoutingAdmissionReplayResult{}, &ValidationError{Code: ReasonCodeModelCatalogReplayNotIdempotent, Message: fixture.Cases[index].CaseID}
		}
		first.ReplayDigest = second.Digest
		first.Idempotent = first.Digest == first.ReplayDigest
		result.Cases = append(result.Cases, first)
	}
	return result, nil
}

func replayModelCatalogRoutingAdmissionCase(input ModelCatalogRoutingAdmissionFixtureCase) (ModelCatalogRoutingAdmissionReplayCase, error) {
	caseID := strings.TrimSpace(input.CaseID)
	if caseID == "" {
		return ModelCatalogRoutingAdmissionReplayCase{}, &ValidationError{Code: ReasonCodeModelCatalogSchemaDrift, Message: "case_id is required"}
	}
	catalogSnapshot, err := catalog.New(input.Catalog)
	if err != nil {
		return ModelCatalogRoutingAdmissionReplayCase{}, &ValidationError{Code: catalog.ErrorCode(err), Message: err.Error()}
	}
	mode := strings.ToLower(strings.TrimSpace(input.Mode))
	var result catalog.RoutingAdmissionResult
	switch mode {
	case "", "audit":
		result, err = catalog.AuditRoutingAdmission(catalogSnapshot, input.Input, input.Credentials, input.Strict)
	case "resolver":
		result, err = catalog.ResolveRoutingAdmission(catalogSnapshot, input.Input, input.Credentials, input.Strict)
	default:
		return ModelCatalogRoutingAdmissionReplayCase{}, &ValidationError{Code: ReasonCodeModelCatalogSchemaDrift, Message: fmt.Sprintf("unsupported mode %q", input.Mode)}
	}
	if err != nil {
		return ModelCatalogRoutingAdmissionReplayCase{}, &ValidationError{Code: catalog.ErrorCode(err), Message: err.Error()}
	}
	if err := compareModelCatalogRoutingAdmissionExpected(result, input.Expected); err != nil {
		return ModelCatalogRoutingAdmissionReplayCase{}, err
	}
	output := ModelCatalogRoutingAdmissionReplayCase{
		CaseID:            caseID,
		CatalogGeneration: result.CatalogGeneration,
		Status:            result.Status,
		Selected:          result.Selected,
		Fallback:          result.Fallback,
		Reasons:           append([]string(nil), result.Reasons...),
		CandidateOrder:    append([]catalog.Identity(nil), result.CandidateOrder...),
		ResolverActivated: result.ResolverActivated,
	}
	digestBytes, err := json.Marshal(output)
	if err != nil {
		return ModelCatalogRoutingAdmissionReplayCase{}, &ValidationError{Code: ReasonCodeModelCatalogSchemaDrift, Message: err.Error()}
	}
	digest := sha256.Sum256(digestBytes)
	output.Digest = hex.EncodeToString(digest[:])
	return output, nil
}

func compareModelCatalogRoutingAdmissionExpected(result catalog.RoutingAdmissionResult, expected ModelCatalogRoutingAdmissionExpected) error {
	if expected.CatalogGeneration != "" && result.CatalogGeneration != expected.CatalogGeneration {
		return &ValidationError{Code: ReasonCodeModelCatalogGenerationDrift, Message: "catalog generation mismatch"}
	}
	if expected.Status != "" && result.Status != expected.Status {
		return &ValidationError{Code: ReasonCodeModelCatalogSelectionDrift, Message: "status mismatch"}
	}
	if expected.Selected != nil && !reflect.DeepEqual(result.Selected, expected.Selected) {
		return &ValidationError{Code: ReasonCodeModelCatalogSelectionDrift, Message: "selected identity mismatch"}
	}
	if expected.Fallback != nil && !reflect.DeepEqual(result.Fallback, expected.Fallback) {
		return &ValidationError{Code: ReasonCodeModelCatalogSelectionDrift, Message: "fallback identity mismatch"}
	}
	if expected.Reasons != nil && !reflect.DeepEqual(result.Reasons, expected.Reasons) {
		return &ValidationError{Code: ReasonCodeModelCatalogReasonOrderDrift, Message: "reason order mismatch"}
	}
	if expected.CanonicalDigest != "" && result.CanonicalDigest != expected.CanonicalDigest {
		return &ValidationError{Code: ReasonCodeModelCatalogDigestDrift, Message: "canonical digest mismatch"}
	}
	if expected.ResolverActivated != result.ResolverActivated {
		return &ValidationError{Code: ReasonCodeModelCatalogSelectionDrift, Message: "resolver activation mismatch"}
	}
	if expected.CandidateOrder != nil && !reflect.DeepEqual(result.CandidateOrder, expected.CandidateOrder) {
		return &ValidationError{Code: ReasonCodeModelCatalogSelectionDrift, Message: "candidate order mismatch"}
	}
	return nil
}
