package diagnosticsreplay

import (
	"errors"
	"reflect"
	"sort"
	"testing"

	"github.com/FelixSeptem/baymax/runtime/evalcontract"
)

func TestFirstErrorAttributionFixtureSuccessIsDeterministic(t *testing.T) {
	fixture := readFirstErrorAttributionFixture(t, "eval_first_error_attribution.v1.json")

	first, err := EvaluateFirstErrorAttributionFixture(fixture)
	if err != nil {
		t.Fatal(err)
	}
	second, err := EvaluateFirstErrorAttributionFixture(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("repeated replay changed output:\nfirst:  %#v\nsecond: %#v", first, second)
	}
	if len(first.Cases) != 1 || !first.Cases[0].Passed || first.Cases[0].BaselineIdentity == "" || first.Cases[0].BaselineIdentity != first.Cases[0].CandidateIdentity {
		t.Fatalf("unexpected normalized success output: %#v", first)
	}
}

func TestFirstErrorAttributionFixtureClassifiesSemanticDrift(t *testing.T) {
	fixture := readFirstErrorAttributionFixture(t, "eval_first_error_attribution_semantic_drift.json")

	result, err := EvaluateFirstErrorAttributionFixture(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Cases) != 1 || result.Cases[0].Passed {
		t.Fatalf("expected semantic drift output: %#v", result)
	}
	actual := append([]string(nil), result.Cases[0].Drifts...)
	sort.Strings(actual)
	expected := []string{
		evalcontract.ReasonFirstErrorCauseDrift,
		evalcontract.ReasonFirstErrorKindDrift,
		evalcontract.ReasonFirstErrorStepDrift,
	}
	sort.Strings(expected)
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("drifts = %v, want %v", actual, expected)
	}
}

func TestFirstErrorAttributionFixtureEvaluatesMemoryApplicationOffline(t *testing.T) {
	fixture := readFirstErrorAttributionFixture(t, "eval_first_error_attribution_memory_application.json")

	result, err := EvaluateFirstErrorAttributionFixture(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Cases) != 1 || !result.Cases[0].Passed || result.Cases[0].NormalizedKind != "memory_application" || result.Cases[0].NormalizedOwner != "memory" {
		t.Fatalf("unexpected memory application output: %#v", result)
	}
}

func TestFirstErrorAttributionFixtureRejectsBodyBearingEvidence(t *testing.T) {
	fixture := readFirstErrorAttributionFixture(t, "eval_first_error_attribution_privacy_violation.json")

	result, err := EvaluateFirstErrorAttributionFixture(fixture)
	var validationErr *ValidationError
	if err == nil || !errors.As(err, &validationErr) || validationErr.Code != evalcontract.ReasonFirstErrorPrivacyViolation {
		t.Fatalf("expected privacy rejection, got result=%#v err=%v", result, err)
	}
	if len(result.Cases) != 0 {
		t.Fatalf("privacy rejection emitted partial success: %#v", result)
	}
}

func TestFirstErrorAttributionFixtureRejectsMalformedVersion(t *testing.T) {
	raw := []byte(`{"version":"eval_first_error_attribution.v2","cases":[{"name":"unsupported"}]}`)
	fixture, err := ParseFirstErrorAttributionFixtureJSON(raw)
	var validationErr *ValidationError
	if err == nil || !errors.As(err, &validationErr) || validationErr.Code != evalcontract.ReasonFirstErrorSchemaDrift {
		t.Fatalf("expected schema rejection, got fixture=%#v err=%v", fixture, err)
	}
}

func readFirstErrorAttributionFixture(t *testing.T, name string) FirstErrorAttributionFixture {
	t.Helper()
	raw := mustReadFixture(t, name)
	fixture, err := ParseFirstErrorAttributionFixtureJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	return fixture
}
