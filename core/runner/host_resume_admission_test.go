package runner

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/tool/local"
)

func eventDecision(cp *types.TerminalOutcome) types.DynamicActionDecision {
	return types.DynamicActionDecision{Decision: types.DynamicActionDecisionConfirm, Token: "opaque-events", RunID: "run-events", SessionID: "session-events", CheckpointID: cp.CheckpointID, CheckpointVersion: cp.CheckpointVersion, CheckpointDigest: cp.CheckpointDigest, IdempotencyKey: "resume-key", Reference: "confirmation"}
}

func TestHostResumePublicPauseCannotMutateSource(t *testing.T) {
	for _, stream := range []bool{false, true} {
		e, pause, events := newDynamicActionResumeEventScenario(t, stream)
		d := eventDecision(pause)
		pause.PendingAction.Token = "caller-mutated"
		result, err := e.ResumeDynamicAction(context.Background(), d, events, stream)
		if err != nil || result.FinalAnswer != "confirmed" {
			t.Fatalf("public pause changed source: stream=%v result=%+v err=%v", stream, result, err)
		}
	}
}

func TestHostResumeRetainedResultIsDetached(t *testing.T) {
	for _, stream := range []bool{false, true} {
		e, pause, events := newDynamicActionResumeEventScenario(t, stream)
		d := eventDecision(pause)
		d.Decision = types.DynamicActionDecisionDeny
		first, err := e.ResumeDynamicAction(context.Background(), d, events, stream)
		if err != nil {
			t.Fatal(err)
		}
		first.TerminalOutcome.State = types.RunStateCompleted
		duplicate, err := e.ResumeDynamicAction(context.Background(), d, events, stream)
		if err != nil || duplicate.TerminalOutcome.State != types.RunStateCanceled {
			t.Fatalf("caller mutated retained result: %+v err=%v", duplicate, err)
		}
		duplicate.TerminalOutcome.SourceReason = "caller-mutated"
		last, err := e.ResumeDynamicAction(context.Background(), d, events, stream)
		if err != nil || last.TerminalOutcome.SourceReason != "deny" {
			t.Fatalf("duplicate mutated retained result: %+v err=%v", last, err)
		}
	}
}

func TestHostResumeRetainedFailureAndCollections(t *testing.T) {
	e, pause, events := newDynamicActionResumeEventScenario(t, false)
	d := eventDecision(pause)
	failure := errors.New("provider failed")
	e.model.(*fakeModel).generate = func(context.Context, types.ModelRequest) (types.ModelResponse, error) {
		return types.ModelResponse{}, failure
	}
	first, err := e.ResumeDynamicAction(context.Background(), d, events, false)
	var executionErr *types.DynamicActionContinuationError
	if !errors.As(err, &executionErr) || !errors.Is(err, failure) || first.Error == nil {
		t.Fatalf("continuation error lost: result=%+v err=%v", first, err)
	}
	*executionErr = types.DynamicActionContinuationError{}
	first.Error.Message = "caller-mutated"
	duplicate, duplicateErr := e.ResumeDynamicAction(context.Background(), d, events, false)
	if !errors.Is(duplicateErr, failure) || duplicate.Error.Message == first.Error.Message {
		t.Fatalf("retained failure changed: result=%+v err=%v", duplicate, duplicateErr)
	}
	// Exercise permitted structured diagnostic details through the public read
	// boundary, with distinct scalar types retained rather than JSON roundtrip.
	type detail struct{ Values []string }
	cp := e.dynamicCheckpoints[d.RunID]
	cp.result.Warnings = []string{"warning"}
	cp.result.ToolCalls = []types.ToolCallSummary{{Name: "tool", Error: &types.ClassifiedError{Message: "tool failed", Details: map[string]any{"nested": []any{map[string]any{"typed": []int{7}}, &detail{Values: []string{"original"}}}}}}}
	e.dynamicCheckpoints[d.RunID] = cp
	read, _ := e.ResumeDynamicAction(context.Background(), d, events, false)
	read.Warnings[0] = "changed"
	read.ToolCalls[0].Error.Message = "changed"
	nested := read.ToolCalls[0].Error.Details["nested"].([]any)
	nested[0].(map[string]any)["typed"].([]int)[0] = 9
	nested[1].(*detail).Values[0] = "changed"
	again, _ := e.ResumeDynamicAction(context.Background(), d, events, false)
	if !reflect.DeepEqual(again, cp.result) {
		t.Fatalf("retained collections changed: %+v", again)
	}
	stored := again.ToolCalls[0].Error.Details["nested"].([]any)
	if again.Warnings[0] != "warning" || again.ToolCalls[0].Error.Message != "tool failed" || stored[0].(map[string]any)["typed"].([]int)[0] != 7 || stored[1].(*detail).Values[0] != "original" {
		t.Fatal("public duplicate aliases retained collections")
	}
}

func trustedResumeResult(r types.HostResumeRequest) types.HostResumeResult {
	f := types.HostResumeFacts{Plan: types.HostResumePlan{ID: "plan", Version: 1, Digest: "plan-digest"}, BusinessRevision: "revision", AuthorityDigest: "authority", ToolCapabilityDigest: "tools", WriteScopeDigest: "scope", ConfirmationReference: "confirmation", EffectReference: "effect", DecisionAuthorized: true, ExecutionAuthorized: true, CancellationAuthorized: true, ConfirmationValid: true, EffectEligible: true}
	return types.HostResumeResult{Version: types.HostResumeAdmissionVersion, Status: types.RunResumeAdmissionAccepted, Binding: types.HostResumeBinding{Version: types.HostResumeAdmissionVersion, Checkpoint: r.Detached().Checkpoint, Plan: f.Plan, BusinessRevision: f.BusinessRevision, AuthorityDigest: f.AuthorityDigest, ToolCapabilityDigest: f.ToolCapabilityDigest, WriteScopeDigest: f.WriteScopeDigest, ConfirmationReference: f.ConfirmationReference, EffectReference: f.EffectReference}, Current: f}
}

func TestHostResumeRejectionPreservesCheckpointAndKey(t *testing.T) {
	for _, stream := range []bool{false, true} {
		for _, reason := range []types.HostResumeReason{types.HostResumeAuthorityDenied, types.HostResumeRevisionDrift, types.HostResumePlanDrift, types.HostResumeScopeDrift, types.HostResumeConfirmationInvalid, types.HostResumeEffectConflict, types.HostResumeBindingMismatch, types.HostResumeInvalidBinding, types.HostResumeRevalidationFailed} {
			t.Run(string(reason)+map[bool]string{true: "/stream", false: "/run"}[stream], func(t *testing.T) {
				var bound types.HostResumeResult
				calls := 0
				deny := true
				verifier := types.HostResumeVerifierFunc(func(_ context.Context, r types.HostResumeRequest) (types.HostResumeResult, error) {
					calls++
					x := bound
					if deny {
						switch reason {
						case types.HostResumeAuthorityDenied:
							x.Current.DecisionAuthorized = false
						case types.HostResumeRevisionDrift:
							x.Current.BusinessRevision = "changed"
						case types.HostResumePlanDrift:
							x.Current.Plan.Version++
						case types.HostResumeScopeDrift:
							x.Current.WriteScopeDigest = "changed"
						case types.HostResumeConfirmationInvalid:
							x.Current.ConfirmationValid = false
						case types.HostResumeEffectConflict:
							x.Current.EffectEligible = false
						case types.HostResumeBindingMismatch:
							x.Binding.Checkpoint.CheckpointID = "other"
						case types.HostResumeInvalidBinding:
							x.Version = "unsupported"
						case types.HostResumeRevalidationFailed:
							return x, errors.New("secret callback body")
						}
					}
					// Mutating callback's detached snapshot must not change source.
					r.Checkpoint.PendingAction.Token = "malicious"
					return x, nil
				})
				e, cp, events := newDynamicActionResumeEventScenario(t, stream, WithHostResumeVerifier(verifier))
				bound = trustedResumeResult(types.HostResumeRequest{Checkpoint: e.dynamicCheckpoints["run-events"].checkpoint})
				d := eventDecision(cp)
				before := len(events.events)
				_, err := e.ResumeDynamicAction(context.Background(), d, events, stream)
				if !errors.Is(err, reason) {
					t.Fatalf("err=%v want=%s", err, reason)
				}
				if len(events.events) != before || e.dynamicCheckpoints[d.RunID].resumed {
					t.Fatal("rejection mutated source")
				}
				deny = false
				r, err := e.ResumeDynamicAction(context.Background(), d, events, stream)
				if err != nil || r.FinalAnswer != "confirmed" || r.Iterations != 2 {
					t.Fatalf("retry result=%+v err=%v", r, err)
				}
				if _, err = e.ResumeDynamicAction(context.Background(), d, events, stream); err != nil {
					t.Fatal(err)
				}
				if calls != 2 || events.count("run.dynamic_action.resolved") != 1 || events.canceledFinishedCount() != 0 {
					t.Fatal("duplicate revalidated or added events")
				}
			})
		}
	}
}

func TestHostResumeNilPanicAndCanceledContextFailClosed(t *testing.T) {
	for _, name := range []string{"nil", "typed nil", "panic", "canceled"} {
		t.Run(name, func(t *testing.T) {
			var v types.HostResumeVerifier
			if name == "typed nil" {
				v = types.HostResumeVerifierFunc(nil)
			}
			if name == "panic" {
				v = types.HostResumeVerifierFunc(func(context.Context, types.HostResumeRequest) (types.HostResumeResult, error) { panic("secret") })
			}
			ctx := context.Background()
			if name == "canceled" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			e, cp, ev := newDynamicActionResumeEventScenario(t, false, WithHostResumeVerifier(v))
			_, err := e.ResumeDynamicAction(ctx, eventDecision(cp), ev, false)
			if !errors.Is(err, types.HostResumeRevalidationFailed) || e.dynamicCheckpoints["run-events"].resumed || ev.count("run.dynamic_action.resolved") != 0 {
				t.Fatalf("err=%v consumed=%v", err, e.dynamicCheckpoints["run-events"].resumed)
			}
		})
	}
}

func TestHostResumeRechecksSourceAfterCallback(t *testing.T) {
	for _, change := range []string{"cancel", "expiry", "replacement", "context"} {
		t.Run(change, func(t *testing.T) {
			var e *Engine
			var bound types.HostResumeResult
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			v := types.HostResumeVerifierFunc(func(_ context.Context, _ types.HostResumeRequest) (types.HostResumeResult, error) {
				if _, ok := e.ActiveRun("run-events"); !ok {
					t.Fatal("source not queryable")
				}
				switch change {
				case "cancel":
					if _, err := e.CancelRun("run-events", "session-events"); err != nil {
						t.Fatal(err)
					}
				case "expiry", "replacement":
					e.dynamicActionMu.Lock()
					cp := e.dynamicCheckpoints["run-events"]
					if change == "expiry" {
						cp.checkpoint.CreatedAt = time.Now().Add(-25 * time.Hour)
					} else {
						cp.generation++
					}
					e.dynamicCheckpoints["run-events"] = cp
					e.dynamicActionMu.Unlock()
				case "context":
					cancel()
				}
				return bound, nil
			})
			var cp *types.TerminalOutcome
			var ev *dynamicActionEventCollector
			e, cp, ev = newDynamicActionResumeEventScenario(t, false, WithHostResumeVerifier(v))
			bound = trustedResumeResult(types.HostResumeRequest{Checkpoint: e.dynamicCheckpoints["run-events"].checkpoint})
			_, err := e.ResumeDynamicAction(ctx, eventDecision(cp), ev, false)
			if err == nil || e.dynamicCheckpoints["run-events"].resumed || ev.count("run.dynamic_action.resolved") != 0 {
				t.Fatalf("race accepted: err=%v", err)
			}
		})
	}
}

func TestHostResumeConcurrentCallbacksCommitOnce(t *testing.T) {
	for _, same := range []bool{false, true} {
		var bound types.HostResumeResult
		entered := make(chan struct{}, 2)
		release := make(chan struct{})
		v := types.HostResumeVerifierFunc(func(context.Context, types.HostResumeRequest) (types.HostResumeResult, error) {
			entered <- struct{}{}
			<-release
			return bound, nil
		})
		e, cp, events := newDynamicActionResumeEventScenario(t, false, WithHostResumeVerifier(v))
		var continuations atomic.Int32
		e.model.(*fakeModel).generate = func(context.Context, types.ModelRequest) (types.ModelResponse, error) {
			continuations.Add(1)
			return types.ModelResponse{FinalAnswer: "continued"}, nil
		}
		bound = trustedResumeResult(types.HostResumeRequest{Checkpoint: e.dynamicCheckpoints["run-events"].checkpoint})
		d := eventDecision(cp)
		other := d
		if !same {
			other.IdempotencyKey = "other"
		}
		var wg sync.WaitGroup
		errs := make(chan error, 2)
		for _, decision := range []types.DynamicActionDecision{d, other} {
			wg.Add(1)
			go func(d types.DynamicActionDecision) {
				defer wg.Done()
				_, err := e.ResumeDynamicAction(context.Background(), d, events, false)
				errs <- err
			}(decision)
		}
		<-entered
		<-entered
		close(release)
		wg.Wait()
		close(errs)
		success := 0
		for err := range errs {
			if err == nil {
				success++
			} else if err != ErrDynamicActionConflict && err != ErrDynamicActionTerminal && err != ErrDynamicActionInProgress {
				t.Fatal(err)
			}
		}
		if (!same && success != 1) || (same && success < 1) {
			t.Fatalf("same=%v successes=%d", same, success)
		}
		if continuations.Load() != 1 || events.count("run.dynamic_action.resolved") != 1 || events.canceledFinishedCount() != 0 {
			t.Fatal("concurrent callbacks committed multiple continuations or events")
		}
	}
}

func TestHostResumeDuplicateRequiresFullIdentity(t *testing.T) {
	for _, guarded := range []bool{false, true} {
		t.Run(fmt.Sprintf("guard=%v", guarded), func(t *testing.T) {
			var bound types.HostResumeResult
			calls := 0
			var opts []Option
			if guarded {
				opts = append(opts, WithHostResumeVerifier(types.HostResumeVerifierFunc(func(context.Context, types.HostResumeRequest) (types.HostResumeResult, error) {
					calls++
					return bound, nil
				})))
			}
			e, cp, events := newDynamicActionResumeEventScenario(t, false, opts...)
			bound = trustedResumeResult(types.HostResumeRequest{Checkpoint: e.dynamicCheckpoints["run-events"].checkpoint})
			d := eventDecision(cp)
			first, err := e.ResumeDynamicAction(context.Background(), d, events, false)
			if err != nil {
				t.Fatal(err)
			}
			for _, field := range []string{"run", "session", "checkpoint", "version", "digest", "token", "reference", "key", "kind", "mode"} {
				t.Run(field, func(t *testing.T) {
					changed := d
					switch field {
					case "run":
						changed.RunID += "-changed"
					case "session":
						changed.SessionID += "-changed"
					case "checkpoint":
						changed.CheckpointID += "-changed"
					case "version":
						changed.CheckpointVersion += "-changed"
					case "digest":
						changed.CheckpointDigest += "-changed"
					case "token":
						changed.Token += "-changed"
					case "reference":
						changed.Reference += "-changed"
					case "key":
						changed.IdempotencyKey += "-changed"
					case "kind":
						changed.Decision = types.DynamicActionDecisionDeny
					}
					result, err := e.ResumeDynamicAction(context.Background(), changed, events, field == "mode")
					if err == nil {
						t.Fatal("non-equivalent request returned duplicate success")
					}
					if result.RunID != "" || result.FinalAnswer != "" || result.TerminalOutcome != nil {
						t.Fatal("non-equivalent request disclosed retained result")
					}
				})
			}
			// Surrounding spaces normalize to the exact previously admitted identity.
			d.RunID = " " + d.RunID + " "
			d.SessionID = " " + d.SessionID + " "
			d.CheckpointID = " " + d.CheckpointID + " "
			d.CheckpointVersion = " " + d.CheckpointVersion + " "
			d.CheckpointDigest = " " + d.CheckpointDigest + " "
			d.Token = " " + d.Token + " "
			d.Reference = " " + d.Reference + " "
			d.IdempotencyKey = " " + d.IdempotencyKey + " "
			duplicate, err := e.ResumeDynamicAction(context.Background(), d, events, false)
			if err != nil || duplicate.FinalAnswer != first.FinalAnswer || duplicate.Iterations != first.Iterations {
				t.Fatalf("normalized duplicate=%+v err=%v", duplicate, err)
			}
			if events.count("run.dynamic_action.resolved") != 1 || (guarded && calls != 1) {
				t.Fatal("duplicate added resolution or verifier call")
			}
		})
	}
}

func TestHostResumeInFlightIsNotEmptySuccess(t *testing.T) {
	e, cp, _ := newDynamicActionResumeEventScenario(t, false)
	entered, release := make(chan struct{}), make(chan struct{})
	e.model.(*fakeModel).generate = func(context.Context, types.ModelRequest) (types.ModelResponse, error) {
		close(entered)
		<-release
		return types.ModelResponse{FinalAnswer: "continued"}, nil
	}
	d := eventDecision(cp)
	done := make(chan error, 1)
	go func() { _, err := e.ResumeDynamicAction(context.Background(), d, nil, false); done <- err }()
	<-entered
	_, err := e.ResumeDynamicAction(context.Background(), d, nil, false)
	close(release)
	if first := <-done; first != nil {
		t.Fatal(first)
	}
	if err == nil || !strings.Contains(err.Error(), "in progress") {
		t.Fatalf("in-flight duplicate err=%v", err)
	}
}

func TestHostResumeContinuesIteration(t *testing.T) {
	for _, stream := range []bool{false, true} {
		e, cp, _ := newDynamicActionResumeEventScenario(t, stream)
		result, err := e.ResumeDynamicAction(context.Background(), eventDecision(cp), nil, stream)
		if err != nil || result.Iterations != 2 {
			t.Fatalf("stream=%v iterations=%d err=%v", stream, result.Iterations, err)
		}
	}
}

func TestHostResumeNewRunAndRetryDoNotConsumeCheckpoint(t *testing.T) {
	for _, stream := range []bool{false, true} {
		e, cp, events := newDynamicActionResumeEventScenario(t, stream, WithHostResumeVerifier(nil))
		original := e.dynamicCheckpoints["run-events"]
		model := e.model.(*fakeModel)
		model.generate = func(context.Context, types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{FinalAnswer: "new run"}, nil
		}
		model.stream = func(_ context.Context, _ types.ModelRequest, emit func(types.ModelEvent) error) error {
			return emit(types.ModelEvent{Type: types.ModelEventTypeFinalAnswer, TextDelta: "new run"})
		}
		req := types.RunRequest{RunID: "new-run", SessionID: "session-events", Input: "new input"}
		var result types.RunResult
		var err error
		if stream {
			result, err = e.Stream(context.Background(), req, nil)
		} else {
			result, err = e.Run(context.Background(), req, nil)
		}
		if err != nil || result.FinalAnswer != "new run" {
			t.Fatalf("new run=%+v err=%v", result, err)
		}
		previous := types.RunRef{RunID: "run-events", SessionID: "session-events", State: types.RunStateInputRequired, Source: types.ProtocolSourceRunner}
		req.RunID = "retry-run"
		if _, err := e.RetryRun(context.Background(), previous, req, nil, stream); err == nil {
			t.Fatal("retry accepted a paused source as recovery")
		}
		previous.RunID, previous.State = "failed-run", types.RunStateFailed
		if _, err := e.RetryRun(context.Background(), previous, req, nil, stream); err != nil {
			t.Fatal(err)
		}
		current := e.dynamicCheckpoints["run-events"]
		if current.generation != original.generation || current.resumed || current.checkpoint.State != types.RunStateInputRequired || current.checkpoint.Iteration != original.checkpoint.Iteration || events.count("run.dynamic_action.resolved") != 0 {
			t.Fatal("new run or retry consumed the paused checkpoint")
		}
		if _, err := e.ResumeDynamicAction(context.Background(), eventDecision(cp), nil, stream); err != types.HostResumeRevalidationFailed {
			t.Fatalf("original checkpoint guard bypassed: %v", err)
		}
	}
}

func TestHostResumeNextPendingActionIsNewGeneration(t *testing.T) {
	for _, stream := range []bool{false, true} {
		t.Run(fmt.Sprint(stream), func(t *testing.T) {
			reg := local.NewRegistry()
			toolCalls, modelCalls := 0, 0
			_, err := reg.Register(&fakeTool{name: "prepare", invoke: func(context.Context, map[string]any) (types.ToolResult, error) {
				toolCalls++
				return types.ToolResult{PendingAction: &types.DynamicActionReference{Token: "token", Kind: "prepare", Resumable: true, RunID: "run-next", SessionID: "session-next", Iteration: toolCalls, CallID: "same-call", Source: "test", Digest: "same-digest"}}, nil
			}})
			if err != nil {
				t.Fatal(err)
			}
			generate := func(context.Context, types.ModelRequest) (types.ModelResponse, error) {
				modelCalls++
				if modelCalls <= 2 {
					return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "same-call", Name: "local.prepare"}}}, nil
				}
				return types.ModelResponse{FinalAnswer: "done"}, nil
			}
			model := &fakeModel{generate: generate, stream: func(ctx context.Context, r types.ModelRequest, onEvent func(types.ModelEvent) error) error {
				x, err := generate(ctx, r)
				if err != nil {
					return err
				}
				if len(x.ToolCalls) > 0 {
					return onEvent(types.ModelEvent{Type: types.ModelEventTypeToolCall, ToolCall: &x.ToolCalls[0]})
				}
				return onEvent(types.ModelEvent{Type: types.ModelEventTypeFinalAnswer, TextDelta: x.FinalAnswer})
			}}
			e := New(model, WithLocalRegistry(reg))
			req := types.RunRequest{RunID: "run-next", SessionID: "session-next", Input: "x"}
			var first types.RunResult
			if stream {
				first, err = e.Stream(context.Background(), req, nil)
			} else {
				first, err = e.Run(context.Background(), req, nil)
			}
			if err != nil {
				t.Fatal(err)
			}
			d := types.DynamicActionDecision{Decision: types.DynamicActionDecisionConfirm, Token: "token", RunID: req.RunID, SessionID: req.SessionID, CheckpointID: first.TerminalOutcome.CheckpointID, CheckpointVersion: first.TerminalOutcome.CheckpointVersion, CheckpointDigest: first.TerminalOutcome.CheckpointDigest, IdempotencyKey: "key"}
			second, err := e.ResumeDynamicAction(context.Background(), d, nil, stream)
			if err != nil || second.TerminalOutcome == nil || second.TerminalOutcome.State != types.RunStateInputRequired || second.Iterations != 2 {
				t.Fatalf("second=%+v err=%v", second, err)
			}
			if second.TerminalOutcome.CheckpointID == d.CheckpointID {
				t.Fatal("checkpoint identity reused")
			}
			if _, err = e.ResumeDynamicAction(context.Background(), d, nil, stream); err != ErrDynamicActionStale {
				t.Fatalf("old decision err=%v", err)
			}
			d.CheckpointID = second.TerminalOutcome.CheckpointID
			d.IdempotencyKey = "second-key"
			last, err := e.ResumeDynamicAction(context.Background(), d, nil, stream)
			if err != nil || last.FinalAnswer != "done" || last.Iterations != 3 || modelCalls != 3 || toolCalls != 2 {
				t.Fatalf("last=%+v err=%v calls=%d/%d", last, err, modelCalls, toolCalls)
			}
		})
	}
}

func TestHostResumeCancellationAuthorityAndRestartScope(t *testing.T) {
	for _, stream := range []bool{false, true} {
		for _, kind := range []types.DynamicActionDecisionKind{types.DynamicActionDecisionDeny, types.DynamicActionDecisionTimeout} {
			var bound types.HostResumeResult
			calls := 0
			v := types.HostResumeVerifierFunc(func(context.Context, types.HostResumeRequest) (types.HostResumeResult, error) {
				calls++
				return bound, nil
			})
			e, cp, ev := newDynamicActionResumeEventScenario(t, stream, WithHostResumeVerifier(v))
			d := eventDecision(cp)
			d.Decision = kind
			bound = trustedResumeResult(types.HostResumeRequest{Checkpoint: e.dynamicCheckpoints[d.RunID].checkpoint})
			bound.Current.ExecutionAuthorized = false
			bound.Current.BusinessRevision = "changed"
			r, err := e.ResumeDynamicAction(context.Background(), d, ev, stream)
			if err != nil || r.TerminalOutcome.State != types.RunStateCanceled || ev.count("run.dynamic_action.resolved") != 1 || ev.canceledFinishedCount() != 1 {
				t.Fatalf("result=%+v err=%v", r, err)
			}
			if _, err = e.ResumeDynamicAction(context.Background(), d, ev, stream); err != nil || calls != 1 {
				t.Fatal("cancel duplicate revalidated")
			}
			providerCalls := 0
			fresh := New(&fakeModel{generate: func(context.Context, types.ModelRequest) (types.ModelResponse, error) {
				providerCalls++
				return types.ModelResponse{}, nil
			}}, WithHostResumeVerifier(v))
			if _, err = fresh.ResumeDynamicAction(context.Background(), d, nil, stream); err != ErrDynamicActionUnknown || providerCalls != 0 || calls != 1 {
				t.Fatalf("metadata-only restart err=%v calls=%d", err, providerCalls)
			}
		}
	}
}
