package main

import (
	"context"
	"fmt"
	"time"

	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
)

type streamingModel struct{}

func (m *streamingModel) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
	return types.ModelResponse{FinalAnswer: "unused"}, nil
}

func (m *streamingModel) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	for _, d := range []string{"hello ", "from ", "stream"} {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
			if err := onEvent(types.ModelEvent{Type: types.ModelEventTypeOutputTextDelta, TextDelta: d}); err != nil {
				return err
			}
		}
	}
	return nil
}

func main() {
	eng := runner.New(&streamingModel{})
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Millisecond)
	defer cancel()
	res, err := eng.Stream(ctx, types.RunRequest{Input: "interrupt me"}, nil)
	fmt.Printf("final=%q err=%v\n", res.FinalAnswer, err)
}
