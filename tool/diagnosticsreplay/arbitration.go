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
	ArbitrationFixtureVersionA48V1    = "a48.v1"
	ArbitrationFixtureVersionA49V1    = "a49.v1"
	ArbitrationFixtureVersionA50V1    = "a50.v1"
	ArbitrationFixtureVersionA51V1    = "a51.v1"
	ArbitrationFixtureVersionA52V1    = "a52.v1"
	ArbitrationFixtureVersionMemoryV1 = "memory.v1"

	ReasonCodePrecedenceDrift               = "precedence_drift"
	ReasonCodeTieBreakDrift                 = "tie_break_drift"
	ReasonCodeTaxonomyDrift                 = "taxonomy_drift"
	ReasonCodeSecondaryOrderDrift           = "secondary_order_drift"
	ReasonCodeSecondaryCountDrift           = "secondary_count_drift"
	ReasonCodeHintTaxonomyDrift             = "hint_taxonomy_drift"
	ReasonCodeRuleVersionDrift              = "rule_version_drift"
	ReasonCodeVersionMismatch               = "version_mismatch"
	ReasonCodeUnsupportedVersion            = "unsupported_version"
	ReasonCodeCrossVersionSemanticDrift     = "cross_version_semantic_drift"
	ReasonCodeSandboxPolicyDrift            = "sandbox_policy_drift"
	ReasonCodeSandboxFallbackDrift          = "sandbox_fallback_drift"
	ReasonCodeSandboxTimeoutDrift           = "sandbox_timeout_drift"
	ReasonCodeSandboxCapabilityDrift        = "sandbox_capability_drift"
	ReasonCodeSandboxResourcePolicyDrift    = "sandbox_resource_policy_drift"
	ReasonCodeSandboxSessionLifecycleDrift  = "sandbox_session_lifecycle_drift"
	ReasonCodeSandboxRolloutPhaseDrift      = "sandbox_rollout_phase_drift"
	ReasonCodeSandboxHealthBudgetDrift      = "sandbox_health_budget_drift"
	ReasonCodeSandboxCapacityActionDrift    = "sandbox_capacity_action_drift"
	ReasonCodeSandboxFreezeStateDrift       = "sandbox_freeze_state_drift"
	ReasonCodeMemoryModeDrift               = "memory_mode_drift"
	ReasonCodeMemoryProfileDrift            = "memory_profile_drift"
	ReasonCodeMemoryContractVersionDrift    = "memory_contract_version_drift"
	ReasonCodeMemoryFallbackDrift           = "memory_fallback_drift"
	ReasonCodeMemoryErrorTaxonomyDrift      = "memory_error_taxonomy_drift"
	ReasonCodeMemoryOperationAggregateDrift = "memory_operation_aggregate_drift"
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
	SandboxMode                            string   `json:"sandbox_mode,omitempty"`
	SandboxBackend                         string   `json:"sandbox_backend,omitempty"`
	SandboxProfile                         string   `json:"sandbox_profile,omitempty"`
	SandboxSessionMode                     string   `json:"sandbox_session_mode,omitempty"`
	SandboxRequiredCapabilities            []string `json:"sandbox_required_capabilities,omitempty"`
	SandboxDecision                        string   `json:"sandbox_decision,omitempty"`
	SandboxReasonCode                      string   `json:"sandbox_reason_code,omitempty"`
	SandboxFallbackUsed                    bool     `json:"sandbox_fallback_used,omitempty"`
	SandboxFallbackReason                  string   `json:"sandbox_fallback_reason,omitempty"`
	SandboxTimeoutTotal                    int      `json:"sandbox_timeout_total,omitempty"`
	SandboxLaunchFailedTotal               int      `json:"sandbox_launch_failed_total,omitempty"`
	SandboxCapabilityMismatchTotal         int      `json:"sandbox_capability_mismatch_total,omitempty"`
	SandboxQueueWaitMsP95                  int64    `json:"sandbox_queue_wait_ms_p95,omitempty"`
	SandboxExecLatencyMsP95                int64    `json:"sandbox_exec_latency_ms_p95,omitempty"`
	SandboxExitCodeLast                    int      `json:"sandbox_exit_code_last,omitempty"`
	SandboxOOMTotal                        int      `json:"sandbox_oom_total,omitempty"`
	SandboxResourceCPUMsTotal              int64    `json:"sandbox_resource_cpu_ms_total,omitempty"`
	SandboxResourceMemoryPeakBytesP95      int64    `json:"sandbox_resource_memory_peak_bytes_p95,omitempty"`
	SandboxRolloutPhase                    string   `json:"sandbox_rollout_phase,omitempty"`
	SandboxHealthBudgetStatus              string   `json:"sandbox_health_budget_status,omitempty"`
	SandboxCapacityAction                  string   `json:"sandbox_capacity_action,omitempty"`
	SandboxFreezeState                     bool     `json:"sandbox_freeze_state,omitempty"`
	SandboxFreezeReasonCode                string   `json:"sandbox_freeze_reason_code,omitempty"`
	MemoryMode                             string   `json:"memory_mode,omitempty"`
	MemoryProvider                         string   `json:"memory_provider,omitempty"`
	MemoryProfile                          string   `json:"memory_profile,omitempty"`
	MemoryContractVersion                  string   `json:"memory_contract_version,omitempty"`
	MemoryQueryTotal                       int      `json:"memory_query_total,omitempty"`
	MemoryUpsertTotal                      int      `json:"memory_upsert_total,omitempty"`
	MemoryDeleteTotal                      int      `json:"memory_delete_total,omitempty"`
	MemoryErrorTotal                       int      `json:"memory_error_total,omitempty"`
	MemoryFallbackTotal                    int      `json:"memory_fallback_total,omitempty"`
	MemoryFallbackReasonCode               string   `json:"memory_fallback_reason_code,omitempty"`
	MemoryReasonCode                       string   `json:"memory_reason_code,omitempty"`
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
		version != ArbitrationFixtureVersionA50V1 &&
		version != ArbitrationFixtureVersionA51V1 &&
		version != ArbitrationFixtureVersionA52V1 &&
		version != ArbitrationFixtureVersionMemoryV1 {
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
		SandboxMode:                            strings.ToLower(strings.TrimSpace(in.SandboxMode)),
		SandboxBackend:                         strings.ToLower(strings.TrimSpace(in.SandboxBackend)),
		SandboxProfile:                         strings.ToLower(strings.TrimSpace(in.SandboxProfile)),
		SandboxSessionMode:                     strings.ToLower(strings.TrimSpace(in.SandboxSessionMode)),
		SandboxDecision:                        strings.ToLower(strings.TrimSpace(in.SandboxDecision)),
		SandboxReasonCode:                      strings.ToLower(strings.TrimSpace(in.SandboxReasonCode)),
		SandboxFallbackUsed:                    in.SandboxFallbackUsed,
		SandboxFallbackReason:                  strings.ToLower(strings.TrimSpace(in.SandboxFallbackReason)),
		SandboxTimeoutTotal:                    in.SandboxTimeoutTotal,
		SandboxLaunchFailedTotal:               in.SandboxLaunchFailedTotal,
		SandboxCapabilityMismatchTotal:         in.SandboxCapabilityMismatchTotal,
		SandboxQueueWaitMsP95:                  in.SandboxQueueWaitMsP95,
		SandboxExecLatencyMsP95:                in.SandboxExecLatencyMsP95,
		SandboxExitCodeLast:                    in.SandboxExitCodeLast,
		SandboxOOMTotal:                        in.SandboxOOMTotal,
		SandboxResourceCPUMsTotal:              in.SandboxResourceCPUMsTotal,
		SandboxResourceMemoryPeakBytesP95:      in.SandboxResourceMemoryPeakBytesP95,
		SandboxRolloutPhase:                    strings.ToLower(strings.TrimSpace(in.SandboxRolloutPhase)),
		SandboxHealthBudgetStatus:              strings.ToLower(strings.TrimSpace(in.SandboxHealthBudgetStatus)),
		SandboxCapacityAction:                  strings.ToLower(strings.TrimSpace(in.SandboxCapacityAction)),
		SandboxFreezeState:                     in.SandboxFreezeState,
		SandboxFreezeReasonCode:                strings.ToLower(strings.TrimSpace(in.SandboxFreezeReasonCode)),
		MemoryMode:                             strings.ToLower(strings.TrimSpace(in.MemoryMode)),
		MemoryProvider:                         strings.ToLower(strings.TrimSpace(in.MemoryProvider)),
		MemoryProfile:                          strings.ToLower(strings.TrimSpace(in.MemoryProfile)),
		MemoryContractVersion:                  strings.ToLower(strings.TrimSpace(in.MemoryContractVersion)),
		MemoryQueryTotal:                       in.MemoryQueryTotal,
		MemoryUpsertTotal:                      in.MemoryUpsertTotal,
		MemoryDeleteTotal:                      in.MemoryDeleteTotal,
		MemoryErrorTotal:                       in.MemoryErrorTotal,
		MemoryFallbackTotal:                    in.MemoryFallbackTotal,
		MemoryFallbackReasonCode:               strings.ToLower(strings.TrimSpace(in.MemoryFallbackReasonCode)),
		MemoryReasonCode:                       strings.ToLower(strings.TrimSpace(in.MemoryReasonCode)),
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
	if out.SandboxTimeoutTotal < 0 {
		out.SandboxTimeoutTotal = 0
	}
	if out.SandboxLaunchFailedTotal < 0 {
		out.SandboxLaunchFailedTotal = 0
	}
	if out.SandboxCapabilityMismatchTotal < 0 {
		out.SandboxCapabilityMismatchTotal = 0
	}
	if out.SandboxQueueWaitMsP95 < 0 {
		out.SandboxQueueWaitMsP95 = 0
	}
	if out.SandboxExecLatencyMsP95 < 0 {
		out.SandboxExecLatencyMsP95 = 0
	}
	if out.SandboxOOMTotal < 0 {
		out.SandboxOOMTotal = 0
	}
	if out.SandboxResourceCPUMsTotal < 0 {
		out.SandboxResourceCPUMsTotal = 0
	}
	if out.SandboxResourceMemoryPeakBytesP95 < 0 {
		out.SandboxResourceMemoryPeakBytesP95 = 0
	}
	if out.MemoryQueryTotal < 0 {
		out.MemoryQueryTotal = 0
	}
	if out.MemoryUpsertTotal < 0 {
		out.MemoryUpsertTotal = 0
	}
	if out.MemoryDeleteTotal < 0 {
		out.MemoryDeleteTotal = 0
	}
	if out.MemoryErrorTotal < 0 {
		out.MemoryErrorTotal = 0
	}
	if out.MemoryFallbackTotal < 0 {
		out.MemoryFallbackTotal = 0
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
	for i := range in.SandboxRequiredCapabilities {
		item := strings.ToLower(strings.TrimSpace(in.SandboxRequiredCapabilities[i]))
		if item == "" {
			continue
		}
		out.SandboxRequiredCapabilities = append(out.SandboxRequiredCapabilities, item)
	}
	if len(out.SandboxRequiredCapabilities) == 0 {
		out.SandboxRequiredCapabilities = nil
	}
	return out
}

func validateArbitrationObservation(version, caseName, lane string, obs ArbitrationObservation) error {
	if version == ArbitrationFixtureVersionMemoryV1 {
		return validateMemoryArbitrationObservation(caseName, lane, obs)
	}
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
	if version != ArbitrationFixtureVersionA50V1 && version != ArbitrationFixtureVersionA51V1 && version != ArbitrationFixtureVersionA52V1 {
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
	if version == ArbitrationFixtureVersionA51V1 || version == ArbitrationFixtureVersionA52V1 {
		if err := validateSandboxArbitrationObservation(caseName, lane, obs); err != nil {
			return err
		}
	}
	if version == ArbitrationFixtureVersionA52V1 {
		if err := validateSandboxRolloutArbitrationObservation(caseName, lane, obs); err != nil {
			return err
		}
	}
	return nil
}

func assertArbitrationEquivalent(version, caseName string, expected, actual ArbitrationObservation, lane string) error {
	if arbitrationObservationsEqual(version, expected, actual) {
		return nil
	}
	if version == ArbitrationFixtureVersionMemoryV1 {
		return assertMemoryArbitrationEquivalent(caseName, lane, expected, actual)
	}
	if version == ArbitrationFixtureVersionA50V1 || version == ArbitrationFixtureVersionA51V1 || version == ArbitrationFixtureVersionA52V1 {
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
	if version == ArbitrationFixtureVersionA51V1 || version == ArbitrationFixtureVersionA52V1 {
		if err := assertSandboxArbitrationEquivalent(caseName, lane, expected, actual); err != nil {
			return err
		}
	}
	if version == ArbitrationFixtureVersionA52V1 {
		if err := assertSandboxRolloutArbitrationEquivalent(caseName, lane, expected, actual); err != nil {
			return err
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
	if version == ArbitrationFixtureVersionA50V1 || version == ArbitrationFixtureVersionA51V1 || version == ArbitrationFixtureVersionA52V1 {
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
	if (version == ArbitrationFixtureVersionA49V1 || version == ArbitrationFixtureVersionA50V1 || version == ArbitrationFixtureVersionA51V1 || version == ArbitrationFixtureVersionA52V1) &&
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

func validateSandboxArbitrationObservation(caseName, lane string, obs ArbitrationObservation) error {
	switch strings.TrimSpace(obs.SandboxMode) {
	case runtimeconfig.SecuritySandboxModeObserve, runtimeconfig.SecuritySandboxModeEnforce:
	default:
		return &ValidationError{
			Code:    ReasonCodeSandboxPolicyDrift,
			Message: fmt.Sprintf("case %q %s sandbox_mode must be observe|enforce", caseName, lane),
		}
	}
	switch strings.TrimSpace(obs.SandboxDecision) {
	case runtimeconfig.SecuritySandboxActionHost, runtimeconfig.SecuritySandboxActionSandbox, runtimeconfig.SecuritySandboxActionDeny:
	default:
		return &ValidationError{
			Code:    ReasonCodeSandboxPolicyDrift,
			Message: fmt.Sprintf("case %q %s sandbox_decision must be host|sandbox|deny", caseName, lane),
		}
	}
	if !strings.HasPrefix(strings.TrimSpace(obs.SandboxReasonCode), "sandbox.") {
		return &ValidationError{
			Code:    ReasonCodeSandboxPolicyDrift,
			Message: fmt.Sprintf("case %q %s sandbox_reason_code must be sandbox.* canonical code", caseName, lane),
		}
	}
	switch strings.TrimSpace(obs.SandboxSessionMode) {
	case runtimeconfig.SecuritySandboxSessionModePerCall, runtimeconfig.SecuritySandboxSessionModePerSession:
	default:
		return &ValidationError{
			Code:    ReasonCodeSandboxSessionLifecycleDrift,
			Message: fmt.Sprintf("case %q %s sandbox_session_mode must be per_call|per_session", caseName, lane),
		}
	}
	if obs.SandboxFallbackUsed && strings.TrimSpace(obs.SandboxFallbackReason) == "" {
		return &ValidationError{
			Code:    ReasonCodeSandboxFallbackDrift,
			Message: fmt.Sprintf("case %q %s sandbox_fallback_reason is required when fallback_used=true", caseName, lane),
		}
	}
	if !obs.SandboxFallbackUsed && strings.TrimSpace(obs.SandboxFallbackReason) != "" {
		return &ValidationError{
			Code:    ReasonCodeSandboxFallbackDrift,
			Message: fmt.Sprintf("case %q %s sandbox_fallback_reason must be empty when fallback_used=false", caseName, lane),
		}
	}
	return nil
}

func validateSandboxRolloutArbitrationObservation(caseName, lane string, obs ArbitrationObservation) error {
	switch strings.TrimSpace(obs.SandboxRolloutPhase) {
	case runtimeconfig.SecuritySandboxRolloutPhaseObserve,
		runtimeconfig.SecuritySandboxRolloutPhaseCanary,
		runtimeconfig.SecuritySandboxRolloutPhaseBaseline,
		runtimeconfig.SecuritySandboxRolloutPhaseFull,
		runtimeconfig.SecuritySandboxRolloutPhaseFrozen:
	default:
		return &ValidationError{
			Code:    ReasonCodeSandboxRolloutPhaseDrift,
			Message: fmt.Sprintf("case %q %s sandbox_rollout_phase must be observe|canary|baseline|full|frozen", caseName, lane),
		}
	}
	switch strings.TrimSpace(obs.SandboxHealthBudgetStatus) {
	case runtimeconfig.SandboxHealthBudgetWithinBudget,
		runtimeconfig.SandboxHealthBudgetNearBudget,
		runtimeconfig.SandboxHealthBudgetBreached:
	default:
		return &ValidationError{
			Code:    ReasonCodeSandboxHealthBudgetDrift,
			Message: fmt.Sprintf("case %q %s sandbox_health_budget_status must be within_budget|near_budget|breached", caseName, lane),
		}
	}
	switch strings.TrimSpace(obs.SandboxCapacityAction) {
	case runtimeconfig.SandboxCapacityActionAllow,
		runtimeconfig.SandboxCapacityActionThrottle,
		runtimeconfig.SandboxCapacityActionDeny:
	default:
		return &ValidationError{
			Code:    ReasonCodeSandboxCapacityActionDrift,
			Message: fmt.Sprintf("case %q %s sandbox_capacity_action must be allow|throttle|deny", caseName, lane),
		}
	}
	if obs.SandboxFreezeState && strings.TrimSpace(obs.SandboxFreezeReasonCode) == "" {
		return &ValidationError{
			Code:    ReasonCodeSandboxFreezeStateDrift,
			Message: fmt.Sprintf("case %q %s sandbox_freeze_reason_code is required when sandbox_freeze_state=true", caseName, lane),
		}
	}
	if !obs.SandboxFreezeState && strings.TrimSpace(obs.SandboxFreezeReasonCode) != "" {
		return &ValidationError{
			Code:    ReasonCodeSandboxFreezeStateDrift,
			Message: fmt.Sprintf("case %q %s sandbox_freeze_reason_code must be empty when sandbox_freeze_state=false", caseName, lane),
		}
	}
	return nil
}

func assertSandboxArbitrationEquivalent(caseName, lane string, expected, actual ArbitrationObservation) error {
	if expected.SandboxMode != actual.SandboxMode ||
		expected.SandboxBackend != actual.SandboxBackend ||
		expected.SandboxProfile != actual.SandboxProfile ||
		expected.SandboxDecision != actual.SandboxDecision ||
		expected.SandboxReasonCode != actual.SandboxReasonCode {
		return &ValidationError{
			Code:    ReasonCodeSandboxPolicyDrift,
			Message: fmt.Sprintf("case %q %s sandbox policy drift expected=%#v actual=%#v", caseName, lane, expected, actual),
		}
	}
	if expected.SandboxFallbackUsed != actual.SandboxFallbackUsed ||
		expected.SandboxFallbackReason != actual.SandboxFallbackReason ||
		expected.SandboxLaunchFailedTotal != actual.SandboxLaunchFailedTotal {
		return &ValidationError{
			Code:    ReasonCodeSandboxFallbackDrift,
			Message: fmt.Sprintf("case %q %s sandbox fallback drift expected=%#v actual=%#v", caseName, lane, expected, actual),
		}
	}
	if expected.SandboxTimeoutTotal != actual.SandboxTimeoutTotal {
		return &ValidationError{
			Code:    ReasonCodeSandboxTimeoutDrift,
			Message: fmt.Sprintf("case %q %s sandbox timeout drift expected=%d actual=%d", caseName, lane, expected.SandboxTimeoutTotal, actual.SandboxTimeoutTotal),
		}
	}
	if !equalStringSlice(expected.SandboxRequiredCapabilities, actual.SandboxRequiredCapabilities) ||
		expected.SandboxCapabilityMismatchTotal != actual.SandboxCapabilityMismatchTotal {
		return &ValidationError{
			Code:    ReasonCodeSandboxCapabilityDrift,
			Message: fmt.Sprintf("case %q %s sandbox capability drift expected=%#v actual=%#v", caseName, lane, expected, actual),
		}
	}
	if expected.SandboxSessionMode != actual.SandboxSessionMode ||
		expected.SandboxQueueWaitMsP95 != actual.SandboxQueueWaitMsP95 ||
		expected.SandboxExecLatencyMsP95 != actual.SandboxExecLatencyMsP95 {
		return &ValidationError{
			Code:    ReasonCodeSandboxSessionLifecycleDrift,
			Message: fmt.Sprintf("case %q %s sandbox session lifecycle drift expected=%#v actual=%#v", caseName, lane, expected, actual),
		}
	}
	if expected.SandboxExitCodeLast != actual.SandboxExitCodeLast ||
		expected.SandboxOOMTotal != actual.SandboxOOMTotal ||
		expected.SandboxResourceCPUMsTotal != actual.SandboxResourceCPUMsTotal ||
		expected.SandboxResourceMemoryPeakBytesP95 != actual.SandboxResourceMemoryPeakBytesP95 {
		return &ValidationError{
			Code:    ReasonCodeSandboxResourcePolicyDrift,
			Message: fmt.Sprintf("case %q %s sandbox resource policy drift expected=%#v actual=%#v", caseName, lane, expected, actual),
		}
	}
	return nil
}

func assertSandboxRolloutArbitrationEquivalent(caseName, lane string, expected, actual ArbitrationObservation) error {
	if expected.SandboxRolloutPhase != actual.SandboxRolloutPhase {
		return &ValidationError{
			Code:    ReasonCodeSandboxRolloutPhaseDrift,
			Message: fmt.Sprintf("case %q %s sandbox rollout phase drift expected=%q actual=%q", caseName, lane, expected.SandboxRolloutPhase, actual.SandboxRolloutPhase),
		}
	}
	if expected.SandboxHealthBudgetStatus != actual.SandboxHealthBudgetStatus {
		return &ValidationError{
			Code:    ReasonCodeSandboxHealthBudgetDrift,
			Message: fmt.Sprintf("case %q %s sandbox health budget drift expected=%q actual=%q", caseName, lane, expected.SandboxHealthBudgetStatus, actual.SandboxHealthBudgetStatus),
		}
	}
	if expected.SandboxCapacityAction != actual.SandboxCapacityAction {
		return &ValidationError{
			Code:    ReasonCodeSandboxCapacityActionDrift,
			Message: fmt.Sprintf("case %q %s sandbox capacity action drift expected=%q actual=%q", caseName, lane, expected.SandboxCapacityAction, actual.SandboxCapacityAction),
		}
	}
	if expected.SandboxFreezeState != actual.SandboxFreezeState ||
		expected.SandboxFreezeReasonCode != actual.SandboxFreezeReasonCode {
		return &ValidationError{
			Code:    ReasonCodeSandboxFreezeStateDrift,
			Message: fmt.Sprintf("case %q %s sandbox freeze state drift expected=%#v actual=%#v", caseName, lane, expected, actual),
		}
	}
	return nil
}

func arbitrationObservationsEqual(version string, left, right ArbitrationObservation) bool {
	if version == ArbitrationFixtureVersionMemoryV1 {
		return left.MemoryMode == right.MemoryMode &&
			left.MemoryProvider == right.MemoryProvider &&
			left.MemoryProfile == right.MemoryProfile &&
			left.MemoryContractVersion == right.MemoryContractVersion &&
			left.MemoryQueryTotal == right.MemoryQueryTotal &&
			left.MemoryUpsertTotal == right.MemoryUpsertTotal &&
			left.MemoryDeleteTotal == right.MemoryDeleteTotal &&
			left.MemoryErrorTotal == right.MemoryErrorTotal &&
			left.MemoryFallbackTotal == right.MemoryFallbackTotal &&
			left.MemoryFallbackReasonCode == right.MemoryFallbackReasonCode &&
			left.MemoryReasonCode == right.MemoryReasonCode
	}
	if left.RuntimePrimaryDomain != right.RuntimePrimaryDomain ||
		left.RuntimePrimaryCode != right.RuntimePrimaryCode ||
		left.RuntimePrimarySource != right.RuntimePrimarySource ||
		left.RuntimePrimaryConflictTotal != right.RuntimePrimaryConflictTotal {
		return false
	}
	if version == ArbitrationFixtureVersionA48V1 {
		return true
	}
	if left.RuntimeSecondaryReasonCount != right.RuntimeSecondaryReasonCount ||
		left.RuntimeArbitrationRuleVersion != right.RuntimeArbitrationRuleVersion ||
		left.RuntimeRemediationHintCode != right.RuntimeRemediationHintCode ||
		left.RuntimeRemediationHintDomain != right.RuntimeRemediationHintDomain {
		return false
	}
	if !equalStringSlice(left.RuntimeSecondaryReasonCodes, right.RuntimeSecondaryReasonCodes) {
		return false
	}
	if version == ArbitrationFixtureVersionA49V1 {
		return true
	}
	if left.RuntimeArbitrationRuleRequestedVersion != right.RuntimeArbitrationRuleRequestedVersion ||
		left.RuntimeArbitrationRuleEffectiveVersion != right.RuntimeArbitrationRuleEffectiveVersion ||
		left.RuntimeArbitrationRuleVersionSource != right.RuntimeArbitrationRuleVersionSource ||
		left.RuntimeArbitrationRulePolicyAction != right.RuntimeArbitrationRulePolicyAction ||
		left.RuntimeArbitrationRuleUnsupportedTotal != right.RuntimeArbitrationRuleUnsupportedTotal ||
		left.RuntimeArbitrationRuleMismatchTotal != right.RuntimeArbitrationRuleMismatchTotal {
		return false
	}
	if version == ArbitrationFixtureVersionA50V1 {
		return true
	}
	if left.SandboxMode != right.SandboxMode ||
		left.SandboxBackend != right.SandboxBackend ||
		left.SandboxProfile != right.SandboxProfile ||
		left.SandboxSessionMode != right.SandboxSessionMode ||
		!equalStringSlice(left.SandboxRequiredCapabilities, right.SandboxRequiredCapabilities) ||
		left.SandboxDecision != right.SandboxDecision ||
		left.SandboxReasonCode != right.SandboxReasonCode ||
		left.SandboxFallbackUsed != right.SandboxFallbackUsed ||
		left.SandboxFallbackReason != right.SandboxFallbackReason ||
		left.SandboxTimeoutTotal != right.SandboxTimeoutTotal ||
		left.SandboxLaunchFailedTotal != right.SandboxLaunchFailedTotal ||
		left.SandboxCapabilityMismatchTotal != right.SandboxCapabilityMismatchTotal ||
		left.SandboxQueueWaitMsP95 != right.SandboxQueueWaitMsP95 ||
		left.SandboxExecLatencyMsP95 != right.SandboxExecLatencyMsP95 ||
		left.SandboxExitCodeLast != right.SandboxExitCodeLast ||
		left.SandboxOOMTotal != right.SandboxOOMTotal ||
		left.SandboxResourceCPUMsTotal != right.SandboxResourceCPUMsTotal ||
		left.SandboxResourceMemoryPeakBytesP95 != right.SandboxResourceMemoryPeakBytesP95 {
		return false
	}
	if version == ArbitrationFixtureVersionA51V1 {
		return true
	}
	if left.SandboxRolloutPhase != right.SandboxRolloutPhase ||
		left.SandboxHealthBudgetStatus != right.SandboxHealthBudgetStatus ||
		left.SandboxCapacityAction != right.SandboxCapacityAction ||
		left.SandboxFreezeState != right.SandboxFreezeState ||
		left.SandboxFreezeReasonCode != right.SandboxFreezeReasonCode {
		return false
	}
	return true
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

func validateMemoryArbitrationObservation(caseName, lane string, obs ArbitrationObservation) error {
	switch strings.TrimSpace(obs.MemoryMode) {
	case runtimeconfig.RuntimeMemoryModeBuiltinFilesystem, runtimeconfig.RuntimeMemoryModeExternalSPI:
	default:
		return &ValidationError{
			Code:    ReasonCodeMemoryModeDrift,
			Message: fmt.Sprintf("case %q %s memory_mode must be external_spi|builtin_filesystem", caseName, lane),
		}
	}
	if strings.TrimSpace(obs.MemoryProvider) == "" {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s memory_provider is required", caseName, lane),
		}
	}
	if strings.TrimSpace(obs.MemoryProfile) == "" {
		return &ValidationError{
			Code:    ReasonCodeSchemaMismatch,
			Message: fmt.Sprintf("case %q %s memory_profile is required", caseName, lane),
		}
	}
	if strings.TrimSpace(obs.MemoryContractVersion) != runtimeconfig.RuntimeMemoryContractVersionV1 {
		return &ValidationError{
			Code:    ReasonCodeMemoryContractVersionDrift,
			Message: fmt.Sprintf("case %q %s memory_contract_version must be %q", caseName, lane, runtimeconfig.RuntimeMemoryContractVersionV1),
		}
	}
	if obs.MemoryMode == runtimeconfig.RuntimeMemoryModeBuiltinFilesystem &&
		strings.TrimSpace(obs.MemoryProvider) != runtimeconfig.RuntimeMemoryModeBuiltinFilesystem {
		return &ValidationError{
			Code:    ReasonCodeMemoryProfileDrift,
			Message: fmt.Sprintf("case %q %s builtin_filesystem mode must use memory_provider=builtin_filesystem", caseName, lane),
		}
	}
	if obs.MemoryMode == runtimeconfig.RuntimeMemoryModeExternalSPI &&
		strings.TrimSpace(obs.MemoryProvider) == runtimeconfig.RuntimeMemoryModeBuiltinFilesystem {
		return &ValidationError{
			Code:    ReasonCodeMemoryProfileDrift,
			Message: fmt.Sprintf("case %q %s external_spi mode must not use memory_provider=builtin_filesystem", caseName, lane),
		}
	}
	if obs.MemoryFallbackTotal > 0 && strings.TrimSpace(obs.MemoryFallbackReasonCode) == "" {
		return &ValidationError{
			Code:    ReasonCodeMemoryFallbackDrift,
			Message: fmt.Sprintf("case %q %s memory_fallback_reason_code is required when memory_fallback_total>0", caseName, lane),
		}
	}
	if obs.MemoryFallbackTotal == 0 && strings.TrimSpace(obs.MemoryFallbackReasonCode) != "" {
		return &ValidationError{
			Code:    ReasonCodeMemoryFallbackDrift,
			Message: fmt.Sprintf("case %q %s memory_fallback_reason_code must be empty when memory_fallback_total=0", caseName, lane),
		}
	}
	if strings.TrimSpace(obs.MemoryFallbackReasonCode) != "" &&
		!strings.HasPrefix(strings.TrimSpace(obs.MemoryFallbackReasonCode), "memory.fallback.") {
		return &ValidationError{
			Code:    ReasonCodeMemoryFallbackDrift,
			Message: fmt.Sprintf("case %q %s memory_fallback_reason_code must be memory.fallback.* canonical code", caseName, lane),
		}
	}
	if obs.MemoryErrorTotal > 0 && strings.TrimSpace(obs.MemoryReasonCode) == "" {
		return &ValidationError{
			Code:    ReasonCodeMemoryErrorTaxonomyDrift,
			Message: fmt.Sprintf("case %q %s memory_reason_code is required when memory_error_total>0", caseName, lane),
		}
	}
	if strings.TrimSpace(obs.MemoryReasonCode) != "" &&
		!strings.HasPrefix(strings.TrimSpace(obs.MemoryReasonCode), "memory.") {
		return &ValidationError{
			Code:    ReasonCodeMemoryErrorTaxonomyDrift,
			Message: fmt.Sprintf("case %q %s memory_reason_code must be memory.* canonical code", caseName, lane),
		}
	}
	return nil
}

func assertMemoryArbitrationEquivalent(caseName, lane string, expected, actual ArbitrationObservation) error {
	if expected.MemoryMode != actual.MemoryMode {
		return &ValidationError{
			Code:    ReasonCodeMemoryModeDrift,
			Message: fmt.Sprintf("case %q %s memory mode drift expected=%q actual=%q", caseName, lane, expected.MemoryMode, actual.MemoryMode),
		}
	}
	if expected.MemoryProvider != actual.MemoryProvider || expected.MemoryProfile != actual.MemoryProfile {
		return &ValidationError{
			Code:    ReasonCodeMemoryProfileDrift,
			Message: fmt.Sprintf("case %q %s memory profile drift expected_provider/profile=%q/%q actual_provider/profile=%q/%q", caseName, lane, expected.MemoryProvider, expected.MemoryProfile, actual.MemoryProvider, actual.MemoryProfile),
		}
	}
	if expected.MemoryContractVersion != actual.MemoryContractVersion {
		return &ValidationError{
			Code:    ReasonCodeMemoryContractVersionDrift,
			Message: fmt.Sprintf("case %q %s memory contract version drift expected=%q actual=%q", caseName, lane, expected.MemoryContractVersion, actual.MemoryContractVersion),
		}
	}
	if expected.MemoryFallbackTotal != actual.MemoryFallbackTotal ||
		expected.MemoryFallbackReasonCode != actual.MemoryFallbackReasonCode {
		return &ValidationError{
			Code:    ReasonCodeMemoryFallbackDrift,
			Message: fmt.Sprintf("case %q %s memory fallback drift expected=%#v actual=%#v", caseName, lane, expected, actual),
		}
	}
	if expected.MemoryReasonCode != actual.MemoryReasonCode {
		return &ValidationError{
			Code:    ReasonCodeMemoryErrorTaxonomyDrift,
			Message: fmt.Sprintf("case %q %s memory error taxonomy drift expected_reason=%q actual_reason=%q", caseName, lane, expected.MemoryReasonCode, actual.MemoryReasonCode),
		}
	}
	if expected.MemoryQueryTotal != actual.MemoryQueryTotal ||
		expected.MemoryUpsertTotal != actual.MemoryUpsertTotal ||
		expected.MemoryDeleteTotal != actual.MemoryDeleteTotal ||
		expected.MemoryErrorTotal != actual.MemoryErrorTotal {
		return &ValidationError{
			Code:    ReasonCodeMemoryOperationAggregateDrift,
			Message: fmt.Sprintf("case %q %s memory operation aggregate drift expected=%#v actual=%#v", caseName, lane, expected, actual),
		}
	}
	return nil
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
		runtimeconfig.ReadinessCodeRuntimeManagerUnavailable,
		runtimeconfig.ReadinessCodeSandboxRolloutPhaseInvalid,
		runtimeconfig.ReadinessCodeSandboxRolloutFrozen:
		return 2
	case runtimeconfig.ReadinessCodeAdapterRequiredUnavailable, runtimeconfig.ReadinessCodeAdapterRequiredCircuitOpen:
		return 3
	case runtimeconfig.ReadinessCodeSchedulerFallback,
		runtimeconfig.ReadinessCodeMailboxFallback,
		runtimeconfig.ReadinessCodeRecoveryFallback,
		runtimeconfig.ReadinessCodeAdapterOptionalUnavailable,
		runtimeconfig.ReadinessCodeAdapterOptionalCircuitOpen,
		runtimeconfig.ReadinessCodeAdapterDegraded,
		runtimeconfig.ReadinessCodeAdapterHalfOpenDegraded,
		runtimeconfig.ReadinessCodeSandboxRolloutHealthBreached,
		runtimeconfig.ReadinessCodeSandboxRolloutCapacityBlocked:
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
