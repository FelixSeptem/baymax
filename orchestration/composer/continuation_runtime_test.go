package composer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/context/hostprojection"
	"github.com/FelixSeptem/baymax/orchestration/scheduler"
)

func TestComposerResumeChildContinuationUsesExistingTaskAndDeduplicates(t *testing.T) {
	store := NewMemoryChildContinuationStore("child")
	continuation := testChildContinuation(t)
	sched := mustScheduler(t, scheduler.NewMemoryStore())
	if _, err := sched.Enqueue(context.Background(), continuation.Claimed.Record.Task); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	claimed, ok, err := sched.Claim(context.Background(), "worker")
	if err != nil || !ok {
		t.Fatalf("Claim() = %#v, %v", claimed, err)
	}
	continuation.Claimed = claimed
	continuation, err = SealChildContinuation(continuation)
	if err != nil {
		t.Fatalf("re-seal continuation error = %v", err)
	}
	if err := store.Save(context.Background(), continuation); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	var calls int
	executor := ChildContinuationExecutorFunc(func(_ context.Context, got ChildContinuation) (scheduler.TerminalCommit, error) {
		calls++
		if got.Claimed.Record.Task.TaskID != continuation.Claimed.Record.Task.TaskID {
			t.Fatalf("task id = %q, want %q", got.Claimed.Record.Task.TaskID, continuation.Claimed.Record.Task.TaskID)
		}
		return scheduler.TerminalCommit{TaskID: got.Claimed.Record.Task.TaskID, AttemptID: got.Claimed.Attempt.AttemptID, Status: scheduler.TaskStateSucceeded}, nil
	})
	comp := &Composer{scheduler: sched, childContinuationStore: store, childContinuationNamespace: "child", childContinuationExecutor: executor, now: time.Now, runStat: map[string]*runStat{}}
	request := ChildResumeRequest{Ref: continuation.Ref, GraphDigest: continuation.GraphDigest, Generation: continuation.Generation, Target: continuation.Target, IdempotencyKey: "resume-1", AuthorityDigest: continuation.AuthorityDigest, PlanDigest: continuation.PlanDigest, WriteScopeDigest: continuation.WriteScopeDigest, EffectNamespace: continuation.EffectNamespace, OccurrenceID: continuation.OccurrenceID, CausationID: continuation.CausationID}
	first, err := comp.ResumeChildContinuation(context.Background(), request)
	if err != nil {
		t.Fatalf("first resume error = %v", err)
	}
	if first.Commit.Status != scheduler.TaskStateSucceeded || calls != 1 {
		t.Fatalf("first result = %#v, calls=%d", first, calls)
	}
	second, err := comp.ResumeChildContinuation(context.Background(), request)
	if err != nil {
		t.Fatalf("duplicate resume error = %v", err)
	}
	if !second.AsyncAccepted && second.Commit.Status != scheduler.TaskStateSucceeded || calls != 1 || !second.CommitMeta.Duplicate {
		t.Fatalf("duplicate result = %#v, calls=%d", second, calls)
	}
	if _, err := comp.ResumeChildContinuation(context.Background(), ChildResumeRequest{Ref: continuation.Ref, GraphDigest: continuation.GraphDigest, Generation: continuation.Generation, Target: continuation.Target, IdempotencyKey: "different"}); !errors.Is(err, ErrChildContinuationConflict) {
		t.Fatalf("conflicting resume error = %v", err)
	}
}

func TestComposerResumeChildContinuationFailsClosedWithoutExecutor(t *testing.T) {
	store := NewMemoryChildContinuationStore("child")
	continuation := testChildContinuation(t)
	if err := store.Save(context.Background(), continuation); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	comp := &Composer{scheduler: mustScheduler(t, scheduler.NewMemoryStore()), childContinuationStore: store, childContinuationNamespace: "child", now: time.Now}
	_, err := comp.ResumeChildContinuation(context.Background(), ChildResumeRequest{Ref: continuation.Ref, GraphDigest: continuation.GraphDigest, Generation: continuation.Generation, Target: continuation.Target, IdempotencyKey: "resume-1"})
	if !errors.Is(err, ErrChildContinuationUnsupported) {
		t.Fatalf("resume error = %v", err)
	}
}

func TestComposerResumeChildContinuationRejectsExpiredLease(t *testing.T) {
	store := NewMemoryChildContinuationStore("child")
	continuation := testChildContinuation(t)
	continuation.Claimed.Attempt.LeaseExpiresAt = time.Now().Add(-time.Second)
	continuation, err := SealChildContinuation(continuation)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(context.Background(), continuation); err != nil {
		t.Fatal(err)
	}
	sched := mustScheduler(t, scheduler.NewMemoryStore())
	if _, err := sched.Enqueue(context.Background(), continuation.Claimed.Record.Task); err != nil {
		t.Fatal(err)
	}
	claimed, ok, err := sched.Claim(context.Background(), "worker")
	if err != nil || !ok {
		t.Fatalf("Claim() = %#v, %v", claimed, err)
	}
	comp := &Composer{scheduler: sched, childContinuationStore: store, childContinuationNamespace: "child", childContinuationExecutor: ChildContinuationExecutorFunc(func(context.Context, ChildContinuation) (scheduler.TerminalCommit, error) {
		t.Fatal("executor activated")
		return scheduler.TerminalCommit{}, nil
	}), now: func() time.Time { return time.Now().Add(time.Hour) }, runStat: map[string]*runStat{}}
	_, err = comp.ResumeChildContinuation(context.Background(), ChildResumeRequest{Ref: continuation.Ref, GraphDigest: continuation.GraphDigest, Generation: continuation.Generation, Target: continuation.Target, IdempotencyKey: "resume"})
	if !errors.Is(err, ErrChildContinuationRecoveryRequired) {
		t.Fatalf("lease error = %v", err)
	}
}

func TestComposerResumeChildContinuationRejectsForeignProjectionBeforeResolverOrClaim(t *testing.T) {
	store := NewMemoryChildContinuationStore("child")
	continuation := testChildContinuation(t)
	continuation.HostContextBinding = &hostprojection.SourceBinding{
		Form:               hostprojection.SourceFormChild,
		OwnerNamespace:     continuation.Ref.OwnerNamespace,
		SourceID:           continuation.RunID,
		NodeID:             continuation.Claimed.Record.Task.TaskID,
		ContinuationID:     continuation.Claimed.Attempt.AttemptID,
		ContinuationDigest: continuation.GraphDigest,
		Generation:         continuation.Generation,
	}
	var err error
	continuation, err = SealChildContinuation(continuation)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(context.Background(), continuation); err != nil {
		t.Fatal(err)
	}

	var resolverCalls int
	resolver := hostContextResolverFunc(func(context.Context, hostprojection.SourceReference) (hostprojection.Material, error) {
		resolverCalls++
		return hostprojection.Material{Role: "private", Content: "must-not-resolve"}, nil
	})
	foreignBinding := *continuation.HostContextBinding
	foreignBinding.NodeID = "another-child-task"
	projection := hostprojection.Descriptor{
		SchemaVersion: hostprojection.SchemaVersionV1,
		ProjectionID:  "projection-child-foreign", ProjectionVersion: 1,
		CapabilityDigest: "capability-child", Capabilities: []hostprojection.Capability{hostprojection.CapabilityRead},
		SourceReferences: []hostprojection.SourceReference{{Reference: "source-child", Digest: "source-digest"}},
		Binding:          &foreignBinding,
	}
	projection.Digest, err = hostprojection.CanonicalDigest(projection)
	if err != nil {
		t.Fatal(err)
	}
	request := ChildResumeRequest{
		Ref: continuation.Ref, GraphDigest: continuation.GraphDigest, Generation: continuation.Generation,
		Target: continuation.Target, IdempotencyKey: "resume-foreign-projection",
		HostContextProjection: &projection,
		HostContextResolver:   resolver,
	}
	comp := &Composer{scheduler: mustScheduler(t, scheduler.NewMemoryStore()), childContinuationStore: store, childContinuationNamespace: "child", now: time.Now}
	_, err = comp.ResumeChildContinuation(context.Background(), request)
	if !errors.Is(err, ErrChildContinuationConflict) {
		t.Fatalf("foreign projection error = %v, want conflict", err)
	}
	if resolverCalls != 0 {
		t.Fatalf("resolver calls = %d, want 0", resolverCalls)
	}
	if _, err := store.ClaimResume(context.Background(), request); err != nil {
		t.Fatalf("foreign projection consumed the resume claim: %v", err)
	}
}

func TestComposerResumeChildContinuationResolvesBoundContextBeforeExecution(t *testing.T) {
	store := NewMemoryChildContinuationStore("child")
	continuation := testChildContinuation(t)
	continuation.CompletedEffectRefs = []string{"effect-already-committed"}
	continuation.HostContextBinding = &hostprojection.SourceBinding{
		Form: hostprojection.SourceFormChild, OwnerNamespace: continuation.Ref.OwnerNamespace,
		SourceID: continuation.RunID, NodeID: continuation.Claimed.Record.Task.TaskID,
		ContinuationID: continuation.Claimed.Attempt.AttemptID, ContinuationDigest: continuation.GraphDigest,
		Generation: continuation.Generation,
	}
	sched := mustScheduler(t, scheduler.NewMemoryStore())
	if _, err := sched.Enqueue(context.Background(), continuation.Claimed.Record.Task); err != nil {
		t.Fatal(err)
	}
	claimed, ok, err := sched.Claim(context.Background(), "worker")
	if err != nil || !ok {
		t.Fatalf("Claim() = %#v, %v", claimed, err)
	}
	continuation.Claimed = claimed
	continuation.HostContextBinding.NodeID = claimed.Record.Task.TaskID
	continuation.HostContextBinding.ContinuationID = claimed.Attempt.AttemptID
	continuation, err = SealChildContinuation(continuation)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(context.Background(), continuation); err != nil {
		t.Fatal(err)
	}
	projection := hostprojection.Descriptor{
		SchemaVersion: hostprojection.SchemaVersionV1, ProjectionID: "projection-child-valid", ProjectionVersion: 1,
		CapabilityDigest: "capability-child", Capabilities: []hostprojection.Capability{hostprojection.CapabilityRead},
		SourceReferences: []hostprojection.SourceReference{{Reference: "source-child", Digest: "source-digest"}},
		Binding:          continuation.HostContextBinding,
	}
	projection.Digest, err = hostprojection.CanonicalDigest(projection)
	if err != nil {
		t.Fatal(err)
	}
	var resolverCalls int
	resolver := hostContextResolverFunc(func(context.Context, hostprojection.SourceReference) (hostprojection.Material, error) {
		resolverCalls++
		return hostprojection.Material{Role: "private", Content: "authorized-context"}, nil
	})
	var received string
	executor := ChildContinuationContextExecutorFunc(func(_ context.Context, got ChildContinuation, resolution hostprojection.Resolution) (scheduler.TerminalCommit, error) {
		if len(got.CompletedEffectRefs) != 1 || got.CompletedEffectRefs[0] != "effect-already-committed" {
			t.Fatalf("completed effect refs = %v", got.CompletedEffectRefs)
		}
		received = resolution.Materials()[0].Content
		return scheduler.TerminalCommit{TaskID: got.Claimed.Record.Task.TaskID, AttemptID: got.Claimed.Attempt.AttemptID, Status: scheduler.TaskStateSucceeded}, nil
	})
	comp := &Composer{scheduler: sched, childContinuationStore: store, childContinuationNamespace: "child", childContextContinuationExecutor: executor, now: time.Now, runStat: map[string]*runStat{}}
	request := ChildResumeRequest{Ref: continuation.Ref, GraphDigest: continuation.GraphDigest, Generation: continuation.Generation, Target: continuation.Target, IdempotencyKey: "resume-context", HostContextProjection: &projection, HostContextResolver: resolver}
	result, err := comp.ResumeChildContinuation(context.Background(), request)
	if err != nil || result.Commit.Status != scheduler.TaskStateSucceeded {
		t.Fatalf("resume result = %#v, err=%v", result, err)
	}
	if resolverCalls != 1 || received != "authorized-context" {
		t.Fatalf("resolver calls=%d received=%q", resolverCalls, received)
	}
}

func TestComposerResumeChildContinuationRechecksSchedulerFenceAfterResolver(t *testing.T) {
	store := NewMemoryChildContinuationStore("child")
	continuation := testChildContinuation(t)
	sched := mustScheduler(t, scheduler.NewMemoryStore())
	if _, err := sched.Enqueue(context.Background(), continuation.Claimed.Record.Task); err != nil {
		t.Fatal(err)
	}
	claimed, ok, err := sched.Claim(context.Background(), "worker")
	if err != nil || !ok {
		t.Fatalf("Claim() = %#v, %v", claimed, err)
	}
	continuation.Claimed = claimed
	continuation.HostContextBinding = &hostprojection.SourceBinding{
		Form: hostprojection.SourceFormChild, OwnerNamespace: "child", SourceID: continuation.RunID,
		NodeID: claimed.Record.Task.TaskID, ContinuationID: claimed.Attempt.AttemptID,
		ContinuationDigest: continuation.GraphDigest, Generation: continuation.Generation,
	}
	continuation, err = SealChildContinuation(continuation)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(context.Background(), continuation); err != nil {
		t.Fatal(err)
	}
	projection := hostprojection.Descriptor{
		SchemaVersion: hostprojection.SchemaVersionV1, ProjectionID: "projection-child-fence", ProjectionVersion: 1,
		CapabilityDigest: "capability-child", Capabilities: []hostprojection.Capability{hostprojection.CapabilityRead},
		SourceReferences: []hostprojection.SourceReference{{Reference: "opaque://source-child-fence", Digest: "source-child-fence"}},
		Binding:          continuation.HostContextBinding,
	}
	projection.Digest, err = hostprojection.CanonicalDigest(projection)
	if err != nil {
		t.Fatal(err)
	}
	resolver := hostContextResolverFunc(func(context.Context, hostprojection.SourceReference) (hostprojection.Material, error) {
		if _, err := sched.Complete(context.Background(), scheduler.TerminalCommit{TaskID: claimed.Record.Task.TaskID, AttemptID: claimed.Attempt.AttemptID, Status: scheduler.TaskStateSucceeded}); err != nil {
			t.Fatalf("resolver terminal mutation: %v", err)
		}
		return hostprojection.Material{Role: "private", Content: "authorized-before-fence"}, nil
	})
	executorCalls := 0
	comp := &Composer{
		scheduler: sched, childContinuationStore: store, childContinuationNamespace: "child", now: time.Now,
		childContextContinuationExecutor: ChildContinuationContextExecutorFunc(func(context.Context, ChildContinuation, hostprojection.Resolution) (scheduler.TerminalCommit, error) {
			executorCalls++
			return scheduler.TerminalCommit{}, nil
		}),
	}
	request := ChildResumeRequest{Ref: continuation.Ref, GraphDigest: continuation.GraphDigest, Generation: continuation.Generation, Target: continuation.Target, IdempotencyKey: "resume-fence", HostContextProjection: &projection, HostContextResolver: resolver}
	if _, err := comp.ResumeChildContinuation(context.Background(), request); !errors.Is(err, ErrChildContinuationRecoveryRequired) {
		t.Fatalf("fenced resume error = %v, want recovery required", err)
	}
	if executorCalls != 0 {
		t.Fatalf("executor calls = %d, want 0", executorCalls)
	}
	if _, err := store.ClaimResume(context.Background(), request); err != nil {
		t.Fatalf("resolver fence consumed resume claim: %v", err)
	}
}

type hostContextResolverFunc func(context.Context, hostprojection.SourceReference) (hostprojection.Material, error)

type ChildContinuationContextExecutorFunc func(context.Context, ChildContinuation, hostprojection.Resolution) (scheduler.TerminalCommit, error)

func (f ChildContinuationContextExecutorFunc) ResumeChildContinuationWithContext(ctx context.Context, continuation ChildContinuation, resolution hostprojection.Resolution) (scheduler.TerminalCommit, error) {
	return f(ctx, continuation, resolution)
}

func (f hostContextResolverFunc) ResolveHostContext(ctx context.Context, ref hostprojection.SourceReference) (hostprojection.Material, error) {
	return f(ctx, ref)
}

func TestComposerPersistChildPausePreservesEachSafePoint(t *testing.T) {
	for _, reason := range []ChildPauseReason{ChildPauseInputRequired, ChildPauseApprovalRequired, ChildPauseBudgetExhausted} {
		t.Run(string(reason), func(t *testing.T) {
			store := NewMemoryChildContinuationStore("child")
			comp := &Composer{childContinuationStore: store}
			base := testChildContinuation(t)
			paused, err := comp.PersistChildPause(context.Background(), base, ChildPauseError{Reason: reason, PendingActionReference: "pending", ConsumedBudget: 7, OriginalDeadline: base.OriginalDeadline})
			if err != nil {
				t.Fatal(err)
			}
			if paused.PauseReason != reason || paused.PendingActionReference != "pending" || paused.ConsumedBudget != 7 {
				t.Fatalf("pause = %#v", paused)
			}
		})
	}
}

func TestDispatchChildPersistsExplicitLocalPauseBeforeTerminalCommit(t *testing.T) {
	store := NewMemoryChildContinuationStore("child")
	comp := &Composer{scheduler: mustScheduler(t, scheduler.NewMemoryStore()), childContinuationStore: store, childContinuationNamespace: "child", childWorkerID: "worker", childPollInterval: time.Millisecond, now: time.Now, runStat: map[string]*runStat{}}
	req := ChildDispatchRequest{Target: ChildTargetLocal, Task: scheduler.Task{TaskID: "paused-child", RunID: "run", ParentRunID: "parent"}, ParentRemainingBudget: time.Second, ChildTimeout: time.Second, ContinuationRunner: ChildContinuationRunnerFunc(func(context.Context, scheduler.ClaimedTask) (map[string]any, ChildPauseError, error) {
		return nil, ChildPauseError{Reason: ChildPauseApprovalRequired, PendingActionReference: "approval"}, nil
	})}
	result, err := comp.DispatchChild(context.Background(), req)
	if err != nil || result.Continuation == nil || result.PauseReason != ChildPauseApprovalRequired {
		t.Fatalf("pause result = %#v, err=%v", result, err)
	}
	loaded, err := store.Load(context.Background(), *result.Continuation)
	if err != nil || loaded.PauseReason != ChildPauseApprovalRequired || loaded.PendingActionReference != "approval" {
		t.Fatalf("loaded pause = %#v, err=%v", loaded, err)
	}
}

func TestDispatchChildPersistsOwnerBoundHostContextWithPause(t *testing.T) {
	store := NewMemoryChildContinuationStore("child")
	comp := &Composer{scheduler: mustScheduler(t, scheduler.NewMemoryStore()), childContinuationStore: store, childContinuationNamespace: "child", childWorkerID: "worker", childPollInterval: time.Millisecond, now: time.Now, runStat: map[string]*runStat{}}
	binding := hostprojection.SourceBinding{Form: hostprojection.SourceFormChild}
	req := ChildDispatchRequest{
		Target: ChildTargetLocal, Task: scheduler.Task{TaskID: "paused-child-context", RunID: "run", ParentRunID: "parent"},
		ParentRemainingBudget: time.Second, ChildTimeout: time.Second, HostContextBinding: &binding,
		ContinuationRunner: ChildContinuationRunnerFunc(func(context.Context, scheduler.ClaimedTask) (map[string]any, ChildPauseError, error) {
			return nil, ChildPauseError{Reason: ChildPauseInputRequired, PendingActionReference: "input"}, nil
		}),
	}
	result, err := comp.DispatchChild(context.Background(), req)
	if err != nil || result.Continuation == nil {
		t.Fatalf("pause result = %#v, err=%v", result, err)
	}
	loaded, err := store.Load(context.Background(), *result.Continuation)
	if err != nil {
		t.Fatal(err)
	}
	want := hostprojection.SourceBinding{
		Form: hostprojection.SourceFormChild, OwnerNamespace: "child", SourceID: "run",
		NodeID: "paused-child-context", ContinuationID: loaded.Claimed.Attempt.AttemptID, ContinuationDigest: "paused-child-context", Generation: 1,
	}
	if loaded.HostContextBinding == nil || *loaded.HostContextBinding != want {
		t.Fatalf("host context binding = %#v, want %#v", loaded.HostContextBinding, want)
	}
}
