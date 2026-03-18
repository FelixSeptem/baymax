package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	"gopkg.in/yaml.v3"
)

const (
	ReasonSchedule = "workflow.schedule"
	ReasonRetry    = "workflow.retry"
	ReasonResume   = "workflow.resume"
)

type StepStatus string

const (
	StepStatusPending   StepStatus = "pending"
	StepStatusRunning   StepStatus = "running"
	StepStatusSucceeded StepStatus = "succeeded"
	StepStatusFailed    StepStatus = "failed"
	StepStatusSkipped   StepStatus = "skipped"
	StepStatusCanceled  StepStatus = "canceled"
)

type StepKind string

const (
	StepKindRunner StepKind = "runner"
	StepKindTool   StepKind = "tool"
	StepKindMCP    StepKind = "mcp"
	StepKindSkill  StepKind = "skill"
)

type StepCondition string

const (
	ConditionAlways    StepCondition = "always"
	ConditionOnSuccess StepCondition = "on_success"
	ConditionOnFailure StepCondition = "on_failure"
)

type ValidationErrorCode string

const (
	ErrCodeWorkflowIDRequired      ValidationErrorCode = "workflow_id_required"
	ErrCodeStepIDRequired          ValidationErrorCode = "step_id_required"
	ErrCodeDuplicateStepID         ValidationErrorCode = "duplicate_step_id"
	ErrCodeMissingDependency       ValidationErrorCode = "missing_dependency"
	ErrCodeDependencyCycle         ValidationErrorCode = "dependency_cycle"
	ErrCodeUnsupportedCondition    ValidationErrorCode = "unsupported_condition"
	ErrCodeUnsupportedStepKind     ValidationErrorCode = "unsupported_step_kind"
	ErrCodeInvalidRetryMaxAttempts ValidationErrorCode = "invalid_retry_max_attempts"
	ErrCodeInvalidRetryBackoff     ValidationErrorCode = "invalid_retry_backoff"
	ErrCodeInvalidStepTimeout      ValidationErrorCode = "invalid_step_timeout"
	ErrCodeNoSteps                 ValidationErrorCode = "no_steps"
)

type ValidationError struct {
	Code    ValidationErrorCode `json:"code"`
	StepID  string              `json:"step_id,omitempty"`
	Field   string              `json:"field,omitempty"`
	Message string              `json:"message"`
}

func (e ValidationError) Error() string {
	if strings.TrimSpace(e.StepID) == "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("%s[%s]: %s", e.Code, e.StepID, e.Message)
}

type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	if len(e) == 1 {
		return e[0].Error()
	}
	return fmt.Sprintf("%s (and %d more)", e[0].Error(), len(e)-1)
}

type Retry struct {
	MaxAttempts int           `json:"max_attempts,omitempty" yaml:"max_attempts,omitempty"`
	Backoff     time.Duration `json:"backoff,omitempty" yaml:"backoff,omitempty"`
}

type Step struct {
	StepID    string         `json:"step_id,omitempty" yaml:"step_id,omitempty"`
	Step      string         `json:"step,omitempty" yaml:"step,omitempty"`
	TaskID    string         `json:"task_id,omitempty" yaml:"task_id,omitempty"`
	Kind      StepKind       `json:"kind,omitempty" yaml:"kind,omitempty"`
	DependsOn []string       `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`
	Condition StepCondition  `json:"condition,omitempty" yaml:"condition,omitempty"`
	Retry     Retry          `json:"retry,omitempty" yaml:"retry,omitempty"`
	Timeout   time.Duration  `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Payload   map[string]any `json:"payload,omitempty" yaml:"payload,omitempty"`
}

type Definition struct {
	WorkflowID string `json:"workflow_id" yaml:"workflow_id"`
	Steps      []Step `json:"steps" yaml:"steps"`
}

type Planner interface {
	Plan(def Definition) ([]string, error)
}

type CheckpointStore interface {
	Load(ctx context.Context, workflowID string) (Checkpoint, bool, error)
	Save(ctx context.Context, cp Checkpoint) error
}

type StepAdapter interface {
	Execute(ctx context.Context, workflowID string, step Step, attempt int) (StepOutput, error)
}

type DispatchAdapter struct {
	Runner func(ctx context.Context, workflowID string, step Step, attempt int) (StepOutput, error)
	Tool   func(ctx context.Context, workflowID string, step Step, attempt int) (StepOutput, error)
	MCP    func(ctx context.Context, workflowID string, step Step, attempt int) (StepOutput, error)
	Skill  func(ctx context.Context, workflowID string, step Step, attempt int) (StepOutput, error)
}

func (a DispatchAdapter) Execute(ctx context.Context, workflowID string, step Step, attempt int) (StepOutput, error) {
	switch normalizeStepKind(step.Kind) {
	case StepKindRunner:
		if a.Runner == nil {
			return StepOutput{}, errors.New("workflow runner adapter is missing")
		}
		return a.Runner(ctx, workflowID, step, attempt)
	case StepKindTool:
		if a.Tool == nil {
			return StepOutput{}, errors.New("workflow tool adapter is missing")
		}
		return a.Tool(ctx, workflowID, step, attempt)
	case StepKindMCP:
		if a.MCP == nil {
			return StepOutput{}, errors.New("workflow mcp adapter is missing")
		}
		return a.MCP(ctx, workflowID, step, attempt)
	case StepKindSkill:
		if a.Skill == nil {
			return StepOutput{}, errors.New("workflow skill adapter is missing")
		}
		return a.Skill(ctx, workflowID, step, attempt)
	default:
		return StepOutput{}, fmt.Errorf("unsupported workflow step kind %q", step.Kind)
	}
}

type StepOutput struct {
	Payload map[string]any `json:"payload,omitempty"`
}

type CheckpointStep struct {
	Status   StepStatus `json:"status"`
	Attempts int        `json:"attempts"`
}

type Checkpoint struct {
	WorkflowID  string                    `json:"workflow_id"`
	RunID       string                    `json:"run_id,omitempty"`
	Status      string                    `json:"workflow_status,omitempty"`
	ResumeCount int                       `json:"workflow_resume_count,omitempty"`
	Steps       map[string]CheckpointStep `json:"steps,omitempty"`
	UpdatedAt   time.Time                 `json:"updated_at"`
}

type MemoryCheckpointStore struct {
	mu   sync.Mutex
	data map[string]Checkpoint
}

func NewMemoryCheckpointStore() *MemoryCheckpointStore {
	return &MemoryCheckpointStore{data: map[string]Checkpoint{}}
}

func (s *MemoryCheckpointStore) Load(_ context.Context, workflowID string) (Checkpoint, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp, ok := s.data[strings.TrimSpace(workflowID)]
	if !ok {
		return Checkpoint{}, false, nil
	}
	return cp, true, nil
}

func (s *MemoryCheckpointStore) Save(_ context.Context, cp Checkpoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := strings.TrimSpace(cp.WorkflowID)
	if key == "" {
		return errors.New("workflow checkpoint requires workflow_id")
	}
	s.data[key] = cp
	return nil
}

type FileCheckpointStore struct {
	root string
}

func NewFileCheckpointStore(root string) (*FileCheckpointStore, error) {
	path := strings.TrimSpace(root)
	if path == "" {
		return nil, errors.New("workflow checkpoint root is required")
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir workflow checkpoint root: %w", err)
	}
	return &FileCheckpointStore{root: path}, nil
}

func (s *FileCheckpointStore) Load(_ context.Context, workflowID string) (Checkpoint, bool, error) {
	path := s.filePath(workflowID)
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Checkpoint{}, false, nil
	}
	if err != nil {
		return Checkpoint{}, false, fmt.Errorf("read workflow checkpoint: %w", err)
	}
	var cp Checkpoint
	if err := json.Unmarshal(raw, &cp); err != nil {
		return Checkpoint{}, false, fmt.Errorf("decode workflow checkpoint: %w", err)
	}
	return cp, true, nil
}

func (s *FileCheckpointStore) Save(_ context.Context, cp Checkpoint) error {
	if strings.TrimSpace(cp.WorkflowID) == "" {
		return errors.New("workflow checkpoint requires workflow_id")
	}
	if err := os.MkdirAll(s.root, 0o755); err != nil {
		return fmt.Errorf("mkdir workflow checkpoint root: %w", err)
	}
	raw, err := json.Marshal(cp)
	if err != nil {
		return fmt.Errorf("encode workflow checkpoint: %w", err)
	}
	if err := os.WriteFile(s.filePath(cp.WorkflowID), raw, 0o600); err != nil {
		return fmt.Errorf("write workflow checkpoint: %w", err)
	}
	return nil
}

func (s *FileCheckpointStore) filePath(workflowID string) string {
	clean := strings.ToLower(strings.TrimSpace(workflowID))
	clean = strings.ReplaceAll(clean, " ", "_")
	clean = strings.ReplaceAll(clean, "/", "_")
	clean = strings.ReplaceAll(clean, "\\", "_")
	return filepath.Join(s.root, clean+".json")
}

type RunRequest struct {
	RunID  string     `json:"run_id,omitempty"`
	Resume bool       `json:"resume,omitempty"`
	DSL    Definition `json:"dsl"`
}

type StepResult struct {
	StepID   string     `json:"step_id"`
	TaskID   string     `json:"task_id,omitempty"`
	Status   StepStatus `json:"status"`
	Attempts int        `json:"step_attempt"`
	Reason   string     `json:"reason,omitempty"`
	Error    string     `json:"error,omitempty"`
	Output   StepOutput `json:"output,omitempty"`
}

type RunResult struct {
	RunID               string       `json:"run_id,omitempty"`
	WorkflowID          string       `json:"workflow_id"`
	WorkflowStatus      string       `json:"workflow_status"`
	WorkflowStepTotal   int          `json:"workflow_step_total"`
	WorkflowStepFailed  int          `json:"workflow_step_failed"`
	WorkflowResumeCount int          `json:"workflow_resume_count"`
	Steps               []StepResult `json:"steps"`
	ExecutionOrder      []string     `json:"execution_order,omitempty"`
}

func (r RunResult) RunFinishedPayload() map[string]any {
	return map[string]any{
		"workflow_id":           r.WorkflowID,
		"workflow_status":       r.WorkflowStatus,
		"workflow_step_total":   r.WorkflowStepTotal,
		"workflow_step_failed":  r.WorkflowStepFailed,
		"workflow_resume_count": r.WorkflowResumeCount,
	}
}

type StreamEvent struct {
	Kind       string      `json:"kind"`
	Step       *StepResult `json:"step,omitempty"`
	Checkpoint *Checkpoint `json:"checkpoint,omitempty"`
	Result     *RunResult  `json:"result,omitempty"`
}

type Option func(*Engine)

func WithStepAdapter(adapter StepAdapter) Option {
	return func(e *Engine) { e.adapter = adapter }
}

func WithCheckpointStore(store CheckpointStore) Option {
	return func(e *Engine) { e.checkpoints = store }
}

func WithTimelineEmitter(handler types.EventHandler) Option {
	return func(e *Engine) { e.timelineEmitter = handler }
}

func WithDefaultStepTimeout(timeout time.Duration) Option {
	return func(e *Engine) { e.defaultStepTimeout = timeout }
}

type Engine struct {
	adapter            StepAdapter
	checkpoints        CheckpointStore
	timelineEmitter    types.EventHandler
	defaultStepTimeout time.Duration
	now                func() time.Time
}

func New(opts ...Option) *Engine {
	e := &Engine{
		adapter:            DispatchAdapter{},
		checkpoints:        NewMemoryCheckpointStore(),
		defaultStepTimeout: 3 * time.Second,
		now:                time.Now,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(e)
		}
	}
	return e
}

func ParseDefinition(raw []byte) (Definition, error) {
	var def Definition
	if err := json.Unmarshal(raw, &def); err == nil {
		return normalizeDefinition(def), nil
	}
	if err := yaml.Unmarshal(raw, &def); err != nil {
		return Definition{}, fmt.Errorf("parse workflow dsl: %w", err)
	}
	return normalizeDefinition(def), nil
}

func normalizeDefinition(def Definition) Definition {
	def.WorkflowID = strings.TrimSpace(def.WorkflowID)
	for i := range def.Steps {
		if strings.TrimSpace(def.Steps[i].StepID) == "" {
			def.Steps[i].StepID = strings.TrimSpace(def.Steps[i].Step)
		}
		def.Steps[i].StepID = strings.TrimSpace(def.Steps[i].StepID)
		def.Steps[i].TaskID = strings.TrimSpace(def.Steps[i].TaskID)
		if def.Steps[i].TaskID == "" {
			def.Steps[i].TaskID = def.Steps[i].StepID
		}
		def.Steps[i].Kind = normalizeStepKind(def.Steps[i].Kind)
		def.Steps[i].Condition = normalizeCondition(def.Steps[i].Condition)
		def.Steps[i].DependsOn = normalizeDependsOn(def.Steps[i].DependsOn)
	}
	return def
}

func normalizeStepKind(kind StepKind) StepKind {
	k := strings.ToLower(strings.TrimSpace(string(kind)))
	if k == "" {
		return StepKindRunner
	}
	return StepKind(k)
}

func normalizeCondition(condition StepCondition) StepCondition {
	c := strings.ToLower(strings.TrimSpace(string(condition)))
	if c == "" {
		return ConditionAlways
	}
	return StepCondition(c)
}

func normalizeDependsOn(items []string) []string {
	out := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		key := strings.TrimSpace(item)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out
}

func ValidateDefinition(def Definition) ValidationErrors {
	violations := make([]ValidationError, 0)
	if strings.TrimSpace(def.WorkflowID) == "" {
		violations = append(violations, ValidationError{
			Code:    ErrCodeWorkflowIDRequired,
			Field:   "workflow_id",
			Message: "workflow_id is required",
		})
	}
	if len(def.Steps) == 0 {
		violations = append(violations, ValidationError{
			Code:    ErrCodeNoSteps,
			Field:   "steps",
			Message: "steps must not be empty",
		})
		return violations
	}

	stepsByID := map[string]Step{}
	for i := range def.Steps {
		step := def.Steps[i]
		if strings.TrimSpace(step.StepID) == "" {
			violations = append(violations, ValidationError{
				Code:    ErrCodeStepIDRequired,
				Field:   "steps.step",
				Message: "step id is required",
			})
			continue
		}
		if _, ok := stepsByID[step.StepID]; ok {
			violations = append(violations, ValidationError{
				Code:    ErrCodeDuplicateStepID,
				StepID:  step.StepID,
				Field:   "steps.step",
				Message: "duplicate step id",
			})
			continue
		}
		stepsByID[step.StepID] = step

		switch step.Kind {
		case StepKindRunner, StepKindTool, StepKindMCP, StepKindSkill:
		default:
			violations = append(violations, ValidationError{
				Code:    ErrCodeUnsupportedStepKind,
				StepID:  step.StepID,
				Field:   "steps.kind",
				Message: fmt.Sprintf("unsupported step kind %q", step.Kind),
			})
		}
		switch step.Condition {
		case ConditionAlways, ConditionOnSuccess, ConditionOnFailure:
		default:
			violations = append(violations, ValidationError{
				Code:    ErrCodeUnsupportedCondition,
				StepID:  step.StepID,
				Field:   "steps.condition",
				Message: fmt.Sprintf("unsupported condition %q", step.Condition),
			})
		}
		if step.Retry.MaxAttempts < 0 {
			violations = append(violations, ValidationError{
				Code:    ErrCodeInvalidRetryMaxAttempts,
				StepID:  step.StepID,
				Field:   "steps.retry.max_attempts",
				Message: "retry max_attempts must be >= 0",
			})
		}
		if step.Retry.Backoff < 0 {
			violations = append(violations, ValidationError{
				Code:    ErrCodeInvalidRetryBackoff,
				StepID:  step.StepID,
				Field:   "steps.retry.backoff",
				Message: "retry backoff must be >= 0",
			})
		}
		if step.Timeout < 0 {
			violations = append(violations, ValidationError{
				Code:    ErrCodeInvalidStepTimeout,
				StepID:  step.StepID,
				Field:   "steps.timeout",
				Message: "timeout must be >= 0",
			})
		}
	}

	for _, step := range def.Steps {
		if strings.TrimSpace(step.StepID) == "" {
			continue
		}
		for _, dep := range step.DependsOn {
			if dep == step.StepID {
				violations = append(violations, ValidationError{
					Code:    ErrCodeDependencyCycle,
					StepID:  step.StepID,
					Field:   "steps.depends_on",
					Message: "step cannot depend on itself",
				})
				continue
			}
			if _, ok := stepsByID[dep]; !ok {
				violations = append(violations, ValidationError{
					Code:    ErrCodeMissingDependency,
					StepID:  step.StepID,
					Field:   "steps.depends_on",
					Message: fmt.Sprintf("missing dependency %q", dep),
				})
			}
		}
	}

	if len(violations) > 0 {
		return violations
	}
	if hasCycle(def.Steps) {
		return ValidationErrors{
			{
				Code:    ErrCodeDependencyCycle,
				Field:   "steps.depends_on",
				Message: "workflow dependency graph contains a cycle",
			},
		}
	}
	return nil
}

func hasCycle(steps []Step) bool {
	adj := map[string][]string{}
	for _, step := range steps {
		adj[step.StepID] = append([]string(nil), step.DependsOn...)
	}
	visited := map[string]int{}
	var dfs func(string) bool
	dfs = func(id string) bool {
		switch visited[id] {
		case 1:
			return true
		case 2:
			return false
		}
		visited[id] = 1
		for _, dep := range adj[id] {
			if dfs(dep) {
				return true
			}
		}
		visited[id] = 2
		return false
	}
	keys := make([]string, 0, len(adj))
	for k := range adj {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, id := range keys {
		if dfs(id) {
			return true
		}
	}
	return false
}

func (e *Engine) Plan(def Definition) ([]string, error) {
	def = normalizeDefinition(def)
	if errs := ValidateDefinition(def); len(errs) > 0 {
		return nil, errs
	}
	stepsByID := map[string]Step{}
	inDegree := map[string]int{}
	edges := map[string][]string{}
	for _, step := range def.Steps {
		stepsByID[step.StepID] = step
		inDegree[step.StepID] = 0
	}
	for _, step := range def.Steps {
		for _, dep := range step.DependsOn {
			edges[dep] = append(edges[dep], step.StepID)
			inDegree[step.StepID]++
		}
	}
	ready := make([]string, 0)
	for id, deg := range inDegree {
		if deg == 0 {
			ready = append(ready, id)
		}
	}
	sort.Strings(ready)

	order := make([]string, 0, len(def.Steps))
	for len(ready) > 0 {
		id := ready[0]
		ready = ready[1:]
		order = append(order, id)
		next := append([]string(nil), edges[id]...)
		sort.Strings(next)
		for _, to := range next {
			inDegree[to]--
			if inDegree[to] == 0 {
				ready = append(ready, to)
			}
		}
		sort.Strings(ready)
	}
	if len(order) != len(def.Steps) {
		return nil, ValidationErrors{{Code: ErrCodeDependencyCycle, Message: "workflow dependency graph contains a cycle"}}
	}
	return order, nil
}

func (e *Engine) Run(ctx context.Context, req RunRequest) (RunResult, error) {
	return e.execute(ctx, req, nil)
}

func (e *Engine) Stream(ctx context.Context, req RunRequest, onEvent func(StreamEvent) error) (RunResult, error) {
	return e.execute(ctx, req, onEvent)
}

func (e *Engine) execute(ctx context.Context, req RunRequest, onEvent func(StreamEvent) error) (RunResult, error) {
	def := normalizeDefinition(req.DSL)
	if errs := ValidateDefinition(def); len(errs) > 0 {
		return RunResult{}, errs
	}

	stepsByID := map[string]Step{}
	for _, step := range def.Steps {
		stepsByID[step.StepID] = step
	}
	results := map[string]*StepResult{}
	for _, step := range def.Steps {
		item := &StepResult{
			StepID: step.StepID,
			TaskID: step.TaskID,
			Status: StepStatusPending,
		}
		results[step.StepID] = item
	}

	seq := int64(0)
	resumeCount := 0
	if req.Resume && e.checkpoints != nil {
		cp, ok, err := e.checkpoints.Load(ctx, def.WorkflowID)
		if err != nil {
			return RunResult{}, err
		}
		if ok {
			resumeCount = cp.ResumeCount + 1
			for stepID, state := range cp.Steps {
				current, ok := results[stepID]
				if !ok {
					continue
				}
				switch state.Status {
				case StepStatusSucceeded, StepStatusSkipped:
					current.Status = state.Status
					current.Attempts = state.Attempts
					e.emitTimeline(ctx, req.RunID, def.WorkflowID, stepID, state.Status, ReasonResume, &seq)
					if onEvent != nil {
						snap := *current
						if err := onEvent(StreamEvent{Kind: "workflow.resumed.step", Step: &snap}); err != nil {
							return RunResult{}, err
						}
					}
				default:
					// failed/canceled/pending steps remain eligible to continue on resume.
					current.Status = StepStatusPending
					current.Attempts = 0
				}
			}
		}
	}

	executionOrder := make([]string, 0, len(def.Steps))
	for {
		ready := readySteps(def.Steps, results)
		if len(ready) == 0 {
			break
		}
		for _, step := range ready {
			record := results[step.StepID]
			if record.Status != StepStatusPending {
				continue
			}
			executionOrder = append(executionOrder, step.StepID)

			if !conditionMatched(step, results) {
				record.Status = StepStatusSkipped
				record.Reason = "condition.not_matched"
				e.emitTimeline(ctx, req.RunID, def.WorkflowID, step.StepID, record.Status, ReasonSchedule, &seq)
				if onEvent != nil {
					snap := *record
					if err := onEvent(StreamEvent{Kind: "workflow.step", Step: &snap}); err != nil {
						return RunResult{}, err
					}
				}
				if err := e.saveCheckpoint(ctx, req.RunID, def.WorkflowID, "running", resumeCount, results); err != nil {
					return RunResult{}, err
				}
				continue
			}

			maxAttempts := step.Retry.MaxAttempts + 1
			if maxAttempts <= 0 {
				maxAttempts = 1
			}
			for attempt := 1; attempt <= maxAttempts; attempt++ {
				record.Attempts = attempt
				record.Status = StepStatusRunning
				e.emitTimeline(ctx, req.RunID, def.WorkflowID, step.StepID, StepStatusRunning, ReasonSchedule, &seq)

				timeout := step.Timeout
				if timeout <= 0 {
					timeout = e.defaultStepTimeout
				}
				if timeout <= 0 {
					timeout = 3 * time.Second
				}
				stepCtx, cancel := context.WithTimeout(ctx, timeout)
				output, err := e.adapter.Execute(stepCtx, def.WorkflowID, step, attempt)
				cancel()
				if err == nil {
					record.Status = StepStatusSucceeded
					record.Reason = ""
					record.Error = ""
					record.Output = output
					e.emitTimeline(ctx, req.RunID, def.WorkflowID, step.StepID, StepStatusSucceeded, ReasonSchedule, &seq)
					break
				}

				record.Output = StepOutput{}
				record.Error = err.Error()
				if errors.Is(ctx.Err(), context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
					record.Status = StepStatusCanceled
					record.Reason = "cancel.propagated"
					e.emitTimeline(ctx, req.RunID, def.WorkflowID, step.StepID, StepStatusCanceled, ReasonSchedule, &seq)
					break
				}
				if errors.Is(err, context.DeadlineExceeded) {
					record.Status = StepStatusFailed
					record.Reason = "step.timeout"
				} else {
					record.Status = StepStatusFailed
					record.Reason = "step.error"
				}
				if attempt < maxAttempts {
					e.emitTimeline(ctx, req.RunID, def.WorkflowID, step.StepID, StepStatusPending, ReasonRetry, &seq)
					if step.Retry.Backoff > 0 {
						timer := time.NewTimer(step.Retry.Backoff)
						select {
						case <-ctx.Done():
							timer.Stop()
							record.Status = StepStatusCanceled
							record.Reason = "cancel.propagated"
						case <-timer.C:
						}
					}
					continue
				}
				e.emitTimeline(ctx, req.RunID, def.WorkflowID, step.StepID, record.Status, ReasonSchedule, &seq)
			}

			if onEvent != nil {
				snap := *record
				if err := onEvent(StreamEvent{Kind: "workflow.step", Step: &snap}); err != nil {
					return RunResult{}, err
				}
			}
			if err := e.saveCheckpoint(ctx, req.RunID, def.WorkflowID, "running", resumeCount, results); err != nil {
				return RunResult{}, err
			}
		}
	}

	out := buildResult(req.RunID, def.WorkflowID, resumeCount, def.Steps, results, executionOrder)
	if err := e.saveCheckpoint(ctx, req.RunID, def.WorkflowID, out.WorkflowStatus, resumeCount, results); err != nil {
		return RunResult{}, err
	}
	if onEvent != nil {
		if err := onEvent(StreamEvent{
			Kind:   "workflow.completed",
			Result: &out,
		}); err != nil {
			return RunResult{}, err
		}
	}
	if errors.Is(ctx.Err(), context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return out, ctx.Err()
	}
	return out, nil
}

func readySteps(steps []Step, results map[string]*StepResult) []Step {
	ready := make([]Step, 0)
	for _, step := range steps {
		rec, ok := results[step.StepID]
		if !ok || rec.Status != StepStatusPending {
			continue
		}
		allDepsTerminal := true
		for _, dep := range step.DependsOn {
			depRec, ok := results[dep]
			if !ok || !isTerminal(depRec.Status) {
				allDepsTerminal = false
				break
			}
		}
		if allDepsTerminal {
			ready = append(ready, step)
		}
	}
	sort.Slice(ready, func(i, j int) bool { return ready[i].StepID < ready[j].StepID })
	return ready
}

func conditionMatched(step Step, results map[string]*StepResult) bool {
	switch step.Condition {
	case ConditionOnSuccess:
		for _, dep := range step.DependsOn {
			depRec := results[dep]
			if depRec == nil || depRec.Status != StepStatusSucceeded {
				return false
			}
		}
		return true
	case ConditionOnFailure:
		if len(step.DependsOn) == 0 {
			return false
		}
		for _, dep := range step.DependsOn {
			depRec := results[dep]
			if depRec == nil {
				continue
			}
			if depRec.Status == StepStatusFailed || depRec.Status == StepStatusCanceled {
				return true
			}
		}
		return false
	default:
		return true
	}
}

func isTerminal(status StepStatus) bool {
	switch status {
	case StepStatusSucceeded, StepStatusFailed, StepStatusSkipped, StepStatusCanceled:
		return true
	default:
		return false
	}
}

func buildResult(runID, workflowID string, resumeCount int, steps []Step, results map[string]*StepResult, executionOrder []string) RunResult {
	out := RunResult{
		RunID:               strings.TrimSpace(runID),
		WorkflowID:          workflowID,
		WorkflowResumeCount: resumeCount,
		WorkflowStepTotal:   len(steps),
		ExecutionOrder:      append([]string(nil), executionOrder...),
	}
	keys := make([]string, 0, len(results))
	for key := range results {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		item := results[key]
		if item == nil {
			continue
		}
		out.Steps = append(out.Steps, *item)
		if item.Status == StepStatusFailed {
			out.WorkflowStepFailed++
		}
	}
	out.WorkflowStatus = "succeeded"
	for _, step := range out.Steps {
		if step.Status == StepStatusFailed || step.Status == StepStatusCanceled {
			out.WorkflowStatus = "failed"
			break
		}
	}
	return out
}

func (e *Engine) saveCheckpoint(
	ctx context.Context,
	runID, workflowID, status string,
	resumeCount int,
	results map[string]*StepResult,
) error {
	if e.checkpoints == nil {
		return nil
	}
	cp := Checkpoint{
		WorkflowID:  workflowID,
		RunID:       strings.TrimSpace(runID),
		Status:      strings.TrimSpace(status),
		ResumeCount: resumeCount,
		UpdatedAt:   e.now(),
		Steps:       map[string]CheckpointStep{},
	}
	for stepID, result := range results {
		if result == nil {
			continue
		}
		cp.Steps[stepID] = CheckpointStep{
			Status:   result.Status,
			Attempts: result.Attempts,
		}
	}
	return e.checkpoints.Save(ctx, cp)
}

func (e *Engine) emitTimeline(
	ctx context.Context,
	runID, workflowID, stepID string,
	status StepStatus,
	reason string,
	seq *int64,
) {
	if e == nil || e.timelineEmitter == nil || seq == nil {
		return
	}
	*seq = *seq + 1
	payload := map[string]any{
		"phase":       string(types.ActionPhaseRun),
		"status":      string(status),
		"sequence":    *seq,
		"reason":      reason,
		"workflow_id": workflowID,
	}
	if strings.TrimSpace(stepID) != "" {
		payload["step_id"] = stepID
		payload["task_id"] = stepID
	}
	e.timelineEmitter.OnEvent(ctx, types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    types.EventTypeActionTimeline,
		RunID:   strings.TrimSpace(runID),
		Time:    e.now(),
		Payload: payload,
	})
}
