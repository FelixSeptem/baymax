package catalog

import (
	"reflect"
	"strings"
	"testing"
)

func TestNormalizeRoutingAdmissionCanonicalizesEquivalentCandidates(t *testing.T) {
	first := RoutingAdmissionInput{
		Version:           RoutingAdmissionVersionV1,
		CatalogGeneration: "generation-1",
		Source:            RoutingAdmissionSourceHost,
		Candidates: []RoutingCandidate{
			{Identity: Identity{Provider: " OpenAI ", Model: " GPT-4.1-mini "}, Priority: intPtr(2)},
			{Identity: Identity{Provider: "Local", Model: " Small "}, Priority: intPtr(1)},
		},
		Required: []string{" streaming "},
		Optional: []string{"VISION", "vision"},
	}
	second := first
	second.Candidates = []RoutingCandidate{
		{Identity: Identity{Provider: "local", Model: "small"}, Priority: intPtr(1)},
		{Identity: Identity{Provider: "openai", Model: "gpt-4.1-mini"}, Priority: intPtr(2)},
	}
	second.Required = []string{"streaming"}
	second.Optional = []string{"vision"}

	gotFirst, err := NormalizeRoutingAdmission(first)
	if err != nil {
		t.Fatalf("NormalizeRoutingAdmission(first) error = %v", err)
	}
	gotSecond, err := NormalizeRoutingAdmission(second)
	if err != nil {
		t.Fatalf("NormalizeRoutingAdmission(second) error = %v", err)
	}
	if !reflect.DeepEqual(gotFirst, gotSecond) {
		t.Fatalf("equivalent inputs normalized differently: first=%#v second=%#v", gotFirst, gotSecond)
	}
	if gotFirst.CanonicalDigest == "" {
		t.Fatal("CanonicalDigest is empty")
	}
}

func TestNormalizeRoutingAdmissionRejectsDuplicateAndPrivacyMaterial(t *testing.T) {
	duplicate := RoutingAdmissionInput{
		Version:           RoutingAdmissionVersionV1,
		CatalogGeneration: "generation-1",
		Source:            RoutingAdmissionSourceHost,
		Candidates: []RoutingCandidate{
			{Identity: Identity{Provider: "openai", Model: "gpt-4.1-mini"}},
			{Identity: Identity{Provider: " OPENAI ", Model: " GPT-4.1-MINI "}},
		},
	}
	if _, err := NormalizeRoutingAdmission(duplicate); !HasReason(err, ReasonAuditDuplicateCandidate) {
		t.Fatalf("duplicate normalization error = %v, want reason %q", err, ReasonAuditDuplicateCandidate)
	}

	privacy := duplicate
	privacy.Candidates = []RoutingCandidate{{Identity: Identity{Provider: "openai", Model: "gpt-4.1-mini"}, Metadata: "sk-live-token"}}
	if _, err := NormalizeRoutingAdmission(privacy); !HasReason(err, ReasonAuditPrivacyViolation) {
		t.Fatalf("privacy normalization error = %v, want reason %q", err, ReasonAuditPrivacyViolation)
	}
}

func TestNormalizeRoutingAdmissionRejectsBoundsAndUnknownSource(t *testing.T) {
	input := RoutingAdmissionInput{
		Version:           RoutingAdmissionVersionV1,
		CatalogGeneration: "generation-1",
		Source:            "remote-discovery",
		Candidates:        []RoutingCandidate{{Identity: Identity{Provider: "openai", Model: "gpt-4.1-mini"}}},
	}
	if _, err := NormalizeRoutingAdmission(input); !HasReason(err, ReasonAuditUnsupportedSource) {
		t.Fatalf("unknown source error = %v, want reason %q", err, ReasonAuditUnsupportedSource)
	}

	input.Source = RoutingAdmissionSourceHost
	input.CatalogGeneration = ""
	if _, err := NormalizeRoutingAdmission(input); !HasReason(err, ReasonAuditInvalidGeneration) {
		t.Fatalf("invalid generation error = %v, want reason %q", err, ReasonAuditInvalidGeneration)
	}
}

func TestAuditRoutingAdmissionUsesExactIdentityAndPreservesGeneration(t *testing.T) {
	catalog := mustCatalog(t,
		Descriptor{Provider: "openai", Model: "fast", ContextWindow: 1, Capabilities: []string{"streaming"}},
		Descriptor{Provider: "openai", Model: "vision", ContextWindow: 1, Capabilities: []string{"streaming", "vision"}},
	)
	result, err := AuditRoutingAdmission(catalog, RoutingAdmissionInput{
		Version:           RoutingAdmissionVersionV1,
		CatalogGeneration: "generation-7",
		Source:            RoutingAdmissionSourceHost,
		Candidates:        []RoutingCandidate{{Identity: Identity{Provider: "openai", Model: "fast"}}},
		Required:          []string{"streaming"},
		Optional:          []string{"vision"},
		Strategy:          "best_effort",
	}, map[string]CredentialEvidence{"openai": {Provider: "openai", Status: CredentialAvailable}}, false)
	if err != nil {
		t.Fatalf("AuditRoutingAdmission() error = %v", err)
	}
	if result.CatalogGeneration != "generation-7" {
		t.Fatalf("CatalogGeneration = %q, want generation-7", result.CatalogGeneration)
	}
	if result.Selected == nil || *result.Selected != (Identity{Provider: "openai", Model: "fast"}) {
		t.Fatalf("Selected = %#v, want openai/fast", result.Selected)
	}
	if result.Status != StatusDegraded || result.Fallback != nil {
		t.Fatalf("result = %#v, want exact-identity degraded admission without fallback", result)
	}
}

func TestAuditRoutingAdmissionBlocksRequiredCapabilityBeforeSelection(t *testing.T) {
	catalog := mustCatalog(t, Descriptor{Provider: "openai", Model: "fast", ContextWindow: 1, Capabilities: []string{"streaming"}})
	result, err := AuditRoutingAdmission(catalog, RoutingAdmissionInput{
		Version:           RoutingAdmissionVersionV1,
		CatalogGeneration: "generation-8",
		Source:            RoutingAdmissionSourceHost,
		Candidates:        []RoutingCandidate{{Identity: Identity{Provider: "openai", Model: "fast"}}},
		Required:          []string{"tool_call"},
	}, map[string]CredentialEvidence{"openai": {Provider: "openai", Status: CredentialAvailable}}, false)
	if err != nil {
		t.Fatalf("AuditRoutingAdmission() error = %v", err)
	}
	if result.Status != StatusBlocked || result.Selected != nil {
		t.Fatalf("result = %#v, want blocked with no selection", result)
	}
	if !reflect.DeepEqual(result.Reasons, []string{ReasonAuditNoAdmissibleCandidate}) {
		t.Fatalf("Reasons = %#v, want no-admissible classification", result.Reasons)
	}
}

func TestAuditRoutingAdmissionRejectsAmbiguousAdmissibleCandidates(t *testing.T) {
	catalog := mustCatalog(t,
		Descriptor{Provider: "openai", Model: "fast", ContextWindow: 1},
		Descriptor{Provider: "local", Model: "small", ContextWindow: 1},
	)
	result, err := AuditRoutingAdmission(catalog, RoutingAdmissionInput{
		Version:           RoutingAdmissionVersionV1,
		CatalogGeneration: "generation-9",
		Source:            RoutingAdmissionSourceHost,
		Candidates: []RoutingCandidate{
			{Identity: Identity{Provider: "openai", Model: "fast"}, Priority: intPtr(2)},
			{Identity: Identity{Provider: "local", Model: "small"}, Priority: intPtr(1)},
		},
	}, map[string]CredentialEvidence{
		"openai": {Provider: "openai", Status: CredentialAvailable},
		"local":  {Provider: "local", Status: CredentialAvailable},
	}, false)
	if err != nil {
		t.Fatalf("AuditRoutingAdmission() error = %v", err)
	}
	if result.Status != StatusBlocked || result.Selected != nil {
		t.Fatalf("result = %#v, want ambiguous blocked result", result)
	}
	if !reflect.DeepEqual(result.Reasons, []string{ReasonAuditAmbiguousSelection}) {
		t.Fatalf("Reasons = %#v, want ambiguous selection", result.Reasons)
	}
}

func TestAuditRoutingAdmissionStrictnessPreservesCredentialReason(t *testing.T) {
	catalog := mustCatalog(t, Descriptor{Provider: "openai", Model: "fast", ContextWindow: 1})
	input := RoutingAdmissionInput{
		Version:           RoutingAdmissionVersionV1,
		CatalogGeneration: "generation-10",
		Source:            RoutingAdmissionSourceHost,
		Candidates:        []RoutingCandidate{{Identity: Identity{Provider: "openai", Model: "fast"}}},
	}
	strict, err := AuditRoutingAdmission(catalog, input, map[string]CredentialEvidence{
		"openai": {Provider: "openai", Status: CredentialUnverified},
	}, true)
	if err != nil {
		t.Fatalf("strict audit error = %v", err)
	}
	if strict.Status != StatusBlocked || strict.Selected != nil {
		t.Fatalf("strict result = %#v, want blocked", strict)
	}

	nonstrict, err := AuditRoutingAdmission(catalog, input, map[string]CredentialEvidence{
		"openai": {Provider: "openai", Status: CredentialUnverified},
	}, false)
	if err != nil {
		t.Fatalf("non-strict audit error = %v", err)
	}
	if nonstrict.Status != StatusDegraded || nonstrict.Selected == nil {
		t.Fatalf("non-strict result = %#v, want degraded selection", nonstrict)
	}
	if !reflect.DeepEqual(nonstrict.Reasons, []string{ReasonCredentialUnverified}) {
		t.Fatalf("non-strict reasons = %#v, want credential reason", nonstrict.Reasons)
	}
}

func TestAuditRoutingAdmissionRedactsSensitiveCredentialReason(t *testing.T) {
	catalog := mustCatalog(t, Descriptor{Provider: "openai", Model: "fast", ContextWindow: 1})
	result, err := AuditRoutingAdmission(catalog, RoutingAdmissionInput{
		Version:           RoutingAdmissionVersionV1,
		CatalogGeneration: "generation-15",
		Source:            RoutingAdmissionSourceHost,
		Candidates:        []RoutingCandidate{{Identity: Identity{Provider: "openai", Model: "fast"}}},
	}, map[string]CredentialEvidence{
		"openai": {Provider: "openai", Status: CredentialAvailable, Reason: "sk-live-secret"},
	}, false)
	if err != nil {
		t.Fatalf("AuditRoutingAdmission() error = %v", err)
	}
	if strings.Contains(result.Credential.Reason, "sk-live-secret") || strings.Contains(strings.ToLower(result.Credential.Reason), "secret") {
		t.Fatalf("credential reason leaked sensitive material: %#v", result.Credential)
	}
}

func TestNormalizeRoutingAdmissionRejectsConflictingPriorityAndOverflow(t *testing.T) {
	priority := RoutingAdmissionInput{
		Version:           RoutingAdmissionVersionV1,
		CatalogGeneration: "generation-11",
		Source:            RoutingAdmissionSourceHost,
		Candidates: []RoutingCandidate{
			{Identity: Identity{Provider: "openai", Model: "a"}, Priority: intPtr(1)},
			{Identity: Identity{Provider: "openai", Model: "b"}, Priority: intPtr(1)},
		},
	}
	if _, err := NormalizeRoutingAdmission(priority); !HasReason(err, ReasonAuditPriorityConflict) {
		t.Fatalf("priority error = %v, want reason %q", err, ReasonAuditPriorityConflict)
	}

	overflow := RoutingAdmissionInput{
		Version:           RoutingAdmissionVersionV1,
		CatalogGeneration: "generation-12",
		Source:            RoutingAdmissionSourceHost,
		Candidates:        make([]RoutingCandidate, MaxRoutingAdmissionCandidates+1),
	}
	if _, err := NormalizeRoutingAdmission(overflow); !HasReason(err, ReasonAuditOverflow) {
		t.Fatalf("overflow error = %v, want reason %q", err, ReasonAuditOverflow)
	}
}

func TestResolveRoutingAdmissionSelectsHighestPriorityAdmissibleCandidate(t *testing.T) {
	catalog := mustCatalog(t,
		Descriptor{Provider: "openai", Model: "fast", ContextWindow: 1, Capabilities: []string{"streaming"}},
		Descriptor{Provider: "local", Model: "small", ContextWindow: 1, Capabilities: []string{"streaming", "vision"}},
	)
	input := RoutingAdmissionInput{
		Version:           RoutingAdmissionVersionV1,
		CatalogGeneration: "generation-13",
		Source:            RoutingAdmissionSourceHost,
		Candidates: []RoutingCandidate{
			{Identity: Identity{Provider: "openai", Model: "fast"}, Priority: intPtr(1)},
			{Identity: Identity{Provider: "local", Model: "small"}, Priority: intPtr(2)},
		},
		Required: []string{"streaming"},
		Optional: []string{"vision"},
		Strategy: "best_effort",
	}
	result, err := ResolveRoutingAdmission(catalog, input, map[string]CredentialEvidence{
		"openai": {Provider: "openai", Status: CredentialAvailable},
		"local":  {Provider: "local", Status: CredentialAvailable},
	}, false)
	if err != nil {
		t.Fatalf("ResolveRoutingAdmission() error = %v", err)
	}
	if !result.ResolverActivated {
		t.Fatal("ResolverActivated = false, want true")
	}
	if result.Selected == nil || *result.Selected != (Identity{Provider: "local", Model: "small"}) {
		t.Fatalf("Selected = %#v, want local/small", result.Selected)
	}
	if len(result.CandidateOutcomes) != 1 || result.CandidateOutcomes[0].Identity != (Identity{Provider: "openai", Model: "fast"}) {
		t.Fatalf("CandidateOutcomes = %#v, want skipped lower-priority candidate", result.CandidateOutcomes)
	}
}

func TestResolveRoutingAdmissionDoesNotUseLastWriteWinsOnPriorityConflict(t *testing.T) {
	_, err := ResolveRoutingAdmission(mustCatalog(t, Descriptor{Provider: "openai", Model: "fast", ContextWindow: 1}), RoutingAdmissionInput{
		Version:           RoutingAdmissionVersionV1,
		CatalogGeneration: "generation-14",
		Source:            RoutingAdmissionSourceHost,
		Candidates: []RoutingCandidate{
			{Identity: Identity{Provider: "openai", Model: "fast"}, Priority: intPtr(1)},
			{Identity: Identity{Provider: "local", Model: "small"}, Priority: intPtr(1)},
		},
	}, nil, false)
	if !HasReason(err, ReasonAuditPriorityConflict) {
		t.Fatalf("ResolveRoutingAdmission() error = %v, want reason %q", err, ReasonAuditPriorityConflict)
	}
}

func TestAuditRoutingAdmissionRetainsObservedGenerationAcrossCatalogReload(t *testing.T) {
	firstCatalog := mustCatalog(t, Descriptor{Provider: "openai", Model: "fast", ContextWindow: 1})
	input := RoutingAdmissionInput{
		Version:           RoutingAdmissionVersionV1,
		CatalogGeneration: "generation-before-reload",
		Source:            RoutingAdmissionSourceHost,
		Candidates:        []RoutingCandidate{{Identity: Identity{Provider: "openai", Model: "fast"}}},
	}
	before, err := AuditRoutingAdmission(firstCatalog, input, map[string]CredentialEvidence{"openai": {Provider: "openai", Status: CredentialAvailable}}, false)
	if err != nil {
		t.Fatalf("first audit: %v", err)
	}
	secondCatalog := mustCatalog(t, Descriptor{Provider: "openai", Model: "fast", ContextWindow: 1}, Descriptor{Provider: "local", Model: "small", ContextWindow: 1})
	input.CatalogGeneration = "generation-after-reload"
	after, err := AuditRoutingAdmission(secondCatalog, input, map[string]CredentialEvidence{"openai": {Provider: "openai", Status: CredentialAvailable}}, false)
	if err != nil {
		t.Fatalf("second audit: %v", err)
	}
	if before.CatalogGeneration != "generation-before-reload" || after.CatalogGeneration != "generation-after-reload" {
		t.Fatalf("generation facts changed unexpectedly: before=%#v after=%#v", before, after)
	}
}

func intPtr(value int) *int { return &value }
