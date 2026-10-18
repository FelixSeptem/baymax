package hostprojection

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAdmissionGapFixtureUsesSyntheticOpaqueReferences(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "admission_gap.v1.json"))
	if err != nil {
		t.Fatalf("read admission gap fixture: %v", err)
	}
	var fixture struct {
		Version string `json:"version"`
		Cases   []struct {
			CaseID          string `json:"case_id"`
			SchemaVersion   string `json:"schema_version"`
			SourceReference string `json:"source_reference"`
			Expectation     string `json:"expectation"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("decode admission gap fixture: %v", err)
	}
	if fixture.Version != "host_context_projection_admission_gap.v1" || len(fixture.Cases) != 2 {
		t.Fatalf("fixture = %#v, want versioned valid/invalid pair", fixture)
	}
	for _, item := range fixture.Cases {
		if !strings.HasPrefix(item.SourceReference, "opaque://") || strings.Contains(strings.ToLower(item.SourceReference), "body") {
			t.Fatalf("fixture case %q uses non-opaque source reference %q", item.CaseID, item.SourceReference)
		}
	}
}

func TestAdmitAcceptsDetachedCanonicalProjection(t *testing.T) {
	descriptor := validDescriptor(t)
	admitted, err := Admit(descriptor)
	if err != nil {
		t.Fatalf("Admit() error = %v", err)
	}

	descriptor.SourceReferences[0].Reference = "opaque://source/mutated-after-admission"
	descriptor.Capabilities[0] = CapabilityRead

	got := admitted.Descriptor()
	if got.SourceReferences[0].Reference != "opaque://source/alpha" {
		t.Fatalf("admitted reference = %q, want detached source reference", got.SourceReferences[0].Reference)
	}
	if got.Capabilities[0] != CapabilityReference {
		t.Fatalf("admitted capability = %q, want detached capability", got.Capabilities[0])
	}
	if got.Digest != descriptorDigest(t, got) {
		t.Fatalf("admitted digest = %q, want canonical digest", got.Digest)
	}
}

func TestAdmitRejectsMalformedDescriptorBeforeResolution(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Descriptor)
		code   Code
	}{
		{
			name: "unknown schema",
			mutate: func(d *Descriptor) {
				d.SchemaVersion = "host_context_projection.v2"
			},
			code: CodeUnsupportedVersion,
		},
		{
			name: "missing identity",
			mutate: func(d *Descriptor) {
				d.ProjectionID = ""
			},
			code: CodeInvalidDescriptor,
		},
		{
			name: "duplicate reference",
			mutate: func(d *Descriptor) {
				d.SourceReferences = append(d.SourceReferences, d.SourceReferences[0])
			},
			code: CodeDuplicateReference,
		},
		{
			name: "oversized reference",
			mutate: func(d *Descriptor) {
				d.SourceReferences[0].Reference = strings.Repeat("r", DefaultLimits().MaxReferenceBytes+1)
			},
			code: CodeLimitExceeded,
		},
		{
			name: "invalid capability",
			mutate: func(d *Descriptor) {
				d.Capabilities = []Capability{"host_context.write"}
			},
			code: CodeInvalidCapability,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			descriptor := validDescriptor(t)
			tc.mutate(&descriptor)
			_, err := Admit(descriptor)
			if err == nil {
				t.Fatal("Admit() error = nil, want rejection")
			}
			classified, ok := err.(*Error)
			if !ok || classified.Code != tc.code {
				t.Fatalf("Admit() error = %#v, want code %q", err, tc.code)
			}
		})
	}
}

func TestAdmitAcceptsExactContractBounds(t *testing.T) {
	descriptor := validDescriptor(t)
	limits := DefaultLimits()
	descriptor.SourceReferences = make([]SourceReference, limits.MaxReferences)
	for index := range descriptor.SourceReferences {
		prefix := fmt.Sprintf("opaque://source/%02d-", index)
		descriptor.SourceReferences[index] = SourceReference{
			Reference: prefix + strings.Repeat("r", limits.MaxReferenceBytes-len(prefix)),
			Digest:    strings.Repeat("d", limits.MaxDigestBytes),
		}
	}
	descriptor.CapabilityDigest = strings.Repeat("c", limits.MaxDigestBytes)
	descriptor.Digest = descriptorDigest(t, descriptor)

	if _, err := Admit(descriptor); err != nil {
		t.Fatalf("Admit() exact bounds error = %v", err)
	}
}

func validDescriptor(t *testing.T) Descriptor {
	t.Helper()
	descriptor := Descriptor{
		SchemaVersion:     SchemaVersionV1,
		ProjectionID:      "projection-alpha",
		ProjectionVersion: 1,
		CapabilityDigest:  "capability-digest-alpha",
		Capabilities:      []Capability{CapabilityReference},
		SourceReferences: []SourceReference{{
			Reference: "opaque://source/alpha",
			Digest:    "source-digest-alpha",
		}},
	}
	descriptor.Digest = descriptorDigest(t, descriptor)
	return descriptor
}

func descriptorDigest(t *testing.T, descriptor Descriptor) string {
	t.Helper()
	digest, err := CanonicalDigest(descriptor)
	if err != nil {
		t.Fatalf("CanonicalDigest() error = %v", err)
	}
	return digest
}
