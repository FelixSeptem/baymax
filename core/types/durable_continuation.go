package types

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	NativeContinuationVersion           = "native_continuation.v1"
	NativeContinuationMaxEnvelopeBytes  = 64 * 1024
	NativeContinuationMaxReferenceBytes = 256
	NativeContinuationMaxDigestBytes    = 128
	NativeContinuationMaxReferences     = 256
	NativeContinuationMaxMaterialBytes  = 1024 * 1024
	NativeContinuationMaxEntries        = 1024
)

var (
	ErrCheckpointMalformed        = errors.New("checkpoint malformed")
	ErrCheckpointSchema           = errors.New("checkpoint schema unsupported")
	ErrCheckpointCorrupt          = errors.New("checkpoint corrupt")
	ErrCheckpointForeign          = errors.New("checkpoint foreign")
	ErrCheckpointExpired          = errors.New("checkpoint expired")
	ErrCheckpointNotFound         = errors.New("checkpoint not found")
	ErrCheckpointUnavailable      = errors.New("checkpoint store unavailable")
	ErrCheckpointConflict         = errors.New("checkpoint conflict")
	ErrCheckpointInProgress       = errors.New("checkpoint resume in progress")
	ErrCheckpointRecoveryRequired = errors.New("checkpoint recovery required")
	ErrCheckpointTerminal         = errors.New("checkpoint terminal")
	ErrCheckpointUnsupported      = errors.New("checkpoint unsupported")
)

type PauseReason string

const (
	PauseReasonInputRequired    PauseReason = "input_required"
	PauseReasonApprovalRequired PauseReason = "approval_required"
	PauseReasonBudgetExhausted  PauseReason = "budget_exhausted"
)

type NativeCheckpointRef struct {
	SchemaVersion           string      `json:"schema_version"`
	CheckpointID            string      `json:"checkpoint_id"`
	Digest                  string      `json:"digest"`
	SourceComponent         string      `json:"source_component"`
	OwnerNamespace          string      `json:"owner_namespace"`
	RunID                   string      `json:"run_id"`
	SessionID               string      `json:"session_id"`
	ParentRunID             string      `json:"parent_run_id,omitempty"`
	HostCorrelation         string      `json:"host_correlation,omitempty"`
	PauseReason             PauseReason `json:"pause_reason"`
	SourcePosition          string      `json:"source_position"`
	PredecessorCheckpointID string      `json:"predecessor_checkpoint_id,omitempty"`
	ResumeAttemptID         string      `json:"resume_attempt_id,omitempty"`
	AuthorityDigest         string      `json:"authority_digest,omitempty"`
	PlanDigest              string      `json:"plan_digest,omitempty"`
	WriteScopeDigest        string      `json:"write_scope_digest,omitempty"`
	OccurrenceID            string      `json:"occurrence_id,omitempty"`
	EffectNamespace         string      `json:"effect_namespace,omitempty"`
	CreatedAt               time.Time   `json:"created_at"`
	ExpiresAt               time.Time   `json:"expires_at"`
}

type DurableCheckpointRef = NativeCheckpointRef

func (r NativeCheckpointRef) Validate() error {
	if r.SchemaVersion != NativeContinuationVersion {
		if r.SchemaVersion == "" {
			return fmt.Errorf("schema_version: %w", ErrCheckpointMalformed)
		}
		return fmt.Errorf("schema_version: %w", ErrCheckpointSchema)
	}
	for name, value := range map[string]string{"checkpoint_id": r.CheckpointID, "source_component": r.SourceComponent, "owner_namespace": r.OwnerNamespace, "run_id": r.RunID, "session_id": r.SessionID, "source_position": r.SourcePosition} {
		if err := nativeString(name, value, NativeContinuationMaxReferenceBytes, true); err != nil {
			return err
		}
	}
	for name, value := range map[string]string{"parent_run_id": r.ParentRunID, "host_correlation": r.HostCorrelation, "predecessor_checkpoint_id": r.PredecessorCheckpointID, "resume_attempt_id": r.ResumeAttemptID, "occurrence_id": r.OccurrenceID, "effect_namespace": r.EffectNamespace} {
		if value != "" {
			if err := nativeString(name, value, NativeContinuationMaxReferenceBytes, true); err != nil {
				return err
			}
		}
	}
	for name, value := range map[string]string{"digest": r.Digest, "authority_digest": r.AuthorityDigest, "plan_digest": r.PlanDigest, "write_scope_digest": r.WriteScopeDigest} {
		if value != "" {
			if err := nativeString(name, value, NativeContinuationMaxDigestBytes, true); err != nil {
				return err
			}
		}
	}
	if r.Digest == "" {
		return fmt.Errorf("digest required: %w", ErrCheckpointMalformed)
	}
	if r.PauseReason != PauseReasonInputRequired && r.PauseReason != PauseReasonApprovalRequired && r.PauseReason != PauseReasonBudgetExhausted {
		return fmt.Errorf("pause_reason: %w", ErrCheckpointUnsupported)
	}
	if r.CreatedAt.IsZero() || !r.ExpiresAt.After(r.CreatedAt) || r.ExpiresAt.Sub(r.CreatedAt) > 24*time.Hour {
		return fmt.Errorf("invalid checkpoint expiry: %w", ErrCheckpointMalformed)
	}
	return nil
}

type DurableBudget struct {
	Revision     string    `json:"revision,omitempty"`
	Limit        int64     `json:"limit,omitempty"`
	Consumed     int64     `json:"consumed,omitempty"`
	ToolLimit    int64     `json:"tool_limit,omitempty"`
	ToolConsumed int64     `json:"tool_consumed,omitempty"`
	Deadline     time.Time `json:"deadline,omitempty"`
}

type PersistedRunCheckpoint struct {
	Ref NativeCheckpointRef `json:"ref"`
	// Checkpoint is an optional compatibility projection for legacy host
	// admission. Native ref and continuation material remain authoritative.
	Checkpoint         RunCheckpoint   `json:"checkpoint,omitempty"`
	Generation         uint64          `json:"generation"`
	Material           json.RawMessage `json:"material"`
	MaterialDigest     string          `json:"material_digest"`
	DependencyProfile  string          `json:"dependency_profile"`
	DependencyRevision string          `json:"dependency_revision"`
	Mode               string          `json:"mode"`
	Iteration          int             `json:"iteration"`
	ToolResultRefs     []string        `json:"tool_result_refs,omitempty"`
	CompletedEntries   int             `json:"completed_entries"`
	PendingEntries     int             `json:"pending_entries"`
	Budget             DurableBudget   `json:"budget,omitempty"`
	Deadline           time.Time       `json:"deadline,omitempty"`
}

func SealRunCheckpoint(p PersistedRunCheckpoint) (PersistedRunCheckpoint, error) {
	if p.Ref.Digest == "" {
		p.Ref.Digest = "pending"
	}
	if err := p.validateShape(); err != nil {
		return PersistedRunCheckpoint{}, err
	}
	p.Material = append(json.RawMessage(nil), p.Material...)
	p.ToolResultRefs = append([]string(nil), p.ToolResultRefs...)
	p.MaterialDigest = digestBytes(p.Material)
	p.Ref.Digest = p.checkpointDigest()
	if err := p.Validate(); err != nil {
		return PersistedRunCheckpoint{}, err
	}
	return p, nil
}
func (p PersistedRunCheckpoint) Detached() PersistedRunCheckpoint {
	p.Material = append(json.RawMessage(nil), p.Material...)
	p.ToolResultRefs = append([]string(nil), p.ToolResultRefs...)
	p.Checkpoint.ToolResultRefs = append([]string(nil), p.Checkpoint.ToolResultRefs...)
	if p.Checkpoint.PendingAction != nil {
		a := *p.Checkpoint.PendingAction
		p.Checkpoint.PendingAction = &a
	}
	return p
}
func (p PersistedRunCheckpoint) Validate() error {
	if err := p.validateShape(); err != nil {
		return err
	}
	if p.MaterialDigest != digestBytes(p.Material) || p.Ref.Digest != p.checkpointDigest() {
		return ErrCheckpointCorrupt
	}
	envelope := p
	envelope.Material = nil
	b, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("encode envelope: %w", ErrCheckpointMalformed)
	}
	if len(b) > NativeContinuationMaxEnvelopeBytes {
		return fmt.Errorf("envelope too large: %w", ErrCheckpointMalformed)
	}
	return nil
}
func (p PersistedRunCheckpoint) validateShape() error {
	if err := p.Ref.Validate(); err != nil {
		return err
	}
	if p.Generation == 0 {
		return fmt.Errorf("generation required: %w", ErrCheckpointMalformed)
	}
	if len(p.Material) == 0 || len(p.Material) > NativeContinuationMaxMaterialBytes || !json.Valid(p.Material) {
		return fmt.Errorf("material bound or encoding: %w", ErrCheckpointMalformed)
	}
	if p.Iteration < 0 || p.CompletedEntries < 0 || p.PendingEntries < 0 || p.CompletedEntries+p.PendingEntries > NativeContinuationMaxEntries {
		return fmt.Errorf("entry bounds: %w", ErrCheckpointMalformed)
	}
	if len(p.ToolResultRefs) > NativeContinuationMaxReferences {
		return fmt.Errorf("reference count: %w", ErrCheckpointMalformed)
	}
	for _, v := range p.ToolResultRefs {
		if err := nativeString("tool_result_ref", v, NativeContinuationMaxReferenceBytes, true); err != nil {
			return err
		}
	}
	for n, v := range map[string]string{"dependency_profile": p.DependencyProfile, "dependency_revision": p.DependencyRevision, "mode": p.Mode} {
		if err := nativeString(n, v, NativeContinuationMaxReferenceBytes, true); err != nil {
			return err
		}
	}
	if p.Mode != "run" && p.Mode != "stream" {
		return fmt.Errorf("mode: %w", ErrCheckpointMalformed)
	}
	if p.Budget.Consumed < 0 || p.Budget.Limit < 0 || (p.Budget.Limit > 0 && p.Budget.Consumed > p.Budget.Limit) {
		return fmt.Errorf("budget: %w", ErrCheckpointMalformed)
	}
	if p.Budget.ToolConsumed < 0 || p.Budget.ToolLimit < 0 || (p.Budget.ToolLimit > 0 && p.Budget.ToolConsumed > p.Budget.ToolLimit) {
		return fmt.Errorf("tool budget: %w", ErrCheckpointMalformed)
	}
	if !p.Deadline.IsZero() && p.Ref.ExpiresAt.After(p.Deadline) {
		return fmt.Errorf("expiry after deadline: %w", ErrCheckpointMalformed)
	}
	return nil
}
func (p PersistedRunCheckpoint) checkpointDigest() string {
	p.Ref.Digest = ""
	p.MaterialDigest = digestBytes(p.Material)
	b, _ := json.Marshal(p)
	return digestBytes(b)
}

type ResumeRequest struct {
	Checkpoint         NativeCheckpointRef   `json:"checkpoint"`
	Decision           DynamicActionDecision `json:"decision,omitempty"`
	Mode               string                `json:"mode"`
	DependencyProfile  string                `json:"dependency_profile"`
	DependencyRevision string                `json:"dependency_revision"`
	AuthorityDigest    string                `json:"authority_digest,omitempty"`
	PlanDigest         string                `json:"plan_digest,omitempty"`
	WriteScopeDigest   string                `json:"write_scope_digest,omitempty"`
	Generation         uint64                `json:"generation"`
	IdempotencyKey     string                `json:"idempotency_key"`
	Budget             DurableBudget         `json:"budget,omitempty"`
	ProposedBudget     *DurableBudget        `json:"proposed_budget,omitempty"`
}

func (r ResumeRequest) Validate() error {
	if err := r.Checkpoint.Validate(); err != nil {
		return err
	}
	for n, v := range map[string]string{"mode": r.Mode, "dependency_profile": r.DependencyProfile, "dependency_revision": r.DependencyRevision, "idempotency_key": r.IdempotencyKey} {
		if err := nativeString(n, v, NativeContinuationMaxReferenceBytes, true); err != nil {
			return err
		}
	}
	if r.Mode != "run" && r.Mode != "stream" {
		return ErrCheckpointMalformed
	}
	if r.Generation == 0 {
		return ErrCheckpointMalformed
	}
	for name, value := range map[string]string{"decision_token": r.Decision.Token, "decision_run_id": r.Decision.RunID, "decision_session_id": r.Decision.SessionID, "decision_checkpoint_id": r.Decision.CheckpointID, "decision_checkpoint_version": r.Decision.CheckpointVersion, "decision_idempotency_key": r.Decision.IdempotencyKey, "decision_reference": r.Decision.Reference} {
		if value != "" {
			if err := nativeString(name, value, NativeContinuationMaxReferenceBytes, true); err != nil {
				return err
			}
		}
	}
	if r.Decision.CheckpointDigest != "" {
		if err := nativeString("decision_digest", r.Decision.CheckpointDigest, NativeContinuationMaxDigestBytes, true); err != nil {
			return err
		}
	}
	for name, value := range map[string]string{"authority_digest": r.AuthorityDigest, "plan_digest": r.PlanDigest, "write_scope_digest": r.WriteScopeDigest} {
		if value != "" {
			if err := nativeString(name, value, NativeContinuationMaxDigestBytes, true); err != nil {
				return err
			}
		}
	}
	if r.Decision.Decision != "" {
		if err := r.Decision.Validate(); err != nil {
			return fmt.Errorf("decision: %w", ErrCheckpointMalformed)
		}
		d := r.Decision.Normalized()
		if d.IdempotencyKey != strings.TrimSpace(r.IdempotencyKey) || d.RunID != r.Checkpoint.RunID || d.SessionID != r.Checkpoint.SessionID || d.CheckpointID != r.Checkpoint.CheckpointID || d.CheckpointVersion != r.Checkpoint.SchemaVersion || d.CheckpointDigest != r.Checkpoint.Digest {
			return ErrCheckpointConflict
		}
	}
	if r.Budget.Limit < 0 || r.Budget.Consumed < 0 || r.Budget.ToolLimit < 0 || r.Budget.ToolConsumed < 0 || r.Budget.Limit > 0 && r.Budget.Consumed > r.Budget.Limit || r.Budget.ToolLimit > 0 && r.Budget.ToolConsumed > r.Budget.ToolLimit {
		return ErrCheckpointMalformed
	}
	if r.Budget.Revision != "" {
		if err := nativeString("budget_revision", r.Budget.Revision, NativeContinuationMaxReferenceBytes, true); err != nil {
			return err
		}
	}
	if r.ProposedBudget != nil {
		proposal := r
		proposal.Budget = *r.ProposedBudget
		proposal.ProposedBudget = nil
		if err := proposal.Validate(); err != nil {
			return err
		}
	}
	return nil
}
func (r ResumeRequest) IdentityDigest() string { b, _ := json.Marshal(r); return digestBytes(b) }

type ResumeClaimState string

const (
	ResumeClaimClaimed   ResumeClaimState = "claimed"
	ResumeClaimRunning   ResumeClaimState = "running"
	ResumeClaimCompleted ResumeClaimState = "completed"
	ResumeClaimCanceled  ResumeClaimState = "canceled"
)

type ResumeClaim struct {
	Request       ResumeRequest    `json:"request"`
	AttemptID     string           `json:"attempt_id"`
	Generation    uint64           `json:"generation"`
	State         ResumeClaimState `json:"state"`
	Duplicate     bool             `json:"duplicate,omitempty"`
	Result        *ResumeResult    `json:"result,omitempty"`
	RetainedUntil time.Time        `json:"retained_until,omitempty"`
}
type ResumeResult struct {
	State      RunState                `json:"state"`
	Projection json.RawMessage         `json:"projection,omitempty"`
	Successor  *PersistedRunCheckpoint `json:"successor,omitempty"`
	ErrorCode  string                  `json:"error_code,omitempty"`
}
type CheckpointControlAction string

const (
	CheckpointControlCancel   CheckpointControlAction = "cancel"
	CheckpointControlTerminal CheckpointControlAction = "terminal"
)

type CheckpointControl struct {
	Checkpoint NativeCheckpointRef     `json:"checkpoint"`
	Generation uint64                  `json:"generation"`
	Action     CheckpointControlAction `json:"action"`
	Reason     string                  `json:"reason,omitempty"`
}

type DurableRunCheckpointStore interface {
	Save(context.Context, PersistedRunCheckpoint) error
	Load(context.Context, NativeCheckpointRef) (PersistedRunCheckpoint, error)
	ReadResumeClaim(context.Context, NativeCheckpointRef) (ResumeClaim, error)
	ClaimResume(context.Context, ResumeRequest) (ResumeClaim, error)
	MarkRunning(context.Context, ResumeClaim) (ResumeClaim, error)
	CommitResumeResult(context.Context, ResumeClaim, ResumeResult) (ResumeClaim, error)
	Control(context.Context, CheckpointControl) (ResumeClaim, error)
	CrossProcessRestore() bool
}

type ContinuationLimits struct {
	EnvelopeBytes int
	MaterialBytes int
	References    int
	Entries       int
}

func (l ContinuationLimits) Normalized() (ContinuationLimits, error) {
	d := ContinuationLimits{NativeContinuationMaxEnvelopeBytes, NativeContinuationMaxMaterialBytes, NativeContinuationMaxReferences, NativeContinuationMaxEntries}
	v := []*int{&l.EnvelopeBytes, &l.MaterialBytes, &l.References, &l.Entries}
	dv := []int{d.EnvelopeBytes, d.MaterialBytes, d.References, d.Entries}
	for i, p := range v {
		if *p < 0 || *p > dv[i] {
			return ContinuationLimits{}, ErrCheckpointMalformed
		}
		if *p == 0 {
			*p = dv[i]
		}
	}
	return l, nil
}

func nativeString(name, value string, max int, required bool) error {
	if value == "" {
		if required {
			return fmt.Errorf("%s required: %w", name, ErrCheckpointMalformed)
		}
		return nil
	}
	if len(value) > max {
		return fmt.Errorf("%s too long: %w", name, ErrCheckpointMalformed)
	}
	if !utf8.ValidString(value) || strings.TrimSpace(value) != value || strings.IndexFunc(value, unicode.IsControl) >= 0 || strings.IndexFunc(value, unicode.IsSpace) >= 0 {
		return fmt.Errorf("%s invalid: %w", name, ErrCheckpointMalformed)
	}
	return nil
}
func digestBytes(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
