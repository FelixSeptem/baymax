package conformance

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/FelixSeptem/baymax/core/types"
)

// FixtureVersionRequestV1 is the versioned namespace for the request-side
// projection contract: ModelRequest -> provider SDK request.
const FixtureVersionRequestV1 = "provider_request_projection.v1"

// Request-side gap and drift classifications. These codes are part of the
// contract: diagnostics replay, gate scripts, specs, and docs use the same
// vocabulary.
const (
	ReasonRequestSchemaDrift               = "provider_request_schema_drift"
	ReasonRequestRoleProjectionDrift       = "provider_request_role_projection_drift"
	ReasonRequestToolResultNativeDrift     = "provider_request_tool_result_native_drift"
	ReasonRequestPartOrderingDrift         = "provider_request_part_ordering_drift"
	ReasonRequestStablePrefixDrift         = "provider_request_stable_prefix_drift"
	ReasonRequestToolOrderDrift            = "provider_request_tool_order_drift"
	ReasonRequestCapabilityProjectionDrift = "provider_request_capability_projection_drift"
	ReasonRequestRunStreamParityDrift      = "provider_request_run_stream_parity_drift"
	ReasonCacheUsageProjectionDrift        = "provider_cache_usage_projection_drift"
	ReasonRequestOverflowDrift             = "provider_request_overflow_drift"
	ReasonRequestContractDrift             = "provider_request_contract_drift"
)

// RequestPartVocabulary is the bounded, provider-neutral set of normalized
// request part kinds. Provider SDK shapes MUST be normalized into these kinds
// before entering this package.
const (
	RequestPartUserText           = "user_text"
	RequestPartSystemText         = "system_text"
	RequestPartAssistantText      = "assistant_text"
	RequestPartToolResultNative   = "tool_result_native"
	RequestPartToolResultEnvelope = "tool_result_envelope"
	RequestPartStablePrefix       = "stable_prefix"
)

var requestPartVocabulary = []string{
	RequestPartUserText,
	RequestPartSystemText,
	RequestPartAssistantText,
	RequestPartToolResultNative,
	RequestPartToolResultEnvelope,
	RequestPartStablePrefix,
}

var requestRoleVocabulary = []string{"system", "user", "assistant", "tool"}

var requestGapVocabulary = []string{
	ReasonRequestRoleProjectionDrift,
	ReasonRequestToolResultNativeDrift,
	ReasonRequestStablePrefixDrift,
	ReasonRequestToolOrderDrift,
	ReasonRequestCapabilityProjectionDrift,
}

// Bounds. Request-side bounds are tighter than the stream-edge fixture bounds
// because a request projection carries structural facts rather than content.
const (
	MaxRequestCases      = 64
	MaxRequestParts      = 128
	MaxRequestRoles      = 8
	MaxRequestTools      = 64
	MaxRequestIdentifier = 256
	MaxRequestBytes      = 2 << 20
	MaxCacheTokens       = 1 << 60
)

const (
	CacheUsageSourceOpenAIResponses = "openai_responses"
	CacheUsageSourceAnthropic       = "anthropic_messages"
	CacheUsageSourceGemini          = "gemini_generate_content"
)

var cacheUsageSourceVocabulary = []string{
	CacheUsageSourceOpenAIResponses,
	CacheUsageSourceAnthropic,
	CacheUsageSourceGemini,
}

// RequestProjectionError carries a stable classification code so that replay
// and gate tooling can act on the failure without string matching.
type RequestProjectionError struct {
	Code    string
	Message string
}

func (e *RequestProjectionError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func requestErrorf(code, format string, args ...any) *RequestProjectionError {
	return &RequestProjectionError{Code: code, Message: fmt.Sprintf(format, args...)}
}

// RequestFacts describes what the runtime admitted into types.ModelRequest. It
// is provider-neutral: no provider SDK type may appear here. Only digests and
// counters are recorded; no raw prompt, reasoning, or tool output body.
type RequestFacts struct {
	Roles          []string          `json:"roles,omitempty"`
	ToolOrder      []string          `json:"tool_order,omitempty"`
	ToolResults    []RequestToolFact `json:"tool_results,omitempty"`
	Capabilities   []string          `json:"capabilities,omitempty"`
	SkillFragments int               `json:"skill_fragments,omitempty"`
	PrefixVersion  string            `json:"prefix_version,omitempty"`
	InputBytes     int               `json:"input_bytes,omitempty"`
}

// RequestToolFact is a bounded reference to one canonical tool-result feedback
// item. The tool output body itself is never stored; only its digest.
type RequestToolFact struct {
	CallID       string `json:"call_id,omitempty"`
	Name         string `json:"name,omitempty"`
	Status       string `json:"status,omitempty"`
	ResultDigest string `json:"result_digest,omitempty"`
}

// RequestProjection is the normalized, bounded description of what actually
// reached the provider SDK boundary. Overflow is intentionally absent: an
// over-bound projection fails fast and is never truncated into a valid state.
type RequestProjection struct {
	Roles                []string             `json:"roles,omitempty"`
	Parts                []string             `json:"parts,omitempty"`
	ToolResultNative     bool                 `json:"tool_result_native"`
	ToolResultCorrelated bool                 `json:"tool_result_correlated"`
	ToolOrder            []string             `json:"tool_order,omitempty"`
	Capabilities         []string             `json:"capabilities,omitempty"`
	StablePrefixDigest   string               `json:"stable_prefix_digest,omitempty"`
	StablePrefixFirst    bool                 `json:"stable_prefix_first"`
	ContentDigest        string               `json:"content_digest,omitempty"`
	TotalBytes           int                  `json:"total_bytes,omitempty"`
	CacheUsage           CacheUsageProjection `json:"cache_usage"`
}

// CacheUsageProjection carries bounded, provider-neutral cache accounting.
// Provider-native fields are translated only at their owning adapter boundary.
type CacheUsageProjection = types.CacheUsageProjection

// RequestProjectionFixture is the versioned request-side conformance fixture.
type RequestProjectionFixture struct {
	Version string                  `json:"version"`
	Cases   []RequestProjectionCase `json:"cases"`
}

// RequestProjectionCase pins one request-side observation.
//
// DeclaredGaps is the evidence anchor for known semantic loss. It is NOT a
// tolerance: any change in the computed gap set (newly introduced loss, or a
// silent fix) fails validation until the contract is updated explicitly.
type RequestProjectionCase struct {
	Name         string             `json:"name"`
	Provider     string             `json:"provider"`
	Mode         string             `json:"mode"`
	Source       RequestFacts       `json:"source"`
	Observed     RequestProjection  `json:"observed"`
	Expected     *RequestProjection `json:"expected,omitempty"`
	Run          *RequestProjection `json:"run,omitempty"`
	Stream       *RequestProjection `json:"stream,omitempty"`
	DeclaredGaps []string           `json:"declared_gaps,omitempty"`
	FirstDigest  string             `json:"first_digest"`
	ReplayDigest string             `json:"replay_digest"`
}

// CanonicalRequestProjection returns the deterministic, bounded byte form used
// for digests and drift comparison.
func CanonicalRequestProjection(in RequestProjection) ([]byte, error) {
	if err := ValidateRequestProjection(in); err != nil {
		return nil, err
	}
	return json.Marshal(in)
}

// RequestProjectionDigest returns the SHA-256 digest of the canonical form.
func RequestProjectionDigest(in RequestProjection) (string, error) {
	raw, err := CanonicalRequestProjection(in)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

// ValidateRequestProjection enforces the bounded, provider-neutral request
// projection schema.
func ValidateRequestProjection(in RequestProjection) error {
	if len(in.Roles) > MaxRequestRoles {
		return requestErrorf(ReasonRequestOverflowDrift, "roles exceed %d items", MaxRequestRoles)
	}
	for _, role := range in.Roles {
		if !containsString(requestRoleVocabulary, role) {
			return requestErrorf(ReasonRequestSchemaDrift, "unsupported role %q", role)
		}
	}
	if len(in.Parts) > MaxRequestParts {
		return requestErrorf(ReasonRequestOverflowDrift, "parts exceed %d items", MaxRequestParts)
	}
	for _, part := range in.Parts {
		if !containsString(requestPartVocabulary, part) {
			return requestErrorf(ReasonRequestSchemaDrift, "unsupported part %q", part)
		}
	}
	if len(in.ToolOrder) > MaxRequestTools {
		return requestErrorf(ReasonRequestOverflowDrift, "tool_order exceeds %d items", MaxRequestTools)
	}
	if len(in.Capabilities) > MaxRequestTools {
		return requestErrorf(ReasonRequestOverflowDrift, "capabilities exceed %d items", MaxRequestTools)
	}
	if in.TotalBytes < 0 {
		return requestErrorf(ReasonRequestSchemaDrift, "total_bytes must not be negative")
	}
	for _, value := range []string{in.StablePrefixDigest, in.ContentDigest} {
		if len(value) > MaxRequestIdentifier {
			return requestErrorf(ReasonRequestOverflowDrift, "digest exceeds %d bytes", MaxRequestIdentifier)
		}
	}
	return ValidateCacheUsageProjection(in.CacheUsage)
}

// ValidateCacheUsageProjection enforces explicit source, non-negative bounded
// counters, and the read + write = total relationship.
func ValidateCacheUsageProjection(in CacheUsageProjection) error {
	if !in.Available {
		if in.ReadTokens != 0 || in.WriteTokens != 0 || in.TotalTokens != 0 || in.SourceKind != "" || in.SourceVersion != "" {
			return requestErrorf(ReasonCacheUsageProjectionDrift, "cache usage unavailable but accounting fields are populated")
		}
		return nil
	}
	if in.ReadTokens < 0 || in.WriteTokens < 0 || in.TotalTokens < 0 {
		return requestErrorf(ReasonCacheUsageProjectionDrift, "cache token accounting must not be negative")
	}
	if in.ReadTokens > MaxCacheTokens || in.WriteTokens > MaxCacheTokens || in.TotalTokens > MaxCacheTokens {
		return requestErrorf(ReasonRequestOverflowDrift, "cache token accounting exceeds %d", MaxCacheTokens)
	}
	if in.SourceKind == "" || !containsString(cacheUsageSourceVocabulary, in.SourceKind) || in.SourceVersion != "v1" {
		return requestErrorf(ReasonCacheUsageProjectionDrift, "cache usage source kind/version is missing or unsupported")
	}
	if in.ReadTokens > MaxCacheTokens-in.WriteTokens || in.ReadTokens+in.WriteTokens != in.TotalTokens {
		return requestErrorf(ReasonCacheUsageProjectionDrift, "cache total must equal read plus write tokens")
	}
	return nil
}

// ValidateCacheUsageBaselineUnavailable enforces the additive + nullable +
// default cache-usage baseline for adapters that have no cache accounting
// source yet. Availability MUST stay false and read/write tokens MUST stay
// zero: reporting a cache hit that was never observed would be a fabricated
// diagnostic, which is worse than reporting nothing.
//
// When an adapter gains a real accounting source, this baseline is retired by
// an explicit contract change rather than relaxed in place.
func ValidateCacheUsageBaselineUnavailable(in RequestProjection) error {
	if in.CacheUsage.Available {
		return requestErrorf(ReasonCacheUsageProjectionDrift, "adapter claims cache usage availability without an accounting source")
	}
	if in.CacheUsage.ReadTokens != 0 || in.CacheUsage.WriteTokens != 0 || in.CacheUsage.TotalTokens != 0 || in.CacheUsage.SourceKind != "" || in.CacheUsage.SourceVersion != "" {
		return requestErrorf(
			ReasonCacheUsageProjectionDrift,
			"adapter fabricated cache usage: read=%d write=%d total=%d source=%q/%q",
			in.CacheUsage.ReadTokens,
			in.CacheUsage.WriteTokens,
			in.CacheUsage.TotalTokens,
			in.CacheUsage.SourceKind,
			in.CacheUsage.SourceVersion,
		)
	}
	return nil
}

// ValidateRequestFacts enforces the bounded request-side facts schema.
func ValidateRequestFacts(in RequestFacts) error {
	if len(in.Roles) > MaxRequestRoles {
		return requestErrorf(ReasonRequestOverflowDrift, "source roles exceed %d items", MaxRequestRoles)
	}
	for _, role := range in.Roles {
		if !containsString(requestRoleVocabulary, role) {
			return requestErrorf(ReasonRequestSchemaDrift, "unsupported source role %q", role)
		}
	}
	if len(in.ToolOrder) > MaxRequestTools {
		return requestErrorf(ReasonRequestOverflowDrift, "source tool_order exceeds %d items", MaxRequestTools)
	}
	if len(in.ToolResults) > MaxRequestTools {
		return requestErrorf(ReasonRequestOverflowDrift, "source tool_results exceed %d items", MaxRequestTools)
	}
	if len(in.Capabilities) > MaxRequestTools {
		return requestErrorf(ReasonRequestOverflowDrift, "source capabilities exceed %d items", MaxRequestTools)
	}
	if len(in.PrefixVersion) > MaxRequestIdentifier {
		return requestErrorf(ReasonRequestOverflowDrift, "source prefix_version exceeds %d bytes", MaxRequestIdentifier)
	}
	if in.InputBytes < 0 {
		return requestErrorf(ReasonRequestSchemaDrift, "source input_bytes must not be negative")
	}
	for i := range in.ToolResults {
		item := in.ToolResults[i]
		for _, value := range []string{item.CallID, item.Name, item.Status, item.ResultDigest} {
			if len(value) > MaxRequestIdentifier {
				return requestErrorf(ReasonRequestOverflowDrift, "source tool_results[%d] identifier exceeds %d bytes", i, MaxRequestIdentifier)
			}
		}
	}
	return nil
}

// ClassifyRequestGaps returns the sorted set of semantic losses between what
// the runtime admitted (source) and what reached the provider SDK (observed).
// An empty result means the projection preserved every audited dimension.
func ClassifyRequestGaps(source RequestFacts, observed RequestProjection) []string {
	gaps := make([]string, 0, len(requestGapVocabulary))
	if !rolesCovered(source.Roles, observed.Roles) {
		gaps = append(gaps, ReasonRequestRoleProjectionDrift)
	}
	if len(source.ToolResults) > 0 && !observed.ToolResultNative {
		gaps = append(gaps, ReasonRequestToolResultNativeDrift)
	}
	if strings.TrimSpace(source.PrefixVersion) != "" && strings.TrimSpace(observed.StablePrefixDigest) == "" {
		gaps = append(gaps, ReasonRequestStablePrefixDrift)
	}
	if !sameStringSequence(source.ToolOrder, observed.ToolOrder) {
		gaps = append(gaps, ReasonRequestToolOrderDrift)
	}
	if !sameStringSet(source.Capabilities, observed.Capabilities) {
		gaps = append(gaps, ReasonRequestCapabilityProjectionDrift)
	}
	slices.Sort(gaps)
	return slices.Compact(gaps)
}

// ClassifyRequestProjectionDrift compares a recorded expectation against an
// observed projection and returns the first stable code that differs.
func ClassifyRequestProjectionDrift(expected, observed RequestProjection) string {
	if !sameStringSet(expected.Roles, observed.Roles) {
		return ReasonRequestRoleProjectionDrift
	}
	if !reflect.DeepEqual(expected.Parts, observed.Parts) {
		return ReasonRequestPartOrderingDrift
	}
	if expected.ToolResultNative != observed.ToolResultNative || expected.ToolResultCorrelated != observed.ToolResultCorrelated {
		return ReasonRequestToolResultNativeDrift
	}
	if expected.StablePrefixDigest != observed.StablePrefixDigest || expected.StablePrefixFirst != observed.StablePrefixFirst {
		return ReasonRequestStablePrefixDrift
	}
	if !reflect.DeepEqual(expected.ToolOrder, observed.ToolOrder) {
		return ReasonRequestToolOrderDrift
	}
	if !reflect.DeepEqual(expected.Capabilities, observed.Capabilities) {
		return ReasonRequestCapabilityProjectionDrift
	}
	if !reflect.DeepEqual(expected.CacheUsage, observed.CacheUsage) {
		return ReasonCacheUsageProjectionDrift
	}
	if expected.ContentDigest != observed.ContentDigest || expected.TotalBytes != observed.TotalBytes {
		return ReasonRequestPartOrderingDrift
	}
	return ""
}

// ClassifyRunStreamParity compares the run and stream projections of the same
// request and returns a parity code when they are not request-semantically equal.
func ClassifyRunStreamParity(run, stream RequestProjection) string {
	if code := ClassifyRequestProjectionDrift(run, stream); code != "" {
		return ReasonRequestRunStreamParityDrift
	}
	return ""
}

// ProjectRequestTextEnvelope normalizes an adapter projection that collapsed the
// whole request into a single text payload. It is the shared, provider-neutral
// normalization used by adapter audit tests: the adapters currently hand the
// SDK one string, optionally carrying canonical tool-result feedback as a text
// envelope, so role structure, native tool-result attribution, tool order,
// stable prefix and capability requirements are all absent from the projection.
func ProjectRequestTextEnvelope(payload string, toolResults []RequestToolFact) RequestProjection {
	projection := RequestProjection{
		Roles:         []string{"user"},
		Parts:         []string{RequestPartUserText},
		ContentDigest: RequestContentDigest(payload),
		TotalBytes:    len(payload),
	}
	if len(toolResults) > 0 {
		projection.Parts = append(projection.Parts, RequestPartToolResultEnvelope)
		projection.ToolResultCorrelated = true
		projection.ToolOrder = make([]string, 0, len(toolResults))
		for i := range toolResults {
			projection.ToolOrder = append(projection.ToolOrder, strings.TrimSpace(toolResults[i].Name))
			if strings.TrimSpace(toolResults[i].CallID) == "" || strings.TrimSpace(toolResults[i].Name) == "" {
				projection.ToolResultCorrelated = false
			}
		}
	}
	return projection
}

// ProjectRequestNative describes the provider-neutral facts emitted by a
// provider-owned native request builder. It intentionally records only bounded
// role/part/order/correlation facts; provider SDK values and raw content stay
// outside the conformance package.
func ProjectRequestNative(source RequestFacts) RequestProjection {
	projection := RequestProjection{
		Roles:      append([]string(nil), source.Roles...),
		Parts:      make([]string, 0, len(source.Roles)+1+len(source.ToolResults)),
		ToolOrder:  append([]string(nil), source.ToolOrder...),
		TotalBytes: source.InputBytes,
	}
	for _, role := range source.Roles {
		switch strings.ToLower(strings.TrimSpace(role)) {
		case "system":
			projection.Parts = append(projection.Parts, RequestPartSystemText)
		case "user":
			projection.Parts = append(projection.Parts, RequestPartUserText)
		case "assistant":
			projection.Parts = append(projection.Parts, RequestPartAssistantText)
		}
	}
	if source.InputBytes > 0 {
		projection.Parts = append(projection.Parts, RequestPartUserText)
	}
	if len(source.ToolResults) > 0 {
		projection.Parts = append(projection.Parts, RequestPartToolResultNative)
		projection.ToolResultNative = true
		projection.ToolResultCorrelated = true
		for _, result := range source.ToolResults {
			if strings.TrimSpace(result.CallID) == "" || strings.TrimSpace(result.Name) == "" {
				projection.ToolResultCorrelated = false
				break
			}
		}
	}
	return projection
}

// RequestContentDigest returns the bounded content digest used by request
// projection facts. Raw payload text never enters the contract.
func RequestContentDigest(payload string) string {
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}

// ParseRequestProjectionFixtureJSON parses and validates the bounded request
// projection fixture. Validation is side-effect free.
func ParseRequestProjectionFixtureJSON(raw []byte) (RequestProjectionFixture, error) {
	var fixture RequestProjectionFixture
	if len(raw) > MaxRequestBytes {
		return fixture, requestErrorf(ReasonRequestOverflowDrift, "fixture exceeds %d bytes", MaxRequestBytes)
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		return RequestProjectionFixture{}, requestErrorf(ReasonRequestSchemaDrift, "decode fixture: %v", err)
	}
	if fixture.Version != FixtureVersionRequestV1 {
		return RequestProjectionFixture{}, requestErrorf(ReasonRequestSchemaDrift, "unsupported fixture version %q", fixture.Version)
	}
	if len(fixture.Cases) == 0 || len(fixture.Cases) > MaxRequestCases {
		return RequestProjectionFixture{}, requestErrorf(ReasonRequestSchemaDrift, "cases must contain 1..%d items", MaxRequestCases)
	}
	for i := range fixture.Cases {
		if err := ValidateRequestProjectionCase(&fixture.Cases[i]); err != nil {
			var classified *RequestProjectionError
			if errors.As(err, &classified) {
				return RequestProjectionFixture{}, &RequestProjectionError{
					Code:    classified.Code,
					Message: fmt.Sprintf("cases[%d]: %s", i, classified.Message),
				}
			}
			return RequestProjectionFixture{}, requestErrorf(ReasonRequestSchemaDrift, "cases[%d]: %v", i, err)
		}
	}
	return fixture, nil
}

// ValidateRequestProjectionCase enforces one fixture case, including declared
// gap consistency and Run/Stream parity.
func ValidateRequestProjectionCase(c *RequestProjectionCase) error {
	if strings.TrimSpace(c.Name) == "" || strings.TrimSpace(c.Provider) == "" || strings.TrimSpace(c.Mode) == "" {
		return requestErrorf(ReasonRequestSchemaDrift, "name, provider, and mode are required")
	}
	if c.Provider != "openai" && c.Provider != "anthropic" && c.Provider != "gemini" {
		return requestErrorf(ReasonRequestSchemaDrift, "unsupported provider %q", c.Provider)
	}
	if c.Mode != "run" && c.Mode != "stream" {
		return requestErrorf(ReasonRequestSchemaDrift, "unsupported mode %q", c.Mode)
	}
	if err := ValidateRequestFacts(c.Source); err != nil {
		return err
	}
	if err := ValidateRequestProjection(c.Observed); err != nil {
		return err
	}
	if c.Expected != nil {
		if err := ValidateRequestProjection(*c.Expected); err != nil {
			return err
		}
		if code := ClassifyRequestProjectionDrift(*c.Expected, c.Observed); code != "" {
			return requestErrorf(code, "expected projection does not match observed projection")
		}
	}
	if c.Run != nil && c.Stream != nil {
		if err := ValidateRequestProjection(*c.Run); err != nil {
			return err
		}
		if err := ValidateRequestProjection(*c.Stream); err != nil {
			return err
		}
		if code := ClassifyRunStreamParity(*c.Run, *c.Stream); code != "" {
			return requestErrorf(code, "run/stream request projection parity violated")
		}
	}
	declared := normalizeGapSet(c.DeclaredGaps)
	for _, gap := range declared {
		if !containsString(requestGapVocabulary, gap) {
			return requestErrorf(ReasonRequestSchemaDrift, "unsupported declared gap %q", gap)
		}
	}
	computed := ClassifyRequestGaps(c.Source, c.Observed)
	switch {
	case len(declared) == 0 && len(computed) > 0:
		return requestErrorf(computed[0], "undeclared request projection gap in %q", c.Name)
	case len(declared) > 0 && len(computed) == 0:
		return requestErrorf(ReasonRequestContractDrift, "declared gap %q no longer reproduces; update the contract explicitly", strings.Join(declared, ","))
	default:
		for _, gap := range computed {
			if !containsString(declared, gap) {
				return requestErrorf(gap, "undeclared request projection gap in %q", c.Name)
			}
		}
		for _, gap := range declared {
			if !containsString(computed, gap) {
				return requestErrorf(ReasonRequestContractDrift, "declared gap %q no longer reproduces; update the contract explicitly", gap)
			}
		}
	}
	digest, err := RequestProjectionDigest(c.Observed)
	if err != nil {
		return err
	}
	if strings.TrimSpace(c.FirstDigest) == "" || c.FirstDigest != c.ReplayDigest {
		return requestErrorf(ReasonRequestSchemaDrift, "idempotency digests must be equal and non-empty")
	}
	if c.FirstDigest != digest {
		return requestErrorf(ReasonRequestSchemaDrift, "recorded digest does not match canonical projection digest")
	}
	return nil
}

func normalizeGapSet(in []string) []string {
	out := make([]string, 0, len(in))
	for _, item := range in {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	slices.Sort(out)
	return slices.Compact(out)
}

func rolesCovered(source, observed []string) bool {
	for _, role := range source {
		if !containsString(observed, role) {
			return false
		}
	}
	return true
}

func sameStringSet(left, right []string) bool {
	if len(left) == 0 && len(right) == 0 {
		return true
	}
	l := append([]string(nil), left...)
	r := append([]string(nil), right...)
	slices.Sort(l)
	slices.Sort(r)
	return reflect.DeepEqual(slices.Compact(l), slices.Compact(r))
}

// sameStringSequence compares ordered slices: tool order is semantic because it
// determines the provider request shape and the stable prefix boundary.
func sameStringSequence(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func containsString(haystack []string, needle string) bool {
	for _, item := range haystack {
		if item == needle {
			return true
		}
	}
	return false
}
