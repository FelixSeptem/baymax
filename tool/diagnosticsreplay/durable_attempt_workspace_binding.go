package diagnosticsreplay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

const DurableAttemptWorkspaceBindingFixtureV1 = "durable_attempt_workspace_binding.v1"

const (
	ReasonCodeDurableAttemptWorkspaceSchemaDrift       = "durable_attempt_workspace_schema_drift"
	ReasonCodeDurableAttemptWorkspaceBindingDrift      = "durable_attempt_workspace_binding_drift"
	ReasonCodeDurableAttemptWorkspaceAssociationDrift  = "durable_attempt_workspace_association_drift"
	ReasonCodeDurableAttemptWorkspaceIntegrityDrift    = "durable_attempt_workspace_integrity_drift"
	ReasonCodeDurableAttemptWorkspaceStaleAttemptDrift = "durable_attempt_workspace_stale_attempt_drift"
)

type DurableAttemptWorkspaceBindingFixture struct {
	Version string                               `json:"version"`
	Cases   []DurableAttemptWorkspaceBindingCase `json:"cases"`
}

type DurableAttemptWorkspaceBindingCase struct {
	Name            string                   `json:"name"`
	TaskID          string                   `json:"task_id"`
	AttemptID       string                   `json:"attempt_id"`
	Attempt         int                      `json:"attempt"`
	LeaseGeneration int                      `json:"lease_generation"`
	Workspace       *DurableWorkspaceBinding `json:"workspace,omitempty"`
	Retry           *DurableRetryBinding     `json:"retry,omitempty"`
	WorkspaceState  string                   `json:"workspace_state,omitempty"`
	TerminalResult  string                   `json:"terminal_result,omitempty"`
	Recovery        string                   `json:"recovery,omitempty"`
	Expected        string                   `json:"expected"`
	Observed        string                   `json:"observed,omitempty"`
}

type DurableWorkspaceBinding struct {
	WorkspaceID     string `json:"workspace_id"`
	ChangeSetID     string `json:"change_set_id"`
	BeforeIntegrity string `json:"before_integrity,omitempty"`
	AfterIntegrity  string `json:"after_integrity,omitempty"`
	CheckpointID    string `json:"checkpoint_id,omitempty"`
}

type DurableRetryBinding struct {
	Mode string `json:"mode"`
}

func ParseDurableAttemptWorkspaceBindingFixtureJSON(raw []byte) (DurableAttemptWorkspaceBindingFixture, error) {
	if len(raw) > 2<<20 {
		return DurableAttemptWorkspaceBindingFixture{}, schemaDurableWorkspaceError("fixture exceeds 2 MiB")
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var f DurableAttemptWorkspaceBindingFixture
	if err := dec.Decode(&f); err != nil {
		return f, schemaDurableWorkspaceError(err.Error())
	}
	if strings.TrimSpace(f.Version) != DurableAttemptWorkspaceBindingFixtureV1 {
		return f, schemaDurableWorkspaceError(fmt.Sprintf("unsupported fixture version %q", f.Version))
	}
	if len(f.Cases) == 0 || len(f.Cases) > 256 {
		return f, schemaDurableWorkspaceError("cases must contain 1..256 items")
	}
	for i := range f.Cases {
		if err := validateDurableWorkspaceCase(&f.Cases[i]); err != nil {
			return f, err
		}
	}
	return f, nil
}

func validateDurableWorkspaceCase(c *DurableAttemptWorkspaceBindingCase) error {
	c.Name, c.TaskID, c.AttemptID, c.Expected, c.Observed = strings.TrimSpace(c.Name), strings.TrimSpace(c.TaskID), strings.TrimSpace(c.AttemptID), strings.TrimSpace(c.Expected), strings.TrimSpace(c.Observed)
	if c.Name == "" || c.TaskID == "" || c.AttemptID == "" || c.Expected == "" {
		return schemaDurableWorkspaceError("name, task_id, attempt_id, and expected are required")
	}
	if len(c.Name) > 128 || len(c.TaskID) > 256 || len(c.AttemptID) > 256 {
		return schemaDurableWorkspaceError("identifier exceeds bound")
	}
	if c.Attempt <= 0 || c.LeaseGeneration < 0 {
		return schemaDurableWorkspaceError("attempt must be > 0 and lease_generation must be >= 0")
	}
	c.WorkspaceState = strings.ToLower(strings.TrimSpace(c.WorkspaceState))
	if c.WorkspaceState != "" {
		switch c.WorkspaceState {
		case "absent", "valid", "missing", "dirty", "conflict", "drift", "checkpoint_mismatch":
		default:
			return schemaDurableWorkspaceError(fmt.Sprintf("unsupported workspace_state %q", c.WorkspaceState))
		}
	}
	c.TerminalResult = strings.ToLower(strings.TrimSpace(c.TerminalResult))
	if c.TerminalResult != "" {
		switch c.TerminalResult {
		case "accepted", "stale", "conflict", "duplicate":
		default:
			return schemaDurableWorkspaceError(fmt.Sprintf("unsupported terminal_result %q", c.TerminalResult))
		}
	}
	c.Recovery = strings.ToLower(strings.TrimSpace(c.Recovery))
	if c.Recovery != "" {
		switch c.Recovery {
		case "none", "reconciled", "strict_reject", "compatible_downgrade":
		default:
			return schemaDurableWorkspaceError(fmt.Sprintf("unsupported recovery %q", c.Recovery))
		}
	}
	if c.Workspace != nil {
		w := c.Workspace
		if strings.TrimSpace(w.WorkspaceID) == "" || strings.TrimSpace(w.ChangeSetID) == "" {
			return &ValidationError{Code: ReasonCodeDurableAttemptWorkspaceAssociationDrift, Message: "workspace_id and change_set_id are required"}
		}
		for _, v := range []string{w.WorkspaceID, w.ChangeSetID, w.BeforeIntegrity, w.AfterIntegrity, w.CheckpointID} {
			if len(v) > 256 {
				return schemaDurableWorkspaceError("workspace reference exceeds bound")
			}
		}
	}
	if c.Retry != nil {
		c.Retry.Mode = strings.ToLower(strings.TrimSpace(c.Retry.Mode))
		if c.Retry.Mode != "reuse_after_integrity_validation" && c.Retry.Mode != "rebind_with_rollover" {
			return &ValidationError{Code: ReasonCodeDurableAttemptWorkspaceBindingDrift, Message: fmt.Sprintf("unsupported retry mode %q", c.Retry.Mode)}
		}
	}
	return nil
}

func schemaDurableWorkspaceError(msg string) *ValidationError {
	return &ValidationError{Code: ReasonCodeDurableAttemptWorkspaceSchemaDrift, Message: msg}
}
