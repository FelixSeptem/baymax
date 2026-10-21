package composer

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/orchestration/scheduler"
)

func testChildContinuation(t *testing.T) ChildContinuation {
	t.Helper()
	now := time.Now().UTC()
	task := scheduler.Task{TaskID: "child-task", RunID: "parent-run", ParentRunID: "root-run", MaxAttempts: 2}
	record := scheduler.TaskRecord{Task: task, State: scheduler.TaskStateRunning, CurrentAttempt: "attempt-1", CreatedAt: now, UpdatedAt: now}
	claimed := scheduler.ClaimedTask{Record: record, Attempt: scheduler.Attempt{AttemptID: "attempt-1", Attempt: 1, Status: scheduler.AttemptStatusRunning, StartedAt: now, LeaseExpiresAt: now.Add(time.Minute)}}
	continuation, err := SealChildContinuation(ChildContinuation{
		Ref:                    ChildContinuationRef{OwnerNamespace: "child"},
		Target:                 ChildTargetLocal,
		Claimed:                claimed,
		GraphID:                "fanout-graph",
		GraphVersion:           "v1",
		GraphDigest:            "graph-digest",
		ParentRunID:            "root-run",
		RunID:                  "parent-run",
		SessionID:              "session-1",
		CausationID:            "cause-1",
		AuthorityDigest:        "authority-1",
		PlanDigest:             "plan-1",
		WriteScopeDigest:       "scope-1",
		EffectNamespace:        "effect-1",
		OccurrenceID:           "occurrence-1",
		PauseReason:            ChildPauseInputRequired,
		PendingActionReference: "action-1",
		Generation:             1,
		CreatedAt:              now,
		ExpiresAt:              now.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("SealChildContinuation() error = %v", err)
	}
	return continuation
}

func TestMemoryChildContinuationStoreRoundTripsAndDeduplicates(t *testing.T) {
	store := NewMemoryChildContinuationStore("child")
	continuation := testChildContinuation(t)
	if err := store.Save(context.Background(), continuation); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	loaded, err := store.Load(context.Background(), continuation.Ref)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Ref.Digest != continuation.Ref.Digest || loaded.Claimed.Attempt.AttemptID != "attempt-1" {
		t.Fatalf("loaded continuation = %#v", loaded)
	}
	request := ChildResumeRequest{Ref: continuation.Ref, GraphDigest: continuation.GraphDigest, Generation: continuation.Generation, Target: ChildTargetLocal, IdempotencyKey: "resume-1"}
	claim, err := store.ClaimResume(context.Background(), request)
	if err != nil {
		t.Fatalf("ClaimResume() error = %v", err)
	}
	duplicate, err := store.ClaimResume(context.Background(), request)
	if !errors.Is(err, ErrChildContinuationInProgress) || !duplicate.Duplicate {
		t.Fatalf("duplicate claim = %#v, err=%v", duplicate, err)
	}
	if _, err := store.MarkResumeRunning(context.Background(), claim); err != nil {
		t.Fatalf("MarkResumeRunning() error = %v", err)
	}
	if _, err := store.ClaimResume(context.Background(), ChildResumeRequest{Ref: continuation.Ref, GraphDigest: continuation.GraphDigest, Generation: continuation.Generation, Target: ChildTargetLocal, IdempotencyKey: "different"}); !errors.Is(err, ErrChildContinuationConflict) {
		t.Fatalf("conflicting claim error = %v", err)
	}
}

func TestFileChildContinuationStoreReopensAcrossInstances(t *testing.T) {
	root := t.TempDir()
	store, err := NewFileChildContinuationStore(filepath.Join(root, "child"), "child")
	if err != nil {
		t.Fatalf("NewFileChildContinuationStore() error = %v", err)
	}
	continuation := testChildContinuation(t)
	if err := store.Save(context.Background(), continuation); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	reopened, err := NewFileChildContinuationStore(filepath.Join(root, "child"), "child")
	if err != nil {
		t.Fatalf("reopen error = %v", err)
	}
	loaded, err := reopened.Load(context.Background(), continuation.Ref)
	if err != nil {
		t.Fatalf("reopened Load() error = %v", err)
	}
	if loaded.Ref.Digest != continuation.Ref.Digest {
		t.Fatalf("reopened digest = %q, want %q", loaded.Ref.Digest, continuation.Ref.Digest)
	}
}

func TestChildContinuationRejectsForeignAndOversizedMaterial(t *testing.T) {
	continuation := testChildContinuation(t)
	continuation.Ref.OwnerNamespace = "other"
	if err := NewMemoryChildContinuationStore("child").Save(context.Background(), continuation); !errors.Is(err, ErrChildContinuationForeign) {
		t.Fatalf("foreign save error = %v", err)
	}
	continuation = testChildContinuation(t)
	continuation.PendingActionReference = string(make([]byte, ChildContinuationMaxReferenceBytes+1))
	if _, err := SealChildContinuation(continuation); !errors.Is(err, ErrChildContinuationMalformed) {
		t.Fatalf("oversized seal error = %v", err)
	}
}
