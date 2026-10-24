package history

import (
	"fmt"
	"sync"

	"github.com/FelixSeptem/baymax/core/types"
)

// AppendHealth retains the most recent source history append failure for an
// owner that emits callback projections asynchronously from its state changes.
type AppendHealth struct {
	mu     sync.RWMutex
	err    error
	source types.SourceRef
}

func (h *AppendHealth) Record(source types.SourceRef, err error) {
	if h == nil || err == nil {
		return
	}
	h.mu.Lock()
	h.err = err
	h.source = source
	h.mu.Unlock()
}

func (h *AppendHealth) Err() error {
	if h == nil {
		return nil
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.err == nil {
		return nil
	}
	return fmt.Errorf("%w: %s/%s/%s: %v", types.ErrSourceUnavailable, h.source.Kind, h.source.Namespace, h.source.ID, h.err)
}
