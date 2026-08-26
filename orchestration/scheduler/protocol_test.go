package scheduler

import (
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
)

func TestProtocolStepFromSchedulerAttemptPreservesCorrelation(t *testing.T) {
	ref, err := ProtocolAttemptRef(Task{TaskID: "task-1", RunID: "run-1", WorkflowID: "workflow-1", TeamID: "team-1", StepID: "step-1", AgentID: "agent-1"}, Attempt{AttemptID: "attempt-1"}, "session-1")
	if err != nil {
		t.Fatalf("ProtocolAttemptRef() error = %v", err)
	}
	if ref.StepID != "attempt-1" || ref.RunID != "run-1" || ref.ParentStepID != "step-1" || ref.Kind != types.ProtocolStepKindScheduler || ref.Source != types.ProtocolSourceScheduler {
		t.Fatalf("ref = %#v", ref)
	}
}
