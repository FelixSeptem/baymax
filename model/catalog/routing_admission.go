package catalog

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/FelixSeptem/baymax/adapter/capability"
)

const (
	RoutingAdmissionVersionV1  = "model_catalog_routing_admission.v1"
	RoutingAdmissionSourceHost = "host"

	MaxRoutingAdmissionCandidates       = 32
	MaxRoutingAdmissionCapabilities     = 32
	MaxRoutingAdmissionReasons          = 32
	MaxRoutingAdmissionIdentityLength   = 128
	MaxRoutingAdmissionGenerationLength = 128
	MaxRoutingAdmissionMetadataLength   = 128
	MaxRoutingAdmissionSerializedBytes  = 16 * 1024

	ReasonAuditUnsupportedVersion    = "provider.catalog.audit.unsupported_version"
	ReasonAuditUnsupportedSource     = "provider.catalog.audit.unsupported_source"
	ReasonAuditOverflow              = "provider.catalog.audit.overflow"
	ReasonAuditDuplicateCandidate    = "provider.catalog.audit.duplicate_candidate"
	ReasonAuditPriorityConflict      = "provider.catalog.audit.priority_conflict"
	ReasonAuditAmbiguousSelection    = "provider.catalog.audit.ambiguous_selection"
	ReasonAuditInvalidGeneration     = "provider.catalog.audit.invalid_generation"
	ReasonAuditPrivacyViolation      = "provider.catalog.audit.privacy_violation"
	ReasonAuditNoCandidates          = "provider.catalog.audit.no_candidates"
	ReasonAuditNoAdmissibleCandidate = "provider.catalog.audit.no_admissible_candidate"
)

// RoutingCandidate is a host-supplied candidate identity. Metadata is accepted
// only as a bounded, non-sensitive audit hint and is never emitted in the
// normalized representation.
type RoutingCandidate struct {
	Identity Identity `json:"identity"`
	Priority *int     `json:"priority,omitempty"`
	Metadata string   `json:"metadata,omitempty"`
}

// RoutingAdmissionInput is the provider-neutral, replayable audit input. The
// CatalogGeneration is supplied by the host and is intentionally distinct from
// Catalog.Version so an admitted result can retain the generation it observed.
type RoutingAdmissionInput struct {
	Version           string             `json:"version"`
	CatalogGeneration string             `json:"catalog_generation"`
	Source            string             `json:"source"`
	Candidates        []RoutingCandidate `json:"candidates"`
	Required          []string           `json:"required_capabilities,omitempty"`
	Optional          []string           `json:"optional_capabilities,omitempty"`
	Strategy          string             `json:"strategy,omitempty"`
}

// NormalizedRoutingAdmission is the canonical form used for replay and digest
// comparisons. It contains no endpoint, credential, provider response, prompt,
// or other unbounded payload.
type NormalizedRoutingAdmission struct {
	Version           string             `json:"version"`
	CatalogGeneration string             `json:"catalog_generation"`
	Source            string             `json:"source"`
	Candidates        []RoutingCandidate `json:"candidates"`
	Required          []string           `json:"required_capabilities,omitempty"`
	Optional          []string           `json:"optional_capabilities,omitempty"`
	Strategy          string             `json:"strategy"`
	CanonicalDigest   string             `json:"canonical_digest"`
}

// RoutingCandidateOutcome records the bounded result for a candidate that was
// evaluated but not selected by the conditional resolver.
type RoutingCandidateOutcome struct {
	Identity Identity `json:"identity"`
	Status   string   `json:"status"`
	Reasons  []string `json:"reasons,omitempty"`
}

// RoutingAdmissionResult is the bounded audit projection returned by an exact
// identity audit. More than one independently admissible candidate is reported
// as ambiguous; this intentionally does not activate a router.
type RoutingAdmissionResult struct {
	Version           string                    `json:"version"`
	CatalogGeneration string                    `json:"catalog_generation"`
	Source            string                    `json:"source"`
	Status            string                    `json:"status"`
	Selected          *Identity                 `json:"selected,omitempty"`
	Fallback          *Identity                 `json:"fallback,omitempty"`
	Capabilities      capability.Outcome        `json:"capabilities"`
	Credential        CredentialEvidence        `json:"credential"`
	Reasons           []string                  `json:"reasons,omitempty"`
	CandidateOrder    []Identity                `json:"candidate_order,omitempty"`
	CandidateOutcomes []RoutingCandidateOutcome `json:"candidate_outcomes,omitempty"`
	ResolverActivated bool                      `json:"resolver_activated,omitempty"`
	CanonicalDigest   string                    `json:"canonical_digest"`
}

// NormalizeRoutingAdmission validates and canonicalizes a host-supplied audit
// request without performing I/O or mutating runtime state.
func NormalizeRoutingAdmission(input RoutingAdmissionInput) (NormalizedRoutingAdmission, error) {
	version := strings.TrimSpace(input.Version)
	if version != RoutingAdmissionVersionV1 {
		return NormalizedRoutingAdmission{}, reasonError(ReasonAuditUnsupportedVersion, version)
	}
	generation := strings.TrimSpace(input.CatalogGeneration)
	if generation == "" || len(generation) > MaxRoutingAdmissionGenerationLength || containsControl(generation) {
		return NormalizedRoutingAdmission{}, reasonError(ReasonAuditInvalidGeneration, "catalog generation is empty or invalid")
	}
	source := strings.ToLower(strings.TrimSpace(input.Source))
	if source != RoutingAdmissionSourceHost {
		return NormalizedRoutingAdmission{}, reasonError(ReasonAuditUnsupportedSource, source)
	}
	if len(input.Candidates) > MaxRoutingAdmissionCandidates {
		return NormalizedRoutingAdmission{}, reasonError(ReasonAuditOverflow, "candidate count exceeds bound")
	}
	if len(input.Required) > MaxRoutingAdmissionCapabilities || len(input.Optional) > MaxRoutingAdmissionCapabilities {
		return NormalizedRoutingAdmission{}, reasonError(ReasonAuditOverflow, "capability count exceeds bound")
	}

	candidates := make([]RoutingCandidate, 0, len(input.Candidates))
	seen := make(map[string]struct{}, len(input.Candidates))
	priorities := make(map[int]struct{}, len(input.Candidates))
	for _, raw := range input.Candidates {
		identity := normalizeIdentity(raw.Identity)
		if identity.Provider == "" || identity.Model == "" || len(identity.Provider) > MaxRoutingAdmissionIdentityLength || len(identity.Model) > MaxRoutingAdmissionIdentityLength || containsControl(identity.Provider) || containsControl(identity.Model) {
			return NormalizedRoutingAdmission{}, reasonError(ReasonAuditOverflow, "candidate identity is empty or exceeds bound")
		}
		key := identityKey(identity)
		if _, exists := seen[key]; exists {
			return NormalizedRoutingAdmission{}, reasonError(ReasonAuditDuplicateCandidate, key)
		}
		seen[key] = struct{}{}
		if raw.Priority != nil {
			if *raw.Priority < 0 {
				return NormalizedRoutingAdmission{}, reasonError(ReasonAuditPriorityConflict, "priority must be non-negative")
			}
			if _, exists := priorities[*raw.Priority]; exists {
				return NormalizedRoutingAdmission{}, reasonError(ReasonAuditPriorityConflict, fmt.Sprintf("priority %d is not unique", *raw.Priority))
			}
			priorities[*raw.Priority] = struct{}{}
		}
		metadata := strings.TrimSpace(raw.Metadata)
		if len(metadata) > MaxRoutingAdmissionMetadataLength || containsControl(metadata) || looksSensitive(metadata) {
			return NormalizedRoutingAdmission{}, reasonError(ReasonAuditPrivacyViolation, "candidate metadata contains sensitive material")
		}
		candidate := RoutingCandidate{Identity: identity}
		if raw.Priority != nil {
			priority := *raw.Priority
			candidate.Priority = &priority
		}
		candidates = append(candidates, candidate)
	}
	sort.Slice(candidates, func(i, j int) bool {
		left, right := candidates[i], candidates[j]
		if left.Priority != nil && right.Priority == nil {
			return true
		}
		if left.Priority == nil && right.Priority != nil {
			return false
		}
		if left.Priority != nil && right.Priority != nil && *left.Priority != *right.Priority {
			return *left.Priority > *right.Priority
		}
		return identityKey(left.Identity) < identityKey(right.Identity)
	})

	normalized := NormalizedRoutingAdmission{
		Version:           version,
		CatalogGeneration: generation,
		Source:            source,
		Candidates:        candidates,
		Required:          normalizeCapabilities(input.Required),
		Optional:          normalizeCapabilities(input.Optional),
		Strategy:          strategyOrDefault(input.Strategy),
	}
	if len(normalized.Required) > MaxRoutingAdmissionCapabilities || len(normalized.Optional) > MaxRoutingAdmissionCapabilities {
		return NormalizedRoutingAdmission{}, reasonError(ReasonAuditOverflow, "normalized capability count exceeds bound")
	}
	canonical, err := json.Marshal(normalized)
	if err != nil {
		return NormalizedRoutingAdmission{}, reasonError(ReasonAuditOverflow, "canonical encoding failed")
	}
	if len(canonical) > MaxRoutingAdmissionSerializedBytes {
		return NormalizedRoutingAdmission{}, reasonError(ReasonAuditOverflow, "canonical audit size exceeds bound")
	}
	digest := sha256.Sum256(canonical)
	normalized.CanonicalDigest = hex.EncodeToString(digest[:])
	return normalized, nil
}

// AuditRoutingAdmission evaluates each supplied identity through the existing
// exact-identity admission path. A single admissible candidate is projected;
// multiple admissible candidates are an explicit, deterministic ambiguity.
func AuditRoutingAdmission(c Catalog, input RoutingAdmissionInput, credentials map[string]CredentialEvidence, strict bool) (RoutingAdmissionResult, error) {
	normalized, err := NormalizeRoutingAdmission(input)
	if err != nil {
		return RoutingAdmissionResult{}, err
	}
	result := RoutingAdmissionResult{
		Version:           normalized.Version,
		CatalogGeneration: normalized.CatalogGeneration,
		Source:            normalized.Source,
		Status:            StatusBlocked,
		CanonicalDigest:   normalized.CanonicalDigest,
	}
	for _, candidate := range normalized.Candidates {
		result.CandidateOrder = append(result.CandidateOrder, candidate.Identity)
	}
	if len(normalized.Candidates) == 0 {
		result.Reasons = []string{ReasonAuditNoCandidates}
		return result, nil
	}

	admissible := make([]Admission, 0, len(normalized.Candidates))
	for _, candidate := range normalized.Candidates {
		admission, evalErr := Evaluate(c, Request{
			Identity: candidate.Identity,
			Required: normalized.Required,
			Optional: normalized.Optional,
			Strategy: normalized.Strategy,
		}, credentials, strict)
		if evalErr != nil {
			return RoutingAdmissionResult{}, evalErr
		}
		if admission.Status != StatusBlocked {
			admissible = append(admissible, admission)
		}
	}
	if len(admissible) == 0 {
		result.Reasons = []string{ReasonAuditNoAdmissibleCandidate}
		return result, nil
	}
	if len(admissible) > 1 {
		result.Reasons = []string{ReasonAuditAmbiguousSelection}
		return result, nil
	}
	selected := admissible[0]
	result.Status = selected.Status
	result.Selected = identityPtr(selected.Selected)
	result.Fallback = identityPtrOrNil(selected.Fallback)
	result.Capabilities = selected.Capabilities
	result.Credential = redactCredentialEvidence(selected.Credential)
	result.Reasons = append([]string(nil), selected.Reasons...)
	if len(result.Reasons) > MaxRoutingAdmissionReasons {
		result.Reasons = result.Reasons[:MaxRoutingAdmissionReasons]
	}
	return result, nil
}

// ResolveRoutingAdmission is the conditional, pure candidate resolver. It is
// intentionally separate from AuditRoutingAdmission so callers must opt into
// the behavior after audit evidence has activated it. Candidate order is
// canonicalized before evaluation; the first independently admissible
// candidate wins and all skipped candidates retain bounded reasons.
func ResolveRoutingAdmission(c Catalog, input RoutingAdmissionInput, credentials map[string]CredentialEvidence, strict bool) (RoutingAdmissionResult, error) {
	normalized, err := NormalizeRoutingAdmission(input)
	if err != nil {
		return RoutingAdmissionResult{}, err
	}
	result := RoutingAdmissionResult{
		Version:           normalized.Version,
		CatalogGeneration: normalized.CatalogGeneration,
		Source:            normalized.Source,
		Status:            StatusBlocked,
		CanonicalDigest:   normalized.CanonicalDigest,
		ResolverActivated: true,
	}
	admissions := make([]Admission, len(normalized.Candidates))
	for _, candidate := range normalized.Candidates {
		result.CandidateOrder = append(result.CandidateOrder, candidate.Identity)
	}
	if len(normalized.Candidates) == 0 {
		result.Reasons = []string{ReasonAuditNoCandidates}
		return result, nil
	}

	for index, candidate := range normalized.Candidates {
		admission, evalErr := Evaluate(c, Request{
			Identity: candidate.Identity,
			Required: normalized.Required,
			Optional: normalized.Optional,
			Strategy: normalized.Strategy,
		}, credentials, strict)
		if evalErr != nil {
			return RoutingAdmissionResult{}, evalErr
		}
		admissions[index] = admission
	}

	selectedIndex := -1
	for index, admission := range admissions {
		if admission.Status != StatusBlocked {
			selectedIndex = index
			break
		}
	}
	if selectedIndex < 0 {
		for index, admission := range admissions {
			result.CandidateOutcomes = append(result.CandidateOutcomes, RoutingCandidateOutcome{
				Identity: normalized.Candidates[index].Identity,
				Status:   admission.Status,
				Reasons:  boundedReasons(admission.Reasons),
			})
		}
		result.Reasons = []string{ReasonAuditNoAdmissibleCandidate}
		return result, nil
	}

	selected := admissions[selectedIndex]
	for index, admission := range admissions {
		if index == selectedIndex {
			continue
		}
		result.CandidateOutcomes = append(result.CandidateOutcomes, RoutingCandidateOutcome{
			Identity: normalized.Candidates[index].Identity,
			Status:   admission.Status,
			Reasons:  boundedReasons(admission.Reasons),
		})
	}
	result.Status = selected.Status
	result.Selected = identityPtr(selected.Selected)
	result.Fallback = identityPtrOrNil(selected.Fallback)
	result.Capabilities = selected.Capabilities
	result.Credential = redactCredentialEvidence(selected.Credential)
	result.Reasons = boundedReasons(selected.Reasons)
	return result, nil
}

func identityPtr(identity Identity) *Identity {
	copy := identity
	return &copy
}

func identityPtrOrNil(identity *Identity) *Identity {
	if identity == nil {
		return nil
	}
	copy := *identity
	return &copy
}

func containsControl(value string) bool {
	for _, r := range value {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}

func looksSensitive(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return false
	}
	for _, marker := range []string{"http://", "https://", "endpoint", "credential", "password", "secret", "token", "sk-"} {
		if strings.Contains(value, marker) {
			return true
		}
	}
	return false
}

func boundedReasons(reasons []string) []string {
	if len(reasons) == 0 {
		return nil
	}
	if len(reasons) > MaxRoutingAdmissionReasons {
		reasons = reasons[:MaxRoutingAdmissionReasons]
	}
	out := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		reason = boundedReason(reason)
		if reason == "" {
			continue
		}
		out = append(out, reason)
	}
	return out
}

func redactCredentialEvidence(evidence CredentialEvidence) CredentialEvidence {
	evidence.Provider = normalizeIdentity(Identity{Provider: evidence.Provider}).Provider
	evidence.Status = strings.ToLower(strings.TrimSpace(evidence.Status))
	evidence.Reason = boundedReason(evidence.Reason)
	if looksSensitive(evidence.Reason) {
		switch evidence.Status {
		case CredentialAvailable:
			evidence.Reason = "credential.available"
		case CredentialUnverified:
			evidence.Reason = ReasonCredentialUnverified
		case CredentialInvalid:
			evidence.Reason = ReasonCredentialInvalid
		default:
			evidence.Reason = ReasonCredentialMissing
		}
	}
	return evidence
}

// ErrorCode returns the stable catalog classification for a validation or
// negotiation error. It is intended for offline replay and gate tooling; callers
// should not parse human-readable error text.
func ErrorCode(err error) string {
	if err == nil {
		return ""
	}
	if validation, ok := err.(validationError); ok {
		return validation.code
	}
	if negotiation, ok := err.(*capability.NegotiationError); ok {
		return negotiation.Code
	}
	message := err.Error()
	if strings.HasPrefix(message, "[") {
		if end := strings.IndexByte(message, ']'); end > 1 {
			return message[1:end]
		}
	}
	return ""
}
