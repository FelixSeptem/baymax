package diagnosticsreplay

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/FelixSeptem/baymax/model/conformance"
)

const ProviderContextCacheEvidenceFixtureV1 = conformance.FixtureVersionProviderContextCacheEvidenceV1

const (
	ReasonCodeContextCacheEvidenceSchemaDrift       = conformance.ReasonContextCacheEvidenceSchemaDrift
	ReasonCodeContextCacheEvidenceUnknownVersion    = conformance.ReasonContextCacheEvidenceUnknownVersion
	ReasonCodeContextCacheEvidenceReferenceDrift    = conformance.ReasonContextCacheEvidenceReferenceDrift
	ReasonCodeContextCacheEvidencePrivacyDrift      = conformance.ReasonContextCacheEvidencePrivacyDrift
	ReasonCodeContextCacheEvidenceOverflowDrift     = conformance.ReasonContextCacheEvidenceOverflowDrift
	ReasonCodeContextCacheEvidenceDuplicateConflict = conformance.ReasonContextCacheEvidenceDuplicateConflict
	ReasonCodeContextCacheEvidenceRoleDrift         = conformance.ReasonContextCacheEvidenceRoleDrift
	ReasonCodeContextCacheEvidenceSkillTailDrift    = conformance.ReasonContextCacheEvidenceSkillTailDrift
	ReasonCodeContextCacheEvidenceToolResultDrift   = conformance.ReasonContextCacheEvidenceToolResultDrift
	ReasonCodeContextCacheEvidenceOrderingDrift     = conformance.ReasonContextCacheEvidenceOrderingDrift
	ReasonCodeContextCacheEvidenceCacheParityDrift  = conformance.ReasonContextCacheEvidenceCacheParityDrift
	ReasonCodeContextCacheEvidenceCostDrift         = conformance.ReasonContextCacheEvidenceCostDrift
	ReasonCodeContextCacheEvidenceMissing           = conformance.ReasonContextCacheEvidenceMissing
	ReasonCodeContextCacheEvidenceVerdictDrift      = conformance.ReasonContextCacheEvidenceVerdictDrift
)

type ProviderContextCacheEvidenceReplayCase struct {
	CaseID         string   `json:"case_id"`
	Provider       string   `json:"provider"`
	ProjectionCase string   `json:"projection_case"`
	Verdict        string   `json:"verdict"`
	Reasons        []string `json:"reasons,omitempty"`
	Digest         string   `json:"digest"`
	ReplayDigest   string   `json:"replay_digest"`
	Idempotent     bool     `json:"idempotent"`
}

type ProviderContextCacheEvidenceReplayResult struct {
	Version string                                   `json:"version"`
	Cases   []ProviderContextCacheEvidenceReplayCase `json:"cases"`
}

// ReplayProviderContextCacheEvidenceJSON validates and replays evidence using
// syntactic projection references. It remains offline and does not load or
// execute any provider fixture.
func ReplayProviderContextCacheEvidenceJSON(raw []byte) (ProviderContextCacheEvidenceReplayResult, error) {
	return replayProviderContextCacheEvidenceJSON(raw, nil)
}

// ReplayProviderContextCacheEvidenceJSONWithProjectionDigests additionally
// resolves references against a caller-owned map keyed by provider and case
// identity. The map contains only canonical digests, never raw payloads.
func ReplayProviderContextCacheEvidenceJSONWithProjectionDigests(raw []byte, projectionDigests map[string]string) (ProviderContextCacheEvidenceReplayResult, error) {
	return replayProviderContextCacheEvidenceJSON(raw, projectionDigests)
}

func replayProviderContextCacheEvidenceJSON(raw []byte, projectionDigests map[string]string) (ProviderContextCacheEvidenceReplayResult, error) {
	if len(raw) > 1<<20 {
		return ProviderContextCacheEvidenceReplayResult{}, &ValidationError{Code: ReasonCodeContextCacheEvidenceOverflowDrift, Message: "fixture exceeds bound"}
	}
	var fixture conformance.ProviderContextCacheEvidenceFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		return ProviderContextCacheEvidenceReplayResult{}, &ValidationError{Code: ReasonCodeContextCacheEvidenceSchemaDrift, Message: err.Error()}
	}
	if err := conformance.ValidateProviderContextCacheEvidenceFixture(fixture); err != nil {
		return ProviderContextCacheEvidenceReplayResult{}, classifyProviderContextCacheEvidenceError(err)
	}
	result := ProviderContextCacheEvidenceReplayResult{Version: fixture.Version, Cases: make([]ProviderContextCacheEvidenceReplayCase, 0, len(fixture.Cases))}
	for _, input := range fixture.Cases {
		if projectionDigests != nil {
			key := strings.TrimSpace(input.Reference.Provider) + "\x00" + strings.TrimSpace(input.Reference.ProjectionCase)
			if expected, ok := projectionDigests[key]; !ok || !strings.EqualFold(strings.TrimSpace(expected), strings.TrimSpace(input.Reference.ProjectionDigest)) {
				return ProviderContextCacheEvidenceReplayResult{}, &ValidationError{Code: ReasonCodeContextCacheEvidenceReferenceDrift, Message: input.CaseID + ": projection reference digest mismatch"}
			}
		}
		first, err := replayProviderContextCacheEvidenceCase(input)
		if err != nil {
			return ProviderContextCacheEvidenceReplayResult{}, err
		}
		second, err := replayProviderContextCacheEvidenceCase(input)
		if err != nil {
			return ProviderContextCacheEvidenceReplayResult{}, err
		}
		if first.Digest != second.Digest || first.Verdict != second.Verdict || !sameStringList(first.Reasons, second.Reasons) {
			return ProviderContextCacheEvidenceReplayResult{}, &ValidationError{Code: ReasonCodeContextCacheEvidenceVerdictDrift, Message: input.CaseID + ": replay is not deterministic"}
		}
		if input.Digest != "" && input.Digest != first.Digest || input.ReplayDigest != "" && input.ReplayDigest != first.Digest {
			return ProviderContextCacheEvidenceReplayResult{}, &ValidationError{Code: ReasonCodeContextCacheEvidenceVerdictDrift, Message: input.CaseID + ": recorded digest mismatch"}
		}
		if input.Verdict != "" && input.Verdict != first.Verdict {
			return ProviderContextCacheEvidenceReplayResult{}, &ValidationError{Code: ReasonCodeContextCacheEvidenceVerdictDrift, Message: input.CaseID + ": recorded verdict mismatch"}
		}
		result.Cases = append(result.Cases, first)
	}
	return result, nil
}

func replayProviderContextCacheEvidenceCase(input conformance.ProviderContextCacheEvidenceCase) (ProviderContextCacheEvidenceReplayCase, error) {
	verdict, reasons, err := conformance.DeriveProviderContextCacheEvidenceVerdict(input)
	if err != nil {
		return ProviderContextCacheEvidenceReplayCase{}, classifyProviderContextCacheEvidenceError(err)
	}
	digest, err := conformance.ProviderContextCacheEvidenceDigest(input)
	if err != nil {
		return ProviderContextCacheEvidenceReplayCase{}, classifyProviderContextCacheEvidenceError(err)
	}
	return ProviderContextCacheEvidenceReplayCase{
		CaseID: input.CaseID, Provider: input.Reference.Provider, ProjectionCase: input.Reference.ProjectionCase,
		Verdict: verdict, Reasons: reasons, Digest: digest, ReplayDigest: digest, Idempotent: true,
	}, nil
}

func classifyProviderContextCacheEvidenceError(err error) error {
	var classified *conformance.ProviderContextCacheEvidenceError
	if errors.As(err, &classified) {
		return &ValidationError{Code: classified.Code, Message: classified.Message}
	}
	return &ValidationError{Code: ReasonCodeContextCacheEvidenceSchemaDrift, Message: err.Error()}
}

func sameStringList(left, right []string) bool {
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
