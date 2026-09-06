package extension

import (
	"fmt"
	"sync"
)

type Generation struct {
	Generation ActivationGeneration
	Descriptor Descriptor
}

type GenerationManager struct {
	mu     sync.RWMutex
	active Generation
	state  map[string]any
}

func NewGenerationManager() *GenerationManager {
	return &GenerationManager{state: map[string]any{}}
}

func (m *GenerationManager) Reload(descriptor Descriptor) (Generation, error) {
	validated, err := ValidateDescriptor(descriptor)
	if err != nil {
		return Generation{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	next := m.active.Generation + 1
	m.active = Generation{Generation: next, Descriptor: validated}
	return m.active, nil
}

func (m *GenerationManager) ActiveGeneration() ActivationGeneration {
	if m == nil {
		return 0
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.active.Generation
}

func (m *GenerationManager) AcceptsEvent(generation ActivationGeneration) bool {
	return m != nil && generation != 0 && generation == m.ActiveGeneration()
}

func (m *GenerationManager) SetState(key string, value any) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.state == nil {
		m.state = map[string]any{}
	}
	m.state[key] = value
}

func (m *GenerationManager) State(key string) any {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state[key]
}

func (m *GenerationManager) Snapshot() map[string]any {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]any, len(m.state))
	for key, value := range m.state {
		out[key] = value
	}
	return out
}

func (m *GenerationManager) Commit(values map[string]any) error {
	if m == nil {
		return fmt.Errorf("generation manager is nil")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.state == nil {
		m.state = map[string]any{}
	}
	for key, value := range values {
		m.state[key] = value
	}
	return nil
}
