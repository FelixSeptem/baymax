package composer

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/orchestration/scheduler"
	"github.com/FelixSeptem/baymax/orchestration/teams"
	"github.com/FelixSeptem/baymax/orchestration/workflow"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
	"github.com/FelixSeptem/baymax/tool/local"
)

const (
	defaultChildWorkerID    = "composer-child-worker"
	defaultChildPollTimeout = 20 * time.Millisecond
)

type ChildTarget string

const (
	ChildTargetLocal ChildTarget = "local"
	ChildTargetA2A   ChildTarget = "a2a"
)

type LocalChildRunner interface {
	RunChild(ctx context.Context, task scheduler.Task) (map[string]any, error)
}

type LocalChildRunnerFunc func(ctx context.Context, task scheduler.Task) (map[string]any, error)

func (f LocalChildRunnerFunc) RunChild(ctx context.Context, task scheduler.Task) (map[string]any, error) {
	return f(ctx, task)
}

type ChildDispatchRequest struct {
	Task                 scheduler.Task
	Target               ChildTarget
	ParentDepth          int
	ParentActiveChildren int
	ChildTimeout         time.Duration
	WorkerID             string
	PollInterval         time.Duration
	LocalRunner          LocalChildRunner
}

type ChildDispatchResult struct {
	Record     scheduler.TaskRecord
	Claimed    scheduler.ClaimedTask
	Commit     scheduler.TerminalCommit
	CommitMeta scheduler.CommitResult
	Retryable  bool
}

type Runner interface {
	Run(ctx context.Context, req types.RunRequest, h types.EventHandler) (types.RunResult, error)
	Stream(ctx context.Context, req types.RunRequest, h types.EventHandler) (types.RunResult, error)
}

type TeamsEngine interface {
	Run(ctx context.Context, plan teams.Plan) (teams.Result, error)
	Stream(ctx context.Context, plan teams.Plan, onEvent func(teams.StreamEvent) error) (teams.Result, error)
}

type WorkflowEngine interface {
	Run(ctx context.Context, req workflow.RunRequest) (workflow.RunResult, error)
	Stream(ctx context.Context, req workflow.RunRequest, onEvent func(workflow.StreamEvent) error) (workflow.RunResult, error)
}

type Option func(*Composer)

func WithRuntimeManager(mgr *runtimeconfig.Manager) Option {
	return func(c *Composer) {
		c.runtimeMgr = mgr
	}
}

func WithEventHandler(handler types.EventHandler) Option {
	return func(c *Composer) {
		c.handler = handler
	}
}

func WithLocalRegistry(registry *local.Registry) Option {
	return func(c *Composer) {
		c.localRegistry = registry
	}
}

func WithRunner(engine Runner) Option {
	return func(c *Composer) {
		c.runner = engine
	}
}

func WithTeams(engine TeamsEngine) Option {
	return func(c *Composer) {
		c.teams = engine
	}
}

func WithWorkflow(engine WorkflowEngine) Option {
	return func(c *Composer) {
		c.workflow = engine
	}
}

func WithScheduler(s *scheduler.Scheduler) Option {
	return func(c *Composer) {
		c.scheduler = s
		c.managedScheduler = false
	}
}

func WithSchedulerStore(store scheduler.QueueStore) Option {
	return func(c *Composer) {
		c.schedulerStore = store
	}
}

func WithRecoveryStore(store RecoveryStore) Option {
	return func(c *Composer) {
		c.recoveryStore = store
		c.managedRecoveryStore = false
	}
}

func WithA2AClient(client scheduler.A2AClient) Option {
	return func(c *Composer) {
		c.a2aClient = client
	}
}

func WithChildWorkerID(workerID string) Option {
	return func(c *Composer) {
		c.childWorkerID = strings.TrimSpace(workerID)
	}
}

func WithChildPollInterval(interval time.Duration) Option {
	return func(c *Composer) {
		c.childPollInterval = interval
	}
}

type Builder struct {
	model types.ModelClient
	opts  []Option
}

func NewBuilder(model types.ModelClient) *Builder {
	return &Builder{model: model}
}

func (b *Builder) WithRuntimeManager(mgr *runtimeconfig.Manager) *Builder {
	b.opts = append(b.opts, WithRuntimeManager(mgr))
	return b
}

func (b *Builder) WithEventHandler(handler types.EventHandler) *Builder {
	b.opts = append(b.opts, WithEventHandler(handler))
	return b
}

func (b *Builder) WithLocalRegistry(registry *local.Registry) *Builder {
	b.opts = append(b.opts, WithLocalRegistry(registry))
	return b
}

func (b *Builder) WithA2AClient(client scheduler.A2AClient) *Builder {
	b.opts = append(b.opts, WithA2AClient(client))
	return b
}

func (b *Builder) WithRecoveryStore(store RecoveryStore) *Builder {
	b.opts = append(b.opts, WithRecoveryStore(store))
	return b
}

func (b *Builder) WithSchedulerStore(store scheduler.QueueStore) *Builder {
	b.opts = append(b.opts, WithSchedulerStore(store))
	return b
}

func (b *Builder) WithChildWorkerID(workerID string) *Builder {
	b.opts = append(b.opts, WithChildWorkerID(workerID))
	return b
}

func (b *Builder) WithChildPollInterval(interval time.Duration) *Builder {
	b.opts = append(b.opts, WithChildPollInterval(interval))
	return b
}

func (b *Builder) Build() (*Composer, error) {
	return New(b.model, b.opts...)
}

type Composer struct {
	runtimeMgr     *runtimeconfig.Manager
	handler        types.EventHandler
	localRegistry  *local.Registry
	runner         Runner
	teams          TeamsEngine
	workflow       WorkflowEngine
	scheduler      *scheduler.Scheduler
	schedulerStore scheduler.QueueStore
	recoveryStore  RecoveryStore
	a2aClient      scheduler.A2AClient

	schedulerMu                sync.RWMutex
	managedScheduler           bool
	schedulerSignature         string
	schedulerConfiguredBackend string
	schedulerBackend           string
	schedulerFallback          bool
	schedulerFallbackReason    string
	schedulerQueueLimit        int
	schedulerRetryMaxAttempts  int
	schedulerGuardrails        scheduler.Guardrails

	managedRecoveryStore      bool
	recoverySignature         string
	recoveryConfiguredBackend string
	recoveryBackend           string
	recoveryPath              string
	recoveryEnabled           bool
	recoveryFallback          bool
	recoveryFallbackReason    string
	recoveryConflictPolicy    string

	now               func() time.Time
	childWorkerID     string
	childPollInterval time.Duration

	runMu   sync.Mutex
	runStat map[string]*runStat
}

type runStat struct {
	ChildTotal             int
	ChildFailed            int
	BudgetReject           int
	Backend                string
	BackendFallback        bool
	FallbackReason         string
	ComposerManaged        bool
	RecoveryEnabled        bool
	RecoveryRecovered      bool
	RecoveryReplayTotal    int
	RecoveryConflict       bool
	RecoveryConflictCode   string
	RecoveryFallback       bool
	RecoveryFallbackReason string
}

func New(model types.ModelClient, opts ...Option) (*Composer, error) {
	c := &Composer{
		now:                  time.Now,
		childWorkerID:        defaultChildWorkerID,
		childPollInterval:    defaultChildPollTimeout,
		managedScheduler:     true,
		managedRecoveryStore: true,
		runStat:              map[string]*runStat{},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}

	if strings.TrimSpace(c.childWorkerID) == "" {
		c.childWorkerID = defaultChildWorkerID
	}
	if c.childPollInterval <= 0 {
		c.childPollInterval = defaultChildPollTimeout
	}

	if c.runner == nil {
		if model == nil {
			return nil, fmt.Errorf("composer requires a model when runner is not injected")
		}
		runnerOpts := make([]runner.Option, 0, 2)
		if c.runtimeMgr != nil {
			runnerOpts = append(runnerOpts, runner.WithRuntimeManager(c.runtimeMgr))
		}
		if c.localRegistry != nil {
			runnerOpts = append(runnerOpts, runner.WithLocalRegistry(c.localRegistry))
		}
		c.runner = runner.New(model, runnerOpts...)
	}
	if c.teams == nil {
		c.teams = teams.New(teams.WithTimelineEmitter(c.handler))
	}
	if c.workflow == nil {
		c.workflow = workflow.New(workflow.WithTimelineEmitter(c.handler))
	}

	if c.scheduler == nil {
		if err := c.initScheduler(c.effectiveConfig()); err != nil {
			return nil, err
		}
	} else {
		c.managedScheduler = false
		c.schedulerConfiguredBackend = "custom"
		c.schedulerBackend = "custom"
	}
	if err := c.initRecovery(c.effectiveConfig()); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Composer) Run(ctx context.Context, req types.RunRequest, h types.EventHandler) (types.RunResult, error) {
	c.refreshSchedulerForNextAttempt()
	c.refreshRecoveryForNextAttempt()
	return c.runner.Run(ctx, req, c.bridgeHandler(h))
}

func (c *Composer) Stream(ctx context.Context, req types.RunRequest, h types.EventHandler) (types.RunResult, error) {
	c.refreshSchedulerForNextAttempt()
	c.refreshRecoveryForNextAttempt()
	return c.runner.Stream(ctx, req, c.bridgeHandler(h))
}

func (c *Composer) Teams() TeamsEngine {
	return c.teams
}

func (c *Composer) Workflow() WorkflowEngine {
	return c.workflow
}

func (c *Composer) Scheduler() *scheduler.Scheduler {
	c.schedulerMu.RLock()
	defer c.schedulerMu.RUnlock()
	return c.scheduler
}

func (c *Composer) SchedulerStats(ctx context.Context) (scheduler.Stats, error) {
	s := c.Scheduler()
	if s == nil {
		return scheduler.Stats{}, errors.New("scheduler is not initialized")
	}
	return s.Stats(ctx)
}

func (c *Composer) SpawnChild(ctx context.Context, req ChildDispatchRequest) (scheduler.TaskRecord, error) {
	c.refreshSchedulerForNextAttempt()
	s := c.Scheduler()
	if s == nil {
		return scheduler.TaskRecord{}, errors.New("scheduler is not initialized")
	}

	task := req.Task
	if task.MaxAttempts <= 0 && c.schedulerRetryMaxAttempts > 0 {
		task.MaxAttempts = c.schedulerRetryMaxAttempts
	}
	if c.schedulerQueueLimit > 0 {
		stats, err := s.Stats(ctx)
		if err == nil && stats.QueueTotal >= c.schedulerQueueLimit {
			return scheduler.TaskRecord{}, fmt.Errorf("scheduler queue_limit exceeded: queue_total=%d limit=%d", stats.QueueTotal, c.schedulerQueueLimit)
		}
	}

	record, err := s.SpawnChild(ctx, scheduler.SpawnRequest{
		Task:                 task,
		ParentDepth:          req.ParentDepth,
		ParentActiveChildren: req.ParentActiveChildren,
		ChildTimeout:         req.ChildTimeout,
	})
	if err != nil {
		var budgetErr *scheduler.BudgetError
		if errors.As(err, &budgetErr) {
			c.addBudgetReject(strings.TrimSpace(req.Task.RunID))
		}
		return scheduler.TaskRecord{}, err
	}
	c.maybePersistRecoverySnapshot(ctx, strings.TrimSpace(record.Task.RunID))
	return record, nil
}

func (c *Composer) DispatchChild(ctx context.Context, req ChildDispatchRequest) (ChildDispatchResult, error) {
	record, err := c.SpawnChild(ctx, req)
	if err != nil {
		return ChildDispatchResult{}, err
	}
	s := c.Scheduler()
	if s == nil {
		return ChildDispatchResult{}, errors.New("scheduler is not initialized")
	}

	workerID := strings.TrimSpace(req.WorkerID)
	if workerID == "" {
		workerID = c.childWorkerID
	}
	claimed, ok, err := s.Claim(ctx, workerID)
	if err != nil {
		return ChildDispatchResult{}, err
	}
	if !ok {
		return ChildDispatchResult{}, errors.New("scheduler claim returned no task")
	}
	if claimed.Record.Task.TaskID != record.Task.TaskID {
		return ChildDispatchResult{}, fmt.Errorf(
			"claimed task mismatch: got=%q want=%q",
			claimed.Record.Task.TaskID,
			record.Task.TaskID,
		)
	}

	execution, execErr := c.executeChild(ctx, req, claimed)
	commitMeta, commitErr := c.CommitChildTerminal(ctx, execution.Commit)
	out := ChildDispatchResult{
		Record:     claimed.Record,
		Claimed:    claimed,
		Commit:     execution.Commit,
		CommitMeta: commitMeta,
		Retryable:  execution.Retryable,
	}
	if commitErr != nil {
		return out, commitErr
	}
	return out, execErr
}

func (c *Composer) CommitChildTerminal(ctx context.Context, commit scheduler.TerminalCommit) (scheduler.CommitResult, error) {
	s := c.Scheduler()
	if s == nil {
		return scheduler.CommitResult{}, errors.New("scheduler is not initialized")
	}
	var (
		result scheduler.CommitResult
		err    error
	)
	switch commit.Status {
	case scheduler.TaskStateSucceeded:
		result, err = s.Complete(ctx, commit)
	case scheduler.TaskStateFailed:
		result, err = s.Fail(ctx, commit)
	default:
		return scheduler.CommitResult{}, fmt.Errorf("unsupported terminal status %q", commit.Status)
	}
	if err != nil {
		return scheduler.CommitResult{}, err
	}
	if !result.Duplicate {
		c.addChildOutcome(result.Record.Task.RunID, commit.Status == scheduler.TaskStateFailed)
	}
	c.maybePersistRecoverySnapshot(ctx, strings.TrimSpace(result.Record.Task.RunID))
	return result, nil
}

func (c *Composer) executeChild(
	ctx context.Context,
	req ChildDispatchRequest,
	claimed scheduler.ClaimedTask,
) (scheduler.A2AExecution, error) {
	switch req.Target {
	case ChildTargetLocal:
		return c.executeLocalChild(ctx, req, claimed)
	case ChildTargetA2A:
		if c.a2aClient == nil {
			err := errors.New("a2a client is not configured")
			return scheduler.A2AExecution{
				Commit: scheduler.TerminalCommit{
					TaskID:       claimed.Record.Task.TaskID,
					AttemptID:    claimed.Attempt.AttemptID,
					Status:       scheduler.TaskStateFailed,
					ErrorMessage: err.Error(),
					ErrorClass:   types.ErrMCP,
					ErrorLayer:   "transport",
					CommittedAt:  c.now(),
				},
			}, err
		}
		pollInterval := req.PollInterval
		if pollInterval <= 0 {
			pollInterval = c.childPollInterval
		}
		return scheduler.ExecuteClaimWithA2A(ctx, c.a2aClient, claimed, pollInterval)
	default:
		err := fmt.Errorf("unsupported child target %q", req.Target)
		return scheduler.A2AExecution{
			Commit: scheduler.TerminalCommit{
				TaskID:       claimed.Record.Task.TaskID,
				AttemptID:    claimed.Attempt.AttemptID,
				Status:       scheduler.TaskStateFailed,
				ErrorMessage: err.Error(),
				ErrorClass:   types.ErrContext,
				ErrorLayer:   "semantic",
				CommittedAt:  c.now(),
			},
		}, err
	}
}

func (c *Composer) executeLocalChild(
	ctx context.Context,
	req ChildDispatchRequest,
	claimed scheduler.ClaimedTask,
) (scheduler.A2AExecution, error) {
	if req.LocalRunner == nil {
		err := errors.New("local child runner is required")
		return scheduler.A2AExecution{
			Commit: scheduler.TerminalCommit{
				TaskID:       claimed.Record.Task.TaskID,
				AttemptID:    claimed.Attempt.AttemptID,
				Status:       scheduler.TaskStateFailed,
				ErrorMessage: err.Error(),
				ErrorClass:   types.ErrTool,
				ErrorLayer:   "local",
				CommittedAt:  c.now(),
			},
		}, err
	}
	result, err := req.LocalRunner.RunChild(ctx, claimed.Record.Task)
	if err != nil {
		class := types.ErrTool
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			class = types.ErrPolicyTimeout
		}
		return scheduler.A2AExecution{
			Commit: scheduler.TerminalCommit{
				TaskID:       claimed.Record.Task.TaskID,
				AttemptID:    claimed.Attempt.AttemptID,
				Status:       scheduler.TaskStateFailed,
				ErrorMessage: err.Error(),
				ErrorClass:   class,
				ErrorLayer:   "local",
				CommittedAt:  c.now(),
			},
		}, err
	}
	return scheduler.A2AExecution{
		Commit: scheduler.TerminalCommit{
			TaskID:      claimed.Record.Task.TaskID,
			AttemptID:   claimed.Attempt.AttemptID,
			Status:      scheduler.TaskStateSucceeded,
			Result:      cloneMap(result),
			CommittedAt: c.now(),
		},
	}, nil
}

func (c *Composer) bridgeHandler(perCall types.EventHandler) types.EventHandler {
	return eventHandlerFunc(func(ctx context.Context, ev types.Event) {
		out := ev
		if out.Type == "run.finished" {
			out = c.injectRunSummary(out)
		}
		if c.handler != nil {
			c.handler.OnEvent(ctx, out)
		}
		if perCall != nil {
			perCall.OnEvent(ctx, out)
		}
	})
}

func (c *Composer) injectRunSummary(ev types.Event) types.Event {
	payload := cloneMap(ev.Payload)
	runID := strings.TrimSpace(ev.RunID)
	stats := c.snapshotRunStat(runID)

	payload["composer_managed"] = true
	if stats.Backend != "" {
		payload["scheduler_backend"] = stats.Backend
	}
	if stats.BackendFallback {
		payload["scheduler_backend_fallback"] = true
	}
	if stats.FallbackReason != "" {
		payload["scheduler_backend_fallback_reason"] = stats.FallbackReason
	}
	payload["subagent_child_total"] = stats.ChildTotal
	payload["subagent_child_failed"] = stats.ChildFailed
	payload["subagent_budget_reject_total"] = stats.BudgetReject
	payload["recovery_enabled"] = stats.RecoveryEnabled
	if stats.RecoveryRecovered {
		payload["recovery_recovered"] = true
	}
	if stats.RecoveryReplayTotal > 0 {
		payload["recovery_replay_total"] = stats.RecoveryReplayTotal
	}
	if stats.RecoveryConflict {
		payload["recovery_conflict"] = true
	}
	if stats.RecoveryConflictCode != "" {
		payload["recovery_conflict_code"] = stats.RecoveryConflictCode
	}
	if stats.RecoveryFallback {
		payload["recovery_fallback_used"] = true
	}
	if stats.RecoveryFallbackReason != "" {
		payload["recovery_fallback_reason"] = stats.RecoveryFallbackReason
	}

	if s := c.Scheduler(); s != nil {
		summary, err := s.Stats(context.Background())
		if err == nil {
			payload["scheduler_backend"] = strings.TrimSpace(summary.Backend)
			if strings.TrimSpace(summary.QoSMode) != "" {
				payload["scheduler_qos_mode"] = strings.TrimSpace(summary.QoSMode)
			}
			payload["scheduler_queue_total"] = summary.QueueTotal
			payload["scheduler_claim_total"] = summary.ClaimTotal
			payload["scheduler_reclaim_total"] = summary.ReclaimTotal
			payload["scheduler_priority_claim_total"] = summary.PriorityClaimTotal
			payload["scheduler_fairness_yield_total"] = summary.FairnessYieldTotal
			payload["scheduler_retry_backoff_total"] = summary.RetryBackoffTotal
			payload["scheduler_dead_letter_total"] = summary.DeadLetterTotal
		}
	}

	ev.Payload = payload
	return ev
}

func (c *Composer) snapshotRunStat(runID string) runStat {
	stat := runStat{
		ComposerManaged:        true,
		Backend:                strings.TrimSpace(c.schedulerBackend),
		BackendFallback:        c.schedulerFallback,
		FallbackReason:         strings.TrimSpace(c.schedulerFallbackReason),
		RecoveryEnabled:        c.recoveryEnabled,
		RecoveryFallback:       c.recoveryFallback,
		RecoveryFallbackReason: strings.TrimSpace(c.recoveryFallbackReason),
	}
	if runID == "" {
		return stat
	}
	c.runMu.Lock()
	defer c.runMu.Unlock()
	current, ok := c.runStat[runID]
	if !ok {
		c.runStat[runID] = &stat
		return stat
	}
	return *current
}

func (c *Composer) addChildOutcome(runID string, failed bool) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return
	}
	c.runMu.Lock()
	defer c.runMu.Unlock()
	stat := c.ensureRunStat(runID)
	stat.ChildTotal++
	if failed {
		stat.ChildFailed++
	}
}

func (c *Composer) addBudgetReject(runID string) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return
	}
	c.runMu.Lock()
	defer c.runMu.Unlock()
	stat := c.ensureRunStat(runID)
	stat.BudgetReject++
}

func (c *Composer) ensureRunStat(runID string) *runStat {
	if stat, ok := c.runStat[runID]; ok {
		return stat
	}
	stat := &runStat{
		ComposerManaged:        true,
		Backend:                strings.TrimSpace(c.schedulerBackend),
		BackendFallback:        c.schedulerFallback,
		FallbackReason:         strings.TrimSpace(c.schedulerFallbackReason),
		RecoveryEnabled:        c.recoveryEnabled,
		RecoveryFallback:       c.recoveryFallback,
		RecoveryFallbackReason: strings.TrimSpace(c.recoveryFallbackReason),
	}
	c.runStat[runID] = stat
	return stat
}

func (c *Composer) initScheduler(cfg runtimeconfig.Config) error {
	backend := strings.TrimSpace(strings.ToLower(cfg.Scheduler.Backend))
	if backend == "" {
		backend = runtimeconfig.SchedulerBackendMemory
	}

	store := c.schedulerStore
	fallback := false
	fallbackReason := ""
	configuredBackend := backend
	if store == nil {
		var err error
		switch backend {
		case runtimeconfig.SchedulerBackendFile:
			store, err = scheduler.NewFileStore(cfg.Scheduler.Path)
			if err != nil {
				store = scheduler.NewMemoryStore()
				fallback = true
				fallbackReason = "scheduler.backend.file_init_failed"
				backend = runtimeconfig.SchedulerBackendMemory
			}
		default:
			backend = runtimeconfig.SchedulerBackendMemory
			store = scheduler.NewMemoryStore()
		}
	}

	leaseTimeout := cfg.Scheduler.LeaseTimeout
	if leaseTimeout <= 0 {
		leaseTimeout = 2 * time.Second
	}
	guardrails := scheduler.Guardrails{
		MaxDepth:           cfg.Subagent.MaxDepth,
		MaxActiveChildren:  cfg.Subagent.MaxActiveChildren,
		ChildTimeoutBudget: cfg.Subagent.ChildTimeoutBudget,
	}
	governance := c.schedulerGovernanceConfig(cfg)
	s, err := scheduler.New(
		store,
		scheduler.WithTimelineEmitter(c.handler),
		scheduler.WithLeaseTimeout(leaseTimeout),
		scheduler.WithGuardrails(guardrails),
		scheduler.WithGovernance(governance),
	)
	if err != nil {
		return err
	}

	c.schedulerMu.Lock()
	c.scheduler = s
	c.schedulerStore = store
	c.schedulerConfiguredBackend = configuredBackend
	c.schedulerBackend = backend
	c.schedulerFallback = fallback
	c.schedulerFallbackReason = fallbackReason
	c.schedulerQueueLimit = cfg.Scheduler.QueueLimit
	c.schedulerRetryMaxAttempts = cfg.Scheduler.RetryMaxAttempts
	c.schedulerGuardrails = guardrails
	c.schedulerSignature = c.schedulerConfigSignature(cfg)
	c.schedulerMu.Unlock()
	return nil
}

func (c *Composer) refreshSchedulerForNextAttempt() {
	if c == nil || !c.managedScheduler || c.runtimeMgr == nil {
		return
	}
	cfg := c.runtimeMgr.EffectiveConfig()
	signature := c.schedulerConfigSignature(cfg)

	c.schedulerMu.RLock()
	if c.schedulerSignature == signature {
		c.schedulerMu.RUnlock()
		return
	}
	store := c.schedulerStore
	c.schedulerMu.RUnlock()
	if store == nil {
		return
	}

	leaseTimeout := cfg.Scheduler.LeaseTimeout
	if leaseTimeout <= 0 {
		leaseTimeout = 2 * time.Second
	}
	guardrails := scheduler.Guardrails{
		MaxDepth:           cfg.Subagent.MaxDepth,
		MaxActiveChildren:  cfg.Subagent.MaxActiveChildren,
		ChildTimeoutBudget: cfg.Subagent.ChildTimeoutBudget,
	}
	governance := c.schedulerGovernanceConfig(cfg)
	updated, err := scheduler.New(
		store,
		scheduler.WithTimelineEmitter(c.handler),
		scheduler.WithLeaseTimeout(leaseTimeout),
		scheduler.WithGuardrails(guardrails),
		scheduler.WithGovernance(governance),
	)
	if err != nil {
		return
	}

	c.schedulerMu.Lock()
	c.scheduler = updated
	c.schedulerConfiguredBackend = strings.TrimSpace(strings.ToLower(cfg.Scheduler.Backend))
	c.schedulerQueueLimit = cfg.Scheduler.QueueLimit
	c.schedulerRetryMaxAttempts = cfg.Scheduler.RetryMaxAttempts
	c.schedulerGuardrails = guardrails
	c.schedulerSignature = signature
	c.schedulerMu.Unlock()
}

func (c *Composer) schedulerConfigSignature(cfg runtimeconfig.Config) string {
	return fmt.Sprintf(
		"%d|%d|%d|%d|%d|%d|%s|%d|%t|%t|%d|%d|%.4f|%.4f",
		cfg.Scheduler.LeaseTimeout.Milliseconds(),
		cfg.Subagent.MaxDepth,
		cfg.Subagent.MaxActiveChildren,
		cfg.Subagent.ChildTimeoutBudget.Milliseconds(),
		cfg.Scheduler.QueueLimit,
		cfg.Scheduler.RetryMaxAttempts,
		strings.TrimSpace(strings.ToLower(cfg.Scheduler.QoS.Mode)),
		cfg.Scheduler.QoS.Fairness.MaxConsecutiveClaimsPerPriority,
		cfg.Scheduler.DLQ.Enabled,
		cfg.Scheduler.Retry.Backoff.Enabled,
		cfg.Scheduler.Retry.Backoff.Initial.Milliseconds(),
		cfg.Scheduler.Retry.Backoff.Max.Milliseconds(),
		cfg.Scheduler.Retry.Backoff.Multiplier,
		cfg.Scheduler.Retry.Backoff.JitterRatio,
	)
}

func (c *Composer) schedulerGovernanceConfig(cfg runtimeconfig.Config) scheduler.GovernanceConfig {
	mode := scheduler.QoSMode(strings.TrimSpace(strings.ToLower(cfg.Scheduler.QoS.Mode)))
	if mode == "" {
		mode = scheduler.QoSModeFIFO
	}
	return scheduler.GovernanceConfig{
		QoS: mode,
		Fairness: scheduler.FairnessConfig{
			MaxConsecutiveClaimsPerPriority: cfg.Scheduler.QoS.Fairness.MaxConsecutiveClaimsPerPriority,
		},
		DLQ: scheduler.DLQConfig{
			Enabled: cfg.Scheduler.DLQ.Enabled,
		},
		Backoff: scheduler.RetryBackoffConfig{
			Enabled:     cfg.Scheduler.Retry.Backoff.Enabled,
			Initial:     cfg.Scheduler.Retry.Backoff.Initial,
			Max:         cfg.Scheduler.Retry.Backoff.Max,
			Multiplier:  cfg.Scheduler.Retry.Backoff.Multiplier,
			JitterRatio: cfg.Scheduler.Retry.Backoff.JitterRatio,
		},
	}
}

func (c *Composer) effectiveConfig() runtimeconfig.Config {
	if c.runtimeMgr == nil {
		return runtimeconfig.DefaultConfig()
	}
	return c.runtimeMgr.EffectiveConfig()
}

type eventHandlerFunc func(ctx context.Context, ev types.Event)

func (f eventHandlerFunc) OnEvent(ctx context.Context, ev types.Event) {
	f(ctx, ev)
}

func cloneMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
