package integration

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/host/eventadapter"
	"github.com/FelixSeptem/baymax/integration/fakes"
	"github.com/FelixSeptem/baymax/orchestration/composer"
	"github.com/FelixSeptem/baymax/orchestration/planbridge"
	"github.com/FelixSeptem/baymax/orchestration/scheduler"
	"github.com/FelixSeptem/baymax/orchestration/teams"
	"github.com/FelixSeptem/baymax/orchestration/workflow"
	"github.com/FelixSeptem/baymax/tool/diagnosticsreplay"
)

func bridgeDescriptor(shape planbridge.Shape) planbridge.Descriptor {
	return planbridge.Descriptor{SchemaVersion: planbridge.SchemaVersion, PlanID: "plan", PlanVersion: 1, Shape: shape, Delivery: planbridge.DeliveryInline, Trigger: planbridge.TriggerManual, AuthorityDigest: "authority", ToolCapabilityDigest: "tools", InputRef: "input", InputDigest: "input-digest", HandlerRef: "handler", HandlerDigest: "handler-digest"}
}
func bridgeBinding() planbridge.Binding {
	return planbridge.Binding{SemanticDigest: "semantics", Admit: func(context.Context, planbridge.Identity) (planbridge.Admission, error) {
		return planbridge.Admission{AuthorityDigest: "authority", ToolCapabilityDigest: "tools", SemanticDigest: "semantics", ParentRemainingBudget: time.Second}, nil
	}}
}
func bridgeCompile(t *testing.T, shape planbridge.Shape, b planbridge.Binding) *planbridge.Compiled {
	t.Helper()
	d, e := planbridge.Describe(bridgeDescriptor(shape), b, planbridge.Limits{})
	if e != nil {
		t.Fatal(e)
	}
	p, e := planbridge.Compile(d, b, planbridge.Limits{})
	if e != nil {
		t.Fatal(e)
	}
	return p
}
func bridgeExecute(t *testing.T, p *planbridge.Compiled, stream bool, x planbridge.Execution) (planbridge.Result, error) {
	t.Helper()
	if stream {
		return p.Stream(context.Background(), x)
	}
	return p.Run(context.Background(), x)
}

func TestHostExecutionPlanRealDirectParityAndPause(t *testing.T) {
	for _, pending := range []bool{false, true} {
		transcripts := make([]diagnosticsreplay.NativeExecutionTranscript, 0, 2)
		for _, stream := range []bool{false, true} {
			var models, tools atomic.Int32
			engine := nativeEngine(t, func(_ context.Context, r types.ModelRequest) (types.ModelResponse, error) {
				models.Add(1)
				if len(r.ToolResult) > 0 {
					return types.ModelResponse{FinalAnswer: "done"}, nil
				}
				return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "native-call", Name: "local.prepare"}}}, nil
			}, &fakes.Tool{NameValue: "prepare", InvokeFn: func(context.Context, map[string]any) (types.ToolResult, error) {
				tools.Add(1)
				if pending {
					return types.ToolResult{PendingAction: &types.DynamicActionReference{Token: "token", Kind: "prepare", RunID: "native-run", SessionID: "native-session", CallID: "native-call", Iteration: 1, Source: "test", Digest: "digest", Resumable: true}}, nil
				}
				return types.ToolResult{Content: "result"}, nil
			}})
			b := bridgeBinding()
			b.Direct = &planbridge.DirectBinding{Runner: engine, Request: types.RunRequest{RunID: "native-run", SessionID: "native-session", Input: "fake", Capabilities: types.CapabilityRequirements{Required: []types.ModelCapability{types.ModelCapabilityStreaming}}}}
			p := bridgeCompile(t, planbridge.ShapeDirect, b)
			if models.Load() != 0 || tools.Load() != 0 {
				t.Fatal("compile activated source")
			}
			raw := &nativeRawCollector{}
			var events []types.NativeExecutionEvent
			out, err := bridgeExecute(t, p, stream, planbridge.Execution{Handler: raw, Sink: eventadapter.SinkFunc(func(_ context.Context, e types.NativeExecutionEvent) error { events = append(events, e); return nil })})
			if err != nil || out.Direct == nil || tools.Load() != 1 {
				t.Fatalf("direct %+v %v", out, err)
			}
			want := types.RunStateCompleted
			if pending {
				want = types.RunStateInputRequired
			}
			if out.Direct.TerminalOutcome.State != want || len(raw.events) != len(events) || len(events) == 0 {
				t.Fatal("source outcome/events lost")
			}
			for i, e := range events {
				if e.NativeType != raw.events[i].Type || e.Event.RunID != raw.events[i].RunID || e.HostReferences.PlanDigest != p.Identity().Digest || e.HostReferences.PlanVersion != "1" {
					t.Fatalf("correlation %+v", e)
				}
			}
			if pending {
				if events[len(events)-1].IsTerminal() || out.Direct.TerminalOutcome.CheckpointID == "" || models.Load() != 1 {
					t.Fatal("pause converted or continued")
				}
				_, err = planbridge.TerminalOnly(*out.Direct, nil)
				var continuation *planbridge.ContinuationError
				if !errors.As(err, &continuation) || continuation.Result.TerminalOutcome.CheckpointID != out.Direct.TerminalOutcome.CheckpointID {
					t.Fatal("checkpoint lost")
				}
			}
			transcripts = append(transcripts, diagnosticsreplay.NativeExecutionTranscript{Events: events, Observation: nativeEvidence(*out.Observation)})
		}
		if err := diagnosticsreplay.CompareNativeExecutionParity(transcripts[0], transcripts[1]); err != nil {
			t.Fatal(err)
		}
	}
}

func TestHostExecutionPlanNativeTeamsAndWorkflowParity(t *testing.T) {
	for _, strategy := range []teams.Strategy{teams.StrategyParallel, teams.StrategyVote} {
		results := make([]teams.Result, 0, 2)
		for _, stream := range []bool{false, true} {
			b := bridgeBinding()
			b.Team = &planbridge.TeamBinding{TerminalOnly: true, Plan: teams.Plan{RunID: "run", TeamID: "team", Strategy: strategy, Tasks: []teams.Task{{TaskID: "a", AgentID: "worker"}, {TaskID: "b", AgentID: "worker"}}}}
			for i := range b.Team.Plan.Tasks {
				b.Team.Plan.Tasks[i].Runner = teams.TaskRunnerFunc(func(ctx context.Context, task teams.Task) (teams.TaskResult, error) {
					e := nativeEngine(t, func(context.Context, types.ModelRequest) (types.ModelResponse, error) {
						return types.ModelResponse{FinalAnswer: "answer"}, nil
					}, nil)
					r, err := planbridge.TerminalOnly(e.Run(ctx, types.RunRequest{RunID: task.TaskID, Input: "fake"}, nil))
					return teams.TaskResult{Output: r.FinalAnswer, Vote: "yes"}, err
				})
			}
			p := bridgeCompile(t, planbridge.ShapeTeam, b)
			var mu sync.Mutex
			events := 0
			out, err := bridgeExecute(t, p, stream, planbridge.Execution{Sink: eventadapter.SinkFunc(func(_ context.Context, e types.NativeExecutionEvent) error {
				mu.Lock()
				defer mu.Unlock()
				events++
				if e.HostReferences.PlanDigest != p.Identity().Digest {
					t.Error("host identity lost")
				}
				return nil
			})})
			if err != nil || events == 0 || out.Team.TeamTaskFailed != 0 {
				t.Fatalf("teams %+v %v", out, err)
			}
			native, err := teams.New().Run(context.Background(), b.Team.Plan)
			if err != nil || !reflect.DeepEqual(native, *out.Team) {
				t.Fatalf("native team drift: %v", err)
			}
			results = append(results, *out.Team)
		}
		if !reflect.DeepEqual(results[0], results[1]) {
			t.Fatal("team Run/Stream drift")
		}
	}
	results := make([]workflow.RunResult, 0, 2)
	for _, stream := range []bool{false, true} {
		var attempts atomic.Int32
		b := bridgeBinding()
		b.Workflow = &planbridge.WorkflowBinding{TerminalOnly: true, Request: workflow.RunRequest{RunID: "run", DSL: workflow.Definition{WorkflowID: "workflow", Steps: []workflow.Step{{StepID: "a", Retry: workflow.Retry{MaxAttempts: 2}}, {StepID: "b", DependsOn: []string{"a"}}}}}, Adapter: workflow.DispatchAdapter{Runner: func(ctx context.Context, _ string, s workflow.Step, attempt int) (workflow.StepOutput, error) {
			attempts.Add(1)
			if s.StepID == "a" && attempt == 1 {
				return workflow.StepOutput{}, errors.New("source retry")
			}
			e := nativeEngine(t, func(context.Context, types.ModelRequest) (types.ModelResponse, error) {
				return types.ModelResponse{FinalAnswer: "done"}, nil
			}, nil)
			r, err := planbridge.TerminalOnly(e.Run(ctx, types.RunRequest{RunID: s.StepID, Input: "fake"}, nil))
			return workflow.StepOutput{Payload: map[string]any{"answer": r.FinalAnswer}}, err
		}}}
		p := bridgeCompile(t, planbridge.ShapeWorkflow, b)
		raw := &nativeRawCollector{}
		out, err := bridgeExecute(t, p, stream, planbridge.Execution{Handler: raw})
		if err != nil || out.Workflow.WorkflowStepFailed != 0 || attempts.Load() != 3 || len(raw.events) == 0 {
			t.Fatalf("workflow %+v %v attempts=%d", out, err, attempts.Load())
		}
		native, err := workflow.New(workflow.WithStepAdapter(b.Workflow.Adapter)).Run(context.Background(), b.Workflow.Request)
		if err != nil || !reflect.DeepEqual(native, *out.Workflow) {
			t.Fatalf("native workflow drift: %v", err)
		}
		results = append(results, *out.Workflow)
	}
	if !reflect.DeepEqual(results[0], results[1]) {
		t.Fatal("workflow Run/Stream drift")
	}
}

func TestHostExecutionPlanChildNativeFailureAndIsolation(t *testing.T) {
	for _, failure := range []error{nil, errors.New("child failed"), context.Canceled, context.DeadlineExceeded} {
		for _, stream := range []bool{false, true} {
			var calls atomic.Int32
			callback := composer.LocalChildRunnerFunc(func(context.Context, scheduler.Task) (map[string]any, error) {
				calls.Add(1)
				return map[string]any{"answer": "child"}, failure
			})
			req := composer.ChildDispatchRequest{Target: composer.ChildTargetLocal, Task: scheduler.Task{TaskID: "child", RunID: "child-run", ParentRunID: "parent", StepID: "parent-step", MaxAttempts: 1}, ParentRemainingBudget: time.Second, ChildTimeout: time.Second, LocalRunner: callback}
			b := bridgeBinding()
			b.Child = &planbridge.ChildBinding{TerminalOnly: true, Request: req}
			p := bridgeCompile(t, planbridge.ShapeSubagent, b)
			raw := &nativeRawCollector{}
			out, err := bridgeExecute(t, p, stream, planbridge.Execution{Handler: raw})
			if !errors.Is(err, failure) || calls.Load() != 1 || out.Child == nil || len(raw.events) == 0 {
				t.Fatalf("child %+v %v", out, err)
			}
			nativeOwner, err := composer.New(nil, composer.WithRunner(nativeEngine(t, func(context.Context, types.ModelRequest) (types.ModelResponse, error) {
				return types.ModelResponse{FinalAnswer: "unused"}, nil
			}, nil)))
			if err != nil {
				t.Fatal(err)
			}
			native, nativeErr := nativeOwner.DispatchChild(context.Background(), req)
			if !errors.Is(nativeErr, failure) || native.Commit.Status != out.Child.Commit.Status || native.CommitMeta.Record.State != out.Child.CommitMeta.Record.State || native.Record.Task.TimeoutResolution.ResolvedTimeout != out.Child.Record.Task.TimeoutResolution.ResolvedTimeout || out.Child.Claimed.Record.Task.ParentRunID != "parent" || out.Child.Commit.AttemptID != out.Child.Claimed.Attempt.AttemptID {
				t.Fatalf("native child drift: %+v %+v", native, out.Child)
			}
		}
	}
	// Existing shared queue is independent and remains unchanged across bridge execution.
	shared, err := scheduler.New(scheduler.NewMemoryStore())
	if err != nil {
		t.Fatal(err)
	}
	before, err := shared.Enqueue(context.Background(), scheduler.Task{TaskID: "unrelated", RunID: "other"})
	if err != nil {
		t.Fatal(err)
	}
	b := bridgeBinding()
	b.Child = &planbridge.ChildBinding{TerminalOnly: true, Request: composer.ChildDispatchRequest{Target: composer.ChildTargetLocal, Task: scheduler.Task{TaskID: "bridge-child", RunID: "child", ParentRunID: "parent"}, ParentRemainingBudget: time.Second, LocalRunner: composer.LocalChildRunnerFunc(func(context.Context, scheduler.Task) (map[string]any, error) { return nil, nil })}}
	p := bridgeCompile(t, planbridge.ShapeSubagent, b)
	if _, err = p.Run(context.Background(), planbridge.Execution{}); err != nil {
		t.Fatal(err)
	}
	after, ok, err := shared.Get(context.Background(), "unrelated")
	if err != nil || !ok || !reflect.DeepEqual(before, after) {
		t.Fatalf("shared queue mutated: %v", err)
	}
}

func TestHostExecutionPlanObserverFailureAndRejection(t *testing.T) {
	engine := nativeEngine(t, func(context.Context, types.ModelRequest) (types.ModelResponse, error) {
		return types.ModelResponse{FinalAnswer: "done"}, nil
	}, nil)
	b := bridgeBinding()
	b.Direct = &planbridge.DirectBinding{Runner: engine, Request: types.RunRequest{RunID: "run", Input: "fake"}}
	p := bridgeCompile(t, planbridge.ShapeDirect, b)
	out, err := p.Run(context.Background(), planbridge.Execution{Sink: eventadapter.SinkFunc(func(context.Context, types.NativeExecutionEvent) error { return errors.New("observer failed") })})
	if err != nil || out.Direct.TerminalOutcome.State != types.RunStateCompleted || !out.Observation.Incomplete {
		t.Fatal("observer controls source")
	}
	d, err := planbridge.Describe(bridgeDescriptor(planbridge.ShapeDirect), b, planbridge.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	for _, axis := range []string{"mailbox", "resume", "pause"} {
		next := d
		switch axis {
		case "mailbox":
			next.Delivery = planbridge.DeliveryMailbox
		case "resume":
			next.Trigger = planbridge.TriggerResumed
		case "pause":
			next.RequiredCapabilities = []string{"durable_continuation"}
		}
		_, err = planbridge.Compile(next, b, planbridge.Limits{})
		var pe *planbridge.Error
		if !errors.As(err, &pe) || pe.Code != planbridge.UnsupportedCapability {
			t.Fatalf("unsupported accepted: %v", err)
		}
	}
}
