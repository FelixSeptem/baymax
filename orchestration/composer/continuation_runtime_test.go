package composer

import (
	"context"
	"errors"
	"testing"
	"time"

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
