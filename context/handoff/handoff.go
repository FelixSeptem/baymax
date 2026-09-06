// Package handoff defines the bounded, reference-first runtime handoff
// projection used when context is compressed or a run is resumed.
package handoff

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

const VersionV1 = "handoff.v1"

type Cut string

const (
	CutFinalizedEvent  Cut = "finalized_event"
	CutToolFinalized   Cut = "tool_finalized"
	CutCheckpoint      Cut = "checkpoint"
	CutFlushedStream   Cut = "flushed_stream"
	CutUnflushedStream Cut = "unflushed_stream"
)

type EvidenceKind string

const (
	EvidenceFact EvidenceKind = "fact"
)

type ReferenceKind string

const (
	ReferenceArtifact       ReferenceKind = "artifact"
	ReferenceCheckpoint     ReferenceKind = "checkpoint"
	ReferenceSessionHistory ReferenceKind = "session_history"
	ReferenceSnapshot       ReferenceKind = "snapshot"
)

const (
	FallbackQualityBelowThreshold = "handoff_quality_below_threshold"
	FallbackInvalidCut            = "handoff_cut_invalid"
	FallbackReferenceLoss         = "handoff_reference_loss"
	FallbackGenerationFailure     = "handoff_generation_failure"
)

var ErrReferenceNotFound = errors.New("handoff reference not found")

type Evidence struct {
	Kind      EvidenceKind `json:"kind"`
	Value     string       `json:"value"`
	SourceID  string       `json:"source_id"`
	Protected bool         `json:"protected,omitempty"`
}

type Inference struct {
	Value      string   `json:"value"`
	SourceIDs  []string `json:"source_ids"`
	Confidence float64  `json:"confidence"`
}

type Reference struct {
	Kind   ReferenceKind `json:"kind"`
	ID     string        `json:"id"`
	Digest string        `json:"digest,omitempty"`
}

type Action struct {
	Description string `json:"description"`
	SourceID    string `json:"source_id,omitempty"`
}

type Quality struct {
	Score              float64 `json:"score"`
	Threshold          float64 `json:"threshold"`
	ProtectedEvidence  bool    `json:"protected_evidence"`
	ReferencesResolved bool    `json:"references_resolved"`
	RestoreReady       bool    `json:"restore_ready"`
}

// CompressionProjection carries only source-owned compression governance
// metadata. Bodies remain owned by the assembler, journal, spill backend, and
// checkpoint/history stores.
type CompressionProjection struct {
	PressureZone              string         `json:"pressure_zone,omitempty"`
	PressureReason            string         `json:"pressure_reason,omitempty"`
	PressureTriggerSource     string         `json:"pressure_trigger_source,omitempty"`
	CompactionFallback        bool           `json:"compaction_fallback,omitempty"`
	CompactionFallbackReason  string         `json:"compaction_fallback_reason,omitempty"`
	CompactionQualityScore    float64        `json:"compaction_quality_score,omitempty"`
	RetainedEvidenceCount     int            `json:"retained_evidence_count,omitempty"`
	SpillCount                int            `json:"spill_count,omitempty"`
	SwapBackCount             int            `json:"swap_back_count,omitempty"`
	LifecycleTierStats        map[string]int `json:"lifecycle_tier_stats,omitempty"`
	TierTransitionReason      string         `json:"tier_transition_reason,omitempty"`
	ColdStoreGovernanceAction string         `json:"cold_store_governance_action,omitempty"`
	RecoveryConsistencyMarker string         `json:"recovery_consistency_marker,omitempty"`
}

type Fallback struct {
	Reason string `json:"reason"`
}

type Handoff struct {
	Version            string                `json:"version"`
	RunID              string                `json:"run_id"`
	SessionID          string                `json:"session_id,omitempty"`
	SourceCheckpointID string                `json:"source_checkpoint_id,omitempty"`
	Cut                Cut                   `json:"cut"`
	Objective          string                `json:"objective"`
	Completed          []string              `json:"completed,omitempty"`
	Pending            []string              `json:"pending,omitempty"`
	FailedAttempts     []string              `json:"failed_attempts,omitempty"`
	FileChanges        []string              `json:"file_changes,omitempty"`
	ToolResults        []string              `json:"tool_results,omitempty"`
	PolicyState        map[string]string     `json:"policy_state,omitempty"`
	SandboxState       map[string]string     `json:"sandbox_state,omitempty"`
	AdmissionState     map[string]string     `json:"admission_state,omitempty"`
	References         []Reference           `json:"references,omitempty"`
	NextActions        []Action              `json:"next_actions,omitempty"`
	Facts              []Evidence            `json:"facts,omitempty"`
	Inferences         []Inference           `json:"inferences,omitempty"`
	NeedsConfirmation  []string              `json:"needs_confirmation,omitempty"`
	Quality            Quality               `json:"quality"`
	Compression        CompressionProjection `json:"compression,omitempty"`
	Fallback           *Fallback             `json:"fallback,omitempty"`
}

type Limits struct {
	MaxItems          int
	MaxSerializedSize int
}

func DefaultLimits() Limits { return Limits{MaxItems: 128, MaxSerializedSize: 64 * 1024} }

func (h Handoff) Validate(l Limits) error {
	if h.Version != VersionV1 {
		return fmt.Errorf("unsupported handoff version %q", h.Version)
	}
	if strings.TrimSpace(h.RunID) == "" || strings.TrimSpace(h.Objective) == "" {
		return errors.New("handoff run_id and objective are required")
	}
	if h.Quality.Score < 0 || h.Quality.Score > 1 || h.Quality.Threshold < 0 || h.Quality.Threshold > 1 {
		return errors.New("handoff quality score and threshold must be in [0,1]")
	}
	if h.Compression.RetainedEvidenceCount < 0 || h.Compression.SpillCount < 0 || h.Compression.SwapBackCount < 0 {
		return errors.New("handoff compression counters must be non-negative")
	}
	switch h.Cut {
	case CutFinalizedEvent, CutToolFinalized, CutCheckpoint, CutFlushedStream:
	default:
		return fmt.Errorf("invalid handoff cut %q", h.Cut)
	}
	if l.MaxItems > 0 && itemCount(h) > l.MaxItems {
		return fmt.Errorf("handoff item count exceeds limit")
	}
	seenRefs := make(map[string]struct{}, len(h.References))
	for _, ref := range h.References {
		if !validReferenceKind(ref.Kind) || strings.TrimSpace(ref.ID) == "" {
			return errors.New("handoff reference kind and id are required")
		}
		key := string(ref.Kind) + ":" + strings.TrimSpace(ref.ID)
		if _, ok := seenRefs[key]; ok {
			return fmt.Errorf("duplicate handoff reference %q", key)
		}
		seenRefs[key] = struct{}{}
	}
	for _, fact := range h.Facts {
		if fact.Kind != EvidenceFact || strings.TrimSpace(fact.Value) == "" || strings.TrimSpace(fact.SourceID) == "" {
			return errors.New("handoff facts require kind, value, and source_id")
		}
	}
	for _, inference := range h.Inferences {
		if strings.TrimSpace(inference.Value) == "" || len(inference.SourceIDs) == 0 || inference.Confidence < 0 || inference.Confidence > 1 {
			return errors.New("handoff inferences require provenance and confidence in [0,1]")
		}
	}
	for _, action := range h.NextActions {
		if strings.TrimSpace(action.Description) == "" {
			return errors.New("handoff next action description is required")
		}
	}
	if l.MaxSerializedSize > 0 {
		encoded, err := json.Marshal(h)
		if err != nil {
			return fmt.Errorf("marshal handoff: %w", err)
		}
		if len(encoded) > l.MaxSerializedSize {
			return errors.New("handoff serialized size exceeds limit")
		}
	}
	return nil
}

func validReferenceKind(kind ReferenceKind) bool {
	switch kind {
	case ReferenceArtifact, ReferenceCheckpoint, ReferenceSessionHistory, ReferenceSnapshot:
		return true
	default:
		return false
	}
}

func itemCount(h Handoff) int {
	return len(h.Completed) + len(h.Pending) + len(h.FailedAttempts) + len(h.FileChanges) + len(h.ToolResults) + len(h.References) + len(h.NextActions) + len(h.Facts) + len(h.Inferences) + len(h.NeedsConfirmation)
}

type Message struct {
	Role    string
	Content string
}

type BuildRequest struct {
	RunID     string
	SessionID string
	Cut       Cut
	Objective string
	Messages  []Message
}

type BuildOptions struct {
	MinQuality  float64
	Compression CompressionProjection
}

func Build(req BuildRequest, opts BuildOptions) (Handoff, error) {
	h := Handoff{Version: VersionV1, RunID: strings.TrimSpace(req.RunID), SessionID: strings.TrimSpace(req.SessionID), Cut: req.Cut, Objective: strings.TrimSpace(req.Objective)}
	for _, msg := range req.Messages {
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			continue
		}
		h.Facts = append(h.Facts, Evidence{Kind: EvidenceFact, Value: content, SourceID: "message"})
	}
	h.Compression = opts.Compression
	h.Quality = Quality{Score: 0.8, Threshold: opts.MinQuality, ProtectedEvidence: len(h.Facts) > 0 || opts.Compression.RetainedEvidenceCount > 0, ReferencesResolved: true, RestoreReady: true}
	if opts.Compression.CompactionQualityScore > 0 {
		h.Quality.Score = opts.Compression.CompactionQualityScore
	}
	if opts.Compression.CompactionFallback {
		h.Quality.RestoreReady = false
	}
	if opts.MinQuality > 0 && h.Quality.Score < opts.MinQuality {
		h.Fallback = &Fallback{Reason: FallbackQualityBelowThreshold}
	}
	if err := h.Validate(DefaultLimits()); err != nil {
		return Handoff{}, err
	}
	return h, nil
}

type Resolver interface {
	Resolve(Reference) error
}

type ResolverFunc func(Reference) error

func (f ResolverFunc) Resolve(ref Reference) error { return f(ref) }

// OwnerResolver is a reference-first adapter over the existing body owners.
// It validates access through the authoritative owner and never copies source
// bodies into the handoff projection.
type OwnerResolver struct {
	Artifact       Resolver
	Checkpoint     Resolver
	SessionHistory Resolver
	Snapshot       Resolver
}

func (r OwnerResolver) Resolve(ref Reference) error {
	var owner Resolver
	switch ref.Kind {
	case ReferenceArtifact:
		owner = r.Artifact
	case ReferenceCheckpoint:
		owner = r.Checkpoint
	case ReferenceSessionHistory:
		owner = r.SessionHistory
	case ReferenceSnapshot:
		owner = r.Snapshot
	default:
		return ErrReferenceNotFound
	}
	if owner == nil {
		return ErrReferenceNotFound
	}
	return owner.Resolve(ref)
}

type RestoreResult struct {
	HandoffID string
	Restored  bool
}

// RestoreOperationStore is the durable boundary for replay-safe restore
// identity. Implementations may persist the operation outside the assembler.
type RestoreOperationStore interface {
	Lookup(handoffID string) (RestoreResult, bool, error)
	Save(handoffID string, result RestoreResult) error
}

func RestoreWithStore(h Handoff, resolver Resolver, store RestoreOperationStore) (RestoreResult, error) {
	if err := h.Validate(DefaultLimits()); err != nil {
		return RestoreResult{}, err
	}
	id := ID(h)
	if store != nil {
		if result, found, err := store.Lookup(id); err != nil {
			return RestoreResult{}, err
		} else if found {
			return result, nil
		}
	}
	result, err := Restore(h, resolver)
	if err != nil {
		return RestoreResult{}, err
	}
	if store != nil {
		if err := store.Save(id, result); err != nil {
			return RestoreResult{}, err
		}
	}
	return result, nil
}

func Restore(h Handoff, resolver Resolver) (RestoreResult, error) {
	if err := h.Validate(DefaultLimits()); err != nil {
		return RestoreResult{}, err
	}
	for _, ref := range h.References {
		if resolver == nil {
			return RestoreResult{}, ErrReferenceNotFound
		}
		if err := resolver.Resolve(ref); err != nil {
			return RestoreResult{}, fmt.Errorf("resolve %s/%s: %w", ref.Kind, ref.ID, err)
		}
	}
	return RestoreResult{HandoffID: ID(h), Restored: true}, nil
}

// ID returns the deterministic identity used to deduplicate restore attempts.
// Callers should validate the handoff before relying on the value.
func ID(h Handoff) string {
	payload, _ := json.Marshal(h.Stable())
	sum := sha256.Sum256(payload)
	return "handoff_" + hex.EncodeToString(sum[:8])
}

func (h Handoff) Stable() Handoff {
	out := h
	out.Completed = sortedCopy(h.Completed)
	out.Pending = sortedCopy(h.Pending)
	out.FailedAttempts = sortedCopy(h.FailedAttempts)
	out.FileChanges = sortedCopy(h.FileChanges)
	out.ToolResults = sortedCopy(h.ToolResults)
	return out
}

func sortedCopy(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}
