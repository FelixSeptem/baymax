package scheduler

import (
	"fmt"
	"strings"

	"github.com/FelixSeptem/baymax/core/types"
)

// ProtocolAttemptRef projects a scheduler attempt into a protocol StepRef.
func ProtocolAttemptRef(task Task, attempt Attempt, sessionID string) (types.StepRef, error) {
	ref := types.StepRef{StepID: strings.TrimSpace(attempt.AttemptID), RunID: strings.TrimSpace(task.RunID), SessionID: strings.TrimSpace(sessionID), ParentStepID: strings.TrimSpace(task.StepID), Kind: types.ProtocolStepKindScheduler, Source: types.ProtocolSourceScheduler, CausationID: strings.TrimSpace(task.TaskID)}
	if err := ref.ValidateProtocolReference(); err != nil {
		return types.StepRef{}, fmt.Errorf("scheduler protocol attempt: %w", err)
	}
	return ref, nil
}
