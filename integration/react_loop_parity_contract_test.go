package integration

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/integration/fakes"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
	"github.com/FelixSeptem/baymax/tool/local"
)

type scriptedGenerateStep struct {
	response      types.ModelResponse
	err           error
	assertRequest func(req types.ModelRequest) error
}

type scriptedStreamStep struct {
	events        []types.ModelEvent
	err           error
	assertRequest func(req types.ModelRequest) error
}

type scriptedReactModel struct {
	mu            sync.Mutex
	generateSteps []scriptedGenerateStep
	streamSteps   []scriptedStreamStep
	generateCalls int
	streamCalls   int
}

func (m *scriptedReactModel) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
	_ = ctx
	m.mu.Lock()
	idx := m.generateCalls
	m.generateCalls++
	var step scriptedGenerateStep
	if idx < len(m.generateSteps) {
		step = m.generateSteps[idx]
	}
	m.mu.Unlock()
	if step.assertRequest != nil {
		if err := step.assertRequest(req); err != nil {
			return types.ModelResponse{}, err
		}
	}
	if idx >= len(m.generateSteps) {
		return types.ModelResponse{FinalAnswer: "done"}, nil
	}
	return step.response, step.err
}

func (m *scriptedReactModel) Stream(
	ctx context.Context,
	req types.ModelRequest,
	onEvent func(types.ModelEvent) error,
) error {
	m.mu.Lock()
	idx := m.streamCalls
	m.streamCalls++
	var step scriptedStreamStep
	if idx < len(m.streamSteps) {
		step = m.streamSteps[idx]
	}
	m.mu.Unlock()
	if step.assertRequest != nil {
		if err := step.assertRequest(req); err != nil {
			return err
		}
	}
	if idx >= len(m.streamSteps) {
		return onEvent(types.ModelEvent{
			Type:      types.ModelEventTypeOutputTextDelta,
			TextDelta: "done",
		})
	}
	for _, ev := range step.events {
		if err := onEvent(ev); err != nil {
			return err
		}
	}
	if step.err != nil {
		return step.err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func (m *scriptedReactModel) ProviderName() string {
	return "react-scripted"
}

func (m *scriptedReactModel) DiscoverCapabilities(
	ctx context.Context,
	req types.ModelRequest,
) (types.ProviderCapabilities, error) {
	_ = ctx
	return types.ProviderCapabilities{
		Provider: "react-scripted",
		Model:    req.Model,
		Support: map[types.ModelCapability]types.CapabilitySupport{
			types.ModelCapabilityStreaming: types.CapabilitySupportSupported,
			types.ModelCapabilityToolCall:  types.CapabilitySupportSupported,
		},
		Source: "integration-test",
	}, nil
}

func TestReactLoopRunStreamParityIntegrationContract(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		runModel := &scriptedReactModel{
			generateSteps: []scriptedGenerateStep{
				{
					response: types.ModelResponse{
						ToolCalls: []types.ToolCall{
							{
								CallID: "call-success-1",
								Name:   "local.echo",
								Args:   map[string]any{"q": "hello"},
							},
						},
					},
				},
				{
					assertRequest: func(req types.ModelRequest) error {
						if len(req.ToolResult) != 1 || req.ToolResult[0].CallID != "call-success-1" {
							return fmt.Errorf("run lane missing canonical tool feedback: %#v", req.ToolResult)
						}
						return nil
					},
					response: types.ModelResponse{FinalAnswer: "ok"},
				},
			},
		}
		streamModel := &scriptedReactModel{
			streamSteps: []scriptedStreamStep{
				{
					events: []types.ModelEvent{
						{
							Type: types.ModelEventTypeToolCall,
							ToolCall: &types.ToolCall{
								CallID: "call-success-1",
								Name:   "local.echo",
								Args:   map[string]any{"q": "hello"},
							},
						},
						{Type: types.ModelEventTypeResponseCompleted},
					},
				},
				{
					assertRequest: func(req types.ModelRequest) error {
						if len(req.ToolResult) != 1 || req.ToolResult[0].CallID != "call-success-1" {
							return fmt.Errorf("stream lane missing canonical tool feedback: %#v", req.ToolResult)
						}
						return nil
					},
					events: []types.ModelEvent{
						{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "ok"},
						{Type: types.ModelEventTypeResponseCompleted},
					},
				},
			},
		}
		runEngine := newReactParityEngine(t, runModel, "echo", func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
			return types.ToolResult{Content: "ok"}, nil
		})
		streamEngine := newReactParityEngine(t, streamModel, "echo", func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
			return types.ToolResult{Content: "ok"}, nil
		})

		req := types.RunRequest{RunID: "run-a56-react-parity-success", Input: "hello"}
		policy := types.DefaultLoopPolicy()
		policy.MaxIterations = 4
		policy.ToolCallLimit = 4
		req.Policy = &policy
		runCollector := &eventCollector{}
		streamCollector := &eventCollector{}
		runRes, runErr := runEngine.Run(context.Background(), req, runCollector)
		streamRes, streamErr := streamEngine.Stream(context.Background(), req, streamCollector)
		if runErr != nil || streamErr != nil {
			t.Fatalf("run/stream should both succeed, runErr=%v streamErr=%v", runErr, streamErr)
		}
		if runRes.Error != nil || streamRes.Error != nil {
			t.Fatalf("unexpected classified errors, run=%#v stream=%#v", runRes.Error, streamRes.Error)
		}
		if runRes.FinalAnswer != "ok" || streamRes.FinalAnswer != "ok" {
			t.Fatalf("final answer mismatch run=%q stream=%q", runRes.FinalAnswer, streamRes.FinalAnswer)
		}
		runPayload := lastRunFinishedPayload(t, runCollector.events)
		streamPayload := lastRunFinishedPayload(t, streamCollector.events)
		assertReactParityPayload(t, runPayload, streamPayload, runtimeconfig.RuntimeReactTerminationCompleted)
		if runPayload["react_tool_call_total"] != 1 || streamPayload["react_tool_call_total"] != 1 {
			t.Fatalf("react_tool_call_total mismatch run=%#v stream=%#v", runPayload["react_tool_call_total"], streamPayload["react_tool_call_total"])
		}
	})

	t.Run("budget_exhausted", func(t *testing.T) {
		runModel := &scriptedReactModel{
			generateSteps: []scriptedGenerateStep{
				{
					response: types.ModelResponse{
						ToolCalls: []types.ToolCall{{CallID: "call-budget-1", Name: "local.echo", Args: map[string]any{"q": "1"}}},
					},
				},
				{
					assertRequest: func(req types.ModelRequest) error {
						if len(req.ToolResult) != 1 || req.ToolResult[0].CallID != "call-budget-1" {
							return fmt.Errorf("run lane missing tool feedback before budget check: %#v", req.ToolResult)
						}
						return nil
					},
					response: types.ModelResponse{
						ToolCalls: []types.ToolCall{{CallID: "call-budget-2", Name: "local.echo", Args: map[string]any{"q": "2"}}},
					},
				},
			},
		}
		streamModel := &scriptedReactModel{
			streamSteps: []scriptedStreamStep{
				{
					events: []types.ModelEvent{
						{
							Type: types.ModelEventTypeToolCall,
							ToolCall: &types.ToolCall{
								CallID: "call-budget-1",
								Name:   "local.echo",
								Args:   map[string]any{"q": "1"},
							},
						},
						{Type: types.ModelEventTypeResponseCompleted},
					},
				},
				{
					assertRequest: func(req types.ModelRequest) error {
						if len(req.ToolResult) != 1 || req.ToolResult[0].CallID != "call-budget-1" {
							return fmt.Errorf("stream lane missing tool feedback before budget check: %#v", req.ToolResult)
						}
						return nil
					},
					events: []types.ModelEvent{
						{
							Type: types.ModelEventTypeToolCall,
							ToolCall: &types.ToolCall{
								CallID: "call-budget-2",
								Name:   "local.echo",
								Args:   map[string]any{"q": "2"},
							},
						},
						{Type: types.ModelEventTypeResponseCompleted},
					},
				},
			},
		}
		runEngine := newReactParityEngine(t, runModel, "echo", func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
			return types.ToolResult{Content: "ok"}, nil
		})
		streamEngine := newReactParityEngine(t, streamModel, "echo", func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
			return types.ToolResult{Content: "ok"}, nil
		})

		policy := types.DefaultLoopPolicy()
		policy.MaxIterations = 6
		policy.ToolCallLimit = 1
		req := types.RunRequest{
			RunID:  "run-a56-react-parity-budget",
			Input:  "budget",
			Policy: &policy,
		}
		runCollector := &eventCollector{}
		streamCollector := &eventCollector{}
		runRes, runErr := runEngine.Run(context.Background(), req, runCollector)
		streamRes, streamErr := streamEngine.Stream(context.Background(), req, streamCollector)
		if runErr == nil || streamErr == nil {
			t.Fatalf("run/stream should both fail on budget, runErr=%v streamErr=%v", runErr, streamErr)
		}
		if runRes.Error == nil || streamRes.Error == nil {
			t.Fatalf("missing classified errors run=%#v stream=%#v", runRes.Error, streamRes.Error)
		}
		if runRes.Error.Class != types.ErrIterationLimit || streamRes.Error.Class != types.ErrIterationLimit {
			t.Fatalf("error class mismatch run=%#v stream=%#v", runRes.Error, streamRes.Error)
		}
		runPayload := lastRunFinishedPayload(t, runCollector.events)
		streamPayload := lastRunFinishedPayload(t, streamCollector.events)
		assertReactParityPayload(t, runPayload, streamPayload, runtimeconfig.RuntimeReactTerminationToolCallLimitExceeded)
		if runPayload["react_tool_call_budget_hit_total"] != 1 || streamPayload["react_tool_call_budget_hit_total"] != 1 {
			t.Fatalf("react_tool_call_budget_hit_total mismatch run=%#v stream=%#v", runPayload["react_tool_call_budget_hit_total"], streamPayload["react_tool_call_budget_hit_total"])
		}
	})

	t.Run("tool_dispatch_failure", func(t *testing.T) {
		runModel := &scriptedReactModel{
			generateSteps: []scriptedGenerateStep{
				{
					response: types.ModelResponse{
						ToolCalls: []types.ToolCall{{CallID: "call-fail-1", Name: "local.explode"}},
					},
				},
			},
		}
		streamModel := &scriptedReactModel{
			streamSteps: []scriptedStreamStep{
				{
					events: []types.ModelEvent{
						{
							Type: types.ModelEventTypeToolCall,
							ToolCall: &types.ToolCall{
								CallID: "call-fail-1",
								Name:   "local.explode",
							},
						},
					},
				},
			},
		}
		runEngine := newReactParityEngine(t, runModel, "explode", func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
			return types.ToolResult{}, errors.New("dispatch failed")
		})
		streamEngine := newReactParityEngine(t, streamModel, "explode", func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
			return types.ToolResult{}, errors.New("dispatch failed")
		})

		req := types.RunRequest{RunID: "run-a56-react-parity-tool-fail", Input: "tool-fail"}
		runCollector := &eventCollector{}
		streamCollector := &eventCollector{}
		runRes, runErr := runEngine.Run(context.Background(), req, runCollector)
		streamRes, streamErr := streamEngine.Stream(context.Background(), req, streamCollector)
		if runErr == nil || streamErr == nil {
			t.Fatalf("run/stream should both fail on tool dispatch, runErr=%v streamErr=%v", runErr, streamErr)
		}
		if runRes.Error == nil || streamRes.Error == nil {
			t.Fatalf("missing classified errors run=%#v stream=%#v", runRes.Error, streamRes.Error)
		}
		if runRes.Error.Class != types.ErrTool || streamRes.Error.Class != types.ErrTool {
			t.Fatalf("error class mismatch run=%#v stream=%#v", runRes.Error, streamRes.Error)
		}
		runPayload := lastRunFinishedPayload(t, runCollector.events)
		streamPayload := lastRunFinishedPayload(t, streamCollector.events)
		assertReactParityPayload(t, runPayload, streamPayload, runtimeconfig.RuntimeReactTerminationToolDispatchFailed)
	})

	t.Run("provider_error", func(t *testing.T) {
		runModel := &scriptedReactModel{
			generateSteps: []scriptedGenerateStep{
				{
					err: errors.New("provider unavailable"),
				},
			},
		}
		streamModel := &scriptedReactModel{
			streamSteps: []scriptedStreamStep{
				{
					err: errors.New("provider unavailable"),
				},
			},
		}
		runEngine := newReactParityEngine(t, runModel, "", nil)
		streamEngine := newReactParityEngine(t, streamModel, "", nil)

		req := types.RunRequest{RunID: "run-a56-react-parity-provider-fail", Input: "provider-fail"}
		runCollector := &eventCollector{}
		streamCollector := &eventCollector{}
		runRes, runErr := runEngine.Run(context.Background(), req, runCollector)
		streamRes, streamErr := streamEngine.Stream(context.Background(), req, streamCollector)
		if runErr == nil || streamErr == nil {
			t.Fatalf("run/stream should both fail on provider error, runErr=%v streamErr=%v", runErr, streamErr)
		}
		if runRes.Error == nil || streamRes.Error == nil {
			t.Fatalf("missing classified errors run=%#v stream=%#v", runRes.Error, streamRes.Error)
		}
		if runRes.Error.Class != types.ErrModel || streamRes.Error.Class != types.ErrModel {
			t.Fatalf("error class mismatch run=%#v stream=%#v", runRes.Error, streamRes.Error)
		}
		runPayload := lastRunFinishedPayload(t, runCollector.events)
		streamPayload := lastRunFinishedPayload(t, streamCollector.events)
		assertReactParityPayload(t, runPayload, streamPayload, runtimeconfig.RuntimeReactTerminationProviderError)
	})
}

func newReactParityEngine(
	t *testing.T,
	model types.ModelClient,
	toolName string,
	invoke func(ctx context.Context, args map[string]any) (types.ToolResult, error),
) *runner.Engine {
	t.Helper()
	opts := make([]runner.Option, 0, 1)
	if toolName != "" {
		reg := local.NewRegistry()
		_, err := reg.Register(&fakes.Tool{
			NameValue:   toolName,
			SchemaValue: map[string]any{"type": "object"},
			InvokeFn:    invoke,
		})
		if err != nil {
			t.Fatalf("register tool %q: %v", toolName, err)
		}
		opts = append(opts, runner.WithLocalRegistry(reg))
	}
	return runner.New(model, opts...)
}

func lastRunFinishedPayload(t *testing.T, events []types.Event) map[string]any {
	t.Helper()
	for i := len(events) - 1; i >= 0; i-- {
		ev := events[i]
		if ev.Type != "run.finished" {
			continue
		}
		return ev.Payload
	}
	t.Fatalf("missing run.finished event in %#v", nonTimelineEvents(events))
	return nil
}

func assertReactParityPayload(t *testing.T, runPayload map[string]any, streamPayload map[string]any, wantReason string) {
	t.Helper()
	if runPayload["react_termination_reason"] != wantReason {
		t.Fatalf("run react_termination_reason=%#v, want %q", runPayload["react_termination_reason"], wantReason)
	}
	if streamPayload["react_termination_reason"] != wantReason {
		t.Fatalf("stream react_termination_reason=%#v, want %q", streamPayload["react_termination_reason"], wantReason)
	}
	keys := []string{
		"react_iteration_total",
		"react_tool_call_total",
		"react_tool_call_budget_hit_total",
		"react_iteration_budget_hit_total",
	}
	for _, key := range keys {
		if runPayload[key] != streamPayload[key] {
			t.Fatalf("run/stream parity mismatch key=%s run=%#v stream=%#v", key, runPayload[key], streamPayload[key])
		}
	}
}
