package runner

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/FelixSeptem/baymax/core/checkpoint"
	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
	"github.com/FelixSeptem/baymax/tool/local"
)

func TestNativeContinuationNewEngineRestoresCompletedToolBoundary(t *testing.T) {
	for _, stream := range []bool{false, true} {
		t.Run(map[bool]string{false: "run", true: "stream"}[stream], func(t *testing.T) {
			const prefix = "BAYMAX_NATIVE_POLICY_TEST"
			t.Setenv(prefix+"_CONTEXT_ASSEMBLER_ENABLED", "false")
			t.Setenv(prefix+"_CONCURRENCY_LOCAL_MAX_WORKERS", "3")
			t.Setenv(prefix+"_CONCURRENCY_LOCAL_QUEUE_SIZE", "7")
			t.Setenv(prefix+"_CONCURRENCY_BACKPRESSURE", "reject")
			mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: prefix})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = mgr.Close() })
			store, err := checkpoint.NewMemoryStore(checkpoint.Options{Namespace: "runner", Capacity: 16})
			if err != nil {
				t.Fatal(err)
			}
			binding := NativeContinuationOptions{Store: store, OwnerNamespace: "runner", DependencyProfile: "offline", DependencyRevision: "v1"}
			engine, _, _ := newDynamicActionResumeEventScenario(t, stream, WithNativeContinuation(binding), WithRuntimeManager(mgr))
			ref, err := engine.NativeCheckpoint(context.Background(), "run-events")
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			assertPolicy := func(ctx context.Context) {
				t.Helper()
				policy := nativeExecutionFrom(ctx).Request.Policy
				if policy == nil || policy.LocalDispatch.MaxWorkers != 3 || policy.LocalDispatch.QueueSize != 7 || policy.LocalDispatch.Backpressure != types.BackpressureReject {
					t.Fatalf("restored runtime policy=%+v", policy)
				}
			}
			model := &fakeModel{
				generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
					assertPolicy(ctx)
					calls++
					if req.RunID != "run-events" || len(req.ToolResult) != 1 {
						t.Fatalf("restored request=%#v", req)
					}
					return types.ModelResponse{FinalAnswer: "restored"}, nil
				},
				stream: func(ctx context.Context, req types.ModelRequest, emit func(types.ModelEvent) error) error {
					assertPolicy(ctx)
					calls++
					if req.RunID != "run-events" || len(req.ToolResult) != 1 {
						t.Fatalf("restored request=%#v", req)
					}
					return emit(types.ModelEvent{Type: types.ModelEventTypeFinalAnswer, TextDelta: "restored"})
				},
			}
			fresh := New(model, WithNativeContinuation(binding), WithRuntimeManager(mgr))
			request := types.HostResumeRequest{NativeCheckpoint: &ref, Decision: types.DynamicActionDecision{
				Decision: types.DynamicActionDecisionConfirm, Token: "opaque-events", RunID: ref.RunID, SessionID: ref.SessionID,
				CheckpointID: ref.CheckpointID, CheckpointVersion: ref.SchemaVersion, CheckpointDigest: ref.Digest, IdempotencyKey: "restore-1",
			}}
			result, err := fresh.ResumeFromCheckpoint(context.Background(), request, nil, stream)
			if err != nil || result.RunID != ref.RunID || result.FinalAnswer != "restored" || result.Iterations != 2 {
				t.Fatalf("result=%#v err=%v", result, err)
			}
			duplicate, err := New(model, WithNativeContinuation(binding), WithRuntimeManager(mgr)).ResumeFromCheckpoint(context.Background(), request, nil, stream)
			if err != nil || duplicate.FinalAnswer != result.FinalAnswer || calls != 1 {
				t.Fatalf("duplicate=%#v calls=%d err=%v", duplicate, calls, err)
			}
			request.Decision.IdempotencyKey = "conflict"
			if _, err := fresh.ResumeFromCheckpoint(context.Background(), request, nil, stream); !errors.Is(err, types.ErrCheckpointConflict) {
				t.Fatalf("conflict=%v", err)
			}
		})
	}
}

func TestNativeContinuationMissingStoreFailsBeforeProvider(t *testing.T) {
	calls := 0
	e := New(&fakeModel{generate: func(context.Context, types.ModelRequest) (types.ModelResponse, error) {
		calls++
		return types.ModelResponse{}, nil
	}})
	if _, err := e.ResumeFromCheckpoint(context.Background(), types.HostResumeRequest{}, nil, false); !errors.Is(err, types.ErrCheckpointUnavailable) || calls != 0 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

func TestNativeContinuationApprovalAndBudgetPauseBeforeDispatch(t *testing.T) {
	for _, reason := range []types.PauseReason{types.PauseReasonApprovalRequired, types.PauseReasonBudgetExhausted} {
		for _, stream := range []bool{false, true} {
			t.Run(string(reason)+map[bool]string{false: "/run", true: "/stream"}[stream], func(t *testing.T) {
				store, err := checkpoint.NewMemoryStore(checkpoint.Options{Namespace: "runner", Capacity: 16})
				if err != nil {
					t.Fatal(err)
				}
				var toolCalls atomic.Int32
				modelCalls := 0
				reg := local.NewRegistry()
				_, err = reg.Register(&fakeTool{name: "pending", invoke: func(context.Context, map[string]any) (types.ToolResult, error) {
					toolCalls.Add(1)
					return types.ToolResult{Content: "ok"}, nil
				}})
				if err != nil {
					t.Fatal(err)
				}
				calls := []types.ToolCall{{CallID: "call", Name: "local.pending"}}
				if reason == types.PauseReasonBudgetExhausted {
					calls = append(calls, types.ToolCall{CallID: "second", Name: "local.pending"})
				}
				model := &fakeModel{generate: func(_ context.Context, req types.ModelRequest) (types.ModelResponse, error) {
					modelCalls++
					if len(req.ToolResult) > 0 {
						return types.ModelResponse{FinalAnswer: "done"}, nil
					}
					return types.ModelResponse{ToolCalls: calls}, nil
				}, stream: func(_ context.Context, req types.ModelRequest, emit func(types.ModelEvent) error) error {
					modelCalls++
					if len(req.ToolResult) > 0 {
						return emit(types.ModelEvent{Type: types.ModelEventTypeFinalAnswer, TextDelta: "done"})
					}
					for _, call := range calls {
						if err := emit(types.ModelEvent{Type: types.ModelEventTypeToolCall, ToolCall: &call}); err != nil {
							return err
						}
					}
					return nil
				}}
				budgetAuthorizations := 0
				binding := NativeContinuationOptions{Store: store, OwnerNamespace: "runner", DependencyProfile: "offline", DependencyRevision: "v1", Approval: true, Budget: true, BudgetOwner: func(_ context.Context, _ types.NativeCheckpointRef, _ types.DurableBudget, proposal types.DurableBudget) (types.DurableBudget, error) {
					budgetAuthorizations++
					if budgetAuthorizations > 1 {
						return types.DurableBudget{}, types.ErrCheckpointConflict
					}
					proposal.Revision = "authorized-v2"
					return proposal, nil
				}}
				opts := []Option{WithNativeContinuation(binding), WithLocalRegistry(reg)}
				if reason == types.PauseReasonApprovalRequired {
					opts = append(opts, WithActionGateMatcher(alwaysRequireConfirmation{}))
				}
				policy := types.DefaultLoopPolicy()
				if reason == types.PauseReasonBudgetExhausted {
					policy.ToolCallLimit = 1
				}
				engine := New(model, opts...)
				req := types.RunRequest{RunID: "native-pending", SessionID: "session", Input: "x", Policy: &policy}
				var paused types.RunResult
				if stream {
					paused, err = engine.Stream(context.Background(), req, nil)
				} else {
					paused, err = engine.Run(context.Background(), req, nil)
				}
				if err != nil || paused.NativeCheckpoint == nil || paused.NativeCheckpoint.PauseReason != reason || toolCalls.Load() != 0 || modelCalls != 1 {
					t.Fatalf("pause=%#v tools=%d models=%d err=%v", paused, toolCalls.Load(), modelCalls, err)
				}
				ref := *paused.NativeCheckpoint
				request := types.HostResumeRequest{NativeCheckpoint: &ref, Decision: types.DynamicActionDecision{Decision: types.DynamicActionDecisionConfirm, Token: paused.TerminalOutcome.PendingAction.Token, RunID: ref.RunID, SessionID: ref.SessionID, CheckpointID: ref.CheckpointID, CheckpointVersion: ref.SchemaVersion, CheckpointDigest: ref.Digest, IdempotencyKey: "confirm"}}
				if reason == types.PauseReasonBudgetExhausted {
					p, err := store.Load(context.Background(), ref)
					if err != nil {
						t.Fatal(err)
					}
					budget := p.Budget
					budget.Revision = "v2"
					budget.ToolLimit = 4
					request.NativeBudget = &budget
				}
				fresh := New(model, opts...)
				result, err := fresh.ResumeFromCheckpoint(context.Background(), request, nil, stream)
				if err != nil || result.FinalAnswer != "done" || int(toolCalls.Load()) != len(calls) || modelCalls != 2 {
					t.Fatalf("result=%#v tools=%d models=%d err=%v", result, toolCalls.Load(), modelCalls, err)
				}
				duplicate, err := New(model, opts...).ResumeFromCheckpoint(context.Background(), request, nil, stream)
				if err != nil || duplicate.FinalAnswer != result.FinalAnswer || int(toolCalls.Load()) != len(calls) || modelCalls != 2 {
					t.Fatalf("retained duplicate=%#v authorizations=%d tools=%d models=%d err=%v", duplicate, budgetAuthorizations, toolCalls.Load(), modelCalls, err)
				}
			})
		}
	}
}

type alwaysRequireConfirmation struct{}

func TestNativeContinuationDisabledBudgetPreservesIterationLimit(t *testing.T) {
	for _, stream := range []bool{false, true} {
		t.Run(map[bool]string{false: "run", true: "stream"}[stream], func(t *testing.T) {
			store, err := checkpoint.NewMemoryStore(checkpoint.Options{Namespace: "runner", Capacity: 4})
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			response := func() types.ModelResponse {
				calls++
				if calls > 2 {
					return types.ModelResponse{FinalAnswer: "unexpected extra iteration"}
				}
				return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "call", Name: "local.pending"}}}
			}
			model := &fakeModel{generate: func(context.Context, types.ModelRequest) (types.ModelResponse, error) {
				return response(), nil
			}, stream: func(_ context.Context, _ types.ModelRequest, emit func(types.ModelEvent) error) error {
				resp := response()
				if resp.FinalAnswer != "" {
					return emit(types.ModelEvent{Type: types.ModelEventTypeFinalAnswer, TextDelta: resp.FinalAnswer})
				}
				return emit(types.ModelEvent{Type: types.ModelEventTypeToolCall, ToolCall: &resp.ToolCalls[0]})
			}}
			reg := local.NewRegistry()
			if _, err := reg.Register(&fakeTool{name: "pending", invoke: func(context.Context, map[string]any) (types.ToolResult, error) {
				return types.ToolResult{Content: "ok"}, nil
			}}); err != nil {
				t.Fatal(err)
			}
			engine := New(model, WithLocalRegistry(reg), WithNativeContinuation(NativeContinuationOptions{Store: store, OwnerNamespace: "runner", DependencyProfile: "offline", DependencyRevision: "v1"}))
			policy := types.DefaultLoopPolicy()
			policy.MaxIterations = 2
			req := types.RunRequest{RunID: "limited", SessionID: "session", Policy: &policy}
			if stream {
				_, err = engine.Stream(context.Background(), req, nil)
			} else {
				_, err = engine.Run(context.Background(), req, nil)
			}
			if err == nil || calls > 2 {
				t.Fatalf("iteration limit bypassed: calls=%d err=%v", calls, err)
			}
		})
	}
}

func (alwaysRequireConfirmation) Evaluate(context.Context, types.ActionGateCheck) (types.ActionGateDecision, error) {
	return types.ActionGateDecisionRequireConfirm, nil
}
