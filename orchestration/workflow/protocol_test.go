package workflow

import (
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
)

func TestProtocolStepFromWorkflowStepPreservesCorrelation(t *testing.T) {
	ref, err := ProtocolStepRef(Step{StepID: "step-1", TaskID: "task-1", AgentID: "agent-1", Kind: StepKindA2A}, "run-1", "session-1")
	if err != nil {
		t.Fatalf("ProtocolStepRef() error = %v", err)
	}
	if ref.StepID != "step-1" || ref.RunID != "run-1" || ref.Kind != types.ProtocolStepKindA2A || ref.Source != types.ProtocolSourceWorkflow {
		t.Fatalf("ref = %#v", ref)
	}
}
