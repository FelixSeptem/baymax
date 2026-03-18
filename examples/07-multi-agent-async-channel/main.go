package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/observability/event"
	"github.com/FelixSeptem/baymax/orchestration/composer"
	"github.com/FelixSeptem/baymax/orchestration/scheduler"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

type task struct {
	ID   string
	Text string
}

type stubModel struct{}

func (stubModel) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
	_ = ctx
	_ = req
	return types.ModelResponse{FinalAnswer: "composer local child-run completed"}, nil
}

func (stubModel) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	_ = ctx
	_ = req
	return onEvent(types.ModelEvent{
		Type:      types.ModelEventTypeFinalAnswer,
		TextDelta: "composer stream local child-run completed",
	})
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
	handler := eventHandlerFunc(func(ctx context.Context, ev types.Event) {
		dispatcher.Emit(ctx, ev)
	})

	comp, err := composer.NewBuilder(stubModel{}).
		WithRuntimeManager(mgr).
		WithEventHandler(handler).
		Build()
	if err != nil {
		panic(err)
	}

	coordinatorToWorker := make(chan task, 8)
	workerToCoordinator := make(chan string, 8)
	var wg sync.WaitGroup

	go func() {
		defer close(workerToCoordinator)
		for t := range coordinatorToWorker {
			out, err := comp.DispatchChild(ctx, composer.ChildDispatchRequest{
				Task: scheduler.Task{
					TaskID: t.ID,
					RunID:  runID,
					Payload: map[string]any{
						"text": t.Text,
					},
				},
				Target:               composer.ChildTargetLocal,
				ParentDepth:          0,
				ParentActiveChildren: 0,
				ChildTimeout:         300 * time.Millisecond,
				LocalRunner: composer.LocalChildRunnerFunc(func(ctx context.Context, task scheduler.Task) (map[string]any, error) {
					_ = ctx
					text, _ := task.Payload["text"].(string)
					time.Sleep(40 * time.Millisecond)
					return map[string]any{"output": "processed:" + text}, nil
				}),
			})
			if err != nil {
				workerToCoordinator <- "error:" + err.Error()
				wg.Done()
				continue
			}
			value, _ := out.Commit.Result["output"].(string)
			workerToCoordinator <- value
			wg.Done()
		}
	}()

	for i := 1; i <= 4; i++ {
		wg.Add(1)
		coordinatorToWorker <- task{
			ID:   fmt.Sprintf("t-%d", i),
			Text: fmt.Sprintf("payload-%d", i),
		}
	}
	close(coordinatorToWorker)

	results := make([]string, 0, 4)
	for r := range workerToCoordinator {
		results = append(results, r)
	}
	wg.Wait()
	if _, err := comp.Run(ctx, types.RunRequest{
		RunID: runID,
		Input: "summarize local child-run results",
	}, nil); err != nil {
		panic(err)
	}
	runs := mgr.RecentRuns(1)
	if len(runs) > 0 {
		fmt.Printf("composer summary backend=%s child_total=%d child_failed=%d budget_reject=%d\n",
			runs[0].SchedulerBackend,
			runs[0].SubagentChildTotal,
			runs[0].SubagentChildFailed,
			runs[0].SubagentBudgetRejectTotal,
		)
	}
	fmt.Printf("channel results=%v\n", results)
}

type eventHandlerFunc func(ctx context.Context, ev types.Event)

func (f eventHandlerFunc) OnEvent(ctx context.Context, ev types.Event) {
	f(ctx, ev)
}
