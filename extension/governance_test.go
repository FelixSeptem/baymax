package extension

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateDescriptorRejectsMissingIdentity(t *testing.T) {
	_, err := ValidateDescriptor(Descriptor{})
	if err == nil {
		t.Fatal("expected missing identity error")
	}
	if got := ErrorCode(err); got != ReasonMissingField {
		t.Fatalf("error code = %q, want %q", got, ReasonMissingField)
	}
}

func TestResolveResourcesUsesDeterministicPrecedence(t *testing.T) {
	candidates := []Descriptor{
		{Name: "lint", Kind: KindHook, Source: SourceUserAuto, Version: "1.0.0", Compat: ">=0.1.0 <1.0.0", Digest: "sha256:user"},
		{Name: "lint", Kind: KindHook, Source: SourceProjectExplicit, Version: "1.0.0", Compat: ">=0.1.0 <1.0.0", Digest: "sha256:project"},
	}
	resolved, err := ResolveResources(candidates)
	if err != nil {
		t.Fatalf("ResolveResources() error = %v", err)
	}
	if len(resolved.Selected) != 1 || resolved.Selected[0].Digest != "sha256:project" {
		t.Fatalf("selected = %#v, want project resource", resolved.Selected)
	}
	if len(resolved.Conflicts) != 1 {
		t.Fatalf("conflicts = %#v, want one conflict", resolved.Conflicts)
	}
}

func TestResolveResourcesRejectsUnorderedEqualPrecedenceConflict(t *testing.T) {
	candidates := []Descriptor{
		{Name: "lint", Kind: KindHook, Source: SourceProjectExplicit, Version: "1.0.0", Compat: ">=0.1.0 <1.0.0", Digest: "sha256:a"},
		{Name: "lint", Kind: KindHook, Source: SourceProjectExplicit, Version: "1.0.0", Compat: ">=0.1.0 <1.0.0", Digest: "sha256:b"},
	}
	_, err := ResolveResources(candidates)
	if err == nil {
		t.Fatal("expected equal precedence conflict")
	}
	if got := ErrorCode(err); got != ReasonAmbiguousConflict {
		t.Fatalf("error code = %q, want %q", got, ReasonAmbiguousConflict)
	}
}

func TestValidateDescriptorAcceptsCapabilitiesAndDigest(t *testing.T) {
	d := Descriptor{
		Name:     "lint",
		Kind:     KindHook,
		Version:  "1.2.3",
		Compat:   ">=0.1.0 <1.0.0",
		Digest:   "sha256:abc",
		Required: []string{"tool.read"},
		Optional: []string{"ui.notify"},
		Source:   SourceProjectExplicit,
	}
	if _, err := ValidateDescriptor(d); err != nil {
		t.Fatalf("ValidateDescriptor() error = %v", err)
	}
}

func TestDiscoverFileDerivesStableDigest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "extension.ts")
	if err := os.WriteFile(path, []byte("export default {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	first, err := DiscoverFile(path, SourceProjectExplicit, Descriptor{
		Name: "lint", Kind: KindHook, Version: "1.0.0", Compat: ">=0.1.0 <1.0.0",
	})
	if err != nil {
		t.Fatalf("DiscoverFile() error = %v", err)
	}
	second, err := DiscoverFile(path, SourceProjectExplicit, Descriptor{
		Name: "lint", Kind: KindHook, Version: "1.0.0", Compat: ">=0.1.0 <1.0.0",
	})
	if err != nil {
		t.Fatalf("DiscoverFile() second error = %v", err)
	}
	if first.Digest == "" || first.Digest != second.Digest {
		t.Fatalf("digest = %q, second = %q", first.Digest, second.Digest)
	}
}

func TestDiscoverFilesSortsPathsBeforeResolving(t *testing.T) {
	dir := t.TempDir()
	firstPath := filepath.Join(dir, "b.ts")
	secondPath := filepath.Join(dir, "a.ts")
	if err := os.WriteFile(firstPath, []byte("b"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secondPath, []byte("a"), 0o600); err != nil {
		t.Fatal(err)
	}
	descriptors := map[string]Descriptor{
		firstPath:  {Name: "b", Kind: KindHook, Version: "1.0.0", Compat: ">=0.1.0 <1.0.0"},
		secondPath: {Name: "a", Kind: KindHook, Version: "1.0.0", Compat: ">=0.1.0 <1.0.0"},
	}
	resolved, err := DiscoverFiles([]string{firstPath, secondPath}, SourceProjectAuto, descriptors)
	if err != nil {
		t.Fatalf("DiscoverFiles() error = %v", err)
	}
	if len(resolved) != 2 || resolved[0].Name != "a" || resolved[1].Name != "b" {
		t.Fatalf("resolved order = %#v, want a,b", resolved)
	}
}

func TestNegotiateCapabilitiesUsesCanonicalAdapterReasons(t *testing.T) {
	d := Descriptor{
		Name: "lint", Kind: KindHook, Version: "1.0.0", Compat: ">=0.1.0 <1.0.0",
		Digest: "sha256:abc", Source: SourceProjectExplicit,
		Required: []string{"tool.read"}, Optional: []string{"ui.notify"},
	}
	result, err := NegotiateCapabilities(d, []string{"tool.read"}, true)
	if err != nil {
		t.Fatalf("NegotiateCapabilities() error = %v", err)
	}
	if !result.Accepted || !result.Downgraded || result.Reasons[0] != "adapter.capability.optional_downgraded" {
		t.Fatalf("result = %#v, want accepted optional downgrade", result)
	}
}

func TestNegotiateCapabilitiesRejectsMissingRequired(t *testing.T) {
	d := Descriptor{
		Name: "lint", Kind: KindHook, Version: "1.0.0", Compat: ">=0.1.0 <1.0.0",
		Digest: "sha256:abc", Source: SourceProjectExplicit, Required: []string{"tool.write"},
	}
	_, err := NegotiateCapabilities(d, []string{"tool.read"}, false)
	if err == nil {
		t.Fatal("expected missing required capability")
	}
	if ErrorCode(err) != "adapter.capability.missing_required" {
		t.Fatalf("error code = %q", ErrorCode(err))
	}
}

func TestAdmissionBlocksInvalidExtensionWithoutActivation(t *testing.T) {
	d := Descriptor{Name: "lint", Kind: KindHook, Version: "1.0.0", Compat: ">=0.1.0 <1.0.0", Source: SourceProjectExplicit}
	decision, err := Admit(d, AdmissionInput{RuntimeVersion: "0.2.0", AvailableCapabilities: []string{"tool.read"}})
	if err != nil {
		t.Fatalf("Admit() error = %v", err)
	}
	if decision.Outcome != AdmissionDeny || decision.Activated {
		t.Fatalf("decision = %#v, want denied and inactive", decision)
	}
}

func TestAdmissionAllowsValidExtension(t *testing.T) {
	d := Descriptor{Name: "lint", Kind: KindHook, Version: "1.0.0", Compat: ">=0.1.0 <1.0.0", Digest: "sha256:abc", Source: SourceProjectExplicit, Required: []string{"tool.read"}}
	decision, err := Admit(d, AdmissionInput{RuntimeVersion: "0.2.0", AvailableCapabilities: []string{"tool.read"}})
	if err != nil {
		t.Fatalf("Admit() error = %v", err)
	}
	if decision.Outcome != AdmissionAllow || !decision.Activated {
		t.Fatalf("decision = %#v, want allowed and active", decision)
	}
}
