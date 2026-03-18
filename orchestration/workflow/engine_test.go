package workflow

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

type timelineCollector struct {
	events []types.Event
}

func (c *timelineCollector) OnEvent(_ context.Context, ev types.Event) {
	c.events = append(c.events, ev)
}

func TestParseDefinitionSupportsJSONAndYAML(t *testing.T) {
	jsonRaw := []byte(`{"workflow_id":"wf-json","steps":[{"step":"s1","kind":"runner"}]}`)
	jsonDef, err := ParseDefinition(jsonRaw)
	if err != nil {
		t.Fatalf("ParseDefinition(json) failed: %v", err)
	}
	if jsonDef.WorkflowID != "wf-json" || len(jsonDef.Steps) != 1 || jsonDef.Steps[0].StepID != "s1" {
		t.Fatalf("parsed json definition mismatch: %#v", jsonDef)
	}

	yamlRaw := []byte(`
workflow_id: wf-yaml
steps:
  - step: s1
    kind: runner
`)
	yamlDef, err := ParseDefinition(yamlRaw)
	if err != nil {
		t.Fatalf("ParseDefinition(yaml) failed: %v", err)
	}
	if yamlDef.WorkflowID != "wf-yaml" || len(yamlDef.Steps) != 1 || yamlDef.Steps[0].StepID != "s1" {
		t.Fatalf("parsed yaml definition mismatch: %#v", yamlDef)
	}
}

func TestValidateDefinitionCatchesStructuralErrors(t *testing.T) {
	def := Definition{
		WorkflowID: "wf-bad",
		Steps: []Step{
			{StepID: "a", DependsOn: []string{"b"}, Kind: StepKind("custom")},
			{StepID: "a", DependsOn: []string{"a"}},
		},
	}
	errs := ValidateDefinition(normalizeDefinition(def))
	if len(errs) == 0 {
		t.Fatal("expected validation errors")
	}
	codes := map[ValidationErrorCode]struct{}{}
	for _, item := range errs {
		codes[item.Code] = struct{}{}
	}
	required := []ValidationErrorCode{
		ErrCodeDuplicateStepID,
		ErrCodeMissingDependency,
		ErrCodeUnsupportedStepKind,
		ErrCodeDependencyCycle,
	}
	for _, code := range required {
		if _, ok := codes[code]; !ok {
			t.Fatalf("missing expected validation code %q in %+v", code, errs)
		}
	}
}

func TestPlanDeterministicStableOrder(t *testing.T) {
	engine := New()
	def := Definition{
		WorkflowID: "wf-plan",
		Steps: []Step{
			{StepID: "c", DependsOn: []string{"a"}},
			{StepID: "b"},
			{StepID: "a"},
		},
	}
	order, err := engine.Plan(def)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}
	got := strings.Join(order, ",")
	want := "a,b,c"
	if got != want {
		t.Fatalf("order = %q, want %q", got, want)
	}
}

func TestRetryAndTimeoutSemantics(t *testing.T) {
	attempts := 0
	engine := New(
		WithDefaultStepTimeout(80*time.Millisecond),
		WithStepAdapter(DispatchAdapter{
			Runner: func(ctx context.Context, workflowID string, step Step, attempt int) (StepOutput, error) {
				attempts++
				if step.StepID == "slow" {
					<-ctx.Done()
					return StepOutput{}, ctx.Err()
				}
				if attempts < 3 {
					return StepOutput{}, errors.New("transient")
				}
				return StepOutput{Payload: map[string]any{"ok": true}}, nil
			},
		}),
	)

	req := RunRequest{
		DSL: Definition{
			WorkflowID: "wf-retry",
			Steps: []Step{
				{StepID: "retryable", Kind: StepKindRunner, Retry: Retry{MaxAttempts: 2}},
				{StepID: "slow", Kind: StepKindRunner, Retry: Retry{MaxAttempts: 0}, Timeout: 30 * time.Millisecond},
			},
		},
	}
	res, err := engine.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	byID := map[string]StepResult{}
	for _, item := range res.Steps {
		byID[item.StepID] = item
	}
	if byID["retryable"].Status != StepStatusSucceeded || byID["retryable"].Attempts != 3 {
		t.Fatalf("retryable step mismatch: %#v", byID["retryable"])
	}
	if byID["slow"].Status != StepStatusFailed || byID["slow"].Reason != "step.timeout" {
		t.Fatalf("slow timeout mismatch: %#v", byID["slow"])
	}
}

func TestCheckpointResumeSemantics(t *testing.T) {
	store := NewMemoryCheckpointStore()
	runSlow := true
	engine := New(
		WithCheckpointStore(store),
		WithStepAdapter(DispatchAdapter{
			Runner: func(ctx context.Context, workflowID string, step Step, attempt int) (StepOutput, error) {
				if step.StepID == "s2" && runSlow {
					return StepOutput{}, errors.New("boom")
				}
				return StepOutput{Payload: map[string]any{"step": step.StepID}}, nil
			},
		}),
	)

	base := Definition{
		WorkflowID: "wf-resume",
		Steps: []Step{
			{StepID: "s1", Kind: StepKindRunner},
			{StepID: "s2", Kind: StepKindRunner, DependsOn: []string{"s1"}},
		},
	}
	first, err := engine.Run(context.Background(), RunRequest{RunID: "run-1", DSL: base})
	if err != nil {
		t.Fatalf("first run failed: %v", err)
	}
	if first.WorkflowStatus != "failed" {
		t.Fatalf("first workflow status = %q, want failed", first.WorkflowStatus)
	}

	runSlow = false
	second, err := engine.Run(context.Background(), RunRequest{RunID: "run-2", Resume: true, DSL: base})
	if err != nil {
		t.Fatalf("resume run failed: %v", err)
	}
	if second.WorkflowResumeCount != 1 {
		t.Fatalf("workflow_resume_count = %d, want 1", second.WorkflowResumeCount)
	}
	if second.WorkflowStatus != "succeeded" {
		t.Fatalf("resume workflow status = %q, want succeeded", second.WorkflowStatus)
	}
}

func TestRunStreamSemanticEquivalence(t *testing.T) {
	engine := New(
		WithStepAdapter(DispatchAdapter{
			Runner: func(ctx context.Context, workflowID string, step Step, attempt int) (StepOutput, error) {
				return StepOutput{Payload: map[string]any{"step": step.StepID}}, nil
			},
		}),
	)
	req := RunRequest{
		RunID: "run-eq",
		DSL: Definition{
			WorkflowID: "wf-eq",
			Steps: []Step{
				{StepID: "a", Kind: StepKindRunner},
				{StepID: "b", Kind: StepKindRunner, DependsOn: []string{"a"}},
			},
		},
	}
	runRes, runErr := engine.Run(context.Background(), req)
	streamEvents := 0
	streamRes, streamErr := engine.Stream(context.Background(), req, func(ev StreamEvent) error {
		streamEvents++
		return nil
	})
	if runErr != nil || streamErr != nil {
		t.Fatalf("run/stream errors mismatch run=%v stream=%v", runErr, streamErr)
	}
	if streamEvents == 0 {
		t.Fatal("stream should emit events")
	}
	if runRes.WorkflowStatus != streamRes.WorkflowStatus {
		t.Fatalf("workflow status mismatch run=%q stream=%q", runRes.WorkflowStatus, streamRes.WorkflowStatus)
	}
	if strings.Join(runRes.ExecutionOrder, ",") != strings.Join(streamRes.ExecutionOrder, ",") {
		t.Fatalf("execution order mismatch run=%v stream=%v", runRes.ExecutionOrder, streamRes.ExecutionOrder)
	}
	if runRes.WorkflowStepTotal != streamRes.WorkflowStepTotal || runRes.WorkflowStepFailed != streamRes.WorkflowStepFailed {
		t.Fatalf("workflow aggregate mismatch run=%#v stream=%#v", runRes, streamRes)
	}
}

func TestTimelineCarriesWorkflowMetadataAndReasonNamespace(t *testing.T) {
	collector := &timelineCollector{}
	store := NewMemoryCheckpointStore()
	failOnce := true
	engine := New(
		WithTimelineEmitter(collector),
		WithCheckpointStore(store),
		WithStepAdapter(DispatchAdapter{
			Runner: func(ctx context.Context, workflowID string, step Step, attempt int) (StepOutput, error) {
				if step.StepID == "r" && failOnce {
					failOnce = false
					return StepOutput{}, errors.New("retry me")
				}
				return StepOutput{}, nil
			},
		}),
	)
	req := RunRequest{
		RunID: "run-meta-1",
		DSL: Definition{
			WorkflowID: "wf-meta",
			Steps: []Step{
				{StepID: "r", Kind: StepKindRunner, Retry: Retry{MaxAttempts: 1}},
			},
		},
	}
	if _, err := engine.Run(context.Background(), req); err != nil {
		t.Fatalf("first run failed: %v", err)
	}
	if _, err := engine.Run(context.Background(), RunRequest{RunID: "run-meta-2", Resume: true, DSL: req.DSL}); err != nil {
		t.Fatalf("resume run failed: %v", err)
	}

	seen := map[string]bool{}
	for _, ev := range collector.events {
		if ev.Type != types.EventTypeActionTimeline {
			continue
		}
		reason, _ := ev.Payload["reason"].(string)
		seen[reason] = true
		if !strings.HasPrefix(reason, "workflow.") {
			t.Fatalf("reason namespace mismatch: %q", reason)
		}
		workflowID, _ := ev.Payload["workflow_id"].(string)
		if workflowID != "wf-meta" {
			t.Fatalf("workflow_id = %q, want wf-meta", workflowID)
		}
		if _, ok := ev.Payload["step_id"]; !ok {
			t.Fatalf("step_id missing in payload: %#v", ev.Payload)
		}
	}
	for _, reason := range []string{ReasonSchedule, ReasonRetry, ReasonResume} {
		if !seen[reason] {
			t.Fatalf("missing reason %q in %v", reason, seen)
		}
	}
}
