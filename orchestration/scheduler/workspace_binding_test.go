package scheduler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

func TestWorkspaceBindingRetainedAcrossClaimAndRejectsMismatch(t *testing.T) {
	ctx := context.Background()
	now := time.Unix(100, 0).UTC()
	workspace := &types.WorkspaceProvenance{WorkspaceID: "ws-1", ChangeSetID: "change-1", BeforeIntegrity: "before", AfterIntegrity: "after", ProducedByRunID: "run-1", ProducedByStepID: "step-1"}
	store := NewMemoryStore()
	if _, err := store.Enqueue(ctx, Task{TaskID: "task-1", RunID: "run-1", StepID: "step-1", WorkspaceProvenance: workspace}, now); err != nil {
		t.Fatal(err)
	}
	claimed, ok, err := store.Claim(ctx, "worker-1", now, time.Minute)
	if err != nil || !ok || claimed.Attempt.WorkspaceProvenance == nil || claimed.Attempt.WorkspaceProvenance.WorkspaceID != "ws-1" {
		t.Fatalf("claimed=%#v ok=%v err=%v", claimed, ok, err)
	}
	_, err = store.CommitTerminal(ctx, TerminalCommit{TaskID: "task-1", AttemptID: claimed.Attempt.AttemptID, Status: TaskStateSucceeded, WorkspaceProvenance: &types.WorkspaceProvenance{WorkspaceID: "ws-other", ChangeSetID: "change-1", BeforeIntegrity: "before", AfterIntegrity: "after", ProducedByRunID: "run-1", ProducedByStepID: "step-1"}})
	if !errors.Is(err, ErrWorkspaceBindingMismatch) {
		t.Fatalf("err=%v, want %v", err, ErrWorkspaceBindingMismatch)
	}
}
