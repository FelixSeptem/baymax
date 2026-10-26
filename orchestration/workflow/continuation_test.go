package workflow

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/context/hostprojection"
)

func openTestFileWorkflowStore(t *testing.T, root, namespace string) *FileWorkflowContinuationStore {
	t.Helper()
	store, err := NewFileWorkflowContinuationStore(root, namespace)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

type rejectingCheckpointStore struct{}

func (rejectingCheckpointStore) Load(context.Context, string) (Checkpoint, bool, error) {
	return Checkpoint{}, false, errors.New("legacy checkpoint load must not run")
}

func (rejectingCheckpointStore) Save(context.Context, Checkpoint) error {
	return errors.New("legacy checkpoint save must not run")
}

func TestEngineResumeContinuationRestoresSameGraphWithoutRerunningTerminalStep(t *testing.T) {
	store := NewMemoryWorkflowContinuationStore("workflow")
	firstStepCalls := 0
	first := New(
		WithWorkflowContinuationStore(store, "workflow"),
		WithStepAdapter(DispatchAdapter{Runner: func(_ context.Context, _ string, step Step, _ int) (StepOutput, error) {
			if step.StepID == "first" {
				firstStepCalls++
				return StepOutput{}, nil
			}
			return StepOutput{}, PauseError{Reason: PauseInputRequired, PendingActionReference: "input-1"}
		}}),
	)
	def := Definition{WorkflowID: "wf-resume-native", Steps: []Step{{StepID: "first", Kind: StepKindRunner}, {StepID: "second", Kind: StepKindRunner, DependsOn: []string{"first"}}}}
	paused, err := first.Run(context.Background(), RunRequest{RunID: "run-native", SessionID: "session-native", DSL: def})
	if !errors.Is(err, ErrWorkflowPaused) {
		t.Fatalf("pause error = %v", err)
	}
	if paused.WorkflowCheckpoint == nil || paused.WorkflowStatus != string(PauseInputRequired) {
		t.Fatalf("paused result = %#v", paused)
	}

	secondStepCalls := 0
	fresh := New(
		WithWorkflowContinuationStore(store, "workflow"),
		WithStepAdapter(DispatchAdapter{Runner: func(_ context.Context, _ string, step Step, _ int) (StepOutput, error) {
			if step.StepID == "first" {
				firstStepCalls++
				return StepOutput{}, errors.New("first step must not rerun")
			}
			secondStepCalls++
			return StepOutput{}, nil
		}}),
	)
	resumed, err := fresh.ResumeContinuation(context.Background(), WorkflowResumeRequest{Ref: *paused.WorkflowCheckpoint, GraphDigest: paused.WorkflowGraphDigest, Generation: paused.WorkflowCheckpointGeneration, IdempotencyKey: "resume-1"}, nil)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if resumed.RunID != "run-native" || resumed.WorkflowStatus != "succeeded" || firstStepCalls != 1 || secondStepCalls != 1 {
		t.Fatalf("resume result = %#v calls first=%d second=%d", resumed, firstStepCalls, secondStepCalls)
	}
	duplicate, err := fresh.ResumeContinuation(context.Background(), WorkflowResumeRequest{Ref: *paused.WorkflowCheckpoint, GraphDigest: paused.WorkflowGraphDigest, Generation: paused.WorkflowCheckpointGeneration, IdempotencyKey: "resume-1"}, nil)
	if err != nil {
		t.Fatalf("duplicate resume: %v", err)
	}
	if duplicate.WorkflowStatus != "succeeded" || firstStepCalls != 1 || secondStepCalls != 1 {
		t.Fatalf("duplicate reran graph: %#v calls first=%d second=%d", duplicate, firstStepCalls, secondStepCalls)
	}
}

func TestEngineResumeContinuationRestoresFromFileStoreOnNewEngine(t *testing.T) {
	store := openTestFileWorkflowStore(t, t.TempDir(), "workflow")
	completed := 0
	first := New(
		WithWorkflowContinuationStore(store, "workflow"),
		WithStepAdapter(DispatchAdapter{Runner: func(_ context.Context, _ string, step Step, _ int) (StepOutput, error) {
			if step.StepID == "completed" {
				completed++
				return StepOutput{}, nil
			}
			return StepOutput{}, PauseError{Reason: PauseInputRequired, PendingActionReference: "input"}
		}}),
	)
	definition := Definition{WorkflowID: "wf-file-engine", Steps: []Step{{StepID: "completed", Kind: StepKindRunner}, {StepID: "pending", Kind: StepKindRunner, DependsOn: []string{"completed"}}}}
	paused, err := first.Run(context.Background(), RunRequest{RunID: "run-file-engine", SessionID: "session-file-engine", DSL: definition})
	if !errors.Is(err, ErrWorkflowPaused) {
		t.Fatalf("pause: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	freshStore := openTestFileWorkflowStore(t, store.root, "workflow")
	pending := 0
	fresh := New(
		WithWorkflowContinuationStore(freshStore, "workflow"),
		WithStepAdapter(DispatchAdapter{Runner: func(_ context.Context, _ string, step Step, _ int) (StepOutput, error) {
			if step.StepID == "completed" {
				completed++
				return StepOutput{}, errors.New("completed step reran")
			}
			pending++
			return StepOutput{}, nil
		}}),
	)
	result, err := fresh.ResumeContinuation(context.Background(), WorkflowResumeRequest{Ref: *paused.WorkflowCheckpoint, GraphDigest: paused.WorkflowGraphDigest, Generation: paused.WorkflowCheckpointGeneration, IdempotencyKey: "resume"}, nil)
	if err != nil {
		t.Fatalf("new engine resume: %v", err)
	}
	if result.WorkflowStatus != "succeeded" || completed != 1 || pending != 1 {
		t.Fatalf("result=%#v completed=%d pending=%d", result, completed, pending)
	}
}

func TestWorkflowContinuationRestoresExpandedAliasGraphWithoutRerun(t *testing.T) {
	store := NewMemoryWorkflowContinuationStore("workflow")
	firstCalls := map[string]int{}
	first := New(
		WithGraphComposabilityEnabled(true),
		WithWorkflowContinuationStore(store, "workflow"),
		WithStepAdapter(DispatchAdapter{Runner: func(_ context.Context, _ string, step Step, _ int) (StepOutput, error) {
			firstCalls[step.StepID]++
			if step.StepID == "flow/b" {
				return StepOutput{}, PauseError{Reason: PauseInputRequired, PendingActionReference: "input"}
			}
			return StepOutput{}, nil
		}}),
	)
	definition := Definition{
		WorkflowID: "wf-expanded-continuation",
		Subgraphs: map[string]Subgraph{
			"flow": {Steps: []Step{{StepID: "a", Kind: StepKindRunner}, {StepID: "b", Kind: StepKindRunner, DependsOn: []string{"a"}}}},
		},
		Steps: []Step{{StepID: "flow-node", UseSubgraph: "flow", Alias: "flow"}, {StepID: "tail", Kind: StepKindRunner, DependsOn: []string{"flow-node"}}},
	}
	paused, err := first.Run(context.Background(), RunRequest{RunID: "run-expanded", SessionID: "session", DSL: definition})
	if !errors.Is(err, ErrWorkflowPaused) {
		t.Fatalf("pause: %v", err)
	}
	if firstCalls["flow/a"] != 1 || firstCalls["flow/b"] != 1 {
		t.Fatalf("initial expanded calls=%#v", firstCalls)
	}
	resumedCalls := map[string]int{}
	resumer := New(
		WithGraphComposabilityEnabled(true),
		WithWorkflowContinuationStore(store, "workflow"),
		WithStepAdapter(DispatchAdapter{Runner: func(_ context.Context, _ string, step Step, _ int) (StepOutput, error) {
			resumedCalls[step.StepID]++
			if step.StepID == "flow/a" {
				return StepOutput{}, errors.New("expanded terminal step reran")
			}
			return StepOutput{}, nil
		}}),
	)
	result, err := resumer.ResumeContinuation(context.Background(), WorkflowResumeRequest{Ref: *paused.WorkflowCheckpoint, GraphDigest: paused.WorkflowGraphDigest, Generation: paused.WorkflowCheckpointGeneration, IdempotencyKey: "resume"}, nil)
	if err != nil || result.WorkflowStatus != "succeeded" {
		t.Fatalf("expanded resume result=%#v err=%v", result, err)
	}
	if resumedCalls["flow/a"] != 0 || resumedCalls["flow/b"] != 1 || resumedCalls["tail"] != 1 {
		t.Fatalf("resumed expanded calls=%#v", resumedCalls)
	}
}

func TestEngineResumeContinuationUsesSourceStateInsteadOfLegacyCheckpointStore(t *testing.T) {
	store := NewMemoryWorkflowContinuationStore("workflow")
	first := New(
		WithWorkflowContinuationStore(store, "workflow"),
		WithStepAdapter(DispatchAdapter{Runner: func(_ context.Context, _ string, step Step, _ int) (StepOutput, error) {
			if step.StepID == "first" {
				return StepOutput{}, nil
			}
			return StepOutput{}, PauseError{Reason: PauseInputRequired}
		}}),
	)
	definition := Definition{WorkflowID: "wf-source-only", Steps: []Step{{StepID: "first", Kind: StepKindRunner}, {StepID: "second", Kind: StepKindRunner, DependsOn: []string{"first"}}}}
	paused, err := first.Run(context.Background(), RunRequest{RunID: "run-source-only", SessionID: "session", DSL: definition})
	if !errors.Is(err, ErrWorkflowPaused) {
		t.Fatalf("pause: %v", err)
	}

	resumed := New(
		WithWorkflowContinuationStore(store, "workflow"),
		WithCheckpointStore(rejectingCheckpointStore{}),
		WithStepAdapter(DispatchAdapter{Runner: func(context.Context, string, Step, int) (StepOutput, error) { return StepOutput{}, nil }}),
	)
	result, err := resumed.ResumeContinuation(context.Background(), WorkflowResumeRequest{Ref: *paused.WorkflowCheckpoint, GraphDigest: paused.WorkflowGraphDigest, Generation: paused.WorkflowCheckpointGeneration, IdempotencyKey: "resume"}, nil)
	if err != nil || result.WorkflowStatus != "succeeded" {
		t.Fatalf("source restore result=%#v err=%v", result, err)
	}
}

func TestWorkflowContinuationRejectsResumeBindingDriftBeforeDispatch(t *testing.T) {
	store := NewMemoryWorkflowContinuationStore("workflow")
	first := New(
		WithWorkflowContinuationStore(store, "workflow"),
		WithStepAdapter(DispatchAdapter{Runner: func(context.Context, string, Step, int) (StepOutput, error) {
			return StepOutput{}, PauseError{Reason: PauseInputRequired}
		}}),
	)
	paused, err := first.Run(context.Background(), RunRequest{
		RunID: "run-binding", SessionID: "session-binding", AuthorityDigest: "authority", PlanDigest: "plan", WriteScopeDigest: "scope", EffectNamespace: "effects", OccurrenceID: "occurrence",
		DSL: Definition{WorkflowID: "wf-binding", Steps: []Step{{StepID: "step", Kind: StepKindRunner}}},
	})
	if !errors.Is(err, ErrWorkflowPaused) {
		t.Fatalf("pause: %v", err)
	}
	dispatches := 0
	resumer := New(WithWorkflowContinuationStore(store, "workflow"), WithStepAdapter(DispatchAdapter{Runner: func(context.Context, string, Step, int) (StepOutput, error) {
		dispatches++
		return StepOutput{}, nil
	}}))
	_, err = resumer.ResumeContinuation(context.Background(), WorkflowResumeRequest{
		Ref: *paused.WorkflowCheckpoint, GraphDigest: paused.WorkflowGraphDigest, Generation: paused.WorkflowCheckpointGeneration, IdempotencyKey: "resume",
		AuthorityDigest: "authority", PlanDigest: "different", WriteScopeDigest: "scope", EffectNamespace: "effects", OccurrenceID: "occurrence",
	}, nil)
	if !errors.Is(err, ErrWorkflowContinuationForeign) || dispatches != 0 {
		t.Fatalf("binding drift err=%v dispatches=%d", err, dispatches)
	}
}

func TestWorkflowContinuationRejectsForeignStepProjectionBeforeResolverOrClaim(t *testing.T) {
	store := NewMemoryWorkflowContinuationStore("workflow")
	first := New(WithWorkflowContinuationStore(store, "workflow"), WithStepAdapter(DispatchAdapter{Runner: func(context.Context, string, Step, int) (StepOutput, error) {
		return StepOutput{}, PauseError{Reason: PauseInputRequired}
	}}))
	paused, err := first.Run(context.Background(), RunRequest{RunID: "run-context", SessionID: "session-context", HostContextBinding: &hostprojection.SourceBinding{Form: hostprojection.SourceFormWorkflow}, DSL: Definition{WorkflowID: "wf-context", Steps: []Step{{StepID: "pending", Kind: StepKindRunner}}}})
	if !errors.Is(err, ErrWorkflowPaused) || paused.WorkflowCheckpoint == nil {
		t.Fatalf("pause result=%#v err=%v", paused, err)
	}
	continuation, err := store.Load(context.Background(), *paused.WorkflowCheckpoint)
	if err != nil {
		t.Fatal(err)
	}
	foreignBinding := hostprojection.SourceBinding{Form: hostprojection.SourceFormWorkflow, OwnerNamespace: "workflow", SourceID: "run-context", NodeID: "other-step", ContinuationID: continuation.Ref.CheckpointID, ContinuationDigest: continuation.GraphDigest, Generation: continuation.Generation}
	projection := hostprojection.Descriptor{SchemaVersion: hostprojection.SchemaVersionV1, ProjectionID: "wf-foreign", ProjectionVersion: 1, CapabilityDigest: "capability", Capabilities: []hostprojection.Capability{hostprojection.CapabilityRead}, SourceReferences: []hostprojection.SourceReference{{Reference: "source", Digest: "digest"}}, Binding: &foreignBinding}
	projection.Digest, err = hostprojection.CanonicalDigest(projection)
	if err != nil {
		t.Fatal(err)
	}
	resolverCalls := 0
	resolver := workflowContextResolverFunc(func(context.Context, hostprojection.SourceReference) (hostprojection.Material, error) {
		resolverCalls++
		return hostprojection.Material{Role: "private", Content: "must-not-resolve"}, nil
	})
	request := WorkflowResumeRequest{Ref: *paused.WorkflowCheckpoint, GraphDigest: paused.WorkflowGraphDigest, Generation: paused.WorkflowCheckpointGeneration, IdempotencyKey: "resume-foreign", HostContextProjection: &projection, HostContextResolver: resolver}
	_, err = New(WithWorkflowContinuationStore(store, "workflow"), WithStepAdapter(contextTestAdapter{})).ResumeContinuation(context.Background(), request, nil)
	if !errors.Is(err, ErrWorkflowContinuationConflict) || resolverCalls != 0 {
		t.Fatalf("resume error=%v resolver calls=%d", err, resolverCalls)
	}
	if _, err := store.ClaimResume(context.Background(), request); err != nil {
		t.Fatalf("foreign step projection consumed claim: %v", err)
	}
}

func TestWorkflowContinuationResolvesBoundContextOnlyForPendingStep(t *testing.T) {
	store := NewMemoryWorkflowContinuationStore("workflow")
	completedCalls, pauseCalls := 0, 0
	first := New(WithWorkflowContinuationStore(store, "workflow"), WithStepAdapter(DispatchAdapter{Runner: func(_ context.Context, _ string, step Step, _ int) (StepOutput, error) {
		if step.StepID == "completed" {
			completedCalls++
			return StepOutput{EffectReference: "effect-completed"}, nil
		}
		pauseCalls++
		return StepOutput{}, PauseError{Reason: PauseInputRequired}
	}}))
	def := Definition{WorkflowID: "wf-context-resume", Steps: []Step{{StepID: "completed", Kind: StepKindRunner}, {StepID: "pending", Kind: StepKindRunner, DependsOn: []string{"completed"}}}}
	paused, err := first.Run(context.Background(), RunRequest{RunID: "run-context-resume", SessionID: "session-context-resume", HostContextBinding: &hostprojection.SourceBinding{Form: hostprojection.SourceFormWorkflow}, DSL: def})
	if !errors.Is(err, ErrWorkflowPaused) || paused.WorkflowCheckpoint == nil {
		t.Fatalf("pause result=%#v err=%v", paused, err)
	}
	continuation, err := store.Load(context.Background(), *paused.WorkflowCheckpoint)
	if err != nil || continuation.HostContextBinding == nil {
		t.Fatalf("continuation binding=%#v err=%v", continuation.HostContextBinding, err)
	}
	projection := hostprojection.Descriptor{SchemaVersion: hostprojection.SchemaVersionV1, ProjectionID: "wf-pending-context", ProjectionVersion: 1, CapabilityDigest: "capability", Capabilities: []hostprojection.Capability{hostprojection.CapabilityRead}, SourceReferences: []hostprojection.SourceReference{{Reference: "source", Digest: "digest"}}, Binding: continuation.HostContextBinding}
	projection.Digest, err = hostprojection.CanonicalDigest(projection)
	if err != nil {
		t.Fatal(err)
	}
	resolverCalls, contextCalls := 0, 0
	resolver := workflowContextResolverFunc(func(context.Context, hostprojection.SourceReference) (hostprojection.Material, error) {
		resolverCalls++
		return hostprojection.Material{Role: "private", Content: "authorized-context"}, nil
	})
	adapter := contextTestAdapter{execute: func(_ context.Context, _ string, step Step, _ int, resolution hostprojection.Resolution) (StepOutput, error) {
		contextCalls++
		if step.StepID != "pending" || resolution.Materials()[0].Content != "authorized-context" {
			t.Fatalf("step=%q resolution=%#v", step.StepID, resolution.Materials())
		}
		return StepOutput{}, nil
	}}
	request := WorkflowResumeRequest{Ref: *paused.WorkflowCheckpoint, GraphDigest: paused.WorkflowGraphDigest, Generation: paused.WorkflowCheckpointGeneration, IdempotencyKey: "resume-context", HostContextProjection: &projection, HostContextResolver: resolver}
	result, err := New(WithWorkflowContinuationStore(store, "workflow"), WithStepAdapter(adapter)).ResumeContinuation(context.Background(), request, nil)
	if err != nil || result.WorkflowStatus != "succeeded" || resolverCalls != 1 || contextCalls != 1 || completedCalls != 1 || pauseCalls != 1 {
		t.Fatalf("result=%#v err=%v resolver=%d context=%d completed=%d paused=%d", result, err, resolverCalls, contextCalls, completedCalls, pauseCalls)
	}
}

func TestWorkflowContextResumeRunStreamParity(t *testing.T) {
	newPaused := func(t *testing.T) (WorkflowContinuationStore, RunResult) {
		t.Helper()
		store := NewMemoryWorkflowContinuationStore("workflow")
		engine := New(WithWorkflowContinuationStore(store, "workflow"), WithStepAdapter(DispatchAdapter{Runner: func(context.Context, string, Step, int) (StepOutput, error) {
			return StepOutput{}, PauseError{Reason: PauseInputRequired}
		}}))
		paused, err := engine.Run(context.Background(), RunRequest{
			RunID: "run-context-parity", SessionID: "session-context-parity",
			HostContextBinding: &hostprojection.SourceBinding{Form: hostprojection.SourceFormWorkflow},
			DSL:                Definition{WorkflowID: "wf-context-parity", Steps: []Step{{StepID: "pending", Kind: StepKindRunner}}},
		})
		if !errors.Is(err, ErrWorkflowPaused) || paused.WorkflowCheckpoint == nil {
			t.Fatalf("pause=%#v err=%v", paused, err)
		}
		return store, paused
	}
	projectionFor := func(t *testing.T, store WorkflowContinuationStore, paused RunResult) hostprojection.Descriptor {
		t.Helper()
		continuation, err := store.Load(context.Background(), *paused.WorkflowCheckpoint)
		if err != nil || continuation.HostContextBinding == nil {
			t.Fatalf("continuation=%#v err=%v", continuation, err)
		}
		projection := hostprojection.Descriptor{
			SchemaVersion: hostprojection.SchemaVersionV1, ProjectionID: "workflow-context-parity", ProjectionVersion: 1,
			CapabilityDigest: "workflow-context-capability", Capabilities: []hostprojection.Capability{hostprojection.CapabilityRead},
			SourceReferences: []hostprojection.SourceReference{{Reference: "opaque://workflow/parity", Digest: "workflow-parity-source"}},
			Binding:          continuation.HostContextBinding,
		}
		projection.Digest, err = hostprojection.CanonicalDigest(projection)
		if err != nil {
			t.Fatal(err)
		}
		return projection
	}
	newResolver := func(calls *int) workflowContextResolverFunc {
		return func(context.Context, hostprojection.SourceReference) (hostprojection.Material, error) {
			*calls++
			return hostprojection.Material{Role: "private", Content: "parity-context"}, nil
		}
	}
	newAdapter := func(calls *int) contextTestAdapter {
		return contextTestAdapter{execute: func(_ context.Context, _ string, step Step, _ int, resolution hostprojection.Resolution) (StepOutput, error) {
			*calls++
			if step.StepID != "pending" || len(resolution.Materials()) != 1 || resolution.Materials()[0].Content != "parity-context" {
				return StepOutput{}, errors.New("workflow context parity mismatch")
			}
			return StepOutput{EffectReference: "effect:parity"}, nil
		}}
	}

	runStore, runPaused := newPaused(t)
	runProjection := projectionFor(t, runStore, runPaused)
	runResolverCalls, runContextCalls := 0, 0
	runResult, runErr := New(WithWorkflowContinuationStore(runStore, "workflow"), WithStepAdapter(newAdapter(&runContextCalls))).ResumeContinuation(context.Background(), WorkflowResumeRequest{
		Ref: *runPaused.WorkflowCheckpoint, GraphDigest: runPaused.WorkflowGraphDigest, Generation: runPaused.WorkflowCheckpointGeneration,
		IdempotencyKey: "workflow-context-parity", HostContextProjection: &runProjection, HostContextResolver: newResolver(&runResolverCalls),
	}, nil)

	streamStore, streamPaused := newPaused(t)
	streamProjection := projectionFor(t, streamStore, streamPaused)
	streamResolverCalls, streamContextCalls, completedEvents := 0, 0, 0
	streamResult, streamErr := New(WithWorkflowContinuationStore(streamStore, "workflow"), WithStepAdapter(newAdapter(&streamContextCalls))).ResumeContinuation(context.Background(), WorkflowResumeRequest{
		Ref: *streamPaused.WorkflowCheckpoint, GraphDigest: streamPaused.WorkflowGraphDigest, Generation: streamPaused.WorkflowCheckpointGeneration,
		IdempotencyKey: "workflow-context-parity", HostContextProjection: &streamProjection, HostContextResolver: newResolver(&streamResolverCalls),
	}, func(event StreamEvent) error {
		if event.Kind == "workflow.completed" {
			completedEvents++
		}
		return nil
	})
	if runErr != nil || streamErr != nil || runResult.WorkflowStatus != streamResult.WorkflowStatus || runResult.RunID != streamResult.RunID || runResolverCalls != 1 || streamResolverCalls != 1 || runContextCalls != 1 || streamContextCalls != 1 || completedEvents != 1 {
		t.Fatalf("Run/Stream context parity run=%#v err=%v stream=%#v err=%v resolver=%d/%d context=%d/%d completed=%d", runResult, runErr, streamResult, streamErr, runResolverCalls, streamResolverCalls, runContextCalls, streamContextCalls, completedEvents)
	}
}

func TestWorkflowContextResumeRechecksExpiryAfterResolver(t *testing.T) {
	store := NewMemoryWorkflowContinuationStore("workflow")
	deadline := time.Now().Add(time.Minute)
	first := New(WithWorkflowContinuationStore(store, "workflow"), WithStepAdapter(DispatchAdapter{Runner: func(context.Context, string, Step, int) (StepOutput, error) {
		return StepOutput{}, PauseError{Reason: PauseBudgetExhausted, OriginalDeadline: deadline}
	}}))
	paused, err := first.Run(context.Background(), RunRequest{
		RunID: "run-context-expiry-fence", SessionID: "session-context-expiry-fence",
		HostContextBinding: &hostprojection.SourceBinding{Form: hostprojection.SourceFormWorkflow},
		DSL:                Definition{WorkflowID: "wf-context-expiry-fence", Steps: []Step{{StepID: "pending", Kind: StepKindRunner}}},
	})
	if !errors.Is(err, ErrWorkflowPaused) || paused.WorkflowCheckpoint == nil {
		t.Fatalf("pause=%#v err=%v", paused, err)
	}
	continuation, err := store.Load(context.Background(), *paused.WorkflowCheckpoint)
	if err != nil || continuation.HostContextBinding == nil {
		t.Fatalf("continuation=%#v err=%v", continuation, err)
	}
	projection := hostprojection.Descriptor{
		SchemaVersion: hostprojection.SchemaVersionV1, ProjectionID: "workflow-context-expiry-fence", ProjectionVersion: 1,
		CapabilityDigest: "workflow-context-capability", Capabilities: []hostprojection.Capability{hostprojection.CapabilityRead},
		SourceReferences: []hostprojection.SourceReference{{Reference: "opaque://workflow/expiry-fence", Digest: "workflow-expiry-fence"}},
		Binding:          continuation.HostContextBinding,
	}
	projection.Digest, err = hostprojection.CanonicalDigest(projection)
	if err != nil {
		t.Fatal(err)
	}
	resolver := workflowContextResolverFunc(func(context.Context, hostprojection.SourceReference) (hostprojection.Material, error) {
		return hostprojection.Material{Role: "private", Content: "authorized-before-expiry"}, nil
	})
	stepCalls := 0
	engine := New(WithWorkflowContinuationStore(store, "workflow"), WithStepAdapter(contextTestAdapter{execute: func(context.Context, string, Step, int, hostprojection.Resolution) (StepOutput, error) {
		stepCalls++
		return StepOutput{}, nil
	}}))
	nowCalls := 0
	engine.now = func() time.Time {
		nowCalls++
		if nowCalls == 1 {
			return deadline.Add(-time.Second)
		}
		return deadline
	}
	request := WorkflowResumeRequest{
		Ref: *paused.WorkflowCheckpoint, GraphDigest: paused.WorkflowGraphDigest, Generation: paused.WorkflowCheckpointGeneration,
		IdempotencyKey: "workflow-context-expiry-fence", HostContextProjection: &projection, HostContextResolver: resolver,
	}
	if _, err := engine.ResumeContinuation(context.Background(), request, nil); !errors.Is(err, ErrWorkflowContinuationExpired) {
		t.Fatalf("expiry after resolver error=%v, want expired", err)
	}
	if stepCalls != 0 {
		t.Fatalf("step calls=%d, want 0", stepCalls)
	}
	if record := store.records[paused.WorkflowCheckpoint.CheckpointID]; record.Claim != nil {
		t.Fatalf("expiry after resolver consumed resume claim=%#v", record.Claim)
	}
}

type workflowContextResolverFunc func(context.Context, hostprojection.SourceReference) (hostprojection.Material, error)

type contextTestAdapter struct {
	execute func(context.Context, string, Step, int, hostprojection.Resolution) (StepOutput, error)
}

func (contextTestAdapter) Execute(context.Context, string, Step, int) (StepOutput, error) {
	return StepOutput{}, nil
}

func (a contextTestAdapter) ExecuteWithHostContext(ctx context.Context, workflowID string, step Step, attempt int, resolution hostprojection.Resolution) (StepOutput, error) {
	if a.execute == nil {
		return StepOutput{}, nil
	}
	return a.execute(ctx, workflowID, step, attempt, resolution)
}

func (f workflowContextResolverFunc) ResolveHostContext(ctx context.Context, ref hostprojection.SourceReference) (hostprojection.Material, error) {
	return f(ctx, ref)
}

func TestWorkflowContinuationCommitsCancellationTerminalEvidence(t *testing.T) {
	store := NewMemoryWorkflowContinuationStore("workflow")
	first := New(WithWorkflowContinuationStore(store, "workflow"), WithStepAdapter(DispatchAdapter{Runner: func(context.Context, string, Step, int) (StepOutput, error) {
		return StepOutput{}, PauseError{Reason: PauseInputRequired}
	}}))
	paused, err := first.Run(context.Background(), RunRequest{RunID: "run-cancel", SessionID: "session", DSL: Definition{WorkflowID: "wf-cancel", Steps: []Step{{StepID: "step", Kind: StepKindRunner}}}})
	if !errors.Is(err, ErrWorkflowPaused) {
		t.Fatalf("pause: %v", err)
	}
	started := make(chan struct{})
	resumer := New(WithWorkflowContinuationStore(store, "workflow"), WithStepAdapter(DispatchAdapter{Runner: func(ctx context.Context, _ string, _ Step, _ int) (StepOutput, error) {
		close(started)
		<-ctx.Done()
		return StepOutput{}, ctx.Err()
	}}))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	request := WorkflowResumeRequest{Ref: *paused.WorkflowCheckpoint, GraphDigest: paused.WorkflowGraphDigest, Generation: paused.WorkflowCheckpointGeneration, IdempotencyKey: "resume"}
	done := make(chan error, 1)
	go func() {
		_, resumeErr := resumer.ResumeContinuation(ctx, request, nil)
		done <- resumeErr
	}()
	<-started
	cancel()
	err = <-done
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel resume error = %v", err)
	}
	duplicate, err := resumer.ResumeContinuation(context.Background(), request, nil)
	if err != nil || duplicate.WorkflowStatus != "canceled" {
		t.Fatalf("cancellation terminal projection=%#v err=%v", duplicate, err)
	}
}

func TestWorkflowContinuationSealsGraphAndConvergesDuplicateClaims(t *testing.T) {
	now := time.Now().UTC()
	cp := WorkflowContinuation{
		GraphID: "wf", GraphVersion: "v1", GraphDigest: "graph-digest",
		ParentRunID: "parent", RunID: "run", SessionID: "session", CausationID: "cause",
		Ref:         WorkflowContinuationRef{OwnerNamespace: "workflow"},
		PauseReason: PauseInputRequired, CreatedAt: now, ExpiresAt: now.Add(time.Hour),
		Steps:       []WorkflowStepContinuation{{StepID: "a", Status: StepStatusSucceeded, Attempts: 1}},
		PendingStep: "b", Generation: 1,
	}
	sealed, err := SealWorkflowContinuation(cp)
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	store := NewMemoryWorkflowContinuationStore("workflow")
	if err := store.Save(context.Background(), sealed); err != nil {
		t.Fatalf("save: %v", err)
	}
	request := WorkflowResumeRequest{Ref: sealed.Ref, GraphDigest: sealed.GraphDigest, Generation: sealed.Generation, IdempotencyKey: "resume-1"}
	first, err := store.ClaimResume(context.Background(), request)
	if err != nil {
		t.Fatalf("first claim: %v", err)
	}
	second, err := store.ClaimResume(context.Background(), request)
	if !errors.Is(err, ErrWorkflowContinuationInProgress) {
		t.Fatalf("duplicate claim: %v", err)
	}
	if !second.Duplicate || first.AttemptID != second.AttemptID {
		t.Fatalf("duplicate did not converge: first=%#v second=%#v", first, second)
	}
	conflict := request
	conflict.IdempotencyKey = "different"
	if _, err := store.ClaimResume(context.Background(), conflict); !errors.Is(err, ErrWorkflowContinuationConflict) {
		t.Fatalf("conflict error = %v", err)
	}
}

func TestWorkflowContinuationClaimTransitionsToRunningBeforeExecution(t *testing.T) {
	now := time.Now().UTC()
	continuation, err := SealWorkflowContinuation(WorkflowContinuation{
		Ref:     WorkflowContinuationRef{OwnerNamespace: "workflow"},
		GraphID: "wf-running", GraphVersion: "v1", GraphDigest: "graph",
		RunID: "run", SessionID: "session", PauseReason: PauseInputRequired,
		Generation: 1, CreatedAt: now, ExpiresAt: now.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	store := NewMemoryWorkflowContinuationStore("workflow")
	if err := store.Save(context.Background(), continuation); err != nil {
		t.Fatal(err)
	}
	request := WorkflowResumeRequest{Ref: continuation.Ref, GraphDigest: continuation.GraphDigest, Generation: continuation.Generation, IdempotencyKey: "resume"}
	claim, err := store.ClaimResume(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	running, err := store.MarkResumeRunning(context.Background(), claim)
	if err != nil {
		t.Fatalf("mark running: %v", err)
	}
	if running.State != WorkflowResumeClaimRunning || running.AttemptID != claim.AttemptID {
		t.Fatalf("running claim = %#v", running)
	}
	duplicate, err := store.ClaimResume(context.Background(), request)
	if !errors.Is(err, ErrWorkflowContinuationInProgress) || !duplicate.Duplicate || duplicate.State != WorkflowResumeClaimRunning {
		t.Fatalf("duplicate running claim = %#v err=%v", duplicate, err)
	}
}

func TestWorkflowContinuationRejectsConflictingCommittedProjection(t *testing.T) {
	now := time.Now().UTC()
	continuation, err := SealWorkflowContinuation(WorkflowContinuation{
		Ref: WorkflowContinuationRef{OwnerNamespace: "workflow"}, GraphID: "wf-result", GraphVersion: "v1", GraphDigest: "graph",
		RunID: "run", SessionID: "session", PauseReason: PauseInputRequired, Generation: 1, CreatedAt: now, ExpiresAt: now.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	store := NewMemoryWorkflowContinuationStore("workflow")
	if err := store.Save(context.Background(), continuation); err != nil {
		t.Fatal(err)
	}
	claim, err := store.ClaimResume(context.Background(), WorkflowResumeRequest{Ref: continuation.Ref, GraphDigest: continuation.GraphDigest, Generation: continuation.Generation, IdempotencyKey: "resume"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CommitResumeResult(context.Background(), claim, RunResult{RunID: "run", WorkflowID: "wf-result", WorkflowStatus: "succeeded", Steps: []StepResult{{StepID: "a", Status: StepStatusSucceeded}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CommitResumeResult(context.Background(), claim, RunResult{RunID: "run", WorkflowID: "wf-result", WorkflowStatus: "succeeded", Steps: []StepResult{{StepID: "a", Status: StepStatusFailed}}}); !errors.Is(err, ErrWorkflowContinuationConflict) {
		t.Fatalf("conflicting result error = %v", err)
	}
}

func TestWorkflowContinuationRejectsGraphDigestDrift(t *testing.T) {
	now := time.Now().UTC()
	cp := WorkflowContinuation{GraphID: "wf", GraphVersion: "v1", GraphDigest: "graph-digest", RunID: "run", SessionID: "session", PauseReason: PauseApprovalRequired, CreatedAt: now, ExpiresAt: now.Add(time.Hour), Generation: 1, Ref: WorkflowContinuationRef{OwnerNamespace: "workflow"}}
	sealed, err := SealWorkflowContinuation(cp)
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	sealed.GraphDigest = "changed"
	if err := sealed.Validate(); !errors.Is(err, ErrWorkflowContinuationCorrupt) {
		t.Fatalf("validate error = %v", err)
	}
}

func TestWorkflowContinuationRejectsOversizedEnvelope(t *testing.T) {
	now := time.Now().UTC()
	_, err := SealWorkflowContinuation(WorkflowContinuation{
		Ref:     WorkflowContinuationRef{OwnerNamespace: "workflow"},
		GraphID: "wf-oversized", GraphVersion: "v1", GraphDigest: "graph", RunID: "run", SessionID: "session",
		PauseReason: PauseInputRequired, PendingActionReference: strings.Repeat("x", WorkflowContinuationMaxBytes),
		Generation: 1, CreatedAt: now, ExpiresAt: now.Add(time.Hour),
	})
	if !errors.Is(err, ErrWorkflowContinuationMalformed) {
		t.Fatalf("oversized continuation error = %v", err)
	}
}

func TestFileWorkflowContinuationStoreReopensSequentially(t *testing.T) {
	root := t.TempDir()
	first := openTestFileWorkflowStore(t, root, "workflow")
	var err error
	now := time.Now().UTC()
	cp, err := SealWorkflowContinuation(WorkflowContinuation{Ref: WorkflowContinuationRef{OwnerNamespace: "workflow"}, GraphID: "wf-file", GraphVersion: "v1", GraphDigest: "graph", RunID: "run", SessionID: "session", PauseReason: PauseBudgetExhausted, OriginalDeadline: now.Add(time.Hour), Generation: 1, CreatedAt: now, ExpiresAt: now.Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if err = first.Save(context.Background(), cp); err != nil {
		t.Fatal(err)
	}
	if err = first.Close(); err != nil {
		t.Fatal(err)
	}
	second := openTestFileWorkflowStore(t, root, "workflow")
	loaded, err := second.Load(context.Background(), cp.Ref)
	if err != nil {
		t.Fatalf("reopen load: %v (checkpoint=%s)", err, cp.Ref.CheckpointID)
	}
	if loaded.Ref.CheckpointID != cp.Ref.CheckpointID || loaded.GraphDigest != cp.GraphDigest {
		t.Fatalf("reopen mismatch: %#v", loaded)
	}
}

func TestFileWorkflowContinuationStoreReopensClaimAndResult(t *testing.T) {
	root := t.TempDir()
	store := openTestFileWorkflowStore(t, root, "workflow")
	var err error
	now := time.Now().UTC()
	continuation, err := SealWorkflowContinuation(WorkflowContinuation{
		Ref:     WorkflowContinuationRef{OwnerNamespace: "workflow"},
		GraphID: "wf-file-claim", GraphVersion: "v1", GraphDigest: "graph",
		RunID: "run", SessionID: "session", PauseReason: PauseInputRequired,
		Generation: 1, CreatedAt: now, ExpiresAt: now.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(context.Background(), continuation); err != nil {
		t.Fatal(err)
	}
	req := WorkflowResumeRequest{Ref: continuation.Ref, GraphDigest: continuation.GraphDigest, Generation: continuation.Generation, IdempotencyKey: "resume"}
	claim, err := store.ClaimResume(context.Background(), req)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if _, err := store.CommitResumeResult(context.Background(), claim, RunResult{RunID: "run", WorkflowID: "wf-file-claim", WorkflowStatus: "succeeded"}); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopened := openTestFileWorkflowStore(t, root, "workflow")
	duplicate, err := reopened.ClaimResume(context.Background(), req)
	if err != nil {
		t.Fatalf("duplicate after reopen: %v", err)
	}
	if !duplicate.Duplicate || duplicate.Result == nil || duplicate.Result.WorkflowStatus != "succeeded" {
		t.Fatalf("reopened duplicate = %#v", duplicate)
	}
}

func TestFileWorkflowContinuationStoreRequiresRecoveryAfterReopenDuringClaim(t *testing.T) {
	root := t.TempDir()
	store := openTestFileWorkflowStore(t, root, "workflow")
	var err error
	now := time.Now().UTC()
	continuation, err := SealWorkflowContinuation(WorkflowContinuation{
		Ref:     WorkflowContinuationRef{OwnerNamespace: "workflow"},
		GraphID: "wf-file-recovery", GraphVersion: "v1", GraphDigest: "graph",
		RunID: "run", SessionID: "session", PauseReason: PauseApprovalRequired,
		Generation: 1, CreatedAt: now, ExpiresAt: now.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(context.Background(), continuation); err != nil {
		t.Fatal(err)
	}
	req := WorkflowResumeRequest{Ref: continuation.Ref, GraphDigest: continuation.GraphDigest, Generation: continuation.Generation, IdempotencyKey: "resume"}
	if _, err := store.ClaimResume(context.Background(), req); err != nil {
		t.Fatalf("claim: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopened := openTestFileWorkflowStore(t, root, "workflow")
	claim, err := reopened.ClaimResume(context.Background(), req)
	if !errors.Is(err, ErrWorkflowContinuationRecoveryRequired) {
		t.Fatalf("reopened claim error = %v", err)
	}
	if !claim.Duplicate || claim.State != WorkflowResumeClaimRecoveryRequired {
		t.Fatalf("reopened claim = %#v", claim)
	}
}

func TestFileWorkflowContinuationStoreRequiresRecoveryAfterReopenDuringRunningClaim(t *testing.T) {
	root := t.TempDir()
	store := openTestFileWorkflowStore(t, root, "workflow")
	var err error
	now := time.Now().UTC()
	continuation, err := SealWorkflowContinuation(WorkflowContinuation{
		Ref:     WorkflowContinuationRef{OwnerNamespace: "workflow"},
		GraphID: "wf-file-running-recovery", GraphVersion: "v1", GraphDigest: "graph",
		RunID: "run", SessionID: "session", PauseReason: PauseApprovalRequired,
		Generation: 1, CreatedAt: now, ExpiresAt: now.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(context.Background(), continuation); err != nil {
		t.Fatal(err)
	}
	req := WorkflowResumeRequest{Ref: continuation.Ref, GraphDigest: continuation.GraphDigest, Generation: continuation.Generation, IdempotencyKey: "resume"}
	claim, err := store.ClaimResume(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.MarkResumeRunning(context.Background(), claim); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopened := openTestFileWorkflowStore(t, root, "workflow")
	duplicate, err := reopened.ClaimResume(context.Background(), req)
	if !errors.Is(err, ErrWorkflowContinuationRecoveryRequired) || !duplicate.Duplicate || duplicate.State != WorkflowResumeClaimRecoveryRequired {
		t.Fatalf("running claim after reopen = %#v, err=%v", duplicate, err)
	}
	secondDuplicate, secondErr := reopened.ClaimResume(context.Background(), req)
	if !errors.Is(secondErr, ErrWorkflowContinuationRecoveryRequired) || !secondDuplicate.Duplicate || secondDuplicate.State != duplicate.State || secondDuplicate.AttemptID != duplicate.AttemptID {
		t.Fatalf("repeated recovery claim = %#v, err=%v; first = %#v", secondDuplicate, secondErr, duplicate)
	}
	dispatches := 0
	resumer := New(WithWorkflowContinuationStore(reopened, "workflow"), WithStepAdapter(DispatchAdapter{Runner: func(context.Context, string, Step, int) (StepOutput, error) {
		dispatches++
		return StepOutput{}, nil
	}}))
	if _, err := resumer.ResumeContinuation(context.Background(), req, nil); !errors.Is(err, ErrWorkflowContinuationRecoveryRequired) || dispatches != 0 {
		t.Fatalf("uncertain resume err=%v dispatches=%d", err, dispatches)
	}
}

func TestFileWorkflowContinuationStoreRejectsConcurrentOwner(t *testing.T) {
	root := t.TempDir()
	_ = openTestFileWorkflowStore(t, root, "workflow")
	second, err := NewFileWorkflowContinuationStore(root, "workflow")
	if !errors.Is(err, ErrWorkflowContinuationUnavailable) || second != nil {
		t.Fatalf("concurrent owner = %#v, err=%v", second, err)
	}
}

func TestWorkflowContinuationStoreProcessFenceHelper(t *testing.T) {
	root := os.Getenv("BAYMAX_WORKFLOW_STORE_FENCE_ROOT")
	if root == "" {
		return
	}
	store, err := NewFileWorkflowContinuationStore(root, "workflow")
	if err != nil {
		t.Fatalf("open fenced store: %v", err)
	}
	defer func() { _ = store.Close() }()
	_, _ = io.WriteString(os.Stdout, "ready\n")
	_, _ = io.Copy(io.Discard, os.Stdin)
}

func TestFileWorkflowContinuationStoreFencesAcrossProcesses(t *testing.T) {
	root := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=^TestWorkflowContinuationStoreProcessFenceHelper$")
	cmd.Env = append(os.Environ(), "BAYMAX_WORKFLOW_STORE_FENCE_ROOT="+root)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	line := make([]byte, len("ready\n"))
	if _, err := io.ReadFull(stdout, line); err != nil || string(line) != "ready\n" {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("child fence readiness = %q, err=%v", line, err)
	}
	if second, err := NewFileWorkflowContinuationStore(root, "workflow"); !errors.Is(err, ErrWorkflowContinuationUnavailable) || second != nil {
		_ = stdin.Close()
		_ = cmd.Wait()
		t.Fatalf("cross-process concurrent owner = %#v, err=%v", second, err)
	}
	if err := stdin.Close(); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("fence helper: %v", err)
	}
	if successor, err := NewFileWorkflowContinuationStore(root, "workflow"); err != nil {
		t.Fatalf("sequential cross-process reopen: %v", err)
	} else if err := successor.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestWorkflowContinuationCapabilitiesRequireActiveFileStoreFence(t *testing.T) {
	store := openTestFileWorkflowStore(t, t.TempDir(), "workflow")
	engine := New(WithWorkflowContinuationStore(store, "workflow"))
	if _, err := engine.ContinuationCapabilities(context.Background()); err != nil {
		t.Fatalf("active fenced capability error = %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := engine.ContinuationCapabilities(context.Background()); !errors.Is(err, ErrWorkflowContinuationUnavailable) {
		t.Fatalf("closed store capability error = %v", err)
	}
}

func TestFileWorkflowContinuationStoreFailsClosedWhenPersistenceCannotCommit(t *testing.T) {
	store := openTestFileWorkflowStore(t, t.TempDir(), "workflow")
	store.root = store.path("not-a-directory")
	if err := os.WriteFile(store.root, []byte("blocked"), 0o600); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	continuation, err := SealWorkflowContinuation(WorkflowContinuation{
		Ref: WorkflowContinuationRef{OwnerNamespace: "workflow"}, GraphID: "wf-store-fail", GraphVersion: "v1", GraphDigest: "graph",
		RunID: "run", SessionID: "session", PauseReason: PauseInputRequired, Generation: 1, CreatedAt: now, ExpiresAt: now.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(context.Background(), continuation); !errors.Is(err, ErrWorkflowContinuationUnavailable) {
		t.Fatalf("save error = %v", err)
	}
	request := WorkflowResumeRequest{Ref: continuation.Ref, GraphDigest: continuation.GraphDigest, Generation: continuation.Generation, IdempotencyKey: "resume"}
	if _, err := store.ClaimResume(context.Background(), request); !errors.Is(err, ErrWorkflowContinuationUnavailable) {
		t.Fatalf("failed save retained claimable checkpoint: %v", err)
	}
	if _, err := store.ClaimResume(context.Background(), request); !errors.Is(err, ErrWorkflowContinuationUnavailable) {
		t.Fatalf("failed persistence changed retry outcome: %v", err)
	}
}

func TestEngineContinuationSupportsEveryDeclaredPauseReason(t *testing.T) {
	for _, reason := range []WorkflowPauseReason{PauseInputRequired, PauseApprovalRequired, PauseBudgetExhausted} {
		t.Run(string(reason), func(t *testing.T) {
			store := NewMemoryWorkflowContinuationStore("workflow")
			engine := New(WithWorkflowContinuationStore(store, "workflow"), WithStepAdapter(DispatchAdapter{Runner: func(context.Context, string, Step, int) (StepOutput, error) {
				return StepOutput{}, PauseError{Reason: reason, PendingActionReference: "pending"}
			}}))
			result, err := engine.Run(context.Background(), RunRequest{RunID: "run-" + string(reason), SessionID: "session", DSL: Definition{WorkflowID: "wf-" + string(reason), Steps: []Step{{StepID: "pause", Kind: StepKindRunner}}}})
			if !errors.Is(err, ErrWorkflowPaused) || result.WorkflowCheckpoint == nil || result.WorkflowStatus != string(reason) {
				t.Fatalf("pause=%s result=%#v err=%v", reason, result, err)
			}
		})
	}
}

func TestEngineResumeContinuationCreatesSuccessorCheckpointOnSecondPause(t *testing.T) {
	store := NewMemoryWorkflowContinuationStore("workflow")
	calls := 0
	engine := New(
		WithWorkflowContinuationStore(store, "workflow"),
		WithStepAdapter(DispatchAdapter{Runner: func(context.Context, string, Step, int) (StepOutput, error) {
			calls++
			switch calls {
			case 1:
				return StepOutput{}, PauseError{Reason: PauseInputRequired, PendingActionReference: "input-1"}
			case 2:
				return StepOutput{}, PauseError{Reason: PauseApprovalRequired, PendingActionReference: "approval-1"}
			default:
				return StepOutput{}, nil
			}
		}}),
	)
	request := RunRequest{RunID: "run-chain", SessionID: "session-chain", DSL: Definition{WorkflowID: "wf-chain", Steps: []Step{{StepID: "step", Kind: StepKindRunner}}}}
	first, err := engine.Run(context.Background(), request)
	if !errors.Is(err, ErrWorkflowPaused) {
		t.Fatalf("first pause: %v", err)
	}
	resume := WorkflowResumeRequest{Ref: *first.WorkflowCheckpoint, GraphDigest: first.WorkflowGraphDigest, Generation: first.WorkflowCheckpointGeneration, IdempotencyKey: "resume-first"}
	second, err := engine.ResumeContinuation(context.Background(), resume, nil)
	if !errors.Is(err, ErrWorkflowPaused) {
		t.Fatalf("second pause: %v", err)
	}
	if second.WorkflowCheckpoint == nil || second.WorkflowCheckpoint.CheckpointID == first.WorkflowCheckpoint.CheckpointID || second.WorkflowCheckpointGeneration != first.WorkflowCheckpointGeneration+1 {
		t.Fatalf("successor checkpoint = %#v first=%#v", second.WorkflowCheckpoint, first.WorkflowCheckpoint)
	}
	successor, err := store.Load(context.Background(), *second.WorkflowCheckpoint)
	if err != nil {
		t.Fatalf("load successor: %v", err)
	}
	if successor.PredecessorCheckpointID != first.WorkflowCheckpoint.CheckpointID || successor.PauseReason != PauseApprovalRequired {
		t.Fatalf("successor = %#v", successor)
	}
	duplicate, err := engine.ResumeContinuation(context.Background(), resume, nil)
	if err != nil || duplicate.WorkflowCheckpoint == nil || duplicate.WorkflowCheckpoint.CheckpointID != second.WorkflowCheckpoint.CheckpointID || calls != 2 {
		t.Fatalf("duplicate predecessor resume = %#v err=%v calls=%d", duplicate, err, calls)
	}
}

func TestWorkflowContinuationRunStreamParity(t *testing.T) {
	newEngine := func(store WorkflowContinuationStore) *Engine {
		paused := false
		return New(
			WithWorkflowContinuationStore(store, "workflow"),
			WithStepAdapter(DispatchAdapter{Runner: func(context.Context, string, Step, int) (StepOutput, error) {
				if !paused {
					paused = true
					return StepOutput{}, PauseError{Reason: PauseBudgetExhausted, PendingActionReference: "budget-boundary"}
				}
				return StepOutput{}, nil
			}}),
		)
	}
	definition := Definition{WorkflowID: "wf-parity", Steps: []Step{{StepID: "step", Kind: StepKindRunner}}}
	runStore := NewMemoryWorkflowContinuationStore("workflow")
	runEngine := newEngine(runStore)
	runPaused, runErr := runEngine.Run(context.Background(), RunRequest{RunID: "run-parity", SessionID: "session-parity", DSL: definition})
	if !errors.Is(runErr, ErrWorkflowPaused) {
		t.Fatalf("Run pause error = %v", runErr)
	}

	streamStore := NewMemoryWorkflowContinuationStore("workflow")
	streamEngine := newEngine(streamStore)
	streamPauseEvents := 0
	streamPaused, streamErr := streamEngine.Stream(context.Background(), RunRequest{RunID: "run-parity", SessionID: "session-parity", DSL: definition}, func(event StreamEvent) error {
		if event.Kind == "workflow.paused" {
			streamPauseEvents++
		}
		return nil
	})
	if !errors.Is(streamErr, ErrWorkflowPaused) {
		t.Fatalf("Stream pause error = %v", streamErr)
	}
	if streamPauseEvents != 1 || runPaused.WorkflowStatus != streamPaused.WorkflowStatus || runPaused.WorkflowGraphDigest != streamPaused.WorkflowGraphDigest || runPaused.WorkflowCheckpointGeneration != streamPaused.WorkflowCheckpointGeneration {
		t.Fatalf("pause parity run=%#v stream=%#v events=%d", runPaused, streamPaused, streamPauseEvents)
	}

	runResumed, runResumeErr := runEngine.ResumeContinuation(context.Background(), WorkflowResumeRequest{Ref: *runPaused.WorkflowCheckpoint, GraphDigest: runPaused.WorkflowGraphDigest, Generation: runPaused.WorkflowCheckpointGeneration, IdempotencyKey: "resume"}, nil)
	if runResumeErr != nil {
		t.Fatalf("Run resume error = %v", runResumeErr)
	}
	streamCompletedEvents := 0
	streamResumed, streamResumeErr := streamEngine.ResumeContinuation(context.Background(), WorkflowResumeRequest{Ref: *streamPaused.WorkflowCheckpoint, GraphDigest: streamPaused.WorkflowGraphDigest, Generation: streamPaused.WorkflowCheckpointGeneration, IdempotencyKey: "resume"}, func(event StreamEvent) error {
		if event.Kind == "workflow.completed" {
			streamCompletedEvents++
		}
		return nil
	})
	if streamResumeErr != nil {
		t.Fatalf("Stream resume error = %v", streamResumeErr)
	}
	if streamCompletedEvents != 1 || runResumed.WorkflowStatus != streamResumed.WorkflowStatus || runResumed.RunID != streamResumed.RunID || runResumed.WorkflowID != streamResumed.WorkflowID {
		t.Fatalf("resume parity run=%#v stream=%#v events=%d", runResumed, streamResumed, streamCompletedEvents)
	}
}

func TestWorkflowPauseControllerStopsApprovalBeforeEffectDispatch(t *testing.T) {
	store := NewMemoryWorkflowContinuationStore("workflow")
	dispatches := 0
	engine := New(
		WithWorkflowContinuationStore(store, "workflow"),
		WithPauseController(WorkflowPauseControllerFunc(func(context.Context, WorkflowPauseBoundary) error {
			return PauseError{Reason: PauseApprovalRequired, PendingActionReference: "effect:approval"}
		})),
		WithStepAdapter(DispatchAdapter{Tool: func(context.Context, string, Step, int) (StepOutput, error) {
			dispatches++
			return StepOutput{}, nil
		}}),
	)
	result, err := engine.Run(context.Background(), RunRequest{RunID: "run-approval-boundary", SessionID: "session", DSL: Definition{WorkflowID: "wf-approval-boundary", Steps: []Step{{StepID: "effect", Kind: StepKindTool}}}})
	if !errors.Is(err, ErrWorkflowPaused) || result.WorkflowStatus != string(PauseApprovalRequired) {
		t.Fatalf("pause result=%#v err=%v", result, err)
	}
	if dispatches != 0 {
		t.Fatalf("approval pause dispatched effect %d times", dispatches)
	}
	if result.WorkflowCheckpoint == nil {
		t.Fatal("missing source-owned checkpoint")
	}
}

func TestWorkflowBudgetPausePreservesConsumedBudgetAndOriginalDeadline(t *testing.T) {
	store := NewMemoryWorkflowContinuationStore("workflow")
	deadline := time.Now().UTC().Add(30 * time.Minute).Round(0)
	engine := New(
		WithWorkflowContinuationStore(store, "workflow"),
		WithPauseController(WorkflowPauseControllerFunc(func(context.Context, WorkflowPauseBoundary) error {
			return PauseError{Reason: PauseBudgetExhausted, PendingActionReference: "budget-boundary", ConsumedBudget: 7, OriginalDeadline: deadline}
		})),
		WithStepAdapter(DispatchAdapter{Runner: func(context.Context, string, Step, int) (StepOutput, error) {
			return StepOutput{}, nil
		}}),
	)
	result, err := engine.Run(context.Background(), RunRequest{RunID: "run-budget", SessionID: "session", DSL: Definition{WorkflowID: "wf-budget", Steps: []Step{{StepID: "step", Kind: StepKindRunner}}}})
	if !errors.Is(err, ErrWorkflowPaused) {
		t.Fatalf("pause: %v", err)
	}
	continuation, err := store.Load(context.Background(), *result.WorkflowCheckpoint)
	if err != nil {
		t.Fatal(err)
	}
	if continuation.ConsumedBudget != 7 || !continuation.OriginalDeadline.Equal(deadline) {
		t.Fatalf("budget state = %#v", continuation)
	}
	if err := continuation.Validate(); err != nil {
		t.Fatalf("budget continuation validation: %v", err)
	}
}

func TestWorkflowBudgetContinuationRejectsResumeAfterOriginalDeadline(t *testing.T) {
	store := NewMemoryWorkflowContinuationStore("workflow")
	now := time.Now().UTC()
	continuation, err := SealWorkflowContinuation(WorkflowContinuation{
		Ref:     WorkflowContinuationRef{OwnerNamespace: "workflow"},
		GraphID: "wf-expired-budget", GraphVersion: "v1", GraphDigest: workflowDefinitionDigest(Definition{WorkflowID: "wf-expired-budget", Steps: []Step{{StepID: "step", Kind: StepKindRunner}}}),
		RunID: "run", SessionID: "session", PauseReason: PauseBudgetExhausted, ConsumedBudget: 4, OriginalDeadline: now.Add(-time.Minute),
		Definition: Definition{WorkflowID: "wf-expired-budget", Steps: []Step{{StepID: "step", Kind: StepKindRunner}}}, Generation: 1, CreatedAt: now.Add(-2 * time.Minute), ExpiresAt: now.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(context.Background(), continuation); err != nil {
		t.Fatal(err)
	}
	engine := New(WithWorkflowContinuationStore(store, "workflow"), WithStepAdapter(DispatchAdapter{Runner: func(context.Context, string, Step, int) (StepOutput, error) {
		return StepOutput{}, errors.New("expired budget resumed")
	}}))
	_, err = engine.ResumeContinuation(context.Background(), WorkflowResumeRequest{Ref: continuation.Ref, GraphDigest: continuation.GraphDigest, Generation: continuation.Generation, IdempotencyKey: "resume"}, nil)
	if !errors.Is(err, ErrWorkflowContinuationExpired) {
		t.Fatalf("expired deadline resume error = %v", err)
	}
}

func TestWorkflowContinuationRetainsCompletedEffectAndOutputEvidence(t *testing.T) {
	store := NewMemoryWorkflowContinuationStore("workflow")
	engine := New(
		WithWorkflowContinuationStore(store, "workflow"),
		WithStepAdapter(DispatchAdapter{Runner: func(_ context.Context, _ string, step Step, _ int) (StepOutput, error) {
			if step.StepID == "completed" {
				return StepOutput{Payload: map[string]any{"receipt": "ok"}, EffectReference: "effect:completed"}, nil
			}
			return StepOutput{}, PauseError{Reason: PauseInputRequired, PendingActionReference: "input"}
		}}),
	)
	request := RunRequest{
		RunID:     "run-evidence",
		SessionID: "session",
		DSL: Definition{WorkflowID: "wf-evidence", Steps: []Step{
			{StepID: "completed", Kind: StepKindRunner},
			{StepID: "pause", Kind: StepKindRunner, DependsOn: []string{"completed"}},
		}},
	}
	result, err := engine.Run(context.Background(), request)
	if !errors.Is(err, ErrWorkflowPaused) {
		t.Fatalf("pause: %v", err)
	}
	continuation, err := store.Load(context.Background(), *result.WorkflowCheckpoint)
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range continuation.Steps {
		if step.StepID == "completed" {
			if step.EffectReference != "effect:completed" || step.OutputDigest == "" {
				t.Fatalf("completed evidence = %#v", step)
			}
			return
		}
	}
	t.Fatal("completed step evidence missing")
}
