// Package hostprojection defines the bounded, reference-only host context
// projection contract. It deliberately owns neither source bodies nor current
// source authorization; those remain with the host resolver.
package hostprojection

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

const SchemaVersionV1 = "host_context_projection.v1"

// Capability is a finite declaration of the access the host already filtered
// before requesting context assembly. It is never an authorization grant.
type Capability string

const (
	CapabilityRead      Capability = "host_context.read"
	CapabilityReference Capability = "host_context.reference"
)

// SourceReference is opaque host-owned source evidence. Reference and Digest
// must not contain source bodies, credentials, effects, or transcripts.
type SourceReference struct {
	Reference string `json:"reference"`
	Digest    string `json:"digest"`
}

// ParentCapabilityEvidence is immutable evidence supplied for a child
// projection. It can only establish that child capabilities narrow a parent;
// it cannot grant authorization or resume any parent operation.
type ParentCapabilityEvidence struct {
	ProjectionID     string       `json:"projection_id"`
	CapabilityDigest string       `json:"capability_digest"`
	Capabilities     []Capability `json:"capabilities"`
}

// PlanCorrelation is a detached identity reference to a host plan. This
// package deliberately does not import planbridge and cannot compile,
// dispatch, authorize, or resume the referenced plan.
type PlanCorrelation struct {
	PlanID      string `json:"plan_id"`
	PlanVersion int    `json:"plan_version"`
	Digest      string `json:"digest"`
}

// Descriptor is the explicit opt-in input for a host-filtered context
// projection. Its digest covers the canonical form excluding Digest itself.
type Descriptor struct {
	SchemaVersion     string                    `json:"schema_version"`
	ProjectionID      string                    `json:"projection_id"`
	ProjectionVersion int                       `json:"projection_version"`
	Digest            string                    `json:"digest"`
	CapabilityDigest  string                    `json:"capability_digest"`
	Capabilities      []Capability              `json:"capabilities"`
	SourceReferences  []SourceReference         `json:"source_references"`
	Parent            *ParentCapabilityEvidence `json:"parent,omitempty"`
	Plan              *PlanCorrelation          `json:"plan,omitempty"`
}

// Limits are constructor-only contract limits. They are not configuration
// values and no environment, file, or hot-update surface may change them.
type Limits struct {
	MaxSerializedBytes int
	MaxReferences      int
	MaxReferenceBytes  int
	MaxDigestBytes     int
}

func DefaultLimits() Limits {
	return Limits{
		MaxSerializedBytes: 32 * 1024,
		MaxReferences:      64,
		MaxReferenceBytes:  256,
		MaxDigestBytes:     128,
	}
}

// Code is a stable bounded descriptor-validation classification.
type Code string

const (
	CodeUnsupportedVersion     Code = "host_context.unsupported_version"
	CodeInvalidDescriptor      Code = "host_context.invalid_descriptor"
	CodeLimitExceeded          Code = "host_context.limit_exceeded"
	CodeDuplicateReference     Code = "host_context.duplicate_reference"
	CodeInvalidCapability      Code = "host_context.invalid_capability"
	CodeDigestMismatch         Code = "host_context.digest_mismatch"
	CodeResolverDenied         Code = "host_context.resolver_denied"
	CodeSourceNotFound         Code = "host_context.source_not_found"
	CodeCapabilityExpansion    Code = "host_context.capability_expansion"
	CodeParentEvidenceMissing  Code = "host_context.parent_evidence_missing"
	CodeInvalidPlanCorrelation Code = "host_context.invalid_plan_correlation"
)

// Error intentionally exposes only classification and field, never source
// material or callback error text.
type Error struct {
	Code  Code
	Field string
}

func (e *Error) Error() string {
	return fmt.Sprintf("host context projection: %s at %s", e.Code, e.Field)
}

func fail(code Code, field string) error { return &Error{Code: code, Field: field} }

// Snapshot is an immutable detached projection. It returns copies so neither
// caller mutation nor a subsequent consumer mutation alters admitted facts.
type Snapshot struct {
	descriptor Descriptor
	canonical  []byte
}

func (s Snapshot) Descriptor() Descriptor { return cloneDescriptor(s.descriptor) }
func (s Snapshot) Canonical() []byte      { return append([]byte(nil), s.canonical...) }

// Admit verifies a descriptor and returns its detached canonical snapshot.
// It has no resolver, provider, tool, journal, or diagnostic dependency, so a
// malformed descriptor cannot trigger an external side effect here.
func Admit(descriptor Descriptor) (Snapshot, error) {
	normalized, canonical, err := canonicalize(descriptor, DefaultLimits())
	if err != nil {
		return Snapshot{}, err
	}
	if normalized.Digest == "" || len(normalized.Digest) > DefaultLimits().MaxDigestBytes {
		return Snapshot{}, fail(CodeInvalidDescriptor, "digest")
	}
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return Snapshot{}, fail(CodeInvalidDescriptor, "descriptor")
	}
	if len(encoded) > DefaultLimits().MaxSerializedBytes {
		return Snapshot{}, fail(CodeLimitExceeded, "descriptor.bytes")
	}
	expected := digest(canonical)
	if normalized.Digest != expected {
		return Snapshot{}, fail(CodeDigestMismatch, "digest")
	}
	if err := validateParentEvidence(normalized); err != nil {
		return Snapshot{}, err
	}
	if err := validatePlanCorrelation(normalized.Plan, DefaultLimits()); err != nil {
		return Snapshot{}, err
	}
	return Snapshot{descriptor: cloneDescriptor(normalized), canonical: append([]byte(nil), canonical...)}, nil
}

// CanonicalDigest returns the SHA-256 digest hosts put in Descriptor.Digest.
// It validates every immutable contract field except the self-referential
// Digest field, which is intentionally excluded from the canonical form.
func CanonicalDigest(descriptor Descriptor) (string, error) {
	_, canonical, err := canonicalize(descriptor, DefaultLimits())
	if err != nil {
		return "", err
	}
	return digest(canonical), nil
}

func canonicalize(descriptor Descriptor, limits Limits) (Descriptor, []byte, error) {
	normalized := cloneDescriptor(descriptor)
	normalized.SchemaVersion = strings.TrimSpace(normalized.SchemaVersion)
	normalized.ProjectionID = strings.TrimSpace(normalized.ProjectionID)
	normalized.Digest = strings.TrimSpace(normalized.Digest)
	normalized.CapabilityDigest = strings.TrimSpace(normalized.CapabilityDigest)

	if normalized.SchemaVersion != SchemaVersionV1 {
		return Descriptor{}, nil, fail(CodeUnsupportedVersion, "schema_version")
	}
	if normalized.ProjectionID == "" || normalized.ProjectionVersion <= 0 {
		return Descriptor{}, nil, fail(CodeInvalidDescriptor, "projection_identity")
	}
	if len(normalized.CapabilityDigest) == 0 || len(normalized.CapabilityDigest) > limits.MaxDigestBytes {
		return Descriptor{}, nil, fail(CodeInvalidDescriptor, "capability_digest")
	}
	if len(normalized.SourceReferences) == 0 {
		return Descriptor{}, nil, fail(CodeInvalidDescriptor, "source_references")
	}
	if len(normalized.SourceReferences) > limits.MaxReferences {
		return Descriptor{}, nil, fail(CodeLimitExceeded, "source_references")
	}
	if len(normalized.Capabilities) == 0 {
		return Descriptor{}, nil, fail(CodeInvalidCapability, "capabilities")
	}

	seenReferences := make(map[string]struct{}, len(normalized.SourceReferences))
	for index := range normalized.SourceReferences {
		ref := &normalized.SourceReferences[index]
		ref.Reference = strings.TrimSpace(ref.Reference)
		ref.Digest = strings.TrimSpace(ref.Digest)
		if ref.Reference == "" || ref.Digest == "" {
			return Descriptor{}, nil, fail(CodeInvalidDescriptor, "source_references")
		}
		if len(ref.Reference) > limits.MaxReferenceBytes || len(ref.Digest) > limits.MaxDigestBytes {
			return Descriptor{}, nil, fail(CodeLimitExceeded, "source_references")
		}
		if _, exists := seenReferences[ref.Reference]; exists {
			return Descriptor{}, nil, fail(CodeDuplicateReference, "source_references")
		}
		seenReferences[ref.Reference] = struct{}{}
	}
	sort.Slice(normalized.SourceReferences, func(i, j int) bool {
		return normalized.SourceReferences[i].Reference < normalized.SourceReferences[j].Reference
	})

	seenCapabilities := make(map[Capability]struct{}, len(normalized.Capabilities))
	for index := range normalized.Capabilities {
		capability := Capability(strings.TrimSpace(string(normalized.Capabilities[index])))
		if !validCapability(capability) {
			return Descriptor{}, nil, fail(CodeInvalidCapability, "capabilities")
		}
		if _, exists := seenCapabilities[capability]; exists {
			return Descriptor{}, nil, fail(CodeInvalidCapability, "capabilities")
		}
		seenCapabilities[capability] = struct{}{}
		normalized.Capabilities[index] = capability
	}
	sort.Slice(normalized.Capabilities, func(i, j int) bool { return normalized.Capabilities[i] < normalized.Capabilities[j] })
	normalizeOptionalEvidence(&normalized)

	canonical, err := json.Marshal(struct {
		SchemaVersion     string                    `json:"schema_version"`
		ProjectionID      string                    `json:"projection_id"`
		ProjectionVersion int                       `json:"projection_version"`
		CapabilityDigest  string                    `json:"capability_digest"`
		Capabilities      []Capability              `json:"capabilities"`
		SourceReferences  []SourceReference         `json:"source_references"`
		Parent            *ParentCapabilityEvidence `json:"parent,omitempty"`
		Plan              *PlanCorrelation          `json:"plan,omitempty"`
	}{
		SchemaVersion:     normalized.SchemaVersion,
		ProjectionID:      normalized.ProjectionID,
		ProjectionVersion: normalized.ProjectionVersion,
		CapabilityDigest:  normalized.CapabilityDigest,
		Capabilities:      normalized.Capabilities,
		SourceReferences:  normalized.SourceReferences,
		Parent:            normalized.Parent,
		Plan:              normalized.Plan,
	})
	if err != nil {
		return Descriptor{}, nil, fail(CodeInvalidDescriptor, "descriptor")
	}
	if len(canonical) > limits.MaxSerializedBytes {
		return Descriptor{}, nil, fail(CodeLimitExceeded, "descriptor.bytes")
	}
	return normalized, canonical, nil
}

func validCapability(capability Capability) bool {
	switch capability {
	case CapabilityRead, CapabilityReference:
		return true
	default:
		return false
	}
}

func digest(canonical []byte) string {
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func cloneDescriptor(descriptor Descriptor) Descriptor {
	out := descriptor
	out.Capabilities = append([]Capability(nil), descriptor.Capabilities...)
	out.SourceReferences = append([]SourceReference(nil), descriptor.SourceReferences...)
	if descriptor.Parent != nil {
		parent := *descriptor.Parent
		parent.Capabilities = append([]Capability(nil), descriptor.Parent.Capabilities...)
		out.Parent = &parent
	}
	if descriptor.Plan != nil {
		plan := *descriptor.Plan
		out.Plan = &plan
	}
	return out
}

func normalizeOptionalEvidence(descriptor *Descriptor) {
	if descriptor == nil {
		return
	}
	if descriptor.Parent != nil {
		descriptor.Parent.ProjectionID = strings.TrimSpace(descriptor.Parent.ProjectionID)
		descriptor.Parent.CapabilityDigest = strings.TrimSpace(descriptor.Parent.CapabilityDigest)
		for index := range descriptor.Parent.Capabilities {
			descriptor.Parent.Capabilities[index] = Capability(strings.TrimSpace(string(descriptor.Parent.Capabilities[index])))
		}
		sort.Slice(descriptor.Parent.Capabilities, func(i, j int) bool {
			return descriptor.Parent.Capabilities[i] < descriptor.Parent.Capabilities[j]
		})
	}
	if descriptor.Plan != nil {
		descriptor.Plan.PlanID = strings.TrimSpace(descriptor.Plan.PlanID)
		descriptor.Plan.Digest = strings.TrimSpace(descriptor.Plan.Digest)
	}
}

func validateParentEvidence(descriptor Descriptor) error {
	if descriptor.Parent == nil {
		return nil
	}
	parent := descriptor.Parent
	if parent.ProjectionID == "" || parent.CapabilityDigest == "" || len(parent.CapabilityDigest) > DefaultLimits().MaxDigestBytes || len(parent.Capabilities) == 0 {
		return fail(CodeParentEvidenceMissing, "parent")
	}
	parentCapabilities := make(map[Capability]struct{}, len(parent.Capabilities))
	for index := range parent.Capabilities {
		capability := parent.Capabilities[index]
		if !validCapability(capability) {
			return fail(CodeParentEvidenceMissing, "parent.capabilities")
		}
		if _, exists := parentCapabilities[capability]; exists {
			return fail(CodeParentEvidenceMissing, "parent.capabilities")
		}
		parentCapabilities[capability] = struct{}{}
	}
	for _, capability := range descriptor.Capabilities {
		if _, allowed := parentCapabilities[capability]; !allowed {
			return fail(CodeCapabilityExpansion, "capabilities")
		}
	}
	return nil
}

func validatePlanCorrelation(plan *PlanCorrelation, limits Limits) error {
	if plan == nil {
		return nil
	}
	if plan.PlanID == "" || plan.PlanVersion <= 0 || plan.Digest == "" || len(plan.PlanID) > limits.MaxReferenceBytes || len(plan.Digest) > limits.MaxDigestBytes {
		return fail(CodeInvalidPlanCorrelation, "plan")
	}
	return nil
}

// ErrSourceNotFound lets a host resolver report a missing or no-longer
// authorized opaque source without exposing source details.
var ErrSourceNotFound = errors.New("host context source not found")

// Material is transient, current host-authorized content. It is never
// serialized, added to a handoff, or retained by a Snapshot.
type Material struct {
	Role    string
	Content string
}

// Resolver remains host-owned. It rechecks current access each time after
// descriptor admission; a matching historical digest is never authorization.
type Resolver interface {
	ResolveHostContext(context.Context, SourceReference) (Material, error)
}

// Resolution contains only ephemeral materials for the immediate assembly
// path. The private slice prevents accidental durable or mutable retention.
type Resolution struct{ materials []Material }

// Materials returns a defensive copy for the existing assembler bridge.
func (r Resolution) Materials() []Material { return append([]Material(nil), r.materials...) }

// Resolve admits a descriptor before asking the host for every currently
// authorized source. It fails closed: an error returns no partial material and
// maps all host callback details to a finite safe classification.
func Resolve(ctx context.Context, descriptor Descriptor, resolver Resolver) (Snapshot, Resolution, error) {
	snapshot, err := Admit(descriptor)
	if err != nil {
		return Snapshot{}, Resolution{}, err
	}
	if resolver == nil {
		return Snapshot{}, Resolution{}, fail(CodeResolverDenied, "resolver")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	refs := snapshot.Descriptor().SourceReferences
	materials := make([]Material, 0, len(refs))
	for _, reference := range refs {
		material, resolveErr := resolver.ResolveHostContext(ctx, reference)
		if resolveErr != nil {
			if errors.Is(resolveErr, ErrSourceNotFound) {
				return Snapshot{}, Resolution{}, fail(CodeSourceNotFound, "source_references")
			}
			return Snapshot{}, Resolution{}, fail(CodeResolverDenied, "source_references")
		}
		if strings.TrimSpace(material.Content) == "" {
			return Snapshot{}, Resolution{}, fail(CodeResolverDenied, "source_references")
		}
		materials = append(materials, Material{Role: strings.TrimSpace(material.Role), Content: material.Content})
	}
	return snapshot, Resolution{materials: materials}, nil
}
