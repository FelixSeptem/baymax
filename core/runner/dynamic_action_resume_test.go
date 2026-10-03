package runner

import (
	"context"
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/tool/local"
)

func TestDynamicActionPausesBeforeNextModelStepAndResumesSameRun(t *testing.T) {
	reg := local.NewRegistry()
	toolCalls := 0
	_, err := reg.Register(&fakeTool{name: "prepare", invoke: func(context.Context, map[string]any) (types.ToolResult, error) {
		toolCalls++
		return types.ToolResult{PendingAction: &types.DynamicActionReference{
			Token: "opaque-1", Kind: "application.prepare", Resumable: true,
			RunID: "run-dynamic-1", SessionID: "session-1", Iteration: 1, CallID: "call-1", Source: "application", Digest: "digest-1",
		}}, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	modelCalls := 0
	model := &fakeModel{generate: func(_ context.Context, req types.ModelRequest) (types.ModelResponse, error) {
		modelCalls++
		if modelCalls == 1 {
			return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "call-1", Name: "local.prepare"}}}, nil
		}
		if len(req.ToolResult) != 1 || req.ToolResult[0].Result.PendingAction == nil {
			t.Fatalf("resume did not receive checkpointed tool result: %#v", req.ToolResult)
		}
		return types.ModelResponse{FinalAnswer: "done"}, nil
	}}
	engine := New(model, WithLocalRegistry(reg))
	first, err := engine.Run(context.Background(), types.RunRequest{RunID: "run-dynamic-1", SessionID: "session-1", Input: "prepare"}, nil)
	if err != nil {
		t.Fatalf("initial run error: %v", err)
	}
	if first.TerminalOutcome == nil || first.TerminalOutcome.State != types.RunStateInputRequired {
		t.Fatalf("initial outcome = %#v, want input_required", first.TerminalOutcome)
	}
	if modelCalls != 1 || toolCalls != 1 {
		t.Fatalf("calls after pause model=%d tool=%d", modelCalls, toolCalls)
	}
	// The pause projection is additive and contains only bounded references.
	if first.TerminalOutcome.PendingAction == nil {
		t.Fatalf("pending action projection missing: %#v", first.TerminalOutcome.PendingAction)
	}
	if _, ok := engine.ActiveRun("run-dynamic-1"); !ok {
		t.Fatal("paused run control was not retained")
	}
	checkpoint := first.TerminalOutcome
	resumed, err := engine.ResumeDynamicAction(context.Background(), types.DynamicActionDecision{
		Decision: types.DynamicActionDecisionConfirm, Token: "opaque-1", RunID: "run-dynamic-1", SessionID: "session-1",
		CheckpointID: checkpoint.CheckpointID, CheckpointVersion: checkpoint.CheckpointVersion, CheckpointDigest: checkpoint.CheckpointDigest, IdempotencyKey: "idem-1",
	}, nil, false)
	if err != nil {
		t.Fatalf("resume error: %v", err)
	}
	if resumed.RunID != "run-dynamic-1" || resumed.FinalAnswer != "done" {
		t.Fatalf("resumed result = %#v", resumed)
	}
	if modelCalls != 2 || toolCalls != 1 {
		t.Fatalf("calls after resume model=%d tool=%d", modelCalls, toolCalls)
	}
	duplicate, err := engine.ResumeDynamicAction(context.Background(), types.DynamicActionDecision{
		Decision: types.DynamicActionDecisionConfirm, Token: "opaque-1", RunID: "run-dynamic-1", SessionID: "session-1",
		CheckpointID: checkpoint.CheckpointID, CheckpointVersion: checkpoint.CheckpointVersion, CheckpointDigest: checkpoint.CheckpointDigest, IdempotencyKey: "idem-1",
	}, nil, false)
	if err != nil || duplicate.FinalAnswer != "done" {
		t.Fatalf("duplicate resume = %#v, err=%v", duplicate, err)
	}
}

func TestDynamicActionResumeRejectsStaleCheckpoint(t *testing.T) {
	reg := local.NewRegistry()
	_, err := reg.Register(&fakeTool{name: "prepare", invoke: func(context.Context, map[string]any) (types.ToolResult, error) {
		return types.ToolResult{PendingAction: &types.DynamicActionReference{Token: "opaque-stale", Kind: "prepare", Resumable: true, RunID: "run-stale", SessionID: "session-stale", Iteration: 1, CallID: "call-stale", Source: "test", Digest: "digest-stale"}}, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	model := &fakeModel{generate: func(_ context.Context, _ types.ModelRequest) (types.ModelResponse, error) {
		return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "call-stale", Name: "local.prepare"}}}, nil
	}}
	engine := New(model, WithLocalRegistry(reg))
	paused, err := engine.Run(context.Background(), types.RunRequest{RunID: "run-stale", SessionID: "session-stale", Input: "x"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	outcome := paused.TerminalOutcome
	_, err = engine.ResumeDynamicAction(context.Background(), types.DynamicActionDecision{Decision: types.DynamicActionDecisionConfirm, Token: "wrong", RunID: "run-stale", SessionID: "session-stale", CheckpointID: outcome.CheckpointID, CheckpointVersion: outcome.CheckpointVersion, CheckpointDigest: outcome.CheckpointDigest, IdempotencyKey: "stale-1"}, nil, false)
	if err == nil || err != ErrDynamicActionStale {
		t.Fatalf("stale resume err=%v, want %v", err, ErrDynamicActionStale)
	}
}

func TestDynamicActionRunAndStreamPreserveInputRequiredParity(t *testing.T) {
	makeEngine := func(stream bool) (*Engine, *int) {
		reg := local.NewRegistry()
		_, _ = reg.Register(&fakeTool{name: "prepare", invoke: func(context.Context, map[string]any) (types.ToolResult, error) {
			return types.ToolResult{PendingAction: &types.DynamicActionReference{Token: "opaque-parity", Kind: "prepare", Resumable: true, RunID: "run-parity", SessionID: "session-parity", Iteration: 1, CallID: "call-parity", Source: "test", Digest: "digest-parity"}}, nil
		}})
		calls := 0
		model := &fakeModel{generate: func(_ context.Context, _ types.ModelRequest) (types.ModelResponse, error) {
			calls++
			return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "call-parity", Name: "local.prepare"}}}, nil
		}, stream: func(_ context.Context, _ types.ModelRequest, onEvent func(types.ModelEvent) error) error {
			calls++
			return onEvent(types.ModelEvent{Type: types.ModelEventTypeToolCall, ToolCall: &types.ToolCall{CallID: "call-parity", Name: "local.prepare"}})
		}}
		_ = stream
		return New(model, WithLocalRegistry(reg)), &calls
	}
	runEngine, runCalls := makeEngine(false)
	streamEngine, streamCalls := makeEngine(true)
	run, runErr := runEngine.Run(context.Background(), types.RunRequest{RunID: "run-parity", SessionID: "session-parity", Input: "x"}, nil)
	stream, streamErr := streamEngine.Stream(context.Background(), types.RunRequest{RunID: "run-parity", SessionID: "session-parity", Input: "x"}, nil)
	if runErr != nil || streamErr != nil || run.TerminalOutcome == nil || stream.TerminalOutcome == nil {
		t.Fatalf("run=%#v err=%v stream=%#v err=%v", run, runErr, stream, streamErr)
	}
	if run.TerminalOutcome.State != types.RunStateInputRequired || stream.TerminalOutcome.State != types.RunStateInputRequired {
		t.Fatalf("run/stream states: %v/%v", run.TerminalOutcome.State, stream.TerminalOutcome.State)
	}
	if *runCalls != 1 || *streamCalls != 1 {
		t.Fatalf("model calls run/stream=%d/%d", *runCalls, *streamCalls)
	}
}
