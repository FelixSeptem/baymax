package runner

import (
	"context"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

type noopEventHandler struct{}

func (noopEventHandler) OnEvent(_ context.Context, _ types.Event) {}

func benchmarkRunRequest() types.RunRequest {
	return types.RunRequest{
		RunID: "run-benchmark",
		Input: "hello",
		Messages: []types.Message{
			{Role: "user", Content: "hello"},
		},
	}
}

func BenchmarkRunnerLoopHotpathRun(b *testing.B) {
	engine := New(&fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			_ = ctx
			_ = req
			return types.ModelResponse{FinalAnswer: "ok"}, nil
		},
	})
	req := benchmarkRunRequest()
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := engine.Run(ctx, req, nil); err != nil {
			b.Fatalf("run failed: %v", err)
		}
	}
}

func BenchmarkRunnerLoopHotpathStream(b *testing.B) {
	engine := New(&fakeModel{
		stream: func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
			_ = ctx
			_ = req
			return onEvent(types.ModelEvent{
				Type:      types.ModelEventTypeOutputTextDelta,
				TextDelta: "ok",
			})
		},
	})
	req := benchmarkRunRequest()
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := engine.Stream(ctx, req, nil); err != nil {
			b.Fatalf("stream failed: %v", err)
		}
	}
}

func BenchmarkRunnerTimelineEmit(b *testing.B) {
	engine := New(&fakeModel{})
	handler := noopEventHandler{}
	runID := "run-benchmark"
	iteration := 1
	seq := int64(0)
	ctx := context.Background()

	engine.now = func() time.Time {
		return time.Unix(0, 0)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.emitTimeline(
			ctx,
			handler,
			runID,
			iteration,
			&seq,
			types.ActionPhaseRun,
			types.ActionStatusRunning,
			"",
		)
	}
}
