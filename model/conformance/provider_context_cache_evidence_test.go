package conformance

import (
	"errors"
	"testing"
)

func providerEvidenceReference() ProviderContextCacheEvidenceReference {
	return ProviderContextCacheEvidenceReference{
		Provider:         "openai",
		SDKVersion:       "v1",
		ProjectionCase:   "case-1",
		ProjectionDigest: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}
}

func TestProviderContextCacheEvidenceCanonicalDigestIsDeterministic(t *testing.T) {
	input := ProviderContextCacheEvidenceCase{
		CaseID:             "case-1",
		Reference:          providerEvidenceReference(),
		RequiredDimensions: []string{ContextCacheEvidenceDimensionOrdering, ContextCacheEvidenceDimensionRole},
		Dimensions: []ProviderContextCacheEvidenceDimension{
			{Name: ContextCacheEvidenceDimensionRole, Status: ContextCacheEvidenceStatusVerifiedNoDrift},
			{Name: ContextCacheEvidenceDimensionOrdering, Status: ContextCacheEvidenceStatusVerifiedNoDrift},
		},
	}
	first, err := ProviderContextCacheEvidenceDigest(input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ProviderContextCacheEvidenceDigest(input)
	if err != nil {
		t.Fatal(err)
	}
	if first == "" || first != second {
		t.Fatalf("digest = %q/%q, want stable non-empty digest", first, second)
	}
	reversed := input
	if len(reversed.RequiredDimensions) != 2 {
		t.Fatal("test setup unexpectedly changed")
	}
	reversed.RequiredDimensions = []string{ContextCacheEvidenceDimensionRole, ContextCacheEvidenceDimensionOrdering}
	reversed.Dimensions[0], reversed.Dimensions[1] = reversed.Dimensions[1], reversed.Dimensions[0]
	third, err := ProviderContextCacheEvidenceDigest(reversed)
	if err != nil {
		t.Fatal(err)
	}
	if first != third {
		t.Fatalf("reordered evidence digest = %q, want %q", third, first)
	}
}

func TestProviderContextCacheEvidenceRejectsUnknownVersionDuplicateAndPrivacy(t *testing.T) {
	valid := ProviderContextCacheEvidenceFixture{Version: FixtureVersionProviderContextCacheEvidenceV1, Cases: []ProviderContextCacheEvidenceCase{{
		CaseID: "case-1", Reference: providerEvidenceReference(),
		RequiredDimensions: []string{ContextCacheEvidenceDimensionRole},
		Dimensions:         []ProviderContextCacheEvidenceDimension{{Name: ContextCacheEvidenceDimensionRole, Status: ContextCacheEvidenceStatusVerifiedNoDrift}},
	}}}
	if err := ValidateProviderContextCacheEvidenceFixture(valid); err != nil {
		t.Fatal(err)
	}
	unknown := valid
	unknown.Version = "provider_context_cache_evidence.v2"
	assertProviderEvidenceCode(t, ValidateProviderContextCacheEvidenceFixture(unknown), ReasonContextCacheEvidenceUnknownVersion)
	duplicate := valid
	duplicate.Cases = append(duplicate.Cases, duplicate.Cases[0])
	assertProviderEvidenceCode(t, ValidateProviderContextCacheEvidenceFixture(duplicate), ReasonContextCacheEvidenceDuplicateConflict)
	privacy := valid
	privacy.Cases = append([]ProviderContextCacheEvidenceCase(nil), valid.Cases...)
	privacy.Cases[0].Reference.SDKVersion = "bearer token"
	assertProviderEvidenceCode(t, ValidateProviderContextCacheEvidenceFixture(privacy), ReasonContextCacheEvidencePrivacyDrift)
}

func TestProviderContextCacheEvidenceVerdictPrioritizesDriftThenInsufficient(t *testing.T) {
	base := ProviderContextCacheEvidenceCase{
		CaseID: "case-1", Reference: providerEvidenceReference(),
		RequiredDimensions: []string{ContextCacheEvidenceDimensionRole, ContextCacheEvidenceDimensionHostCostP95},
		Dimensions: []ProviderContextCacheEvidenceDimension{
			{Name: ContextCacheEvidenceDimensionRole, Status: ContextCacheEvidenceStatusVerifiedNoDrift},
		},
	}
	verdict, reasons, err := DeriveProviderContextCacheEvidenceVerdict(base)
	if err != nil {
		t.Fatal(err)
	}
	if verdict != ContextCacheEvidenceVerdictInsufficientEvidence || !containsString(reasons, ReasonContextCacheEvidenceMissing) {
		t.Fatalf("verdict = %q reasons = %#v, want insufficient evidence", verdict, reasons)
	}
	base.Dimensions[0].Status = ContextCacheEvidenceStatusDriftConfirmed
	verdict, reasons, err = DeriveProviderContextCacheEvidenceVerdict(base)
	if err != nil {
		t.Fatal(err)
	}
	if verdict != ContextCacheEvidenceVerdictDriftConfirmed || !containsString(reasons, ReasonContextCacheEvidenceRoleDrift) {
		t.Fatalf("verdict = %q reasons = %#v, want confirmed role drift", verdict, reasons)
	}
}

func TestProviderContextCacheEvidencePerformanceBoundaries(t *testing.T) {
	valid := ProviderContextCacheEvidenceCase{
		CaseID: "case-1", Reference: providerEvidenceReference(),
		RequiredDimensions: []string{ContextCacheEvidenceDimensionHostCostP95},
		Dimensions:         []ProviderContextCacheEvidenceDimension{{Name: ContextCacheEvidenceDimensionHostCostP95, Status: ContextCacheEvidenceStatusVerifiedNoDrift}},
		Performance: &ProviderContextCachePerformanceSummary{
			Metric: "cache_cost", Unit: "tokens", SampleCount: 10,
			BaselineP50: 10, BaselineP95: 20, ObservedP50: 11, ObservedP95: 30, ThresholdP95: 5,
		},
	}
	if err := ValidateProviderContextCacheEvidenceCase(valid); err != nil {
		t.Fatal(err)
	}
	verdict, reasons, err := DeriveProviderContextCacheEvidenceVerdict(valid)
	if err != nil {
		t.Fatal(err)
	}
	if verdict != ContextCacheEvidenceVerdictDriftConfirmed || !containsString(reasons, ReasonContextCacheEvidenceCostDrift) {
		t.Fatalf("verdict = %q reasons = %#v, want cost drift", verdict, reasons)
	}
	invalid := valid
	invalid.Performance = &ProviderContextCachePerformanceSummary{Metric: "cache_cost", Unit: "tokens", SampleCount: 1}
	assertProviderEvidenceCode(t, ValidateProviderContextCacheEvidenceCase(invalid), ReasonContextCacheEvidenceCostDrift)
	mixed := valid
	mixed.Performance = &ProviderContextCachePerformanceSummary{Metric: "cache_cost", Unit: "seconds", SampleCount: 10, BaselineP50: 10, BaselineP95: 20, ObservedP50: 11, ObservedP95: 30, ThresholdP95: 5}
	if err := ValidateProviderContextCacheEvidenceCase(mixed); err != nil {
		t.Fatalf("unit is explicit and valid at schema level: %v", err)
	}
}

func assertProviderEvidenceCode(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error %s", code)
	}
	var classified *ProviderContextCacheEvidenceError
	if !errors.As(err, &classified) || classified.Code != code {
		t.Fatalf("error = %v, want code %s", err, code)
	}
}
