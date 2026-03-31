package adapterconformance

import (
	"errors"
	"reflect"
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
	memorypkg "github.com/FelixSeptem/baymax/memory"
)

func TestMemoryAdapterConformanceMainstreamProfileMatrix(t *testing.T) {
	matrix := MainstreamMemoryProfileMatrix()
	gotProfiles := make([]string, 0, len(matrix))
	for _, row := range matrix {
		gotProfiles = append(gotProfiles, row.ProfileID)
		if !row.OfflineDeterministic {
			t.Fatalf("profile %q should be offline deterministic", row.ProfileID)
		}
		if row.ContractVersion != memorypkg.ContractVersionMemoryV1 {
			t.Fatalf("profile %q contract version mismatch: %q", row.ProfileID, row.ContractVersion)
		}
	}
	wantProfiles := []string{
		memorypkg.ProfileMem0,
		memorypkg.ProfileZep,
		memorypkg.ProfileOpenViking,
		memorypkg.ProfileGeneric,
	}
	if !reflect.DeepEqual(gotProfiles, wantProfiles) {
		t.Fatalf("memory profile matrix mismatch: got=%#v want=%#v", gotProfiles, wantProfiles)
	}
}

func TestMemoryAdapterConformanceUnsupportedProfileFailFast(t *testing.T) {
	_, err := ResolveMemoryProfileSuite("unsupported")
	if err == nil {
		t.Fatal("unsupported profile should fail fast")
	}
	ce := &ConformanceError{}
	if !errors.As(err, &ce) {
		t.Fatalf("expected ConformanceError, got %T (%v)", err, err)
	}
	if ce.Class != types.ErrContext || ce.ReasonCode != "memory.profile_unknown" {
		t.Fatalf("unexpected error classification: %#v", ce)
	}
}

func TestMemoryAdapterConformanceRequiredOperationMissing(t *testing.T) {
	result := EvaluateMemoryOperationCoverage(
		[]string{memorypkg.OperationQuery, memorypkg.OperationUpsert, memorypkg.OperationDelete},
		[]string{"metadata_filter"},
		[]string{memorypkg.OperationQuery, memorypkg.OperationUpsert},
	)
	if result.Accepted {
		t.Fatalf("required operation missing should fail fast: %#v", result)
	}
	if result.DriftClass != MemoryDriftRequiredCapabilityMissing {
		t.Fatalf("required operation drift class mismatch: %#v", result)
	}
	if len(result.MissingRequired) != 1 || result.MissingRequired[0] != memorypkg.OperationDelete {
		t.Fatalf("required operation missing mismatch: %#v", result)
	}
}

func TestMemoryAdapterConformanceOptionalOperationDowngrade(t *testing.T) {
	result := EvaluateMemoryOperationCoverage(
		[]string{memorypkg.OperationQuery, memorypkg.OperationUpsert, memorypkg.OperationDelete},
		[]string{"metadata_filter"},
		[]string{memorypkg.OperationQuery, memorypkg.OperationUpsert, memorypkg.OperationDelete},
	)
	if !result.Accepted {
		t.Fatalf("optional operation should allow downgrade path: %#v", result)
	}
	if len(result.DowngradedOptional) != 1 || result.DowngradedOptional[0] != "metadata_filter" {
		t.Fatalf("optional downgrade mismatch: %#v", result)
	}
}

func TestMemoryAdapterConformanceRunStreamEquivalence(t *testing.T) {
	drift, err := EvaluateMemoryRunStreamEquivalence(
		MemoryRunStreamObservation{
			Operations:     []string{memorypkg.OperationQuery, memorypkg.OperationUpsert, memorypkg.OperationDelete},
			FallbackPolicy: memorypkg.FallbackPolicyDegradeToBuiltin,
			FallbackUsed:   true,
			ReasonCode:     "memory.fallback.used",
		},
		MemoryRunStreamObservation{
			Operations:     []string{memorypkg.OperationDelete, memorypkg.OperationUpsert, memorypkg.OperationQuery},
			FallbackPolicy: memorypkg.FallbackPolicyDegradeToBuiltin,
			FallbackUsed:   true,
			ReasonCode:     "memory.fallback.used",
		},
	)
	if err != nil || drift != "" {
		t.Fatalf("equivalent run/stream memory sequence should pass drift=%q err=%v", drift, err)
	}
}

func TestMemoryAdapterConformanceFallbackDriftClassification(t *testing.T) {
	drift, err := EvaluateMemoryRunStreamEquivalence(
		MemoryRunStreamObservation{
			Operations:     []string{memorypkg.OperationQuery},
			FallbackPolicy: memorypkg.FallbackPolicyFailFast,
			FallbackUsed:   false,
			ReasonCode:     "memory.ok",
		},
		MemoryRunStreamObservation{
			Operations:     []string{memorypkg.OperationQuery},
			FallbackPolicy: memorypkg.FallbackPolicyDegradeWithoutMemory,
			FallbackUsed:   true,
			ReasonCode:     "memory.fallback.used",
		},
	)
	if err == nil {
		t.Fatal("fallback drift should fail")
	}
	if drift != MemoryDriftFallback {
		t.Fatalf("fallback drift classification mismatch: %q", drift)
	}
}

func TestMemoryAdapterConformanceErrorTaxonomyValidation(t *testing.T) {
	if err := ValidateMemoryErrorTaxonomy("memory.provider_unavailable"); err != nil {
		t.Fatalf("memory taxonomy should be accepted: %v", err)
	}
	if err := ValidateMemoryErrorTaxonomy("provider_unavailable"); err == nil {
		t.Fatal("invalid memory taxonomy should fail")
	}
}

func TestMemoryAdapterConformanceCanonicalDriftClasses(t *testing.T) {
	got := CanonicalMemoryDriftClasses()
	want := []string{
		MemoryDriftProfileUnknown,
		MemoryDriftRequiredCapabilityMissing,
		MemoryDriftFallback,
		MemoryDriftRunStreamSemantic,
		MemoryDriftErrorTaxonomy,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("canonical memory drift classes mismatch: got=%#v want=%#v", got, want)
	}
}
