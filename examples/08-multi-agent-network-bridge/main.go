package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/FelixSeptem/baymax/a2a"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/observability/event"
	"github.com/FelixSeptem/baymax/orchestration/composer"
	"github.com/FelixSeptem/baymax/orchestration/scheduler"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

type stubModel struct{}

func (stubModel) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
	_ = ctx
	_ = req
	return types.ModelResponse{FinalAnswer: "composer a2a child-run completed"}, nil
}

func (stubModel) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	_ = ctx
	_ = req
	return onEvent(types.ModelEvent{
		Type:      types.ModelEventTypeFinalAnswer,
		TextDelta: "composer stream a2a child-run completed",
	})
}

func main() {
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX"})
	if err != nil {
		panic(err)
	}
	defer func() { _ = mgr.Close() }()

	ctx := context.Background()
	runID := "ex08-run"
	dispatcher := event.NewDispatcher(
		event.NewJSONLoggerWithRuntimeManager(os.Stdout, mgr),
		event.NewRuntimeRecorder(mgr),
	)
	handler := eventHandlerFunc(func(ctx context.Context, ev types.Event) {
		dispatcher.Emit(ctx, ev)
	})

	server := a2a.NewInMemoryServer(a2a.HandlerFunc(func(ctx context.Context, req a2a.TaskRequest) (map[string]any, error) {
		_ = ctx
		return map[string]any{
			"output":      "remote-processed",
			"task_id":     req.TaskID,
			"workflow_id": req.WorkflowID,
			"step_id":     req.StepID,
		}, nil
	}), handler)
	client := a2a.NewClient(server, []a2a.AgentCard{
		{
			AgentID:                "agent-remote",
			PeerID:                 "peer-remote",
			SchemaVersion:          "a2a.v1.0",
			SupportedDeliveryModes: []string{a2a.DeliveryModeCallback, a2a.DeliveryModeSSE},
		},
	}, a2a.DeterministicRouter{RequireAll: true}, a2a.ClientPolicy{
		Timeout:            500 * time.Millisecond,
		RequestMaxAttempts: 1,
	}, handler)

	comp, err := composer.NewBuilder(stubModel{}).
		WithRuntimeManager(mgr).
		WithEventHandler(handler).
		WithA2AClient(client).
		Build()
	if err != nil {
		panic(err)
	}

	for i := 1; i <= 2; i++ {
		taskID := fmt.Sprintf("remote-task-%d", i)
		out, err := comp.DispatchChild(ctx, composer.ChildDispatchRequest{
			Task: scheduler.Task{
				TaskID:     taskID,
				RunID:      runID,
				WorkflowID: "wf-net-bridge",
				TeamID:     "team-net-bridge",
				StepID:     fmt.Sprintf("step-%d", i),
				AgentID:    "agent-main",
				PeerID:     "peer-remote",
				Payload: map[string]any{
					"input": fmt.Sprintf("payload-%d", i),
				},
			},
			Target:       composer.ChildTargetA2A,
			ParentDepth:  0,
			ChildTimeout: 800 * time.Millisecond,
			PollInterval: 20 * time.Millisecond,
		})
		if err != nil {
			panic(err)
		}
		fmt.Printf("dispatch task=%s commit_status=%s result=%v\n", taskID, out.Commit.Status, out.Commit.Result)
	}

	if _, err := comp.Run(ctx, types.RunRequest{RunID: runID, Input: "summarize a2a child-run results"}, nil); err != nil {
		panic(err)
	}
	runs := mgr.RecentRuns(1)
	if len(runs) > 0 {
		fmt.Printf("composer summary backend=%s fallback=%v child_total=%d child_failed=%d\n",
			runs[0].SchedulerBackend,
			runs[0].SchedulerBackendFallback,
			runs[0].SubagentChildTotal,
			runs[0].SubagentChildFailed,
		)
	}
}

type eventHandlerFunc func(ctx context.Context, ev types.Event)

func (f eventHandlerFunc) OnEvent(ctx context.Context, ev types.Event) {
	f(ctx, ev)
}
