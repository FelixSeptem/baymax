package main

import (
	"context"
	"fmt"

	"github.com/FelixSeptem/baymax/adapter/modelcapability"
	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
)

// customModelAdapter is a minimal skeleton that satisfies types.ModelClient.
type customModelAdapter struct{}

func (customModelAdapter) Generate(context.Context, types.ModelRequest) (types.ModelResponse, error) {
	return types.ModelResponse{
		FinalAnswer: "model-adapter-template: hello",
	}, nil
}

func (customModelAdapter) Stream(_ context.Context, _ types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	return onEvent(types.ModelEvent{
		Type:      types.ModelEventTypeFinalAnswer,
		TextDelta: "model-adapter-template: stream-hello",
	})
}

func main() {
	model, err := modelcapability.Wrap(customModelAdapter{}, modelcapability.Config{
		Provider:     "model-adapter-template",
		Capabilities: []types.ModelCapability{types.ModelCapabilityStreaming},
	})
	if err != nil {
		panic(err)
	}
	eng := runner.New(model)
	res, err := eng.Stream(context.Background(), types.RunRequest{
		Input: "ping model adapter",
	}, nil)
	if err != nil {
		panic(err)
	}
	fmt.Println(res.FinalAnswer)
}
