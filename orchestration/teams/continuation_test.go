package teams

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

func TestTeamContinuationFileStoreReopensAndDuplicateClaimConverges(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "team-continuations.json")
	store, err := NewFileTeamContinuationStore(path, "team-owner")
	if err != nil {
		t.Fatalf("NewFileTeamContinuationStore() error = %v", err)
	}
	now := time.Now().UTC()
	continuation, err := SealTeamContinuation(TeamContinuation{
		Ref:    TeamContinuationRef{OwnerNamespace: "team-owner"},
		TeamID: "team-1", GraphVersion: "v1", GraphDigest: "sha256:graph",
		RunID: "run-1", SessionID: "session-1", ParentRunID: "parent-1",
		CausationID: "cause-1", AuthorityDigest: "authority", PlanDigest: "plan",
		WriteScopeDigest: "scope", EffectNamespace: "effects", OccurrenceID: "occurrence",
		Strategy: StrategyParallel, PauseReason: TeamPauseInputRequired,
		Tasks:          []TeamTaskContinuation{{TaskID: "member-1", Status: TaskStatusSucceeded, EffectReference: "effect-1"}, {TaskID: "member-2", Status: TaskStatusPending}},
		PendingTaskIDs: []string{"member-2"}, PendingActionReference: "action-1",
		Generation: 1, CreatedAt: now, ExpiresAt: now.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("SealTeamContinuation() error = %v", err)
	}
	if err := store.Save(ctx, continuation); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	reopened, err := NewFileTeamContinuationStore(path, "team-owner")
	if err != nil {
		t.Fatalf("NewFileTeamContinuationStore(reopen) error = %v", err)
	}
	loaded, err := reopened.Load(ctx, continuation.Ref)
	if err != nil {
		t.Fatalf("Load(reopen) error = %v", err)
	}
	if loaded.Ref != continuation.Ref || len(loaded.Tasks) != 2 || loaded.Tasks[0].OutputDigest != "" {
		t.Fatalf("loaded continuation = %#v", loaded)
	}

	req := TeamResumeRequest{Ref: continuation.Ref, GraphDigest: continuation.GraphDigest, Generation: continuation.Generation, Strategy: StrategyParallel, IdempotencyKey: "resume-1", AuthorityDigest: "authority", PlanDigest: "plan", WriteScopeDigest: "scope", EffectNamespace: "effects", OccurrenceID: "occurrence", CausationID: "cause-1", Deployment: types.DeploymentRequest{OwnerNamespace: "team-owner", WriterFence: "fence:team-owner"}}
	first, err := reopened.ClaimResume(ctx, req)
	if err != nil || first.Duplicate {
		t.Fatalf("ClaimResume(first) = %#v, %v", first, err)
	}
	duplicate, err := reopened.ClaimResume(ctx, req)
	if !errors.Is(err, ErrTeamContinuationInProgress) || !duplicate.Duplicate || duplicate.AttemptID != first.AttemptID {
		t.Fatalf("ClaimResume(duplicate) = %#v, %v", duplicate, err)
	}
}

func TestMemoryTeamContinuationDoesNotAdvertiseCrossProcessRestore(t *testing.T) {
	t.Parallel()
	store := NewMemoryTeamContinuationStore("team-owner")
	engine := New(WithTeamContinuationStore(store, "team-owner"))
	caps, err := engine.ContinuationCapabilities(context.Background())
	if err != nil {
		t.Fatalf("ContinuationCapabilities() error = %v", err)
	}
	if caps.CrossProcessRestore || caps.Deployment.Supported {
		t.Fatalf("memory continuation capabilities = %#v", caps)
	}
}

func TestFileTeamContinuationRollsBackClaimWhenPersistenceFails(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	path := filepath.Join(root, "team-continuations.json")
	store, err := NewFileTeamContinuationStore(path, "team-owner")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	continuation, err := SealTeamContinuation(TeamContinuation{
		Ref:    TeamContinuationRef{OwnerNamespace: "team-owner"},
		TeamID: "team-atomic", GraphVersion: "v1", GraphDigest: "sha256:graph",
		RunID: "run-atomic", SessionID: "session-atomic", ParentRunID: "parent-atomic",
		CausationID: "cause-atomic", AuthorityDigest: "authority", PlanDigest: "plan",
		WriteScopeDigest: "scope", EffectNamespace: "effects", OccurrenceID: "occurrence",
		Strategy: StrategyParallel, PauseReason: TeamPauseInputRequired,
		Tasks:          []TeamTaskContinuation{{TaskID: "member-1", Status: TaskStatusPending}},
		PendingTaskIDs: []string{"member-1"}, Generation: 1, CreatedAt: now, ExpiresAt: now.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(ctx, continuation); err != nil {
		t.Fatal(err)
	}
	request := TeamResumeRequest{
		Ref: continuation.Ref, GraphDigest: continuation.GraphDigest, Generation: continuation.Generation,
		Strategy: StrategyParallel, IdempotencyKey: "resume-atomic", AuthorityDigest: continuation.AuthorityDigest,
		PlanDigest: continuation.PlanDigest, WriteScopeDigest: continuation.WriteScopeDigest,
		EffectNamespace: continuation.EffectNamespace, OccurrenceID: continuation.OccurrenceID,
		CausationID: continuation.CausationID,
		Deployment:  types.DeploymentRequest{OwnerNamespace: "team-owner", WriterFence: "fence:team-owner"},
	}
	// A directory target makes the atomic replacement fail after the candidate
	// mutation, exercising rollback of both memory and durable state.
	store.path = t.TempDir()
	if _, err := store.ClaimResume(ctx, request); !errors.Is(err, ErrTeamContinuationUnavailable) {
		t.Fatalf("ClaimResume() error = %v, want unavailable", err)
	}
	store.path = path
	fresh, err := NewFileTeamContinuationStore(path, "team-owner")
	if err != nil {
		t.Fatal(err)
	}
	persistedClaim, err := fresh.ClaimResume(ctx, request)
	if err != nil || persistedClaim.Duplicate {
		t.Fatalf("fresh ClaimResume(after rollback) = %#v, %v", persistedClaim, err)
	}
	claim, err := store.ClaimResume(ctx, request)
	if err != nil || claim.Duplicate {
		t.Fatalf("ClaimResume(after rollback) = %#v, %v", claim, err)
	}
}

func TestTeamParallelPauseAndFreshProcessResumeRunsOriginalGraph(t *testing.T) {
	path := filepath.Join(t.TempDir(), "team-resume.json")
	store, err := NewFileTeamContinuationStore(path, "team-owner")
	if err != nil {
		t.Fatal(err)
	}
	plan := Plan{RunID: "run-team", SessionID: "session-team", ParentRunID: "parent-team", CausationID: "cause-team", AuthorityDigest: "authority", PlanDigest: "plan", WriteScopeDigest: "scope", EffectNamespace: "effects", OccurrenceID: "occurrence", TeamID: "team-resume", Strategy: StrategyParallel, Tasks: []Task{{TaskID: "member-a", AgentID: "agent-a", Role: RoleWorker, Runner: TaskRunnerFunc(func(context.Context, Task) (TaskResult, error) { return TaskResult{Output: "a"}, nil })}, {TaskID: "member-b", AgentID: "agent-b", Role: RoleWorker, Runner: TaskRunnerFunc(func(context.Context, Task) (TaskResult, error) { return TaskResult{Output: "b"}, nil })}}}
	paused := New(WithTeamContinuationStore(store, "team-owner"), WithTeamPauseController(TeamPauseControllerFunc(func(context.Context, TeamPauseBoundary) error {
		return TeamPauseError{Reason: TeamPauseInputRequired, PendingActionReference: "input"}
	})))
	result, err := paused.Run(context.Background(), plan)
	if !errors.Is(err, ErrTeamPaused) || result.TeamCheckpoint == nil {
		t.Fatalf("pause result=%#v err=%v", result, err)
	}

	reopened, err := NewFileTeamContinuationStore(path, "team-owner")
	if err != nil {
		t.Fatal(err)
	}
	resumed := New(WithTeamContinuationStore(reopened, "team-owner"))
	ref := *result.TeamCheckpoint
	request := TeamResumeRequest{Ref: ref, GraphDigest: ref.Digest, Generation: 1, Strategy: StrategyParallel, IdempotencyKey: "resume-team", AuthorityDigest: plan.AuthorityDigest, PlanDigest: plan.PlanDigest, WriteScopeDigest: plan.WriteScopeDigest, EffectNamespace: plan.EffectNamespace, OccurrenceID: plan.OccurrenceID, CausationID: plan.CausationID, Deployment: types.DeploymentRequest{OwnerNamespace: "team-owner", WriterFence: "fence:team-owner"}}
	// The checkpoint graph digest is the plan digest, not the opaque checkpoint digest.
	checkpoint, loadErr := reopened.Load(context.Background(), ref)
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	request.GraphDigest = checkpoint.GraphDigest
	final, err := resumed.ResumeContinuation(context.Background(), plan, request, nil)
	if err != nil {
		t.Fatalf("ResumeContinuation() error = %v", err)
	}
	if final.TeamTaskTotal != 2 || final.TeamTaskFailed != 0 || final.Tasks[0].Status != TaskStatusSucceeded || final.Tasks[1].Status != TaskStatusSucceeded {
		t.Fatalf("resumed result = %#v", final)
	}
	duplicate, err := resumed.ResumeContinuation(context.Background(), plan, request, nil)
	if err != nil {
		t.Fatalf("duplicate resume error = %v", err)
	}
	if duplicate.TeamTaskTotal != final.TeamTaskTotal || duplicate.Tasks[0].Status != final.Tasks[0].Status {
		t.Fatalf("duplicate result = %#v", duplicate)
	}
}

func TestTeamPauseReasonsPersistAcrossRunAndStream(t *testing.T) {
	for _, reason := range []TeamPauseReason{TeamPauseApprovalRequired, TeamPauseBudgetExhausted} {
		t.Run(string(reason), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), string(reason)+".json")
			store, err := NewFileTeamContinuationStore(path, "team-owner")
			if err != nil {
				t.Fatal(err)
			}
			plan := Plan{RunID: "run-" + string(reason), SessionID: "session-" + string(reason), ParentRunID: "parent", CausationID: "cause", AuthorityDigest: "authority", PlanDigest: "plan", WriteScopeDigest: "scope", EffectNamespace: "effects", OccurrenceID: "occurrence", TeamID: "team-" + string(reason), Strategy: StrategyParallel, Tasks: []Task{{TaskID: "member", AgentID: "agent", Role: RoleWorker, Runner: TaskRunnerFunc(func(context.Context, Task) (TaskResult, error) { return TaskResult{Output: "unexpected"}, nil })}}}
			engine := New(WithTeamContinuationStore(store, "team-owner"), WithTeamPauseController(TeamPauseControllerFunc(func(context.Context, TeamPauseBoundary) error {
				return TeamPauseError{Reason: reason, PendingActionReference: string(reason), ConsumedBudget: 7, OriginalDeadline: time.Now().UTC().Add(time.Hour)}
			})))
			var result Result
			if reason == TeamPauseApprovalRequired {
				result, err = engine.Stream(context.Background(), plan, nil)
			} else {
				result, err = engine.Run(context.Background(), plan)
			}
			if !errors.Is(err, ErrTeamPaused) || result.TeamCheckpoint == nil || result.PauseReason != reason {
				t.Fatalf("pause result=%#v err=%v", result, err)
			}
			reopened, err := NewFileTeamContinuationStore(path, "team-owner")
			if err != nil {
				t.Fatal(err)
			}
			continuation, err := reopened.Load(context.Background(), *result.TeamCheckpoint)
			if err != nil || continuation.PauseReason != reason || continuation.ConsumedBudget != 7 || continuation.OriginalDeadline.IsZero() {
				t.Fatalf("continuation=%#v err=%v", continuation, err)
			}
		})
	}
}

func TestTeamParallelPauseBeforeBatchDispatchKeepsEveryMemberPending(t *testing.T) {
	path := filepath.Join(t.TempDir(), "parallel-batch-pause.json")
	store, err := NewFileTeamContinuationStore(path, "team-owner")
	if err != nil {
		t.Fatal(err)
	}
	plan := Plan{
		RunID: "run-parallel-batch", SessionID: "session-parallel-batch", ParentRunID: "parent", CausationID: "cause", AuthorityDigest: "authority", PlanDigest: "plan", WriteScopeDigest: "scope", EffectNamespace: "effects", OccurrenceID: "occurrence", TeamID: "team-parallel-batch", Strategy: StrategyParallel,
		Tasks: []Task{
			{TaskID: "member-a", AgentID: "agent-a", Role: RoleWorker, Runner: TaskRunnerFunc(func(context.Context, Task) (TaskResult, error) {
				t.Fatal("member-a executed before pause")
				return TaskResult{}, nil
			})},
			{TaskID: "member-b", AgentID: "agent-b", Role: RoleWorker, Runner: TaskRunnerFunc(func(context.Context, Task) (TaskResult, error) {
				t.Fatal("member-b executed before pause")
				return TaskResult{}, nil
			})},
		},
	}
	boundaries := 0
	engine := New(WithTeamContinuationStore(store, "team-owner"), WithTeamPauseController(TeamPauseControllerFunc(func(context.Context, TeamPauseBoundary) error {
		boundaries++
		if boundaries == 2 {
			return TeamPauseError{Reason: TeamPauseInputRequired, PendingActionReference: "input"}
		}
		return nil
	})))
	result, err := engine.Run(context.Background(), plan)
	if !errors.Is(err, ErrTeamPaused) || result.TeamCheckpoint == nil {
		t.Fatalf("pause result=%#v err=%v", result, err)
	}
	continuation, err := store.Load(context.Background(), *result.TeamCheckpoint)
	if err != nil {
		t.Fatal(err)
	}
	if len(continuation.PendingTaskIDs) != 2 || continuation.PendingTaskIDs[0] != "member-a" || continuation.PendingTaskIDs[1] != "member-b" {
		t.Fatalf("pending members = %#v", continuation.PendingTaskIDs)
	}
	for _, task := range continuation.Tasks {
		if task.Status != TaskStatusPending {
			t.Fatalf("member %s status=%s, want pending", task.TaskID, task.Status)
		}
	}
}

func TestTeamContinuationRejectsExpiredOriginalDeadlineBeforeMemberDispatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "expired-original-deadline.json")
	store, err := NewFileTeamContinuationStore(path, "team-owner")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	dispatched := 0
	plan := Plan{RunID: "run", SessionID: "session", ParentRunID: "parent", CausationID: "cause", AuthorityDigest: "authority", PlanDigest: "plan", WriteScopeDigest: "scope", EffectNamespace: "effects", OccurrenceID: "occurrence", TeamID: "team-expired", Strategy: StrategyParallel, Tasks: []Task{{TaskID: "member", AgentID: "agent", Role: RoleWorker, Runner: TaskRunnerFunc(func(context.Context, Task) (TaskResult, error) { dispatched++; return TaskResult{}, nil })}}}
	normalizedPlan, err := normalizePlan(plan)
	if err != nil {
		t.Fatal(err)
	}
	continuation, err := SealTeamContinuation(TeamContinuation{
		Ref: TeamContinuationRef{OwnerNamespace: "team-owner"}, TeamID: "team-expired", GraphVersion: "v1", GraphDigest: teamPlanDigest(normalizedPlan),
		RunID: "run", SessionID: "session", ParentRunID: "parent", CausationID: "cause", AuthorityDigest: "authority", PlanDigest: "plan", WriteScopeDigest: "scope", EffectNamespace: "effects", OccurrenceID: "occurrence",
		Strategy: StrategyParallel, PauseReason: TeamPauseBudgetExhausted, Tasks: []TeamTaskContinuation{{TaskID: "member", Status: TaskStatusPending}}, PendingTaskIDs: []string{"member"},
		OriginalDeadline: now.Add(-time.Minute), Generation: 1, CreatedAt: now.Add(-2 * time.Minute), ExpiresAt: now.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(context.Background(), continuation); err != nil {
		t.Fatal(err)
	}
	request := TeamResumeRequest{Ref: continuation.Ref, GraphDigest: continuation.GraphDigest, Generation: continuation.Generation, Strategy: continuation.Strategy, IdempotencyKey: "resume", AuthorityDigest: continuation.AuthorityDigest, PlanDigest: continuation.PlanDigest, WriteScopeDigest: continuation.WriteScopeDigest, EffectNamespace: continuation.EffectNamespace, OccurrenceID: continuation.OccurrenceID, CausationID: continuation.CausationID, Deployment: types.DeploymentRequest{OwnerNamespace: "team-owner", WriterFence: "fence:team-owner"}}
	_, err = New(WithTeamContinuationStore(store, "team-owner")).ResumeContinuation(context.Background(), plan, request, nil)
	if !errors.Is(err, ErrTeamContinuationRecoveryRequired) {
		t.Fatalf("ResumeContinuation() error = %v, want recovery required", err)
	}
	if dispatched != 0 {
		t.Fatalf("expired continuation dispatched %d members", dispatched)
	}
}

func TestTeamVoteContinuationRestoresPendingMemberWithoutRerunningSettledVote(t *testing.T) {
	build := func(t *testing.T) (*Engine, Plan, TeamResumeRequest, *int) {
		t.Helper()
		store, err := NewFileTeamContinuationStore(filepath.Join(t.TempDir(), "vote-continuation.json"), "team-owner")
		if err != nil {
			t.Fatal(err)
		}
		settledRuns := 0
		pendingRuns := 0
		plan := Plan{RunID: "run-vote", SessionID: "session-vote", ParentRunID: "parent", CausationID: "cause", AuthorityDigest: "authority", PlanDigest: "plan", WriteScopeDigest: "scope", EffectNamespace: "effects", OccurrenceID: "occurrence", TeamID: "team-vote", Strategy: StrategyVote, VoteTieBreak: VoteTieBreakFirstTaskID, Tasks: []Task{
			{TaskID: "settled", AgentID: "agent-a", Role: RoleWorker, Runner: TaskRunnerFunc(func(context.Context, Task) (TaskResult, error) { settledRuns++; return TaskResult{Vote: "yes"}, nil })},
			{TaskID: "pending", AgentID: "agent-b", Role: RoleWorker, Runner: TaskRunnerFunc(func(context.Context, Task) (TaskResult, error) { pendingRuns++; return TaskResult{Vote: "no"}, nil })},
		}}
		normalized, err := normalizePlan(plan)
		if err != nil {
			t.Fatal(err)
		}
		now := time.Now().UTC()
		continuation, err := SealTeamContinuation(TeamContinuation{
			Ref: TeamContinuationRef{OwnerNamespace: "team-owner"}, TeamID: normalized.TeamID, GraphVersion: "v1", GraphDigest: teamPlanDigest(normalized),
			RunID: normalized.RunID, SessionID: normalized.SessionID, ParentRunID: normalized.ParentRunID, CausationID: normalized.CausationID, AuthorityDigest: normalized.AuthorityDigest, PlanDigest: normalized.PlanDigest, WriteScopeDigest: normalized.WriteScopeDigest, EffectNamespace: normalized.EffectNamespace, OccurrenceID: normalized.OccurrenceID,
			Strategy: StrategyVote, PauseReason: TeamPauseApprovalRequired, Tasks: []TeamTaskContinuation{{TaskID: "settled", Status: TaskStatusSucceeded, Vote: "yes", EffectReference: "effect-settled"}, {TaskID: "pending", Status: TaskStatusPending}}, PendingTaskIDs: []string{"pending"},
			Generation: 1, CreatedAt: now, ExpiresAt: now.Add(time.Hour),
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := store.Save(context.Background(), continuation); err != nil {
			t.Fatal(err)
		}
		request := TeamResumeRequest{Ref: continuation.Ref, GraphDigest: continuation.GraphDigest, Generation: continuation.Generation, Strategy: continuation.Strategy, IdempotencyKey: "resume-vote", AuthorityDigest: continuation.AuthorityDigest, PlanDigest: continuation.PlanDigest, WriteScopeDigest: continuation.WriteScopeDigest, EffectNamespace: continuation.EffectNamespace, OccurrenceID: continuation.OccurrenceID, CausationID: continuation.CausationID, Deployment: types.DeploymentRequest{OwnerNamespace: "team-owner", WriterFence: "fence:team-owner"}}
		return New(WithTeamContinuationStore(store, "team-owner")), plan, request, &settledRuns
	}

	runEngine, runPlan, runRequest, runSettled := build(t)
	runResult, err := runEngine.ResumeContinuation(context.Background(), runPlan, runRequest, nil)
	if err != nil {
		t.Fatalf("Run resume: %v", err)
	}
	if *runSettled != 0 || runResult.WinnerVote != "no" || len(runResult.Tasks) != 2 || runResult.Tasks[0].Status != TaskStatusSucceeded || runResult.Tasks[1].Status != TaskStatusSucceeded {
		t.Fatalf("Run result=%#v settledRuns=%d", runResult, *runSettled)
	}

	streamEngine, streamPlan, streamRequest, streamSettled := build(t)
	streamEvents := 0
	streamResult, err := streamEngine.ResumeContinuation(context.Background(), streamPlan, streamRequest, func(StreamEvent) error { streamEvents++; return nil })
	if err != nil {
		t.Fatalf("Stream resume: %v", err)
	}
	if *streamSettled != 0 || streamEvents == 0 || streamResult.WinnerVote != runResult.WinnerVote || streamResult.TeamTaskTotal != runResult.TeamTaskTotal || streamResult.TeamTaskFailed != runResult.TeamTaskFailed {
		t.Fatalf("Stream result=%#v settledRuns=%d events=%d", streamResult, *streamSettled, streamEvents)
	}
}
