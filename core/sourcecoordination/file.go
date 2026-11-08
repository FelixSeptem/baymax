package sourcecoordination

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

const stateFileName = "source-state.json"

type fileState struct {
	Version        string    `json:"version"`
	OwnerNamespace string    `json:"owner_namespace"`
	Revision       uint64    `json:"revision"`
	Payload        []byte    `json:"payload"`
	Fence          string    `json:"fence,omitempty"`
	WriterID       string    `json:"writer_id,omitempty"`
	ExpiresAt      time.Time `json:"expires_at,omitempty"`
}

type File struct {
	mu     sync.Mutex
	dir    string
	opts   Options
	state  fileState
	closed bool
}

func OpenFile(dir string, opts Options) (*File, error) {
	if dir == "" || opts.OwnerNamespace == "" {
		return nil, types.ErrOwnerUnavailable
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, types.ErrOwnerUnavailable
	}
	f := &File{dir: dir, opts: opts}
	if err := f.load(); err != nil {
		return nil, err
	}
	return f, nil
}

func (f *File) Acquire(ctx context.Context, writerID string, ttl time.Duration) (Lease, error) {
	if err := ctx.Err(); err != nil {
		return Lease{}, err
	}
	if f == nil || writerID == "" || ttl <= 0 {
		return Lease{}, types.ErrOwnerUnavailable
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	unlock, err := f.lockAndReload()
	if err != nil {
		return Lease{}, err
	}
	fence, err := randomFence()
	if err != nil {
		unlock()
		return Lease{}, types.ErrOwnerUnavailable
	}
	now := f.opts.Now().UTC()
	f.state.Fence, f.state.WriterID, f.state.ExpiresAt = fence, writerID, now.Add(ttl)
	if err := f.persistLocked(); err != nil {
		unlock()
		return Lease{}, err
	}
	lease := Lease{Version: ContractVersion, OwnerNamespace: f.opts.OwnerNamespace, WriterID: writerID, Fence: fence, ExpiresAt: f.state.ExpiresAt}
	unlock()
	return lease, nil
}

func (f *File) Read(ctx context.Context) (Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	unlock, err := f.lockAndReload()
	if err != nil {
		return Snapshot{}, err
	}
	snapshot := Snapshot{Revision: f.state.Revision, Payload: append([]byte(nil), f.state.Payload...)}
	unlock()
	return snapshot, nil
}

func (f *File) CompareAndSwap(ctx context.Context, lease Lease, revision uint64, payload []byte) (Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	if len(payload) > maxPayloadBytes {
		return Snapshot{}, types.ErrOwnerUnavailable
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	unlock, err := f.lockAndReload()
	if err != nil {
		return Snapshot{}, err
	}
	if err := validateLease(f.opts.OwnerNamespace, f.stateLease(), lease, f.opts.Now().UTC()); err != nil {
		unlock()
		return Snapshot{}, err
	}
	if revision != f.state.Revision {
		unlock()
		return Snapshot{}, types.ErrDeploymentConflict
	}
	old := f.state
	f.state.Revision++
	f.state.Payload = append([]byte(nil), payload...)
	if err := f.persistLocked(); err != nil {
		f.state = old
		unlock()
		return Snapshot{}, err
	}
	snapshot := Snapshot{Revision: f.state.Revision, Payload: append([]byte(nil), f.state.Payload...)}
	unlock()
	return snapshot, nil
}

func (f *File) ValidateLease(ctx context.Context, lease Lease) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if f == nil {
		return types.ErrOwnerUnavailable
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	unlock, err := f.lockAndReload()
	if err != nil {
		return err
	}
	err = validateLease(f.opts.OwnerNamespace, f.stateLease(), lease, f.opts.Now().UTC())
	unlock()
	return err
}

func (f *File) DeploymentScope() types.DeploymentScope {
	if f == nil {
		return types.DefaultDeploymentScope("source-coordinator", "source coordinator is unavailable")
	}
	return types.CoordinatedMultiInstanceScope(f.opts.OwnerNamespace, "fixture:durable-source-store-coordination.v1")
}

func (f *File) Close() error {
	if f == nil {
		return nil
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return nil
	}
	f.closed = true
	return nil
}

func (f *File) checkLocked() error {
	if f.closed {
		return types.ErrOwnerUnavailable
	}
	return nil
}

func (f *File) stateLease() Lease {
	return Lease{Version: ContractVersion, OwnerNamespace: f.state.OwnerNamespace, WriterID: f.state.WriterID, Fence: f.state.Fence, ExpiresAt: f.state.ExpiresAt}
}

func (f *File) lockAndReload() (func(), error) {
	if err := f.checkLocked(); err != nil {
		return func() {}, err
	}
	lock, err := acquireFileLock(filepath.Join(f.dir, ".source-owner.lock"))
	if err != nil {
		return func() {}, types.ErrOwnerUnavailable
	}
	if err := f.load(); err != nil {
		_ = releaseFileLock(lock)
		return func() {}, err
	}
	return func() { _ = releaseFileLock(lock) }, nil
}

func (f *File) load() error {
	raw, err := os.ReadFile(filepath.Join(f.dir, stateFileName))
	if os.IsNotExist(err) {
		f.state = fileState{Version: ContractVersion, OwnerNamespace: f.opts.OwnerNamespace}
		return nil
	}
	if err != nil {
		return types.ErrOwnerUnavailable
	}
	var state fileState
	if json.Unmarshal(raw, &state) != nil || state.Version != ContractVersion || state.OwnerNamespace != f.opts.OwnerNamespace || len(state.Payload) > maxPayloadBytes {
		return types.ErrOwnerUnavailable
	}
	f.state = state
	return nil
}

func (f *File) persistLocked() error {
	raw, err := json.Marshal(f.state)
	if err != nil {
		return types.ErrOwnerUnavailable
	}
	tmp := filepath.Join(f.dir, stateFileName+".tmp")
	path := filepath.Join(f.dir, stateFileName)
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return types.ErrOwnerUnavailable
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return types.ErrOwnerUnavailable
	}
	return nil
}
