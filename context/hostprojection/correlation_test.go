package hostprojection

import "testing"

func TestAdmitAcceptsChildCapabilityNarrowingAndPlanCorrelation(t *testing.T) {
	descriptor := validDescriptor(t)
	descriptor.Capabilities = []Capability{CapabilityReference}
	descriptor.Parent = &ParentCapabilityEvidence{
		ProjectionID:     "parent-projection",
		CapabilityDigest: "parent-capability-digest",
		Capabilities:     []Capability{CapabilityRead, CapabilityReference},
	}
	descriptor.Plan = &PlanCorrelation{PlanID: "plan-alpha", PlanVersion: 3, Digest: "plan-digest-alpha"}
	descriptor.Digest = descriptorDigest(t, descriptor)

	admitted, err := Admit(descriptor)
	if err != nil {
		t.Fatalf("Admit() error = %v", err)
	}
	got := admitted.Descriptor()
	if got.Plan == nil || got.Plan.PlanID != "plan-alpha" {
		t.Fatalf("plan correlation = %#v, want immutable evidence", got.Plan)
	}
	if got.Parent == nil || len(got.Parent.Capabilities) != 2 {
		t.Fatalf("parent evidence = %#v, want detached parent evidence", got.Parent)
	}
}

func TestAdmitRejectsCapabilityExpansionAndMissingParentEvidence(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Descriptor)
		code   Code
	}{
		{
			name: "capability expansion",
			mutate: func(d *Descriptor) {
				d.Capabilities = []Capability{CapabilityRead}
				d.Parent = &ParentCapabilityEvidence{
					ProjectionID:     "parent-projection",
					CapabilityDigest: "parent-capability-digest",
					Capabilities:     []Capability{CapabilityReference},
				}
			},
			code: CodeCapabilityExpansion,
		},
		{
			name: "missing parent capabilities",
			mutate: func(d *Descriptor) {
				d.Parent = &ParentCapabilityEvidence{ProjectionID: "parent-projection", CapabilityDigest: "parent-capability-digest"}
			},
			code: CodeParentEvidenceMissing,
		},
		{
			name: "incomplete plan correlation",
			mutate: func(d *Descriptor) {
				d.Plan = &PlanCorrelation{PlanID: "plan-alpha", PlanVersion: 0}
			},
			code: CodeInvalidPlanCorrelation,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			descriptor := validDescriptor(t)
			tc.mutate(&descriptor)
			descriptor.Digest = descriptorDigest(t, descriptor)
			_, err := Admit(descriptor)
			if err == nil {
				t.Fatal("Admit() error = nil, want rejection")
			}
			if got := errorCode(t, err); got != tc.code {
				t.Fatalf("Admit() code = %q, want %q", got, tc.code)
			}
		})
	}
}

func TestAdmitChecksDigestBeforeCapabilitySubset(t *testing.T) {
	descriptor := validDescriptor(t)
	descriptor.Capabilities = []Capability{CapabilityRead}
	descriptor.Parent = &ParentCapabilityEvidence{
		ProjectionID:     "parent-projection",
		CapabilityDigest: "parent-capability-digest",
		Capabilities:     []Capability{CapabilityReference},
	}
	// Keep the old digest deliberately: both digest and subset are invalid, so
	// admission must report digest drift before examining parent semantics.

	_, err := Admit(descriptor)
	if err == nil {
		t.Fatal("Admit() error = nil, want digest rejection")
	}
	if got := errorCode(t, err); got != CodeDigestMismatch {
		t.Fatalf("Admit() code = %q, want %q", got, CodeDigestMismatch)
	}
}
