package fixture

import (
	"context"

	"github.com/FelixSeptem/baymax/core/types"
)

type FixtureModelAdapter struct{}

func (FixtureModelAdapter) Generate(context.Context, types.ModelRequest) (types.ModelResponse, error) {
	return types.ModelResponse{
		FinalAnswer: "fixture model adapter placeholder",
	}, nil
}

func (FixtureModelAdapter) Stream(_ context.Context, _ types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	return onEvent(types.ModelEvent{
		Type:      types.ModelEventTypeFinalAnswer,
		TextDelta: "fixture model adapter placeholder",
	})
}
