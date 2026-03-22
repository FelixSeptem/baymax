package scheduler

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

const (
	ReasonEnqueue         = "scheduler.enqueue"
	ReasonDelayedEnqueue  = "scheduler.delayed_enqueue"
	ReasonDelayedWait     = "scheduler.delayed_wait"
	ReasonDelayedReady    = "scheduler.delayed_ready"
	ReasonClaim           = "scheduler.claim"
	ReasonHeartbeat       = "scheduler.heartbeat"
	ReasonLeaseExpired    = "scheduler.lease_expired"
	ReasonAwaitingReport  = "scheduler.awaiting_report"
	ReasonAsyncTimeout    = "scheduler.async_timeout"
	ReasonAsyncLateReport = "scheduler.async_late_report"
	ReasonAsyncReconcile  = "scheduler.async_reconcile"
	ReasonRequeue         = "scheduler.requeue"
	ReasonQoSClaim        = "scheduler.qos_claim"
	ReasonFairnessYield   = "scheduler.fairness_yield"
	ReasonRetryBackoff    = "scheduler.retry_backoff"
	ReasonDeadLetter      = "scheduler.dead_letter"
	ReasonSpawn           = "subagent.spawn"
	ReasonJoin            = "subagent.join"
	ReasonBudgetReject    = "subagent.budget_reject"
)

var canonicalReasonSet = map[string]struct{}{
	ReasonEnqueue:         {},
	ReasonDelayedEnqueue:  {},
	ReasonDelayedWait:     {},
	ReasonDelayedReady:    {},
	ReasonClaim:           {},
	ReasonHeartbeat:       {},
	ReasonLeaseExpired:    {},
	ReasonAwaitingReport:  {},
	ReasonAsyncTimeout:    {},
	ReasonAsyncLateReport: {},
	ReasonAsyncReconcile:  {},
	ReasonRequeue:         {},
	ReasonQoSClaim:        {},
	ReasonFairnessYield:   {},
	ReasonRetryBackoff:    {},
	ReasonDeadLetter:      {},
	ReasonSpawn:           {},
	ReasonJoin:            {},
	ReasonBudgetReject:    {},
}

func CanonicalReason(reason string) (string, bool) {
	normalized := strings.TrimSpace(reason)
	_, ok := canonicalReasonSet[normalized]
	if !ok {
		return "", false
	}
	return normalized, true
}

type TaskState string

const (
	TaskStateQueued         TaskState = "queued"
	TaskStateRunning        TaskState = "running"
	TaskStateAwaitingReport TaskState = "awaiting_report"
	TaskStateSucceeded      TaskState = "succeeded"
	TaskStateFailed         TaskState = "failed"
	TaskStateDeadLetter     TaskState = "dead_letter"
)

const (
	TaskPriorityHigh   = "high"
	TaskPriorityNormal = "normal"
	TaskPriorityLow    = "low"
)

type AttemptStatus string

const (
	AttemptStatusRunning   AttemptStatus = "running"
	AttemptStatusSucceeded AttemptStatus = "succeeded"
	AttemptStatusFailed    AttemptStatus = "failed"
	AttemptStatusExpired   AttemptStatus = "expired"
)

type Task struct {
	TaskID      string         `json:"task_id"`
	RunID       string         `json:"run_id,omitempty"`
	WorkflowID  string         `json:"workflow_id,omitempty"`
	TeamID      string         `json:"team_id,omitempty"`
	StepID      string         `json:"step_id,omitempty"`
	AgentID     string         `json:"agent_id,omitempty"`
	PeerID      string         `json:"peer_id,omitempty"`
	ParentRunID string         `json:"parent_run_id,omitempty"`
	Priority    string         `json:"priority,omitempty"`
	Payload     map[string]any `json:"payload,omitempty"`
	MaxAttempts int            `json:"max_attempts,omitempty"`
	NotBefore   time.Time      `json:"not_before,omitempty"`
}

func normalizeTask(in Task) (Task, error) {
	out := in
	out.TaskID = strings.TrimSpace(out.TaskID)
	out.RunID = strings.TrimSpace(out.RunID)
	out.WorkflowID = strings.TrimSpace(out.WorkflowID)
	out.TeamID = strings.TrimSpace(out.TeamID)
	out.StepID = strings.TrimSpace(out.StepID)
	out.AgentID = strings.TrimSpace(out.AgentID)
	out.PeerID = strings.TrimSpace(out.PeerID)
	out.ParentRunID = strings.TrimSpace(out.ParentRunID)
	out.Priority = strings.ToLower(strings.TrimSpace(out.Priority))
	if out.TaskID == "" {
		return Task{}, errors.New("task_id is required")
	}
	if out.Priority == "" {
		if payloadPriority, ok := out.Payload["priority"].(string); ok {
			out.Priority = strings.ToLower(strings.TrimSpace(payloadPriority))
		}
	}
	switch out.Priority {
	case TaskPriorityHigh, TaskPriorityNormal, TaskPriorityLow:
	case "":
		out.Priority = TaskPriorityNormal
	default:
		return Task{}, fmt.Errorf("priority must be one of [%s,%s,%s], got %q", TaskPriorityHigh, TaskPriorityNormal, TaskPriorityLow, out.Priority)
	}
	if out.MaxAttempts <= 0 {
		out.MaxAttempts = 3
	}
	if !out.NotBefore.IsZero() {
		out.NotBefore = out.NotBefore.UTC()
	}
	out.Payload = copyMap(out.Payload)
	return out, nil
}

type Attempt struct {
	AttemptID      string        `json:"attempt_id"`
	Attempt        int           `json:"attempt"`
	WorkerID       string        `json:"worker_id"`
	LeaseToken     string        `json:"lease_token"`
	Status         AttemptStatus `json:"status"`
	StartedAt      time.Time     `json:"started_at"`
	HeartbeatAt    time.Time     `json:"heartbeat_at"`
	LeaseExpiresAt time.Time     `json:"lease_expires_at"`
	TerminalAt     time.Time     `json:"terminal_at,omitempty"`
}

type TaskRecord struct {
	Task                Task             `json:"task"`
	State               TaskState        `json:"state"`
	Attempts            []Attempt        `json:"attempts,omitempty"`
	CurrentAttempt      string           `json:"current_attempt_id,omitempty"`
	AwaitingReportSince time.Time        `json:"awaiting_report_since,omitempty"`
	ReportTimeoutAt     time.Time        `json:"report_timeout_at,omitempty"`
	RemoteTaskID        string           `json:"remote_task_id,omitempty"`
	ResolutionSource    string           `json:"resolution_source,omitempty"`
	TerminalConflict    bool             `json:"terminal_conflict_recorded,omitempty"`
	NextEligibleAt      time.Time        `json:"next_eligible_at,omitempty"`
	DeadLetterCode      string           `json:"dead_letter_code,omitempty"`
	Result              map[string]any   `json:"result,omitempty"`
	ErrorMessage        string           `json:"error_message,omitempty"`
	ErrorClass          types.ErrorClass `json:"error_class,omitempty"`
	ErrorLayer          string           `json:"error_layer,omitempty"`
	CreatedAt           time.Time        `json:"created_at"`
	UpdatedAt           time.Time        `json:"updated_at"`
}

func (r TaskRecord) attemptByID(attemptID string) (Attempt, bool) {
	trimmed := strings.TrimSpace(attemptID)
	if trimmed == "" {
		return Attempt{}, false
	}
	for _, at := range r.Attempts {
		if strings.TrimSpace(at.AttemptID) == trimmed {
			return at, true
		}
	}
	return Attempt{}, false
}

func (r TaskRecord) currentAttempt() (Attempt, bool) {
	return r.attemptByID(r.CurrentAttempt)
}

type ClaimedTask struct {
	Record          TaskRecord `json:"record"`
	Attempt         Attempt    `json:"attempt"`
	TaskPriority    string     `json:"task_priority,omitempty"`
	FairnessYielded bool       `json:"fairness_yielded,omitempty"`
}

type TerminalCommit struct {
	TaskID       string           `json:"task_id"`
	AttemptID    string           `json:"attempt_id"`
	Status       TaskState        `json:"status"`
	Source       string           `json:"source,omitempty"`
	RemoteTaskID string           `json:"remote_task_id,omitempty"`
	Result       map[string]any   `json:"result,omitempty"`
	ErrorMessage string           `json:"error_message,omitempty"`
	ErrorClass   types.ErrorClass `json:"error_class,omitempty"`
	ErrorLayer   string           `json:"error_layer,omitempty"`
	OutcomeKey   string           `json:"outcome_key,omitempty"`
	CommittedAt  time.Time        `json:"committed_at"`
}

func normalizeCommit(in TerminalCommit) (TerminalCommit, error) {
	out := in
	out.TaskID = strings.TrimSpace(out.TaskID)
	out.AttemptID = strings.TrimSpace(out.AttemptID)
	out.Source = strings.ToLower(strings.TrimSpace(out.Source))
	out.RemoteTaskID = strings.TrimSpace(out.RemoteTaskID)
	out.ErrorMessage = strings.TrimSpace(out.ErrorMessage)
	out.ErrorLayer = strings.TrimSpace(out.ErrorLayer)
	out.OutcomeKey = strings.TrimSpace(out.OutcomeKey)
	if out.TaskID == "" {
		return TerminalCommit{}, errors.New("task_id is required")
	}
	if out.AttemptID == "" {
		return TerminalCommit{}, errors.New("attempt_id is required")
	}
	switch out.Status {
	case TaskStateSucceeded, TaskStateFailed:
	default:
		return TerminalCommit{}, fmt.Errorf("terminal status must be succeeded|failed, got %q", out.Status)
	}
	if out.CommittedAt.IsZero() {
		out.CommittedAt = time.Now()
	}
	if out.Status == TaskStateSucceeded {
		out.Result = copyMap(out.Result)
	}
	if out.OutcomeKey == "" {
		out.OutcomeKey = defaultOutcomeKey(out)
	}
	return out, nil
}

func defaultOutcomeKey(commit TerminalCommit) string {
	if commit.Status == TaskStateSucceeded {
		keys := make([]string, 0, len(commit.Result))
		for key := range commit.Result {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		builder := strings.Builder{}
		builder.WriteString(string(commit.Status))
		for _, key := range keys {
			builder.WriteString("|")
			builder.WriteString(key)
		}
		return builder.String()
	}
	if commit.ErrorMessage == "" {
		return string(commit.Status)
	}
	return string(commit.Status) + "|" + commit.ErrorMessage
}

type CommitResult struct {
	Record     TaskRecord `json:"record"`
	Duplicate  bool       `json:"duplicate"`
	LateReport bool       `json:"late_report,omitempty"`
	Conflict   bool       `json:"conflict,omitempty"`
}

type Stats struct {
	Backend                              string `json:"backend"`
	QoSMode                              string `json:"qos_mode,omitempty"`
	QueueTotal                           int    `json:"queue_total"`
	ClaimTotal                           int    `json:"claim_total"`
	ReclaimTotal                         int    `json:"reclaim_total"`
	LeaseExpiredTotal                    int    `json:"lease_expired_total"`
	CompleteTotal                        int    `json:"complete_total"`
	FailTotal                            int    `json:"fail_total"`
	PriorityClaimTotal                   int    `json:"priority_claim_total,omitempty"`
	FairnessYieldTotal                   int    `json:"fairness_yield_total,omitempty"`
	RetryBackoffTotal                    int    `json:"retry_backoff_total,omitempty"`
	DeadLetterTotal                      int    `json:"dead_letter_total,omitempty"`
	DelayedTaskTotal                     int    `json:"delayed_task_total,omitempty"`
	DelayedClaimTotal                    int    `json:"delayed_claim_total,omitempty"`
	DelayedWaitMsP95                     int64  `json:"delayed_wait_ms_p95,omitempty"`
	RecoveryTimeoutReentryTotal          int    `json:"recovery_timeout_reentry_total,omitempty"`
	RecoveryTimeoutReentryExhaustedTotal int    `json:"recovery_timeout_reentry_exhausted_total,omitempty"`
	DuplicateTerminalCommitTotal         int    `json:"duplicate_terminal_commit_total"`
	AsyncAwaitTotal                      int    `json:"async_await_total,omitempty"`
	AsyncTimeoutTotal                    int    `json:"async_timeout_total,omitempty"`
	AsyncReconcilePollTotal              int    `json:"async_reconcile_poll_total,omitempty"`
	AsyncReconcileTerminalByPollTotal    int    `json:"async_reconcile_terminal_by_poll_total,omitempty"`
	AsyncReconcileErrorTotal             int    `json:"async_reconcile_error_total,omitempty"`
	AsyncTerminalConflictTotal           int    `json:"async_terminal_conflict_total,omitempty"`
}

const (
	AsyncLateReportPolicyDropAndRecord = "drop_and_record"
	AsyncResolutionSourceCallback      = "callback"
	AsyncResolutionSourceReconcilePoll = "reconcile_poll"
	AsyncResolutionSourceTimeout       = "timeout"
	AsyncReconcileNotFoundKeepTimeout  = "keep_until_timeout"
)

type AsyncAwaitReconcileConfig struct {
	Enabled        bool          `json:"enabled"`
	Interval       time.Duration `json:"interval"`
	BatchSize      int           `json:"batch_size"`
	JitterRatio    float64       `json:"jitter_ratio"`
	NotFoundPolicy string        `json:"not_found_policy,omitempty"`
}

type AsyncAwaitConfig struct {
	ReportTimeout    time.Duration             `json:"report_timeout"`
	LateReportPolicy string                    `json:"late_report_policy,omitempty"`
	TimeoutTerminal  TaskState                 `json:"timeout_terminal,omitempty"`
	Reconcile        AsyncAwaitReconcileConfig `json:"reconcile"`
}

type ReconcilePollClassification string

const (
	ReconcilePollClassificationPending         ReconcilePollClassification = "pending"
	ReconcilePollClassificationTerminal        ReconcilePollClassification = "terminal"
	ReconcilePollClassificationNotFound        ReconcilePollClassification = "not_found"
	ReconcilePollClassificationRetryableError  ReconcilePollClassification = "retryable_error"
	ReconcilePollClassificationNonRetryableErr ReconcilePollClassification = "non_retryable_error"
)

type ReconcilePollResult struct {
	Classification ReconcilePollClassification `json:"classification"`
	Commit         TerminalCommit              `json:"commit,omitempty"`
}

type ReconcileCycleStats struct {
	PollTotal          int `json:"poll_total,omitempty"`
	TerminalByPoll     int `json:"terminal_by_poll,omitempty"`
	ErrorTotal         int `json:"error_total,omitempty"`
	ConflictTotalDelta int `json:"conflict_total_delta,omitempty"`
}

const (
	RecoveryResumeBoundaryNextAttemptOnly         = "next_attempt_only"
	RecoveryInflightPolicyNoRewind                = "no_rewind"
	RecoveryTimeoutReentryPolicySingleReentryFail = "single_reentry_then_fail"
)

type RecoveryBoundaryConfig struct {
	Enabled                  bool   `json:"enabled"`
	ResumeBoundary           string `json:"resume_boundary,omitempty"`
	InflightPolicy           string `json:"inflight_policy,omitempty"`
	TimeoutReentryPolicy     string `json:"timeout_reentry_policy,omitempty"`
	TimeoutReentryMaxPerTask int    `json:"timeout_reentry_max_per_task,omitempty"`
}

type QueueStore interface {
	Backend() string
	Enqueue(ctx context.Context, task Task, now time.Time) (TaskRecord, error)
	Claim(ctx context.Context, workerID string, now time.Time, leaseTimeout time.Duration) (ClaimedTask, bool, error)
	Heartbeat(ctx context.Context, taskID, attemptID, leaseToken string, now time.Time, leaseTimeout time.Duration) (ClaimedTask, error)
	ExpireLeases(ctx context.Context, now time.Time) ([]ClaimedTask, error)
	ExpireAwaitingReports(ctx context.Context, now time.Time) ([]ClaimedTask, error)
	MarkAwaitingReport(ctx context.Context, taskID, attemptID, remoteTaskID string, now time.Time, reportTimeout time.Duration) (TaskRecord, error)
	ListAwaitingReport(ctx context.Context, now time.Time, limit int) ([]TaskRecord, error)
	RecordAsyncReconcileStats(ctx context.Context, pollTotal, errorTotal int) error
	Requeue(ctx context.Context, taskID, reason string, now time.Time) (TaskRecord, error)
	CommitTerminal(ctx context.Context, commit TerminalCommit) (CommitResult, error)
	CommitAsyncReportTerminal(ctx context.Context, commit TerminalCommit) (CommitResult, error)
	Get(ctx context.Context, taskID string) (TaskRecord, bool, error)
	Stats(ctx context.Context) (Stats, error)
}

var (
	ErrTaskNotFound          = errors.New("scheduler task not found")
	ErrAttemptNotFound       = errors.New("scheduler attempt not found")
	ErrLeaseTokenMismatch    = errors.New("scheduler lease token mismatch")
	ErrLeaseExpired          = errors.New("scheduler lease expired")
	ErrTaskNotClaimable      = errors.New("scheduler task is not claimable")
	ErrTaskNotRunning        = errors.New("scheduler task is not running")
	ErrTaskNotAwaitingReport = errors.New("scheduler task is not awaiting report")
	ErrStaleAttempt          = errors.New("scheduler stale attempt commit")
	ErrSnapshotCorrupt       = errors.New("scheduler snapshot is corrupt")
)

type StoreSnapshot struct {
	Backend         string           `json:"backend"`
	Tasks           []TaskRecord     `json:"tasks,omitempty"`
	Queue           []string         `json:"queue,omitempty"`
	TerminalCommits []TerminalCommit `json:"terminal_commits,omitempty"`
	DelayedWaitMs   []int64          `json:"delayed_wait_ms,omitempty"`
	Stats           Stats            `json:"stats"`
}

type Guardrails struct {
	MaxDepth           int           `json:"max_depth"`
	MaxActiveChildren  int           `json:"max_active_children"`
	ChildTimeoutBudget time.Duration `json:"child_timeout_budget"`
}

type QoSMode string

const (
	QoSModeFIFO     QoSMode = "fifo"
	QoSModePriority QoSMode = "priority"
)

type FairnessConfig struct {
	MaxConsecutiveClaimsPerPriority int `json:"max_consecutive_claims_per_priority"`
}

type DLQConfig struct {
	Enabled bool `json:"enabled"`
}

type RetryBackoffConfig struct {
	Enabled     bool          `json:"enabled"`
	Initial     time.Duration `json:"initial"`
	Max         time.Duration `json:"max"`
	Multiplier  float64       `json:"multiplier"`
	JitterRatio float64       `json:"jitter_ratio"`
}

type GovernanceConfig struct {
	QoS      QoSMode            `json:"qos_mode"`
	Fairness FairnessConfig     `json:"fairness"`
	DLQ      DLQConfig          `json:"dlq"`
	Backoff  RetryBackoffConfig `json:"backoff"`
}

type SpawnRequest struct {
	Task                 Task          `json:"task"`
	ParentDepth          int           `json:"parent_depth"`
	ParentActiveChildren int           `json:"parent_active_children"`
	ChildTimeout         time.Duration `json:"child_timeout"`
}

type BudgetRejectCode string

const (
	BudgetRejectDepth       BudgetRejectCode = "max_depth_exceeded"
	BudgetRejectActiveChild BudgetRejectCode = "max_active_children_exceeded"
	BudgetRejectTimeout     BudgetRejectCode = "child_timeout_budget_exceeded"
)

type BudgetError struct {
	Code    BudgetRejectCode
	Message string
}

func (e *BudgetError) Error() string {
	if e == nil {
		return ""
	}
	return strings.TrimSpace(e.Message)
}

func (g Guardrails) ValidateSpawn(req SpawnRequest) error {
	if g.MaxDepth > 0 && req.ParentDepth+1 > g.MaxDepth {
		return &BudgetError{
			Code:    BudgetRejectDepth,
			Message: fmt.Sprintf("subagent depth %d exceeds max_depth %d", req.ParentDepth+1, g.MaxDepth),
		}
	}
	if g.MaxActiveChildren > 0 && req.ParentActiveChildren >= g.MaxActiveChildren {
		return &BudgetError{
			Code:    BudgetRejectActiveChild,
			Message: fmt.Sprintf("active children %d exceeds max_active_children %d", req.ParentActiveChildren, g.MaxActiveChildren),
		}
	}
	if g.ChildTimeoutBudget > 0 && req.ChildTimeout > g.ChildTimeoutBudget {
		return &BudgetError{
			Code:    BudgetRejectTimeout,
			Message: fmt.Sprintf("child timeout %s exceeds child_timeout_budget %s", req.ChildTimeout, g.ChildTimeoutBudget),
		}
	}
	return nil
}

func copyMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
