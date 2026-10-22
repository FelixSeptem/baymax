package composer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/orchestration/scheduler"
)

const (
	ChildContinuationVersion           = "child_continuation.v1"
	ChildContinuationMaxBytes          = 64 * 1024
	ChildContinuationMaxReferenceBytes = 1024
)

var (
	ErrChildContinuationMalformed        = errors.New("child continuation malformed")
	ErrChildContinuationCorrupt          = errors.New("child continuation corrupt")
	ErrChildContinuationForeign          = errors.New("child continuation foreign")
	ErrChildContinuationExpired          = errors.New("child continuation expired")
	ErrChildContinuationUnavailable      = errors.New("child continuation unavailable")
	ErrChildContinuationConflict         = errors.New("child continuation conflict")
	ErrChildContinuationInProgress       = errors.New("child continuation in progress")
	ErrChildContinuationRecoveryRequired = errors.New("child continuation recovery required")
	ErrChildContinuationUnsupported      = errors.New("child continuation unsupported")
)

type ChildPauseReason string

const (
	ChildPauseInputRequired    ChildPauseReason = "input_required"
	ChildPauseApprovalRequired ChildPauseReason = "approval_required"
	ChildPauseBudgetExhausted  ChildPauseReason = "budget_exhausted"
)

// ChildPauseError is returned by a source-owned child runner at a safe point.
// It carries no executable callback or business payload.
type ChildPauseError struct {
	Reason                 ChildPauseReason
	PendingActionReference string
	ConsumedBudget         int64
	OriginalDeadline       time.Time
}

func (e ChildPauseError) Error() string { return string(e.Reason) + ": child paused" }

func validChildPauseReason(reason ChildPauseReason) bool {
	switch reason {
	case ChildPauseInputRequired, ChildPauseApprovalRequired, ChildPauseBudgetExhausted:
		return true
	default:
		return false
	}
}

type ChildContinuationRef struct {
	SchemaVersion  string                `json:"schema_version"`
	CheckpointID   string                `json:"checkpoint_id"`
	Digest         string                `json:"digest"`
	OwnerNamespace string                `json:"owner_namespace"`
	GraphID        string                `json:"graph_id"`
	GraphVersion   string                `json:"graph_version"`
	RunID          string                `json:"run_id"`
	SessionID      string                `json:"session_id"`
	ParentRunID    string                `json:"parent_run_id,omitempty"`
	CausationID    string                `json:"causation_id,omitempty"`
	OccurrenceID   string                `json:"occurrence_id,omitempty"`
	Deployment     types.DeploymentScope `json:"deployment,omitempty"`
}

func (r ChildContinuationRef) Validate() error {
	if strings.TrimSpace(r.SchemaVersion) != ChildContinuationVersion || strings.TrimSpace(r.CheckpointID) == "" || strings.TrimSpace(r.Digest) == "" {
		return ErrChildContinuationMalformed
	}
	if strings.TrimSpace(r.OwnerNamespace) == "" || strings.TrimSpace(r.GraphID) == "" || strings.TrimSpace(r.GraphVersion) == "" || strings.TrimSpace(r.RunID) == "" {
		return ErrChildContinuationMalformed
	}
	for _, value := range []string{r.CheckpointID, r.Digest, r.OwnerNamespace, r.GraphID, r.GraphVersion, r.RunID, r.SessionID, r.ParentRunID, r.CausationID, r.OccurrenceID} {
		if len([]byte(strings.TrimSpace(value))) > ChildContinuationMaxReferenceBytes {
			return ErrChildContinuationMalformed
		}
	}
	return nil
}

type ChildContinuation struct {
	Ref                    ChildContinuationRef  `json:"ref"`
	Target                 ChildTarget           `json:"target"`
	Claimed                scheduler.ClaimedTask `json:"claimed"`
	GraphID                string                `json:"graph_id"`
	GraphVersion           string                `json:"graph_version"`
	GraphDigest            string                `json:"graph_digest"`
	ParentRunID            string                `json:"parent_run_id,omitempty"`
	RunID                  string                `json:"run_id"`
	SessionID              string                `json:"session_id,omitempty"`
	CausationID            string                `json:"causation_id,omitempty"`
	AuthorityDigest        string                `json:"authority_digest,omitempty"`
	PlanDigest             string                `json:"plan_digest,omitempty"`
	WriteScopeDigest       string                `json:"write_scope_digest,omitempty"`
	EffectNamespace        string                `json:"effect_namespace,omitempty"`
	OccurrenceID           string                `json:"occurrence_id,omitempty"`
	PauseReason            ChildPauseReason      `json:"pause_reason"`
	PendingActionReference string                `json:"pending_action_reference,omitempty"`
	ConsumedBudget         int64                 `json:"consumed_budget,omitempty"`
	OriginalDeadline       time.Time             `json:"original_deadline,omitempty"`
	CompletedEffectRefs    []string              `json:"completed_effect_refs,omitempty"`
	PredecessorID          string                `json:"predecessor_id,omitempty"`
	SuccessorID            string                `json:"successor_id,omitempty"`
	Generation             uint64                `json:"generation"`
	CreatedAt              time.Time             `json:"created_at"`
	ExpiresAt              time.Time             `json:"expires_at"`
}

func (c ChildContinuation) Validate() error {
	if c.Ref.SchemaVersion != ChildContinuationVersion || c.Ref.OwnerNamespace == "" || c.Ref.CheckpointID == "" || c.Ref.Digest == "" {
		return ErrChildContinuationMalformed
	}
	if err := c.Ref.Validate(); err != nil {
		return err
	}
	if c.Target != ChildTargetLocal && c.Target != ChildTargetA2A {
		return ErrChildContinuationMalformed
	}
	if c.GraphID == "" || c.GraphVersion == "" || c.GraphDigest == "" || c.RunID == "" || c.Claimed.Record.Task.TaskID == "" || c.Claimed.Attempt.AttemptID == "" {
		return ErrChildContinuationMalformed
	}
	if c.Ref.GraphID != c.GraphID || c.Ref.GraphVersion != c.GraphVersion || c.Ref.RunID != c.RunID || c.Ref.SessionID != c.SessionID || c.Ref.ParentRunID != c.ParentRunID || c.Ref.CausationID != c.CausationID || c.Ref.OccurrenceID != c.OccurrenceID {
		return ErrChildContinuationForeign
	}
	if !validChildPauseReason(c.PauseReason) {
		return ErrChildContinuationMalformed
	}
	if c.ConsumedBudget < 0 || len(c.CompletedEffectRefs) > 256 {
		return ErrChildContinuationMalformed
	}
	for _, value := range []string{c.GraphID, c.GraphVersion, c.GraphDigest, c.ParentRunID, c.RunID, c.SessionID, c.CausationID, c.AuthorityDigest, c.PlanDigest, c.WriteScopeDigest, c.EffectNamespace, c.OccurrenceID, c.PendingActionReference, c.PredecessorID, c.SuccessorID} {
		if len([]byte(strings.TrimSpace(value))) > ChildContinuationMaxReferenceBytes {
			return ErrChildContinuationMalformed
		}
	}
	if c.CreatedAt.IsZero() || c.ExpiresAt.IsZero() || !c.ExpiresAt.After(c.CreatedAt) {
		return ErrChildContinuationMalformed
	}
	payload, err := json.Marshal(c)
	if err != nil || len(payload) > ChildContinuationMaxBytes {
		return ErrChildContinuationMalformed
	}
	canonical, err := canonicalChildContinuation(c)
	if err != nil {
		return ErrChildContinuationCorrupt
	}
	sum := sha256.Sum256(canonical)
	if c.Ref.Digest != "sha256:"+hex.EncodeToString(sum[:]) || c.Ref.CheckpointID != "child-"+hex.EncodeToString(sum[:16]) {
		return ErrChildContinuationCorrupt
	}
	return nil
}

func canonicalChildContinuation(c ChildContinuation) ([]byte, error) {
	out := c
	out.Ref.CheckpointID = ""
	out.Ref.Digest = ""
	return json.Marshal(out)
}

func SealChildContinuation(in ChildContinuation) (ChildContinuation, error) {
	out := in
	out.Ref.SchemaVersion = ChildContinuationVersion
	out.Ref.OwnerNamespace = strings.TrimSpace(out.Ref.OwnerNamespace)
	out.Ref.GraphID = strings.TrimSpace(out.GraphID)
	out.Ref.GraphVersion = strings.TrimSpace(out.GraphVersion)
	out.Ref.RunID = strings.TrimSpace(out.RunID)
	out.Ref.SessionID = strings.TrimSpace(out.SessionID)
	out.Ref.ParentRunID = strings.TrimSpace(out.ParentRunID)
	out.Ref.CausationID = strings.TrimSpace(out.CausationID)
	out.Ref.OccurrenceID = strings.TrimSpace(out.OccurrenceID)
	out.GraphID = strings.TrimSpace(out.GraphID)
	out.GraphVersion = strings.TrimSpace(out.GraphVersion)
	out.GraphDigest = strings.TrimSpace(out.GraphDigest)
	out.ParentRunID = strings.TrimSpace(out.ParentRunID)
	out.RunID = strings.TrimSpace(out.RunID)
	out.SessionID = strings.TrimSpace(out.SessionID)
	out.CausationID = strings.TrimSpace(out.CausationID)
	out.AuthorityDigest = strings.TrimSpace(out.AuthorityDigest)
	out.PlanDigest = strings.TrimSpace(out.PlanDigest)
	out.WriteScopeDigest = strings.TrimSpace(out.WriteScopeDigest)
	out.EffectNamespace = strings.TrimSpace(out.EffectNamespace)
	out.OccurrenceID = strings.TrimSpace(out.OccurrenceID)
	out.PendingActionReference = strings.TrimSpace(out.PendingActionReference)
	out.PredecessorID = strings.TrimSpace(out.PredecessorID)
	out.SuccessorID = strings.TrimSpace(out.SuccessorID)
	out.CompletedEffectRefs = append([]string(nil), out.CompletedEffectRefs...)
	out.Ref.CheckpointID = ""
	out.Ref.Digest = ""
	canonical, err := canonicalChildContinuation(out)
	if err != nil || len(canonical) > ChildContinuationMaxBytes {
		return ChildContinuation{}, ErrChildContinuationMalformed
	}
	digest := sha256.Sum256(canonical)
	out.Ref.Digest = "sha256:" + hex.EncodeToString(digest[:])
	out.Ref.CheckpointID = "child-" + hex.EncodeToString(digest[:16])
	if err := out.Validate(); err != nil {
		return ChildContinuation{}, err
	}
	return out, nil
}

func (c ChildContinuation) Detached() ChildContinuation {
	out := c
	out.CompletedEffectRefs = append([]string(nil), c.CompletedEffectRefs...)
	out.Claimed.Record.Task.Payload = cloneMap(c.Claimed.Record.Task.Payload)
	out.Claimed.Record.Result = cloneMap(c.Claimed.Record.Result)
	return out
}

type ChildResumeRequest struct {
	Ref              ChildContinuationRef    `json:"ref"`
	GraphDigest      string                  `json:"graph_digest"`
	Generation       uint64                  `json:"generation"`
	Target           ChildTarget             `json:"target"`
	IdempotencyKey   string                  `json:"idempotency_key"`
	AuthorityDigest  string                  `json:"authority_digest,omitempty"`
	PlanDigest       string                  `json:"plan_digest,omitempty"`
	WriteScopeDigest string                  `json:"write_scope_digest,omitempty"`
	EffectNamespace  string                  `json:"effect_namespace,omitempty"`
	OccurrenceID     string                  `json:"occurrence_id,omitempty"`
	CausationID      string                  `json:"causation_id,omitempty"`
	Deployment       types.DeploymentRequest `json:"deployment,omitempty"`
}

func (r ChildResumeRequest) Validate() error {
	if err := r.Ref.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(r.GraphDigest) == "" || strings.TrimSpace(r.IdempotencyKey) == "" {
		return ErrChildContinuationMalformed
	}
	for _, value := range []string{r.GraphDigest, r.IdempotencyKey, r.AuthorityDigest, r.PlanDigest, r.WriteScopeDigest, r.EffectNamespace, r.OccurrenceID, r.CausationID} {
		if len([]byte(strings.TrimSpace(value))) > ChildContinuationMaxReferenceBytes {
			return ErrChildContinuationMalformed
		}
	}
	if r.Target != ChildTargetLocal && r.Target != ChildTargetA2A {
		return ErrChildContinuationMalformed
	}
	return nil
}

func (r ChildResumeRequest) identity() string {
	b, _ := json.Marshal(r)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

type ChildResumeClaim struct {
	Request    ChildResumeRequest   `json:"request"`
	AttemptID  string               `json:"attempt_id"`
	Generation uint64               `json:"generation"`
	State      string               `json:"state"`
	Duplicate  bool                 `json:"duplicate,omitempty"`
	Result     *ChildDispatchResult `json:"result,omitempty"`
}

type ChildContinuationStore interface {
	Save(context.Context, ChildContinuation) error
	Load(context.Context, ChildContinuationRef) (ChildContinuation, error)
	ClaimResume(context.Context, ChildResumeRequest) (ChildResumeClaim, error)
	MarkResumeRunning(context.Context, ChildResumeClaim) (ChildResumeClaim, error)
	CommitResumeResult(context.Context, ChildResumeClaim, ChildDispatchResult) (ChildResumeClaim, error)
}

const (
	childClaimedState   = "claimed"
	childRunningState   = "running"
	childCompletedState = "completed"
	childRecoveryState  = "recovery_required"
)

type childContinuationRecord struct {
	Continuation ChildContinuation `json:"continuation"`
	Claim        *ChildResumeClaim `json:"claim,omitempty"`
	Identity     string            `json:"identity,omitempty"`
}

type MemoryChildContinuationStore struct {
	mu        sync.Mutex
	namespace string
	records   map[string]childContinuationRecord
	now       func() time.Time
}

func (s *MemoryChildContinuationStore) DeploymentScope() types.DeploymentScope {
	if s == nil || strings.TrimSpace(s.namespace) == "" {
		return types.DefaultDeploymentScope("child", "child continuation store is unavailable")
	}
	return types.SameHostSingleWriterScope(s.namespace, "fixture:deployment-scope.v1")
}

func NewMemoryChildContinuationStore(namespace string) *MemoryChildContinuationStore {
	return &MemoryChildContinuationStore{namespace: strings.TrimSpace(namespace), records: map[string]childContinuationRecord{}, now: time.Now}
}

func (s *MemoryChildContinuationStore) Save(ctx context.Context, c ChildContinuation) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.namespace == "" {
		return ErrChildContinuationUnavailable
	}
	if c.Ref.OwnerNamespace != s.namespace {
		return ErrChildContinuationForeign
	}
	if err := c.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if old, ok := s.records[c.Ref.CheckpointID]; ok {
		if old.Continuation.Ref.Digest == c.Ref.Digest {
			return nil
		}
		return ErrChildContinuationConflict
	}
	s.records[c.Ref.CheckpointID] = childContinuationRecord{Continuation: c.Detached()}
	return nil
}

func (s *MemoryChildContinuationStore) Load(ctx context.Context, ref ChildContinuationRef) (ChildContinuation, error) {
	if err := ctx.Err(); err != nil {
		return ChildContinuation{}, err
	}
	if s == nil || s.namespace == "" {
		return ChildContinuation{}, ErrChildContinuationUnavailable
	}
	if err := ref.Validate(); err != nil {
		return ChildContinuation{}, err
	}
	if ref.OwnerNamespace != s.namespace {
		return ChildContinuation{}, ErrChildContinuationForeign
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.records[ref.CheckpointID]
	if !ok {
		return ChildContinuation{}, ErrChildContinuationUnavailable
	}
	if record.Continuation.Ref.Digest != ref.Digest {
		return ChildContinuation{}, ErrChildContinuationForeign
	}
	if !s.now().Before(record.Continuation.ExpiresAt) {
		return ChildContinuation{}, ErrChildContinuationExpired
	}
	if err := record.Continuation.Validate(); err != nil {
		return ChildContinuation{}, err
	}
	return record.Continuation.Detached(), nil
}

func (s *MemoryChildContinuationStore) ClaimResume(ctx context.Context, req ChildResumeRequest) (ChildResumeClaim, error) {
	if err := ctx.Err(); err != nil {
		return ChildResumeClaim{}, err
	}
	if s == nil || s.namespace == "" {
		return ChildResumeClaim{}, ErrChildContinuationUnavailable
	}
	if err := req.Validate(); err != nil {
		return ChildResumeClaim{}, err
	}
	if req.Ref.OwnerNamespace != s.namespace {
		return ChildResumeClaim{}, ErrChildContinuationForeign
	}
	if err := s.DeploymentScope().Check(req.Deployment); err != nil {
		return ChildResumeClaim{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.records[req.Ref.CheckpointID]
	if !ok {
		return ChildResumeClaim{}, ErrChildContinuationUnavailable
	}
	if record.Continuation.Ref.Digest != req.Ref.Digest || record.Continuation.GraphDigest != req.GraphDigest || record.Continuation.Generation != req.Generation || record.Continuation.Target != req.Target {
		return ChildResumeClaim{}, ErrChildContinuationConflict
	}
	if !s.now().Before(record.Continuation.ExpiresAt) {
		return ChildResumeClaim{}, ErrChildContinuationExpired
	}
	identity := req.identity()
	if record.Claim != nil {
		if record.Identity != identity {
			return ChildResumeClaim{}, ErrChildContinuationConflict
		}
		duplicate := *record.Claim
		duplicate.Duplicate = true
		if duplicate.Result == nil {
			if duplicate.State == childRecoveryState {
				return duplicate, ErrChildContinuationRecoveryRequired
			}
			return duplicate, ErrChildContinuationInProgress
		}
		return duplicate, nil
	}
	claim := ChildResumeClaim{Request: req, AttemptID: req.Ref.CheckpointID + ":attempt", Generation: req.Generation + 1, State: childClaimedState}
	record.Claim = &claim
	record.Identity = identity
	s.records[req.Ref.CheckpointID] = record
	return claim, nil
}

func (s *MemoryChildContinuationStore) MarkResumeRunning(ctx context.Context, claim ChildResumeClaim) (ChildResumeClaim, error) {
	if err := ctx.Err(); err != nil {
		return ChildResumeClaim{}, err
	}
	if s == nil {
		return ChildResumeClaim{}, ErrChildContinuationUnavailable
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.records[claim.Request.Ref.CheckpointID]
	if !ok || record.Claim == nil || record.Identity != claim.Request.identity() || record.Claim.AttemptID != claim.AttemptID {
		return ChildResumeClaim{}, ErrChildContinuationConflict
	}
	if record.Claim.Result != nil || record.Claim.State == childCompletedState {
		return ChildResumeClaim{}, ErrChildContinuationConflict
	}
	if record.Claim.State == childRecoveryState {
		return ChildResumeClaim{}, ErrChildContinuationRecoveryRequired
	}
	if record.Claim.State != childClaimedState && record.Claim.State != childRunningState {
		return ChildResumeClaim{}, ErrChildContinuationConflict
	}
	record.Claim.State = childRunningState
	s.records[claim.Request.Ref.CheckpointID] = record
	return *record.Claim, nil
}

func (s *MemoryChildContinuationStore) CommitResumeResult(ctx context.Context, claim ChildResumeClaim, result ChildDispatchResult) (ChildResumeClaim, error) {
	if err := ctx.Err(); err != nil {
		return ChildResumeClaim{}, err
	}
	if s == nil {
		return ChildResumeClaim{}, ErrChildContinuationUnavailable
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.records[claim.Request.Ref.CheckpointID]
	if !ok || record.Claim == nil || record.Claim.AttemptID != claim.AttemptID || record.Identity != claim.Request.identity() {
		return ChildResumeClaim{}, ErrChildContinuationConflict
	}
	if record.Claim.Result != nil {
		left, _ := json.Marshal(record.Claim.Result)
		right, _ := json.Marshal(result)
		if string(left) == string(right) {
			return *record.Claim, nil
		}
		return ChildResumeClaim{}, ErrChildContinuationConflict
	}
	if record.Claim.State != childClaimedState && record.Claim.State != childRunningState {
		return ChildResumeClaim{}, ErrChildContinuationRecoveryRequired
	}
	detached := result
	detached.Record = result.Record
	detached.Record.Task.Payload = cloneMap(result.Record.Task.Payload)
	detached.Record.Result = cloneMap(result.Record.Result)
	detached.Commit.Result = cloneMap(result.Commit.Result)
	record.Claim.Result = &detached
	record.Claim.State = childCompletedState
	s.records[claim.Request.Ref.CheckpointID] = record
	return *record.Claim, nil
}

type FileChildContinuationStore struct {
	mu        sync.Mutex
	root      string
	namespace string
	memory    *MemoryChildContinuationStore
}

func NewFileChildContinuationStore(root, namespace string) (*FileChildContinuationStore, error) {
	if strings.TrimSpace(root) == "" || strings.TrimSpace(namespace) == "" {
		return nil, ErrChildContinuationUnavailable
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, ErrChildContinuationUnavailable
	}
	return &FileChildContinuationStore{root: root, namespace: strings.TrimSpace(namespace), memory: NewMemoryChildContinuationStore(namespace)}, nil
}

func (s *FileChildContinuationStore) path(id string) string {
	clean := strings.NewReplacer("/", "_", "\\", "_", ":", "_").Replace(id)
	return filepath.Join(s.root, clean+".json")
}

func (s *FileChildContinuationStore) persist(id string, record childContinuationRecord) error {
	data, err := json.Marshal(record)
	if err != nil {
		return ErrChildContinuationCorrupt
	}
	tmp, err := os.CreateTemp(s.root, "child-continuation-*.tmp")
	if err != nil {
		return ErrChildContinuationUnavailable
	}
	name := tmp.Name()
	defer func() { _ = os.Remove(name) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return ErrChildContinuationUnavailable
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return ErrChildContinuationUnavailable
	}
	if err := tmp.Close(); err != nil {
		return ErrChildContinuationUnavailable
	}
	if err := os.Rename(name, s.path(id)); err != nil {
		return ErrChildContinuationUnavailable
	}
	return nil
}

func (s *FileChildContinuationStore) hydrate(ref ChildContinuationRef) error {
	raw, err := os.ReadFile(s.path(ref.CheckpointID))
	if err != nil {
		return ErrChildContinuationUnavailable
	}
	var record childContinuationRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return ErrChildContinuationCorrupt
	}
	if err := record.Continuation.Validate(); err != nil {
		return err
	}
	if record.Continuation.Ref.OwnerNamespace != s.namespace || record.Continuation.Ref.Digest != ref.Digest {
		return ErrChildContinuationForeign
	}
	if record.Claim != nil && record.Claim.Result == nil && record.Claim.State == childClaimedState {
		record.Claim.State = childRecoveryState
	}
	s.memory.records[ref.CheckpointID] = record
	return nil
}

func (s *FileChildContinuationStore) Save(ctx context.Context, c ChildContinuation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.memory.Save(ctx, c); err != nil {
		return err
	}
	if err := s.persist(c.Ref.CheckpointID, s.memory.records[c.Ref.CheckpointID]); err != nil {
		delete(s.memory.records, c.Ref.CheckpointID)
		return err
	}
	return nil
}

func (s *FileChildContinuationStore) Load(ctx context.Context, ref ChildContinuationRef) (ChildContinuation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.hydrate(ref); err != nil {
		return ChildContinuation{}, err
	}
	return s.memory.Load(ctx, ref)
}

func (s *FileChildContinuationStore) ClaimResume(ctx context.Context, req ChildResumeRequest) (ChildResumeClaim, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.memory.records[req.Ref.CheckpointID]; !ok {
		if err := s.hydrate(req.Ref); err != nil {
			return ChildResumeClaim{}, err
		}
	}
	before := s.memory.records[req.Ref.CheckpointID]
	claim, err := s.memory.ClaimResume(ctx, req)
	if err != nil {
		return claim, err
	}
	if err := s.persist(req.Ref.CheckpointID, s.memory.records[req.Ref.CheckpointID]); err != nil {
		s.memory.records[req.Ref.CheckpointID] = before
		return ChildResumeClaim{}, err
	}
	return claim, nil
}

func (s *FileChildContinuationStore) MarkResumeRunning(ctx context.Context, claim ChildResumeClaim) (ChildResumeClaim, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.memory.records[claim.Request.Ref.CheckpointID]; !ok {
		if err := s.hydrate(claim.Request.Ref); err != nil {
			return ChildResumeClaim{}, err
		}
	}
	before := s.memory.records[claim.Request.Ref.CheckpointID]
	running, err := s.memory.MarkResumeRunning(ctx, claim)
	if err != nil {
		return running, err
	}
	if err := s.persist(claim.Request.Ref.CheckpointID, s.memory.records[claim.Request.Ref.CheckpointID]); err != nil {
		s.memory.records[claim.Request.Ref.CheckpointID] = before
		return ChildResumeClaim{}, err
	}
	return running, nil
}

func (s *FileChildContinuationStore) CommitResumeResult(ctx context.Context, claim ChildResumeClaim, result ChildDispatchResult) (ChildResumeClaim, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.memory.records[claim.Request.Ref.CheckpointID]; !ok {
		if err := s.hydrate(claim.Request.Ref); err != nil {
			return ChildResumeClaim{}, err
		}
	}
	before := s.memory.records[claim.Request.Ref.CheckpointID]
	committed, err := s.memory.CommitResumeResult(ctx, claim, result)
	if err != nil {
		return ChildResumeClaim{}, err
	}
	if err := s.persist(claim.Request.Ref.CheckpointID, s.memory.records[claim.Request.Ref.CheckpointID]); err != nil {
		s.memory.records[claim.Request.Ref.CheckpointID] = before
		return ChildResumeClaim{}, err
	}
	return committed, nil
}
