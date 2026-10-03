package main

import (
	"context"
	"fmt"

	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/tool/local"
)

type model struct{ calls int }

func (m *model) ProviderName() string { return "dynamic-action-example" }
func (m *model) DiscoverCapabilities(context.Context, types.ModelRequest) (types.ProviderCapabilities, error) {
	return types.ProviderCapabilities{Provider: m.ProviderName(), Support: map[types.ModelCapability]types.CapabilitySupport{types.ModelCapabilityToolCall: types.CapabilitySupportSupported, types.ModelCapabilityStreaming: types.CapabilitySupportSupported}}, nil
}
func (m *model) Generate(context.Context, types.ModelRequest) (types.ModelResponse, error) {
	m.calls++
	if m.calls == 1 {
		return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "prepare-1", Name: "local.prepare"}}}, nil
	}
	return types.ModelResponse{FinalAnswer: "confirmed"}, nil
}
func (m *model) Stream(_ context.Context, _ types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	m.calls++
	if m.calls == 1 {
		return onEvent(types.ModelEvent{Type: types.ModelEventTypeToolCall, ToolCall: &types.ToolCall{CallID: "prepare-1", Name: "local.prepare"}})
	}
	return onEvent(types.ModelEvent{Type: types.ModelEventTypeFinalAnswer, TextDelta: "confirmed"})
}

func newEngine(m *model) *runner.Engine {
	registry := local.NewRegistry()
	_, _ = registry.Register(&prepareTool{})
	return runner.New(m, runner.WithLocalRegistry(registry))
}

type prepareTool struct{}

func (*prepareTool) Name() string               { return "prepare" }
func (*prepareTool) Description() string        { return "register an opaque action" }
func (*prepareTool) JSONSchema() map[string]any { return map[string]any{"type": "object"} }
func (*prepareTool) Invoke(context.Context, map[string]any) (types.ToolResult, error) {
	return types.ToolResult{PendingAction: &types.DynamicActionReference{Token: "opaque-application-action", Kind: "application.prepare", Resumable: true, RunID: "example-run", SessionID: "example-session", Iteration: 1, CallID: "prepare-1", Source: "example-adapter", Digest: "example-digest"}}, nil
}

func resume(e *runner.Engine, paused types.RunResult) (types.RunResult, error) {
	o := paused.TerminalOutcome
	return e.ResumeDynamicAction(context.Background(), types.DynamicActionDecision{Decision: types.DynamicActionDecisionConfirm, Token: o.PendingAction.Token, RunID: o.RunID, SessionID: o.SessionID, CheckpointID: o.CheckpointID, CheckpointVersion: o.CheckpointVersion, CheckpointDigest: o.CheckpointDigest, IdempotencyKey: "example-confirm-1"}, nil, false)
}

func main() {
	m := &model{}
	e := newEngine(m)
	paused, err := e.Run(context.Background(), types.RunRequest{RunID: "example-run", SessionID: "example-session", Input: "prepare"}, nil)
	if err != nil {
		panic(err)
	}
	resumed, err := resume(e, paused)
	if err != nil {
		panic(err)
	}
	fmt.Printf("run_id=%s paused_state=%s resumed_state=%s final=%s\n", paused.RunID, paused.TerminalOutcome.State, resumed.TerminalOutcome.State, resumed.FinalAnswer)
}
