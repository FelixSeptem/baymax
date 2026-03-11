package main

import (
	"context"
	"fmt"

	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
)

type minimalModel struct{}

func (m *minimalModel) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
	return types.ModelResponse{FinalAnswer: "hello from 01-chat-minimal"}, nil
}

func (m *minimalModel) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	return onEvent(types.ModelEvent{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "hello from stream"})
}

func main() {
	eng := runner.New(&minimalModel{})
	res, err := eng.Run(context.Background(), types.RunRequest{Input: "say hi"}, nil)
	if err != nil {
		panic(err)
	}
	fmt.Println(res.FinalAnswer)
}
