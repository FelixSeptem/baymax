package hostprojection

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestResolveRejectsInvalidDescriptorBeforeInvokingHostResolver(t *testing.T) {
	descriptor := validDescriptor(t)
	descriptor.SchemaVersion = "host_context_projection.v2"
	resolver := &countingResolver{}

	_, _, err := Resolve(context.Background(), descriptor, resolver)
	if err == nil {
		t.Fatal("Resolve() error = nil, want rejection")
	}
	if resolver.calls != 0 {
		t.Fatalf("resolver calls = %d, want no side effect before rejection", resolver.calls)
	}
	if got := errorCode(t, err); got != CodeUnsupportedVersion {
		t.Fatalf("Resolve() code = %q, want %q", got, CodeUnsupportedVersion)
	}
}

func TestResolveReturnsOnlyBoundedClassificationForHostFailure(t *testing.T) {
	descriptor := validDescriptor(t)
	resolver := &countingResolver{err: errors.New("host callback exposed private body: secret-source-body")}

	_, _, err := Resolve(context.Background(), descriptor, resolver)
	if err == nil {
		t.Fatal("Resolve() error = nil, want resolver rejection")
	}
	if resolver.calls != 1 {
		t.Fatalf("resolver calls = %d, want 1 after valid admission", resolver.calls)
	}
	if got := errorCode(t, err); got != CodeResolverDenied {
		t.Fatalf("Resolve() code = %q, want %q", got, CodeResolverDenied)
	}
	if strings.Contains(err.Error(), "secret-source-body") {
		t.Fatalf("Resolve() leaked callback error: %v", err)
	}
}

func TestResolveFailsClosedForMissingSource(t *testing.T) {
	descriptor := validDescriptor(t)
	resolver := &countingResolver{err: ErrSourceNotFound}

	_, resolution, err := Resolve(context.Background(), descriptor, resolver)
	if err == nil {
		t.Fatal("Resolve() error = nil, want missing-source rejection")
	}
	if got := errorCode(t, err); got != CodeSourceNotFound {
		t.Fatalf("Resolve() code = %q, want %q", got, CodeSourceNotFound)
	}
	if materials := resolution.Materials(); len(materials) != 0 {
		t.Fatalf("partial materials = %#v, want none", materials)
	}
}

func TestResolveReturnsCurrentAuthorizedMaterialAfterAdmission(t *testing.T) {
	descriptor := validDescriptor(t)
	resolver := &countingResolver{material: Material{Role: "system", Content: "current host-authorized context"}}

	snapshot, resolution, err := Resolve(context.Background(), descriptor, resolver)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if snapshot.Descriptor().ProjectionID != descriptor.ProjectionID {
		t.Fatalf("snapshot identity = %q, want %q", snapshot.Descriptor().ProjectionID, descriptor.ProjectionID)
	}
	if got := resolution.Materials(); len(got) != 1 || got[0].Content != "current host-authorized context" {
		t.Fatalf("materials = %#v, want current authorized material", got)
	}
}

type countingResolver struct {
	calls    int
	material Material
	err      error
}

func (r *countingResolver) ResolveHostContext(_ context.Context, _ SourceReference) (Material, error) {
	r.calls++
	if r.err != nil {
		return Material{}, r.err
	}
	return r.material, nil
}

func errorCode(t *testing.T, err error) Code {
	t.Helper()
	classified, ok := err.(*Error)
	if !ok {
		t.Fatalf("error = %#v, want *Error", err)
	}
	return classified.Code
}
