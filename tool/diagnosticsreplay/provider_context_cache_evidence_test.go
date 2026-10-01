package diagnosticsreplay

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/FelixSeptem/baymax/model/conformance"
)

func admissionEvidenceFixture(t *testing.T) ([]byte, conformance.ProviderContextCacheEvidenceFixture) {
	t.Helper()
	fixture := conformance.ProviderContextCacheEvidenceFixture{
		Version: conformance.FixtureVersionProviderContextCacheEvidenceV1,
		Cases: []conformance.ProviderContextCacheEvidenceCase{
			{
				CaseID: "openai-no-drift", Reference: conformance.ProviderContextCacheEvidenceReference{
					Provider: "openai", SDKVersion: "v1", ProjectionCase: "openai-run", ProjectionDigest: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				},
				RequiredDimensions: []string{conformance.ContextCacheEvidenceDimensionRole, conformance.ContextCacheEvidenceDimensionCacheParity},
				Dimensions: []conformance.ProviderContextCacheEvidenceDimension{
					{Name: conformance.ContextCacheEvidenceDimensionRole, Status: conformance.ContextCacheEvidenceStatusVerifiedNoDrift},
					{Name: conformance.ContextCacheEvidenceDimensionCacheParity, Status: conformance.ContextCacheEvidenceStatusVerifiedNoDrift},
				},
			},
			{
				CaseID: "anthropic-insufficient", Reference: conformance.ProviderContextCacheEvidenceReference{
					Provider: "anthropic", SDKVersion: "v1", ProjectionCase: "anthropic-run", ProjectionDigest: "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
				},
				RequiredDimensions: []string{conformance.ContextCacheEvidenceDimensionHostCostP95},
				Dimensions:         []conformance.ProviderContextCacheEvidenceDimension{{Name: conformance.ContextCacheEvidenceDimensionHostCostP95, Status: conformance.ContextCacheEvidenceStatusUnavailable}},
			},
			{
				CaseID: "gemini-drift", Reference: conformance.ProviderContextCacheEvidenceReference{
					Provider: "gemini", SDKVersion: "v1", ProjectionCase: "gemini-stream", ProjectionDigest: "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210",
				},
				RequiredDimensions: []string{conformance.ContextCacheEvidenceDimensionToolResult},
				Dimensions:         []conformance.ProviderContextCacheEvidenceDimension{{Name: conformance.ContextCacheEvidenceDimensionToolResult, Status: conformance.ContextCacheEvidenceStatusDriftConfirmed}},
			},
		},
	}
	raw, err := json.Marshal(fixture)
	if err != nil {
		t.Fatal(err)
	}
	return raw, fixture
}

func TestReplayProviderContextCacheEvidenceIsDeterministicAndThreeState(t *testing.T) {
	raw, fixture := admissionEvidenceFixture(t)
	digests := map[string]string{}
	for _, item := range fixture.Cases {
		key := item.Reference.Provider + "\x00" + item.Reference.ProjectionCase
		digests[key] = item.Reference.ProjectionDigest
	}
	first, err := ReplayProviderContextCacheEvidenceJSONWithProjectionDigests(raw, digests)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ReplayProviderContextCacheEvidenceJSONWithProjectionDigests(raw, digests)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Cases) != 3 || len(second.Cases) != 3 {
		t.Fatalf("case count = %d/%d, want 3", len(first.Cases), len(second.Cases))
	}
	if first.Cases[0].Verdict != conformance.ContextCacheEvidenceVerdictNoDrift || first.Cases[1].Verdict != conformance.ContextCacheEvidenceVerdictInsufficientEvidence || first.Cases[2].Verdict != conformance.ContextCacheEvidenceVerdictDriftConfirmed {
		t.Fatalf("unexpected verdicts: %#v", first.Cases)
	}
	for i := range first.Cases {
		if first.Cases[i].Digest == "" || first.Cases[i].Digest != first.Cases[i].ReplayDigest || first.Cases[i].Digest != second.Cases[i].Digest {
			t.Fatalf("case %d is not idempotent: %#v / %#v", i, first.Cases[i], second.Cases[i])
		}
	}
}

func TestReplayProviderContextCacheEvidenceRejectsReferenceMismatchAndRecordedDrift(t *testing.T) {
	raw, fixture := admissionEvidenceFixture(t)
	digests := map[string]string{"openai\x00openai-run": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"}
	if _, err := ReplayProviderContextCacheEvidenceJSONWithProjectionDigests(raw, digests); err == nil || !hasAdmissionCode(err, ReasonCodeContextCacheEvidenceReferenceDrift) {
		t.Fatalf("reference mismatch error = %v", err)
	}
	fixture.Cases[0].Verdict = conformance.ContextCacheEvidenceVerdictDriftConfirmed
	mutated, err := json.Marshal(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ReplayProviderContextCacheEvidenceJSON(mutated); err == nil || !hasAdmissionCode(err, ReasonCodeContextCacheEvidenceVerdictDrift) {
		t.Fatalf("recorded verdict mismatch error = %v", err)
	}
}

func TestReplayProviderContextCacheEvidenceParityAndDuplicateDimensions(t *testing.T) {
	base := conformance.ProviderContextCacheEvidenceCase{
		CaseID: "parity", Reference: conformance.ProviderContextCacheEvidenceReference{
			Provider: "openai", SDKVersion: "v1", ProjectionCase: "parity", ProjectionDigest: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		},
		RequiredDimensions: []string{conformance.ContextCacheEvidenceDimensionCacheParity, conformance.ContextCacheEvidenceDimensionRole},
		Dimensions: []conformance.ProviderContextCacheEvidenceDimension{
			{Name: conformance.ContextCacheEvidenceDimensionCacheParity, Status: conformance.ContextCacheEvidenceStatusVerifiedNoDrift},
			{Name: conformance.ContextCacheEvidenceDimensionRole, Status: conformance.ContextCacheEvidenceStatusVerifiedNoDrift},
		},
	}
	first, err := json.Marshal(conformance.ProviderContextCacheEvidenceFixture{Version: conformance.FixtureVersionProviderContextCacheEvidenceV1, Cases: []conformance.ProviderContextCacheEvidenceCase{base}})
	if err != nil {
		t.Fatal(err)
	}
	base.RequiredDimensions[0], base.RequiredDimensions[1] = base.RequiredDimensions[1], base.RequiredDimensions[0]
	base.Dimensions[0], base.Dimensions[1] = base.Dimensions[1], base.Dimensions[0]
	second, err := json.Marshal(conformance.ProviderContextCacheEvidenceFixture{Version: conformance.FixtureVersionProviderContextCacheEvidenceV1, Cases: []conformance.ProviderContextCacheEvidenceCase{base}})
	if err != nil {
		t.Fatal(err)
	}
	left, err := ReplayProviderContextCacheEvidenceJSON(first)
	if err != nil {
		t.Fatal(err)
	}
	right, err := ReplayProviderContextCacheEvidenceJSON(second)
	if err != nil {
		t.Fatal(err)
	}
	if left.Cases[0].Digest != right.Cases[0].Digest || left.Cases[0].Verdict != conformance.ContextCacheEvidenceVerdictNoDrift {
		t.Fatalf("parity evidence changed under ordering: %#v vs %#v", left.Cases[0], right.Cases[0])
	}
	base.Dimensions = append(base.Dimensions, base.Dimensions[0])
	duplicate, err := json.Marshal(conformance.ProviderContextCacheEvidenceFixture{Version: conformance.FixtureVersionProviderContextCacheEvidenceV1, Cases: []conformance.ProviderContextCacheEvidenceCase{base}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ReplayProviderContextCacheEvidenceJSON(duplicate); err == nil || !hasAdmissionCode(err, ReasonCodeContextCacheEvidenceDuplicateConflict) {
		t.Fatalf("duplicate dimension error = %v", err)
	}
}

func hasAdmissionCode(err error, code string) bool {
	var validation *ValidationError
	return errors.As(err, &validation) && validation.Code == code
}
