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
	ReasonEnqueue      = "scheduler.enqueue"
	ReasonClaim        = "scheduler.claim"
	ReasonHeartbeat    = "scheduler.heartbeat"
	ReasonLeaseExpired = "scheduler.lease_expired"
	ReasonRequeue      = "scheduler.requeue"
	ReasonSpawn        = "subagent.spawn"
	ReasonJoin         = "subagent.join"
	ReasonBudgetReject = "subagent.budget_reject"
)

var canonicalReasonSet = map[string]struct{}{
	ReasonEnqueue:      {},
	ReasonClaim:        {},
	ReasonHeartbeat:    {},
	ReasonLeaseExpired: {},
	ReasonRequeue:      {},
	ReasonSpawn:        {},
	ReasonJoin:         {},
	ReasonBudgetReject: {},
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
	TaskStateQueued    TaskState = "queued"
	TaskStateRunning   TaskState = "running"
	TaskStateSucceeded TaskState = "succeeded"
	TaskStateFailed    TaskState = "failed"
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
	Payload     map[string]any `json:"payload,omitempty"`
	MaxAttempts int            `json:"max_attempts,omitempty"`
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
	if out.TaskID == "" {
		return Task{}, errors.New("task_id is required")
	}
	if out.MaxAttempts <= 0 {
		out.MaxAttempts = 3
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
	Task           Task             `json:"task"`
	State          TaskState        `json:"state"`
	Attempts       []Attempt        `json:"attempts,omitempty"`
	CurrentAttempt string           `json:"current_attempt_id,omitempty"`
	Result         map[string]any   `json:"result,omitempty"`
	ErrorMessage   string           `json:"error_message,omitempty"`
	ErrorClass     types.ErrorClass `json:"error_class,omitempty"`
	ErrorLayer     string           `json:"error_layer,omitempty"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
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
	Record  TaskRecord `json:"record"`
	Attempt Attempt    `json:"attempt"`
}

type TerminalCommit struct {
	TaskID       string           `json:"task_id"`
	AttemptID    string           `json:"attempt_id"`
	Status       TaskState        `json:"status"`
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
	Record    TaskRecord `json:"record"`
	Duplicate bool       `json:"duplicate"`
}

type Stats struct {
	Backend                      string `json:"backend"`
	QueueTotal                   int    `json:"queue_total"`
	ClaimTotal                   int    `json:"claim_total"`
	ReclaimTotal                 int    `json:"reclaim_total"`
	LeaseExpiredTotal            int    `json:"lease_expired_total"`
	CompleteTotal                int    `json:"complete_total"`
	FailTotal                    int    `json:"fail_total"`
	DuplicateTerminalCommitTotal int    `json:"duplicate_terminal_commit_total"`
}

type QueueStore interface {
	Backend() string
	Enqueue(ctx context.Context, task Task, now time.Time) (TaskRecord, error)
	Claim(ctx context.Context, workerID string, now time.Time, leaseTimeout time.Duration) (ClaimedTask, bool, error)
	Heartbeat(ctx context.Context, taskID, attemptID, leaseToken string, now time.Time, leaseTimeout time.Duration) (ClaimedTask, error)
	ExpireLeases(ctx context.Context, now time.Time) ([]ClaimedTask, error)
	Requeue(ctx context.Context, taskID, reason string, now time.Time) (TaskRecord, error)
	CommitTerminal(ctx context.Context, commit TerminalCommit) (CommitResult, error)
	Get(ctx context.Context, taskID string) (TaskRecord, bool, error)
	Stats(ctx context.Context) (Stats, error)
}

var (
	ErrTaskNotFound       = errors.New("scheduler task not found")
	ErrAttemptNotFound    = errors.New("scheduler attempt not found")
	ErrLeaseTokenMismatch = errors.New("scheduler lease token mismatch")
	ErrLeaseExpired       = errors.New("scheduler lease expired")
	ErrTaskNotClaimable   = errors.New("scheduler task is not claimable")
	ErrTaskNotRunning     = errors.New("scheduler task is not running")
	ErrStaleAttempt       = errors.New("scheduler stale attempt commit")
)

type Guardrails struct {
	MaxDepth           int           `json:"max_depth"`
	MaxActiveChildren  int           `json:"max_active_children"`
	ChildTimeoutBudget time.Duration `json:"child_timeout_budget"`
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
