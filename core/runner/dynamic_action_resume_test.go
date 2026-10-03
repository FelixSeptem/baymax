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
