package workflow

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
)

const (
	WorkflowContinuationVersion    = "workflow_continuation.v1"
	WorkflowContinuationMaxBytes   = 64 * 1024
	WorkflowContinuationMaxEntries = 1024
)

var (
	ErrWorkflowPaused                       = errors.New("workflow paused")
	ErrWorkflowContinuationMalformed        = errors.New("workflow continuation malformed")
	ErrWorkflowContinuationCorrupt          = errors.New("workflow continuation corrupt")
	ErrWorkflowContinuationForeign          = errors.New("workflow continuation foreign")
	ErrWorkflowContinuationExpired          = errors.New("workflow continuation expired")
	ErrWorkflowContinuationUnavailable      = errors.New("workflow continuation unavailable")
	ErrWorkflowContinuationConflict         = errors.New("workflow continuation conflict")
	ErrWorkflowContinuationInProgress       = errors.New("workflow continuation in progress")
	ErrWorkflowContinuationRecoveryRequired = errors.New("workflow continuation recovery required")
	ErrWorkflowContinuationUnsupported      = errors.New("workflow continuation unsupported")
)

type PauseError struct {
	Reason                 WorkflowPauseReason
	PendingActionReference string
	ConsumedBudget         int64
	OriginalDeadline       time.Time
}

func (e PauseError) Error() string { return string(e.Reason) + ": " + ErrWorkflowPaused.Error() }
func (e PauseError) Unwrap() error { return ErrWorkflowPaused }

// WorkflowPauseBoundary is emitted by the workflow source owner before a
// dispatch boundary. Controllers may request a durable source pause; they do
// not execute or authorize the pending effect.
type WorkflowPauseBoundary struct {
	WorkflowID  string
	RunID       string
	SessionID   string
	ParentRunID string
	CausationID string
	Step        Step
	Attempt     int
}

type WorkflowPauseController interface {
	BeforeDispatch(context.Context, WorkflowPauseBoundary) error
}

type WorkflowPauseControllerFunc func(context.Context, WorkflowPauseBoundary) error

func (f WorkflowPauseControllerFunc) BeforeDispatch(ctx context.Context, boundary WorkflowPauseBoundary) error {
	if f == nil {
		return nil
	}
	return f(ctx, boundary)
}

func WithWorkflowContinuationStore(store WorkflowContinuationStore, namespace string) Option {
	return func(e *Engine) { e.continuations = store; e.continuationNamespace = strings.TrimSpace(namespace) }
}

func WithPauseController(controller WorkflowPauseController) Option {
	return func(e *Engine) { e.pauseController = controller }
}

type WorkflowPauseReason string

const (
	PauseInputRequired    WorkflowPauseReason = "input_required"
	PauseApprovalRequired WorkflowPauseReason = "approval_required"
	PauseBudgetExhausted  WorkflowPauseReason = "budget_exhausted"
)

func (e *Engine) ResumeContinuation(ctx context.Context, request WorkflowResumeRequest, onEvent func(StreamEvent) error) (RunResult, error) {
	if e == nil || e.continuations == nil {
		return RunResult{}, ErrWorkflowContinuationUnavailable
	}
	continuation, err := e.continuations.Load(ctx, request.Ref)
	if err != nil {
		return RunResult{}, err
	}
	if request.GraphDigest != continuation.GraphDigest || request.Generation != continuation.Generation {
		return RunResult{}, ErrWorkflowContinuationConflict
	}
	if request.AuthorityDigest != continuation.AuthorityDigest || request.PlanDigest != continuation.PlanDigest || request.WriteScopeDigest != continuation.WriteScopeDigest || request.EffectNamespace != continuation.EffectNamespace || request.OccurrenceID != continuation.OccurrenceID {
		return RunResult{}, ErrWorkflowContinuationForeign
	}
	if continuation.PauseReason == PauseBudgetExhausted && !continuation.OriginalDeadline.IsZero() && !e.now().Before(continuation.OriginalDeadline) {
		return RunResult{}, ErrWorkflowContinuationExpired
	}
	claim, err := e.continuations.ClaimResume(ctx, request)
	if err != nil {
		return RunResult{}, err
	}
	if claim.Duplicate && claim.Result != nil {
		return *claim.Result, nil
	}
	claim, err = e.continuations.MarkResumeRunning(ctx, claim)
	if err != nil {
		return RunResult{}, err
	}
	result, err := e.executeFromContinuation(ctx, continuation, onEvent)
	if err != nil && !errors.Is(err, ErrWorkflowPaused) && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		return result, err
	}
	if _, commitErr := e.continuations.CommitResumeResult(context.WithoutCancel(ctx), claim, result); commitErr != nil {
		return RunResult{}, commitErr
	}
	return result, err
}

func (e *Engine) restoreContinuationSteps(
	ctx context.Context,
	req RunRequest,
	results map[string]*StepResult,
	stepTimeline map[string]workflowStepTimelineContext,
	stepsByID map[string]Step,
	timelineCtx workflowTimelineContext,
	seq *int64,
	onEvent func(StreamEvent) error,
) error {
	for stepID, state := range req.continuationSteps {
		current, ok := results[stepID]
		if !ok {
			continue
		}
		switch state.Status {
		case StepStatusSucceeded, StepStatusSkipped:
			current.Status = state.Status
			current.Attempts = state.Attempts
			meta, ok := stepTimeline[stepID]
			if !ok {
				meta = newWorkflowStepTimelineContext(stepsByID[stepID])
			}
			e.emitTimeline(ctx, timelineCtx, meta, state.Status, ReasonResume, seq)
			if onEvent != nil {
				snap := *current
				if err := onEvent(StreamEvent{Kind: "workflow.resumed.step", Step: &snap}); err != nil {
					return err
				}
			}
		default:
			current.Status = StepStatusPending
			current.Attempts = 0
		}
	}
	return nil
}

func (e *Engine) persistPausedResult(
	ctx context.Context,
	req RunRequest,
	def Definition,
	results map[string]*StepResult,
	pendingStep string,
	pause PauseError,
	resumeCount int,
	executionOrder []string,
	compileSummary graphCompileSummary,
	onEvent func(StreamEvent) error,
) (RunResult, error) {
	paused, err := e.persistContinuation(ctx, req, def, results, pendingStep, pause)
	if err != nil {
		return RunResult{}, err
	}
	out := buildResult(req.RunID, def.WorkflowID, resumeCount, def.Steps, results, executionOrder, compileSummary)
	out.WorkflowStatus = string(pause.Reason)
	out.WorkflowCheckpoint = &paused.Ref
	out.WorkflowCheckpointGeneration = paused.Generation
	out.WorkflowGraphDigest = paused.GraphDigest
	if onEvent != nil {
		if eventErr := onEvent(StreamEvent{Kind: "workflow.paused", Result: &out}); eventErr != nil {
			return RunResult{}, eventErr
		}
	}
	return out, ErrWorkflowPaused
}

func (e *Engine) persistContinuation(ctx context.Context, req RunRequest, def Definition, results map[string]*StepResult, pendingStep string, pause PauseError) (WorkflowContinuation, error) {
	if e == nil || e.continuations == nil || strings.TrimSpace(e.continuationNamespace) == "" {
		return WorkflowContinuation{}, ErrWorkflowContinuationUnsupported
	}
	now := e.now().UTC()
	if pause.Reason == PauseBudgetExhausted && pause.OriginalDeadline.IsZero() {
		pause.OriginalDeadline = now.Add(24 * time.Hour)
	}
	steps := make([]WorkflowStepContinuation, 0, len(results))
	for id, result := range results {
		if result == nil {
			continue
		}
		outputDigest := ""
		if result.Output.Payload != nil {
			payload, marshalErr := json.Marshal(result.Output.Payload)
			if marshalErr != nil {
				return WorkflowContinuation{}, ErrWorkflowContinuationMalformed
			}
			outputDigest = workflowDigestBytes(payload)
		}
		steps = append(steps, WorkflowStepContinuation{StepID: id, Status: result.Status, Attempts: result.Attempts, EffectReference: strings.TrimSpace(result.Output.EffectReference), OutputDigest: outputDigest})
	}
	sort.Slice(steps, func(i, j int) bool { return steps[i].StepID < steps[j].StepID })
	continuation, err := SealWorkflowContinuation(WorkflowContinuation{
		Ref: WorkflowContinuationRef{OwnerNamespace: e.continuationNamespace}, GraphID: def.WorkflowID, GraphVersion: "v1", GraphDigest: workflowDefinitionDigest(def), ParentRunID: strings.TrimSpace(req.ParentRunID), RunID: strings.TrimSpace(req.RunID), SessionID: strings.TrimSpace(req.SessionID), CausationID: strings.TrimSpace(req.CausationID), AuthorityDigest: strings.TrimSpace(req.AuthorityDigest), PlanDigest: strings.TrimSpace(req.PlanDigest), WriteScopeDigest: strings.TrimSpace(req.WriteScopeDigest), EffectNamespace: strings.TrimSpace(req.EffectNamespace), OccurrenceID: strings.TrimSpace(req.OccurrenceID), PauseReason: pause.Reason, Definition: def, Steps: steps, PendingStep: pendingStep, PendingActionReference: strings.TrimSpace(pause.PendingActionReference), ConsumedBudget: pause.ConsumedBudget, OriginalDeadline: pause.OriginalDeadline, PredecessorCheckpointID: req.continuationPredecessor, Generation: req.continuationGeneration + 1, CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour),
	})
	if err != nil {
		return WorkflowContinuation{}, err
	}
	if err = e.continuations.Save(ctx, continuation); err != nil {
		return WorkflowContinuation{}, err
	}
	return continuation, nil
}

func (e *Engine) executeFromContinuation(ctx context.Context, c WorkflowContinuation, onEvent func(StreamEvent) error) (RunResult, error) {
	if c.Definition.WorkflowID != c.GraphID || workflowDefinitionDigest(c.Definition) != c.GraphDigest {
		return RunResult{}, ErrWorkflowContinuationCorrupt
	}
	steps := make(map[string]CheckpointStep, len(c.Steps))
	for _, step := range c.Steps {
		steps[step.StepID] = CheckpointStep{Status: step.Status, Attempts: step.Attempts}
	}
	return e.execute(ctx, RunRequest{RunID: c.RunID, SessionID: c.SessionID, ParentRunID: c.ParentRunID, CausationID: c.CausationID, AuthorityDigest: c.AuthorityDigest, PlanDigest: c.PlanDigest, WriteScopeDigest: c.WriteScopeDigest, EffectNamespace: c.EffectNamespace, OccurrenceID: c.OccurrenceID, Resume: true, DSL: c.Definition, continuationPredecessor: c.Ref.CheckpointID, continuationGeneration: c.Generation, continuationSteps: steps, sourceContinuation: true}, onEvent)
}

func workflowDefinitionDigest(def Definition) string {
	canonical := normalizeDefinition(def)
	b, _ := json.Marshal(canonical)
	return workflowDigestBytes(b)
}

type WorkflowContinuationRef struct {
	SchemaVersion    string              `json:"schema_version"`
	CheckpointID     string              `json:"checkpoint_id"`
	Digest           string              `json:"digest"`
	OwnerNamespace   string              `json:"owner_namespace"`
	GraphID          string              `json:"graph_id"`
	GraphVersion     string              `json:"graph_version"`
	RunID            string              `json:"run_id"`
	SessionID        string              `json:"session_id"`
	ParentRunID      string              `json:"parent_run_id,omitempty"`
	CausationID      string              `json:"causation_id,omitempty"`
	AuthorityDigest  string              `json:"authority_digest,omitempty"`
	PlanDigest       string              `json:"plan_digest,omitempty"`
	WriteScopeDigest string              `json:"write_scope_digest,omitempty"`
	EffectNamespace  string              `json:"effect_namespace,omitempty"`
	OccurrenceID     string              `json:"occurrence_id,omitempty"`
	PauseReason      WorkflowPauseReason `json:"pause_reason"`
	CreatedAt        time.Time           `json:"created_at"`
	ExpiresAt        time.Time           `json:"expires_at"`
}

func (r WorkflowContinuationRef) Validate() error {
	if r.SchemaVersion != WorkflowContinuationVersion || strings.TrimSpace(r.CheckpointID) == "" || (strings.TrimSpace(r.Digest) == "" && r.Digest != "pending") || strings.TrimSpace(r.OwnerNamespace) == "" || strings.TrimSpace(r.GraphID) == "" || strings.TrimSpace(r.GraphVersion) == "" || strings.TrimSpace(r.RunID) == "" || strings.TrimSpace(r.SessionID) == "" {
		return ErrWorkflowContinuationMalformed
	}
	if r.PauseReason != PauseInputRequired && r.PauseReason != PauseApprovalRequired && r.PauseReason != PauseBudgetExhausted {
		return ErrWorkflowContinuationUnsupported
	}
	if r.CreatedAt.IsZero() || !r.ExpiresAt.After(r.CreatedAt) || r.ExpiresAt.Sub(r.CreatedAt) > 24*time.Hour {
		return ErrWorkflowContinuationMalformed
	}
	for _, value := range []string{r.CheckpointID, r.OwnerNamespace, r.GraphID, r.GraphVersion, r.RunID, r.SessionID, r.ParentRunID, r.CausationID, r.AuthorityDigest, r.PlanDigest, r.WriteScopeDigest, r.EffectNamespace, r.OccurrenceID} {
		if len(value) > 256 || strings.TrimSpace(value) != value || strings.ContainsAny(value, "\r\n\t") {
			return ErrWorkflowContinuationMalformed
		}
	}
	return nil
}

type WorkflowStepContinuation struct {
	StepID          string     `json:"step_id"`
	Status          StepStatus `json:"status"`
	Attempts        int        `json:"attempts"`
	ConsumedBudget  int64      `json:"consumed_budget,omitempty"`
	EffectReference string     `json:"effect_reference,omitempty"`
	OutputDigest    string     `json:"output_digest,omitempty"`
}

type WorkflowContinuation struct {
	Ref                          WorkflowContinuationRef    `json:"ref"`
	GraphID                      string                     `json:"graph_id"`
	GraphVersion                 string                     `json:"graph_version"`
	GraphDigest                  string                     `json:"graph_digest"`
	ParentRunID                  string                     `json:"parent_run_id,omitempty"`
	RunID                        string                     `json:"run_id"`
	SessionID                    string                     `json:"session_id"`
	CausationID                  string                     `json:"causation_id,omitempty"`
	AuthorityDigest              string                     `json:"authority_digest,omitempty"`
	PlanDigest                   string                     `json:"plan_digest,omitempty"`
	WriteScopeDigest             string                     `json:"write_scope_digest,omitempty"`
	EffectNamespace              string                     `json:"effect_namespace,omitempty"`
	OccurrenceID                 string                     `json:"occurrence_id,omitempty"`
	PauseReason                  WorkflowPauseReason        `json:"pause_reason"`
	Definition                   Definition                 `json:"definition"`
	Steps                        []WorkflowStepContinuation `json:"steps,omitempty"`
	PendingStep                  string                     `json:"pending_step,omitempty"`
	PendingActionReference       string                     `json:"pending_action_reference,omitempty"`
	ConsumedBudget               int64                      `json:"consumed_budget,omitempty"`
	OriginalDeadline             time.Time                  `json:"original_deadline,omitempty"`
	TerminalArbitrationReference string                     `json:"terminal_arbitration_reference,omitempty"`
	PredecessorCheckpointID      string                     `json:"predecessor_checkpoint_id,omitempty"`
	SuccessorCheckpointID        string                     `json:"successor_checkpoint_id,omitempty"`
	Generation                   uint64                     `json:"generation"`
	CreatedAt                    time.Time                  `json:"created_at"`
	ExpiresAt                    time.Time                  `json:"expires_at"`
}

func SealWorkflowContinuation(c WorkflowContinuation) (WorkflowContinuation, error) {
	if c.Ref.SchemaVersion == "" {
		c.Ref.SchemaVersion = WorkflowContinuationVersion
	}
	if c.Ref.CheckpointID == "" {
		c.Ref.CheckpointID = c.GraphID + ":" + c.RunID + ":" + fmt.Sprint(c.Generation)
	}
	if c.Ref.GraphID == "" {
		c.Ref.GraphID = c.GraphID
	}
	if c.Ref.GraphVersion == "" {
		c.Ref.GraphVersion = c.GraphVersion
	}
	if c.Ref.RunID == "" {
		c.Ref.RunID = c.RunID
	}
	if c.Ref.SessionID == "" {
		c.Ref.SessionID = c.SessionID
	}
	if c.Ref.ParentRunID == "" {
		c.Ref.ParentRunID = c.ParentRunID
	}
	if c.Ref.CausationID == "" {
		c.Ref.CausationID = c.CausationID
	}
	if c.Ref.AuthorityDigest == "" {
		c.Ref.AuthorityDigest = c.AuthorityDigest
	}
	if c.Ref.PlanDigest == "" {
		c.Ref.PlanDigest = c.PlanDigest
	}
	if c.Ref.WriteScopeDigest == "" {
		c.Ref.WriteScopeDigest = c.WriteScopeDigest
	}
	if c.Ref.EffectNamespace == "" {
		c.Ref.EffectNamespace = c.EffectNamespace
	}
	if c.Ref.OccurrenceID == "" {
		c.Ref.OccurrenceID = c.OccurrenceID
	}
	if c.Ref.PauseReason == "" {
		c.Ref.PauseReason = c.PauseReason
	}
	if c.Ref.CreatedAt.IsZero() {
		c.Ref.CreatedAt = c.CreatedAt
	}
	if c.Ref.ExpiresAt.IsZero() {
		c.Ref.ExpiresAt = c.ExpiresAt
	}
	c.Ref.Digest = ""
	// A sealed checkpoint may validate its shape before the canonical digest is computed.
	c.Ref.Digest = "pending"
	if err := c.validateShape(); err != nil {
		return WorkflowContinuation{}, err
	}
	c.DigestMaterial()
	c.Ref.Digest = workflowDigest(c)
	return c.Detached(), nil
}

func (c WorkflowContinuation) DigestMaterial() { c.Ref.Digest = "" }

func (c WorkflowContinuation) Detached() WorkflowContinuation {
	c.Steps = append([]WorkflowStepContinuation(nil), c.Steps...)
	return c
}

func (c WorkflowContinuation) Validate() error {
	if err := c.validateShape(); err != nil {
		return err
	}
	if c.Ref.Digest != workflowDigest(c) {
		return ErrWorkflowContinuationCorrupt
	}
	return nil
}

func (c WorkflowContinuation) validateShape() error {
	if c.GraphID == "" || c.GraphVersion == "" || c.GraphDigest == "" || c.RunID == "" || c.SessionID == "" || c.Generation == 0 || c.ConsumedBudget < 0 || len(c.Steps) > WorkflowContinuationMaxEntries || (c.Definition.WorkflowID != "" && c.Definition.WorkflowID != c.GraphID) {
		return ErrWorkflowContinuationMalformed
	}
	for _, value := range []string{c.GraphID, c.GraphVersion, c.GraphDigest, c.ParentRunID, c.RunID, c.SessionID, c.CausationID, c.AuthorityDigest, c.PlanDigest, c.WriteScopeDigest, c.EffectNamespace, c.OccurrenceID, c.PendingStep, c.PendingActionReference, c.TerminalArbitrationReference, c.PredecessorCheckpointID, c.SuccessorCheckpointID} {
		if len(value) > 256 || strings.TrimSpace(value) != value || strings.ContainsAny(value, "\r\n\t") {
			return ErrWorkflowContinuationMalformed
		}
	}
	if c.Ref.OwnerNamespace == "" {
		return ErrWorkflowContinuationMalformed
	}
	if c.PauseReason != PauseInputRequired && c.PauseReason != PauseApprovalRequired && c.PauseReason != PauseBudgetExhausted {
		return ErrWorkflowContinuationUnsupported
	}
	if c.PauseReason == PauseBudgetExhausted && c.OriginalDeadline.IsZero() {
		return ErrWorkflowContinuationMalformed
	}
	if c.CreatedAt.IsZero() || !c.ExpiresAt.After(c.CreatedAt) || c.ExpiresAt.Sub(c.CreatedAt) > 24*time.Hour {
		return ErrWorkflowContinuationMalformed
	}
	for _, step := range c.Steps {
		if step.StepID == "" || step.Attempts < 0 || step.ConsumedBudget < 0 {
			return ErrWorkflowContinuationMalformed
		}
	}
	ref := c.Ref
	if ref.GraphID == "" {
		ref.GraphID = c.GraphID
	}
	if ref.GraphVersion == "" {
		ref.GraphVersion = c.GraphVersion
	}
	if ref.RunID == "" {
		ref.RunID = c.RunID
	}
	if ref.SessionID == "" {
		ref.SessionID = c.SessionID
	}
	if ref.PauseReason == "" {
		ref.PauseReason = c.PauseReason
	}
	if ref.CreatedAt.IsZero() {
		ref.CreatedAt = c.CreatedAt
	}
	if ref.ExpiresAt.IsZero() {
		ref.ExpiresAt = c.ExpiresAt
	}
	if err := ref.Validate(); err != nil {
		return err
	}
	envelope, err := json.Marshal(c)
	if err != nil || len(envelope) > WorkflowContinuationMaxBytes {
		return ErrWorkflowContinuationMalformed
	}
	return nil
}

func workflowDigest(c WorkflowContinuation) string {
	copy := c.Detached()
	copy.Ref.Digest = ""
	b, _ := json.Marshal(copy)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func workflowDigestBytes(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

type WorkflowResumeRequest struct {
	Ref              WorkflowContinuationRef `json:"ref"`
	GraphDigest      string                  `json:"graph_digest"`
	Generation       uint64                  `json:"generation"`
	IdempotencyKey   string                  `json:"idempotency_key"`
	AuthorityDigest  string                  `json:"authority_digest,omitempty"`
	PlanDigest       string                  `json:"plan_digest,omitempty"`
	WriteScopeDigest string                  `json:"write_scope_digest,omitempty"`
	EffectNamespace  string                  `json:"effect_namespace,omitempty"`
	OccurrenceID     string                  `json:"occurrence_id,omitempty"`
}

func (r WorkflowResumeRequest) Validate() error {
	if err := r.Ref.Validate(); err != nil {
		return err
	}
	if r.GraphDigest == "" || r.Generation == 0 || r.IdempotencyKey == "" {
		return ErrWorkflowContinuationMalformed
	}
	for _, value := range []string{r.AuthorityDigest, r.PlanDigest, r.WriteScopeDigest, r.EffectNamespace, r.OccurrenceID} {
		if len(value) > 256 || strings.TrimSpace(value) != value || strings.ContainsAny(value, "\r\n\t") {
			return ErrWorkflowContinuationMalformed
		}
	}
	return nil
}

func (r WorkflowResumeRequest) identity() string {
	b, _ := json.Marshal(r)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

type WorkflowResumeClaim struct {
	Request    WorkflowResumeRequest    `json:"request"`
	AttemptID  string                   `json:"attempt_id"`
	Generation uint64                   `json:"generation"`
	State      WorkflowResumeClaimState `json:"state"`
	Duplicate  bool                     `json:"duplicate,omitempty"`
	Result     *RunResult               `json:"result,omitempty"`
}

type WorkflowResumeClaimState string

const (
	WorkflowResumeClaimClaimed          WorkflowResumeClaimState = "claimed"
	WorkflowResumeClaimRunning          WorkflowResumeClaimState = "running"
	WorkflowResumeClaimCompleted        WorkflowResumeClaimState = "completed"
	WorkflowResumeClaimRecoveryRequired WorkflowResumeClaimState = "recovery_required"
)

type WorkflowContinuationStore interface {
	Save(context.Context, WorkflowContinuation) error
	Load(context.Context, WorkflowContinuationRef) (WorkflowContinuation, error)
	ClaimResume(context.Context, WorkflowResumeRequest) (WorkflowResumeClaim, error)
	MarkResumeRunning(context.Context, WorkflowResumeClaim) (WorkflowResumeClaim, error)
	CommitResumeResult(context.Context, WorkflowResumeClaim, RunResult) (WorkflowResumeClaim, error)
}

type workflowRecord struct {
	Continuation WorkflowContinuation `json:"continuation"`
	Claim        *WorkflowResumeClaim `json:"claim,omitempty"`
	Identity     string               `json:"identity,omitempty"`
}

type MemoryWorkflowContinuationStore struct {
	mu        sync.Mutex
	namespace string
	records   map[string]workflowRecord
	now       func() time.Time
}

func NewMemoryWorkflowContinuationStore(namespace string) *MemoryWorkflowContinuationStore {
	return &MemoryWorkflowContinuationStore{namespace: strings.TrimSpace(namespace), records: map[string]workflowRecord{}, now: time.Now}
}

func (s *MemoryWorkflowContinuationStore) Save(ctx context.Context, c WorkflowContinuation) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.namespace == "" {
		return ErrWorkflowContinuationUnavailable
	}
	if err := c.Validate(); err != nil {
		return err
	}
	if c.Ref.OwnerNamespace != s.namespace {
		return ErrWorkflowContinuationForeign
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if old, ok := s.records[c.Ref.CheckpointID]; ok {
		if old.Continuation.Ref.Digest == c.Ref.Digest {
			return nil
		}
		return ErrWorkflowContinuationConflict
	}
	s.records[c.Ref.CheckpointID] = workflowRecord{Continuation: c.Detached()}
	return nil
}

func (s *MemoryWorkflowContinuationStore) Load(ctx context.Context, ref WorkflowContinuationRef) (WorkflowContinuation, error) {
	if err := ctx.Err(); err != nil {
		return WorkflowContinuation{}, err
	}
	if s == nil || s.namespace == "" {
		return WorkflowContinuation{}, ErrWorkflowContinuationUnavailable
	}
	if err := ref.Validate(); err != nil {
		return WorkflowContinuation{}, err
	}
	if ref.OwnerNamespace != s.namespace {
		return WorkflowContinuation{}, ErrWorkflowContinuationForeign
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.records[ref.CheckpointID]
	if !ok {
		return WorkflowContinuation{}, ErrWorkflowContinuationUnavailable
	}
	if r.Continuation.Ref.Digest != ref.Digest {
		return WorkflowContinuation{}, ErrWorkflowContinuationForeign
	}
	if !s.now().Before(r.Continuation.ExpiresAt) {
		return WorkflowContinuation{}, ErrWorkflowContinuationExpired
	}
	if err := r.Continuation.Validate(); err != nil {
		return WorkflowContinuation{}, err
	}
	return r.Continuation.Detached(), nil
}

func (s *MemoryWorkflowContinuationStore) ClaimResume(ctx context.Context, req WorkflowResumeRequest) (WorkflowResumeClaim, error) {
	if err := ctx.Err(); err != nil {
		return WorkflowResumeClaim{}, err
	}
	if s == nil || s.namespace == "" {
		return WorkflowResumeClaim{}, ErrWorkflowContinuationUnavailable
	}
	if err := req.Validate(); err != nil {
		return WorkflowResumeClaim{}, err
	}
	if req.Ref.OwnerNamespace != s.namespace {
		return WorkflowResumeClaim{}, ErrWorkflowContinuationForeign
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.records[req.Ref.CheckpointID]
	if !ok {
		return WorkflowResumeClaim{}, ErrWorkflowContinuationUnavailable
	}
	if err := r.Continuation.Validate(); err != nil {
		return WorkflowResumeClaim{}, err
	}
	if r.Continuation.Ref.Digest != req.Ref.Digest || r.Continuation.GraphDigest != req.GraphDigest || r.Continuation.Generation != req.Generation {
		return WorkflowResumeClaim{}, ErrWorkflowContinuationConflict
	}
	if !s.now().Before(r.Continuation.ExpiresAt) {
		return WorkflowResumeClaim{}, ErrWorkflowContinuationExpired
	}
	identity := req.identity()
	if r.Claim != nil {
		if r.Identity != identity {
			return WorkflowResumeClaim{}, ErrWorkflowContinuationConflict
		}
		duplicate := *r.Claim
		duplicate.Duplicate = true
		if duplicate.Result == nil {
			if duplicate.State == WorkflowResumeClaimRecoveryRequired {
				return duplicate, ErrWorkflowContinuationRecoveryRequired
			}
			return duplicate, ErrWorkflowContinuationInProgress
		}
		return duplicate, nil
	}
	claim := WorkflowResumeClaim{Request: req, AttemptID: req.Ref.CheckpointID + ":attempt", Generation: req.Generation + 1, State: WorkflowResumeClaimClaimed}
	r.Claim = &claim
	r.Identity = identity
	s.records[req.Ref.CheckpointID] = r
	return claim, nil
}

func (s *MemoryWorkflowContinuationStore) MarkResumeRunning(ctx context.Context, claim WorkflowResumeClaim) (WorkflowResumeClaim, error) {
	if err := ctx.Err(); err != nil {
		return WorkflowResumeClaim{}, err
	}
	if s == nil || s.namespace == "" {
		return WorkflowResumeClaim{}, ErrWorkflowContinuationUnavailable
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.records[claim.Request.Ref.CheckpointID]
	if !ok || r.Claim == nil || r.Identity != claim.Request.identity() || r.Claim.AttemptID != claim.AttemptID {
		return WorkflowResumeClaim{}, ErrWorkflowContinuationConflict
	}
	if r.Claim.Result != nil || r.Claim.State == WorkflowResumeClaimCompleted {
		return WorkflowResumeClaim{}, ErrWorkflowContinuationConflict
	}
	if r.Claim.State == WorkflowResumeClaimRecoveryRequired {
		return WorkflowResumeClaim{}, ErrWorkflowContinuationRecoveryRequired
	}
	if r.Claim.State != WorkflowResumeClaimClaimed && r.Claim.State != WorkflowResumeClaimRunning {
		return WorkflowResumeClaim{}, ErrWorkflowContinuationConflict
	}
	r.Claim.State = WorkflowResumeClaimRunning
	s.records[claim.Request.Ref.CheckpointID] = r
	return *r.Claim, nil
}

func (s *MemoryWorkflowContinuationStore) CommitResumeResult(ctx context.Context, claim WorkflowResumeClaim, result RunResult) (WorkflowResumeClaim, error) {
	if err := ctx.Err(); err != nil {
		return WorkflowResumeClaim{}, err
	}
	if s == nil {
		return WorkflowResumeClaim{}, ErrWorkflowContinuationUnavailable
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.records[claim.Request.Ref.CheckpointID]
	if !ok || r.Claim == nil || r.Claim.AttemptID != claim.AttemptID || r.Identity != claim.Request.identity() {
		return WorkflowResumeClaim{}, ErrWorkflowContinuationConflict
	}
	if r.Claim.Result != nil {
		if sameWorkflowRunResult(*r.Claim.Result, result) {
			return *r.Claim, nil
		}
		return WorkflowResumeClaim{}, ErrWorkflowContinuationConflict
	}
	if claim.State != WorkflowResumeClaimClaimed && claim.State != WorkflowResumeClaimRunning {
		return WorkflowResumeClaim{}, ErrWorkflowContinuationRecoveryRequired
	}
	copy := result
	copy.Steps = append([]StepResult(nil), result.Steps...)
	copy.ExecutionOrder = append([]string(nil), result.ExecutionOrder...)
	r.Claim.Result = &copy
	r.Claim.State = WorkflowResumeClaimCompleted
	s.records[claim.Request.Ref.CheckpointID] = r
	return *r.Claim, nil
}

func sameWorkflowRunResult(a, b RunResult) bool {
	left, err := json.Marshal(a)
	if err != nil {
		return false
	}
	right, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return string(left) == string(right)
}

type FileWorkflowContinuationStore struct {
	mu        sync.Mutex
	root      string
	namespace string
	memory    *MemoryWorkflowContinuationStore
}

func (e *Engine) ContinuationCapabilities(ctx context.Context) (types.ContinuationCapabilities, error) {
	if err := ctx.Err(); err != nil {
		return types.ContinuationCapabilities{}, err
	}
	if e == nil || strings.TrimSpace(e.continuationNamespace) != "workflow" {
		return types.ContinuationCapabilities{}, ErrWorkflowContinuationUnsupported
	}
	if _, ok := e.continuations.(*FileWorkflowContinuationStore); !ok {
		return types.ContinuationCapabilities{}, ErrWorkflowContinuationUnavailable
	}
	capabilities := types.ContinuationCapabilities{
		ProfileVersion: types.NativeContinuationVersion, SourceNamespace: "workflow",
		ContractVersion: "baymax.runtime.contract.v1", ModuleRevision: "workflow-continuation-v1",
		APIVersion: "host_runtime_capability_manifest.v1", RetentionPolicy: "same-host single-writer sequential reopen; 24h",
		ExpiryPolicy: "24h or original budget deadline", CheckpointCreation: true, CrossProcessRestore: true,
		PauseReasons: []string{string(PauseInputRequired), string(PauseApprovalRequired), string(PauseBudgetExhausted)},
		EvidenceReferences: map[string]string{
			"checkpoint_creation":   "test:TestEngineResumeContinuationRestoresFromFileStoreOnNewEngine",
			"cross_process_restore": "test:TestEngineResumeContinuationRestoresFromFileStoreOnNewEngine",
			"input_required":        "test:TestWorkflowContinuationRunStreamParity",
			"approval_required":     "test:TestWorkflowPauseControllerStopsApprovalBeforeEffectDispatch",
			"budget_exhausted":      "test:TestWorkflowBudgetPausePreservesConsumedBudgetAndOriginalDeadline",
		},
		Deployment: types.SameHostSingleWriterScope(e.continuationNamespace, "fixture:deployment-scope.v1"),
	}
	if err := capabilities.Validate(); err != nil {
		return types.ContinuationCapabilities{}, ErrWorkflowContinuationUnsupported
	}
	return capabilities, nil
}

func NewFileWorkflowContinuationStore(root, namespace string) (*FileWorkflowContinuationStore, error) {
	if strings.TrimSpace(root) == "" || strings.TrimSpace(namespace) == "" {
		return nil, ErrWorkflowContinuationMalformed
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, ErrWorkflowContinuationUnavailable
	}
	return &FileWorkflowContinuationStore{root: root, namespace: namespace, memory: NewMemoryWorkflowContinuationStore(namespace)}, nil
}

func (s *FileWorkflowContinuationStore) path(id string) string {
	clean := strings.NewReplacer("/", "_", "\\", "_", ":", "_").Replace(id)
	return filepath.Join(s.root, clean+".json")
}

func (s *FileWorkflowContinuationStore) loadRecord(ref WorkflowContinuationRef) (workflowRecord, error) {
	if s == nil || s.memory == nil || s.namespace == "" {
		return workflowRecord{}, ErrWorkflowContinuationUnavailable
	}
	if err := ref.Validate(); err != nil {
		return workflowRecord{}, err
	}
	if ref.OwnerNamespace != s.namespace {
		return workflowRecord{}, ErrWorkflowContinuationForeign
	}
	raw, err := os.ReadFile(s.path(ref.CheckpointID))
	if err != nil {
		return workflowRecord{}, ErrWorkflowContinuationUnavailable
	}
	var record workflowRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return workflowRecord{}, ErrWorkflowContinuationCorrupt
	}
	if err := record.Continuation.Validate(); err != nil {
		return workflowRecord{}, err
	}
	if record.Continuation.Ref.OwnerNamespace != s.namespace || record.Continuation.Ref.Digest != ref.Digest {
		return workflowRecord{}, ErrWorkflowContinuationForeign
	}
	if record.Claim != nil && record.Claim.Result == nil && record.Claim.State == WorkflowResumeClaimClaimed {
		record.Claim.State = WorkflowResumeClaimRecoveryRequired
	}
	return record, nil
}

func (s *FileWorkflowContinuationStore) persistRecord(id string, record workflowRecord) error {
	if err := os.MkdirAll(s.root, 0o700); err != nil {
		return ErrWorkflowContinuationUnavailable
	}
	b, err := json.Marshal(record)
	if err != nil {
		return ErrWorkflowContinuationCorrupt
	}
	tmp, err := os.CreateTemp(s.root, "workflow-continuation-*.tmp")
	if err != nil {
		return ErrWorkflowContinuationUnavailable
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		return ErrWorkflowContinuationUnavailable
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return ErrWorkflowContinuationUnavailable
	}
	if err := tmp.Close(); err != nil {
		return ErrWorkflowContinuationUnavailable
	}
	if err := os.Rename(tmpName, s.path(id)); err != nil {
		return ErrWorkflowContinuationUnavailable
	}
	return nil
}

func (s *FileWorkflowContinuationStore) Save(ctx context.Context, c WorkflowContinuation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	before, existed := s.memory.records[c.Ref.CheckpointID]
	if err := s.memory.Save(ctx, c); err != nil {
		return err
	}
	if err := s.persistRecord(c.Ref.CheckpointID, s.memory.records[c.Ref.CheckpointID]); err != nil {
		if existed {
			s.memory.records[c.Ref.CheckpointID] = before
		} else {
			delete(s.memory.records, c.Ref.CheckpointID)
		}
		return err
	}
	return nil
}
func (s *FileWorkflowContinuationStore) Load(ctx context.Context, ref WorkflowContinuationRef) (WorkflowContinuation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, err := s.loadRecord(ref)
	if err != nil {
		return WorkflowContinuation{}, err
	}
	s.memory.records[ref.CheckpointID] = record
	return s.memory.Load(ctx, ref)
}
func (s *FileWorkflowContinuationStore) ClaimResume(ctx context.Context, req WorkflowResumeRequest) (WorkflowResumeClaim, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.memory.records[req.Ref.CheckpointID]; !ok {
		record, err := s.loadRecord(req.Ref)
		if err != nil {
			return WorkflowResumeClaim{}, err
		}
		s.memory.records[req.Ref.CheckpointID] = record
	}
	before := s.memory.records[req.Ref.CheckpointID]
	claim, err := s.memory.ClaimResume(ctx, req)
	if err != nil {
		return claim, err
	}
	if err := s.persistRecord(req.Ref.CheckpointID, s.memory.records[req.Ref.CheckpointID]); err != nil {
		s.memory.records[req.Ref.CheckpointID] = before
		return WorkflowResumeClaim{}, ErrWorkflowContinuationUnavailable
	}
	return claim, nil
}

func (s *FileWorkflowContinuationStore) MarkResumeRunning(ctx context.Context, claim WorkflowResumeClaim) (WorkflowResumeClaim, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.memory.records[claim.Request.Ref.CheckpointID]; !ok {
		record, err := s.loadRecord(claim.Request.Ref)
		if err != nil {
			return WorkflowResumeClaim{}, err
		}
		s.memory.records[claim.Request.Ref.CheckpointID] = record
	}
	before := s.memory.records[claim.Request.Ref.CheckpointID]
	running, err := s.memory.MarkResumeRunning(ctx, claim)
	if err != nil {
		return running, err
	}
	if err := s.persistRecord(claim.Request.Ref.CheckpointID, s.memory.records[claim.Request.Ref.CheckpointID]); err != nil {
		s.memory.records[claim.Request.Ref.CheckpointID] = before
		return WorkflowResumeClaim{}, ErrWorkflowContinuationUnavailable
	}
	return running, nil
}
func (s *FileWorkflowContinuationStore) CommitResumeResult(ctx context.Context, claim WorkflowResumeClaim, result RunResult) (WorkflowResumeClaim, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	before := s.memory.records[claim.Request.Ref.CheckpointID]
	committed, err := s.memory.CommitResumeResult(ctx, claim, result)
	if err != nil {
		return WorkflowResumeClaim{}, err
	}
	if err := s.persistRecord(claim.Request.Ref.CheckpointID, s.memory.records[claim.Request.Ref.CheckpointID]); err != nil {
		s.memory.records[claim.Request.Ref.CheckpointID] = before
		return WorkflowResumeClaim{}, ErrWorkflowContinuationUnavailable
	}
	return committed, nil
}
