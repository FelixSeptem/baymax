package types

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"
)

// DynamicActionResumeProtocolVersion identifies the bounded, source-owned
// checkpoint contract. It intentionally does not version the application
// action payload represented by Token.
const DynamicActionResumeProtocolVersion = "dynamic_action_resume.v1"

const (
	DynamicActionMaxTokenBytes             = 256
	DynamicActionMaxKindBytes              = 128
	DynamicActionMaxCorrelationBytes       = 256
	DynamicActionMaxDigestBytes            = 128
	DynamicActionMaxCheckpointIDBytes      = 256
	DynamicActionMaxIdempotencyKeyBytes    = 256
	DynamicActionMaxDecisionReferenceBytes = 256
)

// DynamicActionReference is an opaque, bounded reference produced at the
// tool-result boundary. Baymax validates its shape and correlation but never
// interprets Token or stores the application action body.
type DynamicActionReference struct {
	Token     string `json:"token"`
	Kind      string `json:"kind"`
	Resumable bool   `json:"resumable"`
	RunID     string `json:"run_id"`
	SessionID string `json:"session_id"`
	Iteration int    `json:"iteration"`
	CallID    string `json:"call_id"`
	Source    string `json:"source"`
	Digest    string `json:"digest"`
}

func (r DynamicActionReference) Validate() error {
	// Keep these checks explicit rather than iterating over a map: each field
	// has its own bound, and map iteration would both obscure that distinction
	// and make the first reported validation failure nondeterministic.
	checks := []struct {
		name  string
		value string
		limit int
	}{
		{name: "token", value: r.Token, limit: DynamicActionMaxTokenBytes},
		{name: "kind", value: r.Kind, limit: DynamicActionMaxKindBytes},
		{name: "run_id", value: r.RunID, limit: DynamicActionMaxCorrelationBytes},
		{name: "session_id", value: r.SessionID, limit: DynamicActionMaxCorrelationBytes},
		{name: "call_id", value: r.CallID, limit: DynamicActionMaxCorrelationBytes},
		{name: "source", value: r.Source, limit: DynamicActionMaxCorrelationBytes},
		{name: "digest", value: r.Digest, limit: DynamicActionMaxDigestBytes},
	}
	for _, check := range checks {
		if err := validateDynamicActionString(check.value, check.name, check.limit); err != nil {
			return err
		}
	}
	if r.Iteration < 0 {
		return fmt.Errorf("dynamic action: iteration must be >= 0")
	}
	return nil
}

func (r DynamicActionReference) Normalized() DynamicActionReference {
	r.Token = strings.TrimSpace(r.Token)
	r.Kind = strings.TrimSpace(r.Kind)
	r.RunID = strings.TrimSpace(r.RunID)
	r.SessionID = strings.TrimSpace(r.SessionID)
	r.CallID = strings.TrimSpace(r.CallID)
	r.Source = strings.TrimSpace(r.Source)
	r.Digest = strings.TrimSpace(r.Digest)
	return r
}

type DynamicActionDecisionKind string

const (
	DynamicActionDecisionConfirm DynamicActionDecisionKind = "confirm"
	DynamicActionDecisionDeny    DynamicActionDecisionKind = "deny"
	DynamicActionDecisionTimeout DynamicActionDecisionKind = "timeout"
	// Approval/rejection aliases make the transport contract convenient for
	// adapters without introducing a second decision vocabulary.
	DynamicActionDecisionApprove DynamicActionDecisionKind = DynamicActionDecisionConfirm
	DynamicActionDecisionReject  DynamicActionDecisionKind = DynamicActionDecisionDeny
)

// DynamicActionDecision is the source-owned decision reference admitted for a
// paused checkpoint. Reference is opaque to Baymax and may identify a
// business confirmation record without carrying its payload.
type DynamicActionDecision struct {
	Decision          DynamicActionDecisionKind `json:"decision"`
	Token             string                    `json:"token"`
	RunID             string                    `json:"run_id"`
	SessionID         string                    `json:"session_id"`
	CheckpointID      string                    `json:"checkpoint_id"`
	CheckpointVersion string                    `json:"checkpoint_version"`
	CheckpointDigest  string                    `json:"checkpoint_digest"`
	IdempotencyKey    string                    `json:"idempotency_key"`
	Reference         string                    `json:"reference,omitempty"`
}

func (d DynamicActionDecision) Validate() error {
	if d.Decision != DynamicActionDecisionConfirm && d.Decision != DynamicActionDecisionDeny && d.Decision != DynamicActionDecisionTimeout {
		return fmt.Errorf("dynamic action: unsupported decision %q", d.Decision)
	}
	for name, value := range map[string]string{
		"token": d.Token, "run_id": d.RunID, "session_id": d.SessionID,
		"checkpoint_id": d.CheckpointID, "checkpoint_version": d.CheckpointVersion,
		"checkpoint_digest": d.CheckpointDigest, "idempotency_key": d.IdempotencyKey,
	} {
		limit := DynamicActionMaxCorrelationBytes
		if name == "checkpoint_digest" {
			limit = DynamicActionMaxDigestBytes
		}
		if name == "token" {
			limit = DynamicActionMaxTokenBytes
		}
		if err := validateDynamicActionString(value, name, limit); err != nil {
			return err
		}
	}
	if strings.TrimSpace(d.Reference) != "" {
		if err := validateDynamicActionString(d.Reference, "reference", DynamicActionMaxDecisionReferenceBytes); err != nil {
			return err
		}
	}
	return nil
}

// RunCheckpoint contains only Runner recovery metadata and opaque references.
// ToolResultRefs are correlation references, not serialized tool payloads.
type RunCheckpoint struct {
	Version        string                  `json:"version"`
	CheckpointID   string                  `json:"checkpoint_id"`
	RunID          string                  `json:"run_id"`
	SessionID      string                  `json:"session_id"`
	State          RunState                `json:"state"`
	Mode           string                  `json:"mode,omitempty"`
	Iteration      int                     `json:"iteration"`
	ToolResultRefs []string                `json:"tool_result_refs,omitempty"`
	PendingAction  *DynamicActionReference `json:"pending_action,omitempty"`
	Digest         string                  `json:"digest"`
	CreatedAt      time.Time               `json:"created_at,omitempty"`
	ResumeAttempt  int                     `json:"resume_attempt,omitempty"`
}

// DynamicActionCheckpoint is retained as a semantic alias for adapters that
// name the checkpoint after the action boundary.
type DynamicActionCheckpoint = RunCheckpoint

func (c RunCheckpoint) Validate() error {
	if strings.TrimSpace(c.Version) == "" {
		return fmt.Errorf("dynamic action checkpoint: version is required")
	}
	if err := validateDynamicActionString(c.CheckpointID, "checkpoint_id", DynamicActionMaxCheckpointIDBytes); err != nil {
		return err
	}
	if err := validateDynamicActionString(c.RunID, "run_id", DynamicActionMaxCorrelationBytes); err != nil {
		return err
	}
	if err := validateDynamicActionString(c.SessionID, "session_id", DynamicActionMaxCorrelationBytes); err != nil {
		return err
	}
	if c.State != RunStateInputRequired {
		return fmt.Errorf("dynamic action checkpoint: state must be input_required, got %q", c.State)
	}
	if c.Iteration < 0 {
		return fmt.Errorf("dynamic action checkpoint: iteration must be >= 0")
	}
	if err := validateDynamicActionString(c.Digest, "digest", DynamicActionMaxDigestBytes); err != nil {
		return err
	}
	if c.PendingAction == nil {
		return fmt.Errorf("dynamic action checkpoint: pending_action is required")
	}
	if err := c.PendingAction.Validate(); err != nil {
		return fmt.Errorf("validate pending action: %w", err)
	}
	if c.PendingAction.RunID != strings.TrimSpace(c.RunID) || c.PendingAction.SessionID != strings.TrimSpace(c.SessionID) || c.PendingAction.Iteration != c.Iteration {
		return fmt.Errorf("dynamic action checkpoint: pending action correlation mismatch")
	}
	if c.PendingAction.Digest != strings.TrimSpace(c.Digest) {
		return fmt.Errorf("dynamic action checkpoint: pending action digest mismatch")
	}
	if c.ResumeAttempt < 0 {
		return fmt.Errorf("dynamic action checkpoint: resume_attempt must be >= 0")
	}
	return nil
}

type RunResumeAdmissionStatus string

const (
	RunResumeAdmissionAccepted  RunResumeAdmissionStatus = "accepted"
	RunResumeAdmissionRejected  RunResumeAdmissionStatus = "rejected"
	RunResumeAdmissionDuplicate RunResumeAdmissionStatus = "duplicate"
)

// RunResumeAdmission is an idempotent, side-effect-free projection of a
// resume claim. The source Runner owns actual continuation and terminal state.
type RunResumeAdmission struct {
	Status         RunResumeAdmissionStatus  `json:"status"`
	RunID          string                    `json:"run_id"`
	SessionID      string                    `json:"session_id"`
	CheckpointID   string                    `json:"checkpoint_id"`
	IdempotencyKey string                    `json:"idempotency_key"`
	Decision       DynamicActionDecisionKind `json:"decision"`
	ReasonCode     string                    `json:"reason_code,omitempty"`
}

func (a RunResumeAdmission) Validate() error {
	if a.Status != RunResumeAdmissionAccepted && a.Status != RunResumeAdmissionRejected && a.Status != RunResumeAdmissionDuplicate {
		return fmt.Errorf("dynamic action admission: unsupported status %q", a.Status)
	}
	for name, value := range map[string]string{"run_id": a.RunID, "session_id": a.SessionID, "checkpoint_id": a.CheckpointID, "idempotency_key": a.IdempotencyKey} {
		limit := DynamicActionMaxCorrelationBytes
		if name == "checkpoint_id" {
			limit = DynamicActionMaxCheckpointIDBytes
		}
		if err := validateDynamicActionString(value, name, limit); err != nil {
			return err
		}
	}
	if a.Decision != DynamicActionDecisionConfirm && a.Decision != DynamicActionDecisionDeny && a.Decision != DynamicActionDecisionTimeout {
		return fmt.Errorf("dynamic action admission: unsupported decision %q", a.Decision)
	}
	return nil
}

func (a RunResumeAdmission) IsDuplicateOf(other RunResumeAdmission) bool {
	return a.Status == RunResumeAdmissionDuplicate && other.Status == RunResumeAdmissionAccepted &&
		a.RunID == other.RunID && a.SessionID == other.SessionID && a.CheckpointID == other.CheckpointID &&
		a.IdempotencyKey == other.IdempotencyKey && a.Decision == other.Decision
}

// NormalizeRunResumeAdmission validates an admission and makes duplicate
// status explicit without mutating source-owned state.
func NormalizeRunResumeAdmission(admission RunResumeAdmission) (RunResumeAdmission, error) {
	if err := admission.Validate(); err != nil {
		return RunResumeAdmission{}, err
	}
	admission.RunID = strings.TrimSpace(admission.RunID)
	admission.SessionID = strings.TrimSpace(admission.SessionID)
	admission.CheckpointID = strings.TrimSpace(admission.CheckpointID)
	admission.IdempotencyKey = strings.TrimSpace(admission.IdempotencyKey)
	return admission, nil
}

// RunCheckpointStore is an injectable owner of bounded Runner checkpoints.
// Implementations must make AdmitResume compare-and-set/idempotent.
type RunCheckpointStore interface {
	Save(context.Context, RunCheckpoint) error
	Load(context.Context, string, string) (RunCheckpoint, error)
	AdmitResume(context.Context, RunCheckpoint, DynamicActionDecision) (RunResumeAdmission, error)
}

func validateDynamicActionString(value, name string, maxBytes int) error {
	s := strings.TrimSpace(value)
	if s == "" {
		return fmt.Errorf("dynamic action: %s is required", name)
	}
	if len([]byte(s)) > maxBytes {
		return fmt.Errorf("dynamic action: %s exceeds %d bytes", name, maxBytes)
	}
	for _, r := range s {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return fmt.Errorf("dynamic action: %s contains whitespace or control character", name)
		}
	}
	return nil
}
