package composer

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/FelixSeptem/baymax/context/hostprojection"
	"github.com/FelixSeptem/baymax/orchestration/scheduler"
)

func childPauseFromError(err error) (ChildPauseError, bool) {
	var pause ChildPauseError
	if errors.As(err, &pause) && validChildPauseReason(pause.Reason) {
		return pause, true
	}
	var pausePtr *ChildPauseError
	if errors.As(err, &pausePtr) && pausePtr != nil && validChildPauseReason(pausePtr.Reason) {
		return *pausePtr, true
	}
	return ChildPauseError{}, false
}

func (c *Composer) persistChildPause(ctx context.Context, req ChildDispatchRequest, claimed scheduler.ClaimedTask, pause ChildPauseError) (ChildContinuation, error) {
	graphID := strings.TrimSpace(req.GraphID)
	if graphID == "" {
		graphID = strings.TrimSpace(claimed.Record.Task.DispatchBinding)
	}
	if graphID == "" {
		graphID = strings.TrimSpace(claimed.Record.Task.TaskID)
	}
	graphVersion := strings.TrimSpace(req.GraphVersion)
	if graphVersion == "" {
		graphVersion = "v1"
	}
	graphDigest := strings.TrimSpace(req.GraphDigest)
	if graphDigest == "" {
		graphDigest = graphID
	}
	var binding *hostprojection.SourceBinding
	if req.HostContextBinding != nil {
		bound := *req.HostContextBinding
		bound.Form = hostprojection.SourceFormChild
		bound.OwnerNamespace = c.childContinuationNamespace
		bound.SourceID = claimed.Record.Task.RunID
		bound.NodeID = claimed.Record.Task.TaskID
		bound.ContinuationID = claimed.Attempt.AttemptID
		bound.ContinuationDigest = graphDigest
		bound.Generation = 1
		binding = &bound
	}
	continuation := ChildContinuation{
		Ref:    ChildContinuationRef{OwnerNamespace: c.childContinuationNamespace, GraphID: graphID, GraphVersion: graphVersion, RunID: claimed.Record.Task.RunID, SessionID: strings.TrimSpace(req.SessionID), ParentRunID: claimed.Record.Task.ParentRunID, CausationID: strings.TrimSpace(req.CausationID), OccurrenceID: strings.TrimSpace(req.OccurrenceID)},
		Target: req.Target, Claimed: claimed, GraphID: graphID, GraphVersion: graphVersion, GraphDigest: graphDigest,
		ParentRunID: claimed.Record.Task.ParentRunID, RunID: claimed.Record.Task.RunID, SessionID: strings.TrimSpace(req.SessionID), CausationID: strings.TrimSpace(req.CausationID), AuthorityDigest: strings.TrimSpace(req.AuthorityDigest), PlanDigest: strings.TrimSpace(req.PlanDigest), WriteScopeDigest: strings.TrimSpace(req.WriteScopeDigest), EffectNamespace: strings.TrimSpace(req.EffectNamespace), OccurrenceID: strings.TrimSpace(req.OccurrenceID), Generation: 1, CreatedAt: c.now(), ExpiresAt: c.now().Add(24 * time.Hour),
		HostContextBinding: binding,
	}
	return c.PersistChildPause(ctx, continuation, pause)
}

// PersistChildPause records a source-owned safe point before dispatch resumes.
// The caller supplies the already detached graph/task material; this method
// only records pause facts and never changes scheduler state or executes work.
func (c *Composer) PersistChildPause(ctx context.Context, continuation ChildContinuation, pause ChildPauseError) (ChildContinuation, error) {
	if c == nil || c.childContinuationStore == nil {
		return ChildContinuation{}, ErrChildContinuationUnavailable
	}
	if !validChildPauseReason(pause.Reason) {
		return ChildContinuation{}, ErrChildContinuationUnsupported
	}
	continuation.PauseReason = pause.Reason
	continuation.PendingActionReference = strings.TrimSpace(pause.PendingActionReference)
	continuation.ConsumedBudget = pause.ConsumedBudget
	continuation.OriginalDeadline = pause.OriginalDeadline
	sealed, err := SealChildContinuation(continuation)
	if err != nil {
		return ChildContinuation{}, err
	}
	if err := c.childContinuationStore.Save(ctx, sealed); err != nil {
		return ChildContinuation{}, err
	}
	return sealed, nil
}

// ResumeChildContinuation restores one source-owned child safe point. It is
// deliberately separate from Recover: no scheduler snapshot is restored, no
// task is spawned or enqueued, and no callback history is replayed.
func (c *Composer) ResumeChildContinuation(ctx context.Context, request ChildResumeRequest) (ChildDispatchResult, error) {
	if c == nil || c.childContinuationStore == nil {
		return ChildDispatchResult{}, ErrChildContinuationUnavailable
	}
	if err := request.Validate(); err != nil {
		return ChildDispatchResult{}, err
	}
	continuation, err := c.childContinuationStore.Load(ctx, request.Ref)
	if err != nil {
		return ChildDispatchResult{}, err
	}
	if c.childContinuationNamespace != "" && continuation.Ref.OwnerNamespace != c.childContinuationNamespace {
		return ChildDispatchResult{}, ErrChildContinuationForeign
	}
	if err := validateChildResumeBinding(continuation, request); err != nil {
		return ChildDispatchResult{}, err
	}
	var resolved hostprojection.Resolution
	if request.HostContextProjection != nil {
		if continuation.HostContextBinding == nil {
			return ChildDispatchResult{}, ErrChildContinuationUnsupported
		}
		snapshot, admitErr := hostprojection.Admit(*request.HostContextProjection)
		if admitErr != nil || hostprojection.RequireBinding(snapshot, *continuation.HostContextBinding) != nil {
			return ChildDispatchResult{}, ErrChildContinuationConflict
		}
		if err := c.validateChildSchedulerState(ctx, continuation); err != nil {
			return ChildDispatchResult{}, err
		}
		if request.HostContextResolver == nil || c.childContextContinuationExecutor == nil {
			return ChildDispatchResult{}, ErrChildContinuationUnsupported
		}
		_, resolved, err = hostprojection.ResolveBound(ctx, *request.HostContextProjection, *continuation.HostContextBinding, request.HostContextResolver)
		if err != nil {
			return ChildDispatchResult{}, err
		}
		// Re-read the source owner's lease/attempt fence after current
		// authorization. Resolver callbacks are host-owned and may take long
		// enough for the scheduler state to change; never consume a resume claim
		// for a task that became terminal or lost its lease in that window.
		if err := c.validateChildSchedulerState(ctx, continuation); err != nil {
			return ChildDispatchResult{}, err
		}
	}
	claim, err := c.childContinuationStore.ClaimResume(ctx, request)
	if err != nil {
		return ChildDispatchResult{}, err
	}
	if claim.Result != nil {
		out := claim.Result.Detached()
		out.CommitMeta.Duplicate = true
		return out, nil
	}
	if request.HostContextProjection == nil && c.childContinuationExecutor == nil {
		return ChildDispatchResult{}, ErrChildContinuationUnsupported
	}
	if c.scheduler == nil {
		return ChildDispatchResult{}, ErrChildContinuationUnavailable
	}
	record, ok, err := c.scheduler.Get(ctx, continuation.Claimed.Record.Task.TaskID)
	if err != nil {
		return ChildDispatchResult{}, fmt.Errorf("load child task: %w", err)
	}
	if !ok || record.CurrentAttempt != continuation.Claimed.Attempt.AttemptID || record.State != scheduler.TaskStateRunning {
		return ChildDispatchResult{}, ErrChildContinuationRecoveryRequired
	}
	if !continuation.Claimed.Attempt.LeaseExpiresAt.IsZero() && !c.now().Before(continuation.Claimed.Attempt.LeaseExpiresAt) {
		return ChildDispatchResult{}, ErrChildContinuationRecoveryRequired
	}
	if _, err := c.childContinuationStore.MarkResumeRunning(ctx, claim); err != nil {
		return ChildDispatchResult{}, err
	}
	var commit scheduler.TerminalCommit
	if request.HostContextProjection != nil {
		commit, err = c.childContextContinuationExecutor.ResumeChildContinuationWithContext(ctx, continuation, resolved)
	} else {
		commit, err = c.childContinuationExecutor.ResumeChildContinuation(ctx, continuation)
	}
	if err != nil {
		return ChildDispatchResult{}, err
	}
	if strings.TrimSpace(commit.TaskID) != continuation.Claimed.Record.Task.TaskID || strings.TrimSpace(commit.AttemptID) != continuation.Claimed.Attempt.AttemptID {
		return ChildDispatchResult{}, ErrChildContinuationConflict
	}
	commitMeta, err := c.CommitChildTerminal(ctx, commit)
	if err != nil {
		return ChildDispatchResult{}, err
	}
	result := ChildDispatchResult{Record: record, Claimed: continuation.Claimed, Commit: commit, CommitMeta: commitMeta}
	if _, err := c.childContinuationStore.CommitResumeResult(ctx, claim, result); err != nil {
		return ChildDispatchResult{}, err
	}
	return result, nil
}

func (c *Composer) validateChildSchedulerState(ctx context.Context, continuation ChildContinuation) error {
	if c.scheduler == nil {
		return ErrChildContinuationUnavailable
	}
	record, ok, err := c.scheduler.Get(ctx, continuation.Claimed.Record.Task.TaskID)
	if err != nil {
		return fmt.Errorf("load child task: %w", err)
	}
	if !ok || record.CurrentAttempt != continuation.Claimed.Attempt.AttemptID || record.State != scheduler.TaskStateRunning {
		return ErrChildContinuationRecoveryRequired
	}
	if !continuation.Claimed.Attempt.LeaseExpiresAt.IsZero() && !c.now().Before(continuation.Claimed.Attempt.LeaseExpiresAt) {
		return ErrChildContinuationRecoveryRequired
	}
	return nil
}

func validateChildResumeBinding(continuation ChildContinuation, request ChildResumeRequest) error {
	if continuation.GraphDigest != request.GraphDigest || continuation.Generation != request.Generation || continuation.Target != request.Target {
		return ErrChildContinuationConflict
	}
	checks := [][2]string{
		{request.AuthorityDigest, continuation.AuthorityDigest},
		{request.PlanDigest, continuation.PlanDigest},
		{request.WriteScopeDigest, continuation.WriteScopeDigest},
		{request.EffectNamespace, continuation.EffectNamespace},
		{request.OccurrenceID, continuation.OccurrenceID},
		{request.CausationID, continuation.CausationID},
	}
	for _, pair := range checks {
		if strings.TrimSpace(pair[0]) != "" && strings.TrimSpace(pair[0]) != strings.TrimSpace(pair[1]) {
			return ErrChildContinuationConflict
		}
	}
	return nil
}
