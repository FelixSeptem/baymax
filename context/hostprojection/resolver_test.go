package hostprojection

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

const (
	testMaterialRoleLimit    = 64
	testMaterialContentLimit = 64 * 1024
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

func TestResolveRejectsInvalidOrOverLimitMaterialAsWholeResolution(t *testing.T) {
	tests := []struct {
		name       string
		descriptor Descriptor
		materials  []Material
		wantCode   Code
	}{
		{
			name:       "empty role",
			descriptor: validDescriptor(t),
			materials:  []Material{{Content: "authorized"}},
			wantCode:   CodeLimitExceeded,
		},
		{
			name:       "role contains invalid UTF-8",
			descriptor: validDescriptor(t),
			materials:  []Material{{Role: string([]byte{0xff}), Content: "authorized"}},
			wantCode:   CodeLimitExceeded,
		},
		{
			name:       "content contains invalid UTF-8",
			descriptor: validDescriptor(t),
			materials:  []Material{{Role: "system", Content: string([]byte{0xff})}},
			wantCode:   CodeLimitExceeded,
		},
		{
			name:       "role exceeds fixed bound",
			descriptor: validDescriptor(t),
			materials: []Material{{
				Role:    strings.Repeat("r", testMaterialRoleLimit+1),
				Content: "authorized",
			}},
			wantCode: CodeLimitExceeded,
		},
		{
			name:       "role exceeds fixed bound before trimming",
			descriptor: validDescriptor(t),
			materials:  []Material{{Role: strings.Repeat(" ", testMaterialRoleLimit) + "system", Content: "authorized"}},
			wantCode:   CodeLimitExceeded,
		},
		{
			name:       "content exceeds fixed bound",
			descriptor: validDescriptor(t),
			materials: []Material{{
				Role:    "system",
				Content: strings.Repeat("c", testMaterialContentLimit+1),
			}},
			wantCode: CodeLimitExceeded,
		},
		{
			name:       "aggregate exceeds fixed bound",
			descriptor: descriptorWithReferences(t, 4),
			materials: []Material{
				{Role: "s", Content: strings.Repeat("a", testMaterialContentLimit)},
				{Role: "s", Content: strings.Repeat("b", testMaterialContentLimit)},
				{Role: "s", Content: strings.Repeat("c", testMaterialContentLimit)},
				{Role: "s", Content: strings.Repeat("d", testMaterialContentLimit)},
			},
			wantCode: CodeLimitExceeded,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resolver := &sequenceResolver{materials: tc.materials}

			_, resolution, err := Resolve(t.Context(), tc.descriptor, resolver)
			if err == nil {
				t.Fatal("Resolve() error = nil, want material-bound rejection")
			}
			if got := errorCode(t, err); got != tc.wantCode {
				t.Fatalf("Resolve() code = %q, want %q", got, tc.wantCode)
			}
			if got := resolution.Materials(); len(got) != 0 {
				t.Fatalf("Resolve() returned partial materials = %#v", got)
			}
		})
	}
}

func TestResolveAcceptsExactAggregateMaterialBound(t *testing.T) {
	contents := []string{
		strings.Repeat("a", testMaterialContentLimit-4),
		strings.Repeat("b", testMaterialContentLimit),
		strings.Repeat("c", testMaterialContentLimit),
		strings.Repeat("d", testMaterialContentLimit),
	}
	materials := make([]Material, len(contents))
	for i, content := range contents {
		materials[i] = Material{Role: "s", Content: content}
	}

	_, resolution, err := Resolve(t.Context(), descriptorWithReferences(t, len(materials)), &sequenceResolver{materials: materials})
	if err != nil {
		t.Fatalf("Resolve() exact aggregate bound error = %v", err)
	}
	if got := resolution.Materials(); len(got) != len(materials) {
		t.Fatalf("materials = %d, want %d", len(got), len(materials))
	}
}

func TestResolveAcceptsExactPerMaterialBoundsMeasuredInUTF8Bytes(t *testing.T) {
	role := strings.Repeat("r", testMaterialRoleLimit-3) + "你"
	content := strings.Repeat("c", testMaterialContentLimit)
	_, resolution, err := Resolve(t.Context(), validDescriptor(t), &sequenceResolver{materials: []Material{{Role: role, Content: content}}})
	if err != nil {
		t.Fatalf("Resolve() exact per-material bounds error = %v", err)
	}
	if got := resolution.Materials(); len(got) != 1 || len(got[0].Role) != testMaterialRoleLimit || len(got[0].Content) != testMaterialContentLimit {
		t.Fatalf("materials = %#v, want exact byte bounds", got)
	}
}

type countingResolver struct {
	calls    int
	material Material
	err      error
}

type sequenceResolver struct {
	materials []Material
	calls     int
}

func (r *sequenceResolver) ResolveHostContext(_ context.Context, _ SourceReference) (Material, error) {
	if r.calls >= len(r.materials) {
		return Material{}, ErrSourceNotFound
	}
	material := r.materials[r.calls]
	r.calls++
	return material, nil
}

func descriptorWithReferences(t *testing.T, count int) Descriptor {
	t.Helper()
	descriptor := validDescriptor(t)
	descriptor.SourceReferences = make([]SourceReference, count)
	for i := range descriptor.SourceReferences {
		descriptor.SourceReferences[i] = SourceReference{
			Reference: fmt.Sprintf("opaque://source/%d", i),
			Digest:    fmt.Sprintf("source-digest-%d", i),
		}
	}
	descriptor.Digest = descriptorDigest(t, descriptor)
	return descriptor
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
