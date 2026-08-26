package workflow

import (
	"fmt"
	"strings"

	"github.com/FelixSeptem/baymax/core/types"
)

// ProtocolStepRef projects a workflow step while retaining workflow ownership.
func ProtocolStepRef(step Step, runID, sessionID string) (types.StepRef, error) {
	kind := types.ProtocolStepKindWorkflow
	switch step.Kind {
	case StepKindTool, StepKindMCP:
		kind = types.ProtocolStepKindTool
	case StepKindA2A:
		kind = types.ProtocolStepKindA2A
	}
	ref := types.StepRef{StepID: strings.TrimSpace(step.StepID), RunID: strings.TrimSpace(runID), SessionID: strings.TrimSpace(sessionID), ParentStepID: strings.TrimSpace(step.TaskID), Kind: kind, Source: types.ProtocolSourceWorkflow}
	if err := ref.ValidateProtocolReference(); err != nil {
		return types.StepRef{}, fmt.Errorf("workflow protocol step: %w", err)
	}
	return ref, nil
}
