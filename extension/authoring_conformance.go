package extension

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	AuthoringProfileV1     = "external_extension_authoring.v1"
	MaxAuthoringNameBytes  = 96
	MaxAuthoringFieldBytes = 96
	MaxAuthoringTextBytes  = 256
	MaxAuthoringCandidates = 64

	AuthoringStatusPassed     = "passed"
	AuthoringStatusFailed     = "failed"
	AuthoringPhaseSchema      = "schema"
	AuthoringPhaseManifest    = "manifest"
	AuthoringPhaseAdmission   = "admission"
	AuthoringPhaseLifecycle   = "lifecycle"
	AuthoringPhaseReload      = "reload"
	AuthoringPhaseComplete    = "complete"
	AuthoringProjectionRun    = "run"
	AuthoringProjectionStream = "stream"

	ReasonAuthoringMissingField       = "authoring.missing_field"
	ReasonAuthoringInvalidField       = "authoring.invalid_field"
	ReasonAuthoringUnsupportedProfile = "authoring.unsupported_profile"
	ReasonAuthoringFinalizeFailed     = "authoring.finalize_failed"
	ReasonAuthoringReloadFailed       = "authoring.reload_failed"
	ReasonAuthoringStaleGeneration    = "authoring.stale_generation"
	ReasonAuthoringPolicyDenied       = "authoring.policy_denied"
	ReasonAuthoringSandboxDenied      = "authoring.sandbox_denied"
	ReasonAuthoringAllowlistDenied    = "authoring.allowlist_denied"
	ReasonAuthoringEgressDenied       = "authoring.egress_denied"
	ReasonAuthoringReplayDrift        = "authoring.replay_drift"
)

type AuthoringCase struct {
	Profile    string               `json:"profile"`
	Case       string               `json:"case"`
	Candidates []Descriptor         `json:"candidates"`
	Admission  AuthoringAdmission   `json:"admission"`
	Lifecycle  AuthoringLifecycle   `json:"lifecycle"`
	Expected   AuthoringExpectation `json:"expected"`
	Boundaries AuthoringBoundaries  `json:"boundaries"`
}

type AuthoringAdmission struct {
	RuntimeVersion        string   `json:"runtime_version"`
	AvailableCapabilities []string `json:"available_capabilities"`
	BestEffort            bool     `json:"best_effort"`
}

type AuthoringLifecycle struct {
	Action        string        `json:"action"`
	FailurePolicy FailurePolicy `json:"failure_policy"`
}

type AuthoringExpectation struct {
	Status string `json:"status"`
	Phase  string `json:"phase"`
	Actual string `json:"actual"`
}

type AuthoringBoundaries struct {
	Policy    string `json:"policy"`
	Sandbox   string `json:"sandbox"`
	Allowlist string `json:"allowlist"`
	Egress    string `json:"egress"`
}

type AuthoringResult struct {
	Status      string `json:"status"`
	Phase       string `json:"phase"`
	ReasonCode  string `json:"reason_code,omitempty"`
	Field       string `json:"field,omitempty"`
	Expected    string `json:"expected,omitempty"`
	Actual      string `json:"actual,omitempty"`
	Remediation string `json:"remediation,omitempty"`
}

type authoringError struct{ reason, field, message string }

func (e *authoringError) Error() string {
	return fmt.Sprintf("[%s] %s: %s", e.reason, e.field, e.message)
}
func AuthoringErrorReason(err error) string {
	if e, ok := err.(*authoringError); ok {
		return e.reason
	}
	return "authoring.error"
}
func AuthoringErrorField(err error) string {
	if e, ok := err.(*authoringError); ok {
		return e.field
	}
	return ""
}

func DecodeAuthoringCase(raw []byte) (AuthoringCase, error) {
	var c AuthoringCase
	if err := json.Unmarshal(raw, &c); err != nil {
		return AuthoringCase{}, &authoringError{ReasonAuthoringInvalidField, "case", "invalid JSON"}
	}
	if c.Profile != AuthoringProfileV1 {
		return AuthoringCase{}, &authoringError{ReasonAuthoringUnsupportedProfile, "profile", "unsupported profile"}
	}
	c.Case = strings.TrimSpace(c.Case)
	if c.Case == "" {
		return AuthoringCase{}, &authoringError{ReasonAuthoringMissingField, "case", "required field is missing"}
	}
	if len(c.Case) > MaxAuthoringNameBytes {
		return AuthoringCase{}, &authoringError{ReasonAuthoringInvalidField, "case", "value exceeds bound"}
	}
	if len(c.Candidates) == 0 {
		return AuthoringCase{}, &authoringError{ReasonAuthoringMissingField, "candidates", "at least one candidate is required"}
	}
	if len(c.Candidates) > MaxAuthoringCandidates {
		return AuthoringCase{}, &authoringError{ReasonAuthoringInvalidField, "candidates", "candidate count exceeds bound"}
	}
	sort.SliceStable(c.Candidates, func(i, j int) bool {
		if c.Candidates[i].Name != c.Candidates[j].Name {
			return c.Candidates[i].Name < c.Candidates[j].Name
		}
		return c.Candidates[i].Kind < c.Candidates[j].Kind
	})
	c.Admission.RuntimeVersion = strings.TrimSpace(c.Admission.RuntimeVersion)
	c.Admission.AvailableCapabilities = normalizeList(c.Admission.AvailableCapabilities)
	if c.Lifecycle.Action == "" {
		c.Lifecycle.Action = "success"
	}
	if c.Lifecycle.Action != "success" && c.Lifecycle.Action != "timeout" && c.Lifecycle.Action != "panic" && c.Lifecycle.Action != "invalid_result" && c.Lifecycle.Action != "finalize_failure" && c.Lifecycle.Action != "reload_failure" && c.Lifecycle.Action != "stale_generation" {
		return AuthoringCase{}, &authoringError{ReasonAuthoringInvalidField, "lifecycle.action", "unsupported action"}
	}
	if c.Lifecycle.FailurePolicy == "" {
		c.Lifecycle.FailurePolicy = FailurePolicyDeny
	}
	if c.Lifecycle.FailurePolicy != FailurePolicySkip && c.Lifecycle.FailurePolicy != FailurePolicyDeny && c.Lifecycle.FailurePolicy != FailurePolicyDegrade {
		return AuthoringCase{}, &authoringError{ReasonAuthoringInvalidField, "lifecycle.failure_policy", "unsupported policy"}
	}
	for _, boundary := range []struct{ field, value string }{
		{field: "policy", value: c.Boundaries.Policy},
		{field: "sandbox", value: c.Boundaries.Sandbox},
		{field: "allowlist", value: c.Boundaries.Allowlist},
		{field: "egress", value: c.Boundaries.Egress},
	} {
		if boundary.value != "" && boundary.value != "allow" && boundary.value != "deny" {
			return AuthoringCase{}, &authoringError{ReasonAuthoringInvalidField, "boundaries." + boundary.field, "unsupported boundary decision"}
		}
	}
	if c.Expected.Status == "" {
		return AuthoringCase{}, &authoringError{ReasonAuthoringMissingField, "expected.status", "required field is missing"}
	}
	if c.Expected.Status != AuthoringStatusPassed && c.Expected.Status != AuthoringStatusFailed {
		return AuthoringCase{}, &authoringError{ReasonAuthoringInvalidField, "expected.status", "unsupported status"}
	}
	validPhases := map[string]bool{AuthoringPhaseSchema: true, AuthoringPhaseManifest: true, AuthoringPhaseAdmission: true, AuthoringPhaseLifecycle: true, AuthoringPhaseReload: true, AuthoringPhaseComplete: true}
	if !validPhases[c.Expected.Phase] {
		return AuthoringCase{}, &authoringError{ReasonAuthoringInvalidField, "expected.phase", "unsupported phase"}
	}
	return c, nil
}

// EvaluateAuthoringProjection evaluates either the Run or Stream projection.
// Both projections intentionally share the same source-owned conformance path.
func EvaluateAuthoringProjection(raw []byte, projection string) (AuthoringResult, error) {
	if projection != AuthoringProjectionRun && projection != AuthoringProjectionStream {
		return AuthoringResult{}, &authoringError{ReasonAuthoringInvalidField, "projection", "must be run or stream"}
	}
	return EvaluateAuthoringCase(raw)
}

func NewAuthoringResult(status, phase, reason, field, expected, actual string) AuthoringResult {
	r := AuthoringResult{Status: status, Phase: phase, ReasonCode: boundAuthoringText(reason, MaxAuthoringTextBytes), Field: boundAuthoringText(field, MaxAuthoringFieldBytes), Expected: boundAuthoringText(expected, MaxAuthoringTextBytes), Actual: boundAuthoringText(actual, MaxAuthoringTextBytes)}
	if r.ReasonCode != "" {
		r.Remediation = AuthoringRemediation(r.ReasonCode)
	}
	return r
}

func boundAuthoringText(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) <= max {
		return value
	}
	return value[:max]
}

func AuthoringRemediation(reason string) string {
	guidance := map[string]string{
		ReasonAuthoringMissingField:              "add the required field and rerun conformance",
		ReasonAuthoringInvalidField:              "correct the field format or bound and rerun conformance",
		ReasonAuthoringUnsupportedProfile:        "use the supported external_extension_authoring.v1 profile",
		ReasonMissingField:                       "add the required manifest field before activation",
		ReasonInvalidField:                       "correct the manifest field format before activation",
		ReasonAmbiguousConflict:                  "use one digest at each source precedence level",
		ReasonCompatibilityMismatch:              "set a compatibility range containing the runtime version",
		"adapter.capability.missing_required":    "declare only capabilities supplied by the runtime",
		"adapter.capability.optional_downgraded": "accept the optional capability downgrade or provide the capability",
		ReasonTimeout:                            "increase the bounded timeout or make the lifecycle action complete promptly",
		ReasonPanic:                              "recover the lifecycle panic and return a valid result",
		ReasonInvalidResult:                      "return a non-nil bounded lifecycle result",
		ReasonExecutionFailed:                    "repair the lifecycle action and preserve its failure classification",
		ReasonAuthoringFinalizeFailed:            "make finalize idempotent and release extension-owned resources",
		ReasonAuthoringReloadFailed:              "fix the new generation before retrying reload; retain the previous generation",
		ReasonAuthoringStaleGeneration:           "discard events from stale generations and use the active generation",
		ReasonAuthoringPolicyDenied:              "request the required policy admission through the owning runtime",
		ReasonAuthoringSandboxDenied:             "declare and satisfy the configured sandbox boundary",
		ReasonAuthoringAllowlistDenied:           "add an explicit allowlist entry or keep the extension blocked",
		ReasonAuthoringEgressDenied:              "use an approved egress rule or remove the network action",
		ReasonAuthoringReplayDrift:               "inspect canonical expected and actual output before changing a fixture",
	}
	if value, ok := guidance[reason]; ok {
		return value
	}
	return "review the conformance reason and owning contract"
}

// EvaluateAuthoringCase runs an offline case through existing extension owners.
// It only executes controlled actions declared by the case schema.
func EvaluateAuthoringCase(raw []byte) (AuthoringResult, error) {
	c, err := DecodeAuthoringCase(raw)
	if err != nil {
		return NewAuthoringResult(AuthoringStatusFailed, AuthoringPhaseSchema, AuthoringErrorReason(err), AuthoringErrorField(err), "valid case", "invalid"), nil
	}
	resolution, err := ResolveResources(c.Candidates)
	if err != nil {
		phase := AuthoringPhaseAdmission
		if ErrorCode(err) == ReasonMissingField || ErrorCode(err) == ReasonInvalidField {
			phase = AuthoringPhaseManifest
		}
		field := ErrorField(err)
		if field == "" {
			field = "candidates"
		}
		return NewAuthoringResult(AuthoringStatusFailed, phase, ErrorCode(err), field, "", "inactive"), nil
	}
	for _, boundary := range []struct {
		field, value, reason string
	}{
		{field: "policy", value: c.Boundaries.Policy, reason: ReasonAuthoringPolicyDenied},
		{field: "sandbox", value: c.Boundaries.Sandbox, reason: ReasonAuthoringSandboxDenied},
		{field: "allowlist", value: c.Boundaries.Allowlist, reason: ReasonAuthoringAllowlistDenied},
		{field: "egress", value: c.Boundaries.Egress, reason: ReasonAuthoringEgressDenied},
	} {
		if boundary.value == "deny" {
			return NewAuthoringResult(AuthoringStatusFailed, AuthoringPhaseAdmission, boundary.reason, "boundaries."+boundary.field, "", "inactive"), nil
		}
	}
	if len(resolution.Selected) == 0 {
		return NewAuthoringResult(AuthoringStatusFailed, AuthoringPhaseAdmission, ReasonAuthoringMissingField, "candidates", "", "inactive"), nil
	}
	descriptor := resolution.Selected[0]
	capability, err := NegotiateCapabilities(descriptor, c.Admission.AvailableCapabilities, c.Admission.BestEffort)
	if err != nil {
		return NewAuthoringResult(AuthoringStatusFailed, AuthoringPhaseAdmission, ErrorCode(err), "capabilities.required", "", "inactive"), nil
	}
	decision, err := Admit(descriptor, AdmissionInput{RuntimeVersion: c.Admission.RuntimeVersion, AvailableCapabilities: c.Admission.AvailableCapabilities})
	if err != nil {
		return NewAuthoringResult(AuthoringStatusFailed, AuthoringPhaseAdmission, ErrorCode(err), "compat", "", "inactive"), nil
	}
	if decision.Outcome != AdmissionAllow || !decision.Activated {
		return NewAuthoringResult(AuthoringStatusFailed, AuthoringPhaseAdmission, ReasonAdmission, "admission", "", "inactive"), nil
	}
	if capability.Downgraded {
		return NewAuthoringResult(AuthoringStatusPassed, AuthoringPhaseComplete, "adapter.capability.optional_downgraded", "capabilities.optional", "", "degraded"), nil
	}
	if c.Lifecycle.Action == "reload_failure" {
		manager := NewGenerationManager()
		active, reloadErr := manager.Reload(descriptor)
		invalid := descriptor
		invalid.Digest = ""
		if reloadErr != nil {
			return AuthoringResult{}, reloadErr
		}
		if _, reloadErr = manager.Reload(invalid); reloadErr == nil || manager.ActiveGeneration() != active.Generation {
			return NewAuthoringResult(AuthoringStatusFailed, AuthoringPhaseReload, ReasonAuthoringReplayDrift, "lifecycle.action", "previous_generation_active", "reload_isolation_drift"), nil
		}
		return NewAuthoringResult(AuthoringStatusFailed, AuthoringPhaseReload, ReasonAuthoringReloadFailed, "lifecycle.action", "", "previous_generation_active"), nil
	}
	if c.Lifecycle.Action == "stale_generation" {
		manager := NewGenerationManager()
		first, reloadErr := manager.Reload(descriptor)
		if reloadErr != nil {
			return AuthoringResult{}, reloadErr
		}
		next := descriptor
		next.Digest += "b"
		second, reloadErr := manager.Reload(next)
		if reloadErr != nil {
			return AuthoringResult{}, reloadErr
		}
		if manager.AcceptsEvent(first.Generation) || !manager.AcceptsEvent(second.Generation) {
			return NewAuthoringResult(AuthoringStatusFailed, AuthoringPhaseReload, ReasonAuthoringReplayDrift, "lifecycle.action", "stale_event_suppressed", "generation_isolation_drift"), nil
		}
		return NewAuthoringResult(AuthoringStatusFailed, AuthoringPhaseReload, ReasonAuthoringStaleGeneration, "lifecycle.action", "", "stale_event_suppressed"), nil
	}
	result := executeAuthoringAction(c.Lifecycle)
	if result.Outcome != ExecutionSucceeded {
		return NewAuthoringResult(AuthoringStatusFailed, AuthoringPhaseLifecycle, result.Reason, "lifecycle.action", "", string(result.Outcome)), nil
	}
	return NewAuthoringResult(AuthoringStatusPassed, AuthoringPhaseComplete, "", "", "", "activated"), nil
}

func executeAuthoringAction(lifecycle AuthoringLifecycle) ExecutionResult {
	options := ExecutionOptions{FailurePolicy: lifecycle.FailurePolicy, Timeout: 20 * time.Millisecond}
	switch lifecycle.Action {
	case "timeout":
		return Execute(context.Background(), options, func(ctx context.Context) (any, error) { <-ctx.Done(); return nil, ctx.Err() })
	case "panic":
		return Execute(context.Background(), options, func(context.Context) (any, error) { panic("controlled authoring fault") })
	case "invalid_result":
		return Execute(context.Background(), options, func(context.Context) (any, error) { return nil, nil })
	case "finalize_failure":
		return applyFailure(options.FailurePolicy, ReasonAuthoringFinalizeFailed)
	default:
		return Execute(context.Background(), options, func(context.Context) (any, error) { return "ok", nil })
	}
}
