package teams

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/context/hostprojection"
	"github.com/FelixSeptem/baymax/core/types"
)

func TestTeamContinuationResolvesContextOnlyForBoundPendingMember(t *testing.T) {
	store, err := NewFileTeamContinuationStore(filepath.Join(t.TempDir(), "team-context.json"), "team-owner")
	if err != nil {
		t.Fatal(err)
	}
	settledCalls, pendingCalls, resolverCalls, contextCalls := 0, 0, 0, 0
	plan := Plan{RunID: "run-team-context", SessionID: "session-team-context", ParentRunID: "parent", CausationID: "cause", AuthorityDigest: "authority", PlanDigest: "plan", WriteScopeDigest: "scope", EffectNamespace: "effects", OccurrenceID: "occurrence", TeamID: "team-context", Strategy: StrategyParallel, HostContextBindings: map[string]hostprojection.SourceBinding{"pending": {Form: hostprojection.SourceFormTeam}}, Tasks: []Task{
		{TaskID: "settled", AgentID: "agent-a", Role: RoleWorker, Runner: TaskRunnerFunc(func(context.Context, Task) (TaskResult, error) { settledCalls++; return TaskResult{}, nil })},
		{TaskID: "pending", AgentID: "agent-b", Role: RoleWorker, Runner: teamContextRunnerFunc(func(context.Context, Plan, Task, hostprojection.Resolution) (TaskResult, error) {
			pendingCalls++
			contextCalls++
			return TaskResult{Vote: "yes"}, nil
		})},
	}}
	normalized, err := normalizePlan(plan)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	continuation, err := SealTeamContinuation(TeamContinuation{
		Ref: TeamContinuationRef{OwnerNamespace: "team-owner"}, TeamID: normalized.TeamID, GraphVersion: "v1", GraphDigest: teamPlanDigest(normalized), RunID: normalized.RunID, SessionID: normalized.SessionID, ParentRunID: normalized.ParentRunID, CausationID: normalized.CausationID, AuthorityDigest: normalized.AuthorityDigest, PlanDigest: normalized.PlanDigest, WriteScopeDigest: normalized.WriteScopeDigest, EffectNamespace: normalized.EffectNamespace, OccurrenceID: normalized.OccurrenceID,
		Strategy: StrategyParallel, PauseReason: TeamPauseInputRequired, Tasks: []TeamTaskContinuation{{TaskID: "settled", Status: TaskStatusSucceeded}, {TaskID: "pending", Status: TaskStatusPending}}, PendingTaskIDs: []string{"pending"}, Generation: 1, CreatedAt: now, ExpiresAt: now.Add(time.Hour),
		HostContextBindings: map[string]hostprojection.SourceBinding{"pending": {Form: hostprojection.SourceFormTeam, OwnerNamespace: "team-owner", SourceID: normalized.RunID, NodeID: "pending", ContinuationID: normalized.TeamID, ContinuationDigest: teamPlanDigest(normalized), Generation: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(context.Background(), continuation); err != nil {
		t.Fatal(err)
	}
	projection := hostprojection.Descriptor{SchemaVersion: hostprojection.SchemaVersionV1, ProjectionID: "team-pending-context", ProjectionVersion: 1, CapabilityDigest: "capability", Capabilities: []hostprojection.Capability{hostprojection.CapabilityRead}, SourceReferences: []hostprojection.SourceReference{{Reference: "opaque://team/context", Digest: "team-source"}}, Binding: pointerTeamBinding(continuation.HostContextBindings["pending"])}
	projection.Digest, err = hostprojection.CanonicalDigest(projection)
	if err != nil {
		t.Fatal(err)
	}
	resolver := teamContextResolverFunc(func(context.Context, hostprojection.SourceReference) (hostprojection.Material, error) {
		resolverCalls++
		return hostprojection.Material{Role: "private", Content: "team-authorized"}, nil
	})
	request := TeamResumeRequest{Ref: continuation.Ref, GraphDigest: continuation.GraphDigest, Generation: continuation.Generation, Strategy: continuation.Strategy, IdempotencyKey: "resume-context", AuthorityDigest: continuation.AuthorityDigest, PlanDigest: continuation.PlanDigest, WriteScopeDigest: continuation.WriteScopeDigest, EffectNamespace: continuation.EffectNamespace, OccurrenceID: continuation.OccurrenceID, CausationID: continuation.CausationID, Deployment: types.DeploymentRequest{OwnerNamespace: "team-owner", WriterFence: "fence:team-owner"}, HostContextProjections: map[string]*hostprojection.Descriptor{"pending": &projection}, HostContextResolver: resolver}
	result, err := New(WithTeamContinuationStore(store, "team-owner")).ResumeContinuation(context.Background(), plan, request, nil)
	if err != nil || result.Tasks[0].Status != TaskStatusSucceeded || result.Tasks[1].Status != TaskStatusSucceeded || settledCalls != 0 || pendingCalls != 1 || contextCalls != 1 || resolverCalls != 1 {
		t.Fatalf("result=%#v err=%v settled=%d pending=%d context=%d resolver=%d", result, err, settledCalls, pendingCalls, contextCalls, resolverCalls)
	}
}

func TestTeamPausePersistsOwnerBoundContextOnlyForPendingMember(t *testing.T) {
	store, err := NewFileTeamContinuationStore(filepath.Join(t.TempDir(), "team-pause-context.json"), "team-owner")
	if err != nil {
		t.Fatal(err)
	}
	plan := Plan{
		RunID: "run-team-pause-context", SessionID: "session-team-pause-context", ParentRunID: "parent", CausationID: "cause",
		AuthorityDigest: "authority", PlanDigest: "plan", WriteScopeDigest: "scope", EffectNamespace: "effects", OccurrenceID: "occurrence",
		TeamID: "team-pause-context", Strategy: StrategyParallel,
		HostContextBindings: map[string]hostprojection.SourceBinding{
			"pending": {Form: hostprojection.SourceFormTeam},
		},
		Tasks: []Task{
			{TaskID: "settled", AgentID: "agent-a", Role: RoleWorker, Runner: TaskRunnerFunc(func(context.Context, Task) (TaskResult, error) { return TaskResult{}, nil })},
			{TaskID: "pending", AgentID: "agent-b", Role: RoleWorker, Runner: TaskRunnerFunc(func(context.Context, Task) (TaskResult, error) { return TaskResult{}, nil })},
		},
	}
	engine := New(WithTeamContinuationStore(store, "team-owner"))
	result, err := engine.persistPausedResult(context.Background(), plan, []TaskRecord{
		{TaskID: "settled", Status: TaskStatusSucceeded},
		{TaskID: "pending", Status: TaskStatusPending},
	}, TeamPauseError{Reason: TeamPauseInputRequired, PendingActionReference: "input"})
	if err != nil || result.TeamCheckpoint == nil {
		t.Fatalf("persist pause result=%#v err=%v", result, err)
	}
	continuation, err := store.Load(context.Background(), *result.TeamCheckpoint)
	if err != nil {
		t.Fatal(err)
	}
	want := hostprojection.SourceBinding{Form: hostprojection.SourceFormTeam, OwnerNamespace: "team-owner", SourceID: plan.RunID, NodeID: "pending", ContinuationID: plan.TeamID, ContinuationDigest: continuation.GraphDigest, Generation: continuation.Generation}
	if len(continuation.HostContextBindings) != 1 || continuation.HostContextBindings["pending"] != want {
		t.Fatalf("persisted bindings=%#v, want only pending member binding %#v", continuation.HostContextBindings, want)
	}
}

func TestTeamContinuationRejectsForeignMemberProjectionBeforeResolverOrClaim(t *testing.T) {
	store, err := NewFileTeamContinuationStore(filepath.Join(t.TempDir(), "team-foreign-context.json"), "team-owner")
	if err != nil {
		t.Fatal(err)
	}
	plan := Plan{RunID: "run-team-foreign", SessionID: "session-team-foreign", ParentRunID: "parent", CausationID: "cause", AuthorityDigest: "authority", PlanDigest: "plan", WriteScopeDigest: "scope", EffectNamespace: "effects", OccurrenceID: "occurrence", TeamID: "team-foreign", Strategy: StrategyParallel, Tasks: []Task{
		{TaskID: "pending", AgentID: "agent", Role: RoleWorker, Runner: teamContextRunnerFunc(func(context.Context, Plan, Task, hostprojection.Resolution) (TaskResult, error) {
			t.Fatal("foreign member activated")
			return TaskResult{}, nil
		})},
	}}
	normalized, err := normalizePlan(plan)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	continuation, err := SealTeamContinuation(TeamContinuation{
		Ref: TeamContinuationRef{OwnerNamespace: "team-owner"}, TeamID: normalized.TeamID, GraphVersion: "v1", GraphDigest: teamPlanDigest(normalized),
		RunID: normalized.RunID, SessionID: normalized.SessionID, ParentRunID: normalized.ParentRunID, CausationID: normalized.CausationID,
		AuthorityDigest: normalized.AuthorityDigest, PlanDigest: normalized.PlanDigest, WriteScopeDigest: normalized.WriteScopeDigest, EffectNamespace: normalized.EffectNamespace, OccurrenceID: normalized.OccurrenceID,
		Strategy: StrategyParallel, PauseReason: TeamPauseInputRequired, Tasks: []TeamTaskContinuation{{TaskID: "pending", Status: TaskStatusPending}}, PendingTaskIDs: []string{"pending"},
		Generation: 1, CreatedAt: now, ExpiresAt: now.Add(time.Hour),
		HostContextBindings: map[string]hostprojection.SourceBinding{"pending": {Form: hostprojection.SourceFormTeam, OwnerNamespace: "team-owner", SourceID: normalized.RunID, NodeID: "pending", ContinuationID: normalized.TeamID, ContinuationDigest: teamPlanDigest(normalized), Generation: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(context.Background(), continuation); err != nil {
		t.Fatal(err)
	}
	foreign := continuation.HostContextBindings["pending"]
	foreign.NodeID = "sibling"
	projection := hostprojection.Descriptor{SchemaVersion: hostprojection.SchemaVersionV1, ProjectionID: "foreign-team-member", ProjectionVersion: 1, CapabilityDigest: "capability", Capabilities: []hostprojection.Capability{hostprojection.CapabilityRead}, SourceReferences: []hostprojection.SourceReference{{Reference: "opaque://team/context", Digest: "team-source"}}, Binding: &foreign}
	projection.Digest, err = hostprojection.CanonicalDigest(projection)
	if err != nil {
		t.Fatal(err)
	}
	resolverCalls := 0
	request := TeamResumeRequest{Ref: continuation.Ref, GraphDigest: continuation.GraphDigest, Generation: continuation.Generation, Strategy: continuation.Strategy, IdempotencyKey: "resume-foreign", AuthorityDigest: continuation.AuthorityDigest, PlanDigest: continuation.PlanDigest, WriteScopeDigest: continuation.WriteScopeDigest, EffectNamespace: continuation.EffectNamespace, OccurrenceID: continuation.OccurrenceID, CausationID: continuation.CausationID, Deployment: types.DeploymentRequest{OwnerNamespace: "team-owner", WriterFence: "fence:team-owner"}, HostContextProjections: map[string]*hostprojection.Descriptor{"pending": &projection}, HostContextResolver: teamContextResolverFunc(func(context.Context, hostprojection.SourceReference) (hostprojection.Material, error) {
		resolverCalls++
		return hostprojection.Material{Content: "private"}, nil
	})}
	_, err = New(WithTeamContinuationStore(store, "team-owner")).ResumeContinuation(context.Background(), plan, request, nil)
	if !errors.Is(err, ErrTeamContinuationConflict) || resolverCalls != 0 {
		t.Fatalf("ResumeContinuation() error=%v resolverCalls=%d, want conflict before resolver", err, resolverCalls)
	}
	claim, err := store.ClaimResume(context.Background(), request)
	if err != nil || claim.Duplicate {
		t.Fatalf("ClaimResume after rejected projection=%#v err=%v, want unconsumed claim", claim, err)
	}
}

func TestTeamContextResumeRechecksDeadlineAfterResolver(t *testing.T) {
	store, err := NewFileTeamContinuationStore(filepath.Join(t.TempDir(), "team-expiring-context.json"), "team-owner")
	if err != nil {
		t.Fatal(err)
	}
	plan := Plan{RunID: "run-team-expiring", SessionID: "session-team-expiring", ParentRunID: "parent", CausationID: "cause", AuthorityDigest: "authority", PlanDigest: "plan", WriteScopeDigest: "scope", EffectNamespace: "effects", OccurrenceID: "occurrence", TeamID: "team-expiring", Strategy: StrategyParallel, Tasks: []Task{{TaskID: "pending", AgentID: "agent", Role: RoleWorker, Runner: teamContextRunnerFunc(func(context.Context, Plan, Task, hostprojection.Resolution) (TaskResult, error) {
		t.Fatal("expired member activated")
		return TaskResult{}, nil
	})}}}
	normalized, err := normalizePlan(plan)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	deadline := now.Add(time.Second)
	continuation, err := SealTeamContinuation(TeamContinuation{Ref: TeamContinuationRef{OwnerNamespace: "team-owner"}, TeamID: normalized.TeamID, GraphVersion: "v1", GraphDigest: teamPlanDigest(normalized), RunID: normalized.RunID, SessionID: normalized.SessionID, ParentRunID: normalized.ParentRunID, CausationID: normalized.CausationID, AuthorityDigest: normalized.AuthorityDigest, PlanDigest: normalized.PlanDigest, WriteScopeDigest: normalized.WriteScopeDigest, EffectNamespace: normalized.EffectNamespace, OccurrenceID: normalized.OccurrenceID, Strategy: StrategyParallel, PauseReason: TeamPauseBudgetExhausted, Tasks: []TeamTaskContinuation{{TaskID: "pending", Status: TaskStatusPending}}, PendingTaskIDs: []string{"pending"}, OriginalDeadline: deadline, Generation: 1, CreatedAt: now, ExpiresAt: now.Add(time.Hour), HostContextBindings: map[string]hostprojection.SourceBinding{"pending": {Form: hostprojection.SourceFormTeam, OwnerNamespace: "team-owner", SourceID: normalized.RunID, NodeID: "pending", ContinuationID: normalized.TeamID, ContinuationDigest: teamPlanDigest(normalized), Generation: 1}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(context.Background(), continuation); err != nil {
		t.Fatal(err)
	}
	projection := hostprojection.Descriptor{SchemaVersion: hostprojection.SchemaVersionV1, ProjectionID: "expiring-team-context", ProjectionVersion: 1, CapabilityDigest: "capability", Capabilities: []hostprojection.Capability{hostprojection.CapabilityRead}, SourceReferences: []hostprojection.SourceReference{{Reference: "opaque://team/context", Digest: "team-source"}}, Binding: pointerTeamBinding(continuation.HostContextBindings["pending"])}
	projection.Digest, err = hostprojection.CanonicalDigest(projection)
	if err != nil {
		t.Fatal(err)
	}
	request := TeamResumeRequest{Ref: continuation.Ref, GraphDigest: continuation.GraphDigest, Generation: continuation.Generation, Strategy: continuation.Strategy, IdempotencyKey: "resume-expiring", AuthorityDigest: continuation.AuthorityDigest, PlanDigest: continuation.PlanDigest, WriteScopeDigest: continuation.WriteScopeDigest, EffectNamespace: continuation.EffectNamespace, OccurrenceID: continuation.OccurrenceID, CausationID: continuation.CausationID, Deployment: types.DeploymentRequest{OwnerNamespace: "team-owner", WriterFence: "fence:team-owner"}, HostContextProjections: map[string]*hostprojection.Descriptor{"pending": &projection}, HostContextResolver: teamContextResolverFunc(func(context.Context, hostprojection.SourceReference) (hostprojection.Material, error) {
		return hostprojection.Material{Content: "authorized"}, nil
	})}
	engine := New(WithTeamContinuationStore(store, "team-owner"))
	nowCalls := 0
	engine.now = func() time.Time {
		nowCalls++
		if nowCalls == 1 {
			return deadline.Add(-time.Millisecond)
		}
		return deadline
	}
	_, err = engine.ResumeContinuation(context.Background(), plan, request, nil)
	if !errors.Is(err, ErrTeamContinuationRecoveryRequired) {
		t.Fatalf("ResumeContinuation() error=%v, want recovery required", err)
	}
	claim, err := store.ClaimResume(context.Background(), request)
	if err != nil || claim.Duplicate {
		t.Fatalf("ClaimResume after resolver expiry=%#v err=%v, want unconsumed claim", claim, err)
	}
}

func TestTeamContextResumeKeepsParallelSiblingUnbound(t *testing.T) {
	store, err := NewFileTeamContinuationStore(filepath.Join(t.TempDir(), "team-sibling-context.json"), "team-owner")
	if err != nil {
		t.Fatal(err)
	}
	boundCalls, siblingCalls, resolverCalls := 0, 0, 0
	plan := Plan{RunID: "run-team-sibling", SessionID: "session-team-sibling", ParentRunID: "parent", CausationID: "cause", AuthorityDigest: "authority", PlanDigest: "plan", WriteScopeDigest: "scope", EffectNamespace: "effects", OccurrenceID: "occurrence", TeamID: "team-sibling", Strategy: StrategyParallel, Tasks: []Task{
		{TaskID: "bound", AgentID: "agent-bound", Role: RoleWorker, Runner: teamContextRunnerFunc(func(_ context.Context, _ Plan, _ Task, resolution hostprojection.Resolution) (TaskResult, error) {
			boundCalls++
			if got := resolution.Materials(); len(got) != 1 || got[0].Content != "bound-only" {
				t.Fatalf("bound materials=%#v", got)
			}
			return TaskResult{Output: "bound"}, nil
		})},
		{TaskID: "sibling", AgentID: "agent-sibling", Role: RoleWorker, Runner: TaskRunnerFunc(func(context.Context, Task) (TaskResult, error) {
			siblingCalls++
			return TaskResult{Output: "sibling"}, nil
		})},
	}}
	normalized, err := normalizePlan(plan)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	continuation, err := SealTeamContinuation(TeamContinuation{Ref: TeamContinuationRef{OwnerNamespace: "team-owner"}, TeamID: normalized.TeamID, GraphVersion: "v1", GraphDigest: teamPlanDigest(normalized), RunID: normalized.RunID, SessionID: normalized.SessionID, ParentRunID: normalized.ParentRunID, CausationID: normalized.CausationID, AuthorityDigest: normalized.AuthorityDigest, PlanDigest: normalized.PlanDigest, WriteScopeDigest: normalized.WriteScopeDigest, EffectNamespace: normalized.EffectNamespace, OccurrenceID: normalized.OccurrenceID, Strategy: StrategyParallel, PauseReason: TeamPauseInputRequired, Tasks: []TeamTaskContinuation{{TaskID: "bound", Status: TaskStatusPending}, {TaskID: "sibling", Status: TaskStatusPending}}, PendingTaskIDs: []string{"bound", "sibling"}, Generation: 1, CreatedAt: now, ExpiresAt: now.Add(time.Hour), HostContextBindings: map[string]hostprojection.SourceBinding{"bound": {Form: hostprojection.SourceFormTeam, OwnerNamespace: "team-owner", SourceID: normalized.RunID, NodeID: "bound", ContinuationID: normalized.TeamID, ContinuationDigest: teamPlanDigest(normalized), Generation: 1}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(context.Background(), continuation); err != nil {
		t.Fatal(err)
	}
	projection := hostprojection.Descriptor{SchemaVersion: hostprojection.SchemaVersionV1, ProjectionID: "bound-team-member", ProjectionVersion: 1, CapabilityDigest: "capability", Capabilities: []hostprojection.Capability{hostprojection.CapabilityRead}, SourceReferences: []hostprojection.SourceReference{{Reference: "opaque://team/context", Digest: "team-source"}}, Binding: pointerTeamBinding(continuation.HostContextBindings["bound"])}
	projection.Digest, err = hostprojection.CanonicalDigest(projection)
	if err != nil {
		t.Fatal(err)
	}
	request := TeamResumeRequest{Ref: continuation.Ref, GraphDigest: continuation.GraphDigest, Generation: continuation.Generation, Strategy: continuation.Strategy, IdempotencyKey: "resume-sibling", AuthorityDigest: continuation.AuthorityDigest, PlanDigest: continuation.PlanDigest, WriteScopeDigest: continuation.WriteScopeDigest, EffectNamespace: continuation.EffectNamespace, OccurrenceID: continuation.OccurrenceID, CausationID: continuation.CausationID, Deployment: types.DeploymentRequest{OwnerNamespace: "team-owner", WriterFence: "fence:team-owner"}, HostContextProjections: map[string]*hostprojection.Descriptor{"bound": &projection}, HostContextResolver: teamContextResolverFunc(func(context.Context, hostprojection.SourceReference) (hostprojection.Material, error) {
		resolverCalls++
		return hostprojection.Material{Content: "bound-only"}, nil
	})}
	result, err := New(WithTeamContinuationStore(store, "team-owner")).ResumeContinuation(context.Background(), plan, request, nil)
	if err != nil || boundCalls != 1 || siblingCalls != 1 || resolverCalls != 1 || result.TeamTaskFailed != 0 {
		t.Fatalf("result=%#v err=%v bound=%d sibling=%d resolver=%d", result, err, boundCalls, siblingCalls, resolverCalls)
	}
}

func TestTeamContinuationRejectsContextBindingForSettledMember(t *testing.T) {
	now := time.Now().UTC()
	_, err := SealTeamContinuation(TeamContinuation{
		Ref: TeamContinuationRef{OwnerNamespace: "team-owner"}, TeamID: "team-settled-binding", GraphVersion: "v1", GraphDigest: "sha256:graph",
		RunID: "run-settled-binding", SessionID: "session-settled-binding", AuthorityDigest: "authority", PlanDigest: "plan", WriteScopeDigest: "scope", EffectNamespace: "effects", OccurrenceID: "occurrence",
		Strategy: StrategyParallel, PauseReason: TeamPauseInputRequired, Tasks: []TeamTaskContinuation{{TaskID: "settled", Status: TaskStatusSucceeded}},
		Generation: 1, CreatedAt: now, ExpiresAt: now.Add(time.Hour),
		HostContextBindings: map[string]hostprojection.SourceBinding{"settled": {Form: hostprojection.SourceFormTeam, OwnerNamespace: "team-owner", SourceID: "run-settled-binding", NodeID: "settled", ContinuationID: "team-settled-binding", ContinuationDigest: "sha256:graph", Generation: 1}},
	})
	if !errors.Is(err, ErrTeamContinuationForeign) {
		t.Fatalf("SealTeamContinuation() error=%v, want settled member binding rejected", err)
	}
}

func TestTeamPauseRejectsRemoteMemberContextBinding(t *testing.T) {
	store, err := NewFileTeamContinuationStore(filepath.Join(t.TempDir(), "team-remote-context.json"), "team-owner")
	if err != nil {
		t.Fatal(err)
	}
	plan := Plan{
		RunID: "run-team-remote-context", SessionID: "session-team-remote-context", ParentRunID: "parent", CausationID: "cause",
		AuthorityDigest: "authority", PlanDigest: "plan", WriteScopeDigest: "scope", EffectNamespace: "effects", OccurrenceID: "occurrence",
		TeamID: "team-remote-context", Strategy: StrategyParallel,
		HostContextBindings: map[string]hostprojection.SourceBinding{"remote": {Form: hostprojection.SourceFormTeam}},
		Tasks:               []Task{{TaskID: "remote", AgentID: "agent", Role: RoleWorker, Target: TaskTargetRemote, Remote: RemoteTarget{PeerID: "peer"}, RemoteRunner: RemoteTaskRunnerFunc(func(context.Context, Plan, Task) (TaskResult, error) { return TaskResult{}, nil })}},
	}
	_, err = New(WithTeamContinuationStore(store, "team-owner")).persistPausedResult(context.Background(), plan, []TaskRecord{{TaskID: "remote", Status: TaskStatusPending}}, TeamPauseError{Reason: TeamPauseInputRequired})
	if !errors.Is(err, ErrTeamContinuationUnsupported) {
		t.Fatalf("persistPausedResult() error=%v, want remote context binding unsupported", err)
	}
}

func TestTeamVoteContinuationResolvesBoundPendingMemberWithoutRerunningSettledVote(t *testing.T) {
	store, err := NewFileTeamContinuationStore(filepath.Join(t.TempDir(), "team-vote-context.json"), "team-owner")
	if err != nil {
		t.Fatal(err)
	}
	settledCalls, contextCalls := 0, 0
	plan := Plan{RunID: "run-team-vote-context", SessionID: "session-team-vote-context", ParentRunID: "parent", CausationID: "cause", AuthorityDigest: "authority", PlanDigest: "plan", WriteScopeDigest: "scope", EffectNamespace: "effects", OccurrenceID: "occurrence", TeamID: "team-vote-context", Strategy: StrategyVote, VoteTieBreak: VoteTieBreakFirstTaskID, Tasks: []Task{
		{TaskID: "settled", AgentID: "agent-settled", Role: RoleWorker, Runner: TaskRunnerFunc(func(context.Context, Task) (TaskResult, error) {
			settledCalls++
			return TaskResult{Vote: "unexpected"}, nil
		})},
		{TaskID: "pending", AgentID: "agent-pending", Role: RoleWorker, Runner: teamContextRunnerFunc(func(_ context.Context, _ Plan, _ Task, resolution hostprojection.Resolution) (TaskResult, error) {
			contextCalls++
			if got := resolution.Materials(); len(got) != 1 || got[0].Content != "vote-context" {
				t.Fatalf("materials=%#v", got)
			}
			return TaskResult{Vote: "yes"}, nil
		})},
	}}
	normalized, err := normalizePlan(plan)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	continuation, err := SealTeamContinuation(TeamContinuation{Ref: TeamContinuationRef{OwnerNamespace: "team-owner"}, TeamID: normalized.TeamID, GraphVersion: "v1", GraphDigest: teamPlanDigest(normalized), RunID: normalized.RunID, SessionID: normalized.SessionID, ParentRunID: normalized.ParentRunID, CausationID: normalized.CausationID, AuthorityDigest: normalized.AuthorityDigest, PlanDigest: normalized.PlanDigest, WriteScopeDigest: normalized.WriteScopeDigest, EffectNamespace: normalized.EffectNamespace, OccurrenceID: normalized.OccurrenceID, Strategy: StrategyVote, PauseReason: TeamPauseApprovalRequired, Tasks: []TeamTaskContinuation{{TaskID: "settled", Status: TaskStatusSucceeded, Vote: "yes", EffectReference: "effect-settled"}, {TaskID: "pending", Status: TaskStatusPending}}, PendingTaskIDs: []string{"pending"}, Generation: 1, CreatedAt: now, ExpiresAt: now.Add(time.Hour), HostContextBindings: map[string]hostprojection.SourceBinding{"pending": {Form: hostprojection.SourceFormTeam, OwnerNamespace: "team-owner", SourceID: normalized.RunID, NodeID: "pending", ContinuationID: normalized.TeamID, ContinuationDigest: teamPlanDigest(normalized), Generation: 1}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(context.Background(), continuation); err != nil {
		t.Fatal(err)
	}
	projection := hostprojection.Descriptor{SchemaVersion: hostprojection.SchemaVersionV1, ProjectionID: "team-vote-pending", ProjectionVersion: 1, CapabilityDigest: "capability", Capabilities: []hostprojection.Capability{hostprojection.CapabilityRead}, SourceReferences: []hostprojection.SourceReference{{Reference: "opaque://team/vote", Digest: "team-vote-source"}}, Binding: pointerTeamBinding(continuation.HostContextBindings["pending"])}
	projection.Digest, err = hostprojection.CanonicalDigest(projection)
	if err != nil {
		t.Fatal(err)
	}
	request := TeamResumeRequest{Ref: continuation.Ref, GraphDigest: continuation.GraphDigest, Generation: continuation.Generation, Strategy: continuation.Strategy, IdempotencyKey: "resume-vote-context", AuthorityDigest: continuation.AuthorityDigest, PlanDigest: continuation.PlanDigest, WriteScopeDigest: continuation.WriteScopeDigest, EffectNamespace: continuation.EffectNamespace, OccurrenceID: continuation.OccurrenceID, CausationID: continuation.CausationID, Deployment: types.DeploymentRequest{OwnerNamespace: "team-owner", WriterFence: "fence:team-owner"}, HostContextProjections: map[string]*hostprojection.Descriptor{"pending": &projection}, HostContextResolver: teamContextResolverFunc(func(context.Context, hostprojection.SourceReference) (hostprojection.Material, error) {
		return hostprojection.Material{Content: "vote-context"}, nil
	})}
	result, err := New(WithTeamContinuationStore(store, "team-owner")).ResumeContinuation(context.Background(), plan, request, nil)
	if err != nil || settledCalls != 0 || contextCalls != 1 || result.WinnerVote != "yes" || result.Tasks[0].Vote != "yes" || result.Tasks[1].Vote != "yes" {
		t.Fatalf("result=%#v err=%v settled=%d context=%d", result, err, settledCalls, contextCalls)
	}
}

func TestTeamContextResumeRunStreamParity(t *testing.T) {
	type setup struct {
		engine  *Engine
		plan    Plan
		request TeamResumeRequest
		calls   *int
	}
	build := func(t *testing.T, key string) setup {
		t.Helper()
		store, err := NewFileTeamContinuationStore(filepath.Join(t.TempDir(), key+".json"), "team-owner")
		if err != nil {
			t.Fatal(err)
		}
		calls := 0
		plan := Plan{RunID: "run-team-context-parity", SessionID: "session-team-context-parity", ParentRunID: "parent", CausationID: "cause", AuthorityDigest: "authority", PlanDigest: "plan", WriteScopeDigest: "scope", EffectNamespace: "effects", OccurrenceID: "occurrence", TeamID: "team-context-parity", Strategy: StrategyParallel, Tasks: []Task{{TaskID: "pending", AgentID: "agent", Role: RoleWorker, Runner: teamContextRunnerFunc(func(_ context.Context, _ Plan, _ Task, resolution hostprojection.Resolution) (TaskResult, error) {
			calls++
			if got := resolution.Materials(); len(got) != 1 || got[0].Content != "parity-context" {
				t.Fatalf("materials=%#v", got)
			}
			return TaskResult{Output: "ok"}, nil
		})}}}
		normalized, err := normalizePlan(plan)
		if err != nil {
			t.Fatal(err)
		}
		now := time.Now().UTC()
		continuation, err := SealTeamContinuation(TeamContinuation{Ref: TeamContinuationRef{OwnerNamespace: "team-owner"}, TeamID: normalized.TeamID, GraphVersion: "v1", GraphDigest: teamPlanDigest(normalized), RunID: normalized.RunID, SessionID: normalized.SessionID, ParentRunID: normalized.ParentRunID, CausationID: normalized.CausationID, AuthorityDigest: normalized.AuthorityDigest, PlanDigest: normalized.PlanDigest, WriteScopeDigest: normalized.WriteScopeDigest, EffectNamespace: normalized.EffectNamespace, OccurrenceID: normalized.OccurrenceID, Strategy: StrategyParallel, PauseReason: TeamPauseInputRequired, Tasks: []TeamTaskContinuation{{TaskID: "pending", Status: TaskStatusPending}}, PendingTaskIDs: []string{"pending"}, Generation: 1, CreatedAt: now, ExpiresAt: now.Add(time.Hour), HostContextBindings: map[string]hostprojection.SourceBinding{"pending": {Form: hostprojection.SourceFormTeam, OwnerNamespace: "team-owner", SourceID: normalized.RunID, NodeID: "pending", ContinuationID: normalized.TeamID, ContinuationDigest: teamPlanDigest(normalized), Generation: 1}}})
		if err != nil {
			t.Fatal(err)
		}
		if err := store.Save(context.Background(), continuation); err != nil {
			t.Fatal(err)
		}
		projection := hostprojection.Descriptor{SchemaVersion: hostprojection.SchemaVersionV1, ProjectionID: "team-context-parity", ProjectionVersion: 1, CapabilityDigest: "capability", Capabilities: []hostprojection.Capability{hostprojection.CapabilityRead}, SourceReferences: []hostprojection.SourceReference{{Reference: "opaque://team/parity", Digest: "team-parity-source"}}, Binding: pointerTeamBinding(continuation.HostContextBindings["pending"])}
		projection.Digest, err = hostprojection.CanonicalDigest(projection)
		if err != nil {
			t.Fatal(err)
		}
		request := TeamResumeRequest{Ref: continuation.Ref, GraphDigest: continuation.GraphDigest, Generation: continuation.Generation, Strategy: continuation.Strategy, IdempotencyKey: "resume-context-parity", AuthorityDigest: continuation.AuthorityDigest, PlanDigest: continuation.PlanDigest, WriteScopeDigest: continuation.WriteScopeDigest, EffectNamespace: continuation.EffectNamespace, OccurrenceID: continuation.OccurrenceID, CausationID: continuation.CausationID, Deployment: types.DeploymentRequest{OwnerNamespace: "team-owner", WriterFence: "fence:team-owner"}, HostContextProjections: map[string]*hostprojection.Descriptor{"pending": &projection}, HostContextResolver: teamContextResolverFunc(func(context.Context, hostprojection.SourceReference) (hostprojection.Material, error) {
			return hostprojection.Material{Content: "parity-context"}, nil
		})}
		return setup{engine: New(WithTeamContinuationStore(store, "team-owner")), plan: plan, request: request, calls: &calls}
	}
	run := build(t, "run")
	runResult, err := run.engine.ResumeContinuation(context.Background(), run.plan, run.request, nil)
	if err != nil {
		t.Fatalf("run resume: %v", err)
	}
	stream := build(t, "stream")
	events := 0
	streamResult, err := stream.engine.ResumeContinuation(context.Background(), stream.plan, stream.request, func(StreamEvent) error { events++; return nil })
	if err != nil {
		t.Fatalf("stream resume: %v", err)
	}
	if *run.calls != 1 || *stream.calls != 1 || events == 0 || runResult.TeamTaskTotal != streamResult.TeamTaskTotal || runResult.TeamTaskFailed != streamResult.TeamTaskFailed || len(runResult.Tasks) != len(streamResult.Tasks) || runResult.Tasks[0].Status != streamResult.Tasks[0].Status {
		t.Fatalf("run=%#v stream=%#v calls=(%d,%d) events=%d", runResult, streamResult, *run.calls, *stream.calls, events)
	}
}

func TestTeamContextResumeReopensFileOwnerWithBoundMember(t *testing.T) {
	path := filepath.Join(t.TempDir(), "team-context-reopen.json")
	store, err := NewFileTeamContinuationStore(path, "team-owner")
	if err != nil {
		t.Fatal(err)
	}
	contextCalls := 0
	plan := Plan{RunID: "run-team-context-reopen", SessionID: "session-team-context-reopen", ParentRunID: "parent", CausationID: "cause", AuthorityDigest: "authority", PlanDigest: "plan", WriteScopeDigest: "scope", EffectNamespace: "effects", OccurrenceID: "occurrence", TeamID: "team-context-reopen", Strategy: StrategyParallel, Tasks: []Task{{TaskID: "pending", AgentID: "agent", Role: RoleWorker, Runner: teamContextRunnerFunc(func(_ context.Context, _ Plan, _ Task, resolution hostprojection.Resolution) (TaskResult, error) {
		contextCalls++
		if got := resolution.Materials(); len(got) != 1 || got[0].Content != "reopened-context" {
			t.Fatalf("materials=%#v", got)
		}
		return TaskResult{Output: "ok"}, nil
	})}}}
	normalized, err := normalizePlan(plan)
	if err != nil {
		t.Fatal(err)
	}
	normalized.HostContextBindings = map[string]hostprojection.SourceBinding{"pending": {Form: hostprojection.SourceFormTeam}}
	first := New(WithTeamContinuationStore(store, "team-owner"))
	paused, err := first.persistPausedResult(context.Background(), normalized, []TaskRecord{{TaskID: "pending", Status: TaskStatusPending}}, TeamPauseError{Reason: TeamPauseInputRequired})
	if err != nil || paused.TeamCheckpoint == nil {
		t.Fatalf("persistPausedResult() result=%#v err=%v", paused, err)
	}
	reopenedStore, err := NewFileTeamContinuationStore(path, "team-owner")
	if err != nil {
		t.Fatal(err)
	}
	continuation, err := reopenedStore.Load(context.Background(), *paused.TeamCheckpoint)
	if err != nil || continuation.HostContextBindings["pending"].NodeID != "pending" {
		t.Fatalf("reopened continuation=%#v err=%v", continuation, err)
	}
	projection := hostprojection.Descriptor{SchemaVersion: hostprojection.SchemaVersionV1, ProjectionID: "team-context-reopen", ProjectionVersion: 1, CapabilityDigest: "capability", Capabilities: []hostprojection.Capability{hostprojection.CapabilityRead}, SourceReferences: []hostprojection.SourceReference{{Reference: "opaque://team/reopen", Digest: "team-reopen-source"}}, Binding: pointerTeamBinding(continuation.HostContextBindings["pending"])}
	projection.Digest, err = hostprojection.CanonicalDigest(projection)
	if err != nil {
		t.Fatal(err)
	}
	request := TeamResumeRequest{Ref: continuation.Ref, GraphDigest: continuation.GraphDigest, Generation: continuation.Generation, Strategy: continuation.Strategy, IdempotencyKey: "resume-context-reopen", AuthorityDigest: continuation.AuthorityDigest, PlanDigest: continuation.PlanDigest, WriteScopeDigest: continuation.WriteScopeDigest, EffectNamespace: continuation.EffectNamespace, OccurrenceID: continuation.OccurrenceID, CausationID: continuation.CausationID, Deployment: types.DeploymentRequest{OwnerNamespace: "team-owner", WriterFence: "fence:team-owner"}, HostContextProjections: map[string]*hostprojection.Descriptor{"pending": &projection}, HostContextResolver: teamContextResolverFunc(func(context.Context, hostprojection.SourceReference) (hostprojection.Material, error) {
		return hostprojection.Material{Content: "reopened-context"}, nil
	})}
	result, err := New(WithTeamContinuationStore(reopenedStore, "team-owner")).ResumeContinuation(context.Background(), plan, request, nil)
	if err != nil || contextCalls != 1 || result.TeamTaskFailed != 0 {
		t.Fatalf("result=%#v err=%v contextCalls=%d", result, err, contextCalls)
	}
}

type teamContextRunnerFunc func(context.Context, Plan, Task, hostprojection.Resolution) (TaskResult, error)

func (f teamContextRunnerFunc) Run(context.Context, Task) (TaskResult, error) {
	return TaskResult{}, errors.New("context runner must receive bound context")
}
func (f teamContextRunnerFunc) RunWithHostContext(ctx context.Context, plan Plan, task Task, resolution hostprojection.Resolution) (TaskResult, error) {
	return f(ctx, plan, task, resolution)
}

type teamContextResolverFunc func(context.Context, hostprojection.SourceReference) (hostprojection.Material, error)

func (f teamContextResolverFunc) ResolveHostContext(ctx context.Context, ref hostprojection.SourceReference) (hostprojection.Material, error) {
	return f(ctx, ref)
}

func pointerTeamBinding(binding hostprojection.SourceBinding) *hostprojection.SourceBinding {
	return &binding
}

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
