package planbridge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/orchestration/composer"
	"github.com/FelixSeptem/baymax/orchestration/scheduler"
	"github.com/FelixSeptem/baymax/orchestration/teams"
	"github.com/FelixSeptem/baymax/orchestration/workflow"
)

type unsafeString string

func (unsafeString) MarshalJSON() ([]byte, error) { panic("arbitrary marshaler invoked") }
func TestHostExecutionPlanSnapshotsRejectUnsafeBeforeCopy(t *testing.T) {
	cycle := map[string]any{}
	cycle["cycle"] = cycle
	var deep any = "leaf"
	for i := 0; i < 9; i++ {
		deep = []any{deep}
	}
	cases := []any{cycle, deep, make(chan int), new(int), unsafeString("secret"), map[string]any{"huge": strings.Repeat("secret", 12000)}}
	for i, value := range cases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			d, b, _ := directFixture(t)
			d.Shape = ShapeWorkflow
			b.Direct = nil
			b.Workflow = &WorkflowBinding{TerminalOnly: true, Request: workflow.RunRequest{RunID: "run", DSL: workflow.Definition{WorkflowID: "workflow", Steps: []workflow.Step{{StepID: "step", Payload: map[string]any{"input": value}}}}}, Adapter: workflow.DispatchAdapter{Runner: func(context.Context, string, workflow.Step, int) (workflow.StepOutput, error) {
				t.Fatal("activated")
				return workflow.StepOutput{}, nil
			}}}
			_, err := Describe(d, b, Limits{})
			if err == nil {
				t.Fatal("unsafe accepted")
			}
			if strings.Contains(err.Error(), "secret") {
				t.Fatal("payload leaked")
			}
		})
	}
}
func TestHostExecutionPlanNativePreflightWholePlan(t *testing.T) {
	d, b, r := directFixture(t)
	d.Shape = ShapeTeam
	b.Direct = nil
	task := teams.Task{TaskID: "a", AgentID: "worker", Runner: teams.TaskRunnerFunc(func(context.Context, teams.Task) (teams.TaskResult, error) { r.calls++; return teams.TaskResult{}, nil })}
	b.Team = &TeamBinding{TerminalOnly: true, Plan: teams.Plan{RunID: "run", TeamID: "team", Strategy: teams.StrategyParallel, Tasks: []teams.Task{task, {TaskID: "bad", AgentID: "worker"}}}}
	_, e := Describe(d, b, Limits{})
	codeOf(t, e, InvalidPlan)
	if r.calls != 0 {
		t.Fatal("earlier task activated")
	}
	b.Team.Plan.Tasks = []teams.Task{task}
	d, e = Describe(d, b, Limits{})
	if e != nil {
		t.Fatal(e)
	}
	native, e := teams.NormalizePlan(b.Team.Plan)
	if e != nil {
		t.Fatal(e)
	}
	if d.SourceFacts["task_timeout"] != json.Number(fmt.Sprint(int64(native.TaskTimeout))) {
		t.Fatal("default drift")
	}
	p, e := Compile(d, b, Limits{})
	if e != nil {
		t.Fatal(e)
	}
	b.Team.Plan.Tasks[0].TaskID = "mutated"
	out, e := p.Run(context.Background(), Execution{})
	if e != nil || out.Team.Tasks[0].TaskID != "a" {
		t.Fatalf("mutation leaked: %+v %v", out, e)
	}
	d, b, r = directFixture(t)
	d.Shape = ShapeWorkflow
	b.Direct = nil
	b.Workflow = &WorkflowBinding{TerminalOnly: true, Request: workflow.RunRequest{RunID: "run", DSL: workflow.Definition{WorkflowID: "w", Steps: []workflow.Step{{StepID: "a"}, {StepID: "b", DependsOn: []string{"missing"}}}}}, Adapter: workflow.DispatchAdapter{Runner: func(context.Context, string, workflow.Step, int) (workflow.StepOutput, error) {
		r.calls++
		return workflow.StepOutput{}, nil
	}}}
	_, e = Describe(d, b, Limits{})
	codeOf(t, e, InvalidPlan)
	var nativeErrors workflow.ValidationErrors
	if !errors.As(e, &nativeErrors) {
		t.Fatal("native errors not retained")
	}
	if r.calls != 0 {
		t.Fatal("invalid later step executed")
	}
	b.Workflow.Request.DSL.Steps[0].DependsOn = []string{"b"}
	b.Workflow.Request.DSL.Steps[1].DependsOn = []string{"a"}
	_, e = Describe(d, b, Limits{})
	codeOf(t, e, InvalidPlan)
}
func TestHostExecutionPlanChildLiveBudgetIsolation(t *testing.T) {
	d, b, r := directFixture(t)
	d.Shape = ShapeSubagent
	b.Direct = nil
	b.Child = &ChildBinding{TerminalOnly: true, Request: composer.ChildDispatchRequest{Target: composer.ChildTargetLocal, Task: scheduler.Task{TaskID: "child", RunID: "run", ParentRunID: "parent"}, ParentRemainingBudget: time.Second, ChildTimeout: time.Second, LocalRunner: composer.LocalChildRunnerFunc(func(_ context.Context, task scheduler.Task) (map[string]any, error) {
		r.calls++
		return map[string]any{"timeout": int64(task.TimeoutResolution.ResolvedTimeout)}, nil
	})}}
	remaining := time.Second / 2
	b.Admit = func(context.Context, Identity) (Admission, error) {
		return Admission{AuthorityDigest: "authority", ToolCapabilityDigest: "tools", SemanticDigest: "semantics", ParentRemainingBudget: remaining}, nil
	}
	d, e := Describe(d, b, Limits{})
	if e != nil {
		t.Fatal(e)
	}
	p, e := Compile(d, b, Limits{})
	if e != nil {
		t.Fatal(e)
	}
	for _, stream := range []bool{false, true} {
		var out Result
		if stream {
			out, e = p.Stream(context.Background(), Execution{})
		} else {
			out, e = p.Run(context.Background(), Execution{})
		}
		if e != nil {
			t.Fatal(e)
		}
		if out.Child.CommitMeta.Record.State != scheduler.TaskStateSucceeded || !out.Child.Record.Task.TimeoutResolution.ParentBudgetClamped || out.Child.Record.Task.TimeoutResolution.ResolvedTimeout != remaining || out.Child.Claimed.Record.Task.ParentRunID != "parent" {
			t.Fatalf("native clamp/claim: %+v", out.Child)
		}
	}
	remaining = 0
	_, e = p.Run(context.Background(), Execution{})
	var budget *scheduler.BudgetError
	if !errors.As(e, &budget) || budget.Code != scheduler.BudgetRejectParentBudgetExhausted || r.calls != 2 {
		t.Fatal("exhausted parent activated")
	}
	b.Child.TerminalOnly = false
	_, e = Compile(d, b, Limits{})
	codeOf(t, e, UnsupportedCapability)
	d.RequiredCapabilities = []string{"token_streaming"}
	_, e = Compile(d, b, Limits{})
	codeOf(t, e, UnsupportedCapability)
	// There is no queue/scheduler/composer field through which a host can assert
	// isolation of a shared service. Compile-time API isolation is deliberate.
	typ := reflect.TypeOf(ChildBinding{})
	for _, field := range []string{"Scheduler", "Store", "Composer", "Isolated"} {
		if _, ok := typ.FieldByName(field); ok {
			t.Fatalf("shared scheduler escape: %s", field)
		}
	}
}

func TestHostExecutionPlanChildContinuationDelegatesToSourceOwner(t *testing.T) {
	d, b, _ := directFixture(t)
	d.Shape = ShapeSubagent
	b.Direct = nil
	continuation := composer.ChildResumeRequest{Ref: composer.ChildContinuationRef{SchemaVersion: composer.ChildContinuationVersion, CheckpointID: "checkpoint", Digest: "digest", OwnerNamespace: "child", GraphID: "graph", GraphVersion: "v1", RunID: "run"}, GraphDigest: "graph-digest", Generation: 1, Target: composer.ChildTargetLocal, IdempotencyKey: "resume"}
	owner := &testChildContinuationOwner{result: composer.ChildDispatchResult{AsyncAccepted: true, AsyncTaskID: "native"}}
	source := evidencedContinuationSource()
	source.caps.SourceNamespace = "child"
	source.caps.Deployment = types.SameHostSingleWriterScope("child", "contract:deployment")
	manifest, err := CapabilityManifestWithChildContinuation(context.Background(), source)
	if err != nil {
		t.Fatalf("child manifest error = %v", err)
	}
	reader, err := NewCapabilityManifestReader(manifest)
	if err != nil {
		t.Fatalf("child manifest reader error = %v", err)
	}
	b.Child = &ChildBinding{TerminalOnly: false, Resume: &continuation, ContinuationOwner: owner, Manifest: reader}
	d, err = Describe(d, b, Limits{})
	if err != nil {
		t.Fatalf("Describe() error = %v", err)
	}
	p, err := Compile(d, b, Limits{})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	out, err := p.Run(context.Background(), Execution{})
	if err != nil || out.Child == nil || out.Child.AsyncTaskID != "native" || owner.calls != 1 {
		t.Fatalf("resume result = %#v, err=%v, calls=%d", out.Child, err, owner.calls)
	}
}

func TestHostExecutionPlanWorkflowContinuationBindingRequiresOwnerManifestAndStore(t *testing.T) {
	d, b, _ := directFixture(t)
	d.Shape = ShapeWorkflow
	b.Direct = nil
	b.Workflow = &WorkflowBinding{
		TerminalOnly: false,
		Request:      workflow.RunRequest{RunID: "run", DSL: workflow.Definition{WorkflowID: "wf", Steps: []workflow.Step{{StepID: "step", Kind: workflow.StepKindRunner}}}},
		Adapter: workflow.DispatchAdapter{Runner: func(context.Context, string, workflow.Step, int) (workflow.StepOutput, error) {
			t.Fatal("resume binding must not dispatch as a fresh Run")
			return workflow.StepOutput{}, nil
		}},
		Resume: &workflow.WorkflowResumeRequest{Ref: workflow.WorkflowContinuationRef{SchemaVersion: workflow.WorkflowContinuationVersion, CheckpointID: "checkpoint", Digest: "digest", OwnerNamespace: "workflow"}, GraphDigest: "graph", Generation: 1, IdempotencyKey: "resume"},
	}
	_, err := Describe(d, b, Limits{})
	codeOf(t, err, UnsupportedCapability)
}

func TestHostExecutionPlanWorkflowContinuationRestoreDelegatesToReopenedOwnerForRunAndStream(t *testing.T) {
	root := t.TempDir()
	store, err := workflow.NewFileWorkflowContinuationStore(root, "workflow")
	if err != nil {
		t.Fatal(err)
	}
	firstStepCalls := 0
	definition := workflow.Definition{WorkflowID: "workflow-bridge-restore", Steps: []workflow.Step{
		{StepID: "completed", Kind: workflow.StepKindRunner},
		{StepID: "pending", Kind: workflow.StepKindRunner, DependsOn: []string{"completed"}},
	}}
	first := workflow.New(
		workflow.WithWorkflowContinuationStore(store, "workflow"),
		workflow.WithStepAdapter(workflow.DispatchAdapter{Runner: func(_ context.Context, _ string, step workflow.Step, _ int) (workflow.StepOutput, error) {
			if step.StepID == "completed" {
				firstStepCalls++
				return workflow.StepOutput{}, nil
			}
			return workflow.StepOutput{}, workflow.PauseError{Reason: workflow.PauseInputRequired, PendingActionReference: "pending-input"}
		}}),
	)
	paused, err := first.Run(context.Background(), workflow.RunRequest{RunID: "workflow-run", SessionID: "session", AuthorityDigest: "authority", DSL: definition})
	if !errors.Is(err, workflow.ErrWorkflowPaused) || paused.WorkflowCheckpoint == nil {
		t.Fatalf("workflow did not pause with native checkpoint: result=%#v err=%v", paused, err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := workflow.NewFileWorkflowContinuationStore(root, "workflow")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	resume := workflow.WorkflowResumeRequest{Ref: *paused.WorkflowCheckpoint, GraphDigest: paused.WorkflowGraphDigest, Generation: paused.WorkflowCheckpointGeneration, IdempotencyKey: "bridge-resume", AuthorityDigest: "authority"}
	newEngine := workflow.New(
		workflow.WithWorkflowContinuationStore(reopened, "workflow"),
		workflow.WithStepAdapter(workflow.DispatchAdapter{Runner: func(_ context.Context, _ string, step workflow.Step, _ int) (workflow.StepOutput, error) {
			if step.StepID == "completed" {
				t.Fatal("completed step was rerun during restore")
			}
			return workflow.StepOutput{}, nil
		}}),
	)
	manifest, err := CapabilityManifestWithWorkflowContinuation(context.Background(), newEngine)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := NewCapabilityManifestReader(manifest)
	if err != nil {
		t.Fatal(err)
	}
	d, b, _ := directFixture(t)
	d.Shape = ShapeWorkflow
	b.Direct = nil
	b.Workflow = &WorkflowBinding{TerminalOnly: false, Request: workflow.RunRequest{RunID: "workflow-run", SessionID: "session", AuthorityDigest: "authority", DSL: definition}, Adapter: workflow.DispatchAdapter{Runner: func(_ context.Context, _ string, step workflow.Step, _ int) (workflow.StepOutput, error) {
		if step.StepID == "completed" {
			t.Fatal("completed step was rerun through the bridge")
		}
		return workflow.StepOutput{}, nil
	}}, Continuations: reopened, Resume: &resume, Manifest: reader}
	foreignBinding := b
	foreignWorkflow := *b.Workflow
	foreignResume := resume
	foreignResume.AuthorityDigest = "foreign-authority"
	foreignWorkflow.Resume = &foreignResume
	foreignBinding.Workflow = &foreignWorkflow
	if _, err := Describe(d, foreignBinding, Limits{}); err == nil {
		t.Fatal("foreign authority restore binding was accepted")
	} else {
		codeOf(t, err, BindingMismatch)
	}
	d, err = Describe(d, b, Limits{})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := Compile(d, b, Limits{})
	if err != nil {
		t.Fatal(err)
	}
	streamEvents := 0
	out, err := plan.Stream(context.Background(), Execution{WorkflowEvent: func(workflow.StreamEvent) error { streamEvents++; return nil }})
	if err != nil || out.Workflow == nil || out.Workflow.WorkflowStatus != "succeeded" || firstStepCalls != 1 || streamEvents == 0 {
		t.Fatalf("bridged restore result=%#v err=%v first-calls=%d stream-events=%d", out.Workflow, err, firstStepCalls, streamEvents)
	}
	duplicate, err := plan.Run(context.Background(), Execution{})
	if err != nil || duplicate.Workflow == nil || duplicate.Workflow.WorkflowStatus != out.Workflow.WorkflowStatus || firstStepCalls != 1 {
		t.Fatalf("duplicate did not return owner projection: result=%#v err=%v first-calls=%d", duplicate.Workflow, err, firstStepCalls)
	}
}

type testChildContinuationOwner struct {
	result composer.ChildDispatchResult
	calls  int
}

func (o *testChildContinuationOwner) ResumeChildContinuation(context.Context, composer.ChildResumeRequest) (composer.ChildDispatchResult, error) {
	o.calls++
	return o.result, nil
}
func TestHostExecutionPlanExactLimits(t *testing.T) {
	d, b, _ := directFixture(t)
	p, e := Compile(d, b, Limits{})
	if e != nil {
		t.Fatal(e)
	}
	size := len(p.Canonical())
	if _, e = Compile(d, b, Limits{DescriptorBytes: size}); e != nil {
		t.Fatal(e)
	}
	_, e = Compile(d, b, Limits{DescriptorBytes: size - 1})
	codeOf(t, e, LimitExceeded)
	for _, n := range []int{128, 129} {
		d, b, _ = directFixture(t)
		d.Shape = ShapeTeam
		b.Direct = nil
		tasks := make([]teams.Task, n)
		for i := range tasks {
			tasks[i] = teams.Task{TaskID: fmt.Sprint(i), AgentID: "a", Runner: teams.TaskRunnerFunc(func(context.Context, teams.Task) (teams.TaskResult, error) { return teams.TaskResult{}, nil })}
		}
		b.Team = &TeamBinding{TerminalOnly: true, Plan: teams.Plan{RunID: "run", TeamID: "team", Strategy: teams.StrategyParallel, Tasks: tasks}}
		_, e = Describe(d, b, Limits{})
		if n == 128 && e != nil {
			t.Fatal(e)
		}
		if n == 129 {
			codeOf(t, e, LimitExceeded)
		}
	}
	for _, n := range []int{8, 9} {
		var value any = "v"
		for i := 0; i < n; i++ {
			value = []any{value}
		}
		_, e = snapshot(value, 65536, 8)
		if n == 8 && e != nil {
			t.Fatal(e)
		}
		if n == 9 {
			codeOf(t, e, LimitExceeded)
		}
	}
	// Exact encoded binding budget, including escaping and punctuation.
	req := types.RunRequest{RunID: "r", Input: strings.Repeat("x", 200)}
	raw, _ := json.Marshal(req)
	if _, e = snapshot(req, len(raw), 8); e != nil {
		t.Fatal(e)
	}
	_, e = snapshot(req, len(raw)-1, 8)
	codeOf(t, e, LimitExceeded)
}
func TestHostExecutionPlanCanonicalFacts(t *testing.T) {
	d, b, _ := directFixture(t)
	p, e := Compile(d, b, Limits{})
	if e != nil {
		t.Fatal(e)
	}
	original := p.Identity().Digest
	swapped := map[string]any{}
	for k, v := range d.SourceFacts {
		swapped[k] = v
	}
	d.SourceFacts = swapped
	q, e := Compile(d, b, Limits{})
	if e != nil || string(p.Canonical()) != string(q.Canonical()) {
		t.Fatal("map order drift")
	}
	for _, mutate := range []func(*Descriptor){func(d *Descriptor) { d.PlanVersion++ }, func(d *Descriptor) { d.AuthorityDigest = "new" }, func(d *Descriptor) { d.ToolCapabilityDigest = "new" }} {
		next := d
		mutate(&next)
		q, e = Compile(next, b, Limits{})
		if e != nil || q.Identity().Digest == original {
			t.Fatal("semantic digest unchanged")
		}
	}
}
