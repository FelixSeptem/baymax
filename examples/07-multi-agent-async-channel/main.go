package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/observability/event"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

type task struct {
	ID   string
	Text string
}

func main() {
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX"})
	if err != nil {
		panic(err)
	}
	defer func() { _ = mgr.Close() }()

	runID := "ex07-run"
	ctx := context.Background()
	dispatcher := event.NewDispatcher(
		event.NewJSONLoggerWithRuntimeManager(os.Stdout, mgr),
		event.NewRuntimeRecorder(mgr),
	)
	emit := func(t string, payload map[string]any) {
		dispatcher.Emit(ctx, types.Event{
			Version: types.EventSchemaVersionV1,
			Type:    t,
			RunID:   runID,
			Time:    time.Now(),
			Payload: payload,
		})
	}

	coordinatorToWorker := make(chan task, 4)
	workerToCoordinator := make(chan string, 4)
	done := make(chan struct{})

	go func() {
		defer close(done)
		for t := range coordinatorToWorker {
			emit("agent.worker.started", map[string]any{"task_id": t.ID})
			time.Sleep(50 * time.Millisecond)
			workerToCoordinator <- "processed:" + t.Text
			emit("agent.worker.completed", map[string]any{"task_id": t.ID})
		}
		close(workerToCoordinator)
	}()

	emit("agent.coordinator.dispatch", map[string]any{"count": 3})
	for i := 1; i <= 3; i++ {
		coordinatorToWorker <- task{
			ID:   fmt.Sprintf("t-%d", i),
			Text: fmt.Sprintf("payload-%d", i),
		}
	}
	close(coordinatorToWorker)

	results := make([]string, 0, 3)
	for r := range workerToCoordinator {
		results = append(results, r)
		emit("agent.coordinator.collect", map[string]any{"result": r})
	}
	<-done

	emit("agent.coordinator.completed", map[string]any{"results": len(results)})
	fmt.Printf("channel results=%v\n", results)
}
