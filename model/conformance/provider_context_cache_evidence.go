package conformance

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
)

// FixtureVersionProviderContextCacheEvidenceV1 is the versioned, reference-only
// admission evidence contract. It deliberately does not replace the request
// projection fixture: it points at that fixture by case identity and digest.
const FixtureVersionProviderContextCacheEvidenceV1 = "provider_context_cache_evidence.v1"

const (
	ReasonContextCacheEvidenceSchemaDrift       = "provider_context_cache_evidence_schema_drift"
	ReasonContextCacheEvidenceUnknownVersion    = "provider_context_cache_evidence_unknown_version"
	ReasonContextCacheEvidenceReferenceDrift    = "provider_context_cache_evidence_reference_drift"
	ReasonContextCacheEvidencePrivacyDrift      = "provider_context_cache_evidence_privacy_drift"
	ReasonContextCacheEvidenceOverflowDrift     = "provider_context_cache_evidence_overflow_drift"
	ReasonContextCacheEvidenceDuplicateConflict = "provider_context_cache_evidence_duplicate_conflict"
	ReasonContextCacheEvidenceRoleDrift         = "provider_context_cache_role_drift"
	ReasonContextCacheEvidenceSkillTailDrift    = "provider_context_cache_skill_tail_drift"
	ReasonContextCacheEvidenceToolResultDrift   = "provider_context_cache_tool_result_drift"
	ReasonContextCacheEvidenceOrderingDrift     = "provider_context_cache_ordering_drift"
	ReasonContextCacheEvidenceCacheParityDrift  = "provider_context_cache_parity_drift"
	ReasonContextCacheEvidenceCostDrift         = "provider_context_cache_cost_evidence_drift"
	ReasonContextCacheEvidenceMissing           = "provider_context_cache_evidence_missing"
	ReasonContextCacheEvidenceVerdictDrift      = "provider_context_cache_verdict_drift"
)

const (
	ContextCacheEvidenceVerdictNoDrift              = "no-drift"
	ContextCacheEvidenceVerdictDriftConfirmed       = "drift-confirmed"
	ContextCacheEvidenceVerdictInsufficientEvidence = "insufficient-evidence"

	ContextCacheEvidenceStatusVerifiedNoDrift = "verified-no-drift"
	ContextCacheEvidenceStatusDriftConfirmed  = "drift-confirmed"
	ContextCacheEvidenceStatusUnavailable     = "unavailable"
)

const (
	ContextCacheEvidenceDimensionRole        = "role"
	ContextCacheEvidenceDimensionSkillTail   = "skill-tail"
	ContextCacheEvidenceDimensionToolResult  = "tool-result"
	ContextCacheEvidenceDimensionOrdering    = "ordering"
	ContextCacheEvidenceDimensionCacheParity = "cache-parity"
	ContextCacheEvidenceDimensionHostCostP95 = "host-cost-p95"
)

var contextCacheEvidenceDimensions = []string{
	ContextCacheEvidenceDimensionRole,
	ContextCacheEvidenceDimensionSkillTail,
	ContextCacheEvidenceDimensionToolResult,
	ContextCacheEvidenceDimensionOrdering,
	ContextCacheEvidenceDimensionCacheParity,
	ContextCacheEvidenceDimensionHostCostP95,
}

const (
	contextCacheEvidenceMaxCases        = 64
	contextCacheEvidenceMaxDimensions   = 16
	contextCacheEvidenceMaxReasons      = 16
	contextCacheEvidenceMaxString       = 256
	contextCacheEvidenceMaxFixtureBytes = 1 << 20
	contextCacheEvidenceMinSamples      = 2
)

// ProviderContextCacheEvidenceFixture is the bounded top-level fixture.
type ProviderContextCacheEvidenceFixture struct {
	Version string                             `json:"version"`
	Cases   []ProviderContextCacheEvidenceCase `json:"cases"`
}

// ProviderContextCacheEvidenceReference identifies the existing request
// projection evidence without copying its payload into this contract.
type ProviderContextCacheEvidenceReference struct {
	Provider         string `json:"provider"`
	SDKVersion       string `json:"sdk_version"`
	ProjectionCase   string `json:"projection_case"`
	ProjectionDigest string `json:"projection_digest"`
}

// ProviderContextCacheEvidenceDimension is a bounded, semantic observation.
// It intentionally contains no raw SDK fields or content bodies.
type ProviderContextCacheEvidenceDimension struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

// ProviderContextCachePerformanceSummary is an optional host-provided summary.
// Values are unit-explicit and never contain raw samples or pricing data.
type ProviderContextCachePerformanceSummary struct {
	Metric       string  `json:"metric"`
	Unit         string  `json:"unit"`
	SampleCount  int     `json:"sample_count"`
	BaselineP50  float64 `json:"baseline_p50"`
	BaselineP95  float64 `json:"baseline_p95"`
	ObservedP50  float64 `json:"observed_p50"`
	ObservedP95  float64 `json:"observed_p95"`
	ThresholdP95 float64 `json:"threshold_p95"`
}

// ProviderContextCacheEvidenceCase is one independently replayable admission
// decision. Verdict, Reasons, Digest and ReplayDigest are output expectations.
type ProviderContextCacheEvidenceCase struct {
	CaseID             string                                  `json:"case_id"`
	Reference          ProviderContextCacheEvidenceReference   `json:"reference"`
	RequiredDimensions []string                                `json:"required_dimensions,omitempty"`
	Dimensions         []ProviderContextCacheEvidenceDimension `json:"dimensions,omitempty"`
	Performance        *ProviderContextCachePerformanceSummary `json:"performance,omitempty"`
	Verdict            string                                  `json:"verdict"`
	Reasons            []string                                `json:"reasons,omitempty"`
	Digest             string                                  `json:"digest"`
	ReplayDigest       string                                  `json:"replay_digest"`
}

type ProviderContextCacheEvidenceError struct {
	Code    string
	Message string
}

func (e *ProviderContextCacheEvidenceError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func contextCacheEvidenceErrorf(code, format string, args ...any) error {
	return &ProviderContextCacheEvidenceError{Code: code, Message: fmt.Sprintf(format, args...)}
}

// ValidateProviderContextCacheEvidenceFixture validates only the evidence
// schema. It does not resolve projection references or execute providers.
func ValidateProviderContextCacheEvidenceFixture(in ProviderContextCacheEvidenceFixture) error {
	if in.Version != FixtureVersionProviderContextCacheEvidenceV1 {
		return contextCacheEvidenceErrorf(ReasonContextCacheEvidenceUnknownVersion, "unsupported fixture version %q", in.Version)
	}
	if len(in.Cases) == 0 || len(in.Cases) > contextCacheEvidenceMaxCases {
		return contextCacheEvidenceErrorf(ReasonContextCacheEvidenceOverflowDrift, "cases must contain between one and %d items", contextCacheEvidenceMaxCases)
	}
	seen := make(map[string]struct{}, len(in.Cases))
	for i := range in.Cases {
		if err := ValidateProviderContextCacheEvidenceCase(in.Cases[i]); err != nil {
			var classified *ProviderContextCacheEvidenceError
			if errors.As(err, &classified) {
				return contextCacheEvidenceErrorf(classified.Code, "cases[%d]: %s", i, classified.Message)
			}
			return contextCacheEvidenceErrorf(ReasonContextCacheEvidenceSchemaDrift, "cases[%d]: %v", i, err)
		}
		key := strings.TrimSpace(in.Cases[i].CaseID)
		if _, ok := seen[key]; ok {
			return contextCacheEvidenceErrorf(ReasonContextCacheEvidenceDuplicateConflict, "duplicate case_id %q", key)
		}
		seen[key] = struct{}{}
	}
	return nil
}

// ValidateProviderContextCacheEvidenceCase validates bounded fields and
// rejects duplicate dimensions rather than applying last-write-wins behavior.
func ValidateProviderContextCacheEvidenceCase(in ProviderContextCacheEvidenceCase) error {
	if strings.TrimSpace(in.CaseID) == "" {
		return contextCacheEvidenceErrorf(ReasonContextCacheEvidenceSchemaDrift, "case_id is required")
	}
	for name, value := range map[string]string{
		"case_id": in.CaseID, "provider": in.Reference.Provider, "sdk_version": in.Reference.SDKVersion,
		"projection_case": in.Reference.ProjectionCase, "projection_digest": in.Reference.ProjectionDigest,
	} {
		if err := validateContextCacheEvidenceString(name, value); err != nil {
			return err
		}
	}
	if !validContextCacheEvidenceDigest(in.Reference.ProjectionDigest) {
		return contextCacheEvidenceErrorf(ReasonContextCacheEvidenceReferenceDrift, "projection_digest must be a sha256 hex digest")
	}
	if len(in.RequiredDimensions) > contextCacheEvidenceMaxDimensions || len(in.Dimensions) > contextCacheEvidenceMaxDimensions {
		return contextCacheEvidenceErrorf(ReasonContextCacheEvidenceOverflowDrift, "dimensions exceed %d items", contextCacheEvidenceMaxDimensions)
	}
	if err := validateContextCacheEvidenceDimensionNames(in.RequiredDimensions, "required_dimensions"); err != nil {
		return err
	}
	if err := validateContextCacheEvidenceDimensions(in.Dimensions); err != nil {
		return err
	}
	if err := validateContextCacheEvidencePerformance(in.Performance); err != nil {
		return err
	}
	if in.Verdict != "" && !validContextCacheEvidenceVerdict(in.Verdict) {
		return contextCacheEvidenceErrorf(ReasonContextCacheEvidenceSchemaDrift, "unsupported verdict %q", in.Verdict)
	}
	if len(in.Reasons) > contextCacheEvidenceMaxReasons {
		return contextCacheEvidenceErrorf(ReasonContextCacheEvidenceOverflowDrift, "reasons exceed %d items", contextCacheEvidenceMaxReasons)
	}
	for _, reason := range in.Reasons {
		if err := validateContextCacheEvidenceString("reason", reason); err != nil {
			return err
		}
	}
	for name, value := range map[string]string{"digest": in.Digest, "replay_digest": in.ReplayDigest} {
		if value != "" && !validContextCacheEvidenceDigest(value) {
			return contextCacheEvidenceErrorf(ReasonContextCacheEvidenceSchemaDrift, "%s must be a sha256 hex digest", name)
		}
	}
	return nil
}

func validateContextCacheEvidenceDimensionNames(values []string, field string) error {
	seen := make(map[string]struct{}, len(values))
	for _, raw := range values {
		name := strings.TrimSpace(raw)
		if !validContextCacheEvidenceDimension(name) {
			return contextCacheEvidenceErrorf(ReasonContextCacheEvidenceSchemaDrift, "%s contains unsupported dimension %q", field, name)
		}
		if _, ok := seen[name]; ok {
			return contextCacheEvidenceErrorf(ReasonContextCacheEvidenceDuplicateConflict, "%s contains duplicate dimension %q", field, name)
		}
		seen[name] = struct{}{}
	}
	return nil
}

func validateContextCacheEvidenceDimensions(values []ProviderContextCacheEvidenceDimension) error {
	seen := make(map[string]struct{}, len(values))
	for _, item := range values {
		name := strings.TrimSpace(item.Name)
		if !validContextCacheEvidenceDimension(name) {
			return contextCacheEvidenceErrorf(ReasonContextCacheEvidenceSchemaDrift, "unsupported dimension %q", name)
		}
		if _, ok := seen[name]; ok {
			return contextCacheEvidenceErrorf(ReasonContextCacheEvidenceDuplicateConflict, "duplicate dimension %q", name)
		}
		seen[name] = struct{}{}
		if item.Status != ContextCacheEvidenceStatusVerifiedNoDrift && item.Status != ContextCacheEvidenceStatusDriftConfirmed && item.Status != ContextCacheEvidenceStatusUnavailable {
			return contextCacheEvidenceErrorf(ReasonContextCacheEvidenceSchemaDrift, "dimension %q has unsupported status %q", name, item.Status)
		}
		if err := validateContextCacheEvidenceString("dimension reason", item.Reason); err != nil {
			return err
		}
	}
	return nil
}

func validateContextCacheEvidencePerformance(in *ProviderContextCachePerformanceSummary) error {
	if in == nil {
		return nil
	}
	for name, value := range map[string]string{"metric": in.Metric, "unit": in.Unit} {
		if err := validateContextCacheEvidenceString(name, value); err != nil {
			return err
		}
	}
	if strings.TrimSpace(in.Metric) == "" || strings.TrimSpace(in.Unit) == "" {
		return contextCacheEvidenceErrorf(ReasonContextCacheEvidenceSchemaDrift, "performance metric and unit are required")
	}
	if in.SampleCount < contextCacheEvidenceMinSamples {
		return contextCacheEvidenceErrorf(ReasonContextCacheEvidenceCostDrift, "performance sample_count must be at least %d", contextCacheEvidenceMinSamples)
	}
	values := []float64{in.BaselineP50, in.BaselineP95, in.ObservedP50, in.ObservedP95, in.ThresholdP95}
	for _, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
			return contextCacheEvidenceErrorf(ReasonContextCacheEvidenceCostDrift, "performance values must be finite and non-negative")
		}
	}
	if in.BaselineP50 > in.BaselineP95 || in.ObservedP50 > in.ObservedP95 {
		return contextCacheEvidenceErrorf(ReasonContextCacheEvidenceCostDrift, "P50 must not exceed P95")
	}
	return nil
}

// CanonicalProviderContextCacheEvidenceCase returns the deterministic semantic
// form. Output expectation fields are excluded so the digest cannot be made
// self-referential.
func CanonicalProviderContextCacheEvidenceCase(in ProviderContextCacheEvidenceCase) ([]byte, error) {
	normalized, err := normalizeProviderContextCacheEvidenceCase(in)
	if err != nil {
		return nil, err
	}
	semantic := struct {
		CaseID             string                                  `json:"case_id"`
		Reference          ProviderContextCacheEvidenceReference   `json:"reference"`
		RequiredDimensions []string                                `json:"required_dimensions,omitempty"`
		Dimensions         []ProviderContextCacheEvidenceDimension `json:"dimensions,omitempty"`
		Performance        *ProviderContextCachePerformanceSummary `json:"performance,omitempty"`
	}{normalized.CaseID, normalized.Reference, normalized.RequiredDimensions, normalized.Dimensions, normalized.Performance}
	return json.Marshal(semantic)
}

// ProviderContextCacheEvidenceDigest returns the SHA-256 digest of the
// normalized, reference-only case semantics.
func ProviderContextCacheEvidenceDigest(in ProviderContextCacheEvidenceCase) (string, error) {
	raw, err := CanonicalProviderContextCacheEvidenceCase(in)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

// DeriveProviderContextCacheEvidenceVerdict computes the exhaustive admission
// result. Drift wins over missing evidence; missing required evidence wins over
// no-drift. The returned reasons are sorted and stable.
func DeriveProviderContextCacheEvidenceVerdict(in ProviderContextCacheEvidenceCase) (string, []string, error) {
	normalized, err := normalizeProviderContextCacheEvidenceCase(in)
	if err != nil {
		return "", nil, err
	}
	byName := make(map[string]ProviderContextCacheEvidenceDimension, len(normalized.Dimensions))
	for _, dimension := range normalized.Dimensions {
		byName[dimension.Name] = dimension
	}
	reasons := make([]string, 0)
	insufficient := false
	for _, required := range normalized.RequiredDimensions {
		dimension, ok := byName[required]
		if !ok || dimension.Status == ContextCacheEvidenceStatusUnavailable {
			insufficient = true
			if reason := contextCacheEvidenceMissingReason(required); reason != "" {
				reasons = appendUniqueContextCacheEvidenceReason(reasons, reason)
			}
			continue
		}
		if dimension.Status == ContextCacheEvidenceStatusDriftConfirmed {
			reasons = appendUniqueContextCacheEvidenceReason(reasons, contextCacheEvidenceDimensionReason(required))
		}
	}
	if containsString(normalized.RequiredDimensions, ContextCacheEvidenceDimensionHostCostP95) {
		if normalized.Performance == nil {
			insufficient = true
			reasons = appendUniqueContextCacheEvidenceReason(reasons, ReasonContextCacheEvidenceMissing)
		} else if normalized.Performance.ObservedP95 > normalized.Performance.BaselineP95+normalized.Performance.ThresholdP95 {
			reasons = appendUniqueContextCacheEvidenceReason(reasons, ReasonContextCacheEvidenceCostDrift)
		}
	}
	sort.Strings(reasons)
	switch {
	case len(reasons) > 0 && hasConfirmedDriftReasons(reasons):
		return ContextCacheEvidenceVerdictDriftConfirmed, reasons, nil
	case insufficient:
		return ContextCacheEvidenceVerdictInsufficientEvidence, reasons, nil
	default:
		return ContextCacheEvidenceVerdictNoDrift, reasons, nil
	}
}

func normalizeProviderContextCacheEvidenceCase(in ProviderContextCacheEvidenceCase) (ProviderContextCacheEvidenceCase, error) {
	if err := ValidateProviderContextCacheEvidenceCase(in); err != nil {
		return ProviderContextCacheEvidenceCase{}, err
	}
	in.CaseID = strings.TrimSpace(in.CaseID)
	in.Reference.Provider = strings.TrimSpace(in.Reference.Provider)
	in.Reference.SDKVersion = strings.TrimSpace(in.Reference.SDKVersion)
	in.Reference.ProjectionCase = strings.TrimSpace(in.Reference.ProjectionCase)
	in.Reference.ProjectionDigest = strings.ToLower(strings.TrimSpace(in.Reference.ProjectionDigest))
	in.RequiredDimensions = sortedContextCacheEvidenceStrings(in.RequiredDimensions)
	in.Dimensions = append([]ProviderContextCacheEvidenceDimension(nil), in.Dimensions...)
	for i := range in.Dimensions {
		in.Dimensions[i].Name = strings.TrimSpace(in.Dimensions[i].Name)
		in.Dimensions[i].Status = strings.TrimSpace(in.Dimensions[i].Status)
		in.Dimensions[i].Reason = strings.TrimSpace(in.Dimensions[i].Reason)
	}
	sort.Slice(in.Dimensions, func(i, j int) bool { return in.Dimensions[i].Name < in.Dimensions[j].Name })
	in.Reasons = sortedContextCacheEvidenceStrings(in.Reasons)
	return in, nil
}

func validateContextCacheEvidenceString(name, value string) error {
	if len(value) > contextCacheEvidenceMaxString {
		return contextCacheEvidenceErrorf(ReasonContextCacheEvidenceOverflowDrift, "%s exceeds %d bytes", name, contextCacheEvidenceMaxString)
	}
	lower := strings.ToLower(value)
	for _, marker := range []string{"bearer ", "password", "secret", "credential", "raw_payload", "reasoning", "command output"} {
		if strings.Contains(lower, marker) {
			return contextCacheEvidenceErrorf(ReasonContextCacheEvidencePrivacyDrift, "%s contains sensitive material", name)
		}
	}
	return nil
}

func validContextCacheEvidenceDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func validContextCacheEvidenceDimension(value string) bool {
	for _, allowed := range contextCacheEvidenceDimensions {
		if value == allowed {
			return true
		}
	}
	return false
}

func validContextCacheEvidenceVerdict(value string) bool {
	return value == ContextCacheEvidenceVerdictNoDrift || value == ContextCacheEvidenceVerdictDriftConfirmed || value == ContextCacheEvidenceVerdictInsufficientEvidence
}

func sortedContextCacheEvidenceStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}

func contextCacheEvidenceDimensionReason(dimension string) string {
	switch dimension {
	case ContextCacheEvidenceDimensionRole:
		return ReasonContextCacheEvidenceRoleDrift
	case ContextCacheEvidenceDimensionSkillTail:
		return ReasonContextCacheEvidenceSkillTailDrift
	case ContextCacheEvidenceDimensionToolResult:
		return ReasonContextCacheEvidenceToolResultDrift
	case ContextCacheEvidenceDimensionOrdering:
		return ReasonContextCacheEvidenceOrderingDrift
	case ContextCacheEvidenceDimensionCacheParity:
		return ReasonContextCacheEvidenceCacheParityDrift
	case ContextCacheEvidenceDimensionHostCostP95:
		return ReasonContextCacheEvidenceCostDrift
	default:
		return ReasonContextCacheEvidenceVerdictDrift
	}
}

func contextCacheEvidenceMissingReason(dimension string) string {
	if dimension == ContextCacheEvidenceDimensionHostCostP95 {
		return ReasonContextCacheEvidenceMissing
	}
	return ""
}

func hasConfirmedDriftReasons(reasons []string) bool {
	for _, reason := range reasons {
		switch reason {
		case ReasonContextCacheEvidenceRoleDrift, ReasonContextCacheEvidenceSkillTailDrift, ReasonContextCacheEvidenceToolResultDrift, ReasonContextCacheEvidenceOrderingDrift, ReasonContextCacheEvidenceCacheParityDrift, ReasonContextCacheEvidenceCostDrift:
			return true
		}
	}
	return false
}

func appendUniqueContextCacheEvidenceReason(values []string, value string) []string {
	if value == "" || containsString(values, value) {
		return values
	}
	return append(values, value)
}
