package integration

import (
	"context"
	"errors"
	"testing"

	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/integration/fakes"
	"github.com/FelixSeptem/baymax/tool/local"
)

type eventCollector struct {
	events []types.Event
}

func (c *eventCollector) OnEvent(ctx context.Context, ev types.Event) {
	c.events = append(c.events, ev)
}

func TestE2EMultiTurnToolCall(t *testing.T) {
	model := fakes.NewModel([]fakes.ModelStep{
		{Response: types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "c1", Name: "local.echo", Args: map[string]any{"q": "hello"}}}}},
		{Response: types.ModelResponse{FinalAnswer: "done"}},
	})
	reg := local.NewRegistry()
	_, err := reg.Register(&fakes.Tool{
		NameValue:   "echo",
		SchemaValue: map[string]any{"type": "object", "required": []any{"q"}, "properties": map[string]any{"q": map[string]any{"type": "string"}}},
		InvokeFn: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
			return types.ToolResult{Content: args["q"].(string)}, nil
		},
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	eng := runner.New(model, runner.WithLocalRegistry(reg))
	res, err := eng.Run(context.Background(), types.RunRequest{Input: "run"}, nil)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if res.FinalAnswer != "done" {
		t.Fatalf("final answer = %q, want done", res.FinalAnswer)
	}
	if len(res.ToolCalls) != 1 || res.ToolCalls[0].Name != "local.echo" {
		t.Fatalf("tool summary mismatch: %#v", res.ToolCalls)
	}
	if model.Calls() != 2 {
		t.Fatalf("model calls = %d, want 2", model.Calls())
	}
}

func TestE2EMixedLocalAndMCPDispatch(t *testing.T) {
	fakeMCP := &fakes.MCP{}
	reg := local.NewRegistry()
	_, err := reg.Register(&fakes.Tool{
		NameValue:   "mcp_proxy",
		SchemaValue: map[string]any{"type": "object", "required": []any{"tool"}, "properties": map[string]any{"tool": map[string]any{"type": "string"}}},
		InvokeFn: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
			return fakeMCP.CallTool(ctx, args["tool"].(string), args)
		},
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	model := fakes.NewModel([]fakes.ModelStep{
		{Response: types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "c1", Name: "local.mcp_proxy", Args: map[string]any{"tool": "fake.mcp"}}}}},
		{Response: types.ModelResponse{FinalAnswer: "mixed-ok"}},
	})
	eng := runner.New(model, runner.WithLocalRegistry(reg))
	res, err := eng.Run(context.Background(), types.RunRequest{Input: "mixed"}, nil)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if res.FinalAnswer != "mixed-ok" {
		t.Fatalf("final answer = %q, want mixed-ok", res.FinalAnswer)
	}
}

func TestStreamingNoEventLossAndOrdering(t *testing.T) {
	model := fakes.NewModel(nil)
	model.SetStream([]types.ModelEvent{
		{Type: "final_answer", TextDelta: "he"},
		{Type: "final_answer", TextDelta: "llo"},
	}, nil)
	collector := &eventCollector{}
	eng := runner.New(model)
	res, err := eng.Stream(context.Background(), types.RunRequest{Input: "hello"}, collector)
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}
	if res.FinalAnswer != "hello" {
		t.Fatalf("final answer = %q, want hello", res.FinalAnswer)
	}
	if len(collector.events) < 6 {
		t.Fatalf("event count=%d, want>=6", len(collector.events))
	}
	if collector.events[0].Type != "run.started" || collector.events[1].Type != "model.requested" {
		t.Fatalf("unexpected prefix order: %#v", collector.events[:2])
	}
	if collector.events[len(collector.events)-1].Type != "run.finished" {
		t.Fatalf("last event=%s, want run.finished", collector.events[len(collector.events)-1].Type)
	}
}

func TestStreamAndRunSemanticConsistency(t *testing.T) {
	model := fakes.NewModel([]fakes.ModelStep{
		{Response: types.ModelResponse{FinalAnswer: "hello"}},
	})
	model.SetStream([]types.ModelEvent{
		{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "he"},
		{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "llo"},
	}, nil)
	eng := runner.New(model)

	runRes, err := eng.Run(context.Background(), types.RunRequest{Input: "hello"}, nil)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	streamRes, err := eng.Stream(context.Background(), types.RunRequest{Input: "hello"}, nil)
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}
	if runRes.FinalAnswer != streamRes.FinalAnswer {
		t.Fatalf("run/stream mismatch: run=%q stream=%q", runRes.FinalAnswer, streamRes.FinalAnswer)
	}
}

func TestStreamFailFastClassification(t *testing.T) {
	model := fakes.NewModel(nil)
	model.SetStream([]types.ModelEvent{
		{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "partial"},
	}, errors.New("stream failed"))

	eng := runner.New(model)
	res, err := eng.Stream(context.Background(), types.RunRequest{Input: "x"}, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if res.Error == nil || res.Error.Class != types.ErrModel {
		t.Fatalf("error class = %#v, want ErrModel", res.Error)
	}
}
