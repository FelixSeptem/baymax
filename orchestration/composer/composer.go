package composer

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/FelixSeptem/baymax/a2a"
	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/orchestration/scheduler"
	orchestrationsnapshot "github.com/FelixSeptem/baymax/orchestration/snapshot"
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
	Task                  scheduler.Task
	Target                ChildTarget
	Async                 bool
	OperationProfile      string
	RequestTimeout        time.Duration
	ParentDepth           int
	ParentActiveChildren  int
	ParentRemainingBudget time.Duration
	ChildTimeout          time.Duration
	WorkerID              string
	PollInterval          time.Duration
	LocalRunner           LocalChildRunner
}

type ChildDispatchResult struct {
	Record        scheduler.TaskRecord
	Claimed       scheduler.ClaimedTask
	Commit        scheduler.TerminalCommit
	CommitMeta    scheduler.CommitResult
	Retryable     bool
	AsyncAccepted bool
	AsyncTaskID   string
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

	managedMailbox           bool
	mailbox                  *schedulerManagedMailbox
	mailboxSignature         string
	mailboxConfiguredBackend string
	mailboxBackend           string
	mailboxFallback          bool
	mailboxFallbackReason    string
	mailboxPath              string
	mailboxEnabled           bool

	managedRecoveryStore             bool
	recoverySignature                string
	recoveryConfiguredBackend        string
	recoveryBackend                  string
	recoveryPath                     string
	recoveryEnabled                  bool
	recoveryResumeBoundary           string
	recoveryInflightPolicy           string
	recoveryTimeoutReentryPolicy     string
	recoveryTimeoutReentryMaxPerTask int
	recoveryFallback                 bool
	recoveryFallbackReason           string
	recoveryConflictPolicy           string
	stateSnapshotImporter            *orchestrationsnapshot.Importer

	now               func() time.Time
	childWorkerID     string
	childPollInterval time.Duration

	reconcileMu     sync.Mutex
	reconcileCancel context.CancelFunc
	reconcileDone   chan struct{}

	runMu   sync.Mutex
	runStat map[string]*runStat
}

type runStat struct {
	ChildTotal                           int
	ChildFailed                          int
	BudgetReject                         int
	TimeoutParentBudgetClamp             int
	TimeoutParentBudgetReject            int
	ReadinessAdmissionTotal              int
	ReadinessAdmissionBlockedTotal       int
	ReadinessAdmissionDegradedAllowTotal int
	ReadinessAdmissionBypassTotal        int
	ReadinessAdmissionMode               string
	ReadinessAdmissionPrimaryCode        string
	BudgetDecision                       string
	DegradeAction                        string
	BudgetSnapshot                       runtimeconfig.RuntimeAdmissionBudgetSnapshot
	BudgetSnapshotSet                    bool
	PolicyPrecedenceVersion              string
	WinnerStage                          string
	DenySource                           string
	TieBreakReason                       string
	PolicyDecisionPath                   []runtimeconfig.RuntimePolicyCandidate
	AdapterAllowlistDecision             string
	AdapterAllowlistBlockTotal           int
	AdapterAllowlistPrimaryCode          string
	SandboxRolloutPhase                  string
	SandboxCapacityAction                string
	SandboxCapacityDegradedPolicy        string
	ArbitrationRuleRequestedVersion      string
	ArbitrationRuleEffectiveVersion      string
	ArbitrationRuleVersionSource         string
	ArbitrationRulePolicyAction          string
	ArbitrationRuleUnsupportedTotal      int
	ArbitrationRuleMismatchTotal         int
	EffectiveOperationProfile            string
	TimeoutResolutionSource              string
	TimeoutResolutionTrace               string
	CollabHandoffTotal                   int
	CollabDelegationTotal                int
	CollabAggregationTotal               int
	CollabAggregationStrategy            string
	CollabFailFastTotal                  int
	CollabRetryTotal                     int
	CollabRetrySuccessTotal              int
	CollabRetryExhaustedTotal            int
	A2AAsyncReportTotal                  int
	A2AAsyncReportFailed                 int
	A2AAsyncReportRetry                  int
	A2AAsyncReportDedup                  int
	AsyncAwaitTotal                      int
	AsyncTimeoutTotal                    int
	AsyncLateReportTotal                 int
	AsyncReportDedupTotal                int
	Backend                              string
	BackendFallback                      bool
	FallbackReason                       string
	ComposerManaged                      bool
	RecoveryEnabled                      bool
	RecoveryRecovered                    bool
	RecoveryReplayTotal                  int
	RecoveryConflict                     bool
	RecoveryConflictCode                 string
	RecoveryResumeBoundary               string
	RecoveryInflightPolicy               string
	RecoveryFallback                     bool
	RecoveryFallbackReason               string
	asyncReportSeen                      map[string]struct{}
	asyncReportDedupSeen                 map[string]struct{}
	asyncLateReportSeen                  map[string]struct{}
	asyncAwaitSeen                       map[string]struct{}
	timeoutResolutionSeen                map[string]struct{}
	timeoutClampSeen                     map[string]struct{}
}

func New(model types.ModelClient, opts ...Option) (*Composer, error) {
	c := &Composer{
		now:                   time.Now,
		childWorkerID:         defaultChildWorkerID,
		childPollInterval:     defaultChildPollTimeout,
		managedScheduler:      true,
		managedMailbox:        true,
		managedRecoveryStore:  true,
		stateSnapshotImporter: orchestrationsnapshot.NewImporter(),
		runStat:               map[string]*runStat{},
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
		cfg := c.effectiveConfig()
		c.workflow = workflow.New(
			workflow.WithTimelineEmitter(c.handler),
			workflow.WithDefaultStepTimeout(cfg.Workflow.DefaultStepTimeout),
			workflow.WithGraphComposabilityEnabled(cfg.Workflow.GraphComposability.Enabled),
		)
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
	if err := c.initMailbox(c.effectiveConfig()); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Composer) Run(ctx context.Context, req types.RunRequest, h types.EventHandler) (types.RunResult, error) {
	c.refreshSchedulerForNextAttempt()
	c.refreshMailboxForNextAttempt()
	c.refreshRecoveryForNextAttempt()
	req, denied, err := c.guardReadinessAdmission(ctx, req, h)
	if denied != nil || err != nil {
		if denied == nil {
			return types.RunResult{}, err
		}
		return *denied, err
	}
	return c.runner.Run(ctx, req, c.bridgeHandler(h))
}

func (c *Composer) Stream(ctx context.Context, req types.RunRequest, h types.EventHandler) (types.RunResult, error) {
	c.refreshSchedulerForNextAttempt()
	c.refreshMailboxForNextAttempt()
	c.refreshRecoveryForNextAttempt()
	req, denied, err := c.guardReadinessAdmission(ctx, req, h)
	if denied != nil || err != nil {
		if denied == nil {
			return types.RunResult{}, err
		}
		return *denied, err
	}
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
	cfg := c.effectiveConfig()

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

	requestTimeout := req.RequestTimeout
	if requestTimeout <= 0 {
		requestTimeout = req.ChildTimeout
	}
	resolvedTimeout, err := runtimeconfig.ResolveOperationTimeout(cfg, runtimeconfig.TimeoutResolutionInput{
		RequestedProfile: req.OperationProfile,
		DomainTimeout:    cfg.Subagent.ChildTimeoutBudget,
		RequestTimeout:   requestTimeout,
	})
	if err != nil {
		return scheduler.TaskRecord{}, err
	}
	parentRemainingBudget := req.ParentRemainingBudget
	if parentRemainingBudget <= 0 {
		// Compatibility fallback: when parent budget is not explicitly provided, preserve old behavior.
		parentRemainingBudget = resolvedTimeout.EffectiveTimeout
	}

	record, err := s.SpawnChild(ctx, scheduler.SpawnRequest{
		Task:                  task,
		ParentDepth:           req.ParentDepth,
		ParentActiveChildren:  req.ParentActiveChildren,
		ParentRemainingBudget: parentRemainingBudget,
		ChildTimeout:          resolvedTimeout.EffectiveTimeout,
		TimeoutResolution: scheduler.TimeoutResolutionMetadata{
			EffectiveOperationProfile: resolvedTimeout.EffectiveProfile,
			Source:                    resolvedTimeout.Source,
			Trace:                     resolvedTimeout.Trace,
			ResolvedTimeout:           resolvedTimeout.EffectiveTimeout,
		},
	})
	if err != nil {
		var budgetErr *scheduler.BudgetError
		if errors.As(err, &budgetErr) {
			c.addBudgetReject(strings.TrimSpace(req.Task.RunID))
			if budgetErr.Code == scheduler.BudgetRejectParentBudgetExhausted {
				c.addTimeoutParentBudgetReject(strings.TrimSpace(req.Task.RunID))
			}
		}
		return scheduler.TaskRecord{}, err
	}
	c.recordTimeoutResolution(strings.TrimSpace(record.Task.RunID), strings.TrimSpace(record.Task.TaskID), record.Task.TimeoutResolution)
	c.maybePersistRecoverySnapshot(ctx, strings.TrimSpace(record.Task.RunID))
	return record, nil
}

func (c *Composer) DispatchChild(ctx context.Context, req ChildDispatchRequest) (ChildDispatchResult, error) {
	c.refreshMailboxForNextAttempt()
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
	if execution.AsyncAccepted {
		out := ChildDispatchResult{
			Record:        claimed.Record,
			Claimed:       claimed,
			Retryable:     execution.Retryable,
			AsyncAccepted: true,
			AsyncTaskID:   strings.TrimSpace(execution.AsyncTaskID),
		}
		return out, execErr
	}
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
		c.addCollabOutcome(result.Record.Task, commit.Status == scheduler.TaskStateFailed)
	}
	c.maybePersistRecoverySnapshot(ctx, strings.TrimSpace(result.Record.Task.RunID))
	return result, nil
}

func (c *Composer) CommitAsyncReportTerminal(ctx context.Context, commit scheduler.TerminalCommit) (scheduler.CommitResult, error) {
	s := c.Scheduler()
	if s == nil {
		return scheduler.CommitResult{}, errors.New("scheduler is not initialized")
	}
	result, err := s.CommitAsyncReportTerminal(ctx, commit)
	if err != nil {
		return scheduler.CommitResult{}, err
	}
	if !result.Duplicate && !result.LateReport {
		c.addChildOutcome(result.Record.Task.RunID, commit.Status == scheduler.TaskStateFailed)
		c.addCollabOutcome(result.Record.Task, commit.Status == scheduler.TaskStateFailed)
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
		if req.Async {
			return c.executeA2AChildAsync(ctx, claimed)
		}
		pollInterval := req.PollInterval
		if pollInterval <= 0 {
			pollInterval = c.childPollInterval
		}
		// Scheduler-managed child execution is the single retry owner (A33).
		// Keep primitive-layer retry outside this path to avoid compounded retries.
		return scheduler.ExecuteClaimWithA2A(
			ctx,
			c.a2aClient,
			claimed,
			pollInterval,
			scheduler.WithMailboxBridgeProvider(c.mailboxBridgeProvider),
		)
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

func (c *Composer) executeA2AChildAsync(
	ctx context.Context,
	claimed scheduler.ClaimedTask,
) (scheduler.A2AExecution, error) {
	asyncClient, ok := c.a2aClient.(scheduler.A2AAsyncClient)
	if !ok {
		err := errors.New("a2a client does not support async submit")
		return scheduler.A2AExecution{
			Commit: scheduler.TerminalCommit{
				TaskID:       claimed.Record.Task.TaskID,
				AttemptID:    claimed.Attempt.AttemptID,
				Status:       scheduler.TaskStateFailed,
				ErrorMessage: err.Error(),
				ErrorClass:   types.ErrMCP,
				ErrorLayer:   string(a2a.ErrorLayerProtocol),
				CommittedAt:  c.now(),
			},
		}, err
	}

	sink := a2a.NewCallbackReportSink(func(cbCtx context.Context, report a2a.AsyncReport) error {
		if strings.TrimSpace(report.AttemptID) == "" {
			report.AttemptID = strings.TrimSpace(claimed.Attempt.AttemptID)
		}
		if strings.TrimSpace(report.ReportKey) == "" {
			report.ReportKey = a2a.BuildAsyncReportKey(report)
		}
		execution, err := scheduler.ExecutionFromAsyncReport(claimed, report)
		if err != nil {
			return &a2a.AsyncReportDeliveryError{Cause: err, Retryable: false}
		}
		commitMeta, commitErr := c.CommitAsyncReportTerminal(cbCtx, execution.Commit)
		if commitErr != nil {
			retryable := execution.Retryable
			if errors.Is(commitErr, scheduler.ErrTaskNotFound) ||
				errors.Is(commitErr, scheduler.ErrTaskNotRunning) ||
				errors.Is(commitErr, scheduler.ErrTaskNotAwaitingReport) ||
				errors.Is(commitErr, scheduler.ErrStaleAttempt) {
				retryable = false
			}
			return &a2a.AsyncReportDeliveryError{Cause: commitErr, Retryable: retryable}
		}
		runID := strings.TrimSpace(claimed.Record.Task.RunID)
		if commitMeta.LateReport {
			if c.addAsyncLateReportOutcome(runID, report, commitMeta.Duplicate) {
				c.emitAsyncLateReportTimeline(cbCtx, claimed, report)
			}
			return nil
		}
		c.addAsyncReportOutcome(runID, report, commitMeta.Duplicate)
		if commitMeta.Duplicate {
			c.emitAsyncReportDedupTimeline(cbCtx, claimed, report)
		}
		return nil
	})
	ack, err := scheduler.SubmitClaimWithA2AAsync(
		ctx,
		asyncClient,
		claimed,
		sink,
		scheduler.WithMailboxBridgeProvider(c.mailboxBridgeProvider),
	)
	if err != nil {
		class, layer, _ := a2a.ClassifyError(err)
		if class == "" {
			class = types.ErrMCP
		}
		errorLayer := strings.TrimSpace(string(layer))
		if errorLayer == "" {
			errorLayer = string(a2a.ErrorLayerProtocol)
		}
		return scheduler.A2AExecution{
			Commit: scheduler.TerminalCommit{
				TaskID:       claimed.Record.Task.TaskID,
				AttemptID:    claimed.Attempt.AttemptID,
				Status:       scheduler.TaskStateFailed,
				ErrorMessage: strings.TrimSpace(err.Error()),
				ErrorClass:   class,
				ErrorLayer:   errorLayer,
				CommittedAt:  c.now(),
			},
			Retryable: errorLayer == string(a2a.ErrorLayerTransport),
		}, err
	}
	if _, markErr := c.Scheduler().MarkAwaitingReport(ctx, claimed.Record.Task.TaskID, claimed.Attempt.AttemptID, strings.TrimSpace(ack.TaskID)); markErr != nil {
		return scheduler.A2AExecution{
			Commit: scheduler.TerminalCommit{
				TaskID:       claimed.Record.Task.TaskID,
				AttemptID:    claimed.Attempt.AttemptID,
				Status:       scheduler.TaskStateFailed,
				ErrorMessage: strings.TrimSpace(markErr.Error()),
				ErrorClass:   types.ErrContext,
				ErrorLayer:   "scheduler",
				CommittedAt:  c.now(),
			},
		}, markErr
	}
	c.addAsyncAwait(strings.TrimSpace(claimed.Record.Task.RunID), claimed.Record.Task.TaskID, claimed.Attempt.AttemptID)
	return scheduler.A2AExecution{
		AsyncAccepted: true,
		AsyncTaskID:   strings.TrimSpace(ack.TaskID),
	}, nil
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
	payload["timeout_parent_budget_clamp_total"] = stats.TimeoutParentBudgetClamp
	payload["timeout_parent_budget_reject_total"] = stats.TimeoutParentBudgetReject
	if strings.TrimSpace(stats.EffectiveOperationProfile) != "" {
		payload["effective_operation_profile"] = strings.TrimSpace(stats.EffectiveOperationProfile)
	}
	if strings.TrimSpace(stats.TimeoutResolutionSource) != "" {
		payload["timeout_resolution_source"] = strings.TrimSpace(stats.TimeoutResolutionSource)
	}
	if strings.TrimSpace(stats.TimeoutResolutionTrace) != "" {
		payload["timeout_resolution_trace"] = strings.TrimSpace(stats.TimeoutResolutionTrace)
	}
	payload["collab_handoff_total"] = stats.CollabHandoffTotal
	payload["collab_delegation_total"] = stats.CollabDelegationTotal
	payload["collab_aggregation_total"] = stats.CollabAggregationTotal
	if strings.TrimSpace(stats.CollabAggregationStrategy) != "" {
		payload["collab_aggregation_strategy"] = strings.TrimSpace(stats.CollabAggregationStrategy)
	}
	payload["collab_fail_fast_total"] = stats.CollabFailFastTotal
	payload["collab_retry_total"] = stats.CollabRetryTotal
	payload["collab_retry_success_total"] = stats.CollabRetrySuccessTotal
	payload["collab_retry_exhausted_total"] = stats.CollabRetryExhaustedTotal
	payload["a2a_async_report_total"] = stats.A2AAsyncReportTotal
	payload["a2a_async_report_failed"] = stats.A2AAsyncReportFailed
	payload["a2a_async_report_retry_total"] = stats.A2AAsyncReportRetry
	payload["a2a_async_report_dedup_total"] = stats.A2AAsyncReportDedup
	payload["async_await_total"] = stats.AsyncAwaitTotal
	payload["async_timeout_total"] = stats.AsyncTimeoutTotal
	payload["async_late_report_total"] = stats.AsyncLateReportTotal
	payload["async_report_dedup_total"] = stats.AsyncReportDedupTotal
	payload["recovery_enabled"] = stats.RecoveryEnabled
	if strings.TrimSpace(stats.RecoveryResumeBoundary) != "" {
		payload["recovery_resume_boundary"] = strings.TrimSpace(stats.RecoveryResumeBoundary)
	}
	if strings.TrimSpace(stats.RecoveryInflightPolicy) != "" {
		payload["recovery_inflight_policy"] = strings.TrimSpace(stats.RecoveryInflightPolicy)
	}
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
	var readinessFindings []runtimeconfig.ReadinessFinding
	if readiness, err := c.ReadinessPreflight(); err == nil {
		readinessFindings = append([]runtimeconfig.ReadinessFinding(nil), readiness.Findings...)
		summary := readiness.Summary()
		payload["runtime_readiness_status"] = summary.Status
		payload["runtime_readiness_finding_total"] = summary.FindingTotal
		payload["runtime_readiness_blocking_total"] = summary.BlockingTotal
		payload["runtime_readiness_degraded_total"] = summary.DegradedTotal
		if summary.PrimaryCode != "" {
			payload["runtime_readiness_primary_code"] = summary.PrimaryCode
		}
		if len(summary.SecondaryReasonCodes) > 0 {
			payload["runtime_secondary_reason_codes"] = append([]string(nil), summary.SecondaryReasonCodes...)
		}
		payload["runtime_secondary_reason_count"] = summary.SecondaryReasonCount
		if summary.ArbitrationRuleVersion != "" {
			payload["runtime_arbitration_rule_version"] = summary.ArbitrationRuleVersion
		}
		if summary.ArbitrationRuleRequestedVersion != "" {
			payload["runtime_arbitration_rule_requested_version"] = summary.ArbitrationRuleRequestedVersion
		}
		if summary.ArbitrationRuleEffectiveVersion != "" {
			payload["runtime_arbitration_rule_effective_version"] = summary.ArbitrationRuleEffectiveVersion
		}
		if summary.ArbitrationRuleVersionSource != "" {
			payload["runtime_arbitration_rule_version_source"] = summary.ArbitrationRuleVersionSource
		}
		if summary.ArbitrationRulePolicyAction != "" {
			payload["runtime_arbitration_rule_policy_action"] = summary.ArbitrationRulePolicyAction
		}
		payload["runtime_arbitration_rule_unsupported_total"] = summary.ArbitrationRuleUnsupportedTotal
		payload["runtime_arbitration_rule_mismatch_total"] = summary.ArbitrationRuleMismatchTotal
		if summary.RemediationHintCode != "" {
			payload["runtime_remediation_hint_code"] = summary.RemediationHintCode
		}
		if summary.RemediationHintDomain != "" {
			payload["runtime_remediation_hint_domain"] = summary.RemediationHintDomain
		}
		if summary.AdapterHealthStatus != "" {
			payload["adapter_health_status"] = summary.AdapterHealthStatus
		}
		payload["adapter_health_probe_total"] = summary.AdapterHealthProbeTotal
		payload["adapter_health_degraded_total"] = summary.AdapterHealthDegradedTotal
		payload["adapter_health_unavailable_total"] = summary.AdapterHealthUnavailableTotal
		if summary.AdapterHealthPrimaryCode != "" {
			payload["adapter_health_primary_code"] = summary.AdapterHealthPrimaryCode
		}
		payload["adapter_health_backoff_applied_total"] = summary.AdapterHealthBackoffAppliedTotal
		payload["adapter_health_circuit_open_total"] = summary.AdapterHealthCircuitOpenTotal
		payload["adapter_health_circuit_half_open_total"] = summary.AdapterHealthCircuitHalfOpenTotal
		payload["adapter_health_circuit_recover_total"] = summary.AdapterHealthCircuitRecoverTotal
		if summary.AdapterHealthCircuitState != "" {
			payload["adapter_health_circuit_state"] = summary.AdapterHealthCircuitState
		}
		if summary.AdapterHealthGovernancePrimaryCode != "" {
			payload["adapter_health_governance_primary_code"] = summary.AdapterHealthGovernancePrimaryCode
		}
	}
	if strings.TrimSpace(stats.ArbitrationRuleRequestedVersion) != "" {
		payload["runtime_arbitration_rule_requested_version"] = strings.TrimSpace(stats.ArbitrationRuleRequestedVersion)
	}
	if strings.TrimSpace(stats.ArbitrationRuleEffectiveVersion) != "" {
		payload["runtime_arbitration_rule_effective_version"] = strings.TrimSpace(stats.ArbitrationRuleEffectiveVersion)
	}
	if strings.TrimSpace(stats.ArbitrationRuleVersionSource) != "" {
		payload["runtime_arbitration_rule_version_source"] = strings.TrimSpace(stats.ArbitrationRuleVersionSource)
	}
	if strings.TrimSpace(stats.ArbitrationRulePolicyAction) != "" {
		payload["runtime_arbitration_rule_policy_action"] = strings.TrimSpace(stats.ArbitrationRulePolicyAction)
	}
	payload["runtime_arbitration_rule_unsupported_total"] = stats.ArbitrationRuleUnsupportedTotal
	payload["runtime_arbitration_rule_mismatch_total"] = stats.ArbitrationRuleMismatchTotal
	cfg := c.effectiveConfig()
	primary := runtimeconfig.ArbitratePrimaryReason(runtimeconfig.PrimaryReasonArbitrationInput{
		TimeoutParentBudgetRejectTotal: stats.TimeoutParentBudgetReject,
		TimeoutParentBudgetClampTotal:  stats.TimeoutParentBudgetClamp,
		TimeoutExhaustedTotal:          stats.AsyncTimeoutTotal,
		TimeoutResolutionSource:        stats.TimeoutResolutionSource,
		ReadinessFindings:              readinessFindings,
		RequestedRuleVersion:           strings.TrimSpace(stats.ArbitrationRuleRequestedVersion),
		VersionConfig:                  cfg.Runtime.Arbitration.Version,
	})
	if strings.TrimSpace(primary.Domain) != "" {
		payload["runtime_primary_domain"] = strings.TrimSpace(primary.Domain)
	}
	if strings.TrimSpace(primary.Code) != "" {
		payload["runtime_primary_code"] = strings.TrimSpace(primary.Code)
	}
	if strings.TrimSpace(primary.Source) != "" {
		payload["runtime_primary_source"] = strings.TrimSpace(primary.Source)
	}
	payload["runtime_primary_conflict_total"] = primary.ConflictTotal
	if len(primary.SecondaryCodes) > 0 {
		payload["runtime_secondary_reason_codes"] = append([]string(nil), primary.SecondaryCodes...)
	}
	payload["runtime_secondary_reason_count"] = primary.SecondaryCount
	if strings.TrimSpace(primary.RuleVersion) != "" {
		payload["runtime_arbitration_rule_version"] = strings.TrimSpace(primary.RuleVersion)
	}
	if strings.TrimSpace(primary.RuleRequestedVersion) != "" {
		payload["runtime_arbitration_rule_requested_version"] = strings.TrimSpace(primary.RuleRequestedVersion)
	}
	if strings.TrimSpace(primary.RuleEffectiveVersion) != "" {
		payload["runtime_arbitration_rule_effective_version"] = strings.TrimSpace(primary.RuleEffectiveVersion)
	}
	if strings.TrimSpace(primary.RuleVersionSource) != "" {
		payload["runtime_arbitration_rule_version_source"] = strings.TrimSpace(primary.RuleVersionSource)
	}
	if strings.TrimSpace(primary.RulePolicyAction) != "" {
		payload["runtime_arbitration_rule_policy_action"] = strings.TrimSpace(primary.RulePolicyAction)
	}
	payload["runtime_arbitration_rule_unsupported_total"] = primary.RuleUnsupportedTotal
	payload["runtime_arbitration_rule_mismatch_total"] = primary.RuleMismatchTotal
	if strings.TrimSpace(primary.RemediationHintCode) != "" {
		payload["runtime_remediation_hint_code"] = strings.TrimSpace(primary.RemediationHintCode)
	}
	if strings.TrimSpace(primary.RemediationHintDomain) != "" {
		payload["runtime_remediation_hint_domain"] = strings.TrimSpace(primary.RemediationHintDomain)
	}
	payload["runtime_readiness_admission_total"] = stats.ReadinessAdmissionTotal
	payload["runtime_readiness_admission_blocked_total"] = stats.ReadinessAdmissionBlockedTotal
	payload["runtime_readiness_admission_degraded_allow_total"] = stats.ReadinessAdmissionDegradedAllowTotal
	payload["runtime_readiness_admission_bypass_total"] = stats.ReadinessAdmissionBypassTotal
	if strings.TrimSpace(stats.ReadinessAdmissionMode) != "" {
		payload["runtime_readiness_admission_mode"] = strings.TrimSpace(stats.ReadinessAdmissionMode)
	}
	if strings.TrimSpace(stats.ReadinessAdmissionPrimaryCode) != "" {
		payload["runtime_readiness_admission_primary_code"] = strings.TrimSpace(stats.ReadinessAdmissionPrimaryCode)
	}
	if strings.TrimSpace(stats.BudgetDecision) != "" {
		payload["budget_decision"] = strings.TrimSpace(stats.BudgetDecision)
	}
	if strings.TrimSpace(stats.DegradeAction) != "" {
		payload["degrade_action"] = strings.TrimSpace(stats.DegradeAction)
	}
	if stats.BudgetSnapshotSet {
		payload["budget_snapshot"] = stats.BudgetSnapshot
	}
	if strings.TrimSpace(stats.PolicyPrecedenceVersion) != "" {
		payload["policy_precedence_version"] = strings.TrimSpace(stats.PolicyPrecedenceVersion)
	}
	if strings.TrimSpace(stats.WinnerStage) != "" {
		payload["winner_stage"] = strings.TrimSpace(stats.WinnerStage)
	}
	if strings.TrimSpace(stats.DenySource) != "" {
		payload["deny_source"] = strings.TrimSpace(stats.DenySource)
	}
	if strings.TrimSpace(stats.TieBreakReason) != "" {
		payload["tie_break_reason"] = strings.TrimSpace(stats.TieBreakReason)
	}
	if len(stats.PolicyDecisionPath) > 0 {
		payload["policy_decision_path"] = cloneRuntimePolicyCandidates(stats.PolicyDecisionPath)
	}
	if strings.TrimSpace(stats.AdapterAllowlistDecision) != "" {
		payload["adapter_allowlist_decision"] = strings.TrimSpace(stats.AdapterAllowlistDecision)
	}
	payload["adapter_allowlist_block_total"] = stats.AdapterAllowlistBlockTotal
	if strings.TrimSpace(stats.AdapterAllowlistPrimaryCode) != "" {
		payload["adapter_allowlist_primary_code"] = strings.TrimSpace(stats.AdapterAllowlistPrimaryCode)
	}
	if strings.TrimSpace(stats.SandboxRolloutPhase) != "" {
		payload["sandbox_rollout_phase"] = strings.TrimSpace(stats.SandboxRolloutPhase)
	}
	if strings.TrimSpace(stats.SandboxCapacityAction) != "" {
		payload["sandbox_capacity_action"] = strings.TrimSpace(stats.SandboxCapacityAction)
	}
	if strings.TrimSpace(stats.SandboxCapacityDegradedPolicy) != "" {
		payload["sandbox_capacity_degraded_policy"] = strings.TrimSpace(stats.SandboxCapacityDegradedPolicy)
	}
	if c.runtimeMgr != nil {
		rolloutCfg := c.runtimeMgr.EffectiveConfig().Security.Sandbox.Rollout
		payload["sandbox_rollout_effective_ratio"] = rolloutCfg.TrafficRatio
		state := c.runtimeMgr.SandboxRolloutRuntimeState()
		if strings.TrimSpace(state.HealthBudgetStatus) != "" {
			payload["sandbox_health_budget_status"] = strings.TrimSpace(state.HealthBudgetStatus)
		}
		payload["sandbox_health_budget_breach_total"] = state.HealthBudgetBreachTotal
		if _, ok := payload["sandbox_egress_violation_total"]; !ok {
			payload["sandbox_egress_violation_total"] = state.EgressViolationTotal
		}
		payload["sandbox_freeze_state"] = state.FreezeState
		if strings.TrimSpace(state.FreezeReasonCode) != "" {
			payload["sandbox_freeze_reason_code"] = strings.TrimSpace(state.FreezeReasonCode)
		}
		if strings.TrimSpace(state.CapacityAction) != "" {
			payload["sandbox_capacity_action"] = strings.TrimSpace(state.CapacityAction)
		}
		payload["sandbox_capacity_queue_depth"] = state.CapacityQueueDepth
		payload["sandbox_capacity_inflight"] = state.CapacityInflight
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
			payload["scheduler_delayed_task_total"] = summary.DelayedTaskTotal
			payload["scheduler_delayed_claim_total"] = summary.DelayedClaimTotal
			payload["scheduler_delayed_wait_ms_p95"] = summary.DelayedWaitMsP95
			payload["task_board_manual_control_total"] = summary.TaskBoardManualControlTotal
			payload["task_board_manual_control_success_total"] = summary.TaskBoardManualControlSuccessTotal
			payload["task_board_manual_control_rejected_total"] = summary.TaskBoardManualControlRejectedTotal
			payload["task_board_manual_control_idempotent_dedup_total"] = summary.TaskBoardManualControlDedupTotal
			if len(summary.TaskBoardManualControlByAction) > 0 {
				payload["task_board_manual_control_by_action"] = cloneIntMap(summary.TaskBoardManualControlByAction)
			}
			if len(summary.TaskBoardManualControlByReason) > 0 {
				payload["task_board_manual_control_by_reason"] = cloneIntMap(summary.TaskBoardManualControlByReason)
			}
			payload["async_await_total"] = summary.AsyncAwaitTotal
			payload["async_timeout_total"] = summary.AsyncTimeoutTotal
			payload["async_reconcile_poll_total"] = summary.AsyncReconcilePollTotal
			payload["async_reconcile_terminal_by_poll_total"] = summary.AsyncReconcileTerminalByPollTotal
			payload["async_reconcile_error_total"] = summary.AsyncReconcileErrorTotal
			payload["async_terminal_conflict_total"] = summary.AsyncTerminalConflictTotal
			payload["recovery_timeout_reentry_total"] = summary.RecoveryTimeoutReentryTotal
			payload["recovery_timeout_reentry_exhausted_total"] = summary.RecoveryTimeoutReentryExhaustedTotal
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
		RecoveryResumeBoundary: strings.TrimSpace(c.recoveryResumeBoundary),
		RecoveryInflightPolicy: strings.TrimSpace(c.recoveryInflightPolicy),
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
	out := *current
	out.PolicyDecisionPath = cloneRuntimePolicyCandidates(out.PolicyDecisionPath)
	return out
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

func (c *Composer) addTimeoutParentBudgetReject(runID string) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return
	}
	c.runMu.Lock()
	defer c.runMu.Unlock()
	stat := c.ensureRunStat(runID)
	stat.TimeoutParentBudgetReject++
}

func (c *Composer) recordTimeoutResolution(runID, taskID string, meta scheduler.TimeoutResolutionMetadata) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return
	}
	c.runMu.Lock()
	defer c.runMu.Unlock()
	stat := c.ensureRunStat(runID)
	key := strings.TrimSpace(taskID)
	if key == "" {
		key = strings.TrimSpace(meta.Trace)
	}
	if key == "" {
		key = strings.TrimSpace(meta.EffectiveOperationProfile) + "|" + strings.TrimSpace(meta.Source)
	}
	if stat.timeoutResolutionSeen == nil {
		stat.timeoutResolutionSeen = map[string]struct{}{}
	}
	if stat.timeoutClampSeen == nil {
		stat.timeoutClampSeen = map[string]struct{}{}
	}
	if _, ok := stat.timeoutResolutionSeen[key]; !ok {
		stat.timeoutResolutionSeen[key] = struct{}{}
		stat.EffectiveOperationProfile = strings.TrimSpace(meta.EffectiveOperationProfile)
		stat.TimeoutResolutionSource = strings.TrimSpace(meta.Source)
		stat.TimeoutResolutionTrace = strings.TrimSpace(meta.Trace)
	}
	if meta.ParentBudgetClamped {
		if _, ok := stat.timeoutClampSeen[key]; !ok {
			stat.timeoutClampSeen[key] = struct{}{}
			stat.TimeoutParentBudgetClamp++
		}
	}
}

func (c *Composer) addCollabOutcome(task scheduler.Task, failed bool) {
	if c == nil {
		return
	}
	cfg := c.effectiveConfig()
	if !cfg.Composer.Collab.Enabled {
		return
	}
	runID := strings.TrimSpace(task.RunID)
	if runID == "" {
		return
	}
	c.runMu.Lock()
	defer c.runMu.Unlock()
	stat := c.ensureRunStat(runID)
	strategy := strings.TrimSpace(cfg.Composer.Collab.DefaultAggregation)
	if strategy == "" {
		strategy = runtimeconfig.ComposerCollabAggregationAllSettled
	}
	stat.CollabAggregationStrategy = strategy
	primitive := collabPrimitiveFromTask(task)
	switch primitive {
	case "handoff":
		stat.CollabHandoffTotal++
		stat.CollabAggregationTotal++
	case "delegation":
		stat.CollabDelegationTotal++
		stat.CollabAggregationTotal++
	case "aggregation":
		stat.CollabAggregationTotal++
	default:
		if strings.TrimSpace(task.PeerID) != "" {
			stat.CollabDelegationTotal++
			stat.CollabAggregationTotal++
		}
	}
	if failed && strings.EqualFold(strings.TrimSpace(cfg.Composer.Collab.FailurePolicy), runtimeconfig.ComposerCollabFailurePolicyFailFast) {
		stat.CollabFailFastTotal++
	}
}

func (c *Composer) rebuildCollabStatsFromSchedulerSnapshot(runID string, snapshot scheduler.StoreSnapshot) {
	if c == nil {
		return
	}
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return
	}
	cfg := c.effectiveConfig()
	if !cfg.Composer.Collab.Enabled {
		return
	}
	strategy := strings.TrimSpace(cfg.Composer.Collab.DefaultAggregation)
	if strategy == "" {
		strategy = runtimeconfig.ComposerCollabAggregationAllSettled
	}
	failFast := strings.EqualFold(strings.TrimSpace(cfg.Composer.Collab.FailurePolicy), runtimeconfig.ComposerCollabFailurePolicyFailFast)

	handoffTotal := 0
	delegationTotal := 0
	aggregationTotal := 0
	failFastTotal := 0
	for i := range snapshot.Tasks {
		record := snapshot.Tasks[i]
		task := record.Task
		if strings.TrimSpace(task.RunID) != runID {
			continue
		}
		failed := record.State == scheduler.TaskStateFailed || record.State == scheduler.TaskStateDeadLetter
		succeeded := record.State == scheduler.TaskStateSucceeded
		if !failed && !succeeded {
			continue
		}
		switch collabPrimitiveFromTask(task) {
		case "handoff":
			handoffTotal++
			aggregationTotal++
		case "delegation":
			delegationTotal++
			aggregationTotal++
		case "aggregation":
			aggregationTotal++
		default:
			if strings.TrimSpace(task.PeerID) != "" {
				delegationTotal++
				aggregationTotal++
			}
		}
		if failed && failFast {
			failFastTotal++
		}
	}

	c.runMu.Lock()
	defer c.runMu.Unlock()
	stat := c.ensureRunStat(runID)
	stat.CollabHandoffTotal = handoffTotal
	stat.CollabDelegationTotal = delegationTotal
	stat.CollabAggregationTotal = aggregationTotal
	stat.CollabAggregationStrategy = strategy
	stat.CollabFailFastTotal = failFastTotal
}

func collabPrimitiveFromTask(task scheduler.Task) string {
	if len(task.Payload) == 0 {
		return ""
	}
	raw, ok := task.Payload["collab_primitive"]
	if !ok {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(value))
}

func (c *Composer) addAsyncReportOutcome(runID string, report a2a.AsyncReport, duplicate bool) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return
	}
	reportKey := strings.TrimSpace(report.ReportKey)
	if reportKey == "" {
		reportKey = a2a.BuildAsyncReportKey(report)
	}
	c.runMu.Lock()
	defer c.runMu.Unlock()
	stat := c.ensureRunStat(runID)
	if stat.asyncReportSeen == nil {
		stat.asyncReportSeen = map[string]struct{}{}
	}
	if stat.asyncReportDedupSeen == nil {
		stat.asyncReportDedupSeen = map[string]struct{}{}
	}
	if stat.asyncLateReportSeen == nil {
		stat.asyncLateReportSeen = map[string]struct{}{}
	}
	if stat.asyncAwaitSeen == nil {
		stat.asyncAwaitSeen = map[string]struct{}{}
	}
	if duplicate {
		if _, ok := stat.asyncReportDedupSeen[reportKey]; !ok {
			stat.asyncReportDedupSeen[reportKey] = struct{}{}
			stat.A2AAsyncReportDedup++
			stat.AsyncReportDedupTotal++
		}
		return
	}
	if _, exists := stat.asyncReportSeen[reportKey]; exists {
		return
	}
	stat.asyncReportSeen[reportKey] = struct{}{}
	stat.A2AAsyncReportTotal++
	if report.Status == a2a.StatusFailed || report.Status == a2a.StatusCanceled {
		stat.A2AAsyncReportFailed++
	}
	if report.DeliveryAttempt > 1 {
		stat.A2AAsyncReportRetry += report.DeliveryAttempt - 1
	}
}

func (c *Composer) addAsyncLateReportOutcome(runID string, report a2a.AsyncReport, duplicate bool) bool {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return false
	}
	reportKey := strings.TrimSpace(report.ReportKey)
	if reportKey == "" {
		reportKey = a2a.BuildAsyncReportKey(report)
	}
	c.runMu.Lock()
	defer c.runMu.Unlock()
	stat := c.ensureRunStat(runID)
	if stat.asyncLateReportSeen == nil {
		stat.asyncLateReportSeen = map[string]struct{}{}
	}
	if stat.asyncReportDedupSeen == nil {
		stat.asyncReportDedupSeen = map[string]struct{}{}
	}
	firstLate := false
	if _, ok := stat.asyncLateReportSeen[reportKey]; !ok {
		stat.asyncLateReportSeen[reportKey] = struct{}{}
		stat.AsyncLateReportTotal++
		firstLate = true
	}
	if duplicate {
		if _, ok := stat.asyncReportDedupSeen[reportKey]; !ok {
			stat.asyncReportDedupSeen[reportKey] = struct{}{}
			stat.A2AAsyncReportDedup++
			stat.AsyncReportDedupTotal++
		}
	}
	return firstLate
}

func (c *Composer) addAsyncAwait(runID, taskID, attemptID string) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return
	}
	key := strings.TrimSpace(taskID) + "|" + strings.TrimSpace(attemptID)
	if key == "|" {
		return
	}
	c.runMu.Lock()
	defer c.runMu.Unlock()
	stat := c.ensureRunStat(runID)
	if stat.asyncAwaitSeen == nil {
		stat.asyncAwaitSeen = map[string]struct{}{}
	}
	if _, ok := stat.asyncAwaitSeen[key]; ok {
		return
	}
	stat.asyncAwaitSeen[key] = struct{}{}
	stat.AsyncAwaitTotal++
}

func (c *Composer) emitAsyncReportDedupTimeline(ctx context.Context, claimed scheduler.ClaimedTask, report a2a.AsyncReport) {
	if c == nil || c.handler == nil {
		return
	}
	payload := map[string]any{
		"phase":         string(types.ActionPhaseRun),
		"status":        string(mapAsyncReportStatus(report.Status)),
		"reason":        a2a.ReasonAsyncReportDedup,
		"sequence":      c.now().UnixNano(),
		"task_id":       strings.TrimSpace(claimed.Record.Task.TaskID),
		"attempt_id":    strings.TrimSpace(claimed.Attempt.AttemptID),
		"agent_id":      strings.TrimSpace(claimed.Record.Task.AgentID),
		"peer_id":       strings.TrimSpace(claimed.Record.Task.PeerID),
		"workflow_id":   strings.TrimSpace(claimed.Record.Task.WorkflowID),
		"team_id":       strings.TrimSpace(claimed.Record.Task.TeamID),
		"step_id":       strings.TrimSpace(claimed.Record.Task.StepID),
		"report_key":    strings.TrimSpace(report.ReportKey),
		"outcome_key":   strings.TrimSpace(report.OutcomeKey),
		"delivery_mode": a2a.AsyncReportSinkCallback,
	}
	c.handler.OnEvent(ctx, types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    types.EventTypeActionTimeline,
		RunID:   strings.TrimSpace(claimed.Record.Task.RunID),
		Time:    c.now(),
		Payload: payload,
	})
}

func (c *Composer) emitAsyncLateReportTimeline(ctx context.Context, claimed scheduler.ClaimedTask, report a2a.AsyncReport) {
	if c == nil || c.handler == nil {
		return
	}
	payload := map[string]any{
		"phase":              string(types.ActionPhaseRun),
		"status":             string(mapAsyncReportStatus(report.Status)),
		"reason":             scheduler.ReasonAsyncLateReport,
		"sequence":           c.now().UnixNano(),
		"task_id":            strings.TrimSpace(claimed.Record.Task.TaskID),
		"attempt_id":         strings.TrimSpace(claimed.Attempt.AttemptID),
		"agent_id":           strings.TrimSpace(claimed.Record.Task.AgentID),
		"peer_id":            strings.TrimSpace(claimed.Record.Task.PeerID),
		"workflow_id":        strings.TrimSpace(claimed.Record.Task.WorkflowID),
		"team_id":            strings.TrimSpace(claimed.Record.Task.TeamID),
		"step_id":            strings.TrimSpace(claimed.Record.Task.StepID),
		"report_key":         strings.TrimSpace(report.ReportKey),
		"outcome_key":        strings.TrimSpace(report.OutcomeKey),
		"delivery_mode":      a2a.AsyncReportSinkCallback,
		"late_report_policy": runtimeconfig.AsyncLateReportPolicyDropAndRecord,
	}
	c.handler.OnEvent(ctx, types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    types.EventTypeActionTimeline,
		RunID:   strings.TrimSpace(claimed.Record.Task.RunID),
		Time:    c.now(),
		Payload: payload,
	})
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
		RecoveryResumeBoundary: strings.TrimSpace(c.recoveryResumeBoundary),
		RecoveryInflightPolicy: strings.TrimSpace(c.recoveryInflightPolicy),
		RecoveryFallback:       c.recoveryFallback,
		RecoveryFallbackReason: strings.TrimSpace(c.recoveryFallbackReason),
		asyncReportSeen:        map[string]struct{}{},
		asyncReportDedupSeen:   map[string]struct{}{},
		asyncLateReportSeen:    map[string]struct{}{},
		asyncAwaitSeen:         map[string]struct{}{},
		timeoutResolutionSeen:  map[string]struct{}{},
		timeoutClampSeen:       map[string]struct{}{},
	}
	c.runStat[runID] = stat
	return stat
}

func mapAsyncReportStatus(status a2a.TaskStatus) types.ActionStatus {
	switch status {
	case a2a.StatusSubmitted:
		return types.ActionStatusPending
	case a2a.StatusRunning:
		return types.ActionStatusRunning
	case a2a.StatusSucceeded:
		return types.ActionStatusSucceeded
	case a2a.StatusCanceled:
		return types.ActionStatusCanceled
	default:
		return types.ActionStatusFailed
	}
}

func (c *Composer) reconfigureAsyncAwaitReconcileWorker(cfg runtimeconfig.Config) {
	if c == nil {
		return
	}

	c.reconcileMu.Lock()
	defer c.reconcileMu.Unlock()
	c.stopAsyncAwaitReconcileWorkerLocked()

	if !cfg.Scheduler.AsyncAwait.Reconcile.Enabled {
		return
	}
	if c.a2aClient == nil {
		return
	}
	pollClient, ok := c.a2aClient.(scheduler.A2AReconcilePollClient)
	if !ok {
		return
	}
	s := c.Scheduler()
	if s == nil {
		return
	}

	loopCtx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	c.reconcileCancel = cancel
	c.reconcileDone = done
	go c.runAsyncAwaitReconcileLoop(loopCtx, done, s, pollClient)
}

func (c *Composer) stopAsyncAwaitReconcileWorkerLocked() {
	cancel := c.reconcileCancel
	done := c.reconcileDone
	c.reconcileCancel = nil
	c.reconcileDone = nil
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

func (c *Composer) runAsyncAwaitReconcileLoop(
	ctx context.Context,
	done chan struct{},
	s *scheduler.Scheduler,
	pollClient scheduler.A2AReconcilePollClient,
) {
	defer close(done)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		_, _ = s.ReconcileAwaitingReports(ctx, pollClient)
		delay := s.NextAsyncReconcileDelay()
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
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
		scheduler.WithTaskBoardControl(c.schedulerTaskBoardControlConfig(cfg)),
		scheduler.WithAsyncAwait(c.schedulerAsyncAwaitConfig(cfg)),
		scheduler.WithRecoveryBoundary(c.schedulerRecoveryBoundaryConfig(cfg)),
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
	c.reconfigureAsyncAwaitReconcileWorker(cfg)
	c.publishRuntimeReadinessSnapshot()
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
		scheduler.WithTaskBoardControl(c.schedulerTaskBoardControlConfig(cfg)),
		scheduler.WithAsyncAwait(c.schedulerAsyncAwaitConfig(cfg)),
		scheduler.WithRecoveryBoundary(c.schedulerRecoveryBoundaryConfig(cfg)),
	)
	if err != nil {
		return
	}

	c.schedulerMu.Lock()
	c.scheduler = updated
	c.schedulerConfiguredBackend = strings.TrimSpace(strings.ToLower(cfg.Scheduler.Backend))
	c.schedulerBackend = strings.TrimSpace(strings.ToLower(cfg.Scheduler.Backend))
	c.schedulerQueueLimit = cfg.Scheduler.QueueLimit
	c.schedulerRetryMaxAttempts = cfg.Scheduler.RetryMaxAttempts
	c.schedulerGuardrails = guardrails
	c.schedulerSignature = signature
	c.schedulerMu.Unlock()
	c.reconfigureAsyncAwaitReconcileWorker(cfg)
	c.publishRuntimeReadinessSnapshot()
}

func (c *Composer) schedulerConfigSignature(cfg runtimeconfig.Config) string {
	return fmt.Sprintf(
		"%d|%d|%d|%d|%d|%d|%s|%d|%t|%d|%t|%t|%d|%d|%.4f|%.4f|%d|%s|%s|%t|%d|%d|%.4f|%s|%t|%s|%s|%s|%d",
		cfg.Scheduler.LeaseTimeout.Milliseconds(),
		cfg.Subagent.MaxDepth,
		cfg.Subagent.MaxActiveChildren,
		cfg.Subagent.ChildTimeoutBudget.Milliseconds(),
		cfg.Scheduler.QueueLimit,
		cfg.Scheduler.RetryMaxAttempts,
		strings.TrimSpace(strings.ToLower(cfg.Scheduler.QoS.Mode)),
		cfg.Scheduler.QoS.Fairness.MaxConsecutiveClaimsPerPriority,
		cfg.Scheduler.TaskBoard.Control.Enabled,
		cfg.Scheduler.TaskBoard.Control.MaxManualRetryPerTask,
		cfg.Scheduler.DLQ.Enabled,
		cfg.Scheduler.Retry.Backoff.Enabled,
		cfg.Scheduler.Retry.Backoff.Initial.Milliseconds(),
		cfg.Scheduler.Retry.Backoff.Max.Milliseconds(),
		cfg.Scheduler.Retry.Backoff.Multiplier,
		cfg.Scheduler.Retry.Backoff.JitterRatio,
		cfg.Scheduler.AsyncAwait.ReportTimeout.Milliseconds(),
		strings.TrimSpace(strings.ToLower(cfg.Scheduler.AsyncAwait.LateReportPolicy)),
		cfg.Scheduler.AsyncAwait.TimeoutTerminal,
		cfg.Scheduler.AsyncAwait.Reconcile.Enabled,
		cfg.Scheduler.AsyncAwait.Reconcile.Interval.Milliseconds(),
		cfg.Scheduler.AsyncAwait.Reconcile.BatchSize,
		cfg.Scheduler.AsyncAwait.Reconcile.JitterRatio,
		strings.TrimSpace(strings.ToLower(cfg.Scheduler.AsyncAwait.Reconcile.NotFoundPolicy)),
		cfg.Recovery.Enabled,
		strings.TrimSpace(strings.ToLower(cfg.Recovery.ResumeBoundary)),
		strings.TrimSpace(strings.ToLower(cfg.Recovery.InflightPolicy)),
		strings.TrimSpace(strings.ToLower(cfg.Recovery.TimeoutReentryPolicy)),
		cfg.Recovery.TimeoutReentryMaxPerTask,
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

func (c *Composer) schedulerTaskBoardControlConfig(cfg runtimeconfig.Config) scheduler.TaskBoardControlConfig {
	return scheduler.TaskBoardControlConfig{
		Enabled:               cfg.Scheduler.TaskBoard.Control.Enabled,
		MaxManualRetryPerTask: cfg.Scheduler.TaskBoard.Control.MaxManualRetryPerTask,
	}
}

func (c *Composer) schedulerRecoveryBoundaryConfig(cfg runtimeconfig.Config) scheduler.RecoveryBoundaryConfig {
	return scheduler.RecoveryBoundaryConfig{
		Enabled:                  cfg.Recovery.Enabled,
		ResumeBoundary:           strings.TrimSpace(strings.ToLower(cfg.Recovery.ResumeBoundary)),
		InflightPolicy:           strings.TrimSpace(strings.ToLower(cfg.Recovery.InflightPolicy)),
		TimeoutReentryPolicy:     strings.TrimSpace(strings.ToLower(cfg.Recovery.TimeoutReentryPolicy)),
		TimeoutReentryMaxPerTask: cfg.Recovery.TimeoutReentryMaxPerTask,
	}
}

func (c *Composer) schedulerAsyncAwaitConfig(cfg runtimeconfig.Config) scheduler.AsyncAwaitConfig {
	return scheduler.AsyncAwaitConfig{
		ReportTimeout:    cfg.Scheduler.AsyncAwait.ReportTimeout,
		LateReportPolicy: strings.TrimSpace(strings.ToLower(cfg.Scheduler.AsyncAwait.LateReportPolicy)),
		TimeoutTerminal:  scheduler.TaskState(strings.TrimSpace(strings.ToLower(cfg.Scheduler.AsyncAwait.TimeoutTerminal))),
		Reconcile: scheduler.AsyncAwaitReconcileConfig{
			Enabled:        cfg.Scheduler.AsyncAwait.Reconcile.Enabled,
			Interval:       cfg.Scheduler.AsyncAwait.Reconcile.Interval,
			BatchSize:      cfg.Scheduler.AsyncAwait.Reconcile.BatchSize,
			JitterRatio:    cfg.Scheduler.AsyncAwait.Reconcile.JitterRatio,
			NotFoundPolicy: strings.TrimSpace(strings.ToLower(cfg.Scheduler.AsyncAwait.Reconcile.NotFoundPolicy)),
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

func cloneIntMap(in map[string]int) map[string]int {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]int, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func cloneRuntimePolicyCandidates(in []runtimeconfig.RuntimePolicyCandidate) []runtimeconfig.RuntimePolicyCandidate {
	if len(in) == 0 {
		return nil
	}
	out := make([]runtimeconfig.RuntimePolicyCandidate, len(in))
	copy(out, in)
	return out
}
