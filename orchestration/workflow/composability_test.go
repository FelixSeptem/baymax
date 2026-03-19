package workflow

import (
	"context"
	"strings"
	"testing"
)

func TestComposablePlanAndRunUseCanonicalExpandedIDs(t *testing.T) {
	engine := New(
		WithGraphComposabilityEnabled(true),
		WithStepAdapter(DispatchAdapter{
			Runner: func(ctx context.Context, workflowID string, step Step, attempt int) (StepOutput, error) {
				return StepOutput{Payload: map[string]any{"step_id": step.StepID}}, nil
			},
		}),
	)
	def := Definition{
		WorkflowID: "wf-composable",
		Subgraphs: map[string]Subgraph{
			"prepare": {
				Steps: []Step{
					{StepID: "fetch", Kind: StepKindRunner},
					{StepID: "validate", Kind: StepKindRunner, DependsOn: []string{"fetch"}},
				},
			},
		},
		ConditionTemplates: map[string]string{
			"gate": "{{when}}",
		},
		Steps: []Step{
			{StepID: "prepare-node", UseSubgraph: "prepare", Alias: "prepare"},
			{
				StepID:            "finalize",
				Kind:              StepKindRunner,
				DependsOn:         []string{"prepare-node"},
				ConditionTemplate: "gate",
				TemplateVars:      map[string]string{"when": "on_success"},
			},
		},
	}
	order, err := engine.Plan(def)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}
	if got, want := strings.Join(order, ","), "prepare/fetch,prepare/validate,finalize"; got != want {
		t.Fatalf("plan order = %q, want %q", got, want)
	}

	res, err := engine.Run(context.Background(), RunRequest{RunID: "run-composable", DSL: def})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if got, want := strings.Join(res.ExecutionOrder, ","), "prepare/fetch,prepare/validate,finalize"; got != want {
		t.Fatalf("execution_order = %q, want %q", got, want)
	}
	if res.WorkflowSubgraphExpansionTotal != 2 || res.WorkflowConditionTemplateTotal != 1 || res.WorkflowGraphCompileFailed {
		t.Fatalf("compile summary mismatch: %#v", res)
	}
	byID := map[string]StepResult{}
	for _, step := range res.Steps {
		byID[step.StepID] = step
	}
	for _, id := range []string{"prepare/fetch", "prepare/validate", "finalize"} {
		got, ok := byID[id]
		if !ok {
			t.Fatalf("missing expanded step %q in result: %#v", id, res.Steps)
		}
		if got.Status != StepStatusSucceeded {
			t.Fatalf("step %q status=%q, want succeeded", id, got.Status)
		}
	}
}

func TestComposableCompileFailuresArePreDispatchForRunAndStream(t *testing.T) {
	makeDepthOverflow := func() Definition {
		return Definition{
			WorkflowID: "wf-depth",
			Subgraphs: map[string]Subgraph{
				"g1": {Steps: []Step{{StepID: "s1", UseSubgraph: "g2", Alias: "g2"}}},
				"g2": {Steps: []Step{{StepID: "s2", UseSubgraph: "g3", Alias: "g3"}}},
				"g3": {Steps: []Step{{StepID: "s3", UseSubgraph: "g4", Alias: "g4"}}},
				"g4": {Steps: []Step{{StepID: "leaf", Kind: StepKindRunner}}},
			},
			Steps: []Step{{StepID: "root", UseSubgraph: "g1", Alias: "g1"}},
		}
	}
	makeCycle := func() Definition {
		return Definition{
			WorkflowID: "wf-cycle",
			Subgraphs: map[string]Subgraph{
				"a": {Steps: []Step{{StepID: "sa", UseSubgraph: "b", Alias: "b"}}},
				"b": {Steps: []Step{{StepID: "sb", UseSubgraph: "a", Alias: "a"}}},
			},
			Steps: []Step{{StepID: "root", UseSubgraph: "a", Alias: "a"}},
		}
	}
	makeMissingTemplateVar := func() Definition {
		return Definition{
			WorkflowID: "wf-template-missing",
			ConditionTemplates: map[string]string{
				"gate": "{{status}}",
			},
			Steps: []Step{
				{StepID: "s1", Kind: StepKindRunner, ConditionTemplate: "gate"},
			},
		}
	}
	makeTemplateScopeViolation := func() Definition {
		return Definition{
			WorkflowID: "wf-template-scope",
			ConditionTemplates: map[string]string{
				"bad": "{{mode}}",
			},
			Steps: []Step{
				{StepID: "s1", Kind: StepKindRunner, ConditionTemplate: "bad", TemplateVars: map[string]string{"mode": "payload.value"}},
			},
		}
	}
	makeForbiddenOverride := func() Definition {
		kind := StepKindTool
		return Definition{
			WorkflowID: "wf-forbidden-override",
			Subgraphs: map[string]Subgraph{
				"prepare": {
					Steps: []Step{{StepID: "fetch", Kind: StepKindRunner}},
				},
			},
			Steps: []Step{
				{
					StepID:      "prepare-node",
					UseSubgraph: "prepare",
					Alias:       "prepare",
					Overrides: map[string]StepOverride{
						"fetch": {Kind: &kind},
					},
				},
			},
		}
	}
	makeAliasCollision := func() Definition {
		return Definition{
			WorkflowID: "wf-alias-collision",
			Subgraphs: map[string]Subgraph{
				"leaf": {Steps: []Step{{StepID: "s", Kind: StepKindRunner}}},
			},
			Steps: []Step{
				{StepID: "first", UseSubgraph: "leaf", Alias: "dup"},
				{StepID: "second", UseSubgraph: "leaf", Alias: "dup"},
			},
		}
	}

	cases := []struct {
		name string
		def  Definition
		code ValidationErrorCode
	}{
		{name: "depth_overflow", def: makeDepthOverflow(), code: ErrCodeSubgraphDepthExceeded},
		{name: "cycle", def: makeCycle(), code: ErrCodeSubgraphCycle},
		{name: "missing_template_var", def: makeMissingTemplateVar(), code: ErrCodeConditionTemplateVarMissing},
		{name: "template_scope_violation", def: makeTemplateScopeViolation(), code: ErrCodeConditionTemplateScope},
		{name: "forbidden_override_kind", def: makeForbiddenOverride(), code: ErrCodeSubgraphOverrideForbidden},
		{name: "alias_collision", def: makeAliasCollision(), code: ErrCodeSubgraphAliasCollision},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dispatchCount := 0
			engine := New(
				WithGraphComposabilityEnabled(true),
				WithStepAdapter(DispatchAdapter{
					Runner: func(ctx context.Context, workflowID string, step Step, attempt int) (StepOutput, error) {
						dispatchCount++
						return StepOutput{}, nil
					},
				}),
			)
			_, runErr := engine.Run(context.Background(), RunRequest{RunID: "run-" + tc.name, DSL: tc.def})
			_, streamErr := engine.Stream(context.Background(), RunRequest{RunID: "stream-" + tc.name, DSL: tc.def}, func(StreamEvent) error { return nil })
			if dispatchCount != 0 {
				t.Fatalf("dispatch_count=%d, want 0 for compile fail-fast", dispatchCount)
			}
			for label, err := range map[string]error{"run": runErr, "stream": streamErr} {
				if err == nil {
					t.Fatalf("%s expected compile error, got nil", label)
				}
				verrs, ok := err.(ValidationErrors)
				if !ok || len(verrs) == 0 {
					t.Fatalf("%s expected validation errors, got %T: %v", label, err, err)
				}
				if verrs[0].Code != tc.code {
					t.Fatalf("%s validation code=%q, want %q, errors=%+v", label, verrs[0].Code, tc.code, verrs)
				}
			}
		})
	}
}

func TestComposableDisabledFlagKeepsLegacyAndRejectsComposableSyntax(t *testing.T) {
	engine := New(
		WithStepAdapter(DispatchAdapter{
			Runner: func(ctx context.Context, workflowID string, step Step, attempt int) (StepOutput, error) {
				return StepOutput{}, nil
			},
		}),
	)
	legacy := Definition{
		WorkflowID: "wf-legacy",
		Steps: []Step{
			{StepID: "a", Kind: StepKindRunner},
			{StepID: "b", Kind: StepKindRunner, DependsOn: []string{"a"}},
		},
	}
	if _, err := engine.Run(context.Background(), RunRequest{RunID: "legacy", DSL: legacy}); err != nil {
		t.Fatalf("legacy run failed with composability disabled: %v", err)
	}

	composable := Definition{
		WorkflowID: "wf-disabled-composable",
		Subgraphs: map[string]Subgraph{
			"sg": {Steps: []Step{{StepID: "s", Kind: StepKindRunner}}},
		},
		Steps: []Step{{StepID: "x", UseSubgraph: "sg", Alias: "sg"}},
	}
	_, err := engine.Run(context.Background(), RunRequest{RunID: "disabled", DSL: composable})
	if err == nil {
		t.Fatal("expected graph_composability_disabled error, got nil")
	}
	verrs, ok := err.(ValidationErrors)
	if !ok || len(verrs) == 0 || verrs[0].Code != ErrCodeGraphComposabilityDisabled {
		t.Fatalf("unexpected disabled-flag error: %T %#v", err, err)
	}
}

func TestComposableResumeKeepsExpandedStepDeterminism(t *testing.T) {
	store := NewMemoryCheckpointStore()
	callCount := map[string]int{}
	failOnce := true
	engine := New(
		WithGraphComposabilityEnabled(true),
		WithCheckpointStore(store),
		WithStepAdapter(DispatchAdapter{
			Runner: func(ctx context.Context, workflowID string, step Step, attempt int) (StepOutput, error) {
				callCount[step.StepID]++
				if step.StepID == "flow/b" && failOnce {
					failOnce = false
					return StepOutput{}, context.DeadlineExceeded
				}
				return StepOutput{Payload: map[string]any{"step": step.StepID}}, nil
			},
		}),
	)
	def := Definition{
		WorkflowID: "wf-resume-composable",
		Subgraphs: map[string]Subgraph{
			"flow": {
				Steps: []Step{
					{StepID: "a", Kind: StepKindRunner},
					{StepID: "b", Kind: StepKindRunner, DependsOn: []string{"a"}},
				},
			},
		},
		Steps: []Step{
			{StepID: "flow-node", UseSubgraph: "flow", Alias: "flow"},
			{StepID: "tail", Kind: StepKindRunner, DependsOn: []string{"flow-node"}},
		},
	}

	first, err := engine.Run(context.Background(), RunRequest{RunID: "run-first", DSL: def})
	if err != nil {
		t.Fatalf("first run failed: %v", err)
	}
	if first.WorkflowStatus != "failed" {
		t.Fatalf("first workflow status=%q, want failed", first.WorkflowStatus)
	}

	second, err := engine.Run(context.Background(), RunRequest{RunID: "run-second", Resume: true, DSL: def})
	if err != nil {
		t.Fatalf("resume run failed: %v", err)
	}
	if second.WorkflowStatus != "succeeded" || second.WorkflowResumeCount != 1 {
		t.Fatalf("resume result mismatch: %#v", second)
	}
	if callCount["flow/a"] != 1 {
		t.Fatalf("expanded step flow/a reran during resume: count=%d", callCount["flow/a"])
	}
}

func TestComposableRunStreamSemanticEquivalence(t *testing.T) {
	engine := New(
		WithGraphComposabilityEnabled(true),
		WithStepAdapter(DispatchAdapter{
			A2A: func(ctx context.Context, workflowID string, step Step, attempt int) (StepOutput, error) {
				return StepOutput{Payload: map[string]any{"remote": "ok"}}, nil
			},
		}),
	)
	def := Definition{
		WorkflowID: "wf-run-stream-composable",
		Subgraphs: map[string]Subgraph{
			"remote_flow": {
				Steps: []Step{
					{StepID: "remote", Kind: StepKindA2A, TeamID: "team-a15", AgentID: "agent-a15", PeerID: "peer-a15"},
				},
			},
		},
		Steps: []Step{
			{StepID: "remote-node", UseSubgraph: "remote_flow", Alias: "remote"},
		},
	}
	req := RunRequest{RunID: "run-stream-composable", DSL: def}
	runRes, runErr := engine.Run(context.Background(), req)
	streamRes, streamErr := engine.Stream(context.Background(), req, func(StreamEvent) error { return nil })
	if runErr != nil || streamErr != nil {
		t.Fatalf("run/stream errors mismatch run=%v stream=%v", runErr, streamErr)
	}
	if runRes.WorkflowStatus != streamRes.WorkflowStatus {
		t.Fatalf("workflow status mismatch run=%q stream=%q", runRes.WorkflowStatus, streamRes.WorkflowStatus)
	}
	if runRes.WorkflowRemoteTotal != streamRes.WorkflowRemoteTotal || runRes.WorkflowRemoteFailed != streamRes.WorkflowRemoteFailed {
		t.Fatalf("remote aggregate mismatch run=%#v stream=%#v", runRes, streamRes)
	}
	if runRes.WorkflowSubgraphExpansionTotal != streamRes.WorkflowSubgraphExpansionTotal ||
		runRes.WorkflowConditionTemplateTotal != streamRes.WorkflowConditionTemplateTotal {
		t.Fatalf("compile summary mismatch run=%#v stream=%#v", runRes, streamRes)
	}
}
