package teams

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/FelixSeptem/baymax/context/hostprojection"
	"github.com/FelixSeptem/baymax/core/types"
)

const (
	TeamContinuationVersion    = "team_continuation.v1"
	teamContinuationMaxBytes   = 64 * 1024
	teamContinuationMaxEntries = 1024
)

var (
	ErrTeamContinuationMalformed        = errors.New("team continuation malformed")
	ErrTeamContinuationCorrupt          = errors.New("team continuation corrupt")
	ErrTeamContinuationForeign          = errors.New("team continuation foreign")
	ErrTeamContinuationExpired          = errors.New("team continuation expired")
	ErrTeamContinuationUnavailable      = errors.New("team continuation unavailable")
	ErrTeamContinuationConflict         = errors.New("team continuation conflict")
	ErrTeamContinuationInProgress       = errors.New("team continuation in progress")
	ErrTeamContinuationRecoveryRequired = errors.New("team continuation recovery required")
	ErrTeamContinuationUnsupported      = errors.New("team continuation unsupported")
)

type TeamPauseReason string

const (
	TeamPauseInputRequired    TeamPauseReason = "input_required"
	TeamPauseApprovalRequired TeamPauseReason = "approval_required"
	TeamPauseBudgetExhausted  TeamPauseReason = "budget_exhausted"
)

func validTeamPauseReason(reason TeamPauseReason) bool {
	return reason == TeamPauseInputRequired || reason == TeamPauseApprovalRequired || reason == TeamPauseBudgetExhausted
}

// TeamContinuationRef is an opaque source reference. It contains no member
// payload, callback, business commit, or executable runner state.
type TeamContinuationRef struct {
	SchemaVersion  string                `json:"schema_version"`
	CheckpointID   string                `json:"checkpoint_id"`
	Digest         string                `json:"digest"`
	OwnerNamespace string                `json:"owner_namespace"`
	TeamID         string                `json:"team_id"`
	GraphVersion   string                `json:"graph_version"`
	RunID          string                `json:"run_id"`
	SessionID      string                `json:"session_id"`
	ParentRunID    string                `json:"parent_run_id,omitempty"`
	CausationID    string                `json:"causation_id,omitempty"`
	Deployment     types.DeploymentScope `json:"deployment,omitempty"`
}

type TeamTaskContinuation struct {
	TaskID          string     `json:"task_id"`
	Status          TaskStatus `json:"status"`
	Attempts        int        `json:"attempts,omitempty"`
	Vote            string     `json:"vote,omitempty"`
	EffectReference string     `json:"effect_reference,omitempty"`
	OutputDigest    string     `json:"output_digest,omitempty"`
}

type TeamContinuation struct {
	Ref                          TeamContinuationRef                     `json:"ref"`
	TeamID                       string                                  `json:"team_id"`
	GraphVersion                 string                                  `json:"graph_version"`
	GraphDigest                  string                                  `json:"graph_digest"`
	RunID                        string                                  `json:"run_id"`
	SessionID                    string                                  `json:"session_id"`
	ParentRunID                  string                                  `json:"parent_run_id,omitempty"`
	CausationID                  string                                  `json:"causation_id,omitempty"`
	AuthorityDigest              string                                  `json:"authority_digest"`
	PlanDigest                   string                                  `json:"plan_digest"`
	WriteScopeDigest             string                                  `json:"write_scope_digest"`
	EffectNamespace              string                                  `json:"effect_namespace"`
	OccurrenceID                 string                                  `json:"occurrence_id"`
	Strategy                     Strategy                                `json:"strategy"`
	PauseReason                  TeamPauseReason                         `json:"pause_reason"`
	Tasks                        []TeamTaskContinuation                  `json:"tasks"`
	PendingTaskIDs               []string                                `json:"pending_task_ids"`
	PendingActionReference       string                                  `json:"pending_action_reference,omitempty"`
	ConsumedBudget               int64                                   `json:"consumed_budget,omitempty"`
	OriginalDeadline             time.Time                               `json:"original_deadline,omitempty"`
	TerminalArbitrationReference string                                  `json:"terminal_arbitration_reference,omitempty"`
	PredecessorCheckpointID      string                                  `json:"predecessor_checkpoint_id,omitempty"`
	SuccessorCheckpointID        string                                  `json:"successor_checkpoint_id,omitempty"`
	Generation                   uint64                                  `json:"generation"`
	CreatedAt                    time.Time                               `json:"created_at"`
	ExpiresAt                    time.Time                               `json:"expires_at"`
	HostContextBindings          map[string]hostprojection.SourceBinding `json:"host_context_bindings,omitempty"`
}

func (r TeamContinuationRef) Validate() error {
	if r.SchemaVersion != TeamContinuationVersion || strings.TrimSpace(r.CheckpointID) == "" || strings.TrimSpace(r.Digest) == "" || strings.TrimSpace(r.OwnerNamespace) == "" || strings.TrimSpace(r.TeamID) == "" || strings.TrimSpace(r.GraphVersion) == "" || strings.TrimSpace(r.RunID) == "" || strings.TrimSpace(r.SessionID) == "" {
		return ErrTeamContinuationMalformed
	}
	for _, value := range []string{r.CheckpointID, r.Digest, r.OwnerNamespace, r.TeamID, r.GraphVersion, r.RunID, r.SessionID, r.ParentRunID, r.CausationID} {
		if len(value) > 256 || strings.TrimSpace(value) != value || strings.ContainsAny(value, "\r\n\t") {
			return ErrTeamContinuationMalformed
		}
	}
	return nil
}

func (c TeamContinuation) Detached() TeamContinuation {
	out := c
	out.Tasks = append([]TeamTaskContinuation(nil), c.Tasks...)
	out.PendingTaskIDs = append([]string(nil), c.PendingTaskIDs...)
	if c.HostContextBindings != nil {
		out.HostContextBindings = make(map[string]hostprojection.SourceBinding, len(c.HostContextBindings))
		for id, binding := range c.HostContextBindings {
			out.HostContextBindings[id] = binding
		}
	}
	return out
}

func SealTeamContinuation(c TeamContinuation) (TeamContinuation, error) {
	c = c.Detached()
	if c.Ref.SchemaVersion == "" {
		c.Ref.SchemaVersion = TeamContinuationVersion
	}
	if c.Ref.OwnerNamespace == "" {
		return TeamContinuation{}, ErrTeamContinuationMalformed
	}
	c.Ref.TeamID, c.Ref.GraphVersion, c.Ref.RunID, c.Ref.SessionID = c.TeamID, c.GraphVersion, c.RunID, c.SessionID
	c.Ref.ParentRunID, c.Ref.CausationID = c.ParentRunID, c.CausationID
	// Validate the continuation shape before calculating its canonical digest.
	// The final validation below replaces these temporary values with the
	// source-owned checkpoint ID and digest.
	c.Ref.CheckpointID, c.Ref.Digest = "pending", "pending"
	if err := c.validateShape(); err != nil {
		return TeamContinuation{}, err
	}
	c.Ref.CheckpointID, c.Ref.Digest = "", ""
	b, _ := json.Marshal(c)
	h := sha256.Sum256(b)
	c.Ref.CheckpointID = "team-" + hex.EncodeToString(h[:16])
	c.Ref.Digest = "sha256:" + hex.EncodeToString(h[:])
	if err := c.Validate(); err != nil {
		return TeamContinuation{}, err
	}
	return c, nil
}

func (c TeamContinuation) Validate() error {
	if err := c.validateShape(); err != nil {
		return err
	}
	copy := c.Detached()
	copy.Ref.CheckpointID, copy.Ref.Digest = "", ""
	b, _ := json.Marshal(copy)
	h := sha256.Sum256(b)
	if c.Ref.CheckpointID != "team-"+hex.EncodeToString(h[:16]) || c.Ref.Digest != "sha256:"+hex.EncodeToString(h[:]) {
		return ErrTeamContinuationCorrupt
	}
	return nil
}

func (c TeamContinuation) validateShape() error {
	if err := c.Ref.Validate(); err != nil {
		return err
	}
	if c.TeamID == "" || c.GraphVersion == "" || c.GraphDigest == "" || c.RunID == "" || c.SessionID == "" || c.Generation == 0 || c.ConsumedBudget < 0 || !validTeamPauseReason(c.PauseReason) || (c.Strategy != StrategyParallel && c.Strategy != StrategyVote) || c.CreatedAt.IsZero() || !c.ExpiresAt.After(c.CreatedAt) || c.ExpiresAt.Sub(c.CreatedAt) > 24*time.Hour || len(c.Tasks) > teamContinuationMaxEntries || len(c.PendingTaskIDs) > teamContinuationMaxEntries {
		return ErrTeamContinuationMalformed
	}
	if c.Ref.TeamID != c.TeamID || c.Ref.GraphVersion != c.GraphVersion || c.Ref.RunID != c.RunID || c.Ref.SessionID != c.SessionID || c.Ref.ParentRunID != c.ParentRunID || c.Ref.CausationID != c.CausationID {
		return ErrTeamContinuationForeign
	}
	if len(c.HostContextBindings) > teamContinuationMaxEntries {
		return ErrTeamContinuationMalformed
	}
	seen := make(map[string]struct{}, len(c.Tasks))
	statuses := make(map[string]TaskStatus, len(c.Tasks))
	for _, task := range c.Tasks {
		if task.TaskID == "" || task.Attempts < 0 {
			return ErrTeamContinuationMalformed
		}
		if _, ok := seen[task.TaskID]; ok {
			return ErrTeamContinuationMalformed
		}
		seen[task.TaskID] = struct{}{}
		statuses[task.TaskID] = task.Status
	}
	pending := make(map[string]struct{}, len(c.PendingTaskIDs))
	for _, id := range c.PendingTaskIDs {
		if _, ok := seen[id]; !ok || statuses[id] != TaskStatusPending {
			return ErrTeamContinuationMalformed
		}
		if _, duplicate := pending[id]; duplicate {
			return ErrTeamContinuationMalformed
		}
		pending[id] = struct{}{}
	}
	for id, binding := range c.HostContextBindings {
		if _, ok := pending[id]; !ok {
			return ErrTeamContinuationForeign
		}
		if id == "" || binding.Form != hostprojection.SourceFormTeam || binding.OwnerNamespace != c.Ref.OwnerNamespace || binding.SourceID != c.RunID || binding.NodeID != id || binding.ContinuationID != c.TeamID || binding.ContinuationDigest != c.GraphDigest || binding.Generation != c.Generation {
			return ErrTeamContinuationForeign
		}
	}
	b, err := json.Marshal(c)
	if err != nil || len(b) > teamContinuationMaxBytes {
		return ErrTeamContinuationMalformed
	}
	return nil
}

type TeamResumeRequest struct {
	Ref                    TeamContinuationRef                   `json:"ref"`
	GraphDigest            string                                `json:"graph_digest"`
	Generation             uint64                                `json:"generation"`
	Strategy               Strategy                              `json:"strategy"`
	IdempotencyKey         string                                `json:"idempotency_key"`
	AuthorityDigest        string                                `json:"authority_digest"`
	PlanDigest             string                                `json:"plan_digest"`
	WriteScopeDigest       string                                `json:"write_scope_digest"`
	EffectNamespace        string                                `json:"effect_namespace"`
	OccurrenceID           string                                `json:"occurrence_id"`
	CausationID            string                                `json:"causation_id"`
	Deployment             types.DeploymentRequest               `json:"deployment,omitempty"`
	HostContextProjections map[string]*hostprojection.Descriptor `json:"-"`
	HostContextResolver    hostprojection.Resolver               `json:"-"`
}

func (r TeamResumeRequest) Validate() error {
	if err := r.Ref.Validate(); err != nil {
		return err
	}
	if r.GraphDigest == "" || r.Generation == 0 || r.IdempotencyKey == "" || (r.Strategy != StrategyParallel && r.Strategy != StrategyVote) {
		return ErrTeamContinuationMalformed
	}
	return nil
}
func (r TeamResumeRequest) identity() string {
	digests := make(map[string]string, len(r.HostContextProjections))
	for id, descriptor := range r.HostContextProjections {
		if descriptor != nil {
			digests[id] = descriptor.Digest
		}
	}
	b, _ := json.Marshal(struct {
		Request           TeamResumeRequest `json:"request"`
		ProjectionDigests map[string]string `json:"projection_digests,omitempty"`
	}{Request: r, ProjectionDigests: digests})
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

type TeamResumeClaim struct {
	Request    TeamResumeRequest `json:"request"`
	AttemptID  string            `json:"attempt_id"`
	Generation uint64            `json:"generation"`
	State      string            `json:"state"`
	Duplicate  bool              `json:"duplicate,omitempty"`
	Result     *Result           `json:"result,omitempty"`
}

type TeamContinuationStore interface {
	Save(context.Context, TeamContinuation) error
	Load(context.Context, TeamContinuationRef) (TeamContinuation, error)
	ClaimResume(context.Context, TeamResumeRequest) (TeamResumeClaim, error)
	MarkResumeRunning(context.Context, TeamResumeClaim) (TeamResumeClaim, error)
	CommitResumeResult(context.Context, TeamResumeClaim, Result) (TeamResumeClaim, error)
	DeploymentScope() types.DeploymentScope
	CrossProcessRestore() bool
}
type teamContinuationRecord struct {
	Continuation TeamContinuation `json:"continuation"`
	Claim        *TeamResumeClaim `json:"claim,omitempty"`
	Identity     string           `json:"identity,omitempty"`
	Recovered    bool             `json:"recovered,omitempty"`
}

type MemoryTeamContinuationStore struct {
	mu        sync.Mutex
	namespace string
	records   map[string]teamContinuationRecord
	now       func() time.Time
}

func NewMemoryTeamContinuationStore(namespace string) *MemoryTeamContinuationStore {
	return &MemoryTeamContinuationStore{namespace: strings.TrimSpace(namespace), records: map[string]teamContinuationRecord{}, now: time.Now}
}
func (s *MemoryTeamContinuationStore) DeploymentScope() types.DeploymentScope {
	return types.DefaultDeploymentScope(s.namespace, "team continuation store is process-local")
}
func (s *MemoryTeamContinuationStore) CrossProcessRestore() bool { return false }
func (s *MemoryTeamContinuationStore) Save(ctx context.Context, c TeamContinuation) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.namespace == "" {
		return ErrTeamContinuationUnavailable
	}
	if err := c.Validate(); err != nil {
		return err
	}
	if c.Ref.OwnerNamespace != s.namespace {
		return ErrTeamContinuationForeign
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if old, ok := s.records[c.Ref.CheckpointID]; ok {
		if old.Continuation.Ref.Digest == c.Ref.Digest {
			return nil
		}
		return ErrTeamContinuationConflict
	}
	s.records[c.Ref.CheckpointID] = teamContinuationRecord{Continuation: c.Detached()}
	return nil
}
func (s *MemoryTeamContinuationStore) Load(ctx context.Context, ref TeamContinuationRef) (TeamContinuation, error) {
	if err := ctx.Err(); err != nil {
		return TeamContinuation{}, err
	}
	if s == nil || s.namespace == "" {
		return TeamContinuation{}, ErrTeamContinuationUnavailable
	}
	if err := ref.Validate(); err != nil {
		return TeamContinuation{}, err
	}
	if ref.OwnerNamespace != s.namespace {
		return TeamContinuation{}, ErrTeamContinuationForeign
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.records[ref.CheckpointID]
	if !ok {
		return TeamContinuation{}, ErrTeamContinuationUnavailable
	}
	if record.Continuation.Ref.Digest != ref.Digest {
		return TeamContinuation{}, ErrTeamContinuationForeign
	}
	if !s.now().UTC().Before(record.Continuation.ExpiresAt) {
		return TeamContinuation{}, ErrTeamContinuationExpired
	}
	if err := record.Continuation.Validate(); err != nil {
		return TeamContinuation{}, err
	}
	return record.Continuation.Detached(), nil
}
func (s *MemoryTeamContinuationStore) ClaimResume(ctx context.Context, req TeamResumeRequest) (TeamResumeClaim, error) {
	return s.claimResume(ctx, req, s.DeploymentScope())
}

func (s *MemoryTeamContinuationStore) claimResume(ctx context.Context, req TeamResumeRequest, scope types.DeploymentScope) (TeamResumeClaim, error) {
	if err := ctx.Err(); err != nil {
		return TeamResumeClaim{}, err
	}
	if s == nil || s.namespace == "" {
		return TeamResumeClaim{}, ErrTeamContinuationUnavailable
	}
	if err := req.Validate(); err != nil {
		return TeamResumeClaim{}, err
	}
	if req.Ref.OwnerNamespace != s.namespace {
		return TeamResumeClaim{}, ErrTeamContinuationForeign
	}
	if err := scope.Check(req.Deployment); err != nil {
		return TeamResumeClaim{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.records[req.Ref.CheckpointID]
	if !ok {
		return TeamResumeClaim{}, ErrTeamContinuationUnavailable
	}
	c := rec.Continuation
	if c.Ref.Digest != req.Ref.Digest || c.GraphDigest != req.GraphDigest || c.Generation != req.Generation || c.Strategy != req.Strategy {
		return TeamResumeClaim{}, ErrTeamContinuationConflict
	}
	if !s.now().UTC().Before(c.ExpiresAt) {
		return TeamResumeClaim{}, ErrTeamContinuationExpired
	}
	id := req.identity()
	if rec.Claim != nil {
		if rec.Identity != id {
			return TeamResumeClaim{}, ErrTeamContinuationConflict
		}
		dup := *rec.Claim
		dup.Duplicate = true
		if dup.Result != nil {
			return dup, nil
		}
		if rec.Recovered {
			return dup, ErrTeamContinuationRecoveryRequired
		}
		return dup, ErrTeamContinuationInProgress
	}
	claim := TeamResumeClaim{Request: req, AttemptID: req.Ref.CheckpointID + ":attempt", Generation: c.Generation + 1, State: "claimed"}
	rec.Claim = &claim
	rec.Identity = id
	s.records[req.Ref.CheckpointID] = rec
	return claim, nil
}
func (s *MemoryTeamContinuationStore) MarkResumeRunning(ctx context.Context, claim TeamResumeClaim) (TeamResumeClaim, error) {
	if err := ctx.Err(); err != nil {
		return TeamResumeClaim{}, err
	}
	if s == nil {
		return TeamResumeClaim{}, ErrTeamContinuationUnavailable
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.records[claim.Request.Ref.CheckpointID]
	if !ok || rec.Claim == nil || rec.Identity != claim.Request.identity() || rec.Claim.AttemptID != claim.AttemptID {
		return TeamResumeClaim{}, ErrTeamContinuationConflict
	}
	if rec.Recovered {
		return TeamResumeClaim{}, ErrTeamContinuationRecoveryRequired
	}
	if rec.Claim.State != "claimed" && rec.Claim.State != "running" {
		return TeamResumeClaim{}, ErrTeamContinuationConflict
	}
	rec.Claim.State = "running"
	s.records[claim.Request.Ref.CheckpointID] = rec
	return *rec.Claim, nil
}
func (s *MemoryTeamContinuationStore) CommitResumeResult(ctx context.Context, claim TeamResumeClaim, result Result) (TeamResumeClaim, error) {
	if err := ctx.Err(); err != nil {
		return TeamResumeClaim{}, err
	}
	if s == nil {
		return TeamResumeClaim{}, ErrTeamContinuationUnavailable
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.records[claim.Request.Ref.CheckpointID]
	if !ok || rec.Claim == nil || rec.Identity != claim.Request.identity() || rec.Claim.AttemptID != claim.AttemptID {
		return TeamResumeClaim{}, ErrTeamContinuationConflict
	}
	if rec.Claim.Result != nil {
		return *rec.Claim, nil
	}
	if rec.Recovered {
		return TeamResumeClaim{}, ErrTeamContinuationRecoveryRequired
	}
	if rec.Claim.State != "claimed" && rec.Claim.State != "running" {
		return TeamResumeClaim{}, ErrTeamContinuationConflict
	}
	saved := result
	saved.Tasks = append([]TaskRecord(nil), result.Tasks...)
	rec.Claim.Result = &saved
	rec.Claim.State = "completed"
	s.records[claim.Request.Ref.CheckpointID] = rec
	return *rec.Claim, nil
}

type FileTeamContinuationStore struct {
	*MemoryTeamContinuationStore
	path string
}
type teamContinuationSnapshot struct {
	Version   string                            `json:"version"`
	Namespace string                            `json:"namespace"`
	Records   map[string]teamContinuationRecord `json:"records"`
}

func NewFileTeamContinuationStore(path, namespace string) (*FileTeamContinuationStore, error) {
	if strings.TrimSpace(path) == "" || strings.TrimSpace(namespace) == "" {
		return nil, ErrTeamContinuationMalformed
	}
	s := &FileTeamContinuationStore{MemoryTeamContinuationStore: NewMemoryTeamContinuationStore(namespace), path: strings.TrimSpace(path)}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}
func (s *FileTeamContinuationStore) DeploymentScope() types.DeploymentScope {
	if s == nil {
		return types.DefaultDeploymentScope("team", "team continuation store is unavailable")
	}
	return types.SameHostSingleWriterScope(s.namespace, "fixture:team-continuation.v1")
}
func (s *FileTeamContinuationStore) CrossProcessRestore() bool { return s != nil && s.path != "" }
func (s *FileTeamContinuationStore) Save(ctx context.Context, c TeamContinuation) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.namespace == "" {
		return ErrTeamContinuationUnavailable
	}
	if err := c.Validate(); err != nil {
		return err
	}
	if c.Ref.OwnerNamespace != s.namespace {
		return ErrTeamContinuationForeign
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	old, exists := s.records[c.Ref.CheckpointID]
	if exists {
		if old.Continuation.Ref.Digest == c.Ref.Digest {
			return nil
		}
		return ErrTeamContinuationConflict
	}
	s.records[c.Ref.CheckpointID] = teamContinuationRecord{Continuation: c.Detached()}
	if err := s.persistLocked(); err != nil {
		delete(s.records, c.Ref.CheckpointID)
		return err
	}
	return nil
}
func (s *FileTeamContinuationStore) ClaimResume(ctx context.Context, r TeamResumeRequest) (TeamResumeClaim, error) {
	if err := ctx.Err(); err != nil {
		return TeamResumeClaim{}, err
	}
	if s == nil || s.namespace == "" {
		return TeamResumeClaim{}, ErrTeamContinuationUnavailable
	}
	if err := r.Validate(); err != nil {
		return TeamResumeClaim{}, err
	}
	if r.Ref.OwnerNamespace != s.namespace {
		return TeamResumeClaim{}, ErrTeamContinuationForeign
	}
	if err := s.DeploymentScope().Check(r.Deployment); err != nil {
		return TeamResumeClaim{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.records[r.Ref.CheckpointID]
	if !ok {
		return TeamResumeClaim{}, ErrTeamContinuationUnavailable
	}
	c := rec.Continuation
	if c.Ref.Digest != r.Ref.Digest || c.GraphDigest != r.GraphDigest || c.Generation != r.Generation || c.Strategy != r.Strategy {
		return TeamResumeClaim{}, ErrTeamContinuationConflict
	}
	if !s.now().UTC().Before(c.ExpiresAt) {
		return TeamResumeClaim{}, ErrTeamContinuationExpired
	}
	id := r.identity()
	if rec.Claim != nil {
		if rec.Identity != id {
			return TeamResumeClaim{}, ErrTeamContinuationConflict
		}
		dup := *rec.Claim
		dup.Duplicate = true
		if dup.Result != nil {
			return dup, nil
		}
		if rec.Recovered {
			return dup, ErrTeamContinuationRecoveryRequired
		}
		return dup, ErrTeamContinuationInProgress
	}
	claim := TeamResumeClaim{Request: r, AttemptID: r.Ref.CheckpointID + ":attempt", Generation: c.Generation + 1, State: "claimed"}
	old := cloneTeamContinuationRecord(rec)
	rec.Claim = &claim
	rec.Identity = id
	s.records[r.Ref.CheckpointID] = rec
	if err := s.persistLocked(); err != nil {
		s.records[r.Ref.CheckpointID] = old
		return TeamResumeClaim{}, err
	}
	return claim, nil
}
func (s *FileTeamContinuationStore) MarkResumeRunning(ctx context.Context, c TeamResumeClaim) (TeamResumeClaim, error) {
	if err := ctx.Err(); err != nil {
		return TeamResumeClaim{}, err
	}
	if s == nil {
		return TeamResumeClaim{}, ErrTeamContinuationUnavailable
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.records[c.Request.Ref.CheckpointID]
	if !ok || rec.Claim == nil || rec.Identity != c.Request.identity() || rec.Claim.AttemptID != c.AttemptID {
		return TeamResumeClaim{}, ErrTeamContinuationConflict
	}
	if rec.Recovered {
		return TeamResumeClaim{}, ErrTeamContinuationRecoveryRequired
	}
	if rec.Claim.State != "claimed" && rec.Claim.State != "running" {
		return TeamResumeClaim{}, ErrTeamContinuationConflict
	}
	old := cloneTeamContinuationRecord(rec)
	rec.Claim.State = "running"
	s.records[c.Request.Ref.CheckpointID] = rec
	if err := s.persistLocked(); err != nil {
		s.records[c.Request.Ref.CheckpointID] = old
		return TeamResumeClaim{}, err
	}
	return *rec.Claim, nil
}
func (s *FileTeamContinuationStore) CommitResumeResult(ctx context.Context, c TeamResumeClaim, r Result) (TeamResumeClaim, error) {
	if err := ctx.Err(); err != nil {
		return TeamResumeClaim{}, err
	}
	if s == nil {
		return TeamResumeClaim{}, ErrTeamContinuationUnavailable
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.records[c.Request.Ref.CheckpointID]
	if !ok || rec.Claim == nil || rec.Identity != c.Request.identity() || rec.Claim.AttemptID != c.AttemptID {
		return TeamResumeClaim{}, ErrTeamContinuationConflict
	}
	if rec.Claim.Result != nil {
		return *rec.Claim, nil
	}
	if rec.Recovered {
		return TeamResumeClaim{}, ErrTeamContinuationRecoveryRequired
	}
	if rec.Claim.State != "claimed" && rec.Claim.State != "running" {
		return TeamResumeClaim{}, ErrTeamContinuationConflict
	}
	old := cloneTeamContinuationRecord(rec)
	saved := r
	saved.Tasks = append([]TaskRecord(nil), r.Tasks...)
	rec.Claim.Result = &saved
	rec.Claim.State = "completed"
	s.records[c.Request.Ref.CheckpointID] = rec
	if err := s.persistLocked(); err != nil {
		s.records[c.Request.Ref.CheckpointID] = old
		return TeamResumeClaim{}, err
	}
	return *rec.Claim, nil
}
func (s *FileTeamContinuationStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	raw, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return ErrTeamContinuationUnavailable
	}
	var snap teamContinuationSnapshot
	if json.Unmarshal(raw, &snap) != nil || snap.Version != TeamContinuationVersion || snap.Namespace != s.namespace || snap.Records == nil {
		return ErrTeamContinuationCorrupt
	}
	for id, rec := range snap.Records {
		if id != rec.Continuation.Ref.CheckpointID || rec.Continuation.Validate() != nil {
			return ErrTeamContinuationCorrupt
		}
		if rec.Claim != nil && rec.Claim.Result == nil {
			rec.Recovered = true
			snap.Records[id] = rec
		}
	}
	s.records = snap.Records
	return nil
}
func (s *FileTeamContinuationStore) persistLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return ErrTeamContinuationUnavailable
	}
	snap := teamContinuationSnapshot{Version: TeamContinuationVersion, Namespace: s.namespace, Records: s.records}
	raw, err := json.Marshal(snap)
	if err != nil {
		return ErrTeamContinuationUnavailable
	}
	tmp := s.path + ".tmp"
	if err = os.WriteFile(tmp, raw, 0o600); err != nil {
		return ErrTeamContinuationUnavailable
	}
	if err = os.Rename(tmp, s.path); err != nil {
		return ErrTeamContinuationUnavailable
	}
	return nil
}

func cloneTeamContinuationRecord(in teamContinuationRecord) teamContinuationRecord {
	out := in
	out.Continuation = in.Continuation.Detached()
	if in.Claim != nil {
		claim := *in.Claim
		if in.Claim.Result != nil {
			result := *in.Claim.Result
			result.Tasks = append([]TaskRecord(nil), in.Claim.Result.Tasks...)
			claim.Result = &result
		}
		out.Claim = &claim
	}
	return out
}

func (e *Engine) ContinuationCapabilities(context.Context) (types.ContinuationCapabilities, error) {
	if e == nil || e.continuations == nil || e.continuationNamespace == "" {
		return types.ContinuationCapabilities{}, ErrTeamContinuationUnsupported
	}
	scope := e.continuations.DeploymentScope()
	caps := types.ContinuationCapabilities{ProfileVersion: types.NativeContinuationVersion, SourceNamespace: "team", ContractVersion: "host_runtime_capability_manifest.v1", ModuleRevision: "baymax", APIVersion: "v1", RetentionPolicy: "source-owned-team-checkpoint", ExpiryPolicy: "checkpoint-expiry", PauseReasons: []string{"input_required", "approval_required", "budget_exhausted"}, EvidenceReferences: map[string]string{"checkpoint_creation": "fixture:team-continuation.v1", "cross_process_restore": "fixture:team-continuation.v1", "input_required": "fixture:team-continuation.v1", "approval_required": "fixture:team-continuation.v1", "budget_exhausted": "fixture:team-continuation.v1"}, Deployment: scope}
	caps.CheckpointCreation = true
	caps.CrossProcessRestore = e.continuations.CrossProcessRestore() && scope.Supported
	if !caps.CrossProcessRestore {
		caps.UnsupportedReasons = map[string]string{"cross_process_restore": "team continuation store is process-local"}
	}
	return caps, nil
}

func teamPlanDigest(plan Plan) string {
	type taskIdentity struct {
		TaskID   string     `json:"task_id"`
		AgentID  string     `json:"agent_id"`
		Role     Role       `json:"role"`
		Priority int        `json:"priority"`
		Target   TaskTarget `json:"target"`
		PeerID   string     `json:"peer_id,omitempty"`
	}
	tasks := make([]taskIdentity, 0, len(plan.Tasks))
	for _, task := range plan.Tasks {
		tasks = append(tasks, taskIdentity{TaskID: task.TaskID, AgentID: task.AgentID, Role: task.Role, Priority: task.Priority, Target: task.Target, PeerID: task.Remote.PeerID})
	}
	b, _ := json.Marshal(struct {
		TeamID, WorkflowID, StepID string
		Strategy                   Strategy
		Tasks                      []taskIdentity
	}{plan.TeamID, plan.WorkflowID, plan.StepID, plan.Strategy, tasks})
	h := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(h[:])
}

func pendingTaskIDs(plan Plan, records []TaskRecord) []string {
	out := make([]string, 0, len(records))
	for i, record := range records {
		if record.Status == TaskStatusPending && i < len(plan.Tasks) {
			out = append(out, plan.Tasks[i].TaskID)
		}
	}
	sort.Strings(out)
	return out
}

func (e *Engine) persistPausedResult(ctx context.Context, plan Plan, records []TaskRecord, pause TeamPauseError) (Result, error) {
	if e == nil || e.continuations == nil || strings.TrimSpace(e.continuationNamespace) == "" {
		return Result{}, ErrTeamContinuationUnsupported
	}
	now := e.now().UTC()
	if pause.OriginalDeadline.IsZero() {
		pause.OriginalDeadline = now.Add(24 * time.Hour)
	}
	tasks := make([]TeamTaskContinuation, 0, len(records))
	for i, record := range records {
		outputDigest := ""
		if record.Output != nil {
			b, err := json.Marshal(record.Output)
			if err != nil {
				return Result{}, ErrTeamContinuationMalformed
			}
			h := sha256.Sum256(b)
			outputDigest = "sha256:" + hex.EncodeToString(h[:])
		}
		id := record.TaskID
		if id == "" && i < len(plan.Tasks) {
			id = plan.Tasks[i].TaskID
		}
		tasks = append(tasks, TeamTaskContinuation{TaskID: id, Status: record.Status, Vote: record.Vote, OutputDigest: outputDigest})
	}
	pending := pendingTaskIDs(plan, records)
	bindings, err := teamHostContextBindings(plan.HostContextBindings, pending, e.continuationNamespace, plan, teamPlanDigest(plan), 1)
	if err != nil {
		return Result{}, err
	}
	continuation, err := SealTeamContinuation(TeamContinuation{Ref: TeamContinuationRef{OwnerNamespace: e.continuationNamespace}, TeamID: plan.TeamID, GraphVersion: "v1", GraphDigest: teamPlanDigest(plan), RunID: plan.RunID, SessionID: plan.SessionID, ParentRunID: plan.ParentRunID, CausationID: plan.CausationID, AuthorityDigest: plan.AuthorityDigest, PlanDigest: plan.PlanDigest, WriteScopeDigest: plan.WriteScopeDigest, EffectNamespace: plan.EffectNamespace, OccurrenceID: plan.OccurrenceID, Strategy: plan.Strategy, PauseReason: pause.Reason, Tasks: tasks, PendingTaskIDs: pending, PendingActionReference: pause.PendingActionReference, ConsumedBudget: pause.ConsumedBudget, OriginalDeadline: pause.OriginalDeadline, Generation: 1, CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour), HostContextBindings: bindings})
	if err != nil {
		return Result{}, err
	}
	if err := e.continuations.Save(ctx, continuation); err != nil {
		return Result{}, err
	}
	result := buildResult(plan, records)
	result.TeamCheckpoint = &continuation.Ref
	result.PauseReason = pause.Reason
	return result, nil
}

func teamHostContextBindings(input map[string]hostprojection.SourceBinding, pendingIDs []string, ownerNamespace string, plan Plan, graphDigest string, generation uint64) (map[string]hostprojection.SourceBinding, error) {
	if len(input) == 0 || len(pendingIDs) == 0 {
		if len(input) > 0 {
			return nil, ErrTeamContinuationForeign
		}
		return nil, nil
	}
	if len(input) > teamContinuationMaxEntries {
		return nil, ErrTeamContinuationMalformed
	}
	pending := make(map[string]struct{}, len(pendingIDs))
	for _, id := range pendingIDs {
		pending[id] = struct{}{}
	}
	out := make(map[string]hostprojection.SourceBinding, len(input))
	for id, requested := range input {
		if requested.Form != hostprojection.SourceFormTeam {
			return nil, ErrTeamContinuationForeign
		}
		found := false
		local := false
		for _, task := range plan.Tasks {
			if task.TaskID == id {
				found = true
				local = task.Target == "" || task.Target == TaskTargetLocal
				break
			}
		}
		if !found {
			return nil, ErrTeamContinuationForeign
		}
		if !local {
			return nil, ErrTeamContinuationUnsupported
		}
		if _, ok := pending[id]; !ok || requested.Form != hostprojection.SourceFormTeam {
			continue
		}
		out[id] = hostprojection.SourceBinding{
			Form:               hostprojection.SourceFormTeam,
			OwnerNamespace:     ownerNamespace,
			SourceID:           plan.RunID,
			NodeID:             id,
			ContinuationID:     plan.TeamID,
			ContinuationDigest: graphDigest,
			Generation:         generation,
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}
