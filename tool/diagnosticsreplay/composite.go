package diagnosticsreplay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	adapterhealth "github.com/FelixSeptem/baymax/adapter/health"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

const (
	CompositeFixtureVersionA47V1 = "a47.v1"

	ReasonCodeSchemaMismatch = "schema_mismatch"
	ReasonCodeSemanticDrift  = "semantic_drift"
	ReasonCodeOrderingDrift  = "ordering_drift"
)

// CompositeFixture is the versioned fixture envelope for A47 cross-domain replay assertions.
type CompositeFixture struct {
	Version string                 `json:"version"`
	Cases   []CompositeFixtureCase `json:"cases"`
}

// CompositeFixtureCase describes one readiness-timeout-health matrix scenario.
type CompositeFixtureCase struct {
	Name        string                     `json:"name"`
	Dimensions  CompositeFixtureDimensions `json:"dimensions"`
	Run         CompositeObservation       `json:"run"`
	Stream      CompositeObservation       `json:"stream"`
	Expected    CompositeExpected          `json:"expected"`
	Idempotency CompositeIdempotency       `json:"idempotency"`
}

// CompositeFixtureDimensions captures required matrix axes and expected axis values.
type CompositeFixtureDimensions struct {
	ReadinessStatus      string `json:"readiness_status"`
	ReadinessStrict      bool   `json:"readiness_strict"`
	TimeoutSource        string `json:"timeout_source"`
	TimeoutBudgetOutcome string `json:"timeout_budget_outcome"`
	AdapterStatus        string `json:"adapter_status"`
	AdapterRequired      bool   `json:"adapter_required"`
	AdapterCircuitState  string `json:"adapter_circuit_state"`
}

// CompositeObservation captures run/stream observed values before canonical comparison.
type CompositeObservation struct {
	Readiness     CompositeReadiness     `json:"readiness"`
	Timeout       CompositeTimeout       `json:"timeout"`
	AdapterHealth CompositeAdapterHealth `json:"adapter_health"`
	Ordering      []string               `json:"ordering"`
	Additive      map[string]any         `json:"additive,omitempty"`
}

// CompositeExpected is the semantic baseline for deterministic fixture assertions.
type CompositeExpected struct {
	Readiness                 CompositeReadiness     `json:"readiness"`
	Timeout                   CompositeTimeout       `json:"timeout"`
	AdapterHealth             CompositeAdapterHealth `json:"adapter_health"`
	Ordering                  []string               `json:"ordering"`
	NullableCompatibilityKeys []string               `json:"nullable_compatibility_keys,omitempty"`
}

// CompositeReadiness contains canonical readiness semantic fields.
type CompositeReadiness struct {
	Status         string `json:"status"`
	Strict         bool   `json:"strict"`
	PrimaryCode    string `json:"primary_code"`
	ReasonTaxonomy string `json:"reason_taxonomy,omitempty"`
}

// CompositeTimeout contains canonical timeout-resolution semantic fields.
type CompositeTimeout struct {
	Source        string   `json:"source"`
	BudgetOutcome string   `json:"budget_outcome"`
	Trace         []string `json:"trace"`
}

// CompositeAdapterHealth contains canonical adapter-health semantic fields.
type CompositeAdapterHealth struct {
	Status                string `json:"status"`
	Required              bool   `json:"required"`
	CircuitState          string `json:"circuit_state"`
	PrimaryCode           string `json:"primary_code"`
	GovernancePrimaryCode string `json:"governance_primary_code,omitempty"`
	ReasonTaxonomy        string `json:"reason_taxonomy,omitempty"`
}

// CompositeIdempotency captures replay-idempotent logical aggregate expectations.
type CompositeIdempotency struct {
	FirstLogicalIngestTotal  int `json:"first_logical_ingest_total"`
	ReplayLogicalIngestTotal int `json:"replay_logical_ingest_total"`
}

// CompositeReplayOutput is the normalized deterministic output from composite fixture evaluation.
type CompositeReplayOutput struct {
	Version string                      `json:"version"`
	Cases   []CompositeNormalizedOutput `json:"cases"`
}

// CompositeNormalizedOutput is the per-case canonical semantic projection.
type CompositeNormalizedOutput struct {
	Name        string                     `json:"name"`
	Dimensions  CompositeFixtureDimensions `json:"dimensions"`
	Canonical   CompositeObservation       `json:"canonical"`
	Idempotency CompositeIdempotency       `json:"idempotency"`
}

// ParseCompositeFixtureJSON parses a versioned A47 fixture with deterministic schema checks.
func ParseCompositeFixtureJSON(raw []byte) (CompositeFixture, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var fixture CompositeFixture
	if err := dec.Decode(&fixture); err != nil {
		return CompositeFixture{}, &ValidationError{
			Code:    ReasonCodeInvalidJSON,
			Message: err.Error(),
		}
	}
	if strings.TrimSpace(fixture.Version) == "" {
		return CompositeFixture{}, &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: "version is required",
		}
	}
	if strings.TrimSpace(fixture.Version) != CompositeFixtureVersionA47V1 {
		return CompositeFixture{}, &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("unsupported fixture version %q", fixture.Version),
		}
	}
	if len(fixture.Cases) == 0 {
		return CompositeFixture{}, &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: "cases must not be empty",
		}
	}
	if err := validateCompositeSchema(fixture); err != nil {
		return CompositeFixture{}, err
	}
	return fixture, nil
}

// EvaluateCompositeFixtureJSON evaluates an A47 fixture payload into deterministic normalized output.
func EvaluateCompositeFixtureJSON(raw []byte) (CompositeReplayOutput, error) {
	fixture, err := ParseCompositeFixtureJSON(raw)
	if err != nil {
		return CompositeReplayOutput{}, err
	}
	return EvaluateCompositeFixture(fixture)
}

// EvaluateCompositeFixture evaluates parsed fixture cases with fail-fast drift checks.
func EvaluateCompositeFixture(fixture CompositeFixture) (CompositeReplayOutput, error) {
	cases := append([]CompositeFixtureCase(nil), fixture.Cases...)
	sort.Slice(cases, func(i, j int) bool {
		return strings.TrimSpace(cases[i].Name) < strings.TrimSpace(cases[j].Name)
	})

	out := CompositeReplayOutput{
		Version: fixture.Version,
		Cases:   make([]CompositeNormalizedOutput, 0, len(cases)),
	}
	for _, tc := range cases {
		expected := canonicalizeObservation(CompositeObservation{
			Readiness:     tc.Expected.Readiness,
			Timeout:       tc.Expected.Timeout,
			AdapterHealth: tc.Expected.AdapterHealth,
			Ordering:      tc.Expected.Ordering,
		})
		run := canonicalizeObservation(tc.Run)
		stream := canonicalizeObservation(tc.Stream)
		dims := canonicalizeDimensions(tc.Dimensions)

		if err := validateCanonicalSemantics(expected); err != nil {
			return CompositeReplayOutput{}, err
		}
		if err := assertObservationOrdering(tc.Name, expected.Ordering, run.Ordering, "run"); err != nil {
			return CompositeReplayOutput{}, err
		}
		if err := assertObservationOrdering(tc.Name, expected.Ordering, stream.Ordering, "stream"); err != nil {
			return CompositeReplayOutput{}, err
		}
		if err := assertDimensionsMatchCanonical(tc.Name, dims, expected); err != nil {
			return CompositeReplayOutput{}, err
		}
		if err := assertObservationSemanticEquivalent(tc.Name, expected, run, "run"); err != nil {
			return CompositeReplayOutput{}, err
		}
		if err := assertObservationSemanticEquivalent(tc.Name, expected, stream, "stream"); err != nil {
			return CompositeReplayOutput{}, err
		}
		if err := assertObservationSemanticEquivalent(tc.Name, run, stream, "run/stream"); err != nil {
			return CompositeReplayOutput{}, err
		}
		idempotency := tc.Idempotency
		if idempotency.FirstLogicalIngestTotal <= 0 {
			return CompositeReplayOutput{}, &ValidationError{
				Code:    ReasonCodeSchemaMismatch,
				Message: fmt.Sprintf("case %q idempotency.first_logical_ingest_total must be > 0", tc.Name),
			}
		}
		if idempotency.FirstLogicalIngestTotal != idempotency.ReplayLogicalIngestTotal {
			return CompositeReplayOutput{}, &ValidationError{
				Code: ReasonCodeSemanticDrift,
				Message: fmt.Sprintf(
					"case %q replay idempotency drift first=%d replay=%d",
					tc.Name,
					idempotency.FirstLogicalIngestTotal,
					idempotency.ReplayLogicalIngestTotal,
				),
			}
		}
		out.Cases = append(out.Cases, CompositeNormalizedOutput{
			Name:        strings.TrimSpace(tc.Name),
			Dimensions:  dims,
			Canonical:   expected,
			Idempotency: idempotency,
		})
	}
	return out, nil
}

func validateCompositeSchema(fixture CompositeFixture) error {
	seen := map[string]struct{}{}
	coverage := newCompositeCoverage()
	for i, tc := range fixture.Cases {
		name := strings.TrimSpace(tc.Name)
		if name == "" {
			return &ValidationError{Code: ReasonCodeSchemaMismatch, Message: fmt.Sprintf("cases[%d].name is required", i)}
		}
		if _, dup := seen[name]; dup {
			return &ValidationError{Code: ReasonCodeSchemaMismatch, Message: fmt.Sprintf("duplicate case name %q", name)}
		}
		seen[name] = struct{}{}
		dims := canonicalizeDimensions(tc.Dimensions)
		if err := validateDimensions(name, dims); err != nil {
			return err
		}
		coverage.mark(dims)
	}
	return coverage.validate()
}

func validateDimensions(caseName string, dims CompositeFixtureDimensions) error {
	if dims.ReadinessStatus == "" {
		return schemaMismatch(caseName, "dimensions.readiness_status is required")
	}
	if !isReadinessStatus(dims.ReadinessStatus) {
		return schemaMismatch(caseName, "dimensions.readiness_status must be ready|degraded|blocked")
	}
	if !isTimeoutSource(dims.TimeoutSource) {
		return schemaMismatch(caseName, "dimensions.timeout_source must be profile|domain|request")
	}
	if !isTimeoutBudgetOutcome(dims.TimeoutBudgetOutcome) {
		return schemaMismatch(caseName, "dimensions.timeout_budget_outcome must be none|clamped|rejected")
	}
	if !isAdapterStatus(dims.AdapterStatus) {
		return schemaMismatch(caseName, "dimensions.adapter_status must be healthy|degraded|unavailable")
	}
	if !isCircuitState(dims.AdapterCircuitState) {
		return schemaMismatch(caseName, "dimensions.adapter_circuit_state must be closed|open|half_open")
	}
	return nil
}

func schemaMismatch(caseName, message string) error {
	return &ValidationError{Code: ReasonCodeSchemaMismatch, Message: fmt.Sprintf("case %q %s", caseName, message)}
}

func canonicalizeDimensions(in CompositeFixtureDimensions) CompositeFixtureDimensions {
	return CompositeFixtureDimensions{
		ReadinessStatus:      strings.ToLower(strings.TrimSpace(in.ReadinessStatus)),
		ReadinessStrict:      in.ReadinessStrict,
		TimeoutSource:        strings.ToLower(strings.TrimSpace(in.TimeoutSource)),
		TimeoutBudgetOutcome: strings.ToLower(strings.TrimSpace(in.TimeoutBudgetOutcome)),
		AdapterStatus:        strings.ToLower(strings.TrimSpace(in.AdapterStatus)),
		AdapterRequired:      in.AdapterRequired,
		AdapterCircuitState:  strings.ToLower(strings.TrimSpace(in.AdapterCircuitState)),
	}
}

func canonicalizeObservation(in CompositeObservation) CompositeObservation {
	out := CompositeObservation{
		Readiness: CompositeReadiness{
			Status:         strings.ToLower(strings.TrimSpace(in.Readiness.Status)),
			Strict:         in.Readiness.Strict,
			PrimaryCode:    strings.TrimSpace(in.Readiness.PrimaryCode),
			ReasonTaxonomy: strings.TrimSpace(in.Readiness.ReasonTaxonomy),
		},
		Timeout: CompositeTimeout{
			Source:        strings.ToLower(strings.TrimSpace(in.Timeout.Source)),
			BudgetOutcome: strings.ToLower(strings.TrimSpace(in.Timeout.BudgetOutcome)),
			Trace:         canonicalizeStringSlice(in.Timeout.Trace, false),
		},
		AdapterHealth: CompositeAdapterHealth{
			Status:                strings.ToLower(strings.TrimSpace(in.AdapterHealth.Status)),
			Required:              in.AdapterHealth.Required,
			CircuitState:          strings.ToLower(strings.TrimSpace(in.AdapterHealth.CircuitState)),
			PrimaryCode:           strings.TrimSpace(in.AdapterHealth.PrimaryCode),
			GovernancePrimaryCode: strings.TrimSpace(in.AdapterHealth.GovernancePrimaryCode),
			ReasonTaxonomy:        strings.TrimSpace(in.AdapterHealth.ReasonTaxonomy),
		},
		Ordering: canonicalizeStringSlice(in.Ordering, false),
	}
	if out.Readiness.ReasonTaxonomy == "" {
		out.Readiness.ReasonTaxonomy = out.Readiness.PrimaryCode
	}
	if out.AdapterHealth.ReasonTaxonomy == "" {
		out.AdapterHealth.ReasonTaxonomy = out.AdapterHealth.PrimaryCode
	}
	return out
}

func canonicalizeStringSlice(items []string, sortValues bool) []string {
	if len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, raw := range items {
		v := strings.TrimSpace(raw)
		if v == "" {
			continue
		}
		out = append(out, strings.ToLower(v))
	}
	if len(out) == 0 {
		return nil
	}
	if sortValues {
		sort.Strings(out)
	}
	return out
}

func validateCanonicalSemantics(obs CompositeObservation) error {
	if !isReadinessStatus(obs.Readiness.Status) {
		return &ValidationError{Code: ReasonCodeSemanticDrift, Message: fmt.Sprintf("non-canonical readiness status %q", obs.Readiness.Status)}
	}
	if !isCanonicalReadinessCode(obs.Readiness.PrimaryCode) {
		return &ValidationError{Code: ReasonCodeSemanticDrift, Message: fmt.Sprintf("non-canonical readiness code %q", obs.Readiness.PrimaryCode)}
	}
	if !isCanonicalReadinessCode(obs.Readiness.ReasonTaxonomy) {
		return &ValidationError{Code: ReasonCodeSemanticDrift, Message: fmt.Sprintf("non-canonical readiness taxonomy %q", obs.Readiness.ReasonTaxonomy)}
	}
	if !isTimeoutSource(obs.Timeout.Source) {
		return &ValidationError{Code: ReasonCodeSemanticDrift, Message: fmt.Sprintf("non-canonical timeout source %q", obs.Timeout.Source)}
	}
	if !isTimeoutBudgetOutcome(obs.Timeout.BudgetOutcome) {
		return &ValidationError{Code: ReasonCodeSemanticDrift, Message: fmt.Sprintf("non-canonical timeout budget outcome %q", obs.Timeout.BudgetOutcome)}
	}
	if len(obs.Timeout.Trace) == 0 {
		return &ValidationError{Code: ReasonCodeSchemaMismatch, Message: "timeout.trace must not be empty"}
	}
	if !isAdapterStatus(obs.AdapterHealth.Status) {
		return &ValidationError{Code: ReasonCodeSemanticDrift, Message: fmt.Sprintf("non-canonical adapter status %q", obs.AdapterHealth.Status)}
	}
	if !isCircuitState(obs.AdapterHealth.CircuitState) {
		return &ValidationError{Code: ReasonCodeSemanticDrift, Message: fmt.Sprintf("non-canonical adapter circuit state %q", obs.AdapterHealth.CircuitState)}
	}
	if !isCanonicalAdapterCode(obs.AdapterHealth.PrimaryCode) {
		return &ValidationError{Code: ReasonCodeSemanticDrift, Message: fmt.Sprintf("non-canonical adapter code %q", obs.AdapterHealth.PrimaryCode)}
	}
	if taxonomy := strings.TrimSpace(obs.AdapterHealth.ReasonTaxonomy); taxonomy != "" && !isCanonicalAdapterTaxonomy(taxonomy) {
		return &ValidationError{Code: ReasonCodeSemanticDrift, Message: fmt.Sprintf("non-canonical adapter taxonomy %q", taxonomy)}
	}
	if governance := strings.TrimSpace(obs.AdapterHealth.GovernancePrimaryCode); governance != "" && !isCanonicalAdapterCode(governance) {
		return &ValidationError{Code: ReasonCodeSemanticDrift, Message: fmt.Sprintf("non-canonical governance code %q", governance)}
	}
	if len(obs.Ordering) == 0 {
		return &ValidationError{Code: ReasonCodeSchemaMismatch, Message: "ordering must not be empty"}
	}
	return nil
}

func assertDimensionsMatchCanonical(caseName string, dims CompositeFixtureDimensions, canonical CompositeObservation) error {
	if dims.ReadinessStatus != canonical.Readiness.Status {
		return &ValidationError{Code: ReasonCodeSemanticDrift, Message: fmt.Sprintf("case %q readiness status drift dimensions=%q canonical=%q", caseName, dims.ReadinessStatus, canonical.Readiness.Status)}
	}
	if dims.ReadinessStrict != canonical.Readiness.Strict {
		return &ValidationError{Code: ReasonCodeSemanticDrift, Message: fmt.Sprintf("case %q readiness strict drift dimensions=%v canonical=%v", caseName, dims.ReadinessStrict, canonical.Readiness.Strict)}
	}
	if dims.TimeoutSource != canonical.Timeout.Source {
		return &ValidationError{Code: ReasonCodeSemanticDrift, Message: fmt.Sprintf("case %q timeout source drift dimensions=%q canonical=%q", caseName, dims.TimeoutSource, canonical.Timeout.Source)}
	}
	if dims.TimeoutBudgetOutcome != canonical.Timeout.BudgetOutcome {
		return &ValidationError{Code: ReasonCodeSemanticDrift, Message: fmt.Sprintf("case %q timeout budget outcome drift dimensions=%q canonical=%q", caseName, dims.TimeoutBudgetOutcome, canonical.Timeout.BudgetOutcome)}
	}
	if dims.AdapterStatus != canonical.AdapterHealth.Status {
		return &ValidationError{Code: ReasonCodeSemanticDrift, Message: fmt.Sprintf("case %q adapter status drift dimensions=%q canonical=%q", caseName, dims.AdapterStatus, canonical.AdapterHealth.Status)}
	}
	if dims.AdapterRequired != canonical.AdapterHealth.Required {
		return &ValidationError{Code: ReasonCodeSemanticDrift, Message: fmt.Sprintf("case %q adapter required drift dimensions=%v canonical=%v", caseName, dims.AdapterRequired, canonical.AdapterHealth.Required)}
	}
	if dims.AdapterCircuitState != canonical.AdapterHealth.CircuitState {
		return &ValidationError{Code: ReasonCodeSemanticDrift, Message: fmt.Sprintf("case %q adapter circuit state drift dimensions=%q canonical=%q", caseName, dims.AdapterCircuitState, canonical.AdapterHealth.CircuitState)}
	}
	return nil
}

func assertObservationOrdering(caseName string, expected, actual []string, lane string) error {
	if !equalStringSlicesStrict(expected, actual) {
		return &ValidationError{
			Code:    ReasonCodeOrderingDrift,
			Message: fmt.Sprintf("case %q ordering drift on %s expected=%v actual=%v", caseName, lane, expected, actual),
		}
	}
	return nil
}

func assertObservationSemanticEquivalent(caseName string, expected, actual CompositeObservation, lane string) error {
	if err := validateCanonicalSemantics(actual); err != nil {
		return &ValidationError{Code: ReasonCodeSemanticDrift, Message: fmt.Sprintf("case %q %s semantic validation failed: %s", caseName, lane, err.Error())}
	}
	if expected.Readiness != actual.Readiness {
		return &ValidationError{Code: ReasonCodeSemanticDrift, Message: fmt.Sprintf("case %q %s readiness semantic drift expected=%#v actual=%#v", caseName, lane, expected.Readiness, actual.Readiness)}
	}
	if expected.Timeout.Source != actual.Timeout.Source ||
		expected.Timeout.BudgetOutcome != actual.Timeout.BudgetOutcome ||
		!equalStringSlicesStrict(expected.Timeout.Trace, actual.Timeout.Trace) {
		return &ValidationError{Code: ReasonCodeSemanticDrift, Message: fmt.Sprintf("case %q %s timeout semantic drift expected=%#v actual=%#v", caseName, lane, expected.Timeout, actual.Timeout)}
	}
	if expected.AdapterHealth != actual.AdapterHealth {
		return &ValidationError{Code: ReasonCodeSemanticDrift, Message: fmt.Sprintf("case %q %s adapter semantic drift expected=%#v actual=%#v", caseName, lane, expected.AdapterHealth, actual.AdapterHealth)}
	}
	return nil
}

func equalStringSlicesStrict(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if strings.TrimSpace(a[i]) != strings.TrimSpace(b[i]) {
			return false
		}
	}
	return true
}

func isReadinessStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case string(runtimeconfig.ReadinessStatusReady), string(runtimeconfig.ReadinessStatusDegraded), string(runtimeconfig.ReadinessStatusBlocked):
		return true
	default:
		return false
	}
}

func isTimeoutSource(source string) bool {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case runtimeconfig.TimeoutResolutionSourceProfile, runtimeconfig.TimeoutResolutionSourceDomain, runtimeconfig.TimeoutResolutionSourceRequest:
		return true
	default:
		return false
	}
}

func isTimeoutBudgetOutcome(outcome string) bool {
	switch strings.ToLower(strings.TrimSpace(outcome)) {
	case "none", "clamped", "rejected":
		return true
	default:
		return false
	}
}

func isAdapterStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case string(adapterhealth.StatusHealthy), string(adapterhealth.StatusDegraded), string(adapterhealth.StatusUnavailable):
		return true
	default:
		return false
	}
}

func isCircuitState(state string) bool {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case string(adapterhealth.CircuitStateClosed), string(adapterhealth.CircuitStateOpen), string(adapterhealth.CircuitStateHalfOpen):
		return true
	default:
		return false
	}
}

func isCanonicalReadinessCode(code string) bool {
	code = strings.TrimSpace(code)
	if code == "" {
		return false
	}
	_, ok := canonicalReadinessCodes[code]
	return ok
}

func isCanonicalAdapterCode(code string) bool {
	code = strings.TrimSpace(code)
	if code == "" {
		return false
	}
	_, ok := canonicalAdapterCodes[code]
	return ok
}

func isCanonicalAdapterTaxonomy(code string) bool {
	code = strings.TrimSpace(code)
	if code == "" {
		return false
	}
	if strings.HasPrefix(code, "adapter.health.") {
		return true
	}
	return isCanonicalReadinessCode(code)
}

var canonicalReadinessCodes = map[string]struct{}{
	runtimeconfig.ReadinessCodeConfigInvalid:                       {},
	runtimeconfig.ReadinessCodeStrictEscalated:                     {},
	runtimeconfig.ReadinessCodeArbitrationVersionUnsupported:       {},
	runtimeconfig.ReadinessCodeArbitrationVersionMismatch:          {},
	runtimeconfig.ReadinessCodeSchedulerFallback:                   {},
	runtimeconfig.ReadinessCodeSchedulerActivationError:            {},
	runtimeconfig.ReadinessCodeMailboxFallback:                     {},
	runtimeconfig.ReadinessCodeMailboxActivationError:              {},
	runtimeconfig.ReadinessCodeRecoveryFallback:                    {},
	runtimeconfig.ReadinessCodeRecoveryActivationError:             {},
	runtimeconfig.ReadinessCodeRuntimeManagerUnavailable:           {},
	runtimeconfig.ReadinessCodeAdapterRequiredUnavailable:          {},
	runtimeconfig.ReadinessCodeAdapterOptionalUnavailable:          {},
	runtimeconfig.ReadinessCodeAdapterDegraded:                     {},
	runtimeconfig.ReadinessCodeAdapterRequiredCircuitOpen:          {},
	runtimeconfig.ReadinessCodeAdapterOptionalCircuitOpen:          {},
	runtimeconfig.ReadinessCodeAdapterHalfOpenDegraded:             {},
	runtimeconfig.ReadinessCodeAdapterGovernanceRecovered:          {},
	runtimeconfig.ReadinessCodeSandboxRequiredUnavailable:          {},
	runtimeconfig.ReadinessCodeSandboxOptionalUnavailable:          {},
	runtimeconfig.ReadinessCodeSandboxProfileInvalid:               {},
	runtimeconfig.ReadinessCodeSandboxCapabilityMismatch:           {},
	runtimeconfig.ReadinessCodeSandboxSessionModeUnsupported:       {},
	runtimeconfig.ReadinessCodeSandboxRolloutPhaseInvalid:          {},
	runtimeconfig.ReadinessCodeSandboxRolloutHealthBreached:        {},
	runtimeconfig.ReadinessCodeSandboxRolloutFrozen:                {},
	runtimeconfig.ReadinessCodeSandboxRolloutCapacityBlocked:       {},
	runtimeconfig.ReadinessCodeMemoryModeInvalid:                   {},
	runtimeconfig.ReadinessCodeMemoryProfileMissing:                {},
	runtimeconfig.ReadinessCodeMemoryProviderNotSupported:          {},
	runtimeconfig.ReadinessCodeMemorySPIUnavailable:                {},
	runtimeconfig.ReadinessCodeMemoryFilesystemPathInvalid:         {},
	runtimeconfig.ReadinessCodeMemoryContractVersionMismatch:       {},
	runtimeconfig.ReadinessCodeMemoryFallbackPolicyConflict:        {},
	runtimeconfig.ReadinessCodeMemoryFallbackTargetUnavailable:     {},
	runtimeconfig.ReadinessCodeObservabilityExportProfileInvalid:   {},
	runtimeconfig.ReadinessCodeObservabilityExportSinkUnavailable:  {},
	runtimeconfig.ReadinessCodeObservabilityExportAuthInvalid:      {},
	runtimeconfig.ReadinessCodeDiagnosticsBundleOutputUnavailable:  {},
	runtimeconfig.ReadinessCodeDiagnosticsBundlePolicyInvalid:      {},
	runtimeconfig.ReadinessCodeReactLoopDisabled:                   {},
	runtimeconfig.ReadinessCodeReactStreamDispatchUnavailable:      {},
	runtimeconfig.ReadinessCodeReactProviderToolCallingUnsupported: {},
	runtimeconfig.ReadinessCodeReactToolRegistryUnavailable:        {},
	runtimeconfig.ReadinessCodeReactSandboxDependencyUnavailable:   {},
	runtimeconfig.ReadinessAdmissionCodeBypassDisabled:             {},
	runtimeconfig.ReadinessAdmissionCodeReady:                      {},
	runtimeconfig.ReadinessAdmissionCodeBlocked:                    {},
	runtimeconfig.ReadinessAdmissionCodeDegradedAllow:              {},
	runtimeconfig.ReadinessAdmissionCodeDegradedDeny:               {},
	runtimeconfig.ReadinessAdmissionCodeSandboxFrozen:              {},
	runtimeconfig.ReadinessAdmissionCodeSandboxThrottle:            {},
	runtimeconfig.ReadinessAdmissionCodeSandboxThrottledDeny:       {},
	runtimeconfig.ReadinessAdmissionCodeSandboxCapacityDeny:        {},
	runtimeconfig.ReadinessAdmissionCodeUnknownStatus:              {},
	runtimeconfig.ReadinessAdmissionCodeManagerNotReady:            {},
}

var canonicalAdapterCodes = map[string]struct{}{
	adapterhealth.CodeHealthy:         {},
	adapterhealth.CodeDegraded:        {},
	adapterhealth.CodeUnavailable:     {},
	adapterhealth.CodeTargetNotFound:  {},
	adapterhealth.CodeProbeTimeout:    {},
	adapterhealth.CodeProbeFailed:     {},
	adapterhealth.CodeUnknownStatus:   {},
	adapterhealth.CodeBackoffThrottle: {},
	adapterhealth.CodeCircuitOpen:     {},
	adapterhealth.CodeCircuitHalfOpen: {},
	adapterhealth.CodeCircuitRecover:  {},
	adapterhealth.CodeHalfOpenReject:  {},
}

type compositeCoverage struct {
	readinessStatus map[string]bool
	readinessStrict map[bool]bool
	timeoutSource   map[string]bool
	timeoutBudget   map[string]bool
	adapterStatus   map[string]bool
	adapterRequired map[bool]bool
	circuitState    map[string]bool
}

func newCompositeCoverage() *compositeCoverage {
	return &compositeCoverage{
		readinessStatus: map[string]bool{},
		readinessStrict: map[bool]bool{},
		timeoutSource:   map[string]bool{},
		timeoutBudget:   map[string]bool{},
		adapterStatus:   map[string]bool{},
		adapterRequired: map[bool]bool{},
		circuitState:    map[string]bool{},
	}
}

func (c *compositeCoverage) mark(dims CompositeFixtureDimensions) {
	c.readinessStatus[dims.ReadinessStatus] = true
	c.readinessStrict[dims.ReadinessStrict] = true
	c.timeoutSource[dims.TimeoutSource] = true
	c.timeoutBudget[dims.TimeoutBudgetOutcome] = true
	c.adapterStatus[dims.AdapterStatus] = true
	c.adapterRequired[dims.AdapterRequired] = true
	c.circuitState[dims.AdapterCircuitState] = true
}

func (c *compositeCoverage) validate() error {
	requiredReadiness := []string{string(runtimeconfig.ReadinessStatusReady), string(runtimeconfig.ReadinessStatusDegraded), string(runtimeconfig.ReadinessStatusBlocked)}
	for _, status := range requiredReadiness {
		if !c.readinessStatus[status] {
			return &ValidationError{Code: ReasonCodeSchemaMismatch, Message: fmt.Sprintf("matrix coverage missing readiness_status=%s", status)}
		}
	}
	if !c.readinessStrict[true] || !c.readinessStrict[false] {
		return &ValidationError{Code: ReasonCodeSchemaMismatch, Message: "matrix coverage missing readiness_strict true/false"}
	}
	requiredTimeoutSource := []string{runtimeconfig.TimeoutResolutionSourceProfile, runtimeconfig.TimeoutResolutionSourceDomain, runtimeconfig.TimeoutResolutionSourceRequest}
	for _, source := range requiredTimeoutSource {
		if !c.timeoutSource[source] {
			return &ValidationError{Code: ReasonCodeSchemaMismatch, Message: fmt.Sprintf("matrix coverage missing timeout_source=%s", source)}
		}
	}
	if !c.timeoutBudget["clamped"] || !c.timeoutBudget["rejected"] {
		return &ValidationError{Code: ReasonCodeSchemaMismatch, Message: "matrix coverage missing timeout_budget_outcome clamped/rejected"}
	}
	requiredAdapterStatus := []string{string(adapterhealth.StatusHealthy), string(adapterhealth.StatusDegraded), string(adapterhealth.StatusUnavailable)}
	for _, status := range requiredAdapterStatus {
		if !c.adapterStatus[status] {
			return &ValidationError{Code: ReasonCodeSchemaMismatch, Message: fmt.Sprintf("matrix coverage missing adapter_status=%s", status)}
		}
	}
	if !c.adapterRequired[true] || !c.adapterRequired[false] {
		return &ValidationError{Code: ReasonCodeSchemaMismatch, Message: "matrix coverage missing adapter_required true/false"}
	}
	requiredCircuitStates := []string{string(adapterhealth.CircuitStateClosed), string(adapterhealth.CircuitStateOpen), string(adapterhealth.CircuitStateHalfOpen)}
	for _, state := range requiredCircuitStates {
		if !c.circuitState[state] {
			return &ValidationError{Code: ReasonCodeSchemaMismatch, Message: fmt.Sprintf("matrix coverage missing adapter_circuit_state=%s", state)}
		}
	}
	return nil
}
