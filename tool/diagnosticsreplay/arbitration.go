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
	ArbitrationFixtureVersionA49V1 = "a49.v1"
	ArbitrationFixtureVersionA50V1 = "a50.v1"

	ReasonCodePrecedenceDrift           = "precedence_drift"
	ReasonCodeTieBreakDrift             = "tie_break_drift"
	ReasonCodeTaxonomyDrift             = "taxonomy_drift"
	ReasonCodeSecondaryOrderDrift       = "secondary_order_drift"
	ReasonCodeSecondaryCountDrift       = "secondary_count_drift"
	ReasonCodeHintTaxonomyDrift         = "hint_taxonomy_drift"
	ReasonCodeRuleVersionDrift          = "rule_version_drift"
	ReasonCodeVersionMismatch           = "version_mismatch"
	ReasonCodeUnsupportedVersion        = "unsupported_version"
	ReasonCodeCrossVersionSemanticDrift = "cross_version_semantic_drift"
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
	RuntimePrimaryDomain                   string   `json:"runtime_primary_domain"`
	RuntimePrimaryCode                     string   `json:"runtime_primary_code"`
	RuntimePrimarySource                   string   `json:"runtime_primary_source"`
	RuntimePrimaryConflictTotal            int      `json:"runtime_primary_conflict_total"`
	RuntimeSecondaryReasonCodes            []string `json:"runtime_secondary_reason_codes,omitempty"`
	RuntimeSecondaryReasonCount            int      `json:"runtime_secondary_reason_count,omitempty"`
	RuntimeArbitrationRuleVersion          string   `json:"runtime_arbitration_rule_version,omitempty"`
	RuntimeArbitrationRuleRequestedVersion string   `json:"runtime_arbitration_rule_requested_version,omitempty"`
	RuntimeArbitrationRuleEffectiveVersion string   `json:"runtime_arbitration_rule_effective_version,omitempty"`
	RuntimeArbitrationRuleVersionSource    string   `json:"runtime_arbitration_rule_version_source,omitempty"`
	RuntimeArbitrationRulePolicyAction     string   `json:"runtime_arbitration_rule_policy_action,omitempty"`
	RuntimeArbitrationRuleUnsupportedTotal int      `json:"runtime_arbitration_rule_unsupported_total,omitempty"`
	RuntimeArbitrationRuleMismatchTotal    int      `json:"runtime_arbitration_rule_mismatch_total,omitempty"`
	RuntimeRemediationHintCode             string   `json:"runtime_remediation_hint_code,omitempty"`
	RuntimeRemediationHintDomain           string   `json:"runtime_remediation_hint_domain,omitempty"`
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
	version := strings.TrimSpace(fixture.Version)
	if version == "" {
		return ArbitrationFixture{}, &ValidationError{Code: ReasonCodeSchemaMismatch, Message: "version is required"}
	}
	if version != ArbitrationFixtureVersionA48V1 &&
		version != ArbitrationFixtureVersionA49V1 &&
		version != ArbitrationFixtureVersionA50V1 {
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
	fixture.Version = version
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
	version := strings.TrimSpace(fixture.Version)
	cases := append([]ArbitrationFixtureCase(nil), fixture.Cases...)
	sort.Slice(cases, func(i, j int) bool {
		return strings.TrimSpace(cases[i].Name) < strings.TrimSpace(cases[j].Name)
	})
	out := ArbitrationReplayOutput{
		Version: version,
		Cases:   make([]ArbitrationNormalizedOutput, 0, len(cases)),
	}
	for _, tc := range cases {
		name := strings.TrimSpace(tc.Name)
		expected := canonicalizeArbitrationObservation(tc.Expected)
		run := canonicalizeArbitrationObservation(tc.Run)
		stream := canonicalizeArbitrationObservation(tc.Stream)
		if err := validateArbitrationObservation(version, name, "expected", expected); err != nil {
			return ArbitrationReplayOutput{}, err
		}
		if err := validateArbitrationObservation(version, name, "run", run); err != nil {
			return ArbitrationReplayOutput{}, err
		}
		if err := validateArbitrationObservation(version, name, "stream", stream); err != nil {
			return ArbitrationReplayOutput{}, err
		}
		if err := assertArbitrationEquivalent(version, name, expected, run, "run"); err != nil {
			return ArbitrationReplayOutput{}, err
		}
		if err := assertArbitrationEquivalent(version, name, expected, stream, "stream"); err != nil {
			return ArbitrationReplayOutput{}, err
		}
		if err := assertArbitrationEquivalent(version, name, run, stream, "run/stream"); err != nil {
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
		RuntimePrimaryDomain:                   strings.ToLower(strings.TrimSpace(in.RuntimePrimaryDomain)),
		RuntimePrimaryCode:                     strings.TrimSpace(in.RuntimePrimaryCode),
		RuntimePrimarySource:                   strings.ToLower(strings.TrimSpace(in.RuntimePrimarySource)),
		RuntimePrimaryConflictTotal:            in.RuntimePrimaryConflictTotal,
		RuntimeSecondaryReasonCount:            in.RuntimeSecondaryReasonCount,
		RuntimeArbitrationRuleVersion:          strings.ToLower(strings.TrimSpace(in.RuntimeArbitrationRuleVersion)),
		RuntimeArbitrationRuleRequestedVersion: strings.ToLower(strings.TrimSpace(in.RuntimeArbitrationRuleRequestedVersion)),
		RuntimeArbitrationRuleEffectiveVersion: strings.ToLower(strings.TrimSpace(in.RuntimeArbitrationRuleEffectiveVersion)),
		RuntimeArbitrationRuleVersionSource:    strings.ToLower(strings.TrimSpace(in.RuntimeArbitrationRuleVersionSource)),
		RuntimeArbitrationRulePolicyAction:     strings.ToLower(strings.TrimSpace(in.RuntimeArbitrationRulePolicyAction)),
		RuntimeArbitrationRuleUnsupportedTotal: in.RuntimeArbitrationRuleUnsupportedTotal,
		RuntimeArbitrationRuleMismatchTotal:    in.RuntimeArbitrationRuleMismatchTotal,
		RuntimeRemediationHintCode:             strings.TrimSpace(in.RuntimeRemediationHintCode),
		RuntimeRemediationHintDomain:           strings.ToLower(strings.TrimSpace(in.RuntimeRemediationHintDomain)),
	}
	if out.RuntimePrimaryConflictTotal < 0 {
		out.RuntimePrimaryConflictTotal = 0
	}
	if out.RuntimeSecondaryReasonCount < 0 {
		out.RuntimeSecondaryReasonCount = 0
	}
	if out.RuntimeArbitrationRuleUnsupportedTotal < 0 {
		out.RuntimeArbitrationRuleUnsupportedTotal = 0
	}
	if out.RuntimeArbitrationRuleMismatchTotal < 0 {
		out.RuntimeArbitrationRuleMismatchTotal = 0
	}
	for i := range in.RuntimeSecondaryReasonCodes {
		code := strings.TrimSpace(in.RuntimeSecondaryReasonCodes[i])
		if code == "" {
			continue
		}
		out.RuntimeSecondaryReasonCodes = append(out.RuntimeSecondaryReasonCodes, code)
	}
	if len(out.RuntimeSecondaryReasonCodes) == 0 {
		out.RuntimeSecondaryReasonCodes = nil
	}
	return out
}

func validateArbitrationObservation(version, caseName, lane string, obs ArbitrationObservation) error {
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
	if version == ArbitrationFixtureVersionA48V1 {
		return nil
	}
	if version == ArbitrationFixtureVersionA49V1 {
		if strings.TrimSpace(obs.RuntimeArbitrationRuleVersion) == "" {
			return &ValidationError{
				Code:    ReasonCodeSchemaMismatch,
				Message: fmt.Sprintf("case %q %s.runtime_arbitration_rule_version is required", caseName, lane),
			}
		}
		if strings.TrimSpace(obs.RuntimeArbitrationRuleVersion) != runtimeconfig.RuntimeArbitrationRuleVersionA49V1 {
			return &ValidationError{
				Code:    ReasonCodeRuleVersionDrift,
				Message: fmt.Sprintf("case %q %s rule version drift want=%q got=%q", caseName, lane, runtimeconfig.RuntimeArbitrationRuleVersionA49V1, obs.RuntimeArbitrationRuleVersion),
			}
		}
	}
	if len(obs.RuntimeSecondaryReasonCodes) > runtimeconfig.RuntimeArbitrationMaxSecondary {
		return &ValidationError{
			Code:    ReasonCodeSecondaryCountDrift,
			Message: fmt.Sprintf("case %q %s runtime_secondary_reason_codes exceeds max=%d", caseName, lane, runtimeconfig.RuntimeArbitrationMaxSecondary),
		}
	}
	if obs.RuntimeSecondaryReasonCount < len(obs.RuntimeSecondaryReasonCodes) {
		return &ValidationError{
			Code:    ReasonCodeSecondaryCountDrift,
			Message: fmt.Sprintf("case %q %s runtime_secondary_reason_count=%d < len(codes)=%d", caseName, lane, obs.RuntimeSecondaryReasonCount, len(obs.RuntimeSecondaryReasonCodes)),
		}
	}
	seenSecondary := map[string]struct{}{}
	for i := range obs.RuntimeSecondaryReasonCodes {
		secondary := strings.TrimSpace(obs.RuntimeSecondaryReasonCodes[i])
		if !isCanonicalArbitrationCode(secondary) {
			return &ValidationError{
				Code:    ReasonCodeTaxonomyDrift,
				Message: fmt.Sprintf("case %q %s non-canonical secondary code %q", caseName, lane, secondary),
			}
		}
		if secondary == obs.RuntimePrimaryCode {
			return &ValidationError{
				Code:    ReasonCodeSecondaryOrderDrift,
				Message: fmt.Sprintf("case %q %s secondary code repeats primary %q", caseName, lane, secondary),
			}
		}
		if _, ok := seenSecondary[secondary]; ok {
			return &ValidationError{
				Code:    ReasonCodeSecondaryOrderDrift,
				Message: fmt.Sprintf("case %q %s duplicate secondary code %q", caseName, lane, secondary),
			}
		}
		seenSecondary[secondary] = struct{}{}
	}
	hintCode, hintDomain, ok := runtimeconfig.RemediationHintForPrimaryCode(obs.RuntimePrimaryCode)
	if !ok {
		return &ValidationError{
			Code:    ReasonCodeHintTaxonomyDrift,
			Message: fmt.Sprintf("case %q %s unsupported primary for hint taxonomy %q", caseName, lane, obs.RuntimePrimaryCode),
		}
	}
	if strings.TrimSpace(obs.RuntimeRemediationHintCode) == "" || strings.TrimSpace(obs.RuntimeRemediationHintDomain) == "" {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s remediation hint fields are required", caseName, lane),
		}
	}
	if strings.TrimSpace(obs.RuntimeRemediationHintCode) != hintCode || strings.TrimSpace(obs.RuntimeRemediationHintDomain) != hintDomain {
		return &ValidationError{
			Code:    ReasonCodeHintTaxonomyDrift,
			Message: fmt.Sprintf("case %q %s hint taxonomy drift want=%s/%s got=%s/%s", caseName, lane, hintDomain, hintCode, obs.RuntimeRemediationHintDomain, obs.RuntimeRemediationHintCode),
		}
	}
	if version != ArbitrationFixtureVersionA50V1 {
		return nil
	}
	if obs.RuntimeArbitrationRuleVersionSource != runtimeconfig.RuntimeArbitrationVersionSourceDefault &&
		obs.RuntimeArbitrationRuleVersionSource != runtimeconfig.RuntimeArbitrationVersionSourceRequested {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s runtime_arbitration_rule_version_source must be default|requested", caseName, lane),
		}
	}
	switch obs.RuntimeArbitrationRulePolicyAction {
	case runtimeconfig.RuntimeArbitrationPolicyActionNone,
		runtimeconfig.RuntimeArbitrationPolicyActionDisabled,
		runtimeconfig.RuntimeArbitrationPolicyActionFailFastUnsupported,
		runtimeconfig.RuntimeArbitrationPolicyActionFailFastMismatch:
	default:
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s runtime_arbitration_rule_policy_action is invalid", caseName, lane),
		}
	}
	if effective := strings.TrimSpace(obs.RuntimeArbitrationRuleEffectiveVersion); effective != "" && !isSupportedA50Version(effective) {
		return &ValidationError{
			Code:    ReasonCodeRuleVersionDrift,
			Message: fmt.Sprintf("case %q %s runtime_arbitration_rule_effective_version=%q is not registered", caseName, lane, effective),
		}
	}
	if ruleVersion := strings.TrimSpace(obs.RuntimeArbitrationRuleVersion); ruleVersion != "" {
		if !isSupportedA50Version(ruleVersion) {
			return &ValidationError{
				Code:    ReasonCodeRuleVersionDrift,
				Message: fmt.Sprintf("case %q %s runtime_arbitration_rule_version=%q is not registered", caseName, lane, ruleVersion),
			}
		}
		if effective := strings.TrimSpace(obs.RuntimeArbitrationRuleEffectiveVersion); effective != "" && effective != ruleVersion {
			return &ValidationError{
				Code:    ReasonCodeCrossVersionSemanticDrift,
				Message: fmt.Sprintf("case %q %s runtime_arbitration_rule_version=%q mismatches effective=%q", caseName, lane, ruleVersion, effective),
			}
		}
	}
	if requested := strings.TrimSpace(obs.RuntimeArbitrationRuleRequestedVersion); requested != "" &&
		!isSupportedA50Version(requested) &&
		obs.RuntimePrimaryCode != runtimeconfig.ReadinessCodeArbitrationVersionUnsupported {
		return &ValidationError{
			Code:    ReasonCodeUnsupportedVersion,
			Message: fmt.Sprintf("case %q %s requested version %q must produce unsupported classification", caseName, lane, requested),
		}
	}
	switch strings.TrimSpace(obs.RuntimePrimaryCode) {
	case runtimeconfig.ReadinessCodeArbitrationVersionUnsupported:
		if obs.RuntimeArbitrationRulePolicyAction != runtimeconfig.RuntimeArbitrationPolicyActionFailFastUnsupported ||
			obs.RuntimeArbitrationRuleUnsupportedTotal <= 0 {
			return &ValidationError{
				Code:    ReasonCodeUnsupportedVersion,
				Message: fmt.Sprintf("case %q %s unsupported version must set fail_fast_unsupported_version and unsupported_total>0", caseName, lane),
			}
		}
	case runtimeconfig.ReadinessCodeArbitrationVersionMismatch:
		if obs.RuntimeArbitrationRulePolicyAction != runtimeconfig.RuntimeArbitrationPolicyActionFailFastMismatch ||
			obs.RuntimeArbitrationRuleMismatchTotal <= 0 {
			return &ValidationError{
				Code:    ReasonCodeVersionMismatch,
				Message: fmt.Sprintf("case %q %s version mismatch must set fail_fast_version_mismatch and mismatch_total>0", caseName, lane),
			}
		}
	default:
		if obs.RuntimeArbitrationRuleUnsupportedTotal > 0 || obs.RuntimeArbitrationRuleMismatchTotal > 0 {
			return &ValidationError{
				Code:    ReasonCodeCrossVersionSemanticDrift,
				Message: fmt.Sprintf("case %q %s non-version-failure code has unsupported/mismatch counters", caseName, lane),
			}
		}
		if strings.TrimSpace(obs.RuntimeArbitrationRuleEffectiveVersion) == "" {
			return &ValidationError{
				Code:    ReasonCodeCrossVersionSemanticDrift,
				Message: fmt.Sprintf("case %q %s effective version is required for non-version-failure paths", caseName, lane),
			}
		}
	}
	return nil
}

func assertArbitrationEquivalent(version, caseName string, expected, actual ArbitrationObservation, lane string) error {
	if arbitrationObservationsEqual(version, expected, actual) {
		return nil
	}
	if version == ArbitrationFixtureVersionA50V1 {
		if expected.RuntimePrimaryCode != actual.RuntimePrimaryCode {
			if expected.RuntimePrimaryCode == runtimeconfig.ReadinessCodeArbitrationVersionUnsupported ||
				actual.RuntimePrimaryCode == runtimeconfig.ReadinessCodeArbitrationVersionUnsupported {
				return &ValidationError{
					Code: ReasonCodeUnsupportedVersion,
					Message: fmt.Sprintf(
						"case %q %s unsupported version classification drift expected=%q actual=%q",
						caseName,
						lane,
						expected.RuntimePrimaryCode,
						actual.RuntimePrimaryCode,
					),
				}
			}
			if expected.RuntimePrimaryCode == runtimeconfig.ReadinessCodeArbitrationVersionMismatch ||
				actual.RuntimePrimaryCode == runtimeconfig.ReadinessCodeArbitrationVersionMismatch {
				return &ValidationError{
					Code: ReasonCodeVersionMismatch,
					Message: fmt.Sprintf(
						"case %q %s version mismatch classification drift expected=%q actual=%q",
						caseName,
						lane,
						expected.RuntimePrimaryCode,
						actual.RuntimePrimaryCode,
					),
				}
			}
		}
		if expected.RuntimeArbitrationRuleVersion != actual.RuntimeArbitrationRuleVersion ||
			expected.RuntimeArbitrationRuleRequestedVersion != actual.RuntimeArbitrationRuleRequestedVersion ||
			expected.RuntimeArbitrationRuleEffectiveVersion != actual.RuntimeArbitrationRuleEffectiveVersion ||
			expected.RuntimeArbitrationRuleVersionSource != actual.RuntimeArbitrationRuleVersionSource ||
			expected.RuntimeArbitrationRulePolicyAction != actual.RuntimeArbitrationRulePolicyAction ||
			expected.RuntimeArbitrationRuleUnsupportedTotal != actual.RuntimeArbitrationRuleUnsupportedTotal ||
			expected.RuntimeArbitrationRuleMismatchTotal != actual.RuntimeArbitrationRuleMismatchTotal {
			if expected.RuntimePrimaryCode == runtimeconfig.ReadinessCodeArbitrationVersionUnsupported ||
				actual.RuntimePrimaryCode == runtimeconfig.ReadinessCodeArbitrationVersionUnsupported {
				return &ValidationError{
					Code: ReasonCodeUnsupportedVersion,
					Message: fmt.Sprintf(
						"case %q %s unsupported version governance drift expected=%#v actual=%#v",
						caseName,
						lane,
						expected,
						actual,
					),
				}
			}
			if expected.RuntimePrimaryCode == runtimeconfig.ReadinessCodeArbitrationVersionMismatch ||
				actual.RuntimePrimaryCode == runtimeconfig.ReadinessCodeArbitrationVersionMismatch {
				return &ValidationError{
					Code: ReasonCodeVersionMismatch,
					Message: fmt.Sprintf(
						"case %q %s version mismatch governance drift expected=%#v actual=%#v",
						caseName,
						lane,
						expected,
						actual,
					),
				}
			}
			return &ValidationError{
				Code: ReasonCodeCrossVersionSemanticDrift,
				Message: fmt.Sprintf(
					"case %q %s cross-version semantic drift expected=%#v actual=%#v",
					caseName,
					lane,
					expected,
					actual,
				),
			}
		}
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
	if version == ArbitrationFixtureVersionA49V1 {
		if expected.RuntimeArbitrationRuleVersion != actual.RuntimeArbitrationRuleVersion {
			return &ValidationError{
				Code: ReasonCodeRuleVersionDrift,
				Message: fmt.Sprintf(
					"case %q %s rule version drift expected=%q actual=%q",
					caseName,
					lane,
					expected.RuntimeArbitrationRuleVersion,
					actual.RuntimeArbitrationRuleVersion,
				),
			}
		}
		if expected.RuntimeSecondaryReasonCount != actual.RuntimeSecondaryReasonCount {
			return &ValidationError{
				Code: ReasonCodeSecondaryCountDrift,
				Message: fmt.Sprintf(
					"case %q %s secondary count drift expected=%d actual=%d",
					caseName,
					lane,
					expected.RuntimeSecondaryReasonCount,
					actual.RuntimeSecondaryReasonCount,
				),
			}
		}
		if !equalStringSlice(expected.RuntimeSecondaryReasonCodes, actual.RuntimeSecondaryReasonCodes) {
			return &ValidationError{
				Code: ReasonCodeSecondaryOrderDrift,
				Message: fmt.Sprintf(
					"case %q %s secondary order drift expected=%#v actual=%#v",
					caseName,
					lane,
					expected.RuntimeSecondaryReasonCodes,
					actual.RuntimeSecondaryReasonCodes,
				),
			}
		}
	}
	if version == ArbitrationFixtureVersionA50V1 {
		if expected.RuntimeSecondaryReasonCount != actual.RuntimeSecondaryReasonCount {
			return &ValidationError{
				Code: ReasonCodeSecondaryCountDrift,
				Message: fmt.Sprintf(
					"case %q %s secondary count drift expected=%d actual=%d",
					caseName,
					lane,
					expected.RuntimeSecondaryReasonCount,
					actual.RuntimeSecondaryReasonCount,
				),
			}
		}
		if !equalStringSlice(expected.RuntimeSecondaryReasonCodes, actual.RuntimeSecondaryReasonCodes) {
			return &ValidationError{
				Code: ReasonCodeSecondaryOrderDrift,
				Message: fmt.Sprintf(
					"case %q %s secondary order drift expected=%#v actual=%#v",
					caseName,
					lane,
					expected.RuntimeSecondaryReasonCodes,
					actual.RuntimeSecondaryReasonCodes,
				),
			}
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
	if (version == ArbitrationFixtureVersionA49V1 || version == ArbitrationFixtureVersionA50V1) &&
		(expected.RuntimeRemediationHintCode != actual.RuntimeRemediationHintCode ||
			expected.RuntimeRemediationHintDomain != actual.RuntimeRemediationHintDomain) {
		return &ValidationError{
			Code: ReasonCodeHintTaxonomyDrift,
			Message: fmt.Sprintf(
				"case %q %s hint taxonomy drift expected=%s/%s actual=%s/%s",
				caseName,
				lane,
				expected.RuntimeRemediationHintDomain,
				expected.RuntimeRemediationHintCode,
				actual.RuntimeRemediationHintDomain,
				actual.RuntimeRemediationHintCode,
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

func arbitrationObservationsEqual(version string, left, right ArbitrationObservation) bool {
	if left.RuntimePrimaryDomain != right.RuntimePrimaryDomain ||
		left.RuntimePrimaryCode != right.RuntimePrimaryCode ||
		left.RuntimePrimarySource != right.RuntimePrimarySource ||
		left.RuntimePrimaryConflictTotal != right.RuntimePrimaryConflictTotal {
		return false
	}
	if version != ArbitrationFixtureVersionA49V1 {
		if version == ArbitrationFixtureVersionA50V1 {
			if left.RuntimeSecondaryReasonCount != right.RuntimeSecondaryReasonCount ||
				left.RuntimeArbitrationRuleVersion != right.RuntimeArbitrationRuleVersion ||
				left.RuntimeArbitrationRuleRequestedVersion != right.RuntimeArbitrationRuleRequestedVersion ||
				left.RuntimeArbitrationRuleEffectiveVersion != right.RuntimeArbitrationRuleEffectiveVersion ||
				left.RuntimeArbitrationRuleVersionSource != right.RuntimeArbitrationRuleVersionSource ||
				left.RuntimeArbitrationRulePolicyAction != right.RuntimeArbitrationRulePolicyAction ||
				left.RuntimeArbitrationRuleUnsupportedTotal != right.RuntimeArbitrationRuleUnsupportedTotal ||
				left.RuntimeArbitrationRuleMismatchTotal != right.RuntimeArbitrationRuleMismatchTotal ||
				left.RuntimeRemediationHintCode != right.RuntimeRemediationHintCode ||
				left.RuntimeRemediationHintDomain != right.RuntimeRemediationHintDomain {
				return false
			}
			return equalStringSlice(left.RuntimeSecondaryReasonCodes, right.RuntimeSecondaryReasonCodes)
		}
		return true
	}
	if left.RuntimeSecondaryReasonCount != right.RuntimeSecondaryReasonCount ||
		left.RuntimeArbitrationRuleVersion != right.RuntimeArbitrationRuleVersion ||
		left.RuntimeRemediationHintCode != right.RuntimeRemediationHintCode ||
		left.RuntimeRemediationHintDomain != right.RuntimeRemediationHintDomain {
		return false
	}
	return equalStringSlice(left.RuntimeSecondaryReasonCodes, right.RuntimeSecondaryReasonCodes)
}

func equalStringSlice(left, right []string) bool {
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
	case runtimeconfig.ReadinessCodeArbitrationVersionUnsupported, runtimeconfig.ReadinessCodeArbitrationVersionMismatch:
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

func isSupportedA50Version(version string) bool {
	normalized := strings.ToLower(strings.TrimSpace(version))
	if normalized == "" {
		return false
	}
	for _, item := range runtimeconfig.RegisteredRuntimeArbitrationRuleVersions() {
		if normalized == strings.ToLower(strings.TrimSpace(item)) {
			return true
		}
	}
	return false
}
