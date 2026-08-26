package teams

import (
	"fmt"
	"strings"

	"github.com/FelixSeptem/baymax/core/types"
)

// ProtocolTaskRef projects a team task into a protocol StepRef.
func ProtocolTaskRef(plan Plan, task Task, sessionID string) (types.StepRef, error) {
	ref := types.StepRef{StepID: strings.TrimSpace(task.TaskID), RunID: strings.TrimSpace(plan.RunID), SessionID: strings.TrimSpace(sessionID), ParentStepID: strings.TrimSpace(plan.StepID), Kind: types.ProtocolStepKindTeamTask, Source: types.ProtocolSourceTeams}
	if err := ref.ValidateProtocolReference(); err != nil {
		return types.StepRef{}, fmt.Errorf("teams protocol task: %w", err)
	}
	return ref, nil
}
