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

type clarificationRequest struct {
	RequestID      string
	Questions      []string
	ContextSummary string
	TimeoutMs      int64
}

type clarificationResponse struct {
	RequestID string
	Answers   []string
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
	clarifyReq := make(chan clarificationRequest, 1)
	clarifyResp := make(chan clarificationResponse, 1)
	done := make(chan struct{})
	clarifierDone := make(chan struct{})

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

	go func() {
		defer close(clarifierDone)
		for req := range clarifyReq {
			emit("hitl.clarification.requested", map[string]any{
				"clarification_request": map[string]any{
					"request_id":      req.RequestID,
					"questions":       req.Questions,
					"context_summary": req.ContextSummary,
					"timeout_ms":      req.TimeoutMs,
				},
			})
			time.Sleep(20 * time.Millisecond)
			clarifyResp <- clarificationResponse{
				RequestID: req.RequestID,
				Answers:   []string{"priority=high"},
			}
		}
		close(clarifyResp)
	}()

	request := clarificationRequest{
		RequestID:      "clarify-1",
		Questions:      []string{"Please provide processing priority"},
		ContextSummary: "worker needs priority before dispatch",
		TimeoutMs:      5000,
	}
	emit(types.EventTypeActionTimeline, map[string]any{
		"phase":    string(types.ActionPhaseHITL),
		"status":   string(types.ActionStatusPending),
		"reason":   "hitl.await_user",
		"sequence": int64(1),
	})
	clarifyReq <- request
	reply := <-clarifyResp
	emit(types.EventTypeActionTimeline, map[string]any{
		"phase":    string(types.ActionPhaseHITL),
		"status":   string(types.ActionStatusSucceeded),
		"reason":   "hitl.resumed",
		"sequence": int64(2),
	})
	emit("hitl.clarification.resumed", map[string]any{
		"request_id": reply.RequestID,
		"answers":    reply.Answers,
	})

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
	close(clarifyReq)
	<-clarifierDone

	emit("agent.coordinator.completed", map[string]any{"results": len(results)})
	fmt.Printf("channel results=%v\n", results)
}
