package runner

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/FelixSeptem/baymax/core/checkpoint"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/tool/local"
)

func TestNativeContinuationLegacyDuplicateRetainsPredecessor(t *testing.T) {
	for _, stream := range []bool{false, true} {
		t.Run(map[bool]string{false: "run", true: "stream"}[stream], func(t *testing.T) {
			store, err := checkpoint.NewMemoryStore(checkpoint.Options{Namespace: "runner", Capacity: 8})
			if err != nil {
				t.Fatal(err)
			}
			modelCalls, toolCalls := 0, 0
			model := &fakeModel{generate: func(context.Context, types.ModelRequest) (types.ModelResponse, error) {
				modelCalls++
				return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: fmt.Sprint(modelCalls), Name: "local.pause"}}}, nil
			}, stream: func(_ context.Context, _ types.ModelRequest, emit func(types.ModelEvent) error) error {
				modelCalls++
				return emit(types.ModelEvent{Type: types.ModelEventTypeToolCall, ToolCall: &types.ToolCall{CallID: fmt.Sprint(modelCalls), Name: "local.pause"}})
			}}
			reg := local.NewRegistry()
			if _, err := reg.Register(&fakeTool{name: "pause", invoke: func(context.Context, map[string]any) (types.ToolResult, error) {
				toolCalls++
				return types.ToolResult{PendingAction: &types.DynamicActionReference{Token: fmt.Sprint(toolCalls), Kind: "input", Resumable: true, RunID: "legacy", SessionID: "session", Iteration: toolCalls, CallID: fmt.Sprint(toolCalls), Source: "offline", Digest: fmt.Sprint(toolCalls)}}, nil
			}}); err != nil {
				t.Fatal(err)
			}
			e := New(model, WithLocalRegistry(reg), WithNativeContinuation(NativeContinuationOptions{Store: store, OwnerNamespace: "runner", DependencyProfile: "offline", DependencyRevision: "v1"}))
			req := types.RunRequest{RunID: "legacy", SessionID: "session"}
			var first types.RunResult
			if stream {
				first, err = e.Stream(context.Background(), req, nil)
			} else {
				first, err = e.Run(context.Background(), req, nil)
			}
			if err != nil || first.NativeCheckpoint == nil {
				t.Fatalf("initial pause=%+v err=%v", first, err)
			}
			cp := first.TerminalOutcome
			d := types.DynamicActionDecision{Decision: types.DynamicActionDecisionConfirm, Token: cp.PendingAction.Token, RunID: "legacy", SessionID: "session", CheckpointID: cp.CheckpointID, CheckpointVersion: cp.CheckpointVersion, CheckpointDigest: cp.CheckpointDigest, IdempotencyKey: "first"}
			successor, err := e.ResumeDynamicAction(context.Background(), d, nil, stream)
			if err != nil || successor.NativeCheckpoint == nil || successor.NativeCheckpoint.PredecessorCheckpointID != first.NativeCheckpoint.CheckpointID {
				t.Fatalf("successor=%+v err=%v", successor, err)
			}
			duplicate, err := e.ResumeDynamicAction(context.Background(), d, nil, stream)
			if err != nil || duplicate.NativeCheckpoint == nil || duplicate.NativeCheckpoint.CheckpointID != successor.NativeCheckpoint.CheckpointID || modelCalls != 2 || toolCalls != 2 {
				t.Fatalf("predecessor duplicate=%+v model=%d tool=%d err=%v", duplicate, modelCalls, toolCalls, err)
			}
		})
	}
}

type failingNativeStore struct {
	types.DurableRunCheckpointStore
	save, commit bool
}

func (s *failingNativeStore) Save(ctx context.Context, p types.PersistedRunCheckpoint) error {
	if s.save {
		return types.ErrCheckpointUnavailable
	}
	return s.DurableRunCheckpointStore.Save(ctx, p)
}
func (s *failingNativeStore) CommitResumeResult(ctx context.Context, c types.ResumeClaim, r types.ResumeResult) (types.ResumeClaim, error) {
	if s.commit {
		return types.ResumeClaim{}, types.ErrCheckpointUnavailable
	}
	return s.DurableRunCheckpointStore.CommitResumeResult(ctx, c, r)
}

func TestNativeContinuationCancellationAndTerminalWriteFailure(t *testing.T) {
	for _, mode := range []string{"cancel", "commit_failure", "deny", "timeout"} {
		t.Run(mode, func(t *testing.T) {
			memory, err := checkpoint.NewMemoryStore(checkpoint.Options{Namespace: "runner", Capacity: 8})
			if err != nil {
				t.Fatal(err)
			}
			store := &failingNativeStore{DurableRunCheckpointStore: memory}
			binding := NativeContinuationOptions{Store: store, OwnerNamespace: "runner", DependencyProfile: "offline", DependencyRevision: "v1"}
			e, _, _ := newDynamicActionResumeEventScenario(t, false, WithNativeContinuation(binding))
			ref, err := e.NativeCheckpoint(context.Background(), "run-events")
			if err != nil {
				t.Fatal(err)
			}
			request := types.HostResumeRequest{NativeCheckpoint: &ref, Decision: types.DynamicActionDecision{Decision: types.DynamicActionDecisionConfirm, Token: "opaque-events", RunID: ref.RunID, SessionID: ref.SessionID, CheckpointID: ref.CheckpointID, CheckpointVersion: ref.SchemaVersion, CheckpointDigest: ref.Digest, IdempotencyKey: "same"}}
			calls := 0
			fresh := New(&fakeModel{generate: func(context.Context, types.ModelRequest) (types.ModelResponse, error) {
				calls++
				return types.ModelResponse{FinalAnswer: "done"}, nil
			}}, WithNativeContinuation(binding))
			events := &dynamicActionEventCollector{}
			if mode == "cancel" {
				status, err := e.CancelRun(ref.RunID, ref.SessionID)
				if err != nil || status != types.HostAdmissionStatusAccepted {
					t.Fatalf("cancel=%s %v", status, err)
				}
				if events.count("run.finished") != 0 {
					t.Fatal("cancellation acceptance invented terminal")
				}
			}
			if mode == "commit_failure" {
				store.commit = true
			}
			if mode == "deny" {
				request.Decision.Decision = types.DynamicActionDecisionDeny
			}
			if mode == "timeout" {
				request.Decision.Decision = types.DynamicActionDecisionTimeout
			}
			r, err := fresh.ResumeFromCheckpoint(context.Background(), request, events, false)
			switch mode {
			case "cancel":
				if !errors.Is(err, types.ErrCheckpointTerminal) || calls != 0 {
					t.Fatalf("canceled continuation=%+v calls=%d err=%v", r, calls, err)
				}
			case "commit_failure":
				if !errors.Is(err, types.ErrCheckpointUnavailable) || r.TerminalOutcome != nil || events.count("run.finished") != 0 || events.count("run.checkpoint.terminal") != 0 || calls != 1 {
					t.Fatalf("failed commit leaked terminal: %+v events=%+v calls=%d err=%v", r, events.events, calls, err)
				}
				store.commit = false
				if _, err = fresh.ResumeFromCheckpoint(context.Background(), request, nil, false); !errors.Is(err, types.ErrCheckpointInProgress) || calls != 1 {
					t.Fatalf("recovery reran source: calls=%d err=%v", calls, err)
				}
			default:
				if err != nil || r.TerminalOutcome == nil || r.TerminalOutcome.State != types.RunStateCanceled || calls != 0 || events.count("run.checkpoint.terminal") != 1 {
					t.Fatalf("decision=%+v calls=%d err=%v", r, calls, err)
				}
			}
		})
	}
}

func TestNativeContinuationSuccessorIsolatedFromPredecessor(t *testing.T) {
	store, err := checkpoint.NewMemoryStore(checkpoint.Options{Namespace: "runner", Capacity: 8})
	if err != nil {
		t.Fatal(err)
	}
	binding := NativeContinuationOptions{Store: store, OwnerNamespace: "runner", DependencyProfile: "offline", DependencyRevision: "v1"}
	e, _, _ := newDynamicActionResumeEventScenario(t, false, WithNativeContinuation(binding))
	ref, err := e.NativeCheckpoint(context.Background(), "run-events")
	if err != nil {
		t.Fatal(err)
	}
	request := types.HostResumeRequest{NativeCheckpoint: &ref, Decision: types.DynamicActionDecision{Decision: types.DynamicActionDecisionConfirm, Token: "opaque-events", RunID: ref.RunID, SessionID: ref.SessionID, CheckpointID: ref.CheckpointID, CheckpointVersion: ref.SchemaVersion, CheckpointDigest: ref.Digest, IdempotencyKey: "predecessor"}}
	reg := local.NewRegistry()
	toolCalls := 0
	_, err = reg.Register(&fakeTool{name: "second", invoke: func(context.Context, map[string]any) (types.ToolResult, error) {
		toolCalls++
		return types.ToolResult{PendingAction: &types.DynamicActionReference{Token: "second", Kind: "input", Resumable: true, RunID: ref.RunID, SessionID: ref.SessionID, Iteration: 2, CallID: "second", Source: "offline", Digest: "action2"}}, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	modelCalls := 0
	model := &fakeModel{generate: func(context.Context, types.ModelRequest) (types.ModelResponse, error) {
		modelCalls++
		return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "second", Name: "local.second"}}}, nil
	}}
	fresh := New(model, WithLocalRegistry(reg), WithNativeContinuation(binding))
	successor, err := fresh.ResumeFromCheckpoint(context.Background(), request, nil, false)
	if err != nil || successor.NativeCheckpoint == nil || successor.NativeCheckpoint.PredecessorCheckpointID != ref.CheckpointID || successor.NativeCheckpoint.ResumeAttemptID == "" {
		t.Fatalf("successor=%+v err=%v", successor, err)
	}
	dup, err := fresh.ResumeFromCheckpoint(context.Background(), request, nil, false)
	if err != nil || dup.NativeCheckpoint == nil || dup.NativeCheckpoint.CheckpointID != successor.NativeCheckpoint.CheckpointID || toolCalls != 1 || modelCalls != 1 {
		t.Fatalf("predecessor reused successor: %+v tools=%d models=%d err=%v", dup, toolCalls, modelCalls, err)
	}
	last := New(&fakeModel{generate: func(context.Context, types.ModelRequest) (types.ModelResponse, error) {
		return types.ModelResponse{FinalAnswer: "done"}, nil
	}}, WithNativeContinuation(binding))
	result, err := last.ResumeFromCheckpoint(context.Background(), nativeResumeRequest(successor, "successor"), nil, false)
	if err != nil || result.FinalAnswer != "done" || result.Iterations != 3 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}
