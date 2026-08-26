package a2a

import (
	"fmt"
	"strings"

	"github.com/FelixSeptem/baymax/core/types"
)

// ProtocolRefsFromTaskRecord projects A2A task lifecycle and correlation into
// a protocol RunRef and child StepRef.
func ProtocolRefsFromTaskRecord(record TaskRecord, sessionID string) (types.RunRef, types.StepRef, error) {
	runID := strings.TrimSpace(record.TaskID)
	state := types.RunStateWorking
	switch record.Status {
	case StatusSubmitted:
		state = types.RunStateSubmitted
	case StatusSucceeded:
		state = types.RunStateCompleted
	case StatusFailed:
		state = types.RunStateFailed
	case StatusCanceled:
		state = types.RunStateCanceled
	}
	run := types.RunRef{RunID: runID, SessionID: strings.TrimSpace(sessionID), State: state, Source: types.ProtocolSourceA2A}
	step := types.StepRef{StepID: runID, RunID: runID, SessionID: strings.TrimSpace(sessionID), ParentStepID: strings.TrimSpace(record.StepID), Kind: types.ProtocolStepKindA2A, Source: types.ProtocolSourceA2A, CausationID: strings.TrimSpace(record.AttemptID)}
	if err := run.ValidateProtocolReference(); err != nil {
		return types.RunRef{}, types.StepRef{}, fmt.Errorf("a2a protocol run: %w", err)
	}
	if err := step.ValidateProtocolReference(); err != nil {
		return types.RunRef{}, types.StepRef{}, fmt.Errorf("a2a protocol step: %w", err)
	}
	return run, step, nil
}
