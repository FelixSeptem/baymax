package types

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// ProtocolSource identifies the module that owns the source semantics.
type ProtocolSource string

const (
	ProtocolSourceRunner    ProtocolSource = "runner"
	ProtocolSourceWorkflow  ProtocolSource = "workflow"
	ProtocolSourceComposer  ProtocolSource = "composer"
	ProtocolSourceTeams     ProtocolSource = "teams"
	ProtocolSourceScheduler ProtocolSource = "scheduler"
	ProtocolSourceA2A       ProtocolSource = "a2a"
	ProtocolSourceRealtime  ProtocolSource = "realtime"
	ProtocolSourceTimeline  ProtocolSource = "timeline"
)

// RunState is the public lifecycle projection for a managed execution.
type RunState string

const (
	RunStateSubmitted     RunState = "submitted"
	RunStateWorking       RunState = "working"
	RunStateInputRequired RunState = "input_required"
	RunStateCompleted     RunState = "completed"
	RunStateFailed        RunState = "failed"
	RunStateCanceled      RunState = "canceled"
)

// ProtocolStepKind identifies a public execution step without replacing the
// richer source-module step kinds.
type ProtocolStepKind string

const (
	ProtocolStepKindModel     ProtocolStepKind = "model"
	ProtocolStepKindTool      ProtocolStepKind = "tool"
	ProtocolStepKindWorkflow  ProtocolStepKind = "workflow"
	ProtocolStepKindTeamTask  ProtocolStepKind = "team_task"
	ProtocolStepKindScheduler ProtocolStepKind = "scheduler_attempt"
	ProtocolStepKindA2A       ProtocolStepKind = "a2a_task"
	ProtocolStepKindHITL      ProtocolStepKind = "hitl"
)

// ProtocolEventKind identifies an observable lifecycle change.
type ProtocolEventKind string

const (
	ProtocolEventKindProgress ProtocolEventKind = "progress"
	ProtocolEventKindState    ProtocolEventKind = "state"
	ProtocolEventKindError    ProtocolEventKind = "error"
	ProtocolEventKindArtifact ProtocolEventKind = "artifact"
)

// ProtocolFailureBoundary classifies whether protocol projection may expose a
// failure as a recoverable Step outcome. It does not replace ErrorClass.
type ProtocolFailureBoundary string

const (
	ProtocolFailureTool                  ProtocolFailureBoundary = "tool"
	ProtocolFailureBusiness              ProtocolFailureBoundary = "business"
	ProtocolFailureConfiguration         ProtocolFailureBoundary = "configuration"
	ProtocolFailureSecurity              ProtocolFailureBoundary = "security"
	ProtocolFailurePermission            ProtocolFailureBoundary = "permission"
	ProtocolFailureValidation            ProtocolFailureBoundary = "validation"
	ProtocolFailureSnapshotCompatibility ProtocolFailureBoundary = "snapshot_compatibility"
	ProtocolFailureModuleBoundary        ProtocolFailureBoundary = "module_boundary"
)

const AgentRuntimeProtocolName = "agent_runtime_protocol"

const (
	ProtocolReasonCapabilityMissingRequired      = "protocol.capability.missing_required"
	ProtocolReasonCapabilityOptionalDowngraded   = "protocol.capability.optional_downgraded"
	ProtocolReasonCapabilityProfileMismatch      = "protocol.capability.profile_mismatch"
	ProtocolReasonActionUnsupported              = "protocol.action.unsupported"
	ProtocolReasonAdmissionIncompatibleOutcome   = "protocol.admission.incompatible_outcome"
	ProtocolReasonAdmissionPolicyUnknown         = "protocol.admission.policy_unknown"
	ProtocolReasonCheckpointLineageMissingParent = "checkpoint.lineage_missing_parent"
	ProtocolReasonCheckpointHistoryDisconnected  = "checkpoint.history_disconnected"
	ProtocolReasonCheckpointReplayConflict       = "checkpoint.replay_conflict"
	ProtocolReasonWorkspaceProvenanceMissing     = "workspace.provenance_missing"
	ProtocolReasonWorkspaceAssociationMismatch   = "workspace.association_mismatch"
	ProtocolReasonWorkspaceIntegrityDrift        = "workspace.integrity_drift"
)

// ProtocolCapabilityStrategy uses the same request semantics as adapter
// capability negotiation without making core/types depend on adapter packages.
type ProtocolCapabilityStrategy string

const (
	ProtocolCapabilityStrategyFailFast   ProtocolCapabilityStrategy = "fail_fast"
	ProtocolCapabilityStrategyBestEffort ProtocolCapabilityStrategy = "best_effort"
)

type ProtocolCapabilityDecision string

const (
	ProtocolCapabilityDecisionAccepted   ProtocolCapabilityDecision = "accepted"
	ProtocolCapabilityDecisionDowngraded ProtocolCapabilityDecision = "downgraded"
)

// ProtocolCapability is a stable, runtime-declared capability reference.
type ProtocolCapability struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

type ProtocolCapabilityRequest struct {
	ProfileVersion string                     `json:"profile_version,omitempty"`
	Required       []string                   `json:"required,omitempty"`
	Optional       []string                   `json:"optional,omitempty"`
	Strategy       ProtocolCapabilityStrategy `json:"strategy,omitempty"`
}

type ProtocolCapabilityNegotiation struct {
	ProfileVersion string                     `json:"profile_version"`
	Decision       ProtocolCapabilityDecision `json:"decision"`
	ReasonCode     string                     `json:"reason_code,omitempty"`
	Missing        []string                   `json:"missing,omitempty"`
}

type ProtocolAction string

const (
	ProtocolActionCancel ProtocolAction = "cancel"
	ProtocolActionResume ProtocolAction = "resume"
	ProtocolActionRetry  ProtocolAction = "retry"
)

// ProtocolDescriptor declares what a source Runtime projects under one
// effective profile. It is discovery metadata, not an authorization grant.
type ProtocolDescriptor struct {
	ProtocolName   string               `json:"protocol_name"`
	ProfileVersion string               `json:"profile_version"`
	RuntimeID      string               `json:"runtime_id"`
	Source         ProtocolSource       `json:"source,omitempty"`
	Capabilities   []ProtocolCapability `json:"capabilities,omitempty"`
	Actions        []ProtocolAction     `json:"actions,omitempty"`
}

func (d ProtocolDescriptor) Validate() error {
	if strings.TrimSpace(d.ProtocolName) != AgentRuntimeProtocolName {
		return protocolValidationError("protocol_name must be %q", AgentRuntimeProtocolName)
	}
	if strings.TrimSpace(d.ProfileVersion) == "" {
		return protocolValidationError("profile_version is required")
	}
	if strings.TrimSpace(d.RuntimeID) == "" {
		return protocolValidationError("runtime_id is required")
	}
	if d.Source != "" {
		if err := validateProtocolSource(d.Source); err != nil {
			return err
		}
	}
	seenCapabilities := make(map[string]struct{}, len(d.Capabilities))
	for _, capability := range d.Capabilities {
		name := normalizeProtocolName(capability.Name)
		if name == "" {
			return protocolValidationError("capability name is required")
		}
		if _, exists := seenCapabilities[name]; exists {
			return protocolValidationError("duplicate capability %q", name)
		}
		seenCapabilities[name] = struct{}{}
	}
	seenActions := make(map[ProtocolAction]struct{}, len(d.Actions))
	for _, action := range d.Actions {
		normalized := ProtocolAction(normalizeProtocolName(string(action)))
		if normalized == "" {
			return protocolValidationError("action is required")
		}
		if _, exists := seenActions[normalized]; exists {
			return protocolValidationError("duplicate action %q", normalized)
		}
		seenActions[normalized] = struct{}{}
	}
	return nil
}

func (d ProtocolDescriptor) SupportsAction(action ProtocolAction) bool {
	needle := ProtocolAction(normalizeProtocolName(string(action)))
	for _, candidate := range d.Actions {
		if ProtocolAction(normalizeProtocolName(string(candidate))) == needle {
			return true
		}
	}
	return false
}

func ValidateProtocolAction(descriptor ProtocolDescriptor, action ProtocolAction) error {
	if err := descriptor.Validate(); err != nil {
		return err
	}
	if !descriptor.SupportsAction(action) {
		return protocolValidationError("%s: action %q is unavailable", ProtocolReasonActionUnsupported, action)
	}
	return nil
}

func NegotiateProtocolCapabilities(descriptor ProtocolDescriptor, request ProtocolCapabilityRequest) (ProtocolCapabilityNegotiation, error) {
	if err := descriptor.Validate(); err != nil {
		return ProtocolCapabilityNegotiation{}, err
	}
	profile := strings.TrimSpace(request.ProfileVersion)
	if profile != "" && profile != strings.TrimSpace(descriptor.ProfileVersion) {
		return ProtocolCapabilityNegotiation{}, protocolValidationError("%s: requested %q, effective %q", ProtocolReasonCapabilityProfileMismatch, profile, descriptor.ProfileVersion)
	}
	strategy := request.Strategy
	if strategy == "" {
		strategy = ProtocolCapabilityStrategyFailFast
	}
	if strategy != ProtocolCapabilityStrategyFailFast && strategy != ProtocolCapabilityStrategyBestEffort {
		return ProtocolCapabilityNegotiation{}, protocolValidationError("unsupported capability strategy %q", strategy)
	}
	declared := make(map[string]struct{}, len(descriptor.Capabilities))
	for _, capability := range descriptor.Capabilities {
		declared[normalizeProtocolName(capability.Name)] = struct{}{}
	}
	missingRequired := missingProtocolCapabilities(declared, request.Required)
	missingOptional := missingProtocolCapabilities(declared, request.Optional)
	if strategy == ProtocolCapabilityStrategyFailFast && len(missingOptional) > 0 {
		missingRequired = appendUniqueProtocolCapabilities(missingRequired, missingOptional...)
		sort.Strings(missingRequired)
	}
	if len(missingRequired) > 0 {
		return ProtocolCapabilityNegotiation{}, protocolValidationError("%s: %s", ProtocolReasonCapabilityMissingRequired, strings.Join(missingRequired, ","))
	}
	if len(missingOptional) > 0 && strategy == ProtocolCapabilityStrategyBestEffort {
		return ProtocolCapabilityNegotiation{
			ProfileVersion: descriptor.ProfileVersion,
			Decision:       ProtocolCapabilityDecisionDowngraded,
			ReasonCode:     ProtocolReasonCapabilityOptionalDowngraded,
			Missing:        missingOptional,
		}, nil
	}
	return ProtocolCapabilityNegotiation{ProfileVersion: descriptor.ProfileVersion, Decision: ProtocolCapabilityDecisionAccepted}, nil
}

type ProtocolContextScope string

const (
	ProtocolContextScopeSession ProtocolContextScope = "session"
	ProtocolContextScopeRun     ProtocolContextScope = "run"
	ProtocolContextScopeAgent   ProtocolContextScope = "agent"
)

type ProtocolParticipantRef struct {
	ID     string         `json:"id"`
	Role   string         `json:"role"`
	Source ProtocolSource `json:"source,omitempty"`
}

type ProtocolContextLimits struct {
	AllowedMetadataKeys   []string `json:"allowed_metadata_keys,omitempty"`
	MaxParticipants       int      `json:"max_participants"`
	MaxMetadataEntries    int      `json:"max_metadata_entries"`
	MaxMetadataKeyBytes   int      `json:"max_metadata_key_bytes"`
	MaxMetadataValueBytes int      `json:"max_metadata_value_bytes"`
	MaxSerializedBytes    int      `json:"max_serialized_bytes"`
}

// ProtocolSessionContext carries only bounded scalar metadata and references.
type ProtocolSessionContext struct {
	Session      SessionRef               `json:"session"`
	Scope        ProtocolContextScope     `json:"scope"`
	Agent        *ProtocolParticipantRef  `json:"agent,omitempty"`
	Participants []ProtocolParticipantRef `json:"participants,omitempty"`
	Metadata     map[string]string        `json:"metadata,omitempty"`
}

func (c ProtocolSessionContext) Validate(limits ProtocolContextLimits) error {
	if err := c.Session.ValidateProtocolReference(); err != nil {
		return fmt.Errorf("validate session context: %w", err)
	}
	if c.Scope != ProtocolContextScopeSession && c.Scope != ProtocolContextScopeRun && c.Scope != ProtocolContextScopeAgent {
		return protocolValidationError("unsupported context scope %q", c.Scope)
	}
	if limits.MaxParticipants < 0 || limits.MaxMetadataEntries < 0 || limits.MaxMetadataKeyBytes < 0 || limits.MaxMetadataValueBytes < 0 || limits.MaxSerializedBytes < 0 {
		return protocolValidationError("context limits must not be negative")
	}
	if limits.MaxParticipants > 0 && len(c.Participants) > limits.MaxParticipants {
		return protocolValidationError("participant count exceeds limit")
	}
	for _, participant := range c.Participants {
		if strings.TrimSpace(participant.ID) == "" || strings.TrimSpace(participant.Role) == "" {
			return protocolValidationError("participant id and role are required")
		}
		if participant.Source != "" {
			if err := validateProtocolSource(participant.Source); err != nil {
				return err
			}
		}
	}
	if c.Agent != nil {
		if strings.TrimSpace(c.Agent.ID) == "" || strings.TrimSpace(c.Agent.Role) == "" {
			return protocolValidationError("agent id and role are required")
		}
		if c.Agent.Source != "" {
			if err := validateProtocolSource(c.Agent.Source); err != nil {
				return err
			}
		}
	}
	if limits.MaxMetadataEntries > 0 && len(c.Metadata) > limits.MaxMetadataEntries {
		return protocolValidationError("metadata entry count exceeds limit")
	}
	allowed := make(map[string]struct{}, len(limits.AllowedMetadataKeys))
	for _, key := range limits.AllowedMetadataKeys {
		allowed[normalizeProtocolName(key)] = struct{}{}
	}
	metadataKeys := make([]string, 0, len(c.Metadata))
	for key := range c.Metadata {
		metadataKeys = append(metadataKeys, key)
	}
	sort.Strings(metadataKeys)
	for _, key := range metadataKeys {
		value := c.Metadata[key]
		normalizedKey := normalizeProtocolName(key)
		if normalizedKey == "" {
			return protocolValidationError("metadata key is required")
		}
		if len(allowed) == 0 {
			return protocolValidationError("metadata key %q is not allowed", key)
		}
		if _, ok := allowed[normalizedKey]; !ok {
			return protocolValidationError("metadata key %q is not allowed", key)
		}
		if limits.MaxMetadataKeyBytes > 0 && len(key) > limits.MaxMetadataKeyBytes {
			return protocolValidationError("metadata key %q exceeds byte limit", key)
		}
		if limits.MaxMetadataValueBytes > 0 && len(value) > limits.MaxMetadataValueBytes {
			return protocolValidationError("metadata value for %q exceeds byte limit", key)
		}
	}
	if limits.MaxSerializedBytes > 0 {
		encoded, err := json.Marshal(c)
		if err != nil {
			return fmt.Errorf("marshal protocol session context: %w", err)
		}
		if len(encoded) > limits.MaxSerializedBytes {
			return protocolValidationError("context serialized size exceeds limit")
		}
	}
	return nil
}

type ProtocolConcurrentRunPolicy string

const (
	ProtocolConcurrentRunPolicyReject     ProtocolConcurrentRunPolicy = "reject"
	ProtocolConcurrentRunPolicySerialize  ProtocolConcurrentRunPolicy = "serialize"
	ProtocolConcurrentRunPolicyBranch     ProtocolConcurrentRunPolicy = "branch"
	ProtocolConcurrentRunPolicyOptimistic ProtocolConcurrentRunPolicy = "optimistic"
	ProtocolConcurrentRunPolicyUnknown    ProtocolConcurrentRunPolicy = "unknown"
)

type ProtocolRunAdmissionDecision string

const (
	ProtocolRunAdmissionDecisionAdmitted   ProtocolRunAdmissionDecision = "admitted"
	ProtocolRunAdmissionDecisionQueued     ProtocolRunAdmissionDecision = "queued"
	ProtocolRunAdmissionDecisionBranched   ProtocolRunAdmissionDecision = "branched"
	ProtocolRunAdmissionDecisionRejected   ProtocolRunAdmissionDecision = "rejected"
	ProtocolRunAdmissionDecisionUnresolved ProtocolRunAdmissionDecision = "unresolved"
)

// ProtocolRunAdmission projects a source-owned same-Session decision. It is
// intentionally side-effect-free and cannot enqueue, lock, or branch work.
type ProtocolRunAdmission struct {
	SessionID             string                       `json:"session_id"`
	RunID                 string                       `json:"run_id"`
	Policy                ProtocolConcurrentRunPolicy  `json:"policy"`
	Decision              ProtocolRunAdmissionDecision `json:"decision"`
	ReasonCode            string                       `json:"reason_code"`
	ConflictingRunIDs     []string                     `json:"conflicting_run_ids,omitempty"`
	BranchRunID           string                       `json:"branch_run_id,omitempty"`
	SourceOutcomeDeclared bool                         `json:"source_outcome_declared,omitempty"`
}

func (a ProtocolRunAdmission) Validate() error {
	if strings.TrimSpace(a.SessionID) == "" || strings.TrimSpace(a.RunID) == "" {
		return protocolValidationError("admission session_id and run_id are required")
	}
	if strings.TrimSpace(a.ReasonCode) == "" {
		return protocolValidationError("admission reason_code is required")
	}
	if a.Policy == ProtocolConcurrentRunPolicyUnknown && strings.TrimSpace(a.ReasonCode) != ProtocolReasonAdmissionPolicyUnknown && !a.SourceOutcomeDeclared {
		return protocolValidationError("%s: unknown policy requires unresolved reason", ProtocolReasonAdmissionPolicyUnknown)
	}
	if !isValidProtocolConcurrentRunPolicy(a.Policy) || !isValidProtocolRunAdmissionDecision(a.Decision) {
		return protocolValidationError("unsupported admission policy/outcome %q/%q", a.Policy, a.Decision)
	}
	if !isCompatibleProtocolAdmission(a.Policy, a.Decision, a.SourceOutcomeDeclared) {
		return protocolValidationError("%s: policy %q cannot emit outcome %q", ProtocolReasonAdmissionIncompatibleOutcome, a.Policy, a.Decision)
	}
	if a.Decision == ProtocolRunAdmissionDecisionBranched && strings.TrimSpace(a.BranchRunID) == "" {
		return protocolValidationError("branched admission requires branch_run_id")
	}
	return nil
}

// ProjectProtocolRunAdmission validates and clones a source-owned outcome.
// It never queues, branches, locks, or mutates the source decision.
func ProjectProtocolRunAdmission(admission ProtocolRunAdmission) (ProtocolRunAdmission, error) {
	if err := admission.Validate(); err != nil {
		return ProtocolRunAdmission{}, err
	}
	admission.ConflictingRunIDs = append([]string(nil), admission.ConflictingRunIDs...)
	return admission, nil
}

// ProtocolDescriptorForSource creates an opt-in descriptor while preserving
// the source module as the owner of runtime semantics.
func ProtocolDescriptorForSource(source ProtocolSource, runtimeID, profileVersion string, capabilities []ProtocolCapability, actions []ProtocolAction) (ProtocolDescriptor, error) {
	descriptor := ProtocolDescriptor{
		ProtocolName:   AgentRuntimeProtocolName,
		ProfileVersion: strings.TrimSpace(profileVersion),
		RuntimeID:      strings.TrimSpace(runtimeID),
		Source:         source,
		Capabilities:   append([]ProtocolCapability(nil), capabilities...),
		Actions:        append([]ProtocolAction(nil), actions...),
	}
	if err := descriptor.Validate(); err != nil {
		return ProtocolDescriptor{}, err
	}
	return descriptor, nil
}

// ProjectProtocolSessionContext creates a bounded, reference-only context
// projection. Inputs are cloned before validation so a rejected projection
// cannot mutate source-owned Session data.
func ProjectProtocolSessionContext(session SessionRef, scope ProtocolContextScope, agent *ProtocolParticipantRef, participants []ProtocolParticipantRef, metadata map[string]string, limits ProtocolContextLimits) (ProtocolSessionContext, error) {
	context := ProtocolSessionContext{
		Session:      session,
		Scope:        scope,
		Participants: append([]ProtocolParticipantRef(nil), participants...),
		Metadata:     make(map[string]string, len(metadata)),
	}
	for key, value := range metadata {
		context.Metadata[key] = value
	}
	if agent != nil {
		cloned := *agent
		context.Agent = &cloned
	}
	if err := context.Validate(limits); err != nil {
		return ProtocolSessionContext{}, err
	}
	return context, nil
}

// ProtocolReference is implemented by each reference-only protocol DTO.
type ProtocolReference interface {
	ValidateProtocolReference() error
}

// SessionRef identifies a long-lived context boundary.
type SessionRef struct {
	SessionID string         `json:"session_id"`
	Source    ProtocolSource `json:"source"`
}

// RunRef identifies one managed execution within an optional session.
type RunRef struct {
	RunID       string         `json:"run_id"`
	SessionID   string         `json:"session_id,omitempty"`
	State       RunState       `json:"state"`
	Source      ProtocolSource `json:"source"`
	CausationID string         `json:"causation_id,omitempty"`
}

// StepRef identifies one observable unit of execution.
type StepRef struct {
	StepID       string           `json:"step_id"`
	RunID        string           `json:"run_id"`
	SessionID    string           `json:"session_id,omitempty"`
	ParentStepID string           `json:"parent_step_id,omitempty"`
	Kind         ProtocolStepKind `json:"kind"`
	Source       ProtocolSource   `json:"source"`
	CausationID  string           `json:"causation_id,omitempty"`
}

// EventEnvelope is the normalized event view. Sequence is meaningful only
// when the source is realtime.
type EventEnvelope struct {
	EventID       string                      `json:"event_id"`
	RunID         string                      `json:"run_id,omitempty"`
	SessionID     string                      `json:"session_id,omitempty"`
	StepID        string                      `json:"step_id,omitempty"`
	CausationID   string                      `json:"causation_id,omitempty"`
	Source        ProtocolSource              `json:"source"`
	Kind          ProtocolEventKind           `json:"kind"`
	Time          time.Time                   `json:"time"`
	Sequence      int64                       `json:"sequence,omitempty"`
	Payload       map[string]any              `json:"payload,omitempty"`
	StreamBinding *ProtocolEventStreamBinding `json:"stream_binding,omitempty"`
}

// ProtocolEventStreamBinding carries additive subscription correlation on a
// protocol event. It describes source-owned binding state and is not an
// authorization or transport object.
type ProtocolEventStreamBinding struct {
	SubscriptionID   string                  `json:"subscription_id"`
	Phase            EventStreamBindingPhase `json:"phase"`
	ReasonCode       string                  `json:"reason_code"`
	CursorMode       EventStreamStartMode    `json:"cursor_mode"`
	SequenceBoundary int64                   `json:"sequence_boundary,omitempty"`
}

// ArtifactRef describes a produced artifact without copying or owning content.
type ArtifactRef struct {
	ArtifactID       string `json:"artifact_id"`
	Type             string `json:"type"`
	Locator          string `json:"locator"`
	Digest           string `json:"digest,omitempty"`
	ProducedByRunID  string `json:"produced_by_run_id,omitempty"`
	ProducedByStepID string `json:"produced_by_step_id,omitempty"`
}

// CheckpointRef describes a recoverable state reference without owning its
// manifest structure or import behavior.
type CheckpointRef struct {
	CheckpointID        string                  `json:"checkpoint_id"`
	SchemaVersion       string                  `json:"schema_version"`
	SourceComponent     string                  `json:"source_component"`
	Digest              string                  `json:"digest"`
	RunID               string                  `json:"run_id,omitempty"`
	SessionID           string                  `json:"session_id,omitempty"`
	Relation            CheckpointRelation      `json:"relation,omitempty"`
	ParentCheckpointID  string                  `json:"parent_checkpoint_id,omitempty"`
	BranchID            string                  `json:"branch_id,omitempty"`
	HistoryIndex        int                     `json:"history_index,omitempty"`
	RestoreSource       CheckpointRestoreSource `json:"restore_source,omitempty"`
	ReplayKey           string                  `json:"replay_key,omitempty"`
	WorkspaceProvenance *WorkspaceProvenance    `json:"workspace_provenance,omitempty"`
}

// CheckpointRelation describes how a checkpoint relates to its history.
type CheckpointRelation string

const (
	CheckpointRelationRoot    CheckpointRelation = "root"
	CheckpointRelationDerived CheckpointRelation = "derived"
	CheckpointRelationBranch  CheckpointRelation = "branch"
	CheckpointRelationReplay  CheckpointRelation = "replay"
)

// CheckpointRestoreSource identifies the source of a recovery projection.
type CheckpointRestoreSource string

const (
	CheckpointRestoreSourceFresh        CheckpointRestoreSource = "fresh"
	CheckpointRestoreSourceResume       CheckpointRestoreSource = "resume"
	CheckpointRestoreSourceCrossSession CheckpointRestoreSource = "cross_session"
)

// WorkspaceProvenance references a bounded workspace change without owning
// workspace contents or policy decisions.
type WorkspaceProvenance struct {
	WorkspaceID      string `json:"workspace_id"`
	ChangeSetID      string `json:"change_set_id"`
	BeforeIntegrity  string `json:"before_integrity"`
	AfterIntegrity   string `json:"after_integrity"`
	ProducedByRunID  string `json:"produced_by_run_id"`
	ProducedByStepID string `json:"produced_by_step_id"`
	CheckpointID     string `json:"checkpoint_id,omitempty"`
}

func (p WorkspaceProvenance) Validate() error {
	if strings.TrimSpace(p.WorkspaceID) == "" || strings.TrimSpace(p.ChangeSetID) == "" || strings.TrimSpace(p.BeforeIntegrity) == "" || strings.TrimSpace(p.AfterIntegrity) == "" {
		return protocolValidationError("%s", ProtocolReasonWorkspaceProvenanceMissing)
	}
	if strings.TrimSpace(p.ProducedByRunID) == "" || strings.TrimSpace(p.ProducedByStepID) == "" {
		return protocolValidationError("%s", ProtocolReasonWorkspaceAssociationMismatch)
	}
	return nil
}

func (r SessionRef) ValidateProtocolReference() error {
	if strings.TrimSpace(r.SessionID) == "" {
		return protocolValidationError("session_id is required")
	}
	return validateProtocolSource(r.Source)
}

func (r RunRef) ValidateProtocolReference() error {
	if strings.TrimSpace(r.RunID) == "" {
		return protocolValidationError("run_id is required")
	}
	if !isValidRunState(r.State) {
		return protocolValidationError("run state %q is unsupported", r.State)
	}
	return validateProtocolSource(r.Source)
}

func (r StepRef) ValidateProtocolReference() error {
	if strings.TrimSpace(r.StepID) == "" {
		return protocolValidationError("step_id is required")
	}
	if strings.TrimSpace(r.RunID) == "" {
		return protocolValidationError("run_id is required")
	}
	if !isValidProtocolStepKind(r.Kind) {
		return protocolValidationError("step kind %q is unsupported", r.Kind)
	}
	return validateProtocolSource(r.Source)
}

func (e EventEnvelope) ValidateProtocolReference() error {
	if strings.TrimSpace(e.EventID) == "" {
		return protocolValidationError("event_id is required")
	}
	if !isValidProtocolEventKind(e.Kind) {
		return protocolValidationError("event kind %q is unsupported", e.Kind)
	}
	if e.Time.IsZero() {
		return protocolValidationError("event time is required")
	}
	if err := validateProtocolSource(e.Source); err != nil {
		return err
	}
	if e.Source == ProtocolSourceRealtime && e.Sequence <= 0 {
		return protocolValidationError("realtime event sequence must be > 0")
	}
	if e.Source != ProtocolSourceRealtime && e.Sequence != 0 {
		return protocolValidationError("non-realtime event must not carry sequence")
	}
	return nil
}

func (r ArtifactRef) ValidateProtocolReference() error {
	if strings.TrimSpace(r.ArtifactID) == "" {
		return protocolValidationError("artifact_id is required")
	}
	if strings.TrimSpace(r.Type) == "" {
		return protocolValidationError("artifact type is required")
	}
	if strings.TrimSpace(r.Locator) == "" {
		return protocolValidationError("artifact locator is required")
	}
	return nil
}

func (r CheckpointRef) ValidateProtocolReference() error {
	if strings.TrimSpace(r.CheckpointID) == "" {
		return protocolValidationError("checkpoint_id is required")
	}
	if strings.TrimSpace(r.SchemaVersion) == "" {
		return protocolValidationError("checkpoint schema_version is required")
	}
	if strings.TrimSpace(r.SourceComponent) == "" {
		return protocolValidationError("checkpoint source_component is required")
	}
	if strings.TrimSpace(r.Digest) == "" {
		return protocolValidationError("checkpoint digest is required")
	}
	relation := r.Relation
	if relation == "" {
		relation = CheckpointRelationRoot
	}
	switch relation {
	case CheckpointRelationRoot, CheckpointRelationDerived, CheckpointRelationBranch, CheckpointRelationReplay:
	default:
		return protocolValidationError("unsupported checkpoint relation %q", relation)
	}
	if r.HistoryIndex < 0 {
		return protocolValidationError("checkpoint history_index must be >= 0")
	}
	if relation != CheckpointRelationRoot && strings.TrimSpace(r.ParentCheckpointID) == "" {
		return protocolValidationError("%s", ProtocolReasonCheckpointLineageMissingParent)
	}
	if relation == CheckpointRelationBranch && strings.TrimSpace(r.BranchID) == "" {
		return protocolValidationError("checkpoint branch_id is required")
	}
	if relation == CheckpointRelationReplay && strings.TrimSpace(r.ReplayKey) == "" {
		return protocolValidationError("checkpoint replay_key is required")
	}
	if r.RestoreSource != "" {
		switch r.RestoreSource {
		case CheckpointRestoreSourceFresh, CheckpointRestoreSourceResume, CheckpointRestoreSourceCrossSession:
		default:
			return protocolValidationError("unsupported checkpoint restore_source %q", r.RestoreSource)
		}
	}
	if r.WorkspaceProvenance != nil {
		if err := r.WorkspaceProvenance.Validate(); err != nil {
			return err
		}
		if id := strings.TrimSpace(r.WorkspaceProvenance.CheckpointID); id != "" && id != strings.TrimSpace(r.CheckpointID) {
			return protocolValidationError("%s", ProtocolReasonWorkspaceAssociationMismatch)
		}
	}
	return nil
}

// ValidateWorkspaceIntegrity compares an observed pre-restore integrity
// reference with the source-owned provenance reference.
func ValidateWorkspaceIntegrity(observedBefore string, provenance WorkspaceProvenance) error {
	if err := provenance.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(observedBefore) != strings.TrimSpace(provenance.BeforeIntegrity) {
		return protocolValidationError("%s", ProtocolReasonWorkspaceIntegrityDrift)
	}
	return nil
}

// ValidateCheckpointHistory validates an ordered, caller-provided history
// without creating or mutating a checkpoint store.
func ValidateCheckpointHistory(history []CheckpointRef) error {
	seen := make(map[string]struct{}, len(history))
	for index, checkpoint := range history {
		if err := checkpoint.ValidateProtocolReference(); err != nil {
			return err
		}
		id := strings.TrimSpace(checkpoint.CheckpointID)
		if _, exists := seen[id]; exists {
			return protocolValidationError("%s", ProtocolReasonCheckpointHistoryDisconnected)
		}
		seen[id] = struct{}{}
		if checkpoint.HistoryIndex != index {
			return protocolValidationError("%s", ProtocolReasonCheckpointHistoryDisconnected)
		}
		if index == 0 {
			if checkpoint.Relation != "" && checkpoint.Relation != CheckpointRelationRoot {
				return protocolValidationError("%s", ProtocolReasonCheckpointHistoryDisconnected)
			}
			continue
		}
		if strings.TrimSpace(checkpoint.ParentCheckpointID) == "" {
			return protocolValidationError("%s", ProtocolReasonCheckpointHistoryDisconnected)
		}
		if _, exists := seen[strings.TrimSpace(checkpoint.ParentCheckpointID)]; !exists {
			return protocolValidationError("%s", ProtocolReasonCheckpointHistoryDisconnected)
		}
	}
	return nil
}

// ValidateCheckpointReplay checks that a replay key maps to equivalent data.
func ValidateCheckpointReplay(existing, candidate CheckpointRef) error {
	if strings.TrimSpace(existing.ReplayKey) == "" || strings.TrimSpace(candidate.ReplayKey) == "" || strings.TrimSpace(existing.ReplayKey) != strings.TrimSpace(candidate.ReplayKey) {
		return protocolValidationError("%s", ProtocolReasonCheckpointReplayConflict)
	}
	if existing.CheckpointID != candidate.CheckpointID || existing.SchemaVersion != candidate.SchemaVersion || existing.SourceComponent != candidate.SourceComponent || existing.Digest != candidate.Digest || existing.ParentCheckpointID != candidate.ParentCheckpointID || existing.BranchID != candidate.BranchID || existing.HistoryIndex != candidate.HistoryIndex {
		return protocolValidationError("%s", ProtocolReasonCheckpointReplayConflict)
	}
	return nil
}

// ValidateRunStateTransition validates a projection transition without
// applying it to any source Runtime state.
func ValidateRunStateTransition(from, to RunState) error {
	if !isValidRunState(from) || !isValidRunState(to) {
		return protocolValidationError("unsupported run state transition %q -> %q", from, to)
	}
	if from == RunStateCanceled && to == RunStateCanceled {
		return nil
	}
	allowed := map[RunState][]RunState{
		RunStateSubmitted:     {RunStateWorking, RunStateCanceled},
		RunStateWorking:       {RunStateInputRequired, RunStateCompleted, RunStateFailed, RunStateCanceled},
		RunStateInputRequired: {RunStateWorking, RunStateCanceled},
	}
	for _, candidate := range allowed[from] {
		if to == candidate {
			return nil
		}
	}
	return protocolValidationError("invalid run state transition %q -> %q", from, to)
}

// NewRetryRunRef creates a new submitted run causally linked to a terminal,
// non-successful source run. The source RunRef is never mutated.
func NewRetryRunRef(previous RunRef, retryRunID string) (RunRef, error) {
	if err := previous.ValidateProtocolReference(); err != nil {
		return RunRef{}, fmt.Errorf("validate retry source run: %w", err)
	}
	if previous.State != RunStateFailed && previous.State != RunStateCanceled {
		return RunRef{}, protocolValidationError("retry requires failed or canceled run state, got %q", previous.State)
	}
	retry := RunRef{
		RunID:       strings.TrimSpace(retryRunID),
		SessionID:   strings.TrimSpace(previous.SessionID),
		State:       RunStateSubmitted,
		Source:      previous.Source,
		CausationID: strings.TrimSpace(previous.RunID),
	}
	if err := retry.ValidateProtocolReference(); err != nil {
		return RunRef{}, err
	}
	return retry, nil
}

// CanRepresentRecoverableStepOutcome preserves Baymax fail-fast boundaries.
// The caller must first establish recoverability from the owning Runtime path.
func CanRepresentRecoverableStepOutcome(boundary ProtocolFailureBoundary, ownerAllowsRecovery bool) bool {
	if !ownerAllowsRecovery {
		return false
	}
	switch boundary {
	case ProtocolFailureTool, ProtocolFailureBusiness:
		return true
	default:
		return false
	}
}

// NewProtocolRunFromRequest projects a Runner request into a public RunRef.
func NewProtocolRunFromRequest(req RunRequest, state RunState) (RunRef, error) {
	run := RunRef{
		RunID:     strings.TrimSpace(req.RunID),
		SessionID: strings.TrimSpace(req.SessionID),
		State:     state,
		Source:    ProtocolSourceRunner,
	}
	if err := run.ValidateProtocolReference(); err != nil {
		return RunRef{}, err
	}
	return run, nil
}

// MapRunnerResultToProtocol projects terminal Runner output and tool call
// summaries without changing the Runner's execution or retry semantics.
func MapRunnerResultToProtocol(result RunResult, sessionID string) (RunRef, []StepRef, error) {
	outcome, err := TerminalOutcomeFromRunResult(result, sessionID)
	if err != nil {
		return RunRef{}, nil, err
	}
	run := RunRef{
		RunID:     strings.TrimSpace(result.RunID),
		SessionID: strings.TrimSpace(sessionID),
		State:     outcome.State,
		Source:    ProtocolSourceRunner,
	}
	if err := run.ValidateProtocolReference(); err != nil {
		return RunRef{}, nil, err
	}
	steps := make([]StepRef, 0, len(result.ToolCalls))
	for index, call := range result.ToolCalls {
		stepID := strings.TrimSpace(call.CallID)
		if stepID == "" {
			stepID = fmt.Sprintf("tool-%d", index+1)
		}
		step := StepRef{
			StepID:    stepID,
			RunID:     run.RunID,
			SessionID: run.SessionID,
			Kind:      ProtocolStepKindTool,
			Source:    ProtocolSourceRunner,
		}
		if call.Error != nil {
			step.CausationID = string(call.Error.Class)
		}
		if err := step.ValidateProtocolReference(); err != nil {
			return RunRef{}, nil, err
		}
		steps = append(steps, step)
	}
	return run, steps, nil
}

// MapEventToProtocol projects a standard event into a source-scoped envelope.
// Non-realtime sources intentionally never receive a synthetic sequence.
func MapEventToProtocol(ev Event, source ProtocolSource) (EventEnvelope, error) {
	when := ev.Time
	if when.IsZero() {
		return EventEnvelope{}, protocolValidationError("event time is required")
	}
	eventID := strings.TrimSpace(ev.TraceID)
	if eventID == "" {
		eventID = stableProtocolEventID(source, ev.RunID, ev.CallID, ev.Type, when)
	}
	kind := ProtocolEventKindProgress
	if _, ok := ev.Payload["error"]; ok {
		kind = ProtocolEventKindError
	} else if status, _ := ev.Payload["status"].(string); strings.EqualFold(strings.TrimSpace(status), string(ActionStatusFailed)) {
		kind = ProtocolEventKindError
	}
	envelope := EventEnvelope{
		EventID: eventID,
		RunID:   strings.TrimSpace(ev.RunID),
		StepID:  strings.TrimSpace(ev.CallID),
		Source:  source,
		Kind:    kind,
		Time:    when.UTC(),
		Payload: cloneProtocolPayload(ev.Payload),
	}
	if err := envelope.ValidateProtocolReference(); err != nil {
		return EventEnvelope{}, err
	}
	return envelope, nil
}

// MapRealtimeEventToProtocol retains the realtime event identity, sequence,
// session, and run correlation. Realtime remains the source of ordering truth.
func MapRealtimeEventToProtocol(ev RealtimeEventEnvelope) (EventEnvelope, error) {
	if err := ev.Validate(); err != nil {
		return EventEnvelope{}, err
	}
	kind := ProtocolEventKindProgress
	switch ev.Type {
	case RealtimeEventTypeInterrupt, RealtimeEventTypeResume, RealtimeEventTypeAck:
		kind = ProtocolEventKindState
	case RealtimeEventTypeError:
		kind = ProtocolEventKindError
	case RealtimeEventTypeComplete:
		kind = ProtocolEventKindState
	}
	envelope := EventEnvelope{
		EventID:   strings.TrimSpace(ev.EventID),
		RunID:     strings.TrimSpace(ev.RunID),
		SessionID: strings.TrimSpace(ev.SessionID),
		Source:    ProtocolSourceRealtime,
		Kind:      kind,
		Time:      ev.TS.UTC(),
		Sequence:  ev.Seq,
		Payload:   cloneProtocolPayload(ev.Payload),
	}
	if err := envelope.ValidateProtocolReference(); err != nil {
		return EventEnvelope{}, err
	}
	return envelope, nil
}

// MapEventStreamBindingToProtocol projects source-owned binding events to
// canonical protocol envelopes while retaining subscription correlation.
func MapEventStreamBindingToProtocol(projection EventStreamBindingProjection) ([]EventEnvelope, error) {
	if err := projection.Subscription.Validate(); err != nil {
		return nil, err
	}
	if err := projection.Outcome.Validate(projection.Subscription); err != nil {
		return nil, err
	}
	binding := &ProtocolEventStreamBinding{
		SubscriptionID:   strings.TrimSpace(projection.Subscription.SubscriptionID),
		Phase:            projection.Outcome.Phase,
		ReasonCode:       strings.TrimSpace(projection.Outcome.ReasonCode),
		CursorMode:       projection.Subscription.StartMode,
		SequenceBoundary: projection.Outcome.LastSequence,
	}
	events := make([]EventEnvelope, 0, len(projection.Events))
	for _, event := range projection.Events {
		mapped, err := MapRealtimeEventToProtocol(event)
		if err != nil {
			return nil, err
		}
		clonedBinding := *binding
		mapped.StreamBinding = &clonedBinding
		events = append(events, mapped)
	}
	return events, nil
}

// MapHandoffArtifactToProtocol projects a deferred context artifact without
// copying its body into the protocol reference.
func MapHandoffArtifactToProtocol(artifact IsolateHandoffArtifact, runID, stepID string) (ArtifactRef, error) {
	ref := ArtifactRef{
		ArtifactID:       strings.TrimSpace(artifact.ID),
		Type:             strings.TrimSpace(artifact.Type),
		Locator:          strings.TrimSpace(artifact.Locator),
		ProducedByRunID:  strings.TrimSpace(runID),
		ProducedByStepID: strings.TrimSpace(stepID),
	}
	if err := ref.ValidateProtocolReference(); err != nil {
		return ArtifactRef{}, err
	}
	return ref, nil
}

func stableProtocolEventID(source ProtocolSource, runID, callID, eventType string, when time.Time) string {
	seed := strings.Join([]string{string(source), strings.TrimSpace(runID), strings.TrimSpace(callID), strings.TrimSpace(eventType), when.UTC().Format(time.RFC3339Nano)}, "|")
	sum := sha256.Sum256([]byte(seed))
	return "event_" + hex.EncodeToString(sum[:8])
}

func cloneProtocolPayload(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func validateProtocolSource(source ProtocolSource) error {
	switch source {
	case ProtocolSourceRunner, ProtocolSourceWorkflow, ProtocolSourceComposer, ProtocolSourceTeams, ProtocolSourceScheduler, ProtocolSourceA2A, ProtocolSourceRealtime, ProtocolSourceTimeline:
		return nil
	default:
		return protocolValidationError("protocol source %q is unsupported", source)
	}
}

func isValidRunState(state RunState) bool {
	switch state {
	case RunStateSubmitted, RunStateWorking, RunStateInputRequired, RunStateCompleted, RunStateFailed, RunStateCanceled:
		return true
	default:
		return false
	}
}

func isValidProtocolStepKind(kind ProtocolStepKind) bool {
	switch kind {
	case ProtocolStepKindModel, ProtocolStepKindTool, ProtocolStepKindWorkflow, ProtocolStepKindTeamTask, ProtocolStepKindScheduler, ProtocolStepKindA2A, ProtocolStepKindHITL:
		return true
	default:
		return false
	}
}

func isValidProtocolEventKind(kind ProtocolEventKind) bool {
	switch kind {
	case ProtocolEventKindProgress, ProtocolEventKindState, ProtocolEventKindError, ProtocolEventKindArtifact:
		return true
	default:
		return false
	}
}

func protocolValidationError(format string, args ...any) error {
	return fmt.Errorf("agent runtime protocol: "+format, args...)
}

func normalizeProtocolName(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func missingProtocolCapabilities(declared map[string]struct{}, requested []string) []string {
	missing := make([]string, 0, len(requested))
	seen := make(map[string]struct{}, len(requested))
	for _, capability := range requested {
		name := normalizeProtocolName(capability)
		if name == "" {
			continue
		}
		if _, already := seen[name]; already {
			continue
		}
		seen[name] = struct{}{}
		if _, ok := declared[name]; !ok {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	return missing
}

func appendUniqueProtocolCapabilities(base []string, values ...string) []string {
	seen := make(map[string]struct{}, len(base)+len(values))
	out := make([]string, 0, len(base)+len(values))
	for _, value := range append(append([]string(nil), base...), values...) {
		name := normalizeProtocolName(value)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func isValidProtocolConcurrentRunPolicy(policy ProtocolConcurrentRunPolicy) bool {
	switch policy {
	case ProtocolConcurrentRunPolicyReject, ProtocolConcurrentRunPolicySerialize, ProtocolConcurrentRunPolicyBranch, ProtocolConcurrentRunPolicyOptimistic, ProtocolConcurrentRunPolicyUnknown:
		return true
	default:
		return false
	}
}

func isValidProtocolRunAdmissionDecision(decision ProtocolRunAdmissionDecision) bool {
	switch decision {
	case ProtocolRunAdmissionDecisionAdmitted, ProtocolRunAdmissionDecisionQueued, ProtocolRunAdmissionDecisionBranched, ProtocolRunAdmissionDecisionRejected, ProtocolRunAdmissionDecisionUnresolved:
		return true
	default:
		return false
	}
}

func isCompatibleProtocolAdmission(policy ProtocolConcurrentRunPolicy, decision ProtocolRunAdmissionDecision, sourceOutcomeDeclared bool) bool {
	switch policy {
	case ProtocolConcurrentRunPolicyReject:
		return decision == ProtocolRunAdmissionDecisionAdmitted || decision == ProtocolRunAdmissionDecisionRejected
	case ProtocolConcurrentRunPolicySerialize:
		return decision == ProtocolRunAdmissionDecisionAdmitted || decision == ProtocolRunAdmissionDecisionQueued
	case ProtocolConcurrentRunPolicyBranch:
		return decision == ProtocolRunAdmissionDecisionAdmitted || decision == ProtocolRunAdmissionDecisionBranched
	case ProtocolConcurrentRunPolicyOptimistic:
		return decision == ProtocolRunAdmissionDecisionAdmitted || decision == ProtocolRunAdmissionDecisionRejected
	case ProtocolConcurrentRunPolicyUnknown:
		return decision == ProtocolRunAdmissionDecisionUnresolved || sourceOutcomeDeclared
	default:
		return false
	}
}
