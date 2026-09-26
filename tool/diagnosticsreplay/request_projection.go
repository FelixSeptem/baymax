package diagnosticsreplay

import (
	"errors"
	"strings"

	"github.com/FelixSeptem/baymax/model/conformance"
)

// RequestProjectionFixtureV1 is the versioned request-side namespace
// (ModelRequest -> provider SDK request). It is owned by model/conformance;
// replay re-exports it so tooling and gate scripts reference one constant.
const RequestProjectionFixtureV1 = conformance.FixtureVersionRequestV1

// Request-side drift and gap classifications. These are aliases, not copies:
// the replay tooling, the contract layer, the spec, and the gate scripts must
// share exactly one vocabulary. A divergence here is a taxonomy drift, not a
// new code.
const (
	ReasonCodeRequestSchemaDrift               = conformance.ReasonRequestSchemaDrift
	ReasonCodeRequestRoleProjectionDrift       = conformance.ReasonRequestRoleProjectionDrift
	ReasonCodeRequestToolResultNativeDrift     = conformance.ReasonRequestToolResultNativeDrift
	ReasonCodeRequestPartOrderingDrift         = conformance.ReasonRequestPartOrderingDrift
	ReasonCodeRequestStablePrefixDrift         = conformance.ReasonRequestStablePrefixDrift
	ReasonCodeRequestToolOrderDrift            = conformance.ReasonRequestToolOrderDrift
	ReasonCodeRequestCapabilityProjectionDrift = conformance.ReasonRequestCapabilityProjectionDrift
	ReasonCodeRequestRunStreamParityDrift      = conformance.ReasonRequestRunStreamParityDrift
	ReasonCodeCacheUsageProjectionDrift        = conformance.ReasonCacheUsageProjectionDrift
	ReasonCodeRequestOverflowDrift             = conformance.ReasonRequestOverflowDrift
	ReasonCodeRequestContractDrift             = conformance.ReasonRequestContractDrift
)

// RequestProjectionReplayCase is the normalized, deterministic per-case replay
// record. It carries classifications, digests, and counters only: no raw
// prompt, reasoning, or tool output body may ever appear here.
//
// Gaps is the computed semantic-loss set. It is evidence, not a tolerance: a
// non-empty set must be declared by the fixture and a declared set that stops
// reproducing fails replay as provider_request_contract_drift.
type RequestProjectionReplayCase struct {
	Name                    string   `json:"name"`
	Provider                string   `json:"provider"`
	Mode                    string   `json:"mode"`
	Gaps                    []string `json:"gaps,omitempty"`
	Digest                  string   `json:"digest"`
	ReplayDigest            string   `json:"replay_digest"`
	Idempotent              bool     `json:"idempotent"`
	RunStreamParityVerified bool     `json:"run_stream_parity_verified"`
	CacheUsageAvailable     bool     `json:"cache_usage_available"`
	CacheReadTokens         int64    `json:"cache_read_tokens,omitempty"`
	CacheWriteTokens        int64    `json:"cache_write_tokens,omitempty"`
	CacheTotalTokens        int64    `json:"cache_total_tokens,omitempty"`
	CacheSourceKind         string   `json:"cache_source_kind,omitempty"`
	CacheSourceVersion      string   `json:"cache_source_version,omitempty"`
}

// RequestProjectionReplayResult is the normalized replay output. Replaying the
// same fixture twice yields DeepEqual results: replay reads no clock, no
// network, and no runtime state.
type RequestProjectionReplayResult struct {
	Version string                        `json:"version"`
	Cases   []RequestProjectionReplayCase `json:"cases"`
}

// ParseProviderRequestProjectionFixtureJSON parses and validates a versioned
// request-projection fixture without executing it.
func ParseProviderRequestProjectionFixtureJSON(raw []byte) (conformance.RequestProjectionFixture, error) {
	return conformance.ParseRequestProjectionFixtureJSON(raw)
}

// ReplayProviderRequestProjectionFixtureJSON performs the offline, read-only
// replay of a provider_request_projection.v1 fixture.
//
// It calls no provider, executes no tool, touches no runtime state, and
// mutates no counter. Validation failures are returned as *ValidationError with
// the contract's stable code so callers never string-match error text.
func ReplayProviderRequestProjectionFixtureJSON(raw []byte) (RequestProjectionReplayResult, error) {
	fixture, err := conformance.ParseRequestProjectionFixtureJSON(raw)
	if err != nil {
		return RequestProjectionReplayResult{}, classifyRequestProjectionError(err, "")
	}

	result := RequestProjectionReplayResult{
		Version: fixture.Version,
		Cases:   make([]RequestProjectionReplayCase, 0, len(fixture.Cases)),
	}
	for i := range fixture.Cases {
		record, err := replayRequestProjectionCase(fixture.Cases[i])
		if err != nil {
			return RequestProjectionReplayResult{}, classifyRequestProjectionError(err, fixture.Cases[i].Name)
		}
		result.Cases = append(result.Cases, record)
	}
	return result, nil
}

// replayRequestProjectionCase normalizes one already-validated case and proves
// digest idempotency by computing the projection digest twice. A mismatch means
// the canonical form is not deterministic, which is a contract failure rather
// than a data failure.
func replayRequestProjectionCase(c conformance.RequestProjectionCase) (RequestProjectionReplayCase, error) {
	first, err := conformance.RequestProjectionDigest(c.Observed)
	if err != nil {
		return RequestProjectionReplayCase{}, err
	}
	replay, err := conformance.RequestProjectionDigest(c.Observed)
	if err != nil {
		return RequestProjectionReplayCase{}, err
	}
	if first != replay {
		return RequestProjectionReplayCase{}, &ValidationError{
			Code:    ReasonCodeRequestContractDrift,
			Message: "request projection digest is not deterministic",
		}
	}

	record := RequestProjectionReplayCase{
		Name:                strings.TrimSpace(c.Name),
		Provider:            strings.TrimSpace(c.Provider),
		Mode:                strings.TrimSpace(c.Mode),
		Gaps:                conformance.ClassifyRequestGaps(c.Source, c.Observed),
		Digest:              first,
		ReplayDigest:        replay,
		Idempotent:          first == replay,
		CacheUsageAvailable: c.Observed.CacheUsage.Available,
		CacheReadTokens:     c.Observed.CacheUsage.ReadTokens,
		CacheWriteTokens:    c.Observed.CacheUsage.WriteTokens,
		CacheTotalTokens:    c.Observed.CacheUsage.TotalTokens,
		CacheSourceKind:     c.Observed.CacheUsage.SourceKind,
		CacheSourceVersion:  c.Observed.CacheUsage.SourceVersion,
	}
	if c.Run != nil && c.Stream != nil {
		// ParseRequestProjectionFixtureJSON already rejects a parity violation,
		// so reaching here with an empty code proves parity held.
		if code := conformance.ClassifyRunStreamParity(*c.Run, *c.Stream); code != "" {
			return RequestProjectionReplayCase{}, &ValidationError{
				Code:    ReasonCodeRequestRunStreamParityDrift,
				Message: code,
			}
		}
		record.RunStreamParityVerified = true
	}
	return record, nil
}

// classifyRequestProjectionError converts a contract-layer classification into
// a replay ValidationError, preserving the stable code verbatim.
func classifyRequestProjectionError(err error, caseName string) error {
	var classified *conformance.RequestProjectionError
	if errors.As(err, &classified) {
		message := classified.Message
		if name := strings.TrimSpace(caseName); name != "" {
			message = name + ": " + message
		}
		return &ValidationError{Code: classified.Code, Message: message}
	}
	message := err.Error()
	if name := strings.TrimSpace(caseName); name != "" {
		message = name + ": " + message
	}
	return &ValidationError{Code: ReasonCodeRequestSchemaDrift, Message: message}
}
