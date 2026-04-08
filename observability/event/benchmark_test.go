package event

import (
	"context"
	"fmt"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

func BenchmarkRuntimeExporterBatch(b *testing.B) {
	benchmarks := []struct {
		name            string
		maxBatchSize    int
		maxFlushLatency time.Duration
	}{
		{
			name:            "size_32",
			maxBatchSize:    32,
			maxFlushLatency: time.Second,
		},
		{
			name:            "latency_1ms",
			maxBatchSize:    1 << 20,
			maxFlushLatency: time.Millisecond,
		},
	}
	for _, bm := range benchmarks {
		bm := bm
		b.Run(bm.name, func(b *testing.B) {
			exporter := &benchmarkRuntimeExporter{}
			runtime := newRuntimeExporterRuntime(nil)
			queueCapacity := bm.maxBatchSize * 4
			if queueCapacity < 64 {
				queueCapacity = 64
			}
			if queueCapacity > 4096 {
				queueCapacity = 4096
			}
			queue := make(chan types.Event, queueCapacity)
			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan struct{})
			go func() {
				defer close(done)
				runtime.worker(
					ctx,
					exporter,
					queue,
					runtimeconfig.RuntimeObservabilityExportProfileCustom,
					bm.maxBatchSize,
					bm.maxFlushLatency,
				)
			}()

			ev := types.Event{
				Version: types.EventSchemaVersionV1,
				Type:    "benchmark.runtime_exporter_batch",
				RunID:   "bench-exporter",
				Time:    time.Unix(0, 0).UTC(),
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				queue <- ev
			}
			b.StopTimer()

			cancel()
			<-done
			if got := exporter.TotalEvents(); got != int64(b.N) {
				b.Fatalf("exported events=%d, want %d", got, b.N)
			}
		})
	}
}

func BenchmarkEventDispatcherFanout(b *testing.B) {
	fanouts := []int{1, 2, 4}
	for _, fanout := range fanouts {
		fanout := fanout
		b.Run(fmt.Sprintf("fanout_%d", fanout), func(b *testing.B) {
			handlers := []types.EventHandler{
				benchmarkEventHandler(func(context.Context, types.Event) {}),
				benchmarkEventHandler(func(context.Context, types.Event) {}),
				benchmarkEventHandler(func(context.Context, types.Event) {}),
				benchmarkEventHandler(func(context.Context, types.Event) {}),
			}
			dispatcher := NewDispatcherWithOptions(DispatcherOptions{Fanout: fanout}, handlers...)
			ev := types.Event{
				Version: types.EventSchemaVersionV1,
				Type:    "benchmark.dispatcher_fanout",
				RunID:   "bench-dispatcher",
				Time:    time.Unix(0, 0).UTC(),
			}
			ctx := context.Background()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				dispatcher.Emit(ctx, ev)
			}
		})
	}
}

func BenchmarkJSONLoggerEmit(b *testing.B) {
	manager, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX_BENCH_LOGGER"})
	if err != nil {
		b.Fatalf("NewManager failed: %v", err)
	}
	b.Cleanup(func() { _ = manager.Close() })

	payload := map[string]any{
		"status":       "success",
		"latency_ms":   int64(12),
		"tool_calls":   2,
		"access_token": "benchmark-secret",
	}
	ev := types.Event{
		Version:   types.EventSchemaVersionV1,
		Type:      "benchmark.json_logger_emit",
		RunID:     "bench-logger",
		Time:      time.Unix(0, 0).UTC(),
		TraceID:   "trace-bench",
		SpanID:    "span-bench",
		Iteration: 1,
		Payload:   payload,
	}
	ctx := context.Background()

	b.Run("plain", func(b *testing.B) {
		logger := NewJSONLogger(io.Discard)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			logger.OnEvent(ctx, ev)
		}
	})

	b.Run("with_runtime_manager", func(b *testing.B) {
		logger := NewJSONLoggerWithRuntimeManager(io.Discard, manager)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			logger.OnEvent(ctx, ev)
		}
	})
}

type benchmarkRuntimeExporter struct {
	total atomic.Int64
}

func (e *benchmarkRuntimeExporter) ExportEvents(_ context.Context, events []types.Event) error {
	e.total.Add(int64(len(events)))
	return nil
}

func (e *benchmarkRuntimeExporter) Flush(_ context.Context) error {
	return nil
}

func (e *benchmarkRuntimeExporter) Shutdown(_ context.Context) error {
	return nil
}

func (e *benchmarkRuntimeExporter) TotalEvents() int64 {
	return e.total.Load()
}

type benchmarkEventHandler func(context.Context, types.Event)

func (f benchmarkEventHandler) OnEvent(ctx context.Context, ev types.Event) {
	if f == nil {
		return
	}
	f(ctx, ev)
}
