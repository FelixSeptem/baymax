package composer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/FelixSeptem/baymax/orchestration/scheduler"
	"github.com/FelixSeptem/baymax/orchestration/workflow"
)

const (
	RecoveryReasonRestore  = "recovery.restore"
	RecoveryReasonReplay   = "recovery.replay"
	RecoveryReasonConflict = "recovery.conflict"

	RecoverySnapshotVersion = "a9.v1"
)

type RecoveryErrorCode string

const (
	RecoveryErrorSnapshotCorrupt   RecoveryErrorCode = "recovery.snapshot_corrupt"
	RecoveryErrorSnapshotNotFound  RecoveryErrorCode = "recovery.snapshot_not_found"
	RecoveryErrorConflict          RecoveryErrorCode = "recovery.conflict"
	RecoveryErrorPolicyUnsupported RecoveryErrorCode = "recovery.policy_unsupported"
	RecoveryErrorStoreUnavailable  RecoveryErrorCode = "recovery.store_unavailable"
)

type RecoveryError struct {
	Code    RecoveryErrorCode
	Message string
	Cause   error
}

func (e *RecoveryError) Error() string {
	if e == nil {
		return ""
	}
	base := strings.TrimSpace(e.Message)
	if base == "" {
		base = string(e.Code)
	}
	if e.Cause == nil {
		return base
	}
	return base + ": " + e.Cause.Error()
}

func (e *RecoveryError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func IsRecoveryErrorCode(err error, code RecoveryErrorCode) bool {
	if strings.TrimSpace(string(code)) == "" || err == nil {
		return false
	}
	var target *RecoveryError
	if !errors.As(err, &target) || target == nil {
		return false
	}
	return target.Code == code
}

func newRecoveryError(code RecoveryErrorCode, message string, cause error) *RecoveryError {
	return &RecoveryError{
		Code:    code,
		Message: strings.TrimSpace(message),
		Cause:   cause,
	}
}

type RecoveryReplayCursor struct {
	Sequence            int64 `json:"sequence"`
	TerminalCommitCount int   `json:"terminal_commit_count"`
}

type RecoveryRunSnapshot struct {
	RunID  string `json:"run_id"`
	Status string `json:"status,omitempty"`
}

type RecoveryWorkflowSnapshot struct {
	Checkpoint workflow.Checkpoint `json:"checkpoint,omitempty"`
}

type RecoveryA2AInFlightState struct {
	TaskID         string `json:"task_id"`
	AttemptID      string `json:"attempt_id,omitempty"`
	WorkflowID     string `json:"workflow_id,omitempty"`
	TeamID         string `json:"team_id,omitempty"`
	AgentID        string `json:"agent_id,omitempty"`
	PeerID         string `json:"peer_id,omitempty"`
	TaskState      string `json:"task_state,omitempty"`
	CorrelationKey string `json:"correlation_key,omitempty"`
}

type RecoveryA2ASnapshot struct {
	InFlight []RecoveryA2AInFlightState `json:"in_flight,omitempty"`
}

type RecoveryRealtimeInteractionState struct {
	SessionID      string `json:"session_id,omitempty"`
	ResumeCursor   string `json:"resume_cursor,omitempty"`
	EventSeqMax    int64  `json:"event_seq_max,omitempty"`
	InterruptTotal int    `json:"interrupt_total,omitempty"`
	ResumeTotal    int    `json:"resume_total,omitempty"`
	ResumeSource   string `json:"resume_source,omitempty"`
}

type RecoveryIsolateHandoffState struct {
	Detected         bool   `json:"detected,omitempty"`
	Stage2ReasonCode string `json:"stage2_reason_code,omitempty"`
	Stage2Reason     string `json:"stage2_reason,omitempty"`
	Stage2SkipReason string `json:"stage2_skip_reason,omitempty"`
}

type RecoveryInteractionState struct {
	Realtime       RecoveryRealtimeInteractionState `json:"realtime,omitempty"`
	IsolateHandoff RecoveryIsolateHandoffState      `json:"isolate_handoff,omitempty"`
}

type RecoverySnapshot struct {
	Version        string                   `json:"version"`
	UpdatedAt      time.Time                `json:"updated_at"`
	Run            RecoveryRunSnapshot      `json:"run"`
	Workflow       RecoveryWorkflowSnapshot `json:"workflow,omitempty"`
	Scheduler      scheduler.StoreSnapshot  `json:"scheduler"`
	A2A            RecoveryA2ASnapshot      `json:"a2a,omitempty"`
	Interaction    RecoveryInteractionState `json:"interaction,omitempty"`
	Replay         RecoveryReplayCursor     `json:"replay"`
	ConflictPolicy string                   `json:"conflict_policy"`
}

type RecoveryStore interface {
	Backend() string
	Save(ctx context.Context, snapshot RecoverySnapshot) error
	Load(ctx context.Context, runID string) (RecoverySnapshot, bool, error)
}

type MemoryRecoveryStore struct {
	mu   sync.Mutex
	data map[string]RecoverySnapshot
}

func NewMemoryRecoveryStore() *MemoryRecoveryStore {
	return &MemoryRecoveryStore{
		data: map[string]RecoverySnapshot{},
	}
}

func (s *MemoryRecoveryStore) Backend() string {
	return "memory"
}

func (s *MemoryRecoveryStore) Save(_ context.Context, snapshot RecoverySnapshot) error {
	if s == nil {
		return newRecoveryError(RecoveryErrorStoreUnavailable, "recovery memory store is nil", nil)
	}
	normalized, err := normalizeRecoverySnapshot(snapshot, "")
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[normalized.Run.RunID] = normalized
	return nil
}

func (s *MemoryRecoveryStore) Load(_ context.Context, runID string) (RecoverySnapshot, bool, error) {
	if s == nil {
		return RecoverySnapshot{}, false, newRecoveryError(RecoveryErrorStoreUnavailable, "recovery memory store is nil", nil)
	}
	key := strings.TrimSpace(runID)
	if key == "" {
		return RecoverySnapshot{}, false, newRecoveryError(RecoveryErrorSnapshotNotFound, "recovery run_id is required", nil)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	snapshot, ok := s.data[key]
	if !ok {
		return RecoverySnapshot{}, false, nil
	}
	normalized, err := normalizeRecoverySnapshot(snapshot, key)
	if err != nil {
		return RecoverySnapshot{}, false, err
	}
	return normalized, true, nil
}

type FileRecoveryStore struct {
	mu             sync.Mutex
	root           string
	persist        fileRecoveryStoreOptions
	pending        map[string]RecoverySnapshot
	pendingPersist int
	dirtySince     time.Time
}

type FileRecoveryStoreOption func(*fileRecoveryStoreOptions)

type fileRecoveryStoreOptions struct {
	PersistDebounce  time.Duration
	PersistBatchSize int
}

func WithRecoveryPersistDebounce(debounce time.Duration) FileRecoveryStoreOption {
	return func(opts *fileRecoveryStoreOptions) {
		if opts == nil {
			return
		}
		opts.PersistDebounce = debounce
	}
}

func WithRecoveryPersistBatchSize(size int) FileRecoveryStoreOption {
	return func(opts *fileRecoveryStoreOptions) {
		if opts == nil {
			return
		}
		opts.PersistBatchSize = size
	}
}

func normalizeFileRecoveryStoreOptions(opts []FileRecoveryStoreOption) fileRecoveryStoreOptions {
	normalized := fileRecoveryStoreOptions{
		PersistDebounce:  0,
		PersistBatchSize: 1,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&normalized)
		}
	}
	if normalized.PersistDebounce < 0 {
		normalized.PersistDebounce = 0
	}
	if normalized.PersistBatchSize <= 0 {
		normalized.PersistBatchSize = 1
	}
	return normalized
}

func (o fileRecoveryStoreOptions) batchingEnabled() bool {
	return o.PersistDebounce > 0 || o.PersistBatchSize > 1
}

func NewFileRecoveryStore(root string, opts ...FileRecoveryStoreOption) (*FileRecoveryStore, error) {
	path := strings.TrimSpace(root)
	if path == "" {
		return nil, fmt.Errorf("recovery file backend path is required")
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir recovery backend directory: %w", err)
	}
	return &FileRecoveryStore{
		root:    path,
		persist: normalizeFileRecoveryStoreOptions(opts),
		pending: map[string]RecoverySnapshot{},
	}, nil
}

func (s *FileRecoveryStore) Backend() string {
	return "file"
}

func (s *FileRecoveryStore) Save(_ context.Context, snapshot RecoverySnapshot) error {
	if s == nil {
		return newRecoveryError(RecoveryErrorStoreUnavailable, "recovery file store is nil", nil)
	}
	normalized, err := normalizeRecoverySnapshot(snapshot, "")
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.queueAndMaybePersistLocked(normalized, false)
}

func (s *FileRecoveryStore) Load(_ context.Context, runID string) (RecoverySnapshot, bool, error) {
	if s == nil {
		return RecoverySnapshot{}, false, newRecoveryError(RecoveryErrorStoreUnavailable, "recovery file store is nil", nil)
	}
	key := strings.TrimSpace(runID)
	if key == "" {
		return RecoverySnapshot{}, false, newRecoveryError(RecoveryErrorSnapshotNotFound, "recovery run_id is required", nil)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if pending, ok := s.pending[key]; ok {
		normalized, normalizeErr := normalizeRecoverySnapshot(pending, key)
		if normalizeErr != nil {
			return RecoverySnapshot{}, false, normalizeErr
		}
		return normalized, true, nil
	}
	raw, err := os.ReadFile(s.filePath(key))
	if errors.Is(err, os.ErrNotExist) {
		return RecoverySnapshot{}, false, nil
	}
	if err != nil {
		return RecoverySnapshot{}, false, fmt.Errorf("read recovery snapshot: %w", err)
	}
	var snapshot RecoverySnapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return RecoverySnapshot{}, false, newRecoveryError(RecoveryErrorSnapshotCorrupt, "decode recovery snapshot", err)
	}
	normalized, normalizeErr := normalizeRecoverySnapshot(snapshot, key)
	if normalizeErr != nil {
		return RecoverySnapshot{}, false, normalizeErr
	}
	return normalized, true, nil
}

// Flush forces pending batched snapshots to be durably persisted.
// It defines the explicit durability boundary when debounce/group-commit is enabled.
func (s *FileRecoveryStore) Flush() error {
	if s == nil {
		return newRecoveryError(RecoveryErrorStoreUnavailable, "recovery file store is nil", nil)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.flushPersistLocked()
}

func (s *FileRecoveryStore) filePath(runID string) string {
	key := strings.ToLower(strings.TrimSpace(runID))
	key = strings.ReplaceAll(key, " ", "_")
	key = strings.ReplaceAll(key, "/", "_")
	key = strings.ReplaceAll(key, "\\", "_")
	return filepath.Join(s.root, key+".json")
}

func (s *FileRecoveryStore) queueAndMaybePersistLocked(snapshot RecoverySnapshot, force bool) error {
	if !s.persist.batchingEnabled() {
		return s.writeSnapshotLocked(snapshot)
	}
	if s.pending == nil {
		s.pending = map[string]RecoverySnapshot{}
	}
	s.pending[snapshot.Run.RunID] = snapshot
	s.pendingPersist++
	if s.dirtySince.IsZero() {
		s.dirtySince = time.Now()
	}
	if force {
		return s.flushPersistLocked()
	}
	if s.pendingPersist >= s.persist.PersistBatchSize {
		return s.flushPersistLocked()
	}
	if s.persist.PersistDebounce > 0 && time.Since(s.dirtySince) >= s.persist.PersistDebounce {
		return s.flushPersistLocked()
	}
	return nil
}

func (s *FileRecoveryStore) flushPersistLocked() error {
	if s.persist.batchingEnabled() && len(s.pending) == 0 {
		return nil
	}
	if !s.persist.batchingEnabled() {
		return nil
	}
	for _, snapshot := range s.pending {
		if err := s.writeSnapshotLocked(snapshot); err != nil {
			return err
		}
	}
	clear(s.pending)
	s.pendingPersist = 0
	s.dirtySince = time.Time{}
	return nil
}

func (s *FileRecoveryStore) writeSnapshotLocked(snapshot RecoverySnapshot) error {
	raw, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return newRecoveryError(RecoveryErrorSnapshotCorrupt, "encode recovery snapshot", err)
	}
	if err := os.MkdirAll(s.root, 0o755); err != nil {
		return fmt.Errorf("mkdir recovery backend directory: %w", err)
	}
	file := s.filePath(snapshot.Run.RunID)
	tmp := file + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return fmt.Errorf("write recovery snapshot temp file: %w", err)
	}
	if err := os.Rename(tmp, file); err != nil {
		return fmt.Errorf("commit recovery snapshot file: %w", err)
	}
	return nil
}

func normalizeRecoverySnapshot(snapshot RecoverySnapshot, expectedRunID string) (RecoverySnapshot, error) {
	out := snapshot
	out.Version = strings.TrimSpace(out.Version)
	if out.Version == "" {
		out.Version = RecoverySnapshotVersion
	}
	if out.Version != RecoverySnapshotVersion {
		return RecoverySnapshot{}, newRecoveryError(
			RecoveryErrorSnapshotCorrupt,
			fmt.Sprintf("unsupported recovery snapshot version %q", out.Version),
			nil,
		)
	}
	out.Run.RunID = strings.TrimSpace(out.Run.RunID)
	if strings.TrimSpace(expectedRunID) != "" && out.Run.RunID != strings.TrimSpace(expectedRunID) {
		return RecoverySnapshot{}, newRecoveryError(
			RecoveryErrorSnapshotCorrupt,
			fmt.Sprintf("snapshot run_id mismatch: got=%q want=%q", out.Run.RunID, strings.TrimSpace(expectedRunID)),
			nil,
		)
	}
	if out.Run.RunID == "" {
		return RecoverySnapshot{}, newRecoveryError(RecoveryErrorSnapshotCorrupt, "recovery snapshot requires run_id", nil)
	}
	if out.UpdatedAt.IsZero() {
		out.UpdatedAt = time.Now()
	}
	out.ConflictPolicy = strings.ToLower(strings.TrimSpace(out.ConflictPolicy))
	if out.ConflictPolicy == "" {
		out.ConflictPolicy = "fail_fast"
	}
	if out.ConflictPolicy != "fail_fast" {
		return RecoverySnapshot{}, newRecoveryError(
			RecoveryErrorPolicyUnsupported,
			fmt.Sprintf("unsupported recovery conflict_policy %q", out.ConflictPolicy),
			nil,
		)
	}

	sort.Slice(out.Scheduler.Tasks, func(i, j int) bool {
		return strings.TrimSpace(out.Scheduler.Tasks[i].Task.TaskID) < strings.TrimSpace(out.Scheduler.Tasks[j].Task.TaskID)
	})
	out.Scheduler.Queue = normalizeQueue(out.Scheduler.Queue)
	sort.Slice(out.Scheduler.TerminalCommits, func(i, j int) bool {
		left := strings.TrimSpace(out.Scheduler.TerminalCommits[i].TaskID) + "|" + strings.TrimSpace(out.Scheduler.TerminalCommits[i].AttemptID)
		right := strings.TrimSpace(out.Scheduler.TerminalCommits[j].TaskID) + "|" + strings.TrimSpace(out.Scheduler.TerminalCommits[j].AttemptID)
		return left < right
	})
	sort.Slice(out.A2A.InFlight, func(i, j int) bool {
		left := strings.TrimSpace(out.A2A.InFlight[i].TaskID) + "|" + strings.TrimSpace(out.A2A.InFlight[i].AttemptID)
		right := strings.TrimSpace(out.A2A.InFlight[j].TaskID) + "|" + strings.TrimSpace(out.A2A.InFlight[j].AttemptID)
		return left < right
	})
	out.Interaction.Realtime.SessionID = strings.TrimSpace(out.Interaction.Realtime.SessionID)
	out.Interaction.Realtime.ResumeCursor = strings.TrimSpace(out.Interaction.Realtime.ResumeCursor)
	out.Interaction.Realtime.ResumeSource = strings.ToLower(strings.TrimSpace(out.Interaction.Realtime.ResumeSource))
	if out.Interaction.Realtime.EventSeqMax < 0 {
		out.Interaction.Realtime.EventSeqMax = 0
	}
	if out.Interaction.Realtime.InterruptTotal < 0 {
		out.Interaction.Realtime.InterruptTotal = 0
	}
	if out.Interaction.Realtime.ResumeTotal < 0 {
		out.Interaction.Realtime.ResumeTotal = 0
	}
	out.Interaction.IsolateHandoff.Stage2ReasonCode = strings.TrimSpace(out.Interaction.IsolateHandoff.Stage2ReasonCode)
	out.Interaction.IsolateHandoff.Stage2Reason = strings.TrimSpace(out.Interaction.IsolateHandoff.Stage2Reason)
	out.Interaction.IsolateHandoff.Stage2SkipReason = strings.TrimSpace(out.Interaction.IsolateHandoff.Stage2SkipReason)
	if !out.Interaction.IsolateHandoff.Detected {
		out.Interaction.IsolateHandoff.Detected = containsIsolateHandoffMarker(
			out.Interaction.IsolateHandoff.Stage2ReasonCode,
			out.Interaction.IsolateHandoff.Stage2Reason,
			out.Interaction.IsolateHandoff.Stage2SkipReason,
		)
	}

	replayCount := out.Replay.TerminalCommitCount
	if replayCount < 0 {
		replayCount = 0
	}
	out.Replay.TerminalCommitCount = replayCount
	if out.Replay.Sequence < 0 {
		out.Replay.Sequence = 0
	}
	return out, nil
}

func normalizeQueue(queue []string) []string {
	if len(queue) == 0 {
		return nil
	}
	out := make([]string, 0, len(queue))
	seen := map[string]struct{}{}
	for _, raw := range queue {
		key := strings.TrimSpace(raw)
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out
}

func containsIsolateHandoffMarker(markers ...string) bool {
	for i := range markers {
		if strings.Contains(strings.ToLower(strings.TrimSpace(markers[i])), "isolate_handoff") {
			return true
		}
	}
	return false
}
