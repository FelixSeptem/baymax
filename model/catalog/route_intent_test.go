package catalog

import (
	"strings"
	"testing"
)

func TestCompareRouteIntentAdmissionSatisfiedAndCanonicalizes(t *testing.T) {
	capabilitiesAccepted := true
	input := RouteIntentAdmissionInput{
		Version: RouteIntentAdmissionVersionV1,
		Intent: RouteIntent{
			Target:   &Identity{Provider: "OpenAI", Model: "GPT-4o"},
			Allowed:  []Identity{{Provider: "OPENAI", Model: "GPT-4o"}, {Provider: "local", Model: "small"}},
			Required: []string{"tools", "TOOLS"},
		},
		Facts: &RouteIntentAdmissionFacts{CatalogGeneration: "v1", Status: StatusReady, Selected: &Identity{Provider: "openai", Model: "gpt-4o"}, CapabilitiesAccepted: &capabilitiesAccepted},
	}
	result, err := CompareRouteIntentAdmission(input)
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if result.Verdict != RouteIntentVerdictSatisfied || result.Selected == nil || result.Selected.Provider != "openai" {
		t.Fatalf("result = %#v", result)
	}
	if result.CanonicalDigest == "" {
		t.Fatal("canonical digest is empty")
	}

	input.Intent.Allowed = []Identity{{Provider: "local", Model: "small"}, {Provider: "OPENAI", Model: "GPT-4o"}}
	second, err := CompareRouteIntentAdmission(input)
	if err != nil {
		t.Fatalf("equivalent compare: %v", err)
	}
	if result.CanonicalDigest != second.CanonicalDigest {
		t.Fatalf("equivalent input digest drift: %q != %q", result.CanonicalDigest, second.CanonicalDigest)
	}
}

func TestCompareRouteIntentAdmissionDistinguishesGapAndInsufficientEvidence(t *testing.T) {
	gap, err := CompareRouteIntentAdmission(RouteIntentAdmissionInput{
		Version: RouteIntentAdmissionVersionV1,
		Intent:  RouteIntent{Target: &Identity{Provider: "openai", Model: "gpt-4o"}},
		Facts:   &RouteIntentAdmissionFacts{CatalogGeneration: "v1", Status: StatusReady, Selected: &Identity{Provider: "local", Model: "small"}},
	})
	if err != nil {
		t.Fatalf("gap compare: %v", err)
	}
	if gap.Verdict != RouteIntentVerdictRouteGapConfirmed || !strings.Contains(strings.Join(gap.Reasons, ","), ReasonRouteIntentTargetNotSelected) {
		t.Fatalf("gap = %#v", gap)
	}

	insufficient, err := CompareRouteIntentAdmission(RouteIntentAdmissionInput{
		Version: RouteIntentAdmissionVersionV1,
		Intent:  RouteIntent{Target: &Identity{Provider: "openai", Model: "gpt-4o"}},
	})
	if err != nil {
		t.Fatalf("insufficient compare: %v", err)
	}
	if insufficient.Verdict != RouteIntentVerdictInsufficientEvidence || insufficient.Reasons[0] != ReasonRouteIntentMissingFacts {
		t.Fatalf("insufficient = %#v", insufficient)
	}
}

func TestCompareRouteIntentAdmissionParityAndGeneration(t *testing.T) {
	base := RouteIntentAdmissionFacts{
		CatalogGeneration: "v1",
		Status:            StatusReady,
		Selected:          &Identity{Provider: "openai", Model: "gpt-4o"},
		Run:               &RouteIntentAdmissionSnapshot{CatalogGeneration: "v1", Status: StatusReady, Selected: &Identity{Provider: "openai", Model: "gpt-4o"}},
		Stream:            &RouteIntentAdmissionSnapshot{CatalogGeneration: "v1", Status: StatusReady, Selected: &Identity{Provider: "openai", Model: "gpt-4o"}},
	}
	result, err := CompareRouteIntentAdmission(RouteIntentAdmissionInput{
		Version: RouteIntentAdmissionVersionV1,
		Intent:  RouteIntent{Target: &Identity{Provider: "openai", Model: "gpt-4o"}, ExpectedCatalogGeneration: "v1", RequireRunStreamParity: true},
		Facts:   &base,
	})
	if err != nil {
		t.Fatalf("parity compare: %v", err)
	}
	if result.Verdict != RouteIntentVerdictSatisfied || !result.RunStreamParity {
		t.Fatalf("parity result = %#v", result)
	}

	base.Stream.Status = StatusDegraded
	result, err = CompareRouteIntentAdmission(RouteIntentAdmissionInput{
		Version: RouteIntentAdmissionVersionV1,
		Intent:  RouteIntent{Target: &Identity{Provider: "openai", Model: "gpt-4o"}, RequireRunStreamParity: true},
		Facts:   &base,
	})
	if err != nil {
		t.Fatalf("parity drift compare: %v", err)
	}
	if result.Verdict != RouteIntentVerdictRouteGapConfirmed || result.Reasons[0] != ReasonRouteIntentParityMismatch {
		t.Fatalf("parity drift = %#v", result)
	}

	badGeneration := RouteIntentAdmissionInput{
		Version: RouteIntentAdmissionVersionV1,
		Intent:  RouteIntent{Target: &Identity{Provider: "openai", Model: "gpt-4o"}, ExpectedCatalogGeneration: "v2"},
		Facts:   &RouteIntentAdmissionFacts{CatalogGeneration: "v1", Status: StatusReady, Selected: &Identity{Provider: "openai", Model: "gpt-4o"}},
	}
	result, err = CompareRouteIntentAdmission(badGeneration)
	if err != nil {
		t.Fatalf("generation compare: %v", err)
	}
	if result.Verdict != RouteIntentVerdictInsufficientEvidence || result.Reasons[0] != ReasonRouteIntentGenerationMismatch {
		t.Fatalf("generation result = %#v", result)
	}
}

func TestNormalizeRouteIntentAdmissionRejectsBoundsAndPrivacyByConstruction(t *testing.T) {
	if _, err := NormalizeRouteIntentAdmission(RouteIntentAdmissionInput{Version: "bad", Intent: RouteIntent{Target: &Identity{Provider: "p", Model: "m"}}}); err == nil || ErrorCode(err) != ReasonRouteIntentUnsupportedVersion {
		t.Fatalf("unsupported version error = %v", err)
	}
	tooMany := make([]Identity, MaxRouteIntentAllowedCandidates+1)
	for index := range tooMany {
		tooMany[index] = Identity{Provider: "p", Model: "m" + string(rune('a'+index))}
	}
	if _, err := NormalizeRouteIntentAdmission(RouteIntentAdmissionInput{Version: RouteIntentAdmissionVersionV1, Intent: RouteIntent{Allowed: tooMany}}); err == nil || ErrorCode(err) != ReasonRouteIntentOverflow {
		t.Fatalf("overflow error = %v", err)
	}
	if _, err := NormalizeRouteIntentAdmission(RouteIntentAdmissionInput{Version: RouteIntentAdmissionVersionV1, Intent: RouteIntent{Target: &Identity{Provider: "https://provider", Model: "model"}}}); err == nil || ErrorCode(err) != ReasonRouteIntentPrivacyViolation {
		t.Fatalf("privacy error = %v", err)
	}
}
