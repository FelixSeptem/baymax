package catalog

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
)

const (
	RouteIntentAdmissionVersionV1 = "model_route_intent_admission.v1"

	RouteIntentVerdictSatisfied            = "satisfied"
	RouteIntentVerdictRouteGapConfirmed    = "route-gap-confirmed"
	RouteIntentVerdictInsufficientEvidence = "insufficient-evidence"

	ReasonRouteIntentUnsupportedVersion        = "model.route_intent.unsupported_version"
	ReasonRouteIntentOverflow                  = "model.route_intent.overflow"
	ReasonRouteIntentPrivacyViolation          = "model.route_intent.privacy_violation"
	ReasonRouteIntentMissingIntent             = "model.route_intent.missing_intent"
	ReasonRouteIntentMissingFacts              = "model.route_intent.missing_facts"
	ReasonRouteIntentGenerationMismatch        = "model.route_intent.generation_mismatch"
	ReasonRouteIntentAdmissionBlocked          = "model.route_intent.admission_blocked"
	ReasonRouteIntentUnknownStatus             = "model.route_intent.unknown_status"
	ReasonRouteIntentTargetNotSelected         = "model.route_intent.target_not_selected"
	ReasonRouteIntentCandidateNotAllowed       = "model.route_intent.candidate_not_allowed"
	ReasonRouteIntentCapabilityEvidenceMissing = "model.route_intent.capability_evidence_missing"
	ReasonRouteIntentCapabilityGap             = "model.route_intent.capability_gap"
	ReasonRouteIntentCredentialGap             = "model.route_intent.credential_gap"
	ReasonRouteIntentParityMissing             = "model.route_intent.parity_missing"
	ReasonRouteIntentParityMismatch            = "model.route_intent.parity_mismatch"
)

const (
	MaxRouteIntentAllowedCandidates = 32
	MaxRouteIntentCapabilities      = 32
	MaxRouteIntentReasons           = 32
	MaxRouteIntentIdentityLength    = 128
	MaxRouteIntentGenerationLength  = 128
	MaxRouteIntentSerializedBytes   = 16 * 1024
)

// RouteIntent describes an explicit host routing requirement. It is an
// evidence input only; it never selects a model or mutates catalog state.
type RouteIntent struct {
	Target                    *Identity  `json:"target,omitempty"`
	Allowed                   []Identity `json:"allowed,omitempty"`
	Required                  []string   `json:"required_capabilities,omitempty"`
	Optional                  []string   `json:"optional_capabilities,omitempty"`
	ExpectedCatalogGeneration string     `json:"expected_catalog_generation,omitempty"`
	RequireRunStreamParity    bool       `json:"require_run_stream_parity,omitempty"`
}

// RouteIntentAdmissionSnapshot is a bounded admission projection used for
// Run/Stream parity comparison.
type RouteIntentAdmissionSnapshot struct {
	CatalogGeneration    string    `json:"catalog_generation"`
	Status               string    `json:"status"`
	Selected             *Identity `json:"selected,omitempty"`
	Fallback             *Identity `json:"fallback,omitempty"`
	Reasons              []string  `json:"reasons,omitempty"`
	CapabilitiesAccepted *bool     `json:"capabilities_accepted,omitempty"`
	CredentialStatus     string    `json:"credential_status,omitempty"`
}

// RouteIntentAdmissionFacts are existing catalog/admission facts supplied by
// the host. They are references, not a second source of runtime truth.
type RouteIntentAdmissionFacts struct {
	CatalogGeneration      string                        `json:"catalog_generation"`
	Status                 string                        `json:"status"`
	Selected               *Identity                     `json:"selected,omitempty"`
	Fallback               *Identity                     `json:"fallback,omitempty"`
	Reasons                []string                      `json:"reasons,omitempty"`
	CapabilitiesAccepted   *bool                         `json:"capabilities_accepted,omitempty"`
	CapabilitiesDowngraded bool                          `json:"capabilities_downgraded,omitempty"`
	CapabilityReasons      []string                      `json:"capability_reasons,omitempty"`
	CredentialStatus       string                        `json:"credential_status,omitempty"`
	Run                    *RouteIntentAdmissionSnapshot `json:"run,omitempty"`
	Stream                 *RouteIntentAdmissionSnapshot `json:"stream,omitempty"`
}

type RouteIntentAdmissionInput struct {
	Version string                     `json:"version"`
	Intent  RouteIntent                `json:"intent"`
	Facts   *RouteIntentAdmissionFacts `json:"facts"`
}

type NormalizedRouteIntentAdmission struct {
	Version         string                     `json:"version"`
	Intent          RouteIntent                `json:"intent"`
	Facts           *RouteIntentAdmissionFacts `json:"facts"`
	CanonicalDigest string                     `json:"canonical_digest"`
}

type RouteIntentAdmissionResult struct {
	Version           string    `json:"version"`
	Verdict           string    `json:"verdict"`
	CatalogGeneration string    `json:"catalog_generation,omitempty"`
	Selected          *Identity `json:"selected,omitempty"`
	Fallback          *Identity `json:"fallback,omitempty"`
	Reasons           []string  `json:"reasons,omitempty"`
	RunStreamParity   bool      `json:"run_stream_parity,omitempty"`
	CanonicalDigest   string    `json:"canonical_digest"`
}

// NormalizeRouteIntentAdmission validates and canonicalizes an evidence
// request without I/O, provider calls, credential probes, or state mutation.
func NormalizeRouteIntentAdmission(input RouteIntentAdmissionInput) (NormalizedRouteIntentAdmission, error) {
	if strings.TrimSpace(input.Version) != RouteIntentAdmissionVersionV1 {
		return NormalizedRouteIntentAdmission{}, reasonError(ReasonRouteIntentUnsupportedVersion, "unsupported route intent version")
	}
	intent, err := normalizeRouteIntent(input.Intent)
	if err != nil {
		return NormalizedRouteIntentAdmission{}, err
	}
	var facts *RouteIntentAdmissionFacts
	if input.Facts != nil {
		normalizedFacts, factsErr := normalizeRouteIntentFacts(*input.Facts)
		if factsErr != nil {
			return NormalizedRouteIntentAdmission{}, factsErr
		}
		facts = &normalizedFacts
	}
	normalized := NormalizedRouteIntentAdmission{Version: RouteIntentAdmissionVersionV1, Intent: intent, Facts: facts}
	canonical, err := json.Marshal(normalized)
	if err != nil || len(canonical) > MaxRouteIntentSerializedBytes {
		return NormalizedRouteIntentAdmission{}, reasonError(ReasonRouteIntentOverflow, "canonical evidence exceeds bound")
	}
	digest := sha256.Sum256(canonical)
	normalized.CanonicalDigest = hex.EncodeToString(digest[:])
	return normalized, nil
}

// CompareRouteIntentAdmission compares an explicit intent with existing
// admission facts and returns a stable three-valued evidence verdict.
func CompareRouteIntentAdmission(input RouteIntentAdmissionInput) (RouteIntentAdmissionResult, error) {
	normalized, err := NormalizeRouteIntentAdmission(input)
	if err != nil {
		return RouteIntentAdmissionResult{}, err
	}
	result := RouteIntentAdmissionResult{Version: normalized.Version, Verdict: RouteIntentVerdictInsufficientEvidence, CanonicalDigest: normalized.CanonicalDigest}
	if normalized.Facts == nil {
		result.Reasons = []string{ReasonRouteIntentMissingFacts}
		return result, nil
	}
	facts := normalized.Facts
	result.CatalogGeneration = facts.CatalogGeneration
	result.Selected = cloneIdentityPtr(facts.Selected)
	result.Fallback = cloneIdentityPtr(facts.Fallback)
	if normalized.Intent.ExpectedCatalogGeneration != "" && facts.CatalogGeneration != normalized.Intent.ExpectedCatalogGeneration {
		result.Reasons = []string{ReasonRouteIntentGenerationMismatch}
		return result, nil
	}
	if normalized.Intent.RequireRunStreamParity {
		if facts.Run == nil || facts.Stream == nil {
			result.Reasons = []string{ReasonRouteIntentParityMissing}
			return result, nil
		}
		if !routeIntentSnapshotsEqual(*facts.Run, *facts.Stream) {
			result.Verdict = RouteIntentVerdictRouteGapConfirmed
			result.Reasons = []string{ReasonRouteIntentParityMismatch}
			return result, nil
		}
		result.RunStreamParity = true
	}
	if facts.Status != StatusReady && facts.Status != StatusDegraded {
		if facts.Status == StatusBlocked {
			result.Reasons = []string{ReasonRouteIntentAdmissionBlocked}
		} else {
			result.Reasons = []string{ReasonRouteIntentUnknownStatus}
		}
		return result, nil
	}
	if len(normalized.Intent.Required) > 0 || len(normalized.Intent.Optional) > 0 {
		if facts.CapabilitiesAccepted == nil {
			result.Reasons = []string{ReasonRouteIntentCapabilityEvidenceMissing}
			return result, nil
		}
		if !*facts.CapabilitiesAccepted {
			result.Reasons = []string{ReasonRouteIntentCapabilityGap}
			return result, nil
		}
	}
	if facts.CredentialStatus == CredentialMissing || facts.CredentialStatus == CredentialInvalid {
		result.Reasons = []string{ReasonRouteIntentCredentialGap}
		return result, nil
	}
	if facts.Selected == nil {
		result.Reasons = []string{ReasonRouteIntentMissingFacts}
		return result, nil
	}
	if normalized.Intent.Target != nil && !sameIdentity(*normalized.Intent.Target, *facts.Selected) {
		result.Verdict = RouteIntentVerdictRouteGapConfirmed
		result.Reasons = []string{ReasonRouteIntentTargetNotSelected}
		return result, nil
	}
	if len(normalized.Intent.Allowed) > 0 && !containsIdentity(normalized.Intent.Allowed, *facts.Selected) {
		result.Verdict = RouteIntentVerdictRouteGapConfirmed
		result.Reasons = []string{ReasonRouteIntentCandidateNotAllowed}
		return result, nil
	}
	result.Verdict = RouteIntentVerdictSatisfied
	return result, nil
}

func normalizeRouteIntent(intent RouteIntent) (RouteIntent, error) {
	if intent.Target == nil && len(intent.Allowed) == 0 {
		return RouteIntent{}, reasonError(ReasonRouteIntentMissingIntent, "target or allowed candidate is required")
	}
	out := RouteIntent{Required: normalizeCapabilities(intent.Required), Optional: normalizeCapabilities(intent.Optional), RequireRunStreamParity: intent.RequireRunStreamParity}
	if len(out.Required) > MaxRouteIntentCapabilities || len(out.Optional) > MaxRouteIntentCapabilities {
		return RouteIntent{}, reasonError(ReasonRouteIntentOverflow, "capability count exceeds bound")
	}
	if intent.Target != nil {
		identity, err := normalizeRouteIntentIdentity(*intent.Target)
		if err != nil {
			return RouteIntent{}, err
		}
		out.Target = &identity
	}
	if len(intent.Allowed) > MaxRouteIntentAllowedCandidates {
		return RouteIntent{}, reasonError(ReasonRouteIntentOverflow, "allowed candidate count exceeds bound")
	}
	seen := map[string]struct{}{}
	for _, raw := range intent.Allowed {
		identity, err := normalizeRouteIntentIdentity(raw)
		if err != nil {
			return RouteIntent{}, err
		}
		key := identityKey(identity)
		if _, exists := seen[key]; exists {
			return RouteIntent{}, reasonError(ReasonRouteIntentOverflow, "duplicate allowed candidate")
		}
		seen[key] = struct{}{}
		out.Allowed = append(out.Allowed, identity)
	}
	sort.Slice(out.Allowed, func(i, j int) bool { return identityKey(out.Allowed[i]) < identityKey(out.Allowed[j]) })
	generation := strings.TrimSpace(intent.ExpectedCatalogGeneration)
	if len(generation) > MaxRouteIntentGenerationLength || containsControl(generation) {
		return RouteIntent{}, reasonError(ReasonRouteIntentOverflow, "expected generation exceeds bound")
	}
	out.ExpectedCatalogGeneration = generation
	return out, nil
}

func normalizeRouteIntentFacts(facts RouteIntentAdmissionFacts) (RouteIntentAdmissionFacts, error) {
	generation := strings.TrimSpace(facts.CatalogGeneration)
	if generation == "" || len(generation) > MaxRouteIntentGenerationLength || containsControl(generation) {
		return RouteIntentAdmissionFacts{}, reasonError(ReasonRouteIntentGenerationMismatch, "catalog generation is empty or invalid")
	}
	reasons, reasonErr := normalizeRouteIntentReasons(facts.Reasons)
	if reasonErr != nil {
		return RouteIntentAdmissionFacts{}, reasonErr
	}
	capabilityReasons, capabilityReasonErr := normalizeRouteIntentReasons(facts.CapabilityReasons)
	if capabilityReasonErr != nil {
		return RouteIntentAdmissionFacts{}, capabilityReasonErr
	}
	out := RouteIntentAdmissionFacts{CatalogGeneration: generation, Status: strings.ToLower(strings.TrimSpace(facts.Status)), Reasons: reasons, CapabilitiesAccepted: cloneBoolPtr(facts.CapabilitiesAccepted), CapabilitiesDowngraded: facts.CapabilitiesDowngraded, CapabilityReasons: capabilityReasons, CredentialStatus: strings.ToLower(strings.TrimSpace(facts.CredentialStatus))}
	if out.CredentialStatus != "" && out.CredentialStatus != CredentialAvailable && out.CredentialStatus != CredentialMissing && out.CredentialStatus != CredentialInvalid && out.CredentialStatus != CredentialUnverified {
		return RouteIntentAdmissionFacts{}, reasonError(ReasonRouteIntentPrivacyViolation, "invalid credential status")
	}
	var err error
	if out.Selected, err = normalizeRouteIntentIdentityPtr(facts.Selected); err != nil {
		return RouteIntentAdmissionFacts{}, err
	}
	if out.Fallback, err = normalizeRouteIntentIdentityPtr(facts.Fallback); err != nil {
		return RouteIntentAdmissionFacts{}, err
	}
	if facts.Run != nil {
		snapshot, snapshotErr := normalizeRouteIntentSnapshot(*facts.Run)
		if snapshotErr != nil {
			return RouteIntentAdmissionFacts{}, snapshotErr
		}
		out.Run = &snapshot
	}
	if facts.Stream != nil {
		snapshot, snapshotErr := normalizeRouteIntentSnapshot(*facts.Stream)
		if snapshotErr != nil {
			return RouteIntentAdmissionFacts{}, snapshotErr
		}
		out.Stream = &snapshot
	}
	return out, nil
}

func normalizeRouteIntentSnapshot(snapshot RouteIntentAdmissionSnapshot) (RouteIntentAdmissionSnapshot, error) {
	facts, err := normalizeRouteIntentFacts(RouteIntentAdmissionFacts{CatalogGeneration: snapshot.CatalogGeneration, Status: snapshot.Status, Selected: snapshot.Selected, Fallback: snapshot.Fallback, Reasons: snapshot.Reasons, CapabilitiesAccepted: snapshot.CapabilitiesAccepted, CredentialStatus: snapshot.CredentialStatus})
	if err != nil {
		return RouteIntentAdmissionSnapshot{}, err
	}
	return RouteIntentAdmissionSnapshot{CatalogGeneration: facts.CatalogGeneration, Status: facts.Status, Selected: facts.Selected, Fallback: facts.Fallback, Reasons: facts.Reasons, CapabilitiesAccepted: facts.CapabilitiesAccepted, CredentialStatus: facts.CredentialStatus}, nil
}

// RouteIntentFactsFromAdmission converts the existing catalog admission
// projection into route-intent evidence without exposing provider payloads.
func RouteIntentFactsFromAdmission(admission RoutingAdmissionResult) RouteIntentAdmissionFacts {
	accepted := admission.Capabilities.Accepted
	return RouteIntentAdmissionFacts{
		CatalogGeneration:      admission.CatalogGeneration,
		Status:                 admission.Status,
		Selected:               cloneIdentityPtr(admission.Selected),
		Fallback:               cloneIdentityPtr(admission.Fallback),
		Reasons:                boundedReasons(admission.Reasons),
		CapabilitiesAccepted:   &accepted,
		CapabilitiesDowngraded: admission.Capabilities.Downgraded,
		CapabilityReasons:      boundedReasons(admission.Capabilities.Reasons),
		CredentialStatus:       strings.ToLower(strings.TrimSpace(admission.Credential.Status)),
	}
}

func normalizeRouteIntentIdentity(identity Identity) (Identity, error) {
	normalized := normalizeIdentity(identity)
	if normalized.Provider == "" || normalized.Model == "" || len(normalized.Provider) > MaxRouteIntentIdentityLength || len(normalized.Model) > MaxRouteIntentIdentityLength || containsControl(normalized.Provider) || containsControl(normalized.Model) {
		return Identity{}, reasonError(ReasonRouteIntentOverflow, "identity is empty or exceeds bound")
	}
	if looksSensitive(normalized.Provider) || looksSensitive(normalized.Model) {
		return Identity{}, reasonError(ReasonRouteIntentPrivacyViolation, "identity contains sensitive material")
	}
	return normalized, nil
}

func normalizeRouteIntentReasons(values []string) ([]string, error) {
	for _, value := range values {
		if looksSensitive(value) {
			return nil, reasonError(ReasonRouteIntentPrivacyViolation, "reason contains sensitive material")
		}
	}
	return boundedReasons(values), nil
}

func normalizeRouteIntentIdentityPtr(identity *Identity) (*Identity, error) {
	if identity == nil {
		return nil, nil
	}
	normalized, err := normalizeRouteIntentIdentity(*identity)
	if err != nil {
		return nil, err
	}
	return &normalized, nil
}

func routeIntentSnapshotsEqual(left, right RouteIntentAdmissionSnapshot) bool {
	return left.CatalogGeneration == right.CatalogGeneration && left.Status == right.Status && sameIdentityPtr(left.Selected, right.Selected) && sameIdentityPtr(left.Fallback, right.Fallback) && equalStrings(left.Reasons, right.Reasons) && equalBoolPtr(left.CapabilitiesAccepted, right.CapabilitiesAccepted) && left.CredentialStatus == right.CredentialStatus
}

func sameIdentity(left, right Identity) bool { return identityKey(left) == identityKey(right) }

func sameIdentityPtr(left, right *Identity) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return sameIdentity(*left, *right)
}

func containsIdentity(values []Identity, target Identity) bool {
	for _, value := range values {
		if sameIdentity(value, target) {
			return true
		}
	}
	return false
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func cloneIdentityPtr(identity *Identity) *Identity {
	if identity == nil {
		return nil
	}
	copy := *identity
	return &copy
}

func cloneBoolPtr(value *bool) *bool {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func equalBoolPtr(left, right *bool) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}
