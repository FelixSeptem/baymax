package checkpoint

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/sourcecoordination"
	"github.com/FelixSeptem/baymax/core/types"
)

func testCheckpoint(t *testing.T, now time.Time) types.PersistedRunCheckpoint {
	t.Helper()
	p, err := types.SealRunCheckpoint(types.PersistedRunCheckpoint{Ref: types.NativeCheckpointRef{SchemaVersion: types.NativeContinuationVersion, CheckpointID: "checkpoint", SourceComponent: "runner", OwnerNamespace: "tests", RunID: "run", SessionID: "session", PauseReason: types.PauseReasonInputRequired, SourcePosition: "iteration:1", CreatedAt: now, ExpiresAt: now.Add(time.Hour)}, Generation: 1, Material: json.RawMessage(`{"pending":[]}`), DependencyProfile: "model", DependencyRevision: "v1", Mode: "run"})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestCoordinatedFileStoresShareCheckpointStateWithoutDuplicateClaim(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	coordDir := filepath.Join(t.TempDir(), "coord")
	coordA, err := sourcecoordination.OpenFile(coordDir, sourcecoordination.Options{OwnerNamespace: "tests"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = coordA.Close() }()
	coordB, err := sourcecoordination.OpenFile(coordDir, sourcecoordination.Options{OwnerNamespace: "tests"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = coordB.Close() }()
	a, err := OpenFileStore(dir, Options{Namespace: "tests", Capacity: 2, Coordinator: coordA, WriterID: "a"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = a.Close() }()
	p := testCheckpoint(t, time.Now().UTC())
	if err := a.Save(ctx, p); err != nil {
		t.Fatal(err)
	}
	b, err := OpenFileStore(dir, Options{Namespace: "tests", Capacity: 2, Coordinator: coordB, WriterID: "b"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = b.Close() }()
	loaded, err := b.Load(ctx, p.Ref)
	if err != nil || loaded.Ref != p.Ref {
		t.Fatalf("coordinated load = %+v %v", loaded.Ref, err)
	}
	claim, err := b.ClaimResume(ctx, testRequest(p))
	if err != nil {
		t.Fatal(err)
	}
	duplicate, err := a.ClaimResume(ctx, testRequest(p))
	if !errors.Is(err, types.ErrStaleWriter) && !errors.Is(err, types.ErrCheckpointRecoveryRequired) {
		t.Fatalf("duplicate claim error = %v", err)
	}
	if duplicate.AttemptID != claim.AttemptID {
		t.Fatalf("duplicate attempt = %q want %q", duplicate.AttemptID, claim.AttemptID)
	}
	if claim.AttemptID == "" {
		t.Fatal("claim did not produce attempt")
	}
}

func TestCoordinatedFileStoreDoesNotImportLocalSnapshotBeforeCoordinatorPublication(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	dir := t.TempDir()
	checkpoint := testCheckpoint(t, now)
	local, err := OpenFileStore(dir, Options{Namespace: "tests", Capacity: 2})
	if err != nil {
		t.Fatal(err)
	}
	if err = local.Save(ctx, checkpoint); err != nil {
		t.Fatal(err)
	}
	if err = local.Close(); err != nil {
		t.Fatal(err)
	}
	coordinator, err := sourcecoordination.OpenFile(filepath.Join(t.TempDir(), "coord"), sourcecoordination.Options{OwnerNamespace: "tests"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = coordinator.Close() }()
	coordinated, err := OpenFileStore(dir, Options{Namespace: "tests", Capacity: 2, Coordinator: coordinator, WriterID: "writer"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = coordinated.Close() }()
	if _, err = coordinated.Load(ctx, checkpoint.Ref); !errors.Is(err, types.ErrCheckpointNotFound) {
		t.Fatalf("local snapshot leaked into coordinated state: %v", err)
	}
}

func TestCoordinatedFileStoreDoesNotAdvertiseRestoreAfterCoordinatorLoss(t *testing.T) {
	coordinator, err := sourcecoordination.OpenFile(filepath.Join(t.TempDir(), "coord"), sourcecoordination.Options{OwnerNamespace: "tests"})
	if err != nil {
		t.Fatal(err)
	}
	store, err := OpenFileStore(t.TempDir(), Options{Namespace: "tests", Capacity: 1, Coordinator: coordinator, WriterID: "writer"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	if !store.CrossProcessRestore() {
		t.Fatal("coordinated source did not advertise restore while its lease was live")
	}
	if err = coordinator.Close(); err != nil {
		t.Fatal(err)
	}
	if store.CrossProcessRestore() {
		t.Fatal("coordinator loss still advertised cross-process restore")
	}
}

func TestCoordinatedCheckpointMapsCoordinatorOutageToCheckpointUnavailable(t *testing.T) {
	coordinator, err := sourcecoordination.OpenFile(filepath.Join(t.TempDir(), "coord"), sourcecoordination.Options{OwnerNamespace: "tests"})
	if err != nil {
		t.Fatal(err)
	}
	store, err := OpenFileStore(t.TempDir(), Options{Namespace: "tests", Capacity: 1, Coordinator: coordinator, WriterID: "writer"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err = coordinator.Close(); err != nil {
		t.Fatal(err)
	}
	if err = store.Save(context.Background(), testCheckpoint(t, time.Now().UTC())); !errors.Is(err, types.ErrCheckpointUnavailable) {
		t.Fatalf("coordinator outage error = %v, want checkpoint unavailable", err)
	}
}

func TestCoordinatedControlRefreshesClaimBeforeMutatingStaleSnapshot(t *testing.T) {
	ctx := context.Background()
	coordDir := filepath.Join(t.TempDir(), "coord")
	coordA, err := sourcecoordination.OpenFile(coordDir, sourcecoordination.Options{OwnerNamespace: "tests"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = coordA.Close() })
	coordB, err := sourcecoordination.OpenFile(coordDir, sourcecoordination.Options{OwnerNamespace: "tests"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = coordB.Close() })
	storeDir := t.TempDir()
	a, err := OpenFileStore(storeDir, Options{Namespace: "tests", Capacity: 2, Coordinator: coordA, WriterID: "writer-a"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = a.Close() })
	checkpoint := testCheckpoint(t, time.Now().UTC())
	if err = a.Save(ctx, checkpoint); err != nil {
		t.Fatal(err)
	}
	b, err := OpenFileStore(storeDir, Options{Namespace: "tests", Capacity: 2, Coordinator: coordB, WriterID: "writer-b"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = b.Close() })
	if _, err = b.ClaimResume(ctx, testRequest(checkpoint)); err != nil {
		t.Fatal(err)
	}
	if _, err = a.Control(ctx, types.CheckpointControl{Checkpoint: checkpoint.Ref, Generation: checkpoint.Generation, Action: types.CheckpointControlCancel}); !errors.Is(err, types.ErrCheckpointInProgress) {
		t.Fatalf("stale control error = %v, want checkpoint in progress", err)
	}
	if a.records[checkpoint.Ref.CheckpointID].Canceled {
		t.Fatal("stale control mutated its local snapshot")
	}
	claim, err := b.ReadResumeClaim(ctx, checkpoint.Ref)
	if err != nil || claim.AttemptID == "" || claim.State != types.ResumeClaimClaimed {
		t.Fatalf("source claim after stale control = %#v, err=%v", claim, err)
	}
}

func testRequest(p types.PersistedRunCheckpoint) types.ResumeRequest {
	return types.ResumeRequest{Checkpoint: p.Ref, Mode: p.Mode, DependencyProfile: p.DependencyProfile, DependencyRevision: p.DependencyRevision, Generation: p.Generation, IdempotencyKey: "key"}
}

func TestClaimPreservesUnlimitedBudgetsWithConsumption(t *testing.T) {
	for _, budget := range []types.DurableBudget{
		{Consumed: 1},
		{ToolConsumed: 1},
		{Consumed: 1, ToolConsumed: 1},
	} {
		t.Run(fmt.Sprintf("iterations_%d_tools_%d", budget.Consumed, budget.ToolConsumed), func(t *testing.T) {
			ctx := context.Background()
			s, err := NewMemoryStore(Options{Namespace: "tests", Capacity: 1})
			if err != nil {
				t.Fatal(err)
			}
			p := testCheckpoint(t, time.Now().UTC())
			p.Budget = budget
			p, err = types.SealRunCheckpoint(p)
			if err != nil {
				t.Fatal(err)
			}
			if err = s.Save(ctx, p); err != nil {
				t.Fatal(err)
			}
			req := testRequest(p)
			req.Budget = budget
			if _, err = s.ClaimResume(ctx, req); err != nil {
				t.Fatalf("unchanged unlimited budget rejected: %v", err)
			}
		})
	}
}

func TestMemoryStoreClaimCommitAndIsolation(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	s, err := NewMemoryStore(Options{Namespace: "tests", Capacity: 2, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	p := testCheckpoint(t, now)
	if err = s.Save(ctx, p); err != nil {
		t.Fatal(err)
	}
	p.Material[0] = '['
	p, err = s.Load(ctx, p.Ref)
	if err != nil || p.Material[0] != '{' {
		t.Fatalf("detachment: %s %v", p.Material, err)
	}
	r := testRequest(p)
	claim, err := s.ClaimResume(ctx, r)
	if err != nil || claim.Duplicate {
		t.Fatalf("claim: %#v %v", claim, err)
	}
	dup, err := s.ClaimResume(ctx, r)
	if !errors.Is(err, types.ErrCheckpointInProgress) || !dup.Duplicate || dup.AttemptID != claim.AttemptID {
		t.Fatalf("duplicate: %#v %v", dup, err)
	}
	conflict := r
	conflict.AuthorityDigest = "other"
	if _, err = s.ClaimResume(ctx, conflict); !errors.Is(err, types.ErrCheckpointConflict) {
		t.Fatalf("conflict: %v", err)
	}
	claim, err = s.MarkRunning(ctx, claim)
	if err != nil {
		t.Fatal(err)
	}
	claim, err = s.CommitResumeResult(ctx, claim, types.ResumeResult{State: types.RunStateCompleted, Projection: json.RawMessage(`{"ok":true}`)})
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Hour)
	dup, err = s.ClaimResume(ctx, r)
	if err != nil || !dup.Duplicate || dup.Result == nil || dup.Result.State != types.RunStateCompleted {
		t.Fatalf("terminal duplicate: %#v %v", dup, err)
	}
	if !dup.RetainedUntil.After(now) {
		t.Fatal("result not retained for 24h")
	}
}

func TestCheckpointClaimRejectsUnsupportedDeploymentBeforeMutation(t *testing.T) {
	now := time.Now().UTC()
	s, err := NewMemoryStore(Options{Namespace: "tests", Capacity: 1, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	p := testCheckpoint(t, now)
	if err := s.Save(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	req := testRequest(p)
	req.Deployment = types.DeploymentRequest{Profile: types.DeploymentProfileCrossMachine, WriterMode: types.WriterScopeMultiWriter}
	if _, err := s.ClaimResume(context.Background(), req); !errors.Is(err, types.ErrDeploymentUnsupported) {
		t.Fatalf("claim error = %v", err)
	}
	if _, err := s.ReadResumeClaim(context.Background(), p.Ref); !errors.Is(err, types.ErrCheckpointNotFound) {
		t.Fatalf("rejected claim left source mutation: %v", err)
	}
}

func TestClaimGenerationAndCompletionRetention(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	s, _ := NewMemoryStore(Options{Namespace: "tests", Capacity: 2, Now: func() time.Time { return now }})
	p := testCheckpoint(t, now)
	if err := s.Save(ctx, p); err != nil {
		t.Fatal(err)
	}
	claimed, err := s.ClaimResume(ctx, testRequest(p))
	if err != nil {
		t.Fatal(err)
	}
	running, err := s.MarkRunning(ctx, claimed)
	if err != nil {
		t.Fatal(err)
	}
	if running.Generation <= claimed.Generation {
		t.Fatal("running did not advance generation")
	}
	if _, err := s.CommitResumeResult(ctx, claimed, types.ResumeResult{State: types.RunStateCompleted}); !errors.Is(err, types.ErrCheckpointConflict) {
		t.Fatalf("stale claim: %v", err)
	}
	now = now.Add(23 * time.Hour)
	completed, err := s.CommitResumeResult(ctx, running, types.ResumeResult{State: types.RunStateCompleted})
	if err != nil {
		t.Fatal(err)
	}
	if completed.Generation <= running.Generation || !completed.RetainedUntil.Equal(now.Add(24*time.Hour)) {
		t.Fatalf("completion: %#v", completed)
	}
	if _, err := s.Load(ctx, p.Ref); err != nil {
		t.Fatalf("retained load: %v", err)
	}
	if _, err := s.CommitResumeResult(ctx, completed, types.ResumeResult{State: types.RunStateFailed}); !errors.Is(err, types.ErrCheckpointConflict) {
		t.Fatalf("changed terminal result: %v", err)
	}
}

func TestStoreRejectsForeignExpiryCapacityCancelAndContext(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	s, _ := NewMemoryStore(Options{Namespace: "tests", Capacity: 1, Now: func() time.Time { return now }})
	p := testCheckpoint(t, now)
	cancelCtx, cancel := context.WithCancel(ctx)
	cancel()
	if err := s.Save(cancelCtx, p); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled save: %v", err)
	}
	if err := s.Save(ctx, p); err != nil {
		t.Fatal(err)
	}
	foreign := p.Ref
	foreign.OwnerNamespace = "other"
	if _, err := s.Load(ctx, foreign); !errors.Is(err, types.ErrCheckpointForeign) {
		t.Fatalf("foreign: %v", err)
	}
	other := p
	other.Ref.CheckpointID = "other"
	other, _ = types.SealRunCheckpoint(other)
	if err := s.Save(ctx, other); !errors.Is(err, types.ErrCheckpointUnavailable) {
		t.Fatalf("capacity: %v", err)
	}
	if _, err := s.Control(ctx, types.CheckpointControl{Checkpoint: p.Ref, Generation: 1, Action: types.CheckpointControlCancel}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ClaimResume(ctx, testRequest(p)); !errors.Is(err, types.ErrCheckpointTerminal) {
		t.Fatalf("cancel: %v", err)
	}
	s, _ = NewMemoryStore(Options{Namespace: "tests", Capacity: 1, Now: func() time.Time { return now }})
	if err := s.Save(ctx, p); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Hour)
	if _, err := s.Load(ctx, p.Ref); !errors.Is(err, types.ErrCheckpointExpired) {
		t.Fatalf("expiry: %v", err)
	}
}

func TestFileStoreReopenLockCorruptionAndCrashClaim(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	dir := t.TempDir()
	opts := Options{Namespace: "tests", Capacity: 2, Now: func() time.Time { return now }}
	s, err := OpenFileStore(dir, opts)
	if err != nil {
		t.Fatal(err)
	}
	p := testCheckpoint(t, now)
	if err = s.Save(ctx, p); err != nil {
		t.Fatal(err)
	}
	if _, err = OpenFileStore(dir, opts); !errors.Is(err, types.ErrCheckpointUnavailable) {
		t.Fatalf("second writer: %v", err)
	}
	tampered := p.Ref
	tampered.Digest = "tampered"
	if _, err := s.Load(ctx, tampered); !errors.Is(err, types.ErrCheckpointForeign) {
		t.Fatalf("tampered ref: %v", err)
	}
	claim, err := s.ClaimResume(ctx, testRequest(p))
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = OpenFileStore(dir, opts)
	if err != nil {
		t.Fatal(err)
	}
	dup, err := s.ClaimResume(ctx, testRequest(p))
	if !errors.Is(err, types.ErrCheckpointRecoveryRequired) || !dup.Duplicate || dup.AttemptID != claim.AttemptID {
		t.Fatalf("crash claim: %#v %v", dup, err)
	}
	if _, err := s.MarkRunning(ctx, claim); !errors.Is(err, types.ErrCheckpointRecoveryRequired) {
		t.Fatalf("recovered claim restarted: %v", err)
	}
	fresh := testCheckpoint(t, now)
	fresh.Ref.CheckpointID = "fresh"
	fresh, err = types.SealRunCheckpoint(fresh)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Save(ctx, fresh); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ClaimResume(ctx, testRequest(fresh)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ClaimResume(ctx, testRequest(fresh)); !errors.Is(err, types.ErrCheckpointInProgress) {
		t.Fatalf("fresh claim after reopen: %v", err)
	}
	if _, err = s.Load(ctx, p.Ref); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "checkpoints.json"), []byte(`{"corrupt":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = OpenFileStore(dir, opts); !errors.Is(err, types.ErrCheckpointCorrupt) {
		t.Fatalf("corrupt: %v", err)
	}
}

func TestSuccessorPublishesAtomicallyAndOldClaimDoesNotConsumeIt(t *testing.T) {
	now := time.Now().UTC()
	s, _ := NewMemoryStore(Options{Namespace: "tests", Capacity: 2})
	ctx := context.Background()
	p := testCheckpoint(t, now)
	if err := s.Save(ctx, p); err != nil {
		t.Fatal(err)
	}
	r := testRequest(p)
	claim, err := s.ClaimResume(ctx, r)
	if err != nil {
		t.Fatal(err)
	}
	next := testCheckpoint(t, now)
	next.Ref.CheckpointID = "successor"
	next.Ref.PredecessorCheckpointID = p.Ref.CheckpointID
	next.Ref.ResumeAttemptID = claim.AttemptID
	next, _ = types.SealRunCheckpoint(next)
	if _, err = s.CommitResumeResult(ctx, claim, types.ResumeResult{State: types.RunStateInputRequired, Successor: &next}); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Load(ctx, next.Ref); err != nil {
		t.Fatal(err)
	}
	duplicate, err := s.ClaimResume(ctx, r)
	if err != nil || duplicate.Result == nil || duplicate.Result.Successor.Ref.CheckpointID != "successor" {
		t.Fatalf("old result: %#v %v", duplicate, err)
	}
	if _, err = s.ClaimResume(ctx, testRequest(next)); err != nil {
		t.Fatalf("successor unavailable: %v", err)
	}
}

func TestFileStorePersistsTerminalDuplicateAfterCheckpointExpiry(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	dir := t.TempDir()
	opts := Options{Namespace: "tests", Capacity: 3, Now: func() time.Time { return now }}
	s, err := OpenFileStore(dir, opts)
	if err != nil {
		t.Fatal(err)
	}
	p := testCheckpoint(t, now)
	if err = s.Save(ctx, p); err != nil {
		t.Fatal(err)
	}
	claim, err := s.ClaimResume(ctx, testRequest(p))
	if err != nil {
		t.Fatal(err)
	}
	claim, err = s.MarkRunning(ctx, claim)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.CommitResumeResult(ctx, claim, types.ResumeResult{State: types.RunStateCompleted}); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = OpenFileStore(dir, opts)
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(90 * time.Minute)
	dup, err := s.ClaimResume(ctx, testRequest(p))
	if err != nil || dup.Result == nil {
		t.Fatalf("retained result: %#v %v", dup, err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestStoreContextBeforeClaimAndConcurrentClaims(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	s, _ := NewMemoryStore(Options{Namespace: "tests", Capacity: 1})
	p := testCheckpoint(t, now)
	if err := s.Save(ctx, p); err != nil {
		t.Fatal(err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := s.ClaimResume(canceled, testRequest(p)); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled: %v", err)
	}
	var wg sync.WaitGroup
	admitted := make(chan types.ResumeClaim, 16)
	failures := make(chan error, 16)
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			claim, err := s.ClaimResume(ctx, testRequest(p))
			if err == nil {
				admitted <- claim
			} else {
				failures <- err
			}
		}()
	}
	wg.Wait()
	close(admitted)
	close(failures)
	if len(admitted) != 1 {
		t.Fatalf("admitted=%d", len(admitted))
	}
	for err := range failures {
		if !errors.Is(err, types.ErrCheckpointInProgress) {
			t.Fatalf("racing claim: %v", err)
		}
	}
}

func TestClaimDetachesProposedBudgetFromCaller(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	dir := t.TempDir()
	store, err := OpenFileStore(dir, Options{Namespace: "tests", Capacity: 2})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	checkpoint := testCheckpoint(t, now)
	checkpoint.Ref.PauseReason = types.PauseReasonBudgetExhausted
	checkpoint.Budget = types.DurableBudget{Revision: "budget-v1", Limit: 1, Consumed: 1, ToolLimit: 1}
	checkpoint, err = types.SealRunCheckpoint(checkpoint)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.Save(ctx, checkpoint); err != nil {
		t.Fatal(err)
	}
	request := testRequest(checkpoint)
	proposal := checkpoint.Budget
	proposal.Revision = "budget-v2"
	proposal.Limit += 4
	proposal.ToolLimit += 4
	request.ProposedBudget = &proposal
	request.Budget = proposal
	claim, err := store.ClaimResume(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	proposal.Limit = 999
	if _, err = store.MarkRunning(ctx, claim); err != nil {
		t.Fatalf("caller mutation invalidated claim: %v", err)
	}
	if err = store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = OpenFileStore(dir, Options{Namespace: "tests", Capacity: 2})
	if err != nil {
		t.Fatalf("caller mutation corrupted persisted identity: %v", err)
	}
	loaded, err := store.Load(ctx, checkpoint.Ref)
	if err != nil || loaded.Ref.CheckpointID != checkpoint.Ref.CheckpointID {
		t.Fatalf("reopened checkpoint=%+v err=%v", loaded.Ref, err)
	}
}

func TestStoreRejectsTightenedEnvelopeAndOversizedResult(t *testing.T) {
	now := time.Now().UTC()
	p := testCheckpoint(t, now)
	ctx := context.Background()
	s, _ := NewMemoryStore(Options{Namespace: "tests", Capacity: 1, Limits: types.ContinuationLimits{EnvelopeBytes: 1}})
	if err := s.Save(ctx, p); !errors.Is(err, types.ErrCheckpointMalformed) {
		t.Fatalf("tight envelope: %v", err)
	}
	s, _ = NewMemoryStore(Options{Namespace: "tests", Capacity: 1})
	if err := s.Save(ctx, p); err != nil {
		t.Fatal(err)
	}
	claim, err := s.ClaimResume(ctx, testRequest(p))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.CommitResumeResult(ctx, claim, types.ResumeResult{State: types.RunStateCompleted, Projection: json.RawMessage(`"` + strings.Repeat("x", types.NativeContinuationMaxEnvelopeBytes) + `"`)}); !errors.Is(err, types.ErrCheckpointMalformed) {
		t.Fatalf("oversized result: %v", err)
	}
}

func TestFileStoreFailureQuarantinesUncommittedResult(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	dir := t.TempDir()
	s, err := OpenFileStore(dir, Options{Namespace: "tests", Capacity: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	p := testCheckpoint(t, now)
	if err = s.Save(ctx, p); err != nil {
		t.Fatal(err)
	}
	claim, err := s.ClaimResume(ctx, testRequest(p))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Mkdir(filepath.Join(dir, "unwritable"), 0700); err != nil {
		t.Fatal(err)
	}
	// A directory at the destination deterministically rejects atomic replace.
	if err = os.Remove(filepath.Join(dir, storeFileName)); err != nil {
		t.Fatal(err)
	}
	if err = os.Mkdir(filepath.Join(dir, storeFileName), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err = s.CommitResumeResult(ctx, claim, types.ResumeResult{State: types.RunStateCompleted}); !errors.Is(err, types.ErrCheckpointUnavailable) {
		t.Fatalf("write failure: %v", err)
	}
	if _, err = s.ClaimResume(ctx, testRequest(p)); !errors.Is(err, types.ErrCheckpointUnavailable) {
		t.Fatalf("unacknowledged result escaped quarantine: %v", err)
	}
	if s.CrossProcessRestore() {
		t.Fatal("quarantined source advertised restore availability")
	}
}

func TestFileStoreClosedSourceDoesNotAdvertiseRestore(t *testing.T) {
	s, err := OpenFileStore(t.TempDir(), Options{Namespace: "tests", Capacity: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !s.CrossProcessRestore() {
		t.Fatal("open file source did not advertise restore")
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if s.CrossProcessRestore() {
		t.Fatal("closed source advertised restore availability")
	}
}

func TestFileStoreProcessOwnership(t *testing.T) {
	if os.Getenv("BAYMAX_CHECKPOINT_PROCESS_HELPER") == "1" {
		s, err := OpenFileStore(os.Getenv("BAYMAX_CHECKPOINT_PROCESS_DIR"), Options{Namespace: "tests", Capacity: 2})
		if os.Getenv("BAYMAX_CHECKPOINT_PROCESS_LOCKED") == "1" {
			if !errors.Is(err, types.ErrCheckpointUnavailable) {
				t.Fatalf("process writer admitted: %v", err)
			}
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = s.Close() }()
		return
	}
	dir := t.TempDir()
	s, err := OpenFileStore(dir, Options{Namespace: "tests", Capacity: 2})
	if err != nil {
		t.Fatal(err)
	}
	run := func(locked bool) {
		t.Helper()
		cmd := exec.Command(os.Args[0], "-test.run=^TestFileStoreProcessOwnership$")
		cmd.Env = append(os.Environ(), "BAYMAX_CHECKPOINT_PROCESS_HELPER=1", "BAYMAX_CHECKPOINT_PROCESS_DIR="+dir)
		if locked {
			cmd.Env = append(cmd.Env, "BAYMAX_CHECKPOINT_PROCESS_LOCKED=1")
		}
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("process: %v %s", err, out)
		}
	}
	run(true)
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	run(false)
}
