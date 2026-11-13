package instanceregistry

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MemoryRegistry is the in-process default. It keeps the same contract as the
// Redis registry so hosts and tests can run without an external backend;
// cross-instance liveness is out of scope by definition here.
type MemoryRegistry struct {
	mu         sync.Mutex
	instances  map[string]InstanceInfo
	expiry     map[string]time.Time
	bindings   map[string]string
	bindExpiry map[string]time.Time
}

func NewMemoryRegistry() *MemoryRegistry {
	return &MemoryRegistry{
		instances:  map[string]InstanceInfo{},
		expiry:     map[string]time.Time{},
		bindings:   map[string]string{},
		bindExpiry: map[string]time.Time{},
	}
}

func (m *MemoryRegistry) Register(_ context.Context, instanceID, endpoint string, ttl time.Duration) error {
	if m == nil {
		return fmt.Errorf("%w: registry is unavailable", ErrAffinityUnavailable)
	}
	id, err := normalizeInstanceID(instanceID)
	if err != nil {
		return err
	}
	if ttl <= 0 {
		return fmt.Errorf("instance registry: heartbeat ttl must be > 0")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().UTC()
	info, ok := m.instances[id]
	if ok {
		info.Endpoint = endpoint
		info.LastSeenAt = now
	} else {
		info = InstanceInfo{InstanceID: id, Endpoint: endpoint, StartedAt: now, LastSeenAt: now}
	}
	m.instances[id] = info
	m.expiry[id] = now.Add(ttl)
	return nil
}

func (m *MemoryRegistry) List(_ context.Context) ([]InstanceInfo, error) {
	if m == nil {
		return nil, fmt.Errorf("%w: registry is unavailable", ErrAffinityUnavailable)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().UTC()
	out := []InstanceInfo{}
	for id, deadline := range m.expiry {
		if now.After(deadline) {
			continue
		}
		out = append(out, m.instances[id])
	}
	return out, nil
}

func (m *MemoryRegistry) Unregister(_ context.Context, instanceID string) error {
	if m == nil {
		return fmt.Errorf("%w: registry is unavailable", ErrAffinityUnavailable)
	}
	id, err := normalizeInstanceID(instanceID)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.instances, id)
	delete(m.expiry, id)
	for session, owner := range m.bindings {
		if owner == id {
			delete(m.bindings, session)
			delete(m.bindExpiry, session)
		}
	}
	return nil
}

func (m *MemoryRegistry) BindSession(_ context.Context, sessionID, instanceID string, ttl time.Duration) error {
	if m == nil {
		return fmt.Errorf("%w: registry is unavailable", ErrAffinityUnavailable)
	}
	session, err := normalizeInstanceID(sessionID)
	if err != nil {
		return fmt.Errorf("instance registry: session id must be a bounded single token")
	}
	id, err := normalizeInstanceID(instanceID)
	if err != nil {
		return err
	}
	if ttl <= 0 {
		return fmt.Errorf("instance registry: affinity ttl must be > 0")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.instances[id]; !ok {
		return fmt.Errorf("%w: %s", ErrUnknownInstance, id)
	}
	m.bindings[session] = id
	m.bindExpiry[session] = time.Now().UTC().Add(ttl)
	return nil
}

func (m *MemoryRegistry) ResolveSession(_ context.Context, sessionID string) (string, error) {
	if m == nil {
		return "", fmt.Errorf("%w: registry is unavailable", ErrAffinityUnavailable)
	}
	session, err := normalizeInstanceID(sessionID)
	if err != nil {
		return "", fmt.Errorf("instance registry: session id must be a bounded single token")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().UTC()
	if deadline, ok := m.bindExpiry[session]; !ok || now.After(deadline) {
		delete(m.bindings, session)
		delete(m.bindExpiry, session)
		return "", ErrSessionUnbound
	}
	owner := m.bindings[session]
	if deadline, ok := m.expiry[owner]; !ok || now.After(deadline) {
		return "", fmt.Errorf("%w: owning instance %s is gone", ErrAffinityLost, owner)
	}
	return owner, nil
}

// Compile-time interface check.
var _ Registry = (*MemoryRegistry)(nil)
