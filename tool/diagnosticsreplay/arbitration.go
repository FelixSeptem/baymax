package diagnosticsreplay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

const (
	ArbitrationFixtureVersionA48V1 = "a48.v1"
	ReasonCodePrecedenceDrift      = "precedence_drift"
	ReasonCodeTieBreakDrift        = "tie_break_drift"
	ReasonCodeTaxonomyDrift        = "taxonomy_drift"
)

type ArbitrationFixture struct {
	Version string                   `json:"version"`
	Cases   []ArbitrationFixtureCase `json:"cases"`
}

type ArbitrationFixtureCase struct {
	Name        string                 `json:"name"`
	Run         ArbitrationObservation `json:"run"`
	Stream      ArbitrationObservation `json:"stream"`
	Expected    ArbitrationObservation `json:"expected"`
	Idempotency CompositeIdempotency   `json:"idempotency"`
	Additive    map[string]any         `json:"additive,omitempty"`
}

type ArbitrationObservation struct {
	RuntimePrimaryDomain        string `json:"runtime_primary_domain"`
	RuntimePrimaryCode          string `json:"runtime_primary_code"`
	RuntimePrimarySource        string `json:"runtime_primary_source"`
	RuntimePrimaryConflictTotal int    `json:"runtime_primary_conflict_total"`
}

type ArbitrationReplayOutput struct {
	Version string                        `json:"version"`
	Cases   []ArbitrationNormalizedOutput `json:"cases"`
}

type ArbitrationNormalizedOutput struct {
	Name        string                 `json:"name"`
	Canonical   ArbitrationObservation `json:"canonical"`
	Idempotency CompositeIdempotency   `json:"idempotency"`
}

func ParseArbitrationFixtureJSON(raw []byte) (ArbitrationFixture, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var fixture ArbitrationFixture
	if err := dec.Decode(&fixture); err != nil {
		return ArbitrationFixture{}, &ValidationError{Code: ReasonCodeInvalidJSON, Message: err.Error()}
	}
	if strings.TrimSpace(fixture.Version) == "" {
		return ArbitrationFixture{}, &ValidationError{Code: ReasonCodeSchemaMismatch, Message: "version is required"}
	}
	if strings.TrimSpace(fixture.Version) != ArbitrationFixtureVersionA48V1 {
		return ArbitrationFixture{}, &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("unsupported fixture version %q", fixture.Version),
		}
	}
	if len(fixture.Cases) == 0 {
		return ArbitrationFixture{}, &ValidationError{Code: ReasonCodeSchemaMismatch, Message: "cases must not be empty"}
	}
	seen := map[string]struct{}{}
	for i := range fixture.Cases {
		name := strings.TrimSpace(fixture.Cases[i].Name)
		if name == "" {
			return ArbitrationFixture{}, &ValidationError{
				Code:    ReasonCodeSchemaMismatch,
				Message: fmt.Sprintf("cases[%d].name is required", i),
			}
		}
		if _, ok := seen[name]; ok {
			return ArbitrationFixture{}, &ValidationError{
				Code:    ReasonCodeSchemaMismatch,
				Message: fmt.Sprintf("duplicate case name %q", name),
			}
		}
		seen[name] = struct{}{}
	}
	return fixture, nil
}

func EvaluateArbitrationFixtureJSON(raw []byte) (ArbitrationReplayOutput, error) {
	fixture, err := ParseArbitrationFixtureJSON(raw)
	if err != nil {
		return ArbitrationReplayOutput{}, err
	}
	return EvaluateArbitrationFixture(fixture)
}

func EvaluateArbitrationFixture(fixture ArbitrationFixture) (ArbitrationReplayOutput, error) {
	cases := append([]ArbitrationFixtureCase(nil), fixture.Cases...)
	sort.Slice(cases, func(i, j int) bool {
		return strings.TrimSpace(cases[i].Name) < strings.TrimSpace(cases[j].Name)
	})
	out := ArbitrationReplayOutput{
		Version: fixture.Version,
		Cases:   make([]ArbitrationNormalizedOutput, 0, len(cases)),
	}
	for _, tc := range cases {
		name := strings.TrimSpace(tc.Name)
		expected := canonicalizeArbitrationObservation(tc.Expected)
		run := canonicalizeArbitrationObservation(tc.Run)
		stream := canonicalizeArbitrationObservation(tc.Stream)
		if err := validateArbitrationObservation(name, "expected", expected); err != nil {
			return ArbitrationReplayOutput{}, err
		}
		if err := validateArbitrationObservation(name, "run", run); err != nil {
			return ArbitrationReplayOutput{}, err
		}
		if err := validateArbitrationObservation(name, "stream", stream); err != nil {
			return ArbitrationReplayOutput{}, err
		}
		if err := assertArbitrationEquivalent(name, expected, run, "run"); err != nil {
			return ArbitrationReplayOutput{}, err
		}
		if err := assertArbitrationEquivalent(name, expected, stream, "stream"); err != nil {
			return ArbitrationReplayOutput{}, err
		}
		if err := assertArbitrationEquivalent(name, run, stream, "run/stream"); err != nil {
			return ArbitrationReplayOutput{}, err
		}
		if tc.Idempotency.FirstLogicalIngestTotal <= 0 {
			return ArbitrationReplayOutput{}, &ValidationError{
				Code:    ReasonCodeSchemaMismatch,
				Message: fmt.Sprintf("case %q idempotency.first_logical_ingest_total must be > 0", name),
			}
		}
		if tc.Idempotency.FirstLogicalIngestTotal != tc.Idempotency.ReplayLogicalIngestTotal {
			return ArbitrationReplayOutput{}, &ValidationError{
				Code: ReasonCodeSemanticDrift,
				Message: fmt.Sprintf(
					"case %q replay idempotency drift first=%d replay=%d",
					name,
					tc.Idempotency.FirstLogicalIngestTotal,
					tc.Idempotency.ReplayLogicalIngestTotal,
				),
			}
		}
		out.Cases = append(out.Cases, ArbitrationNormalizedOutput{
			Name:        name,
			Canonical:   expected,
			Idempotency: tc.Idempotency,
		})
	}
	return out, nil
}

func canonicalizeArbitrationObservation(in ArbitrationObservation) ArbitrationObservation {
	out := ArbitrationObservation{
		RuntimePrimaryDomain:        strings.ToLower(strings.TrimSpace(in.RuntimePrimaryDomain)),
		RuntimePrimaryCode:          strings.TrimSpace(in.RuntimePrimaryCode),
		RuntimePrimarySource:        strings.ToLower(strings.TrimSpace(in.RuntimePrimarySource)),
		RuntimePrimaryConflictTotal: in.RuntimePrimaryConflictTotal,
	}
	if out.RuntimePrimaryConflictTotal < 0 {
		out.RuntimePrimaryConflictTotal = 0
	}
	return out
}

func validateArbitrationObservation(caseName, lane string, obs ArbitrationObservation) error {
	if strings.TrimSpace(obs.RuntimePrimaryDomain) == "" {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s.runtime_primary_domain is required", caseName, lane),
		}
	}
	if strings.TrimSpace(obs.RuntimePrimarySource) == "" {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s.runtime_primary_source is required", caseName, lane),
		}
	}
	if !isCanonicalArbitrationCode(obs.RuntimePrimaryCode) {
		return &ValidationError{
			Code:    ReasonCodeTaxonomyDrift,
			Message: fmt.Sprintf("case %q %s non-canonical primary code %q", caseName, lane, obs.RuntimePrimaryCode),
		}
	}
	return nil
}

func assertArbitrationEquivalent(caseName string, expected, actual ArbitrationObservation, lane string) error {
	if expected == actual {
		return nil
	}
	if precedenceForArbitrationCode(expected.RuntimePrimaryCode) != precedenceForArbitrationCode(actual.RuntimePrimaryCode) {
		return &ValidationError{
			Code: ReasonCodePrecedenceDrift,
			Message: fmt.Sprintf(
				"case %q %s precedence drift expected=%q actual=%q",
				caseName,
				lane,
				expected.RuntimePrimaryCode,
				actual.RuntimePrimaryCode,
			),
		}
	}
	if expected.RuntimePrimaryCode != actual.RuntimePrimaryCode ||
		expected.RuntimePrimaryConflictTotal != actual.RuntimePrimaryConflictTotal {
		return &ValidationError{
			Code: ReasonCodeTieBreakDrift,
			Message: fmt.Sprintf(
				"case %q %s tie-break drift expected=%#v actual=%#v",
				caseName,
				lane,
				expected,
				actual,
			),
		}
	}
	return &ValidationError{
		Code: ReasonCodeTaxonomyDrift,
		Message: fmt.Sprintf(
			"case %q %s taxonomy drift expected=%#v actual=%#v",
			caseName,
			lane,
			expected,
			actual,
		),
	}
}

func isCanonicalArbitrationCode(code string) bool {
	code = strings.TrimSpace(code)
	if code == "" {
		return false
	}
	if code == runtimeconfig.RuntimePrimaryCodeTimeoutRejected ||
		code == runtimeconfig.RuntimePrimaryCodeTimeoutExhausted ||
		code == runtimeconfig.RuntimePrimaryCodeTimeoutClamped {
		return true
	}
	if _, ok := canonicalReadinessCodes[code]; ok {
		return true
	}
	if _, ok := canonicalAdapterCodes[code]; ok {
		return true
	}
	return false
}

func precedenceForArbitrationCode(code string) int {
	switch strings.TrimSpace(code) {
	case runtimeconfig.RuntimePrimaryCodeTimeoutRejected, runtimeconfig.RuntimePrimaryCodeTimeoutExhausted:
		return 1
	case runtimeconfig.ReadinessCodeConfigInvalid,
		runtimeconfig.ReadinessCodeStrictEscalated,
		runtimeconfig.ReadinessCodeSchedulerActivationError,
		runtimeconfig.ReadinessCodeMailboxActivationError,
		runtimeconfig.ReadinessCodeRecoveryActivationError,
		runtimeconfig.ReadinessCodeRuntimeManagerUnavailable:
		return 2
	case runtimeconfig.ReadinessCodeAdapterRequiredUnavailable, runtimeconfig.ReadinessCodeAdapterRequiredCircuitOpen:
		return 3
	case runtimeconfig.ReadinessCodeSchedulerFallback,
		runtimeconfig.ReadinessCodeMailboxFallback,
		runtimeconfig.ReadinessCodeRecoveryFallback,
		runtimeconfig.ReadinessCodeAdapterOptionalUnavailable,
		runtimeconfig.ReadinessCodeAdapterOptionalCircuitOpen,
		runtimeconfig.ReadinessCodeAdapterDegraded,
		runtimeconfig.ReadinessCodeAdapterHalfOpenDegraded:
		return 4
	default:
		return 5
	}
}
