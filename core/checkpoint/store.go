package checkpoint

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

const storeFileName = "checkpoints.json"

type Options struct {
	Namespace  string
	Capacity   int
	Now        func() time.Time
	Limits     types.ContinuationLimits
	Deployment types.DeploymentScope
}
type record struct {
	Checkpoint types.PersistedRunCheckpoint `json:"checkpoint"`
	State      string                       `json:"state"`
	Claim      *types.ResumeClaim           `json:"claim,omitempty"`
	Identity   string                       `json:"identity,omitempty"`
	Canceled   bool                         `json:"canceled,omitempty"`
	Generation uint64                       `json:"generation,omitempty"`
	Recovered  bool                         `json:"-"`
}
type snapshot struct {
	Version   string            `json:"version"`
	Namespace string            `json:"namespace"`
	Records   map[string]record `json:"records"`
	Checksum  string            `json:"checksum"`
}
type Store struct {
	mu       sync.Mutex
	opts     Options
	records  map[string]record
	dir      string
	lockFile *os.File
	closed   atomic.Bool
	poisoned bool
}
type MemoryStore = Store
type FileStore = Store

func NewMemoryStore(opts Options) (*MemoryStore, error) {
	if err := normalizeOptions(&opts); err != nil {
		return nil, err
	}
	return &Store{opts: opts, records: map[string]record{}}, nil
}
func OpenFileStore(dir string, opts Options) (*FileStore, error) {
	if err := normalizeOptions(&opts); err != nil {
		return nil, err
	}
	if dir == "" {
		return nil, types.ErrCheckpointMalformed
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, wrapUnavailable(err)
	}
	lock, err := acquireLock(filepath.Join(dir, ".writer.lock"))
	if err != nil {
		return nil, wrapUnavailable(err)
	}
	s := &Store{opts: opts, dir: dir, lockFile: lock, records: map[string]record{}}
	if err = s.readSnapshot(); err != nil {
		_ = releaseLock(lock)
		return nil, err
	}
	return s, nil
}
func normalizeOptions(opts *Options) error {
	if opts.Namespace == "" || opts.Capacity <= 0 {
		return types.ErrCheckpointMalformed
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.Deployment == (types.DeploymentScope{}) {
		opts.Deployment = types.SameHostSingleWriterScope(opts.Namespace, "fixture:deployment-scope.v1")
	}
	if err := opts.Deployment.Validate(); err != nil {
		return err
	}
	l, err := opts.Limits.Normalized()
	if err != nil {
		return err
	}
	opts.Limits = l
	return nil
}
func (s *Store) DeploymentScope() types.DeploymentScope {
	if s == nil {
		return types.DefaultDeploymentScope("checkpoint", "checkpoint store is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.opts.Deployment
}
func (s *Store) CrossProcessRestore() bool {
	if s == nil || s.dir == "" || s.closed.Load() {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return !s.poisoned && !s.closed.Load()
}
func (s *Store) Close() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed.Load() {
		return nil
	}
	s.closed.Store(true)
	if s.lockFile != nil {
		return releaseLock(s.lockFile)
	}
	return nil
}
func (s *Store) check(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.closed.Load() {
		return types.ErrCheckpointUnavailable
	}
	return nil
}
func (s *Store) Save(ctx context.Context, p types.PersistedRunCheckpoint) error {
	if err := s.check(ctx); err != nil {
		return err
	}
	if err := p.Validate(); err != nil {
		return err
	}
	if p.Ref.OwnerNamespace != s.opts.Namespace {
		return types.ErrCheckpointForeign
	}
	p = p.Detached()
	if len(p.Material) > s.opts.Limits.MaterialBytes || len(p.ToolResultRefs) > s.opts.Limits.References || p.CompletedEntries+p.PendingEntries > s.opts.Limits.Entries {
		return types.ErrCheckpointMalformed
	}
	if envelopeSize(p) > s.opts.Limits.EnvelopeBytes {
		return types.ErrCheckpointMalformed
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.check(ctx); err != nil {
		return err
	}
	if s.poisoned {
		return types.ErrCheckpointUnavailable
	}
	key := p.Ref.CheckpointID
	if old, ok := s.records[key]; ok {
		if old.Checkpoint.Ref.Digest == p.Ref.Digest {
			return nil
		}
		return types.ErrCheckpointConflict
	}
	if len(s.records) >= s.opts.Capacity {
		return types.ErrCheckpointUnavailable
	}
	s.records[key] = record{Checkpoint: p, State: "persisted", Generation: p.Generation}
	if err := s.publish(); err != nil {
		delete(s.records, key)
		return err
	}
	return nil
}
func (s *Store) Load(ctx context.Context, ref types.NativeCheckpointRef) (types.PersistedRunCheckpoint, error) {
	if err := s.check(ctx); err != nil {
		return types.PersistedRunCheckpoint{}, err
	}
	if err := ref.Validate(); err != nil {
		return types.PersistedRunCheckpoint{}, err
	}
	if ref.OwnerNamespace != s.opts.Namespace {
		return types.PersistedRunCheckpoint{}, types.ErrCheckpointForeign
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.poisoned {
		return types.PersistedRunCheckpoint{}, types.ErrCheckpointUnavailable
	}
	if err := s.check(ctx); err != nil {
		return types.PersistedRunCheckpoint{}, err
	}
	r, ok := s.records[ref.CheckpointID]
	if !ok {
		return types.PersistedRunCheckpoint{}, types.ErrCheckpointNotFound
	}
	if err := s.validateRecord(r, ref, true); err != nil {
		return types.PersistedRunCheckpoint{}, err
	}
	if envelopeSize(r.Checkpoint) > s.opts.Limits.EnvelopeBytes {
		return types.PersistedRunCheckpoint{}, types.ErrCheckpointCorrupt
	}
	return r.Checkpoint.Detached(), nil
}

// ReadResumeClaim reads source facts without admitting or executing a resume.
// Missing claims are distinct from completed results; callers still authorize
// result access and compare the complete request identity before projection.
func (s *Store) ReadResumeClaim(ctx context.Context, ref types.NativeCheckpointRef) (types.ResumeClaim, error) {
	if err := s.check(ctx); err != nil {
		return types.ResumeClaim{}, err
	}
	if err := ref.Validate(); err != nil {
		return types.ResumeClaim{}, err
	}
	if ref.OwnerNamespace != s.opts.Namespace {
		return types.ResumeClaim{}, types.ErrCheckpointForeign
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.check(ctx); err != nil {
		return types.ResumeClaim{}, err
	}
	if s.poisoned {
		return types.ResumeClaim{}, types.ErrCheckpointUnavailable
	}
	r, ok := s.records[ref.CheckpointID]
	if !ok {
		return types.ResumeClaim{}, types.ErrCheckpointNotFound
	}
	if err := s.validateRecord(r, ref, true); err != nil {
		return types.ResumeClaim{}, err
	}
	if r.Canceled {
		return types.ResumeClaim{}, types.ErrCheckpointTerminal
	}
	if r.Claim == nil {
		return types.ResumeClaim{}, types.ErrCheckpointNotFound
	}
	claim := cloneClaim(*r.Claim)
	claim.Duplicate = true
	return claim, nil
}

func (s *Store) ClaimResume(ctx context.Context, req types.ResumeRequest) (types.ResumeClaim, error) {
	if req.ProposedBudget != nil {
		proposed := *req.ProposedBudget
		req.ProposedBudget = &proposed
	}
	if err := s.check(ctx); err != nil {
		return types.ResumeClaim{}, err
	}
	if err := req.Validate(); err != nil {
		return types.ResumeClaim{}, err
	}
	if err := s.opts.Deployment.Check(req.Deployment); err != nil {
		return types.ResumeClaim{}, err
	}
	if req.Checkpoint.OwnerNamespace != s.opts.Namespace {
		return types.ResumeClaim{}, types.ErrCheckpointForeign
	}
	if ctx.Err() != nil {
		return types.ResumeClaim{}, ctx.Err()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.poisoned {
		return types.ResumeClaim{}, types.ErrCheckpointUnavailable
	}
	if err := s.check(ctx); err != nil {
		return types.ResumeClaim{}, err
	}
	key := req.Checkpoint.CheckpointID
	r, ok := s.records[key]
	if !ok {
		return types.ResumeClaim{}, types.ErrCheckpointNotFound
	}
	if r.Canceled {
		return types.ResumeClaim{}, types.ErrCheckpointTerminal
	}
	identity := req.IdentityDigest()
	if r.Claim != nil {
		if err := s.validateRecord(r, req.Checkpoint, r.Claim.Result != nil && r.Identity == identity); err != nil {
			return types.ResumeClaim{}, err
		}
		if r.Identity != identity {
			return types.ResumeClaim{}, types.ErrCheckpointConflict
		}
		dup := cloneClaim(*r.Claim)
		dup.Duplicate = true
		if dup.Result != nil {
			return dup, nil
		}
		if dup.State == types.ResumeClaimClaimed {
			if r.Recovered {
				return dup, types.ErrCheckpointRecoveryRequired
			}
			return dup, types.ErrCheckpointInProgress
		}
		return dup, types.ErrCheckpointInProgress
	}
	if err := s.validateRecord(r, req.Checkpoint, false); err != nil {
		return types.ResumeClaim{}, err
	}
	if req.Generation != r.Checkpoint.Generation {
		return types.ResumeClaim{}, types.ErrCheckpointConflict
	}
	if req.Mode != r.Checkpoint.Mode || req.DependencyProfile != r.Checkpoint.DependencyProfile || req.DependencyRevision != r.Checkpoint.DependencyRevision {
		return types.ResumeClaim{}, types.ErrCheckpointForeign
	}
	if !sameRefIdentity(req.Checkpoint, r.Checkpoint.Ref) {
		return types.ResumeClaim{}, types.ErrCheckpointForeign
	}
	if req.Budget.Consumed != r.Checkpoint.Budget.Consumed || req.Budget.ToolConsumed != r.Checkpoint.Budget.ToolConsumed || req.Budget.Limit > 0 && req.Budget.Limit < req.Budget.Consumed || req.Budget.ToolLimit > 0 && req.Budget.ToolLimit < req.Budget.ToolConsumed {
		return types.ResumeClaim{}, types.ErrCheckpointConflict
	}
	oldBudget := r.Checkpoint.Budget
	if !req.Budget.Deadline.Equal(oldBudget.Deadline) || req.Budget.Limit < oldBudget.Limit || req.Budget.ToolLimit < oldBudget.ToolLimit || req.Budget != oldBudget && (req.Checkpoint.PauseReason != types.PauseReasonBudgetExhausted || req.Budget.Revision == "" || req.Budget.Revision == oldBudget.Revision) {
		return types.ResumeClaim{}, types.ErrCheckpointConflict
	}
	claim := types.ResumeClaim{Request: req, AttemptID: newID(), Generation: nextGeneration(r.Generation), State: types.ResumeClaimClaimed}
	r.Claim = &claim
	r.Identity = identity
	r.State = "claimed"
	r.Generation = claim.Generation
	s.records[key] = r
	if err := s.publish(); err != nil {
		return types.ResumeClaim{}, err
	}
	return cloneClaim(claim), nil
}

func (s *Store) MarkRunning(ctx context.Context, claim types.ResumeClaim) (types.ResumeClaim, error) {
	return s.mutateClaim(ctx, claim, func(r *record) (types.ResumeClaim, error) {
		if r.Recovered {
			return types.ResumeClaim{}, types.ErrCheckpointRecoveryRequired
		}
		if r.Claim.State == types.ResumeClaimRunning {
			return cloneClaim(*r.Claim), nil
		}
		if r.Claim.State != types.ResumeClaimClaimed {
			return types.ResumeClaim{}, types.ErrCheckpointTerminal
		}
		r.Claim.State = types.ResumeClaimRunning
		r.State = "running"
		r.Generation = nextGeneration(r.Generation)
		r.Claim.Generation = r.Generation
		return cloneClaim(*r.Claim), nil
	})
}
func (s *Store) CommitResumeResult(ctx context.Context, claim types.ResumeClaim, result types.ResumeResult) (types.ResumeClaim, error) {
	return s.mutateClaim(ctx, claim, func(r *record) (types.ResumeClaim, error) {
		if r.Claim.Result != nil {
			if sameResult(*r.Claim.Result, result) {
				return cloneClaim(*r.Claim), nil
			}
			return types.ResumeClaim{}, types.ErrCheckpointConflict
		}
		if r.Recovered {
			return types.ResumeClaim{}, types.ErrCheckpointRecoveryRequired
		}
		if r.Claim.State != types.ResumeClaimRunning && r.Claim.State != types.ResumeClaimClaimed {
			return types.ResumeClaim{}, types.ErrCheckpointTerminal
		}
		if err := validateResult(result); err != nil {
			return types.ResumeClaim{}, err
		}
		if result.Successor != nil {
			next := *result.Successor
			if err := next.Validate(); err != nil {
				return types.ResumeClaim{}, err
			}
			if len(next.Material) > s.opts.Limits.MaterialBytes || len(next.ToolResultRefs) > s.opts.Limits.References || next.CompletedEntries+next.PendingEntries > s.opts.Limits.Entries || envelopeSize(next) > s.opts.Limits.EnvelopeBytes {
				return types.ResumeClaim{}, types.ErrCheckpointMalformed
			}
			next = next.Detached()
			if next.Ref.OwnerNamespace != s.opts.Namespace || next.Ref.PredecessorCheckpointID != r.Checkpoint.Ref.CheckpointID || next.Ref.ResumeAttemptID != claim.AttemptID {
				return types.ResumeClaim{}, types.ErrCheckpointForeign
			}
			a, b := next.Ref, r.Checkpoint.Ref
			if a.SourceComponent != b.SourceComponent || a.RunID != b.RunID || a.SessionID != b.SessionID || a.ParentRunID != b.ParentRunID || a.HostCorrelation != b.HostCorrelation || a.AuthorityDigest != b.AuthorityDigest || a.PlanDigest != b.PlanDigest || a.WriteScopeDigest != b.WriteScopeDigest || a.OccurrenceID != b.OccurrenceID || a.EffectNamespace != b.EffectNamespace {
				return types.ResumeClaim{}, types.ErrCheckpointForeign
			}
			if _, ok := s.records[next.Ref.CheckpointID]; ok {
				return types.ResumeClaim{}, types.ErrCheckpointConflict
			}
			if len(s.records) >= s.opts.Capacity {
				return types.ResumeClaim{}, types.ErrCheckpointUnavailable
			}
			s.records[next.Ref.CheckpointID] = record{Checkpoint: next, State: "persisted", Generation: next.Generation}
		}
		res := cloneResult(result)
		r.Claim.Result = &res
		r.Claim.State = types.ResumeClaimCompleted
		r.Claim.RetainedUntil = s.opts.Now().UTC().Add(24 * time.Hour)
		r.State = "completed"
		r.Generation = nextGeneration(r.Generation)
		r.Claim.Generation = r.Generation
		return cloneClaim(*r.Claim), nil
	})
}
func (s *Store) Control(ctx context.Context, c types.CheckpointControl) (types.ResumeClaim, error) {
	if err := s.check(ctx); err != nil {
		return types.ResumeClaim{}, err
	}
	if c.Action != types.CheckpointControlCancel && c.Action != types.CheckpointControlTerminal {
		return types.ResumeClaim{}, types.ErrCheckpointMalformed
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.poisoned {
		return types.ResumeClaim{}, types.ErrCheckpointUnavailable
	}
	if err := s.check(ctx); err != nil {
		return types.ResumeClaim{}, err
	}
	r, ok := s.records[c.Checkpoint.CheckpointID]
	if !ok {
		return types.ResumeClaim{}, types.ErrCheckpointNotFound
	}
	if err := s.validateRecord(r, c.Checkpoint, false); err != nil {
		return types.ResumeClaim{}, err
	}
	if c.Generation != r.Checkpoint.Generation {
		return types.ResumeClaim{}, types.ErrCheckpointConflict
	}
	if r.Claim != nil {
		return types.ResumeClaim{}, types.ErrCheckpointInProgress
	}
	r.Canceled = true
	r.State = string(types.ResumeClaimCanceled)
	s.records[c.Checkpoint.CheckpointID] = r
	if err := s.publish(); err != nil {
		return types.ResumeClaim{}, err
	}
	return types.ResumeClaim{}, nil
}
func (s *Store) mutateClaim(ctx context.Context, claim types.ResumeClaim, fn func(*record) (types.ResumeClaim, error)) (types.ResumeClaim, error) {
	if err := s.check(ctx); err != nil {
		return types.ResumeClaim{}, err
	}
	if ctx.Err() != nil {
		return types.ResumeClaim{}, ctx.Err()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.poisoned {
		return types.ResumeClaim{}, types.ErrCheckpointUnavailable
	}
	if err := s.check(ctx); err != nil {
		return types.ResumeClaim{}, err
	}
	key := claim.Request.Checkpoint.CheckpointID
	r, ok := s.records[key]
	if !ok {
		return types.ResumeClaim{}, types.ErrCheckpointNotFound
	}
	if r.Claim == nil || r.Claim.AttemptID != claim.AttemptID || r.Claim.Generation != claim.Generation || r.Generation != claim.Generation || r.Identity != claim.Request.IdentityDigest() {
		return types.ResumeClaim{}, types.ErrCheckpointConflict
	}
	cloned := cloneClaim(*r.Claim)
	r.Claim = &cloned
	result, err := fn(&r)
	if err != nil {
		return result, err
	}
	s.records[key] = r
	if err = s.publish(); err != nil {
		return types.ResumeClaim{}, err
	}
	return result, nil
}
func (s *Store) validateRecord(r record, ref types.NativeCheckpointRef, allowRetainedResult bool) error {
	if err := r.Checkpoint.Validate(); err != nil {
		return err
	}
	if !sameRefIdentity(ref, r.Checkpoint.Ref) {
		return types.ErrCheckpointForeign
	}
	if !s.opts.Now().Before(r.Checkpoint.Ref.ExpiresAt) && (!allowRetainedResult || r.Claim == nil || r.Claim.Result == nil || !s.opts.Now().Before(r.Claim.RetainedUntil)) {
		return types.ErrCheckpointExpired
	}
	return nil
}

func envelopeSize(p types.PersistedRunCheckpoint) int {
	p.Material = nil
	b, err := json.Marshal(p)
	if err != nil {
		return types.NativeContinuationMaxEnvelopeBytes + 1
	}
	return len(b)
}
func (s *Store) readSnapshot() error {
	path := filepath.Join(s.dir, storeFileName)
	info, statErr := os.Stat(path)
	if errors.Is(statErr, os.ErrNotExist) {
		return nil
	}
	if statErr != nil {
		return wrapUnavailable(statErr)
	}
	if info.Size() > s.maxSnapshotBytes() {
		return types.ErrCheckpointCorrupt
	}
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return wrapUnavailable(err)
	}
	defer func() { _ = f.Close() }()
	b, err := io.ReadAll(io.LimitReader(f, s.maxSnapshotBytes()+1))
	if err != nil {
		return wrapUnavailable(err)
	}
	if int64(len(b)) > s.maxSnapshotBytes() {
		return types.ErrCheckpointCorrupt
	}
	var snap snapshot
	if err = json.Unmarshal(b, &snap); err != nil {
		return types.ErrCheckpointCorrupt
	}
	if snap.Version != types.NativeContinuationVersion || snap.Namespace != s.opts.Namespace || snap.Records == nil {
		return types.ErrCheckpointCorrupt
	}
	checksum := snap.Checksum
	snap.Checksum = ""
	canonical, _ := json.Marshal(snap)
	if checksum != digest(canonical) {
		return types.ErrCheckpointCorrupt
	}
	if len(snap.Records) > s.opts.Capacity {
		return types.ErrCheckpointCorrupt
	}
	for id, r := range snap.Records {
		if id != r.Checkpoint.Ref.CheckpointID || r.Checkpoint.Ref.OwnerNamespace != s.opts.Namespace || r.Checkpoint.Validate() != nil || validateRecordState(r) != nil || len(r.Checkpoint.Material) > s.opts.Limits.MaterialBytes || len(r.Checkpoint.ToolResultRefs) > s.opts.Limits.References || r.Checkpoint.CompletedEntries+r.Checkpoint.PendingEntries > s.opts.Limits.Entries || envelopeSize(r.Checkpoint) > s.opts.Limits.EnvelopeBytes {
			return types.ErrCheckpointCorrupt
		}
	}
	s.records = snap.Records
	for id, r := range s.records {
		if r.Claim != nil && r.Claim.Result == nil {
			r.Recovered = true
			s.records[id] = r
		}
	}
	return nil
}
func (s *Store) publish() (publishErr error) {
	defer func() {
		if publishErr != nil {
			s.poisoned = true
		}
	}()
	if s.dir == "" {
		return nil
	}
	snap := snapshot{Version: types.NativeContinuationVersion, Namespace: s.opts.Namespace, Records: s.records}
	canonical, _ := json.Marshal(snap)
	snap.Checksum = digest(canonical)
	b, err := json.Marshal(snap)
	if err != nil {
		return wrapUnavailable(err)
	}
	if int64(len(b)) > s.maxSnapshotBytes() {
		return types.ErrCheckpointUnavailable
	}
	tmp, err := os.CreateTemp(s.dir, ".checkpoint-*.tmp")
	if err != nil {
		return wrapUnavailable(err)
	}
	name := tmp.Name()
	defer func() { _ = os.Remove(name) }()
	if err = tmp.Chmod(0600); err == nil {
		_, err = tmp.Write(b)
	}
	if err == nil {
		err = tmp.Sync()
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return wrapUnavailable(err)
	}
	if err = atomicReplace(name, filepath.Join(s.dir, storeFileName)); err != nil {
		return wrapUnavailable(err)
	}
	if err = syncDirectory(s.dir); err != nil {
		return wrapUnavailable(err)
	}
	return nil
}
func sameRefIdentity(a, b types.NativeCheckpointRef) bool {
	return a == b
}
func cloneClaim(c types.ResumeClaim) types.ResumeClaim {
	if c.Request.ProposedBudget != nil {
		budget := *c.Request.ProposedBudget
		c.Request.ProposedBudget = &budget
	}
	if c.Result != nil {
		r := cloneResult(*c.Result)
		c.Result = &r
	}
	return c
}
func cloneResult(r types.ResumeResult) types.ResumeResult {
	r.Projection = append(json.RawMessage(nil), r.Projection...)
	if r.Successor != nil {
		x := r.Successor.Detached()
		r.Successor = &x
	}
	return r
}
func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("attempt-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}
func wrapUnavailable(err error) error {
	_ = err
	return types.ErrCheckpointUnavailable
}

func digest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func (s *Store) maxSnapshotBytes() int64 {
	return int64(s.opts.Capacity) * (int64(s.opts.Limits.EnvelopeBytes) + 2*int64(s.opts.Limits.MaterialBytes) + 8192)
}
func validateRecordState(r record) error {
	if r.State == "persisted" {
		if r.Claim != nil || r.Identity != "" || r.Canceled {
			return types.ErrCheckpointCorrupt
		}
		return nil
	}
	if r.Canceled {
		if r.State != string(types.ResumeClaimCanceled) || r.Claim != nil {
			return types.ErrCheckpointCorrupt
		}
		return nil
	}
	if r.Claim == nil || r.Identity == "" || r.Claim.AttemptID == "" || r.Generation == 0 || r.Claim.Generation != r.Generation || r.Claim.Request.Checkpoint != r.Checkpoint.Ref || r.Claim.Request.Validate() != nil || r.Identity != r.Claim.Request.IdentityDigest() {
		return types.ErrCheckpointCorrupt
	}
	switch r.State {
	case "claimed":
		if r.Claim.State != types.ResumeClaimClaimed || r.Claim.Result != nil {
			return types.ErrCheckpointCorrupt
		}
	case "running":
		if r.Claim.State != types.ResumeClaimRunning || r.Claim.Result != nil {
			return types.ErrCheckpointCorrupt
		}
	case "completed":
		if r.Claim.State != types.ResumeClaimCompleted || r.Claim.Result == nil {
			return types.ErrCheckpointCorrupt
		}
	default:
		return types.ErrCheckpointCorrupt
	}
	if r.Claim.Result != nil {
		if r.Claim.RetainedUntil.IsZero() || validateResult(*r.Claim.Result) != nil {
			return types.ErrCheckpointCorrupt
		}
		if r.Claim.Result.Successor != nil && r.Claim.Result.Successor.Validate() != nil {
			return types.ErrCheckpointCorrupt
		}
	}
	return nil
}

func nextGeneration(g uint64) uint64 { return g + 1 }

func sameResult(a, b types.ResumeResult) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}

func validateResult(result types.ResumeResult) error {
	if len(result.Projection) > types.NativeContinuationMaxEnvelopeBytes || len(result.Projection) > 0 && !json.Valid(result.Projection) || len(result.ErrorCode) > types.NativeContinuationMaxReferenceBytes {
		return types.ErrCheckpointMalformed
	}
	if result.Successor == nil && result.State != types.RunStateCompleted && result.State != types.RunStateCanceled && result.State != types.RunStateFailed || result.Successor != nil && result.State != types.RunStateInputRequired {
		return types.ErrCheckpointMalformed
	}
	return nil
}
