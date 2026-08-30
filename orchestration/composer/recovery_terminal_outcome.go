package composer

import (
	"errors"
	"fmt"
	"strings"

	"github.com/FelixSeptem/baymax/core/types"
)

// RecoveryTerminalOutcome projects an already-classified recovery error. The
// recovery owner remains responsible for deciding whether continuation is
// allowed; this helper only records the source classification and phase.
func RecoveryTerminalOutcome(runID string, err error, phase types.ExecutionPhase) (types.TerminalOutcome, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return types.TerminalOutcome{}, fmt.Errorf("run_id is required")
	}
	if err == nil {
		return types.TerminalOutcome{}, fmt.Errorf("recovery error is required")
	}
	if phase != types.ExecutionPhasePreExecution && phase != types.ExecutionPhasePostStart {
		return types.TerminalOutcome{}, fmt.Errorf("unsupported recovery phase %q", phase)
	}
	return types.TerminalOutcome{
		RunID:         runID,
		State:         types.RunStateFailed,
		FailureFamily: types.FailureFamilyRecoveryConflict,
		Phase:         phase,
		SourceReason:  string(recoveryErrorCode(err)),
	}, nil
}

func recoveryErrorCode(err error) RecoveryErrorCode {
	var recoveryErr *RecoveryError
	if errors.As(err, &recoveryErr) && recoveryErr != nil && recoveryErr.Code != "" {
		return recoveryErr.Code
	}
	return RecoveryErrorConflict
}
