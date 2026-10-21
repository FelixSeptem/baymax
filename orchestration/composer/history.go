package composer

import (
	"github.com/FelixSeptem/baymax/core/history"
	"github.com/FelixSeptem/baymax/core/types"
)

// NewNativeEventHistory creates the isolated-child source journal.
func NewNativeEventHistory(source types.SourceRef, options ...history.Option) (*history.Journal, error) {
	if source.Kind != types.HistorySourceChild {
		return nil, types.ErrHistoryInvalid
	}
	return history.New(source, options...)
}

func OpenNativeEventHistory(path string, source types.SourceRef, options ...history.Option) (*history.Journal, error) {
	if source.Kind != types.HistorySourceChild {
		return nil, types.ErrHistoryInvalid
	}
	return history.OpenFile(path, source, options...)
}
