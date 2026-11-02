package sourceadapter

import (
	"context"
	"encoding/json"
	"math"
	"sync"
)

const StateSchema = "source_adapter_state.v1"

// Receipt retains only bounded references, never a payload or business outcome.
type Receipt struct {
	ExternalID    string `json:"external_id"`
	Reference     string `json:"reference"`
	PayloadDigest string `json:"payload_digest"`
	Next          Cursor `json:"next"`
	Fingerprint   string `json:"fingerprint"`
}

type State struct {
	Schema    string    `json:"schema"`
	Metadata  Metadata  `json:"metadata"`
	Revision  uint64    `json:"revision"`
	Confirmed *Cursor   `json:"confirmed,omitempty"`
	Pending   *Receipt  `json:"pending,omitempty"`
	Recent    []Receipt `json:"recent,omitempty"`
}

func InitialState(m Metadata) (State, error) {
	if err := m.Validate(); err != nil {
		return State{}, err
	}
	return State{Schema: StateSchema, Metadata: m}, nil
}

func (r Receipt) validate(m Metadata) error {
	if err := r.Next.validate(m); err != nil {
		return err
	}
	if !validRef(r.ExternalID, MaxReferenceBytes) || !validRef(r.Reference, MaxReferenceBytes) || !validDigest(r.PayloadDigest) || !validDigest(r.Fingerprint) {
		return ErrInvalid
	}
	want := r.Fingerprint
	r.Fingerprint = ""
	b, err := json.Marshal(r)
	if err != nil || digest(b) != want {
		return ErrInvalid
	}
	return nil
}

func (s State) Validate(m Metadata) error {
	if err := m.Validate(); err != nil {
		return err
	}
	if s.Metadata != m {
		return ErrForeign
	}
	if s.Schema != StateSchema || len(s.Recent) > MaxRecentReceipts {
		return ErrInvalid
	}
	if s.Confirmed != nil {
		if err := s.Confirmed.validate(m); err != nil {
			return err
		}
	}
	seen := make(map[string]bool, len(s.Recent))
	for _, r := range s.Recent {
		if err := r.validate(m); err != nil {
			return err
		}
		if seen[r.ExternalID] {
			return ErrConflict
		}
		seen[r.ExternalID] = true
	}
	if (s.Confirmed == nil) != (len(s.Recent) == 0) {
		return ErrInvalid
	}
	if len(s.Recent) > 0 && *s.Confirmed != s.Recent[len(s.Recent)-1].Next {
		return ErrInvalid
	}
	if s.Pending != nil {
		if err := s.Pending.validate(m); err != nil {
			return err
		}
		for _, r := range s.Recent {
			if r.ExternalID == s.Pending.ExternalID && r != *s.Pending {
				return ErrConflict
			}
		}
	}
	if s.Revision == 0 && (s.Confirmed != nil || s.Pending != nil) {
		return ErrInvalid
	}
	return nil
}

func cloneState(s State) State {
	s.Confirmed = cloneCursor(s.Confirmed)
	if s.Pending != nil {
		p := *s.Pending
		s.Pending = &p
	}
	s.Recent = append([]Receipt(nil), s.Recent...)
	return s
}

// Prepare is pure. Persist its result before invoking the consumer.
func Prepare(s State, f Fact) (State, error) {
	r, err := receipt(s.Metadata, f)
	if err != nil {
		return State{}, err
	}
	return PrepareReceipt(s, r)
}

// PrepareReceipt is the pure reference-only transition shared with offline
// replay. Live delivery must use Prepare to validate the actual payload too.
func PrepareReceipt(s State, r Receipt) (State, error) {
	if err := s.Validate(s.Metadata); err != nil {
		return State{}, err
	}
	if err := r.validate(s.Metadata); err != nil {
		return State{}, err
	}
	if s.Pending != nil && *s.Pending != r {
		return State{}, ErrConflict
	}
	for _, old := range s.Recent {
		if old.ExternalID == r.ExternalID && old != r {
			return State{}, ErrConflict
		}
	}
	// A delivery needs both pending and confirmation CAS revisions. Fail
	// before any effect when the pair cannot be represented.
	if s.Revision >= math.MaxUint64-1 {
		return State{}, ErrConflict
	}
	n := cloneState(s)
	n.Revision++
	n.Pending = &r
	return n, nil
}

// Confirm is pure. A retryable disposition never creates a new state. An
// identical recent repeat does not roll the confirmed cursor backward.
func Confirm(s State, d Disposition) (State, error) {
	if err := s.Validate(s.Metadata); err != nil {
		return State{}, err
	}
	if d == Retryable {
		return State{}, ErrRetryable
	}
	if d != Sent && d != Gone {
		return State{}, ErrInvalid
	}
	if s.Pending == nil || s.Revision == math.MaxUint64 {
		return State{}, ErrConflict
	}
	n := cloneState(s)
	n.Revision++
	r := *n.Pending
	n.Pending = nil
	for _, old := range n.Recent {
		if old.ExternalID == r.ExternalID {
			return n, nil
		}
	}
	n.Confirmed = cloneCursor(&r.Next)
	n.Recent = append(n.Recent, r)
	if len(n.Recent) > MaxRecentReceipts {
		n.Recent = append([]Receipt(nil), n.Recent[len(n.Recent)-MaxRecentReceipts:]...)
	}
	return n, nil
}

// MemoryStore supports CAS within this instance, and is process-local only.
type MemoryStore struct {
	mu    sync.Mutex
	state State
}

func NewMemoryStore(m Metadata) (*MemoryStore, error) {
	s, err := InitialState(m)
	if err != nil {
		return nil, err
	}
	return &MemoryStore{state: s}, nil
}

func (s *MemoryStore) Load(ctx context.Context) (State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return State{}, err
	}
	return cloneState(s.state), nil
}

func validateCAS(current State, expected uint64, next State) error {
	if current.Revision != expected || expected == math.MaxUint64 || next.Revision != expected+1 {
		return ErrConflict
	}
	return next.Validate(current.Metadata)
}

func (s *MemoryStore) CompareAndSwap(ctx context.Context, expected uint64, next State) (State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return State{}, err
	}
	if err := validateCAS(s.state, expected, next); err != nil {
		return State{}, err
	}
	s.state = cloneState(next)
	return cloneState(s.state), nil
}
