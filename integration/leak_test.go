package integration

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/integration/fakes"
)

func TestNoGoroutineLeakOnCancelStorm(t *testing.T) {
	before := runtime.NumGoroutine()
	model := fakes.NewModel(nil)
	model.SetStream([]types.ModelEvent{
		{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "a"},
		{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "b"},
	}, nil)
	eng := runner.New(model)

	for i := 0; i < 40; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), time.Microsecond)
		_, _ = eng.Stream(ctx, types.RunRequest{Input: "storm"}, nil)
		cancel()
	}

	time.Sleep(100 * time.Millisecond)
	after := runtime.NumGoroutine()
	if after > before+10 {
		t.Fatalf("possible goroutine leak: before=%d after=%d", before, after)
	}
}
