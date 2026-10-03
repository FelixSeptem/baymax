package runner

import (
	"context"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/tool/local"
)

type dynamicActionEventCollector struct {
	events []types.Event
}

func (c *dynamicActionEventCollector) OnEvent(_ context.Context, ev types.Event) {
	c.events = append(c.events, ev)
}

func (c *dynamicActionEventCollector) count(eventType string) int {
	count := 0
	for _, ev := range c.events {
		if ev.Type == eventType {
			count++
		}
	}
	return count
}

func (c *dynamicActionEventCollector) canceledFinishedCount() int {
	count := 0
	for _, ev := range c.events {
		if ev.Type == "run.finished" && ev.Payload != nil && ev.Payload["state"] == string(types.RunStateCanceled) {
			count++
		}
	}
	return count
}

func (c *dynamicActionEventCollector) lastResolvedDecision() any {
	payload := c.lastResolvedPayload()
	if payload == nil {
		return nil
	}
	return payload["decision"]
}

func (c *dynamicActionEventCollector) lastResolvedPayload() map[string]any {
	for i := len(c.events) - 1; i >= 0; i-- {
		if c.events[i].Type == "run.dynamic_action.resolved" && c.events[i].Payload != nil {
			return c.events[i].Payload
		}
	}
	return nil
}

func newDynamicActionResumeEventScenario(t *testing.T, stream bool) (*Engine, *types.TerminalOutcome, *dynamicActionEventCollector) {
	t.Helper()
	reg := local.NewRegistry()
	_, err := reg.Register(&fakeTool{name: "prepare", invoke: func(context.Context, map[string]any) (types.ToolResult, error) {
		return types.ToolResult{PendingAction: &types.DynamicActionReference{
			Token: "opaque-events", Kind: "prepare", Resumable: true,
			RunID: "run-events", SessionID: "session-events", Iteration: 1,
			CallID: "call-events", Source: "test", Digest: "digest-events",
		}}, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	model := &fakeModel{
		generate: func(_ context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			if len(req.ToolResult) > 0 {
				return types.ModelResponse{FinalAnswer: "confirmed"}, nil
			}
			return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "call-events", Name: "local.prepare"}}}, nil
		},
		stream: func(_ context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
			if len(req.ToolResult) > 0 {
				return onEvent(types.ModelEvent{Type: types.ModelEventTypeFinalAnswer, TextDelta: "confirmed"})
			}
			return onEvent(types.ModelEvent{Type: types.ModelEventTypeToolCall, ToolCall: &types.ToolCall{CallID: "call-events", Name: "local.prepare"}})
		},
	}
	engine := New(model, WithLocalRegistry(reg))
	collector := &dynamicActionEventCollector{}
	var paused types.RunResult
	if stream {
		paused, err = engine.Stream(context.Background(), types.RunRequest{RunID: "run-events", SessionID: "session-events", Input: "x"}, collector)
	} else {
		paused, err = engine.Run(context.Background(), types.RunRequest{RunID: "run-events", SessionID: "session-events", Input: "x"}, collector)
	}
	if err != nil || paused.TerminalOutcome == nil || paused.TerminalOutcome.State != types.RunStateInputRequired {
		t.Fatalf("pause=%#v err=%v", paused, err)
	}
	return engine, paused.TerminalOutcome, collector
}

func resumeDynamicActionForEventTest(t *testing.T, engine *Engine, checkpoint *types.TerminalOutcome, collector *dynamicActionEventCollector, decision types.DynamicActionDecisionKind, stream bool) types.RunResult {
	t.Helper()
	result, err := engine.ResumeDynamicAction(context.Background(), types.DynamicActionDecision{
		Decision: decision, Token: "opaque-events", RunID: "run-events", SessionID: "session-events",
		CheckpointID: checkpoint.CheckpointID, CheckpointVersion: checkpoint.CheckpointVersion,
		CheckpointDigest: checkpoint.CheckpointDigest, IdempotencyKey: "idem-" + string(decision),
	}, collector, stream)
	if err != nil {
		t.Fatalf("resume decision=%s err=%v", decision, err)
	}
	return result
}

func TestDynamicActionResolutionEventsSeparateDecisionAndCancellation(t *testing.T) {
	t.Run("confirm run", func(t *testing.T) {
		engine, checkpoint, collector := newDynamicActionResumeEventScenario(t, false)
		result := resumeDynamicActionForEventTest(t, engine, checkpoint, collector, types.DynamicActionDecisionConfirm, false)
		if result.TerminalOutcome == nil || result.TerminalOutcome.State == types.RunStateCanceled {
			t.Fatalf("confirm result=%#v, want non-canceled terminal outcome", result.TerminalOutcome)
		}
		if collector.count("run.dynamic_action.resolved") != 1 || collector.lastResolvedDecision() != string(types.DynamicActionDecisionConfirm) {
			t.Fatalf("resolution events=%d decision=%v, want one confirm", collector.count("run.dynamic_action.resolved"), collector.lastResolvedDecision())
		}
		assertDynamicActionResolvedPayload(t, collector.lastResolvedPayload(), checkpoint, types.DynamicActionDecisionConfirm)
		if collector.canceledFinishedCount() != 0 {
			t.Fatalf("canceled terminal events=%d, want zero", collector.canceledFinishedCount())
		}
		duplicate := resumeDynamicActionForEventTest(t, engine, checkpoint, collector, types.DynamicActionDecisionConfirm, false)
		if duplicate.FinalAnswer != result.FinalAnswer || collector.count("run.dynamic_action.resolved") != 1 {
			t.Fatalf("duplicate=%#v resolution events=%d, want prior result and one event", duplicate, collector.count("run.dynamic_action.resolved"))
		}
	})

	t.Run("confirm stream", func(t *testing.T) {
		engine, checkpoint, collector := newDynamicActionResumeEventScenario(t, true)
		result := resumeDynamicActionForEventTest(t, engine, checkpoint, collector, types.DynamicActionDecisionConfirm, true)
		if result.TerminalOutcome == nil || result.TerminalOutcome.State == types.RunStateCanceled {
			t.Fatalf("confirm stream result=%#v, want non-canceled terminal outcome", result.TerminalOutcome)
		}
		if collector.count("run.dynamic_action.resolved") != 1 || collector.lastResolvedDecision() != string(types.DynamicActionDecisionConfirm) || collector.canceledFinishedCount() != 0 {
			t.Fatalf("stream resolution=%d decision=%v canceled=%d, want one confirm and zero canceled", collector.count("run.dynamic_action.resolved"), collector.lastResolvedDecision(), collector.canceledFinishedCount())
		}
		assertDynamicActionResolvedPayload(t, collector.lastResolvedPayload(), checkpoint, types.DynamicActionDecisionConfirm)
	})

	for _, decision := range []types.DynamicActionDecisionKind{types.DynamicActionDecisionDeny, types.DynamicActionDecisionTimeout} {
		t.Run(string(decision), func(t *testing.T) {
			engine, checkpoint, collector := newDynamicActionResumeEventScenario(t, false)
			result := resumeDynamicActionForEventTest(t, engine, checkpoint, collector, decision, false)
			if result.TerminalOutcome == nil || result.TerminalOutcome.State != types.RunStateCanceled {
				t.Fatalf("result=%#v, want canceled", result.TerminalOutcome)
			}
			if collector.count("run.dynamic_action.resolved") != 1 || collector.lastResolvedDecision() != string(decision) || collector.canceledFinishedCount() != 1 {
				t.Fatalf("resolution=%d decision=%v canceled=%d, want one resolution and one canceled", collector.count("run.dynamic_action.resolved"), collector.lastResolvedDecision(), collector.canceledFinishedCount())
			}
			assertDynamicActionResolvedPayload(t, collector.lastResolvedPayload(), checkpoint, decision)
		})
	}
}

func assertDynamicActionResolvedPayload(t *testing.T, payload map[string]any, checkpoint *types.TerminalOutcome, decision types.DynamicActionDecisionKind) {
	t.Helper()
	if payload == nil {
		t.Fatal("missing dynamic action resolution payload")
	}
	want := map[string]any{
		"decision":                        string(decision),
		"checkpoint_id":                   checkpoint.CheckpointID,
		"checkpoint_version":              checkpoint.CheckpointVersion,
		"checkpoint_digest":               checkpoint.CheckpointDigest,
		"dynamic_action_resume_attempt":   1,
		"dynamic_action_resume_admission": "accepted",
	}
	for key, expected := range want {
		if got := payload[key]; got != expected {
			t.Fatalf("resolution payload[%q]=%#v, want %#v; payload=%#v", key, got, expected, payload)
		}
	}
}

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

func TestDynamicActionResumeFailClosedMatrix(t *testing.T) {
	reg := local.NewRegistry()
	toolCalls := 0
	_, err := reg.Register(&fakeTool{name: "prepare", invoke: func(context.Context, map[string]any) (types.ToolResult, error) {
		toolCalls++
		return types.ToolResult{PendingAction: &types.DynamicActionReference{Token: "opaque-matrix", Kind: "prepare", Resumable: true, RunID: "run-matrix", SessionID: "session-matrix", Iteration: 1, CallID: "call-matrix", Source: "test", Digest: "digest-matrix"}}, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	modelCalls := 0
	model := &fakeModel{generate: func(_ context.Context, req types.ModelRequest) (types.ModelResponse, error) {
		modelCalls++
		if len(req.ToolResult) > 0 {
			return types.ModelResponse{FinalAnswer: "done"}, nil
		}
		return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "call-matrix", Name: "local.prepare"}}}, nil
	}}
	engine := New(model, WithLocalRegistry(reg))
	paused, err := engine.Run(context.Background(), types.RunRequest{RunID: "run-matrix", SessionID: "session-matrix", Input: "x"}, nil)
	if err != nil || paused.TerminalOutcome == nil {
		t.Fatalf("pause=%#v err=%v", paused, err)
	}
	cp := paused.TerminalOutcome
	base := types.DynamicActionDecision{Decision: types.DynamicActionDecisionConfirm, Token: "opaque-matrix", RunID: "run-matrix", SessionID: "session-matrix", CheckpointID: cp.CheckpointID, CheckpointVersion: cp.CheckpointVersion, CheckpointDigest: cp.CheckpointDigest, IdempotencyKey: "idem-matrix"}
	cases := []struct {
		name   string
		mutate func(*types.DynamicActionDecision)
		want   error
	}{
		{name: "missing token", mutate: func(d *types.DynamicActionDecision) { d.Token = "" }, want: nil},
		{name: "mismatched session", mutate: func(d *types.DynamicActionDecision) { d.SessionID = "other-session" }, want: ErrDynamicActionStale},
		{name: "mismatched digest", mutate: func(d *types.DynamicActionDecision) { d.CheckpointDigest = "other-digest" }, want: ErrDynamicActionStale},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			decision := base
			tc.mutate(&decision)
			_, got := engine.ResumeDynamicAction(context.Background(), decision, nil, false)
			if tc.want == nil {
				if got == nil {
					t.Fatal("missing decision was accepted")
				}
			} else if got != tc.want {
				t.Fatalf("err=%v want=%v", got, tc.want)
			}
		})
	}
	if modelCalls != 1 || toolCalls != 1 {
		t.Fatalf("rejected resumes mutated execution model=%d tool=%d", modelCalls, toolCalls)
	}
	engine.dynamicActionMu.Lock()
	checkpoint := engine.dynamicCheckpoints["run-matrix"]
	checkpoint.checkpoint.CreatedAt = time.Now().UTC().Add(-dynamicActionCheckpointMaxAge - time.Minute)
	engine.dynamicCheckpoints["run-matrix"] = checkpoint
	engine.dynamicActionMu.Unlock()
	if _, err := engine.ResumeDynamicAction(context.Background(), base, nil, false); err != ErrDynamicActionExpired {
		t.Fatalf("expired resume err=%v want=%v", err, ErrDynamicActionExpired)
	}
	// Restore a fresh checkpoint for denial and terminal-state checks.
	engine.dynamicActionMu.Lock()
	checkpoint.checkpoint.CreatedAt = time.Now().UTC()
	engine.dynamicCheckpoints["run-matrix"] = checkpoint
	engine.dynamicActionMu.Unlock()
	denied := base
	denied.Decision = types.DynamicActionDecisionDeny
	denied.IdempotencyKey = "idem-deny"
	if _, err := engine.ResumeDynamicAction(context.Background(), denied, nil, false); err != nil {
		t.Fatalf("deny resume err=%v", err)
	}
	if modelCalls != 1 || toolCalls != 1 {
		t.Fatalf("deny resume invoked execution model=%d tool=%d", modelCalls, toolCalls)
	}
	conflict := denied
	conflict.Decision = types.DynamicActionDecisionConfirm
	conflict.IdempotencyKey = "idem-conflict"
	if _, err := engine.ResumeDynamicAction(context.Background(), conflict, nil, false); err != ErrDynamicActionTerminal {
		t.Fatalf("terminal resume err=%v want=%v", err, ErrDynamicActionTerminal)
	}
}

func TestDynamicActionRunAndStreamPreserveInputRequiredParity(t *testing.T) {
	makeEngine := func(stream bool) (*Engine, *int) {
		reg := local.NewRegistry()
		_, _ = reg.Register(&fakeTool{name: "prepare", invoke: func(context.Context, map[string]any) (types.ToolResult, error) {
			return types.ToolResult{PendingAction: &types.DynamicActionReference{Token: "opaque-parity", Kind: "prepare", Resumable: true, RunID: "run-parity", SessionID: "session-parity", Iteration: 1, CallID: "call-parity", Source: "test", Digest: "digest-parity"}}, nil
		}})
		calls := 0
		model := &fakeModel{generate: func(_ context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			calls++
			if len(req.ToolResult) > 0 {
				return types.ModelResponse{FinalAnswer: "confirmed"}, nil
			}
			return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "call-parity", Name: "local.prepare"}}}, nil
		}, stream: func(_ context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
			calls++
			if len(req.ToolResult) > 0 {
				return onEvent(types.ModelEvent{Type: types.ModelEventTypeFinalAnswer, TextDelta: "confirmed"})
			}
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
	streamResumed, err := streamEngine.ResumeDynamicAction(context.Background(), types.DynamicActionDecision{Decision: types.DynamicActionDecisionConfirm, Token: "opaque-parity", RunID: "run-parity", SessionID: "session-parity", CheckpointID: stream.TerminalOutcome.CheckpointID, CheckpointVersion: stream.TerminalOutcome.CheckpointVersion, CheckpointDigest: stream.TerminalOutcome.CheckpointDigest, IdempotencyKey: "parity-confirm"}, nil, true)
	if err != nil || streamResumed.FinalAnswer != "confirmed" {
		t.Fatalf("stream resume = %#v, err=%v", streamResumed, err)
	}
}
