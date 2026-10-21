package runner

import (
	"github.com/FelixSeptem/baymax/core/history"
	"github.com/FelixSeptem/baymax/core/types"
)

// NewNativeEventHistory creates the runner-owned source journal. The journal
// is observational; callers append source facts and replay them separately
// from Run/Stream execution.
func NewNativeEventHistory(source types.SourceRef, options ...history.Option) (*history.Journal, error) {
	if source.Kind != types.HistorySourceRunner {
		return nil, types.ErrHistoryInvalid
	}
	return history.New(source, options...)
}

func OpenNativeEventHistory(path string, source types.SourceRef, options ...history.Option) (*history.Journal, error) {
	if source.Kind != types.HistorySourceRunner {
		return nil, types.ErrHistoryInvalid
	}
	return history.OpenFile(path, source, options...)
}
