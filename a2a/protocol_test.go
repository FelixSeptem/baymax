package a2a

import (
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
)

func TestProtocolRefsFromA2ATaskPreserveCorrelation(t *testing.T) {
	record := TaskRecord{TaskID: "task-1", WorkflowID: "workflow-1", TeamID: "team-1", StepID: "step-1", AttemptID: "attempt-1", AgentID: "agent-1", PeerID: "peer-1", Status: StatusRunning}
	run, step, err := ProtocolRefsFromTaskRecord(record, "session-1")
	if err != nil {
		t.Fatalf("ProtocolRefsFromTaskRecord() error = %v", err)
	}
	if run.RunID != "task-1" || run.State != types.RunStateWorking || step.StepID != "task-1" || step.ParentStepID != "step-1" || step.Kind != types.ProtocolStepKindA2A || step.Source != types.ProtocolSourceA2A {
		t.Fatalf("run=%#v step=%#v", run, step)
	}
}
