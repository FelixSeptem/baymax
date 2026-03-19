package offline

import (
	"context"

	"github.com/FelixSeptem/baymax/core/types"
)

type OfflineModelAdapter struct{}

func (OfflineModelAdapter) Generate(context.Context, types.ModelRequest) (types.ModelResponse, error) {
	return types.ModelResponse{
		FinalAnswer: "offline model adapter placeholder",
	}, nil
}

func (OfflineModelAdapter) Stream(_ context.Context, _ types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	return onEvent(types.ModelEvent{
		Type:      types.ModelEventTypeFinalAnswer,
		TextDelta: "offline model adapter placeholder",
	})
}
