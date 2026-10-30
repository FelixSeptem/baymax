package teams

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/FelixSeptem/baymax/context/hostprojection"
	"github.com/FelixSeptem/baymax/core/history"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/orchestration/collab"
)

const (
	ReasonDispatch       = "team.dispatch"
	ReasonCollect        = "team.collect"
	ReasonResolve        = "team.resolve"
	ReasonDispatchRemote = "team.dispatch_remote"
	ReasonCollectRemote  = "team.collect_remote"
	ReasonHandoff        = "team.handoff"
	ReasonDelegation     = "team.delegation"
	ReasonAggregation    = "team.aggregation"
)

var ErrTeamPaused = errors.New("team paused")

type Role string

const (
	RoleLeader      Role = "leader"
	RoleWorker      Role = "worker"
	RoleCoordinator Role = "coordinator"
)

type Strategy string

const (
	StrategySerial   Strategy = "serial"
	StrategyParallel Strategy = "parallel"
	StrategyVote     Strategy = "vote"
)

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusSucceeded TaskStatus = "succeeded"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusSkipped   TaskStatus = "skipped"
	TaskStatusCanceled  TaskStatus = "canceled"
)

type FailurePolicy string

const (
	FailurePolicyBestEffort FailurePolicy = "best_effort"
	FailurePolicyFailFast   FailurePolicy = "fail_fast"
)

type VoteTieBreak string

const (
	VoteTieBreakHighestPriority VoteTieBreak = "highest_priority"
	VoteTieBreakFirstTaskID     VoteTieBreak = "first_task_id"
)

type TaskRunner interface {
	Run(ctx context.Context, task Task) (TaskResult, error)
}

type HostContextTaskRunner interface {
	RunWithHostContext(context.Context, Plan, Task, hostprojection.Resolution) (TaskResult, error)
}

type TaskRunnerFunc func(ctx context.Context, task Task) (TaskResult, error)

func (f TaskRunnerFunc) Run(ctx context.Context, task Task) (TaskResult, error) {
	return f(ctx, task)
}

type RemoteTaskRunner interface {
	RunRemote(ctx context.Context, plan Plan, task Task) (TaskResult, error)
}

type RemoteTaskRunnerFunc func(ctx context.Context, plan Plan, task Task) (TaskResult, error)

func (f RemoteTaskRunnerFunc) RunRemote(ctx context.Context, plan Plan, task Task) (TaskResult, error) {
	return f(ctx, plan, task)
}

type TaskResult struct {
	Output any    `json:"output,omitempty"`
	Vote   string `json:"vote,omitempty"`
}

type TaskTarget string

const (
	TaskTargetLocal  TaskTarget = "local"
	TaskTargetRemote TaskTarget = "remote"
)

type RemoteTarget struct {
	PeerID               string         `json:"peer_id,omitempty"`
	Method               string         `json:"method,omitempty"`
	RequiredCapabilities []string       `json:"required_capabilities,omitempty"`
	Payload              map[string]any `json:"payload,omitempty"`
}

type Task struct {
	TaskID          string           `json:"task_id"`
	AgentID         string           `json:"agent_id"`
	Role            Role             `json:"role"`
	Priority        int              `json:"priority,omitempty"`
	CollabPrimitive string           `json:"collab_primitive,omitempty"`
	Target          TaskTarget       `json:"target,omitempty"`
	Remote          RemoteTarget     `json:"remote,omitempty"`
	Runner          TaskRunner       `json:"-"`
	RemoteRunner    RemoteTaskRunner `json:"-"`
}

type Plan struct {
	RunID                   string                                  `json:"run_id,omitempty"`
	SessionID               string                                  `json:"session_id,omitempty"`
	ParentRunID             string                                  `json:"parent_run_id,omitempty"`
	CausationID             string                                  `json:"causation_id,omitempty"`
	AuthorityDigest         string                                  `json:"authority_digest,omitempty"`
	PlanDigest              string                                  `json:"plan_digest,omitempty"`
	WriteScopeDigest        string                                  `json:"write_scope_digest,omitempty"`
	EffectNamespace         string                                  `json:"effect_namespace,omitempty"`
	OccurrenceID            string                                  `json:"occurrence_id,omitempty"`
	TeamID                  string                                  `json:"team_id"`
	WorkflowID              string                                  `json:"workflow_id,omitempty"`
	StepID                  string                                  `json:"step_id,omitempty"`
	Strategy                Strategy                                `json:"strategy,omitempty"`
	Tasks                   []Task                                  `json:"tasks"`
	TaskTimeout             time.Duration                           `json:"task_timeout,omitempty"`
	ParallelMaxWorkers      int                                     `json:"parallel_max_workers,omitempty"`
	Backpressure            types.BackpressureMode                  `json:"backpressure,omitempty"`
	FailurePolicy           FailurePolicy                           `json:"failure_policy,omitempty"`
	VoteTieBreak            VoteTieBreak                            `json:"vote_tie_break,omitempty"`
	TimeoutTerminalStatus   TaskStatus                              `json:"timeout_terminal_status,omitempty"`
	HostContextBindings     map[string]hostprojection.SourceBinding `json:"-"`
	continuationContext     map[string]hostprojection.Resolution
	continuationPredecessor string
	continuationGeneration  uint64
	continuationDeadline    time.Time
	continuationBudget      int64
}

type TaskRecord struct {
	TaskID          string     `json:"task_id"`
	AgentID         string     `json:"agent_id"`
	Role            Role       `json:"role"`
	Target          TaskTarget `json:"target,omitempty"`
	CollabPrimitive string     `json:"collab_primitive,omitempty"`
	PeerID          string     `json:"peer_id,omitempty"`
	Priority        int        `json:"priority,omitempty"`
	Status          TaskStatus `json:"status"`
	Reason          string     `json:"reason,omitempty"`
	Error           string     `json:"error,omitempty"`
	Vote            string     `json:"vote,omitempty"`
	Output          any        `json:"output,omitempty"`
}

type Result struct {
	RunID            string               `json:"run_id,omitempty"`
	TeamID           string               `json:"team_id"`
	WorkflowID       string               `json:"workflow_id,omitempty"`
	StepID           string               `json:"step_id,omitempty"`
	Strategy         Strategy             `json:"team_strategy"`
	WinnerVote       string               `json:"winner_vote,omitempty"`
	Tasks            []TaskRecord         `json:"tasks"`
	TeamTaskTotal    int                  `json:"team_task_total"`
	TeamTaskFailed   int                  `json:"team_task_failed"`
	TeamTaskCanceled int                  `json:"team_task_canceled"`
	TeamRemoteTotal  int                  `json:"team_remote_task_total,omitempty"`
	TeamRemoteFailed int                  `json:"team_remote_task_failed,omitempty"`
	TeamCheckpoint   *TeamContinuationRef `json:"team_checkpoint,omitempty"`
	PauseReason      TeamPauseReason      `json:"pause_reason,omitempty"`
	teamSuccessor    *TeamContinuation
}

func (r Result) RunFinishedPayload() map[string]any {
	payload := make(map[string]any, 9)
	payload["team_id"] = r.TeamID
	payload["workflow_id"] = r.WorkflowID
	payload["step_id"] = r.StepID
	payload["team_strategy"] = string(r.Strategy)
	payload["team_task_total"] = r.TeamTaskTotal
	payload["team_task_failed"] = r.TeamTaskFailed
	payload["team_task_canceled"] = r.TeamTaskCanceled
	payload["team_remote_task_total"] = r.TeamRemoteTotal
	payload["team_remote_task_failed"] = r.TeamRemoteFailed
	return payload
}

type teamTimelineContext struct {
	runID      string
	teamID     string
	workflowID string
	stepID     string
}

func newTeamTimelineContext(plan Plan) teamTimelineContext {
	return teamTimelineContext{
		runID:      strings.TrimSpace(plan.RunID),
		teamID:     strings.TrimSpace(plan.TeamID),
		workflowID: strings.TrimSpace(plan.WorkflowID),
		stepID:     strings.TrimSpace(plan.StepID),
	}
}

type StreamEvent struct {
	Kind   string      `json:"kind"`
	Task   *TaskRecord `json:"task,omitempty"`
	Result *Result     `json:"result,omitempty"`
}

type Option func(*Engine)

type TeamPauseBoundary struct {
	Plan           Plan
	Task           Task
	PendingTaskIDs []string
}

type TeamPauseError struct {
	Reason                 TeamPauseReason
	PendingActionReference string
	ConsumedBudget         int64
	OriginalDeadline       time.Time
}

func (e TeamPauseError) Error() string { return string(e.Reason) + ": team paused" }

type TeamPauseController interface {
	BeforeDispatch(context.Context, TeamPauseBoundary) error
}
type TeamPauseControllerFunc func(context.Context, TeamPauseBoundary) error

func (f TeamPauseControllerFunc) BeforeDispatch(ctx context.Context, b TeamPauseBoundary) error {
	if f == nil {
		return nil
	}
	return f(ctx, b)
}

func WithTeamPauseController(controller TeamPauseController) Option {
	return func(e *Engine) { e.pauseController = controller }
}

func WithTimelineEmitter(handler types.EventHandler) Option {
	return func(e *Engine) {
		e.timelineEmitter = handler
	}
}

// WithNativeEventHistory binds an existing team-owned journal. No journal is
// created by the default engine and append failures never alter team results.
func WithNativeEventHistory(journal *history.Journal) Option {
	return func(e *Engine) {
		if e != nil {
			e.nativeHistory = journal
		}
	}
}

// WithTeamContinuationStore opts a team owner into source-owned continuation.
// A nil store leaves the engine terminal-only.
func WithTeamContinuationStore(store TeamContinuationStore, namespace string) Option {
	return func(e *Engine) {
		if e != nil {
			e.continuations = store
			e.continuationNamespace = strings.TrimSpace(namespace)
		}
	}
}

type Engine struct {
	now                   func() time.Time
	timelineEmitter       types.EventHandler
	nativeHistory         *history.Journal
	nativeHistoryHealth   history.AppendHealth
	continuations         TeamContinuationStore
	continuationNamespace string
	pauseController       TeamPauseController
}

func New(opts ...Option) *Engine {
	e := &Engine{now: time.Now}
	for _, opt := range opts {
		if opt != nil {
			opt(e)
		}
	}
	return e
}

func (e *Engine) Run(ctx context.Context, plan Plan) (Result, error) {
	return e.execute(ctx, plan, nil)
}

func (e *Engine) Stream(ctx context.Context, plan Plan, onEvent func(StreamEvent) error) (Result, error) {
	return e.execute(ctx, plan, onEvent)
}

func (e *Engine) ResumeContinuation(ctx context.Context, plan Plan, request TeamResumeRequest, onEvent func(StreamEvent) error) (Result, error) {
	if e == nil || e.continuations == nil {
		return Result{}, ErrTeamContinuationUnavailable
	}
	continuation, err := e.continuations.Load(ctx, request.Ref)
	if err != nil {
		return Result{}, err
	}
	if request.GraphDigest != continuation.GraphDigest || request.Generation != continuation.Generation || request.Strategy != continuation.Strategy || request.Ref.TeamID != continuation.TeamID {
		return Result{}, ErrTeamContinuationConflict
	}
	if request.AuthorityDigest != continuation.AuthorityDigest || request.PlanDigest != continuation.PlanDigest || request.WriteScopeDigest != continuation.WriteScopeDigest || request.EffectNamespace != continuation.EffectNamespace || request.OccurrenceID != continuation.OccurrenceID || request.CausationID != continuation.CausationID {
		return Result{}, ErrTeamContinuationForeign
	}
	if !continuation.OriginalDeadline.IsZero() && !e.now().UTC().Before(continuation.OriginalDeadline) {
		return Result{}, ErrTeamContinuationRecoveryRequired
	}
	plan, err = normalizePlan(plan)
	if err != nil {
		return Result{}, err
	}
	if teamPlanDigest(plan) != continuation.GraphDigest {
		return Result{}, ErrTeamContinuationForeign
	}
	plan.continuationPredecessor = continuation.Ref.CheckpointID
	plan.continuationGeneration = continuation.Generation
	plan.continuationDeadline = continuation.OriginalDeadline
	plan.continuationBudget = continuation.ConsumedBudget
	resolutions, err := validateAndResolveTeamHostContext(ctx, plan, continuation, request)
	if err != nil {
		return Result{}, err
	}
	// Host authorization is outside the owner store. Do not consume a resume
	// claim when resolving material crosses the original source deadline.
	if !continuation.OriginalDeadline.IsZero() && !e.now().UTC().Before(continuation.OriginalDeadline) {
		return Result{}, ErrTeamContinuationRecoveryRequired
	}
	claim, err := e.continuations.ClaimResume(ctx, request)
	if err != nil {
		return Result{}, err
	}
	if claim.Duplicate && claim.Result != nil {
		return *claim.Result, nil
	}
	claim, err = e.continuations.MarkResumeRunning(ctx, claim)
	if err != nil {
		return Result{}, err
	}
	plan.continuationContext = resolutions
	records := make([]TaskRecord, len(plan.Tasks))
	for i, task := range plan.Tasks {
		records[i] = TaskRecord{TaskID: task.TaskID, AgentID: task.AgentID, Role: task.Role, Target: task.Target, CollabPrimitive: task.CollabPrimitive, PeerID: task.Remote.PeerID, Priority: task.Priority, Status: TaskStatusPending}
	}
	for _, saved := range continuation.Tasks {
		for i := range records {
			if records[i].TaskID == saved.TaskID {
				records[i].Status = saved.Status
				records[i].Vote = saved.Vote
				records[i].Reason = "resume.settled"
			}
		}
	}
	result, runErr := e.executeWithRecords(ctx, plan, records, onEvent)
	if runErr != nil && !errors.Is(runErr, context.Canceled) && !errors.Is(runErr, context.DeadlineExceeded) && !errors.Is(runErr, ErrTeamPaused) {
		return result, runErr
	}
	var committed TeamResumeClaim
	var commitErr error
	if successorStore, ok := e.continuations.(teamSuccessorContinuationStore); ok {
		committed, commitErr = successorStore.CommitResumeResultWithSuccessor(context.WithoutCancel(ctx), claim, result, result.teamSuccessor)
	} else if result.teamSuccessor != nil {
		commitErr = ErrTeamContinuationUnsupported
	} else {
		committed, commitErr = e.continuations.CommitResumeResult(context.WithoutCancel(ctx), claim, result)
	}
	if commitErr != nil {
		return Result{}, commitErr
	}
	if committed.Result != nil {
		return *committed.Result, runErr
	}
	return result, runErr
}

// validateAndResolveTeamHostContext verifies every requested member before
// invoking the host resolver. Resolved material is transient and reaches only
// the matching pending member through the existing TaskRunner seam.
func validateAndResolveTeamHostContext(ctx context.Context, plan Plan, continuation TeamContinuation, request TeamResumeRequest) (map[string]hostprojection.Resolution, error) {
	if len(request.HostContextProjections) == 0 {
		return nil, nil
	}
	if request.HostContextResolver == nil {
		return nil, ErrTeamContinuationUnsupported
	}
	pending := make(map[string]struct{}, len(continuation.PendingTaskIDs))
	for _, id := range continuation.PendingTaskIDs {
		pending[id] = struct{}{}
	}
	tasks := make(map[string]Task, len(plan.Tasks))
	for _, task := range plan.Tasks {
		tasks[task.TaskID] = task
	}
	ids := make([]string, 0, len(request.HostContextProjections))
	for id, descriptor := range request.HostContextProjections {
		binding, ok := continuation.HostContextBindings[id]
		if !ok || descriptor == nil {
			return nil, ErrTeamContinuationConflict
		}
		if _, ok := pending[id]; !ok {
			return nil, ErrTeamContinuationConflict
		}
		task, ok := tasks[id]
		if !ok || task.Target != TaskTargetLocal {
			return nil, ErrTeamContinuationUnsupported
		}
		if _, ok := task.Runner.(HostContextTaskRunner); !ok {
			return nil, ErrTeamContinuationUnsupported
		}
		snapshot, admitErr := hostprojection.Admit(*descriptor)
		if admitErr != nil {
			return nil, admitErr
		}
		if err := hostprojection.RequireBinding(snapshot, binding); err != nil {
			return nil, ErrTeamContinuationConflict
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	resolved := make(map[string]hostprojection.Resolution, len(ids))
	for _, id := range ids {
		binding := continuation.HostContextBindings[id]
		_, resolution, err := hostprojection.ResolveBound(ctx, *request.HostContextProjections[id], binding, request.HostContextResolver)
		if err != nil {
			return nil, err
		}
		resolved[id] = resolution
	}
	return resolved, nil
}

func (e *Engine) execute(ctx context.Context, raw Plan, onEvent func(StreamEvent) error) (Result, error) {
	plan, err := normalizePlan(raw)
	if err != nil {
		return Result{}, err
	}
	records := make([]TaskRecord, len(plan.Tasks))
	for i, task := range plan.Tasks {
		records[i] = TaskRecord{
			TaskID:          task.TaskID,
			AgentID:         task.AgentID,
			Role:            task.Role,
			Target:          task.Target,
			CollabPrimitive: task.CollabPrimitive,
			PeerID:          task.Remote.PeerID,
			Priority:        task.Priority,
			Status:          TaskStatusPending,
		}
	}
	return e.executeWithRecords(ctx, plan, records, onEvent)
}

func (e *Engine) executeWithRecords(ctx context.Context, plan Plan, records []TaskRecord, onEvent func(StreamEvent) error) (Result, error) {
	timelineCtx := newTeamTimelineContext(plan)
	var seq int64
	switch plan.Strategy {
	case StrategySerial:
		if err := e.runSerial(ctx, plan, timelineCtx, &seq, records, onEvent); err != nil {
			var pause TeamPauseError
			if errors.As(err, &pause) {
				paused, saveErr := e.persistPausedResult(ctx, plan, records, pause)
				if saveErr != nil {
					return Result{}, saveErr
				}
				return paused, ErrTeamPaused
			}
			return Result{}, err
		}
	case StrategyParallel:
		if err := e.runParallel(ctx, plan, timelineCtx, &seq, records, onEvent); err != nil {
			var pause TeamPauseError
			if errors.As(err, &pause) {
				paused, saveErr := e.persistPausedResult(ctx, plan, records, pause)
				if saveErr != nil {
					return Result{}, saveErr
				}
				return paused, ErrTeamPaused
			}
			return Result{}, err
		}
	case StrategyVote:
		if err := e.runParallel(ctx, plan, timelineCtx, &seq, records, onEvent); err != nil {
			var pause TeamPauseError
			if errors.As(err, &pause) {
				paused, saveErr := e.persistPausedResult(ctx, plan, records, pause)
				if saveErr != nil {
					return Result{}, saveErr
				}
				return paused, ErrTeamPaused
			}
			return Result{}, err
		}
	default:
		return Result{}, fmt.Errorf("unsupported team strategy %q", plan.Strategy)
	}

	result := buildResult(plan, records)
	if plan.Strategy == StrategyVote {
		result.WinnerVote = resolveVoteWinner(records, plan.VoteTieBreak)
	}
	e.emitTimeline(ctx, timelineCtx, &seq, resolveStatus(result), ReasonResolve, nil)
	if onEvent != nil {
		if err := onEvent(StreamEvent{Kind: "team.resolved", Result: &result}); err != nil {
			return Result{}, err
		}
	}
	if errors.Is(ctx.Err(), context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return result, ctx.Err()
	}
	return result, nil
}

func (e *Engine) runSerial(
	ctx context.Context,
	plan Plan,
	timelineCtx teamTimelineContext,
	seq *int64,
	records []TaskRecord,
	onEvent func(StreamEvent) error,
) error {
	halt := false
	for i := range plan.Tasks {
		if records[i].Status != TaskStatusPending {
			continue
		}
		if halt {
			records[i].Status = TaskStatusSkipped
			records[i].Reason = "policy.fail_fast"
			if strings.EqualFold(strings.TrimSpace(records[i].CollabPrimitive), string(collab.PrimitiveHandoff)) {
				records[i].Reason = ReasonHandoff
			}
			e.emitTimeline(ctx, timelineCtx, seq, records[i].Status, collectReasonForRecord(records[i]), &records[i])
			if onEvent != nil {
				snap := records[i]
				if err := onEvent(StreamEvent{Kind: "task.updated", Task: &snap}); err != nil {
					return err
				}
			}
			continue
		}

		if e.pauseController != nil {
			if err := e.pauseController.BeforeDispatch(ctx, TeamPauseBoundary{Plan: plan, Task: plan.Tasks[i], PendingTaskIDs: pendingTaskIDs(plan, records)}); err != nil {
				return err
			}
		}
		records[i].Status = TaskStatusRunning
		e.emitTimeline(ctx, timelineCtx, seq, records[i].Status, dispatchReasonForRecord(records[i]), &records[i])
		if onEvent != nil {
			snap := records[i]
			if err := onEvent(StreamEvent{Kind: "task.updated", Task: &snap}); err != nil {
				return err
			}
		}

		result := executeTask(ctx, plan, plan.Tasks[i])
		records[i].Status = result.Status
		records[i].Reason = result.Reason
		records[i].Error = result.Error
		records[i].Vote = result.Vote
		records[i].Output = result.Output
		e.emitTimeline(ctx, timelineCtx, seq, records[i].Status, collectReasonForRecord(records[i]), &records[i])
		if onEvent != nil {
			snap := records[i]
			if err := onEvent(StreamEvent{Kind: "task.updated", Task: &snap}); err != nil {
				return err
			}
		}

		if plan.FailurePolicy == FailurePolicyFailFast && (records[i].Status == TaskStatusFailed || records[i].Status == TaskStatusCanceled) {
			halt = true
		}
	}
	return nil
}

func (e *Engine) runParallel(
	ctx context.Context,
	plan Plan,
	timelineCtx teamTimelineContext,
	seq *int64,
	records []TaskRecord,
	onEvent func(StreamEvent) error,
) error {
	pending := make([]int, 0, len(plan.Tasks))
	for i := range records {
		if records[i].Status == TaskStatusPending {
			pending = append(pending, i)
		}
	}
	if len(pending) == 0 {
		return nil
	}
	workPlan := plan
	indexMap := pending
	if len(pending) != len(plan.Tasks) {
		workPlan.Tasks = make([]Task, len(pending))
		for i, original := range pending {
			workPlan.Tasks[i] = plan.Tasks[original]
		}
		indexMap = pending
	}
	runLocal, skippedLocal, skipReason := selectParallelWork(workPlan)
	runIdx, skippedIdx := make([]int, len(runLocal)), make([]int, len(skippedLocal))
	for i, idx := range runLocal {
		runIdx[i] = indexMap[idx]
	}
	for i, idx := range skippedLocal {
		skippedIdx[i] = indexMap[idx]
	}
	for _, idx := range skippedIdx {
		records[idx].Status = TaskStatusSkipped
		records[idx].Reason = skipReason
		e.emitTimeline(ctx, timelineCtx, seq, records[idx].Status, collectReasonForRecord(records[idx]), &records[idx])
		if onEvent != nil {
			snap := records[idx]
			if err := onEvent(StreamEvent{Kind: "task.updated", Task: &snap}); err != nil {
				return err
			}
		}
	}
	if len(runIdx) == 0 {
		return nil
	}

	// A parallel batch has a single source-owned safe boundary: validate every
	// unstarted member before any record is marked running. Otherwise a pause
	// on a later member would persist an earlier, never-executed member as
	// in-flight and fabricate recovery material.
	if e.pauseController != nil {
		pendingIDs := pendingTaskIDs(plan, records)
		for _, idx := range runIdx {
			if err := e.pauseController.BeforeDispatch(ctx, TeamPauseBoundary{Plan: plan, Task: plan.Tasks[idx], PendingTaskIDs: pendingIDs}); err != nil {
				return err
			}
		}
	}
	for _, idx := range runIdx {
		records[idx].Status = TaskStatusRunning
		e.emitTimeline(ctx, timelineCtx, seq, records[idx].Status, dispatchReasonForRecord(records[idx]), &records[idx])
		if onEvent != nil {
			snap := records[idx]
			if err := onEvent(StreamEvent{Kind: "task.updated", Task: &snap}); err != nil {
				return err
			}
		}
	}

	execCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	type item struct {
		index  int
		result taskExecution
	}
	jobs := make(chan int)
	results := make(chan item, len(runIdx))

	var wg sync.WaitGroup
	workers := plan.ParallelMaxWorkers
	if workers <= 0 {
		workers = 1
	}
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				out := executeTask(execCtx, plan, plan.Tasks[idx])
				results <- item{index: idx, result: out}
			}
		}()
	}
	go func() {
		for _, idx := range runIdx {
			jobs <- idx
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	for it := range results {
		records[it.index].Status = it.result.Status
		records[it.index].Reason = it.result.Reason
		records[it.index].Error = it.result.Error
		records[it.index].Vote = it.result.Vote
		records[it.index].Output = it.result.Output
		e.emitTimeline(execCtx, timelineCtx, seq, records[it.index].Status, collectReasonForRecord(records[it.index]), &records[it.index])
		if onEvent != nil {
			snap := records[it.index]
			if err := onEvent(StreamEvent{Kind: "task.updated", Task: &snap}); err != nil {
				cancel()
				return err
			}
		}
		if plan.FailurePolicy == FailurePolicyFailFast &&
			(records[it.index].Status == TaskStatusFailed || records[it.index].Status == TaskStatusCanceled) {
			cancel()
		}
	}
	return nil
}

func buildResult(plan Plan, records []TaskRecord) Result {
	out := Result{
		RunID:         strings.TrimSpace(plan.RunID),
		TeamID:        plan.TeamID,
		WorkflowID:    strings.TrimSpace(plan.WorkflowID),
		StepID:        strings.TrimSpace(plan.StepID),
		Strategy:      plan.Strategy,
		Tasks:         append([]TaskRecord(nil), records...),
		TeamTaskTotal: len(records),
	}
	for _, record := range records {
		if record.Target == TaskTargetRemote {
			out.TeamRemoteTotal++
		}
		switch record.Status {
		case TaskStatusFailed:
			out.TeamTaskFailed++
			if record.Target == TaskTargetRemote {
				out.TeamRemoteFailed++
			}
		case TaskStatusCanceled:
			out.TeamTaskCanceled++
		}
	}
	return out
}

func resolveStatus(result Result) TaskStatus {
	if result.TeamTaskTotal == 0 {
		return TaskStatusSkipped
	}
	succeeded := 0
	skipped := 0
	for _, record := range result.Tasks {
		switch record.Status {
		case TaskStatusSucceeded:
			succeeded++
		case TaskStatusSkipped:
			skipped++
		}
	}
	if succeeded > 0 {
		return TaskStatusSucceeded
	}
	if result.TeamTaskCanceled == result.TeamTaskTotal {
		return TaskStatusCanceled
	}
	if skipped == result.TeamTaskTotal {
		return TaskStatusSkipped
	}
	return TaskStatusFailed
}

type taskExecution struct {
	Status TaskStatus
	Reason string
	Error  string
	Vote   string
	Output any
}

func executeTask(ctx context.Context, plan Plan, task Task) taskExecution {
	if errors.Is(ctx.Err(), context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return taskExecution{Status: TaskStatusCanceled, Reason: "cancel.propagated", Error: ctx.Err().Error()}
	}

	runCtx := ctx
	cancel := func() {}
	timedOut := false
	if plan.TaskTimeout > 0 {
		var timeoutCancel context.CancelFunc
		runCtx, timeoutCancel = context.WithTimeout(ctx, plan.TaskTimeout)
		cancel = timeoutCancel
	}
	defer cancel()

	var (
		result TaskResult
		err    error
	)
	if resolution, hasContext := plan.continuationContext[task.TaskID]; hasContext {
		runner, ok := task.Runner.(HostContextTaskRunner)
		if !ok {
			return taskExecution{Status: TaskStatusFailed, Reason: "host_context.unsupported_capability"}
		}
		result, err = runner.RunWithHostContext(runCtx, plan, task, resolution)
	} else {
		switch task.Target {
		case TaskTargetRemote:
			if task.RemoteRunner == nil {
				return taskExecution{Status: TaskStatusFailed, Reason: "task.remote_runner_missing"}
			}
			result, err = task.RemoteRunner.RunRemote(runCtx, plan, task)
		default:
			if task.Runner == nil {
				return taskExecution{Status: TaskStatusFailed, Reason: "task.runner_missing"}
			}
			result, err = task.Runner.Run(runCtx, task)
		}
	}
	if err == nil {
		return taskExecution{
			Status: TaskStatusSucceeded,
			Vote:   strings.TrimSpace(result.Vote),
			Output: result.Output,
		}
	}

	if errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil {
		timedOut = true
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		if errors.Is(ctx.Err(), context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return taskExecution{Status: TaskStatusCanceled, Reason: "cancel.propagated", Error: err.Error()}
		}
		if timedOut {
			status := plan.TimeoutTerminalStatus
			if status == "" {
				status = TaskStatusCanceled
			}
			return taskExecution{Status: status, Reason: "task.timeout", Error: err.Error()}
		}
	}
	return taskExecution{Status: TaskStatusFailed, Reason: "task.error", Error: err.Error()}
}

func selectParallelWork(plan Plan) (runIdx []int, skippedIdx []int, skippedReason string) {
	total := len(plan.Tasks)
	if total == 0 {
		return nil, nil, ""
	}
	maxWorkers := plan.ParallelMaxWorkers
	if maxWorkers <= 0 || maxWorkers >= total {
		runIdx = make([]int, 0, total)
		for i := 0; i < total; i++ {
			runIdx = append(runIdx, i)
		}
		return runIdx, nil, ""
	}

	switch plan.Backpressure {
	case types.BackpressureReject:
		runIdx = make([]int, 0, maxWorkers)
		for i := 0; i < maxWorkers; i++ {
			runIdx = append(runIdx, i)
		}
		skippedIdx = make([]int, 0, total-maxWorkers)
		for i := maxWorkers; i < total; i++ {
			skippedIdx = append(skippedIdx, i)
		}
		return runIdx, skippedIdx, "backpressure.reject"
	case types.BackpressureDropLowPriority:
		order := make([]int, 0, total)
		for i := range plan.Tasks {
			order = append(order, i)
		}
		sort.Slice(order, func(i, j int) bool {
			left := plan.Tasks[order[i]]
			right := plan.Tasks[order[j]]
			if left.Priority == right.Priority {
				if left.TaskID == right.TaskID {
					return order[i] < order[j]
				}
				return left.TaskID < right.TaskID
			}
			return left.Priority > right.Priority
		})
		keep := map[int]struct{}{}
		for i := 0; i < maxWorkers; i++ {
			keep[order[i]] = struct{}{}
		}
		for i := 0; i < total; i++ {
			if _, ok := keep[i]; ok {
				runIdx = append(runIdx, i)
			} else {
				skippedIdx = append(skippedIdx, i)
			}
		}
		return runIdx, skippedIdx, "backpressure.drop_low_priority"
	default:
		runIdx = make([]int, 0, total)
		for i := 0; i < total; i++ {
			runIdx = append(runIdx, i)
		}
		return runIdx, nil, ""
	}
}

type voteStats struct {
	Count       int
	MaxPriority int
	FirstTaskID string
}

func resolveVoteWinner(records []TaskRecord, tieBreak VoteTieBreak) string {
	statsByVote := map[string]voteStats{}
	for _, record := range records {
		if record.Status != TaskStatusSucceeded {
			continue
		}
		vote := strings.TrimSpace(record.Vote)
		if vote == "" {
			continue
		}
		stats := statsByVote[vote]
		stats.Count++
		if stats.Count == 1 || record.Priority > stats.MaxPriority {
			stats.MaxPriority = record.Priority
		}
		if stats.Count == 1 || strings.TrimSpace(record.TaskID) < stats.FirstTaskID {
			stats.FirstTaskID = strings.TrimSpace(record.TaskID)
		}
		statsByVote[vote] = stats
	}
	if len(statsByVote) == 0 {
		return ""
	}
	candidates := make([]string, 0, len(statsByVote))
	for vote := range statsByVote {
		candidates = append(candidates, vote)
	}
	sort.Strings(candidates)
	best := candidates[0]
	for _, candidate := range candidates[1:] {
		left := statsByVote[best]
		right := statsByVote[candidate]
		if right.Count > left.Count {
			best = candidate
			continue
		}
		if right.Count < left.Count {
			continue
		}
		switch tieBreak {
		case VoteTieBreakFirstTaskID:
			if right.FirstTaskID < left.FirstTaskID || (right.FirstTaskID == left.FirstTaskID && candidate < best) {
				best = candidate
			}
		default:
			if right.MaxPriority > left.MaxPriority {
				best = candidate
				continue
			}
			if right.MaxPriority == left.MaxPriority {
				if right.FirstTaskID < left.FirstTaskID || (right.FirstTaskID == left.FirstTaskID && candidate < best) {
					best = candidate
				}
			}
		}
	}
	return best
}

// NormalizePlan exposes the native pure validation/defaults seam. It detaches
// the task slice before normalization; opaque runners remain host-owned.
func NormalizePlan(plan Plan) (Plan, error) {
	plan.Tasks = append([]Task(nil), plan.Tasks...)
	return normalizePlan(plan)
}

func normalizePlan(plan Plan) (Plan, error) {
	plan.TeamID = strings.TrimSpace(plan.TeamID)
	plan.WorkflowID = strings.TrimSpace(plan.WorkflowID)
	plan.StepID = strings.TrimSpace(plan.StepID)
	if plan.TeamID == "" {
		return Plan{}, errors.New("team_id is required")
	}
	plan.Strategy = Strategy(strings.ToLower(strings.TrimSpace(string(plan.Strategy))))
	if plan.Strategy == "" {
		plan.Strategy = StrategySerial
	}
	switch plan.Strategy {
	case StrategySerial, StrategyParallel, StrategyVote:
	default:
		return Plan{}, fmt.Errorf("unsupported team strategy %q", plan.Strategy)
	}
	if plan.TaskTimeout <= 0 {
		plan.TaskTimeout = 3 * time.Second
	}
	if plan.ParallelMaxWorkers <= 0 {
		plan.ParallelMaxWorkers = 4
	}
	if plan.Backpressure == "" {
		plan.Backpressure = types.BackpressureBlock
	}
	switch plan.Backpressure {
	case types.BackpressureBlock, types.BackpressureReject, types.BackpressureDropLowPriority:
	default:
		return Plan{}, fmt.Errorf("unsupported backpressure mode %q", plan.Backpressure)
	}
	plan.FailurePolicy = FailurePolicy(strings.ToLower(strings.TrimSpace(string(plan.FailurePolicy))))
	if plan.FailurePolicy == "" {
		plan.FailurePolicy = FailurePolicyBestEffort
	}
	switch plan.FailurePolicy {
	case FailurePolicyBestEffort, FailurePolicyFailFast:
	default:
		return Plan{}, fmt.Errorf("unsupported failure policy %q", plan.FailurePolicy)
	}
	if plan.TimeoutTerminalStatus == "" {
		plan.TimeoutTerminalStatus = TaskStatusCanceled
	}
	switch plan.TimeoutTerminalStatus {
	case TaskStatusCanceled, TaskStatusFailed:
	default:
		return Plan{}, fmt.Errorf("timeout terminal status must be canceled|failed, got %q", plan.TimeoutTerminalStatus)
	}
	plan.VoteTieBreak = VoteTieBreak(strings.ToLower(strings.TrimSpace(string(plan.VoteTieBreak))))
	if plan.VoteTieBreak == "" {
		plan.VoteTieBreak = VoteTieBreakHighestPriority
	}
	switch plan.VoteTieBreak {
	case VoteTieBreakHighestPriority, VoteTieBreakFirstTaskID:
	default:
		return Plan{}, fmt.Errorf("unsupported vote tie_break %q", plan.VoteTieBreak)
	}

	if len(plan.Tasks) == 0 {
		return Plan{}, errors.New("team tasks must not be empty")
	}
	seen := map[string]struct{}{}
	for i := range plan.Tasks {
		plan.Tasks[i].TaskID = strings.TrimSpace(plan.Tasks[i].TaskID)
		plan.Tasks[i].AgentID = strings.TrimSpace(plan.Tasks[i].AgentID)
		plan.Tasks[i].Role = Role(strings.ToLower(strings.TrimSpace(string(plan.Tasks[i].Role))))
		plan.Tasks[i].CollabPrimitive = strings.ToLower(strings.TrimSpace(plan.Tasks[i].CollabPrimitive))
		plan.Tasks[i].Target = TaskTarget(strings.ToLower(strings.TrimSpace(string(plan.Tasks[i].Target))))
		if plan.Tasks[i].Target == "" {
			plan.Tasks[i].Target = TaskTargetLocal
		}
		if plan.Tasks[i].Role == "" {
			plan.Tasks[i].Role = RoleWorker
		}
		switch plan.Tasks[i].Role {
		case RoleLeader, RoleWorker, RoleCoordinator:
		default:
			return Plan{}, fmt.Errorf("tasks[%d].role must be leader|worker|coordinator", i)
		}
		if plan.Tasks[i].TaskID == "" {
			return Plan{}, fmt.Errorf("tasks[%d].task_id is required", i)
		}
		if _, ok := seen[plan.Tasks[i].TaskID]; ok {
			return Plan{}, fmt.Errorf("duplicate task_id %q", plan.Tasks[i].TaskID)
		}
		seen[plan.Tasks[i].TaskID] = struct{}{}
		if plan.Tasks[i].AgentID == "" {
			return Plan{}, fmt.Errorf("tasks[%d].agent_id is required", i)
		}
		if plan.Tasks[i].CollabPrimitive != "" {
			if _, err := collab.ParsePrimitive(collab.Primitive(plan.Tasks[i].CollabPrimitive)); err != nil {
				return Plan{}, fmt.Errorf("tasks[%d].collab_primitive is invalid: %w", i, err)
			}
		}
		switch plan.Tasks[i].Target {
		case TaskTargetLocal:
			if plan.Tasks[i].Runner == nil {
				return Plan{}, fmt.Errorf("tasks[%d].runner is required when target=local", i)
			}
		case TaskTargetRemote:
			plan.Tasks[i].Remote.PeerID = strings.TrimSpace(plan.Tasks[i].Remote.PeerID)
			plan.Tasks[i].Remote.Method = strings.TrimSpace(plan.Tasks[i].Remote.Method)
			if plan.Tasks[i].Remote.PeerID == "" {
				return Plan{}, fmt.Errorf("tasks[%d].remote.peer_id is required when target=remote", i)
			}
			if plan.Tasks[i].RemoteRunner == nil {
				return Plan{}, fmt.Errorf("tasks[%d].remote_runner is required when target=remote", i)
			}
		default:
			return Plan{}, fmt.Errorf("tasks[%d].target must be local|remote", i)
		}
	}
	return plan, nil
}

func (e *Engine) emitTimeline(
	ctx context.Context,
	timelineCtx teamTimelineContext,
	seq *int64,
	status TaskStatus,
	reason string,
	task *TaskRecord,
) {
	if e == nil || seq == nil || (e.timelineEmitter == nil && e.nativeHistory == nil) {
		return
	}
	*seq++
	payload := make(map[string]any, 10)
	payload["phase"] = string(types.ActionPhaseRun)
	payload["status"] = string(status)
	payload["sequence"] = *seq
	payload["reason"] = reason
	payload["team_id"] = timelineCtx.teamID
	if timelineCtx.workflowID != "" {
		payload["workflow_id"] = timelineCtx.workflowID
	}
	if timelineCtx.stepID != "" {
		payload["step_id"] = timelineCtx.stepID
	}
	if task != nil {
		payload["agent_id"] = task.AgentID
		payload["task_id"] = task.TaskID
		if task.PeerID != "" {
			payload["peer_id"] = task.PeerID
		}
	}
	e.emitNativeHistory(ctx, timelineCtx, status, payload)
	if e.timelineEmitter == nil {
		return
	}
	e.timelineEmitter.OnEvent(ctx, types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    types.EventTypeActionTimeline,
		RunID:   timelineCtx.runID,
		Time:    e.now(),
		Payload: payload,
	})
}

func (e *Engine) emitNativeHistory(ctx context.Context, timeline teamTimelineContext, status TaskStatus, payload map[string]any) {
	if e == nil || e.nativeHistory == nil {
		return
	}
	source := e.nativeHistory.Source()
	if source.Kind != types.HistorySourceTeam || strings.TrimSpace(source.ID) != strings.TrimSpace(timeline.teamID) {
		return
	}
	nativeType := "team." + string(status)
	if status == TaskStatusSucceeded || status == TaskStatusFailed || status == TaskStatusCanceled {
		nativeType = "run.finished"
		payload = cloneTimelinePayload(payload)
		payload["terminal_state"] = historyTerminalState(string(status))
	}
	_, err := e.nativeHistory.Append(ctx, types.NativeExecutionEvent{
		Version:    types.NativeExecutionEventVersionV1,
		NativeType: nativeType,
		Event:      types.EventEnvelope{RunID: timeline.runID, Source: types.ProtocolSourceTeams, Time: e.now(), Payload: payload},
	})
	e.nativeHistoryHealth.Record(source, err)
}

// NativeEventHistoryError reports append failures from the explicitly bound
// team source journal for the lifetime of this Engine.
func (e *Engine) NativeEventHistoryError() error {
	if e == nil {
		return nil
	}
	return e.nativeHistoryHealth.Err()
}

func historyTerminalState(status string) string {
	if status == string(TaskStatusSucceeded) {
		return "completed"
	}
	return status
}

func cloneTimelinePayload(payload map[string]any) map[string]any {
	result := make(map[string]any, len(payload))
	for key, value := range payload {
		result[key] = value
	}
	return result
}

func dispatchReasonForRecord(record TaskRecord) string {
	switch collab.Primitive(strings.ToLower(strings.TrimSpace(record.CollabPrimitive))) {
	case collab.PrimitiveHandoff:
		return ReasonHandoff
	case collab.PrimitiveDelegation:
		return ReasonDelegation
	}
	if record.Target == TaskTargetRemote {
		return ReasonDispatchRemote
	}
	return ReasonDispatch
}

func collectReasonForRecord(record TaskRecord) string {
	switch collab.Primitive(strings.ToLower(strings.TrimSpace(record.CollabPrimitive))) {
	case collab.PrimitiveHandoff:
		return ReasonHandoff
	case collab.PrimitiveAggregation:
		return ReasonAggregation
	}
	if record.Target == TaskTargetRemote {
		return ReasonCollectRemote
	}
	return ReasonCollect
}
