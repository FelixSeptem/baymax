package types

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const SessionHistoryCheckpointReplayVersionV1 = "session_history_checkpoint_replay.v1"

const (
	SessionHistoryReasonRequired              = "session.history_required"
	SessionHistoryReasonGap                   = "session.history_gap"
	SessionHistoryReasonConflict              = "session.history_conflict"
	SessionHistoryReasonSizeLimit             = "session.history_size_limit"
	SessionHistoryReasonBranchParentMissing   = "session.branch_parent_missing"
	SessionHistoryReasonBranchRunConflict     = "session.branch_run_conflict"
	SessionHistoryReasonCheckpointAssociation = "session.checkpoint_association_mismatch"
	SessionHistoryReasonReplayConflict        = "session.replay_conflict"
	SessionHistoryReasonReplaySideEffect      = "session.replay_side_effect"
	SessionHistoryReasonUnsupportedVersion    = "session.history_unsupported_version"
)

const (
	RestoreModeStrict     = "strict"
	RestoreModeCompatible = "compatible"
)

// HistoryEntry is a bounded reference to one source-owned history position.
// It intentionally does not carry message bodies.
type HistoryEntry struct {
	ID               string `json:"id"`
	ParentID         string `json:"parent_id,omitempty"`
	Position         int64  `json:"position"`
	Digest           string `json:"digest,omitempty"`
	ProducedByRunID  string `json:"produced_by_run_id,omitempty"`
	ProducedByStepID string `json:"produced_by_step_id,omitempty"`
}

type HistoryValidationLimits struct {
	MaxEntries         int `json:"max_entries,omitempty"`
	MaxSerializedBytes int `json:"max_serialized_bytes,omitempty"`
}

// SessionHistoryBoundary is a reference-only, source-owned history projection.
type SessionHistoryBoundary struct {
	Version   string            `json:"version"`
	SessionID string            `json:"session_id"`
	Root      HistoryEntry      `json:"root"`
	Entries   []HistoryEntry    `json:"entries"`
	LeafID    string            `json:"leaf_id"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

func (s SessionHistoryBoundary) Validate(limits HistoryValidationLimits) error {
	if strings.TrimSpace(s.Version) != SessionHistoryCheckpointReplayVersionV1 {
		return fmt.Errorf("%s: version must be %q", SessionHistoryReasonUnsupportedVersion, SessionHistoryCheckpointReplayVersionV1)
	}
	if strings.TrimSpace(s.SessionID) == "" || strings.TrimSpace(s.Root.ID) == "" || strings.TrimSpace(s.LeafID) == "" {
		return fmt.Errorf("%s: session_id, root.id and leaf_id are required", SessionHistoryReasonRequired)
	}
	maxEntries := limits.MaxEntries
	if maxEntries <= 0 {
		maxEntries = 256
	}
	if len(s.Entries) == 0 || len(s.Entries) > maxEntries {
		return fmt.Errorf("%s: entries count=%d limit=%d", SessionHistoryReasonSizeLimit, len(s.Entries), maxEntries)
	}
	seen := make(map[string]HistoryEntry, len(s.Entries))
	entries := append([]HistoryEntry(nil), s.Entries...)
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].Position < entries[j].Position })
	for i, entry := range entries {
		entry.ID = strings.TrimSpace(entry.ID)
		entry.ParentID = strings.TrimSpace(entry.ParentID)
		if entry.ID == "" || entry.Position < 0 {
			return fmt.Errorf("%s: entries[%d] id and non-negative position are required", SessionHistoryReasonRequired, i)
		}
		if previous, ok := seen[entry.ID]; ok {
			if previous != entry {
				return fmt.Errorf("%s: duplicate id %q has conflicting data", SessionHistoryReasonConflict, entry.ID)
			}
			return fmt.Errorf("%s: duplicate id %q", SessionHistoryReasonConflict, entry.ID)
		}
		seen[entry.ID] = entry
		if i == 0 {
			if entry.ID != strings.TrimSpace(s.Root.ID) || entry.Position != s.Root.Position {
				return fmt.Errorf("%s: root does not match first history entry", SessionHistoryReasonGap)
			}
			if entry.ParentID != "" {
				return fmt.Errorf("%s: root parent must be empty", SessionHistoryReasonGap)
			}
			continue
		}
		previous := entries[i-1]
		if entry.Position != previous.Position+1 || entry.ParentID != previous.ID {
			return fmt.Errorf("%s: entry %q does not continue parent=%q position=%d", SessionHistoryReasonGap, entry.ID, previous.ID, entry.Position)
		}
	}
	if _, ok := seen[strings.TrimSpace(s.LeafID)]; !ok {
		return fmt.Errorf("%s: leaf_id %q is not present", SessionHistoryReasonGap, s.LeafID)
	}
	if maxBytes := limits.MaxSerializedBytes; maxBytes > 0 {
		raw, err := json.Marshal(s)
		if err != nil {
			return fmt.Errorf("%s: marshal projection: %w", SessionHistoryReasonSizeLimit, err)
		}
		if len(raw) > maxBytes {
			return fmt.Errorf("%s: serialized bytes=%d limit=%d", SessionHistoryReasonSizeLimit, len(raw), maxBytes)
		}
	}
	return nil
}

func (s SessionHistoryBoundary) NormalizedDigest() string {
	normalized := s
	normalized.Version = strings.TrimSpace(normalized.Version)
	normalized.SessionID = strings.TrimSpace(normalized.SessionID)
	normalized.LeafID = strings.TrimSpace(normalized.LeafID)
	normalized.Entries = append([]HistoryEntry(nil), normalized.Entries...)
	sort.SliceStable(normalized.Entries, func(i, j int) bool { return normalized.Entries[i].Position < normalized.Entries[j].Position })
	raw, err := json.Marshal(normalized)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

type BranchProjection struct {
	SessionID          string `json:"session_id"`
	BranchID           string `json:"branch_id"`
	ParentLeafID       string `json:"parent_leaf_id"`
	ParentRunID        string `json:"parent_run_id,omitempty"`
	ParentCheckpointID string `json:"parent_checkpoint_id,omitempty"`
	RunID              string `json:"run_id"`
}

func (b BranchProjection) Validate() error {
	if strings.TrimSpace(b.SessionID) == "" || strings.TrimSpace(b.BranchID) == "" || strings.TrimSpace(b.ParentLeafID) == "" || strings.TrimSpace(b.RunID) == "" {
		return fmt.Errorf("%s: session_id, branch_id, parent_leaf_id and run_id are required", SessionHistoryReasonBranchParentMissing)
	}
	if strings.TrimSpace(b.ParentRunID) != "" && strings.TrimSpace(b.ParentRunID) == strings.TrimSpace(b.RunID) {
		return fmt.Errorf("%s: branch run must differ from parent run", SessionHistoryReasonBranchRunConflict)
	}
	return nil
}

type RestoreOperation struct {
	OperationID   string `json:"operation_id"`
	Mode          string `json:"mode"`
	SessionID     string `json:"session_id,omitempty"`
	HistoryLeafID string `json:"history_leaf_id,omitempty"`
	CheckpointID  string `json:"checkpoint_id,omitempty"`
	InputDigest   string `json:"input_digest"`
}

func (r RestoreOperation) Validate() error {
	if strings.TrimSpace(r.OperationID) == "" || strings.TrimSpace(r.InputDigest) == "" {
		return fmt.Errorf("%s: operation_id and input_digest are required", SessionHistoryReasonRequired)
	}
	if r.Mode != RestoreModeStrict && r.Mode != RestoreModeCompatible {
		return fmt.Errorf("%s: unsupported restore mode %q", SessionHistoryReasonRequired, r.Mode)
	}
	return nil
}

type ReplayOperation struct {
	OperationID    string `json:"operation_id"`
	FixtureVersion string `json:"fixture_version"`
	InputDigest    string `json:"input_digest"`
	SideEffectFree bool   `json:"side_effect_free"`
}

func (r ReplayOperation) Validate() error {
	if strings.TrimSpace(r.OperationID) == "" || strings.TrimSpace(r.FixtureVersion) == "" || strings.TrimSpace(r.InputDigest) == "" {
		return fmt.Errorf("%s: operation_id, fixture_version and input_digest are required", SessionHistoryReasonRequired)
	}
	if !r.SideEffectFree {
		return fmt.Errorf("%s: replay must be side-effect-free", SessionHistoryReasonReplaySideEffect)
	}
	return nil
}
