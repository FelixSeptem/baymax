package teams

import (
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
)

func TestProtocolStepFromTeamTaskPreservesCorrelation(t *testing.T) {
	ref, err := ProtocolTaskRef(Plan{RunID: "run-1", TeamID: "team-1", WorkflowID: "workflow-1", StepID: "step-1"}, Task{TaskID: "task-1", AgentID: "agent-1"}, "session-1")
	if err != nil {
		t.Fatalf("ProtocolTaskRef() error = %v", err)
	}
	if ref.StepID != "task-1" || ref.RunID != "run-1" || ref.ParentStepID != "step-1" || ref.Kind != types.ProtocolStepKindTeamTask || ref.Source != types.ProtocolSourceTeams {
		t.Fatalf("ref = %#v", ref)
	}
}
