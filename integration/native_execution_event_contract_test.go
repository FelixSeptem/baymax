package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/adapter/modelcapability"
	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/host/eventadapter"
	"github.com/FelixSeptem/baymax/integration/fakes"
	"github.com/FelixSeptem/baymax/observability/event"
	"github.com/FelixSeptem/baymax/orchestration/composer"
	"github.com/FelixSeptem/baymax/orchestration/scheduler"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
	"github.com/FelixSeptem/baymax/tool/diagnosticsreplay"
	"github.com/FelixSeptem/baymax/tool/local"
)

type nativeModel struct {
	generate func(context.Context, types.ModelRequest) (types.ModelResponse, error)
}

func (m nativeModel) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
	return m.generate(ctx, req)
}
func (m nativeModel) Stream(ctx context.Context, req types.ModelRequest, emit func(types.ModelEvent) error) error {
	response, err := m.generate(ctx, req)
	if err != nil {
		return err
	}
	for _, call := range response.ToolCalls {
		if err := emit(types.ModelEvent{Type: types.ModelEventTypeToolCall, ToolCall: &call}); err != nil {
			return err
		}
	}
	if response.FinalAnswer != "" {
		return emit(types.ModelEvent{Type: types.ModelEventTypeFinalAnswer, TextDelta: response.FinalAnswer})
	}
	return nil
}
func nativeEngine(t *testing.T, generate func(context.Context, types.ModelRequest) (types.ModelResponse, error), tool *fakes.Tool) *runner.Engine {
	t.Helper()
	model, err := modelcapability.Wrap(nativeModel{generate}, modelcapability.Config{Provider: "local", Capabilities: []types.ModelCapability{types.ModelCapabilityStreaming, types.ModelCapabilityToolCall}})
	if err != nil {
		t.Fatal(err)
	}
	reg := local.NewRegistry()
	if tool != nil {
		if _, err := reg.Register(tool); err != nil {
			t.Fatal(err)
		}
	}
	return runner.New(model, runner.WithLocalRegistry(reg))
}

type nativeRawCollector struct{ events []types.Event }

func (c *nativeRawCollector) OnEvent(_ context.Context, event types.Event) {
	c.events = append(c.events, event)
}

type nativeFanout struct{ dispatcher *event.Dispatcher }

func (f nativeFanout) OnEvent(ctx context.Context, event types.Event) { f.dispatcher.Emit(ctx, event) }
func nativeObserver(t *testing.T, source types.ProtocolSource, runID string, sink eventadapter.Sink) *eventadapter.Observer {
	t.Helper()
	o, err := eventadapter.New(eventadapter.Options{Projection: types.NativeExecutionOptions{Source: source, ExpectedRunID: runID}, Sink: sink})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(o.Close)
	return o
}
func nativeExecute(ctx context.Context, engine *runner.Engine, stream bool, h types.EventHandler) (types.RunResult, error) {
	req := types.RunRequest{RunID: "native-run", SessionID: "native-session", Input: "fake request"}
	// Stream implicitly requires streaming capability. Make admission inputs
	// identical instead of erasing a capability requirement in normalization.
	req.Capabilities.Required = []types.ModelCapability{types.ModelCapabilityStreaming}
	policy := types.DefaultLoopPolicy()
	policy.ContinueOnToolError = true
	req.Policy = &policy
	if stream {
		return engine.Stream(ctx, req, h)
	}
	return engine.Run(ctx, req, h)
}

func TestNativeExecutionRealRunnerLifecycle(t *testing.T) {
	transcripts := make(map[string]diagnosticsreplay.NativeExecutionTranscript)
	for _, stream := range []bool{false, true} {
		for _, scenario := range []string{"tool", "tool-error", "model-error", "cancel", "deadline"} {
			t.Run(scenario+map[bool]string{false: "/run", true: "/stream"}[stream], func(t *testing.T) {
				ctx := context.Background()
				var cancel context.CancelFunc
				if scenario == "cancel" {
					ctx, cancel = context.WithCancel(ctx)
					cancel()
				}
				if scenario == "deadline" {
					ctx, cancel = context.WithDeadline(ctx, time.Now().Add(-time.Second))
					defer cancel()
				}
				engine := nativeEngine(t, func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
					if err := ctx.Err(); err != nil {
						return types.ModelResponse{}, err
					}
					if scenario == "model-error" {
						return types.ModelResponse{}, errors.New("fake model failure")
					}
					if len(req.ToolResult) == 0 {
						return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "native-call", Name: "local.echo"}}}, nil
					}
					return types.ModelResponse{FinalAnswer: "done"}, nil
				}, &fakes.Tool{NameValue: "echo", InvokeFn: func(context.Context, map[string]any) (types.ToolResult, error) {
					if scenario == "tool-error" {
						return types.ToolResult{}, errors.New("fake tool failure")
					}
					return types.ToolResult{Content: "fake result"}, nil
				}})
				var projected []types.NativeExecutionEvent
				o := nativeObserver(t, types.ProtocolSourceRunner, "native-run", eventadapter.SinkFunc(func(_ context.Context, e types.NativeExecutionEvent) error {
					projected = append(projected, e)
					return nil
				}))
				raw := &nativeRawCollector{}
				h := nativeFanout{event.NewDispatcher(raw, o)}
				result, err := nativeExecute(ctx, engine, stream, h)
				want := types.RunStateCompleted
				if scenario == "model-error" {
					want = types.RunStateFailed
				}
				if scenario == "deadline" {
					want = types.RunStateFailed
				}
				if scenario == "cancel" {
					want = types.RunStateCanceled
				}
				if result.TerminalOutcome == nil || result.TerminalOutcome.State != want {
					t.Fatalf("outcome %#v err %v", result.TerminalOutcome, err)
				}
				if len(raw.events) != len(projected) || o.Status().Incomplete || o.Status().TerminalDelivered != 1 {
					t.Fatalf("source=%d observed=%d status=%#v", len(raw.events), len(projected), o.Status())
				}
				for i, e := range projected {
					source := raw.events[i]
					if e.NativeType != source.Type || e.Event.RunID != source.RunID || e.Iteration != source.Iteration || e.Event.StepID != source.CallID || e.Event.Sequence != 0 {
						t.Fatalf("correlation %d %#v %#v", i, source, e)
					}
					transcripts[scenario+map[bool]string{false: "/run", true: "/stream"}[stream]] = diagnosticsreplay.NativeExecutionTranscript{Events: projected, Observation: nativeEvidence(o.Status())}
				}
				if scenario == "tool" || scenario == "tool-error" {
					found := false
					for _, e := range projected {
						if e.NativeType == "tool.dispatch.started" {
							found = true
						}
					}
					if !found {
						t.Fatal("missing real tool event")
					}
					if len(result.ToolCalls) != 1 || result.ToolCalls[0].CallID != "native-call" {
						t.Fatal("source tool call correlation lost")
					}
				}
			})
		}
	}
	for _, scenario := range []string{"tool", "tool-error", "model-error", "cancel", "deadline"} {
		if err := diagnosticsreplay.CompareNativeExecutionParity(transcripts[scenario+"/run"], transcripts[scenario+"/stream"]); err != nil {
			t.Errorf("%s parity: %v", scenario, err)
		}
	}
}

func nativeEvidence(s eventadapter.Status) diagnosticsreplay.NativeExecutionDeliveryEvidence {
	return diagnosticsreplay.NativeExecutionDeliveryEvidence{State: s.State, Reason: s.Reason, Received: s.Received, Delivered: s.Delivered, Unconfirmed: s.Unconfirmed, Skipped: s.Skipped, TerminalDelivered: s.TerminalDelivered, Incomplete: s.Incomplete}
}

func TestNativeExecutionDynamicActionResume(t *testing.T) {
	transcripts := make(map[string]diagnosticsreplay.NativeExecutionTranscript)
	for _, stream := range []bool{false, true} {
		for _, decision := range []types.DynamicActionDecisionKind{types.DynamicActionDecisionConfirm, types.DynamicActionDecisionDeny, types.DynamicActionDecisionTimeout} {
			t.Run(string(decision)+map[bool]string{false: "/run", true: "/stream"}[stream], func(t *testing.T) {
				engine := nativeEngine(t, func(_ context.Context, req types.ModelRequest) (types.ModelResponse, error) {
					if len(req.ToolResult) > 0 {
						return types.ModelResponse{FinalAnswer: "confirmed"}, nil
					}
					return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "native-call", Name: "local.prepare"}}}, nil
				}, &fakes.Tool{NameValue: "prepare", InvokeFn: func(context.Context, map[string]any) (types.ToolResult, error) {
					return types.ToolResult{PendingAction: &types.DynamicActionReference{Token: "fake-token", Kind: "prepare", Resumable: true, RunID: "native-run", SessionID: "native-session", Iteration: 1, CallID: "native-call", Source: "test", Digest: "fake-digest"}}, nil
				}})
				var got []types.NativeExecutionEvent
				o := nativeObserver(t, types.ProtocolSourceRunner, "native-run", eventadapter.SinkFunc(func(_ context.Context, e types.NativeExecutionEvent) error { got = append(got, e); return nil }))
				paused, err := nativeExecute(context.Background(), engine, stream, o)
				if err != nil || paused.TerminalOutcome.State != types.RunStateInputRequired || !o.Status().Incomplete {
					t.Fatalf("pause %#v %v %#v", paused.TerminalOutcome, err, o.Status())
				}
				if got[len(got)-1].IsTerminal() {
					t.Fatal("pause classified terminal")
				}
				cp := paused.TerminalOutcome
				input := types.DynamicActionDecision{Decision: decision, Token: "fake-token", RunID: "native-run", SessionID: "native-session", CheckpointID: cp.CheckpointID, CheckpointVersion: cp.CheckpointVersion, CheckpointDigest: cp.CheckpointDigest, IdempotencyKey: "fake-idem"}
				result, err := engine.ResumeDynamicAction(context.Background(), input, o, stream)
				if err != nil {
					t.Fatal(err)
				}
				want := types.RunStateCanceled
				if decision == types.DynamicActionDecisionConfirm {
					want = types.RunStateCompleted
				}
				if result.TerminalOutcome.State != want {
					t.Fatalf("outcome %#v", result.TerminalOutcome)
				}
				resolved, canceled := 0, 0
				for _, e := range got {
					if e.NativeType == "run.dynamic_action.resolved" {
						resolved++
						if e.IsTerminal() || e.Event.Payload["decision"] != string(decision) || e.Event.Payload["dynamic_action_resume_admission"] != "accepted" {
							t.Fatal("resolution changed")
						}
					}
					if e.IsTerminal() && e.SourceState() == "canceled" {
						canceled++
					}
				}
				if resolved != 1 || o.Status().TerminalDelivered != 1 || o.Status().Incomplete || decision == types.DynamicActionDecisionConfirm && canceled != 0 || decision != types.DynamicActionDecisionConfirm && canceled != 1 {
					t.Fatalf("resolved=%d canceled=%d %#v", resolved, canceled, o.Status())
				}
				before := len(got)
				if _, err := engine.ResumeDynamicAction(context.Background(), input, o, stream); err != nil || len(got) != before {
					t.Fatalf("duplicate emitted %d events: %v", len(got)-before, err)
				}
				transcripts[string(decision)+map[bool]string{false: "/run", true: "/stream"}[stream]] = diagnosticsreplay.NativeExecutionTranscript{Events: got, Observation: nativeEvidence(o.Status())}
			})
		}
	}
	for _, decision := range []string{"confirm", "deny", "timeout"} {
		if err := diagnosticsreplay.CompareNativeExecutionParity(transcripts[decision+"/run"], transcripts[decision+"/stream"]); err != nil {
			t.Errorf("%s parity: %v", decision, err)
		}
	}
}

func TestNativeExecutionObservationFailureDoesNotChangeSource(t *testing.T) {
	for _, stream := range []bool{false, true} {
		for _, mode := range []string{"sink", "projection", "disconnect"} {
			t.Run(mode+map[bool]string{false: "/run", true: "/stream"}[stream], func(t *testing.T) {
				engine := nativeEngine(t, func(context.Context, types.ModelRequest) (types.ModelResponse, error) {
					return types.ModelResponse{FinalAnswer: "done"}, nil
				}, nil)
				mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX_NATIVE_OBSERVATION_TEST"})
				if err != nil {
					t.Fatal(err)
				}
				defer func() { _ = mgr.Close() }()
				var o *eventadapter.Observer
				opts := eventadapter.Options{Projection: types.NativeExecutionOptions{Source: types.ProtocolSourceRunner, ExpectedRunID: "native-run"}, Sink: eventadapter.SinkFunc(func(_ context.Context, e types.NativeExecutionEvent) error {
					if e.NativeType == "run.finished" {
						return errors.New("fake write then error")
					}
					if mode == "disconnect" {
						o.Close()
					}
					return nil
				})}
				if mode == "projection" {
					limits := types.DefaultNativeExecutionLimits()
					limits.EventBytes = 1
					opts.Projection.Limits = &limits
				}
				o, err = eventadapter.New(opts)
				if err != nil {
					t.Fatal(err)
				}
				defer o.Close()
				raw := &nativeRawCollector{}
				h := nativeFanout{event.NewDispatcher(raw, event.NewRuntimeRecorder(mgr), o)}
				result, err := nativeExecute(context.Background(), engine, stream, h)
				if err != nil || result.TerminalOutcome.State != types.RunStateCompleted {
					t.Fatalf("observer controlled source: %#v %v", result, err)
				}
				if !o.Status().Incomplete || o.Status().TerminalDelivered != 0 || len(raw.events) < 4 {
					t.Fatalf("status=%#v raw=%d", o.Status(), len(raw.events))
				}
				runs := mgr.RecentRuns(5)
				if len(runs) != 1 || runs[0].RunID != "native-run" {
					t.Fatalf("recorder fan-out %#v", runs)
				}
			})
		}
	}
}

func TestNativeExecutionComposerChildSources(t *testing.T) {
	var all []types.NativeExecutionEvent
	sink := eventadapter.SinkFunc(func(_ context.Context, e types.NativeExecutionEvent) error { all = append(all, e); return nil })
	parent := nativeObserver(t, types.ProtocolSourceComposer, "", sink)
	child := nativeObserver(t, types.ProtocolSourceRunner, "child-run", sink)
	model := fakes.NewModel(nil)
	comp, err := composer.NewBuilder(model).WithEventHandler(parent).Build()
	if err != nil {
		t.Fatal(err)
	}
	result, err := comp.DispatchChild(context.Background(), composer.ChildDispatchRequest{Task: scheduler.Task{TaskID: "child-task", RunID: "child-run", ParentRunID: "parent-run", StepID: "parent-step"}, Target: composer.ChildTargetLocal, ParentRemainingBudget: time.Second, ChildTimeout: time.Second, LocalRunner: composer.LocalChildRunnerFunc(func(ctx context.Context, _ scheduler.Task) (map[string]any, error) {
		engine := nativeEngine(t, func(context.Context, types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{FinalAnswer: "child"}, nil
		}, nil)
		r, err := engine.Run(ctx, types.RunRequest{RunID: "child-run", Input: "child"}, child)
		return map[string]any{"answer": r.FinalAnswer}, err
	})})
	if err != nil || result.Commit.Status != scheduler.TaskStateSucceeded {
		t.Fatalf("child %#v %v", result, err)
	}
	before, after, childSeen, referenceSeen := false, false, false, false
	for _, e := range all {
		if e.Event.Sequence != 0 || e.Event.RunID != "child-run" {
			t.Fatalf("child correlation %#v", e)
		}
		if e.Event.Source == types.ProtocolSourceRunner {
			childSeen = true
		} else {
			if childSeen {
				after = true
			} else {
				before = true
			}
			if e.Event.Payload["parent_run_id"] == "parent-run" {
				referenceSeen = true
			}
		}
	}
	if !before || !after || !childSeen || !referenceSeen || parent.Status().Reason != "" || child.Status().Incomplete {
		t.Fatalf("timeline before=%v after=%v child=%v parentRef=%v %#v", before, after, childSeen, referenceSeen, all)
	}
}
