package types

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// FailureFamily is the normalized cross-domain failure classification. Source
// error classes and reason codes remain available alongside this projection.
type FailureFamily string

const (
	FailureFamilyNone             FailureFamily = "none"
	FailureFamilyRejected         FailureFamily = "rejected"
	FailureFamilyPolicyDenied     FailureFamily = "policy_denied"
	FailureFamilyRuntimeFailed    FailureFamily = "runtime_failed"
	FailureFamilyTimedOut         FailureFamily = "timed_out"
	FailureFamilyCanceled         FailureFamily = "canceled"
	FailureFamilyRetryExhausted   FailureFamily = "retry_exhausted"
	FailureFamilyRecoveryConflict FailureFamily = "recovery_conflict"
)

// ExecutionPhase distinguishes a failure before execution from one after
// observable work has started.
type ExecutionPhase string

const (
	ExecutionPhasePreExecution ExecutionPhase = "pre_execution"
	ExecutionPhasePostStart    ExecutionPhase = "post_start"
)

// TerminalOutcome is an additive, source-owned projection of one terminal Run
// result. It does not own retry, queueing, cancellation, or recovery.
type TerminalOutcome struct {
	RunID         string         `json:"run_id"`
	SessionID     string         `json:"session_id,omitempty"`
	State         RunState       `json:"state"`
	FailureFamily FailureFamily  `json:"failure_family"`
	Phase         ExecutionPhase `json:"phase"`
	SourceReason  string         `json:"source_reason,omitempty"`
	ErrorClass    ErrorClass     `json:"error_class,omitempty"`
	Retryable     bool           `json:"retryable,omitempty"`
	Resumable     bool           `json:"resumable,omitempty"`
	Attempt       int            `json:"attempt,omitempty"`
	AttemptLimit  int            `json:"attempt_limit,omitempty"`
	CausationID   string         `json:"causation_id,omitempty"`
}

// TerminalOutcomeFromRunResult derives a normalized projection from the
// existing Runner result without changing source-owned lifecycle decisions.
func TerminalOutcomeFromRunResult(result RunResult, sessionID string) (TerminalOutcome, error) {
	if result.TerminalOutcome != nil {
		outcome := *result.TerminalOutcome
		if strings.TrimSpace(outcome.RunID) == "" {
			outcome.RunID = strings.TrimSpace(result.RunID)
		}
		if strings.TrimSpace(outcome.SessionID) == "" {
			outcome.SessionID = strings.TrimSpace(sessionID)
		}
		return outcome, outcome.Validate()
	}
	outcome := TerminalOutcome{
		RunID:     strings.TrimSpace(result.RunID),
		SessionID: strings.TrimSpace(sessionID),
		Phase:     ExecutionPhasePostStart,
	}
	if result.Error == nil {
		outcome.State = RunStateCompleted
		outcome.FailureFamily = FailureFamilyNone
		return outcome, outcome.Validate()
	}
	outcome.State = RunStateFailed
	outcome.FailureFamily = FailureFamilyRuntimeFailed
	outcome.ErrorClass = result.Error.Class
	outcome.Retryable = result.Error.Retryable
	outcome.SourceReason = detailString(result.Error.Details, "reason_code")
	if outcome.SourceReason == "" {
		outcome.SourceReason = detailString(result.Error.Details, "provider_reason")
	}
	outcome.Resumable = detailBool(result.Error.Details, "resumable")
	outcome.Attempt = detailInt(result.Error.Details, "attempt")
	outcome.AttemptLimit = detailInt(result.Error.Details, "attempt_limit")
	outcome.CausationID = detailString(result.Error.Details, "causation_id")
	if strings.Contains(strings.ToLower(outcome.SourceReason), "cancel") || result.Error.Class == ErrPolicyTimeout && strings.Contains(strings.ToLower(result.Error.Message), "cancel") {
		outcome.State = RunStateCanceled
		outcome.FailureFamily = FailureFamilyCanceled
		outcome.Retryable = false
	}
	lowerReason := strings.ToLower(outcome.SourceReason)
	lowerMessage := strings.ToLower(strings.TrimSpace(result.Error.Message))
	if result.Error.Class == ErrTool && (detailString(result.Error.Details, "validation") != "" || strings.Contains(lowerMessage, "validation")) {
		outcome.FailureFamily = FailureFamilyRejected
		outcome.Phase = ExecutionPhasePreExecution
	}
	if strings.Contains(lowerReason, "policy") || result.Error.Class == ErrSecurity {
		outcome.FailureFamily = FailureFamilyPolicyDenied
		outcome.Phase = ExecutionPhasePreExecution
		if phase := detailString(result.Error.Details, "dispatch_phase"); phase != "" || strings.Contains(lowerMessage, "tool call") || strings.Contains(lowerReason, "sandbox") {
			outcome.Phase = ExecutionPhasePostStart
		}
	}
	if result.Error.Class == ErrSecurity && !strings.Contains(lowerReason, "policy") && !strings.Contains(lowerReason, "deny") {
		outcome.FailureFamily = FailureFamilyRuntimeFailed
		outcome.Phase = ExecutionPhasePostStart
	}
	if result.Error.Class == ErrPolicyTimeout && outcome.State != RunStateCanceled {
		outcome.FailureFamily = FailureFamilyTimedOut
	}
	if strings.Contains(lowerReason, "retry_exhaust") {
		outcome.FailureFamily = FailureFamilyRetryExhausted
	}
	return outcome, outcome.Validate()
}

func detailString(details map[string]any, key string) string {
	if details == nil {
		return ""
	}
	value, ok := details[key]
	if !ok {
		return ""
	}
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func detailBool(details map[string]any, key string) bool {
	if details == nil {
		return false
	}
	value, ok := details[key]
	if !ok {
		return false
	}
	parsed, _ := value.(bool)
	return parsed
}

func detailInt(details map[string]any, key string) int {
	if details == nil {
		return 0
	}
	value, ok := details[key]
	if !ok {
		return 0
	}
	switch parsed := value.(type) {
	case int:
		return parsed
	case int8:
		return int(parsed)
	case int16:
		return int(parsed)
	case int32:
		return int(parsed)
	case int64:
		return int(parsed)
	case uint:
		return int(parsed)
	case uint8:
		return int(parsed)
	case uint16:
		return int(parsed)
	case uint32:
		return int(parsed)
	case uint64:
		return int(parsed)
	case float64:
		return int(parsed)
	default:
		return 0
	}
}

func (o TerminalOutcome) Validate() error {
	if strings.TrimSpace(o.RunID) == "" {
		return fmt.Errorf("terminal_outcome run_id is required")
	}
	if o.State != RunStateCompleted && o.State != RunStateFailed && o.State != RunStateCanceled {
		return fmt.Errorf("terminal_outcome state %q is not terminal", o.State)
	}
	switch o.FailureFamily {
	case FailureFamilyNone, FailureFamilyRejected, FailureFamilyPolicyDenied,
		FailureFamilyRuntimeFailed, FailureFamilyTimedOut, FailureFamilyCanceled,
		FailureFamilyRetryExhausted, FailureFamilyRecoveryConflict:
	default:
		return fmt.Errorf("terminal_outcome failure_family %q is unsupported", o.FailureFamily)
	}
	if o.Phase != ExecutionPhasePreExecution && o.Phase != ExecutionPhasePostStart {
		return fmt.Errorf("terminal_outcome phase %q is unsupported", o.Phase)
	}
	if o.State == RunStateCompleted && o.FailureFamily != FailureFamilyNone {
		return fmt.Errorf("terminal_outcome completed state requires failure_family none")
	}
	if o.State == RunStateCanceled && o.FailureFamily != FailureFamilyCanceled {
		return fmt.Errorf("terminal_outcome canceled state requires failure_family canceled")
	}
	if o.State == RunStateFailed && o.FailureFamily == FailureFamilyNone {
		return fmt.Errorf("terminal_outcome failed state requires a failure family")
	}
	if o.State == RunStateCompleted && o.Phase == ExecutionPhasePreExecution {
		return fmt.Errorf("terminal_outcome completed state cannot be pre_execution")
	}
	if o.Attempt < 0 || o.AttemptLimit < 0 {
		return fmt.Errorf("terminal_outcome attempt values must be non-negative")
	}
	if o.AttemptLimit > 0 && o.Attempt > o.AttemptLimit {
		return fmt.Errorf("terminal_outcome attempt exceeds attempt_limit")
	}
	return nil
}

type TerminalPublishResult string

const (
	TerminalPublishAccepted   TerminalPublishResult = "accepted"
	TerminalPublishIdempotent TerminalPublishResult = "idempotent"
	TerminalPublishConflict   TerminalPublishResult = "conflict"
)

// TerminalOutcomeArbiter applies first-terminal-wins semantics for one Run.
type TerminalOutcomeArbiter struct {
	mu        sync.Mutex
	terminal  TerminalOutcome
	published bool
	conflicts []TerminalOutcome
}

func NewTerminalOutcomeArbiter() *TerminalOutcomeArbiter {
	return &TerminalOutcomeArbiter{}
}

func (a *TerminalOutcomeArbiter) Publish(outcome TerminalOutcome) (TerminalPublishResult, error) {
	if a == nil {
		return "", fmt.Errorf("terminal outcome arbiter is nil")
	}
	if err := outcome.Validate(); err != nil {
		return "", err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.published {
		a.terminal = outcome
		a.published = true
		return TerminalPublishAccepted, nil
	}
	if reflect.DeepEqual(a.terminal, outcome) {
		return TerminalPublishIdempotent, nil
	}
	a.conflicts = append(a.conflicts, outcome)
	return TerminalPublishConflict, nil
}

func (a *TerminalOutcomeArbiter) Terminal() (TerminalOutcome, bool) {
	if a == nil {
		return TerminalOutcome{}, false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.terminal, a.published
}

func (a *TerminalOutcomeArbiter) Conflicts() []TerminalOutcome {
	if a == nil {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]TerminalOutcome(nil), a.conflicts...)
}
