package scheduler

import (
	"fmt"

	"github.com/FelixSeptem/baymax/core/types"
)

func cloneWorkspaceProvenance(in *types.WorkspaceProvenance) *types.WorkspaceProvenance {
	if in == nil {
		return nil
	}
	copy := *in
	return &copy
}

func workspaceProvenanceEqual(a, b *types.WorkspaceProvenance) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func validateTaskWorkspaceBinding(task Task, index int) error {
	if wp := task.WorkspaceProvenance; wp != nil {
		if task.RunID != "" && wp.ProducedByRunID != task.RunID {
			return fmt.Errorf("%w: tasks[%d].workspace_provenance run association mismatch", ErrSnapshotCorrupt, index)
		}
		if task.StepID != "" && wp.ProducedByStepID != task.StepID {
			return fmt.Errorf("%w: tasks[%d].workspace_provenance step association mismatch", ErrSnapshotCorrupt, index)
		}
	}
	return nil
}

func validateAttemptWorkspaceBinding(task, attempt *types.WorkspaceProvenance, taskIndex, attemptIndex int) error {
	if attempt == nil {
		return nil
	}
	if err := attempt.Validate(); err != nil {
		return fmt.Errorf("%w: tasks[%d].attempts[%d].workspace_provenance invalid: %v", ErrSnapshotCorrupt, taskIndex, attemptIndex, err)
	}
	if !workspaceProvenanceEqual(task, attempt) {
		return fmt.Errorf("%w: tasks[%d].attempts[%d].workspace_provenance mismatches task", ErrSnapshotCorrupt, taskIndex, attemptIndex)
	}
	return nil
}

func validateTerminalWorkspaceBinding(record TaskRecord, commit TerminalCommit, index int, attemptID string) error {
	attempt, ok := record.attemptByID(attemptID)
	if !ok {
		return fmt.Errorf("%w: terminal_commits[%d] references unknown attempt %q", ErrSnapshotCorrupt, index, attemptID)
	}
	if commit.WorkspaceProvenance != nil && !workspaceProvenanceEqual(record.Task.WorkspaceProvenance, commit.WorkspaceProvenance) {
		return fmt.Errorf("%w: terminal_commits[%d].workspace_provenance mismatches task", ErrSnapshotCorrupt, index)
	}
	if commit.WorkspaceProvenance != nil && !workspaceProvenanceEqual(attempt.WorkspaceProvenance, commit.WorkspaceProvenance) {
		return fmt.Errorf("%w: terminal_commits[%d].workspace_provenance mismatches attempt", ErrSnapshotCorrupt, index)
	}
	return nil
}
