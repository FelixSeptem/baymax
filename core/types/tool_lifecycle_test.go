package types

import "testing"

func TestToolLifecycleProjectionNormalizesSuccessfulStages(t *testing.T) {
	projection, err := NewToolLifecycleProjection(ToolLifecycleInput{
		RunID: "run-1", StepID: "step-1", CallID: "call-1", ToolName: "local.echo", InputIndex: 2,
		Stages: []ToolLifecycleStageInput{
			{Stage: ToolLifecycleStagePrepare, Outcome: ToolLifecycleOutcomeSucceeded},
			{Stage: ToolLifecycleStageValidate, Outcome: ToolLifecycleOutcomeSucceeded},
			{Stage: ToolLifecycleStageAuthorize, Outcome: ToolLifecycleOutcomeSkipped, ReasonCode: "not_applicable"},
			{Stage: ToolLifecycleStageExecute, Outcome: ToolLifecycleOutcomeSucceeded, Started: true},
			{Stage: ToolLifecycleStageFinalize, Outcome: ToolLifecycleOutcomeSucceeded},
		},
	})
	if err != nil {
		t.Fatalf("NewToolLifecycleProjection() error = %v", err)
	}
	if got := projection.StageNames(); len(got) != 5 || got[0] != ToolLifecycleStagePrepare || got[4] != ToolLifecycleStageFinalize {
		t.Fatalf("stage order = %#v", got)
	}
	if projection.InputIndex != 2 || projection.CallID != "call-1" || !projection.Finalized {
		t.Fatalf("projection correlation/finalized = %#v", projection)
	}
}

func TestToolLifecycleProjectionRejectsInvalidStageOrderAndCorrelation(t *testing.T) {
	_, err := NewToolLifecycleProjection(ToolLifecycleInput{
		CallID: "", ToolName: "local.echo",
		Stages: []ToolLifecycleStageInput{{Stage: ToolLifecycleStageExecute, Outcome: ToolLifecycleOutcomeSucceeded}},
	})
	if err == nil {
		t.Fatal("expected missing call id error")
	}
	_, err = NewToolLifecycleProjection(ToolLifecycleInput{
		CallID: "call-1", ToolName: "local.echo",
		Stages: []ToolLifecycleStageInput{
			{Stage: ToolLifecycleStageValidate, Outcome: ToolLifecycleOutcomeSucceeded},
			{Stage: ToolLifecycleStagePrepare, Outcome: ToolLifecycleOutcomeSucceeded},
		},
	})
	if err == nil {
		t.Fatal("expected stage order error")
	}
}

func TestToolLifecycleProjectionClassifiesFailureAndFinalizeIdempotency(t *testing.T) {
	projection, err := NewToolLifecycleProjection(ToolLifecycleInput{
		CallID: "call-1", ToolName: "local.exec", FailureOrigin: ToolFailureOriginPanic,
		Stages: []ToolLifecycleStageInput{
			{Stage: ToolLifecycleStagePrepare, Outcome: ToolLifecycleOutcomeSucceeded},
			{Stage: ToolLifecycleStageValidate, Outcome: ToolLifecycleOutcomeSucceeded},
			{Stage: ToolLifecycleStageAuthorize, Outcome: ToolLifecycleOutcomeSucceeded},
			{Stage: ToolLifecycleStageExecute, Outcome: ToolLifecycleOutcomeFailed, Started: true},
			{Stage: ToolLifecycleStageFinalize, Outcome: ToolLifecycleOutcomeSucceeded},
		},
	})
	if err != nil {
		t.Fatalf("NewToolLifecycleProjection() error = %v", err)
	}
	if projection.FailureOrigin != ToolFailureOriginPanic || !projection.ExecutionStarted || !projection.Finalized {
		t.Fatalf("failure projection = %#v", projection)
	}
	if !projection.AcceptFinalize() || projection.AcceptFinalize() {
		t.Fatal("finalize should be idempotent")
	}
}

func TestToolLifecycleProjectionPreservesAttemptCount(t *testing.T) {
	projection, err := NewToolLifecycleProjection(ToolLifecycleInput{
		CallID: "call-1", ToolName: "local.echo", InputIndex: 2, AttemptCount: 3,
		Stages: []ToolLifecycleStageInput{
			{Stage: ToolLifecycleStagePrepare, Outcome: ToolLifecycleOutcomeSucceeded},
			{Stage: ToolLifecycleStageValidate, Outcome: ToolLifecycleOutcomeSucceeded},
			{Stage: ToolLifecycleStageAuthorize, Outcome: ToolLifecycleOutcomeNotApplicable},
			{Stage: ToolLifecycleStageExecute, Outcome: ToolLifecycleOutcomeFailed, Started: true},
			{Stage: ToolLifecycleStageFinalize, Outcome: ToolLifecycleOutcomeSucceeded},
		},
	})
	if err != nil {
		t.Fatalf("NewToolLifecycleProjection() error = %v", err)
	}
	if projection.AttemptCount != 3 {
		t.Fatalf("AttemptCount = %d, want 3", projection.AttemptCount)
	}
}
