package catalog

import (
	"reflect"
	"testing"
)

func TestNewNormalizesAndFreezesDescriptors(t *testing.T) {
	catalog, err := New(Input{
		Version: " release-1 ",
		Descriptors: []Descriptor{
			{
				Provider:      " OpenAI ",
				Model:         " GPT-4.1-mini ",
				ContextWindow: 128000,
				Capabilities:  []string{" Streaming ", "vision", "streaming"},
			},
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	descriptor, ok := catalog.Lookup(Identity{Provider: "openai", Model: "gpt-4.1-mini"})
	if !ok {
		t.Fatal("Lookup() found no normalized descriptor")
	}
	if got, want := catalog.Version(), "release-1"; got != want {
		t.Fatalf("Version() = %q, want %q", got, want)
	}
	if got, want := descriptor.Capabilities, []string{"streaming", "vision"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Capabilities = %#v, want %#v", got, want)
	}

	descriptor.Capabilities[0] = "mutated"
	again, ok := catalog.Lookup(Identity{Provider: "openai", Model: "gpt-4.1-mini"})
	if !ok || again.Capabilities[0] != "streaming" {
		t.Fatalf("Lookup() leaked mutable descriptor: %#v", again)
	}
}

func TestNewRejectsDuplicateNormalizedIdentity(t *testing.T) {
	_, err := New(Input{
		Version: "v1",
		Descriptors: []Descriptor{
			{Provider: "openai", Model: "gpt-4.1-mini", ContextWindow: 1},
			{Provider: " OPENAI ", Model: " GPT-4.1-MINI ", ContextWindow: 1},
		},
	})
	if !HasReason(err, ReasonDuplicateDescriptor) {
		t.Fatalf("New() error = %v, want reason %q", err, ReasonDuplicateDescriptor)
	}
}

func TestNewRejectsMalformedCapabilityAndFallbackCycle(t *testing.T) {
	_, err := New(Input{
		Version: "v1",
		Descriptors: []Descriptor{{
			Provider:      "openai",
			Model:         "gpt-4.1-mini",
			ContextWindow: 1,
			Capabilities:  []string{"vision capability"},
		}},
	})
	if !HasReason(err, ReasonInvalidCapability) {
		t.Fatalf("New() malformed capability error = %v, want reason %q", err, ReasonInvalidCapability)
	}

	_, err = New(Input{
		Version: "v1",
		Descriptors: []Descriptor{
			{Provider: "openai", Model: "a", ContextWindow: 1, Fallback: &Identity{Provider: "openai", Model: "b"}},
			{Provider: "openai", Model: "b", ContextWindow: 1, Fallback: &Identity{Provider: "openai", Model: "a"}},
		},
	})
	if !HasReason(err, ReasonInvalidFallback) {
		t.Fatalf("New() fallback cycle error = %v, want reason %q", err, ReasonInvalidFallback)
	}
}

func TestEvaluateBlocksMissingCredentialWithoutExposingMaterial(t *testing.T) {
	catalog := mustCatalog(t, Descriptor{
		Provider:      "openai",
		Model:         "gpt-4.1-mini",
		ContextWindow: 128000,
		Capabilities:  []string{"streaming"},
	})

	admission, err := Evaluate(catalog, Request{
		Identity: Identity{Provider: "openai", Model: "gpt-4.1-mini"},
		Required: []string{"streaming"},
	}, map[string]CredentialEvidence{
		"openai": {Provider: "openai", Status: CredentialMissing, Reason: "credential.missing"},
	}, false)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if admission.Status != StatusBlocked {
		t.Fatalf("Status = %q, want %q", admission.Status, StatusBlocked)
	}
	if got, want := admission.Reasons, []string{ReasonCredentialMissing}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Reasons = %#v, want %#v", got, want)
	}
	if admission.Credential.Reason != "credential.missing" || admission.Credential.Status != CredentialMissing {
		t.Fatalf("Credential = %#v, want redacted evidence", admission.Credential)
	}
}

func TestEvaluateBlocksUnknownCredentialStatus(t *testing.T) {
	catalog := mustCatalog(t, Descriptor{Provider: "openai", Model: "gpt-4.1-mini", ContextWindow: 1})

	admission, err := Evaluate(catalog, Request{Identity: Identity{Provider: "openai", Model: "gpt-4.1-mini"}}, map[string]CredentialEvidence{
		"openai": {Provider: "openai", Status: "token-present"},
	}, false)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if admission.Status != StatusBlocked {
		t.Fatalf("Status = %q, want %q", admission.Status, StatusBlocked)
	}
	if got, want := admission.Reasons, []string{ReasonCredentialInvalid}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Reasons = %#v, want %#v", got, want)
	}
}

func TestEvaluateUsesDeclaredFallbackForOptionalCapability(t *testing.T) {
	catalog := mustCatalog(t,
		Descriptor{
			Provider:      "openai",
			Model:         "fast",
			ContextWindow: 32000,
			Capabilities:  []string{"streaming"},
			Fallback:      &Identity{Provider: "openai", Model: "vision"},
		},
		Descriptor{
			Provider:      "openai",
			Model:         "vision",
			ContextWindow: 128000,
			Capabilities:  []string{"streaming", "vision"},
		},
	)

	admission, err := Evaluate(catalog, Request{
		Identity: Identity{Provider: "openai", Model: "fast"},
		Optional: []string{"vision"},
		Strategy: "best_effort",
	}, map[string]CredentialEvidence{
		"openai": {Provider: "openai", Status: CredentialAvailable},
	}, false)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if admission.Status != StatusDegraded {
		t.Fatalf("Status = %q, want %q", admission.Status, StatusDegraded)
	}
	if admission.Fallback == nil || *admission.Fallback != (Identity{Provider: "openai", Model: "vision"}) {
		t.Fatalf("Fallback = %#v, want openai/vision", admission.Fallback)
	}
	if got, want := admission.Reasons, []string{"adapter.capability.optional_downgraded", ReasonOptionalFallback}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Reasons = %#v, want %#v", got, want)
	}
}

func mustCatalog(t *testing.T, descriptors ...Descriptor) Catalog {
	t.Helper()
	catalog, err := New(Input{Version: "v1", Descriptors: descriptors})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return catalog
}
