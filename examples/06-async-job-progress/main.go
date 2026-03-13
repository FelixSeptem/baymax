package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/observability/event"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

func main() {
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX"})
	if err != nil {
		panic(err)
	}
	defer func() { _ = mgr.Close() }()

	runID := "ex06-run"
	ctx := context.Background()
	dispatcher := event.NewDispatcher(
		event.NewJSONLoggerWithRuntimeManager(os.Stdout, mgr),
		event.NewRuntimeRecorder(mgr),
	)

	jobs := []int{120, 90, 140, 70}
	results := make(chan int, len(jobs))
	progress := make(chan int, len(jobs))
	var wg sync.WaitGroup

	emit := func(t string, payload map[string]any) {
		dispatcher.Emit(ctx, types.Event{
			Version: types.EventSchemaVersionV1,
			Type:    t,
			RunID:   runID,
			Time:    time.Now(),
			Payload: payload,
		})
	}

	emit("job.started", map[string]any{"total": len(jobs)})
	for i, ms := range jobs {
		wg.Add(1)
		go func(idx, delayMs int) {
			defer wg.Done()
			time.Sleep(time.Duration(delayMs) * time.Millisecond)
			results <- delayMs
			progress <- idx
		}(i, ms)
	}

	go func() {
		wg.Wait()
		close(results)
		close(progress)
	}()

	done := 0
	total := 0
	for range progress {
		done++
		emit("job.progress", map[string]any{
			"done":    done,
			"total":   len(jobs),
			"percent": float64(done) * 100 / float64(len(jobs)),
		})
	}
	for v := range results {
		total += v
	}

	emit("job.completed", map[string]any{"avg_latency_ms": total / len(jobs)})
	fmt.Printf("jobs=%d avg_latency_ms=%d\n", len(jobs), total/len(jobs))
}
