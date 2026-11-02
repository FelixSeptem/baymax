// Package sourcecoordination provides source-owner scoped fencing and CAS for
// durable source state. It coordinates detached facts only; execution remains
// owned by the runner, checkpoint, history or orchestration source.
package sourcecoordination

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

const (
	ContractVersion = "source_coordination.v1"
	maxPayloadBytes = 1 << 20
)

// Lease is an opaque source-owner writer fence. It never grants execution
// authority; callers still validate source-specific state before publishing.
type Lease struct {
	Version        string    `json:"version"`
	OwnerNamespace string    `json:"owner_namespace"`
	WriterID       string    `json:"writer_id"`
	Fence          string    `json:"fence"`
	ExpiresAt      time.Time `json:"expires_at"`
}

// Snapshot is an immutable detached source fact at one coordinator revision.
type Snapshot struct {
	Revision uint64 `json:"revision"`
	Payload  []byte `json:"payload"`
}

type Options struct {
	OwnerNamespace string
	Now            func() time.Time
}

// Coordinator fences one source-owner namespace and atomically publishes
// revisions. Implementations may persist this contract externally.
type Coordinator interface {
	Acquire(context.Context, string, time.Duration) (Lease, error)
	Read(context.Context) (Snapshot, error)
	CompareAndSwap(context.Context, Lease, uint64, []byte) (Snapshot, error)
	DeploymentScope() types.DeploymentScope
}

// LeaseValidator is implemented by coordinators that can prove a previously
// acquired fence is still current without mutating source state.
type LeaseValidator interface {
	ValidateLease(context.Context, Lease) error
}

type Memory struct {
	mu    sync.Mutex
	opts  Options
	lease Lease
	snap  Snapshot
}

func NewMemory(opts Options) (*Memory, error) {
	if opts.OwnerNamespace == "" {
		return nil, types.ErrOwnerUnavailable
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	return &Memory{opts: opts}, nil
}

func (m *Memory) Acquire(ctx context.Context, writerID string, ttl time.Duration) (Lease, error) {
	if err := ctx.Err(); err != nil {
		return Lease{}, err
	}
	if m == nil || writerID == "" || ttl <= 0 {
		return Lease{}, types.ErrOwnerUnavailable
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.opts.Now().UTC()
	if !m.lease.ExpiresAt.IsZero() && now.Before(m.lease.ExpiresAt) && m.lease.WriterID != writerID {
		return Lease{}, types.ErrOwnerUnavailable
	}
	fence, err := randomFence()
	if err != nil {
		return Lease{}, types.ErrOwnerUnavailable
	}
	m.lease = Lease{Version: ContractVersion, OwnerNamespace: m.opts.OwnerNamespace, WriterID: writerID, Fence: fence, ExpiresAt: now.Add(ttl)}
	return m.lease, nil
}

func (m *Memory) Read(ctx context.Context) (Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	if m == nil {
		return Snapshot{}, types.ErrOwnerUnavailable
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return cloneSnapshot(m.snap), nil
}

func (m *Memory) CompareAndSwap(ctx context.Context, lease Lease, revision uint64, payload []byte) (Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	if m == nil || len(payload) > maxPayloadBytes {
		return Snapshot{}, types.ErrOwnerUnavailable
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if lease.OwnerNamespace != m.opts.OwnerNamespace {
		return Snapshot{}, types.ErrForeignWriter
	}
	now := m.opts.Now().UTC()
	if m.lease.Fence == "" || m.lease.Fence != lease.Fence || m.lease.WriterID != lease.WriterID || now.After(m.lease.ExpiresAt) {
		return Snapshot{}, types.ErrStaleWriter
	}
	if revision != m.snap.Revision {
		return Snapshot{}, types.ErrDeploymentConflict
	}
	m.snap = Snapshot{Revision: revision + 1, Payload: append([]byte(nil), payload...)}
	return cloneSnapshot(m.snap), nil
}

func (m *Memory) ValidateLease(ctx context.Context, lease Lease) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if m == nil {
		return types.ErrOwnerUnavailable
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return validateLease(m.opts.OwnerNamespace, m.lease, lease, m.opts.Now().UTC())
}

func (m *Memory) DeploymentScope() types.DeploymentScope {
	if m == nil {
		return types.DefaultDeploymentScope("source-coordinator", "source coordinator is unavailable")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.lease.Fence == "" || !m.opts.Now().UTC().Before(m.lease.ExpiresAt) {
		return types.DefaultDeploymentScope(m.opts.OwnerNamespace, "source coordinator lease is unavailable")
	}
	return types.DefaultDeploymentScope(m.opts.OwnerNamespace, "memory coordinator is process-local and cannot advertise durable multi-writer support")
}

func cloneSnapshot(in Snapshot) Snapshot {
	in.Payload = append([]byte(nil), in.Payload...)
	return in
}

func randomFence() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return "fence:" + hex.EncodeToString(raw[:]), nil
}

func validateLease(owner string, current, requested Lease, now time.Time) error {
	if requested.OwnerNamespace != owner {
		return types.ErrForeignWriter
	}
	if current.Fence == "" || current.Fence != requested.Fence || current.WriterID != requested.WriterID || !now.Before(current.ExpiresAt) {
		return types.ErrStaleWriter
	}
	return nil
}
