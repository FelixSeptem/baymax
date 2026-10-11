package main

import (
	"context"
	"fmt"

	"github.com/FelixSeptem/baymax/adapter/modelcapability"
	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/host/eventadapter"
	"github.com/FelixSeptem/baymax/tool/diagnosticsreplay"
	"github.com/FelixSeptem/baymax/tool/local"
)

type fakeModel struct{}

func (fakeModel) Generate(_ context.Context, req types.ModelRequest) (types.ModelResponse, error) {
	if len(req.ToolResult) > 0 {
		return types.ModelResponse{FinalAnswer: "fake completion"}, nil
	}
	return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "fake-call", Name: "local.echo"}}}, nil
}
func (m fakeModel) Stream(ctx context.Context, req types.ModelRequest, emit func(types.ModelEvent) error) error {
	r, err := m.Generate(ctx, req)
	if err != nil {
		return err
	}
	for _, call := range r.ToolCalls {
		if err := emit(types.ModelEvent{Type: types.ModelEventTypeToolCall, ToolCall: &call}); err != nil {
			return err
		}
	}
	if r.FinalAnswer != "" {
		return emit(types.ModelEvent{Type: types.ModelEventTypeFinalAnswer, TextDelta: r.FinalAnswer})
	}
	return nil
}

type fakeTool struct{ pending bool }

func (fakeTool) Name() string               { return "echo" }
func (fakeTool) Description() string        { return "Returns bounded fake facts" }
func (fakeTool) JSONSchema() map[string]any { return map[string]any{"type": "object"} }
func (tool fakeTool) Invoke(context.Context, map[string]any) (types.ToolResult, error) {
	if tool.pending {
		return types.ToolResult{PendingAction: &types.DynamicActionReference{Token: "fake-token", Kind: "prepare", RunID: "fake-run", SessionID: "fake-session", CallID: "fake-call", Iteration: 1, Source: "example", Digest: "fake-digest", Resumable: true}}, nil
	}
	return types.ToolResult{Content: "fake tool result"}, nil
}

func execute(stream, pending bool) (types.RunResult, diagnosticsreplay.NativeExecutionTranscript, error) {
	model, err := modelcapability.Wrap(fakeModel{}, modelcapability.Config{Provider: "local", Capabilities: []types.ModelCapability{types.ModelCapabilityStreaming, types.ModelCapabilityToolCall}})
	if err != nil {
		return types.RunResult{}, diagnosticsreplay.NativeExecutionTranscript{}, err
	}
	registry := local.NewRegistry()
	if _, err := registry.Register(fakeTool{pending: pending}); err != nil {
		return types.RunResult{}, diagnosticsreplay.NativeExecutionTranscript{}, err
	}
	var events []types.NativeExecutionEvent
	observer, err := eventadapter.New(eventadapter.Options{Projection: types.NativeExecutionOptions{Source: types.ProtocolSourceRunner, ExpectedRunID: "fake-run", SessionID: "fake-session", HostReferences: types.NativeExecutionHostReferences{TaskID: "fake-task"}}, Sink: eventadapter.SinkFunc(func(_ context.Context, event types.NativeExecutionEvent) error {
		events = append(events, event)
		return nil
	})})
	if err != nil {
		return types.RunResult{}, diagnosticsreplay.NativeExecutionTranscript{}, err
	}
	defer observer.Close()
	engine := runner.New(model, runner.WithLocalRegistry(registry))
	req := types.RunRequest{RunID: "fake-run", SessionID: "fake-session", Input: "fake request", Capabilities: types.CapabilityRequirements{Required: []types.ModelCapability{types.ModelCapabilityStreaming}}}
	var result types.RunResult
	if stream {
		result, err = engine.Stream(context.Background(), req, observer)
	} else {
		result, err = engine.Run(context.Background(), req, observer)
	}
	s := observer.Status()
	transcript := diagnosticsreplay.NativeExecutionTranscript{Events: events, Observation: diagnosticsreplay.NativeExecutionDeliveryEvidence{State: s.State, Reason: s.Reason, Received: s.Received, Delivered: s.Delivered, Unconfirmed: s.Unconfirmed, Skipped: s.Skipped, TerminalDelivered: s.TerminalDelivered, Incomplete: s.Incomplete}}
	return result, transcript, err
}

func main() {
	run, runEvents, err := execute(false, false)
	if err != nil {
		panic(err)
	}
	stream, streamEvents, err := execute(true, false)
	if err != nil {
		panic(err)
	}
	if err := diagnosticsreplay.CompareNativeExecutionParity(runEvents, streamEvents); err != nil {
		panic(err)
	}
	if runEvents.Observation.Incomplete || streamEvents.Observation.Incomplete {
		panic("incomplete execution observation")
	}
	paused, pauseEvents, err := execute(false, true)
	if err != nil {
		panic(err)
	}
	if paused.TerminalOutcome.State != types.RunStateInputRequired || len(pauseEvents.Events) == 0 {
		panic("missing source pause")
	}
	fmt.Printf("native-events run=%s\nnative-events stream=%s\nobserver=%s\npause-is-terminal=%t\n", run.TerminalOutcome.State, stream.TerminalOutcome.State, streamEvents.Observation.State, pauseEvents.Events[len(pauseEvents.Events)-1].IsTerminal())
}
