package teams

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

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
	RunID                 string                 `json:"run_id,omitempty"`
	TeamID                string                 `json:"team_id"`
	WorkflowID            string                 `json:"workflow_id,omitempty"`
	StepID                string                 `json:"step_id,omitempty"`
	Strategy              Strategy               `json:"strategy,omitempty"`
	Tasks                 []Task                 `json:"tasks"`
	TaskTimeout           time.Duration          `json:"task_timeout,omitempty"`
	ParallelMaxWorkers    int                    `json:"parallel_max_workers,omitempty"`
	Backpressure          types.BackpressureMode `json:"backpressure,omitempty"`
	FailurePolicy         FailurePolicy          `json:"failure_policy,omitempty"`
	VoteTieBreak          VoteTieBreak           `json:"vote_tie_break,omitempty"`
	TimeoutTerminalStatus TaskStatus             `json:"timeout_terminal_status,omitempty"`
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
	RunID            string       `json:"run_id,omitempty"`
	TeamID           string       `json:"team_id"`
	WorkflowID       string       `json:"workflow_id,omitempty"`
	StepID           string       `json:"step_id,omitempty"`
	Strategy         Strategy     `json:"team_strategy"`
	WinnerVote       string       `json:"winner_vote,omitempty"`
	Tasks            []TaskRecord `json:"tasks"`
	TeamTaskTotal    int          `json:"team_task_total"`
	TeamTaskFailed   int          `json:"team_task_failed"`
	TeamTaskCanceled int          `json:"team_task_canceled"`
	TeamRemoteTotal  int          `json:"team_remote_task_total,omitempty"`
	TeamRemoteFailed int          `json:"team_remote_task_failed,omitempty"`
}

func (r Result) RunFinishedPayload() map[string]any {
	return map[string]any{
		"team_id":                 r.TeamID,
		"workflow_id":             r.WorkflowID,
		"step_id":                 r.StepID,
		"team_strategy":           string(r.Strategy),
		"team_task_total":         r.TeamTaskTotal,
		"team_task_failed":        r.TeamTaskFailed,
		"team_task_canceled":      r.TeamTaskCanceled,
		"team_remote_task_total":  r.TeamRemoteTotal,
		"team_remote_task_failed": r.TeamRemoteFailed,
	}
}

type StreamEvent struct {
	Kind   string      `json:"kind"`
	Task   *TaskRecord `json:"task,omitempty"`
	Result *Result     `json:"result,omitempty"`
}

type Option func(*Engine)

func WithTimelineEmitter(handler types.EventHandler) Option {
	return func(e *Engine) {
		e.timelineEmitter = handler
	}
}

type Engine struct {
	now             func() time.Time
	timelineEmitter types.EventHandler
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

	var seq int64
	switch plan.Strategy {
	case StrategySerial:
		if err := e.runSerial(ctx, plan, &seq, records, onEvent); err != nil {
			return Result{}, err
		}
	case StrategyParallel:
		if err := e.runParallel(ctx, plan, &seq, records, onEvent); err != nil {
			return Result{}, err
		}
	case StrategyVote:
		if err := e.runParallel(ctx, plan, &seq, records, onEvent); err != nil {
			return Result{}, err
		}
	default:
		return Result{}, fmt.Errorf("unsupported team strategy %q", plan.Strategy)
	}

	result := buildResult(plan, records)
	if plan.Strategy == StrategyVote {
		result.WinnerVote = resolveVoteWinner(records, plan.VoteTieBreak)
	}
	e.emitTimeline(ctx, plan, &seq, resolveStatus(result), ReasonResolve, nil)
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

func (e *Engine) runSerial(ctx context.Context, plan Plan, seq *int64, records []TaskRecord, onEvent func(StreamEvent) error) error {
	halt := false
	for i := range plan.Tasks {
		if halt {
			records[i].Status = TaskStatusSkipped
			records[i].Reason = "policy.fail_fast"
			if strings.EqualFold(strings.TrimSpace(records[i].CollabPrimitive), string(collab.PrimitiveHandoff)) {
				records[i].Reason = ReasonHandoff
			}
			e.emitTimeline(ctx, plan, seq, records[i].Status, collectReasonForRecord(records[i]), &records[i])
			if onEvent != nil {
				snap := records[i]
				if err := onEvent(StreamEvent{Kind: "task.updated", Task: &snap}); err != nil {
					return err
				}
			}
			continue
		}

		records[i].Status = TaskStatusRunning
		e.emitTimeline(ctx, plan, seq, records[i].Status, dispatchReasonForRecord(records[i]), &records[i])
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
		e.emitTimeline(ctx, plan, seq, records[i].Status, collectReasonForRecord(records[i]), &records[i])
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

func (e *Engine) runParallel(ctx context.Context, plan Plan, seq *int64, records []TaskRecord, onEvent func(StreamEvent) error) error {
	runIdx, skippedIdx, skipReason := selectParallelWork(plan)
	for _, idx := range skippedIdx {
		records[idx].Status = TaskStatusSkipped
		records[idx].Reason = skipReason
		e.emitTimeline(ctx, plan, seq, records[idx].Status, collectReasonForRecord(records[idx]), &records[idx])
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

	for _, idx := range runIdx {
		records[idx].Status = TaskStatusRunning
		e.emitTimeline(ctx, plan, seq, records[idx].Status, dispatchReasonForRecord(records[idx]), &records[idx])
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
		e.emitTimeline(execCtx, plan, seq, records[it.index].Status, collectReasonForRecord(records[it.index]), &records[it.index])
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

func (e *Engine) emitTimeline(ctx context.Context, plan Plan, seq *int64, status TaskStatus, reason string, task *TaskRecord) {
	if e == nil || e.timelineEmitter == nil || seq == nil {
		return
	}
	*seq++
	payload := map[string]any{
		"phase":    string(types.ActionPhaseRun),
		"status":   string(status),
		"sequence": *seq,
		"reason":   reason,
		"team_id":  plan.TeamID,
	}
	if plan.WorkflowID != "" {
		payload["workflow_id"] = plan.WorkflowID
	}
	if plan.StepID != "" {
		payload["step_id"] = plan.StepID
	}
	if task != nil {
		payload["agent_id"] = task.AgentID
		payload["task_id"] = task.TaskID
		if task.PeerID != "" {
			payload["peer_id"] = task.PeerID
		}
	}
	e.timelineEmitter.OnEvent(ctx, types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    types.EventTypeActionTimeline,
		RunID:   strings.TrimSpace(plan.RunID),
		Time:    e.now(),
		Payload: payload,
	})
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
