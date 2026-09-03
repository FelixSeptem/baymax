package types

import (
	"fmt"
	"strings"
)

// ToolLifecycleStage is the bounded logical stage projection for one tool call.
type ToolLifecycleStage string

const (
	ToolLifecycleStagePrepare   ToolLifecycleStage = "prepare"
	ToolLifecycleStageValidate  ToolLifecycleStage = "validate"
	ToolLifecycleStageAuthorize ToolLifecycleStage = "authorize"
	ToolLifecycleStageExecute   ToolLifecycleStage = "execute"
	ToolLifecycleStageFinalize  ToolLifecycleStage = "finalize"
)

// ToolLifecycleOutcome describes the result of one logical stage.
type ToolLifecycleOutcome string

const (
	ToolLifecycleOutcomeSucceeded     ToolLifecycleOutcome = "succeeded"
	ToolLifecycleOutcomeFailed        ToolLifecycleOutcome = "failed"
	ToolLifecycleOutcomeSkipped       ToolLifecycleOutcome = "skipped"
	ToolLifecycleOutcomeNotApplicable ToolLifecycleOutcome = "not_applicable"
)

// ToolFailureOrigin is a bounded source classification layered over existing errors.
type ToolFailureOrigin string

const (
	ToolFailureOriginUnknown                ToolFailureOrigin = "unknown"
	ToolFailureOriginLookup                 ToolFailureOrigin = "lookup"
	ToolFailureOriginValidation             ToolFailureOrigin = "validation"
	ToolFailureOriginPolicyDenied           ToolFailureOrigin = "policy_denied"
	ToolFailureOriginSandboxDenied          ToolFailureOrigin = "sandbox_denied"
	ToolFailureOriginSandboxFailure         ToolFailureOrigin = "sandbox_failure"
	ToolFailureOriginMiddlewareShortCircuit ToolFailureOrigin = "middleware_short_circuit"
	ToolFailureOriginMiddlewareFailure      ToolFailureOrigin = "middleware_failure"
	ToolFailureOriginPanic                  ToolFailureOrigin = "panic"
	ToolFailureOriginTimeout                ToolFailureOrigin = "timeout"
	ToolFailureOriginCanceled               ToolFailureOrigin = "canceled"
	ToolFailureOriginRetryExhausted         ToolFailureOrigin = "retry_exhausted"
	ToolFailureOriginExecution              ToolFailureOrigin = "execution"
)

type ToolLifecycleStageInput struct {
	Stage      ToolLifecycleStage   `json:"stage"`
	Outcome    ToolLifecycleOutcome `json:"outcome"`
	ReasonCode string               `json:"reason_code,omitempty"`
	Started    bool                 `json:"started,omitempty"`
}

type ToolLifecycleInput struct {
	SessionID     string                    `json:"session_id,omitempty"`
	RunID         string                    `json:"run_id,omitempty"`
	StepID        string                    `json:"step_id,omitempty"`
	CallID        string                    `json:"call_id"`
	ToolName      string                    `json:"tool_name"`
	Source        string                    `json:"source,omitempty"`
	InputIndex    int                       `json:"input_index"`
	FailureOrigin ToolFailureOrigin         `json:"failure_origin,omitempty"`
	AttemptCount  int                       `json:"attempt_count,omitempty"`
	Stages        []ToolLifecycleStageInput `json:"stages"`
}

type ToolLifecycleStageProjection struct {
	Stage      ToolLifecycleStage   `json:"stage"`
	Outcome    ToolLifecycleOutcome `json:"outcome"`
	ReasonCode string               `json:"reason_code,omitempty"`
	Started    bool                 `json:"started,omitempty"`
}

type ToolLifecycleProjection struct {
	SessionID        string                         `json:"session_id,omitempty"`
	RunID            string                         `json:"run_id,omitempty"`
	StepID           string                         `json:"step_id,omitempty"`
	CallID           string                         `json:"call_id"`
	ToolName         string                         `json:"tool_name"`
	Source           string                         `json:"source,omitempty"`
	InputIndex       int                            `json:"input_index"`
	FailureOrigin    ToolFailureOrigin              `json:"failure_origin,omitempty"`
	AttemptCount     int                            `json:"attempt_count,omitempty"`
	ExecutionStarted bool                           `json:"execution_started,omitempty"`
	Finalized        bool                           `json:"finalized,omitempty"`
	Stages           []ToolLifecycleStageProjection `json:"stages"`
	finalizeAccepted bool
}

func NewToolLifecycleProjection(in ToolLifecycleInput) (ToolLifecycleProjection, error) {
	callID := strings.TrimSpace(in.CallID)
	toolName := strings.TrimSpace(in.ToolName)
	if callID == "" {
		return ToolLifecycleProjection{}, fmt.Errorf("tool lifecycle call_id is required")
	}
	if toolName == "" {
		return ToolLifecycleProjection{}, fmt.Errorf("tool lifecycle tool_name is required")
	}
	if len(in.Stages) == 0 {
		return ToolLifecycleProjection{}, fmt.Errorf("tool lifecycle stages are required")
	}
	want := []ToolLifecycleStage{ToolLifecycleStagePrepare, ToolLifecycleStageValidate, ToolLifecycleStageAuthorize, ToolLifecycleStageExecute, ToolLifecycleStageFinalize}
	stages := make([]ToolLifecycleStageProjection, len(in.Stages))
	seen := make(map[ToolLifecycleStage]struct{}, len(in.Stages))
	started := false
	for i, stage := range in.Stages {
		if i >= len(want) || stage.Stage != want[i] {
			return ToolLifecycleProjection{}, fmt.Errorf("tool lifecycle stage order invalid at index %d: got %q", i, stage.Stage)
		}
		if _, ok := seen[stage.Stage]; ok {
			return ToolLifecycleProjection{}, fmt.Errorf("tool lifecycle stage %q repeated", stage.Stage)
		}
		if !validToolLifecycleOutcome(stage.Outcome) {
			return ToolLifecycleProjection{}, fmt.Errorf("tool lifecycle outcome %q invalid", stage.Outcome)
		}
		seen[stage.Stage] = struct{}{}
		stages[i] = ToolLifecycleStageProjection{Stage: stage.Stage, Outcome: stage.Outcome, ReasonCode: strings.TrimSpace(stage.ReasonCode), Started: stage.Started}
		if stage.Stage == ToolLifecycleStageExecute && stage.Started {
			started = true
		}
	}
	if len(stages) != len(want) || stages[len(stages)-1].Stage != ToolLifecycleStageFinalize {
		return ToolLifecycleProjection{}, fmt.Errorf("tool lifecycle must end with finalize")
	}
	origin := in.FailureOrigin
	if origin == "" {
		origin = ToolFailureOriginUnknown
	}
	return ToolLifecycleProjection{
		SessionID: strings.TrimSpace(in.SessionID), RunID: strings.TrimSpace(in.RunID), StepID: strings.TrimSpace(in.StepID),
		CallID: callID, ToolName: toolName, Source: strings.TrimSpace(in.Source), InputIndex: in.InputIndex,
		FailureOrigin: origin, AttemptCount: in.AttemptCount, ExecutionStarted: started, Finalized: stages[len(stages)-1].Outcome == ToolLifecycleOutcomeSucceeded,
		Stages: stages,
	}, nil
}

func validToolLifecycleOutcome(outcome ToolLifecycleOutcome) bool {
	switch outcome {
	case ToolLifecycleOutcomeSucceeded, ToolLifecycleOutcomeFailed, ToolLifecycleOutcomeSkipped, ToolLifecycleOutcomeNotApplicable:
		return true
	default:
		return false
	}
}

func (p *ToolLifecycleProjection) StageNames() []ToolLifecycleStage {
	if p == nil {
		return nil
	}
	out := make([]ToolLifecycleStage, 0, len(p.Stages))
	for _, stage := range p.Stages {
		out = append(out, stage.Stage)
	}
	return out
}

// AcceptFinalize accepts one finalization observation and deduplicates repeats.
func (p *ToolLifecycleProjection) AcceptFinalize() bool {
	if p == nil || p.finalizeAccepted {
		return false
	}
	p.finalizeAccepted = true
	p.Finalized = true
	return true
}
