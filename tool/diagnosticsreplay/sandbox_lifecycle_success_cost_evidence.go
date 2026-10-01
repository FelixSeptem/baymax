package diagnosticsreplay

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
)

const SandboxLifecycleSuccessCostEvidenceFixtureV1 = "sandbox_lifecycle_success_cost_evidence.v1"

const (
	SandboxLifecycleSuccessCostVerdictWithinBaseline         = "within-baseline"
	SandboxLifecycleSuccessCostVerdictGapConfirmed           = "lifecycle-gap-confirmed"
	SandboxLifecycleSuccessCostVerdictInsufficient           = "insufficient-evidence"
	SandboxLifecycleKindColdLaunch                           = "cold_launch"
	SandboxLifecycleKindReuse                                = "reuse"
	SandboxLifecycleKindRecovery                             = "recovery"
	SandboxLifecycleSuccessCostReasonSchemaDrift             = "sandbox_lifecycle_schema_drift"
	SandboxLifecycleSuccessCostReasonPrivacyOrBoundViolation = "sandbox_lifecycle_privacy_or_bound_violation"
	SandboxLifecycleSuccessCostReasonContinuityDrift         = "sandbox_lifecycle_continuity_drift"
	SandboxLifecycleSuccessCostReasonBaselineDrift           = "sandbox_lifecycle_baseline_drift"
	SandboxLifecycleSuccessCostReasonTerminalOutcomeDrift    = "sandbox_lifecycle_terminal_outcome_drift"
	SandboxLifecycleSuccessCostReasonParityDrift             = "sandbox_lifecycle_run_stream_parity_drift"
	SandboxLifecycleSuccessCostReasonReplayNotIdempotent     = "sandbox_lifecycle_replay_not_idempotent"
	SandboxLifecycleSuccessCostReasonPhaseDrift              = "sandbox_lifecycle_phase_drift"
	SandboxLifecycleSuccessCostReasonSuccessCostBucketDrift  = "sandbox_success_cost_bucket_drift"
	SandboxLifecycleSuccessCostReasonLifecycleBaselineDrift  = "sandbox_lifecycle_baseline_drift"
	SandboxLifecycleSuccessCostReasonRunStreamParityDrift    = "sandbox_lifecycle_run_stream_parity_drift"
	SandboxLifecycleSuccessCostReasonPrivacyDrift            = "sandbox_lifecycle_privacy_drift"
)

const (
	sandboxLifecycleMaxFixtureBytes = 1 << 20
	sandboxLifecycleMaxCases        = 64
	sandboxLifecycleMaxString       = 256
	sandboxLifecycleMaxPhases       = 16
)

type SandboxLifecycleSuccessCostEvidenceFixture struct {
	Version string                                           `json:"version"`
	Cases   []SandboxLifecycleSuccessCostEvidenceFixtureCase `json:"cases"`
}

type SandboxLifecycleSuccessCostEvidenceFixtureCase struct {
	CaseID   string                                      `json:"case_id"`
	Input    SandboxLifecycleSuccessCostEvidenceInput    `json:"input"`
	Expected SandboxLifecycleSuccessCostEvidenceExpected `json:"expected"`
}

type SandboxLifecycleSuccessCostEvidenceInput struct {
	InvocationRef         string                              `json:"invocation_ref"`
	SessionRef            string                              `json:"session_ref"`
	Backend               string                              `json:"backend"`
	Profile               string                              `json:"profile"`
	SessionMode           string                              `json:"session_mode"`
	LifecycleKind         string                              `json:"lifecycle_kind,omitempty"`
	Phases                []string                            `json:"phases"`
	DurationBucket        string                              `json:"duration_bucket"`
	ResourceBucket        string                              `json:"resource_bucket"`
	RetryOrdinal          int                                 `json:"retry_ordinal"`
	ReasonCode            string                              `json:"reason_code,omitempty"`
	TerminalOutcome       string                              `json:"terminal_outcome"`
	SuccessClassification string                              `json:"success_classification"`
	Baseline              SandboxLifecycleSuccessCostBaseline `json:"baseline"`
	RunStreamParity       bool                                `json:"run_stream_parity"`
}

type SandboxLifecycleSuccessCostBaseline struct {
	DurationBucket string `json:"duration_bucket"`
	ResourceBucket string `json:"resource_bucket"`
}

type SandboxLifecycleSuccessCostEvidenceExpected struct {
	Verdict         string `json:"verdict,omitempty"`
	LifecycleKind   string `json:"lifecycle_kind,omitempty"`
	HasSuccessCost  *bool  `json:"has_success_cost,omitempty"`
	RunStreamParity *bool  `json:"run_stream_parity,omitempty"`
	CanonicalDigest string `json:"canonical_digest,omitempty"`
}

type SandboxLifecycleSuccessCostEvidenceReplayResult struct {
	Version string                                          `json:"version"`
	Cases   []SandboxLifecycleSuccessCostEvidenceReplayCase `json:"cases"`
}

type SandboxLifecycleSuccessCostEvidenceReplayCase struct {
	CaseID          string   `json:"case_id"`
	Verdict         string   `json:"verdict"`
	LifecycleKind   string   `json:"lifecycle_kind"`
	DurationBucket  string   `json:"duration_bucket,omitempty"`
	ResourceBucket  string   `json:"resource_bucket,omitempty"`
	HasSuccessCost  bool     `json:"has_success_cost"`
	Reasons         []string `json:"reasons,omitempty"`
	RunStreamParity bool     `json:"run_stream_parity"`
	Digest          string   `json:"digest"`
	ReplayDigest    string   `json:"replay_digest"`
	Idempotent      bool     `json:"idempotent"`
}

// ReplaySandboxLifecycleSuccessCostEvidenceJSON evaluates only supplied,
// bounded evidence. It has no platform, runtime, network, or persistence side effect.
func ReplaySandboxLifecycleSuccessCostEvidenceJSON(raw []byte) (SandboxLifecycleSuccessCostEvidenceReplayResult, error) {
	if len(raw) == 0 || len(raw) > sandboxLifecycleMaxFixtureBytes {
		return SandboxLifecycleSuccessCostEvidenceReplayResult{}, sandboxLifecycleError(SandboxLifecycleSuccessCostReasonPrivacyOrBoundViolation, "fixture size is outside the allowed bound")
	}
	if err := rejectSandboxLifecycleForbiddenFields(raw); err != nil {
		return SandboxLifecycleSuccessCostEvidenceReplayResult{}, err
	}
	var fixture SandboxLifecycleSuccessCostEvidenceFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		return SandboxLifecycleSuccessCostEvidenceReplayResult{}, sandboxLifecycleError(SandboxLifecycleSuccessCostReasonSchemaDrift, err.Error())
	}
	if fixture.Version != SandboxLifecycleSuccessCostEvidenceFixtureV1 {
		return SandboxLifecycleSuccessCostEvidenceReplayResult{}, sandboxLifecycleError(SandboxLifecycleSuccessCostReasonSchemaDrift, "unsupported evidence fixture version")
	}
	if len(fixture.Cases) == 0 || len(fixture.Cases) > sandboxLifecycleMaxCases {
		return SandboxLifecycleSuccessCostEvidenceReplayResult{}, sandboxLifecycleError(SandboxLifecycleSuccessCostReasonSchemaDrift, "fixture case count is outside the allowed bound")
	}

	result := SandboxLifecycleSuccessCostEvidenceReplayResult{Version: fixture.Version, Cases: make([]SandboxLifecycleSuccessCostEvidenceReplayCase, 0, len(fixture.Cases))}
	for _, item := range fixture.Cases {
		first, err := replaySandboxLifecycleSuccessCostEvidenceCase(item)
		if err != nil {
			return SandboxLifecycleSuccessCostEvidenceReplayResult{}, err
		}
		second, err := replaySandboxLifecycleSuccessCostEvidenceCase(item)
		if err != nil {
			return SandboxLifecycleSuccessCostEvidenceReplayResult{}, err
		}
		if first.Digest != second.Digest {
			return SandboxLifecycleSuccessCostEvidenceReplayResult{}, sandboxLifecycleError(SandboxLifecycleSuccessCostReasonReplayNotIdempotent, item.CaseID)
		}
		if err := compareSandboxLifecycleSuccessCostExpected(first, item.Expected); err != nil {
			return SandboxLifecycleSuccessCostEvidenceReplayResult{}, err
		}
		first.ReplayDigest = second.Digest
		first.Idempotent = true
		result.Cases = append(result.Cases, first)
	}
	return result, nil
}

func replaySandboxLifecycleSuccessCostEvidenceCase(item SandboxLifecycleSuccessCostEvidenceFixtureCase) (SandboxLifecycleSuccessCostEvidenceReplayCase, error) {
	caseID := strings.TrimSpace(item.CaseID)
	if caseID == "" || len(caseID) > sandboxLifecycleMaxString {
		return SandboxLifecycleSuccessCostEvidenceReplayCase{}, sandboxLifecycleError(SandboxLifecycleSuccessCostReasonSchemaDrift, "case_id is required and bounded")
	}
	input, err := normalizeSandboxLifecycleSuccessCostInput(item.Input)
	if err != nil {
		return SandboxLifecycleSuccessCostEvidenceReplayCase{}, err
	}

	lifecycleKind, continuityReasons := classifySandboxLifecycleContinuity(input)
	result := SandboxLifecycleSuccessCostEvidenceReplayCase{
		CaseID: caseID, LifecycleKind: lifecycleKind, RunStreamParity: input.RunStreamParity,
	}
	reasons := append([]string(nil), continuityReasons...)
	if !input.RunStreamParity {
		reasons = append(reasons, SandboxLifecycleSuccessCostReasonRunStreamParityDrift)
	}

	if len(reasons) > 0 {
		result.Verdict = SandboxLifecycleSuccessCostVerdictGapConfirmed
		result.Reasons = sortedSandboxLifecycleReasons(reasons)
		result.Digest = digestSandboxLifecycleSuccessCost(input, result)
		return result, nil
	}
	if input.TerminalOutcome != "success" || input.SuccessClassification != "successful" {
		result.Verdict = SandboxLifecycleSuccessCostVerdictInsufficient
		result.Reasons = []string{SandboxLifecycleSuccessCostReasonTerminalOutcomeDrift}
		result.Digest = digestSandboxLifecycleSuccessCost(input, result)
		return result, nil
	}

	result.DurationBucket, result.ResourceBucket, result.HasSuccessCost = input.DurationBucket, input.ResourceBucket, true
	if input.DurationBucket == input.Baseline.DurationBucket && input.ResourceBucket == input.Baseline.ResourceBucket {
		result.Verdict = SandboxLifecycleSuccessCostVerdictWithinBaseline
	} else {
		result.Verdict = SandboxLifecycleSuccessCostVerdictGapConfirmed
		result.Reasons = []string{SandboxLifecycleSuccessCostReasonSuccessCostBucketDrift, SandboxLifecycleSuccessCostReasonLifecycleBaselineDrift}
	}
	result.Digest = digestSandboxLifecycleSuccessCost(input, result)
	return result, nil
}

func normalizeSandboxLifecycleSuccessCostInput(input SandboxLifecycleSuccessCostEvidenceInput) (SandboxLifecycleSuccessCostEvidenceInput, error) {
	input.InvocationRef = strings.TrimSpace(input.InvocationRef)
	input.SessionRef = strings.TrimSpace(input.SessionRef)
	input.Backend = strings.TrimSpace(input.Backend)
	input.Profile = strings.TrimSpace(input.Profile)
	input.SessionMode = strings.ToLower(strings.TrimSpace(input.SessionMode))
	input.LifecycleKind = strings.ToLower(strings.TrimSpace(input.LifecycleKind))
	input.DurationBucket = strings.ToLower(strings.TrimSpace(input.DurationBucket))
	input.ResourceBucket = strings.ToLower(strings.TrimSpace(input.ResourceBucket))
	input.ReasonCode = strings.ToLower(strings.TrimSpace(input.ReasonCode))
	input.TerminalOutcome = strings.ToLower(strings.TrimSpace(input.TerminalOutcome))
	input.SuccessClassification = strings.ToLower(strings.TrimSpace(input.SuccessClassification))
	input.Baseline.DurationBucket = strings.ToLower(strings.TrimSpace(input.Baseline.DurationBucket))
	input.Baseline.ResourceBucket = strings.ToLower(strings.TrimSpace(input.Baseline.ResourceBucket))
	for index := range input.Phases {
		input.Phases[index] = strings.ToLower(strings.TrimSpace(input.Phases[index]))
	}

	for name, value := range map[string]string{
		"invocation_ref": input.InvocationRef, "session_ref": input.SessionRef, "backend": input.Backend, "profile": input.Profile,
		"duration_bucket": input.DurationBucket, "resource_bucket": input.ResourceBucket, "terminal_outcome": input.TerminalOutcome,
		"success_classification": input.SuccessClassification, "baseline.duration_bucket": input.Baseline.DurationBucket, "baseline.resource_bucket": input.Baseline.ResourceBucket,
	} {
		if value == "" || len(value) > sandboxLifecycleMaxString {
			return SandboxLifecycleSuccessCostEvidenceInput{}, sandboxLifecycleError(SandboxLifecycleSuccessCostReasonSchemaDrift, name+" is required and bounded")
		}
	}
	if len(input.ReasonCode) > sandboxLifecycleMaxString || len(input.LifecycleKind) > sandboxLifecycleMaxString {
		return SandboxLifecycleSuccessCostEvidenceInput{}, sandboxLifecycleError(SandboxLifecycleSuccessCostReasonPrivacyOrBoundViolation, "reason or lifecycle kind exceeds bound")
	}
	if input.SessionMode != "per_call" && input.SessionMode != "per_session" {
		return SandboxLifecycleSuccessCostEvidenceInput{}, sandboxLifecycleError(SandboxLifecycleSuccessCostReasonSchemaDrift, "session_mode must be per_call or per_session")
	}
	if input.LifecycleKind != "" && input.LifecycleKind != SandboxLifecycleKindColdLaunch && input.LifecycleKind != SandboxLifecycleKindReuse && input.LifecycleKind != SandboxLifecycleKindRecovery {
		return SandboxLifecycleSuccessCostEvidenceInput{}, sandboxLifecycleError(SandboxLifecycleSuccessCostReasonSchemaDrift, "unsupported lifecycle_kind")
	}
	if input.TerminalOutcome != "success" && input.TerminalOutcome != "failure" && input.TerminalOutcome != "unknown" {
		return SandboxLifecycleSuccessCostEvidenceInput{}, sandboxLifecycleError(SandboxLifecycleSuccessCostReasonSchemaDrift, "unsupported terminal_outcome")
	}
	if input.SuccessClassification != "successful" && input.SuccessClassification != "failed" && input.SuccessClassification != "unknown" {
		return SandboxLifecycleSuccessCostEvidenceInput{}, sandboxLifecycleError(SandboxLifecycleSuccessCostReasonSchemaDrift, "unsupported success_classification")
	}
	if input.RetryOrdinal < 0 || len(input.Phases) == 0 || len(input.Phases) > sandboxLifecycleMaxPhases {
		return SandboxLifecycleSuccessCostEvidenceInput{}, sandboxLifecycleError(SandboxLifecycleSuccessCostReasonSchemaDrift, "retry_ordinal or phases is outside the allowed bound")
	}
	for _, phase := range input.Phases {
		if phase != "acquire" && phase != "launch" && phase != "execute" && phase != "retry" && phase != "release" && phase != "recover" {
			return SandboxLifecycleSuccessCostEvidenceInput{}, sandboxLifecycleError(SandboxLifecycleSuccessCostReasonSchemaDrift, "unsupported lifecycle phase")
		}
	}
	return input, nil
}

func classifySandboxLifecycleContinuity(input SandboxLifecycleSuccessCostEvidenceInput) (string, []string) {
	kind := input.LifecycleKind
	if kind == "" {
		switch {
		case containsSandboxLifecyclePhase(input.Phases, "recover"):
			kind = SandboxLifecycleKindRecovery
		case input.SessionMode == "per_session" && !containsSandboxLifecyclePhase(input.Phases, "launch"):
			kind = SandboxLifecycleKindReuse
		default:
			kind = SandboxLifecycleKindColdLaunch
		}
	}

	reasons := make([]string, 0, 2)
	continuityReason := func() { reasons = append(reasons, SandboxLifecycleSuccessCostReasonContinuityDrift) }
	phaseReason := func() { reasons = append(reasons, SandboxLifecycleSuccessCostReasonPhaseDrift) }
	if kind == SandboxLifecycleKindReuse {
		if containsSandboxLifecyclePhase(input.Phases, "acquire") || containsSandboxLifecyclePhase(input.Phases, "launch") {
			continuityReason()
		}
	} else if len(input.Phases) == 0 || input.Phases[0] != "acquire" {
		phaseReason()
	}
	if kind == SandboxLifecycleKindColdLaunch && !containsSandboxLifecyclePhase(input.Phases, "launch") {
		continuityReason()
	}
	if !containsSandboxLifecyclePhase(input.Phases, "execute") || !containsSandboxLifecyclePhase(input.Phases, "release") {
		continuityReason()
	}

	phaseOrder := map[string]int{"acquire": 1, "launch": 2, "execute": 3, "retry": 4, "recover": 5, "release": 6}
	previousOrder := 0
	seen := make(map[string]int, len(input.Phases))
	for _, phase := range input.Phases {
		seen[phase]++
		if phase == "execute" && kind == SandboxLifecycleKindRecovery && seen[phase] == 2 {
			continue
		}
		if phaseOrder[phase] < previousOrder {
			phaseReason()
		}
		if seen[phase] > 1 {
			continuityReason()
		}
		if phase != "execute" || seen[phase] == 1 {
			previousOrder = phaseOrder[phase]
		}
	}
	if input.RetryOrdinal != seen["retry"] {
		continuityReason()
	}
	if seen["recover"] > 0 && (seen["retry"] == 0 || input.ReasonCode != "recoverable_failure") {
		continuityReason()
	}
	if kind == SandboxLifecycleKindRecovery && seen["recover"] != 1 {
		continuityReason()
	}
	return kind, sortedSandboxLifecycleReasons(reasons)
}

func compareSandboxLifecycleSuccessCostExpected(actual SandboxLifecycleSuccessCostEvidenceReplayCase, expected SandboxLifecycleSuccessCostEvidenceExpected) error {
	if expected.Verdict != "" && expected.Verdict != actual.Verdict {
		return sandboxLifecycleError(SandboxLifecycleSuccessCostReasonBaselineDrift, "expected verdict differs")
	}
	if expected.LifecycleKind != "" && expected.LifecycleKind != actual.LifecycleKind {
		return sandboxLifecycleError(SandboxLifecycleSuccessCostReasonContinuityDrift, "expected lifecycle kind differs")
	}
	if expected.HasSuccessCost != nil && *expected.HasSuccessCost != actual.HasSuccessCost {
		return sandboxLifecycleError(SandboxLifecycleSuccessCostReasonTerminalOutcomeDrift, "expected success cost differs")
	}
	if expected.RunStreamParity != nil && *expected.RunStreamParity != actual.RunStreamParity {
		return sandboxLifecycleError(SandboxLifecycleSuccessCostReasonParityDrift, "expected run/stream parity differs")
	}
	if expected.CanonicalDigest != "" && expected.CanonicalDigest != actual.Digest {
		return sandboxLifecycleError(SandboxLifecycleSuccessCostReasonBaselineDrift, "expected canonical digest differs")
	}
	return nil
}

func rejectSandboxLifecycleForbiddenFields(raw []byte) error {
	var document any
	if err := json.Unmarshal(raw, &document); err != nil {
		return sandboxLifecycleError(SandboxLifecycleSuccessCostReasonSchemaDrift, err.Error())
	}
	if containsSandboxLifecycleForbiddenField(document) {
		return sandboxLifecycleError(SandboxLifecycleSuccessCostReasonPrivacyDrift, "fixture includes prohibited evidence field")
	}
	return nil
}

func containsSandboxLifecycleForbiddenField(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, nested := range typed {
			normalized := strings.ToLower(strings.TrimSpace(key))
			switch normalized {
			case "command", "command_text", "arguments", "args", "environment", "env", "workdir", "mounts", "path", "paths", "endpoint", "credentials", "credential", "stdout", "stderr", "raw_violation", "raw_violation_payload", "raw_payload", "prompt", "reasoning", "body":
				return true
			}
			if containsSandboxLifecycleForbiddenField(nested) {
				return true
			}
		}
	case []any:
		for _, nested := range typed {
			if containsSandboxLifecycleForbiddenField(nested) {
				return true
			}
		}
	}
	return false
}

func digestSandboxLifecycleSuccessCost(input SandboxLifecycleSuccessCostEvidenceInput, output SandboxLifecycleSuccessCostEvidenceReplayCase) string {
	canonical := struct {
		Input  SandboxLifecycleSuccessCostEvidenceInput      `json:"input"`
		Output SandboxLifecycleSuccessCostEvidenceReplayCase `json:"output"`
	}{Input: input, Output: SandboxLifecycleSuccessCostEvidenceReplayCase{CaseID: output.CaseID, Verdict: output.Verdict, LifecycleKind: output.LifecycleKind, DurationBucket: output.DurationBucket, ResourceBucket: output.ResourceBucket, HasSuccessCost: output.HasSuccessCost, Reasons: output.Reasons, RunStreamParity: output.RunStreamParity}}
	raw, _ := json.Marshal(canonical)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func sandboxLifecycleError(code, message string) *ValidationError {
	return &ValidationError{Code: code, Message: message}
}

func sortedSandboxLifecycleReasons(reasons []string) []string {
	set := make(map[string]struct{}, len(reasons))
	for _, reason := range reasons {
		if reason = strings.TrimSpace(reason); reason != "" {
			set[reason] = struct{}{}
		}
	}
	result := make([]string, 0, len(set))
	for reason := range set {
		result = append(result, reason)
	}
	sort.Strings(result)
	return result
}

func containsSandboxLifecyclePhase(phases []string, want string) bool {
	for _, phase := range phases {
		if phase == want {
			return true
		}
	}
	return false
}
