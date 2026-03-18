package a2a

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

const (
	ReasonSubmit        = "a2a.submit"
	ReasonStatusPoll    = "a2a.status_poll"
	ReasonCallbackRetry = "a2a.callback_retry"
	ReasonResolve       = "a2a.resolve"
)

type TaskStatus string

const (
	StatusSubmitted TaskStatus = "submitted"
	StatusRunning   TaskStatus = "running"
	StatusSucceeded TaskStatus = "succeeded"
	StatusFailed    TaskStatus = "failed"
	StatusCanceled  TaskStatus = "canceled"
)

type ErrorLayer string

const (
	ErrorLayerTransport ErrorLayer = "transport"
	ErrorLayerProtocol  ErrorLayer = "protocol"
	ErrorLayerSemantic  ErrorLayer = "semantic"
)

type TaskRequest struct {
	TaskID               string         `json:"task_id,omitempty"`
	AgentID              string         `json:"agent_id"`
	PeerID               string         `json:"peer_id,omitempty"`
	Method               string         `json:"method,omitempty"`
	RequiredCapabilities []string       `json:"required_capabilities,omitempty"`
	Payload              map[string]any `json:"payload,omitempty"`
}

type TaskRecord struct {
	TaskID        string           `json:"task_id"`
	AgentID       string           `json:"agent_id"`
	PeerID        string           `json:"peer_id"`
	Status        TaskStatus       `json:"status"`
	Progress      map[string]any   `json:"progress,omitempty"`
	Result        map[string]any   `json:"result,omitempty"`
	ErrorClass    types.ErrorClass `json:"error_class,omitempty"`
	A2AErrorLayer string           `json:"a2a_error_layer,omitempty"`
	ErrorCode     string           `json:"error_code,omitempty"`
	ErrorMessage  string           `json:"error_message,omitempty"`
	UpdatedAt     time.Time        `json:"updated_at"`
}

type AgentCard struct {
	AgentID       string   `json:"agent_id"`
	PeerID        string   `json:"peer_id"`
	SchemaVersion string   `json:"schema_version"`
	Endpoint      string   `json:"endpoint,omitempty"`
	Capabilities  []string `json:"capabilities,omitempty"`
	Priority      int      `json:"priority,omitempty"`
}

type Router interface {
	SelectPeer(cards []AgentCard, required []string) (AgentCard, error)
}

type DeterministicRouter struct {
	MaxCandidates int
	RequireAll    bool
}

func (r DeterministicRouter) SelectPeer(cards []AgentCard, required []string) (AgentCard, error) {
	normalizedRequired := normalizeCapabilities(required)
	maxCandidates := r.MaxCandidates
	if maxCandidates <= 0 {
		maxCandidates = len(cards)
	}
	candidates := make([]AgentCard, 0, len(cards))
	for _, card := range cards {
		card.AgentID = strings.TrimSpace(card.AgentID)
		card.PeerID = strings.TrimSpace(card.PeerID)
		card.SchemaVersion = strings.TrimSpace(card.SchemaVersion)
		if card.SchemaVersion == "" {
			card.SchemaVersion = "a2a.v1"
		}
		card.Capabilities = normalizeCapabilities(card.Capabilities)
		if card.PeerID == "" || card.AgentID == "" {
			continue
		}
		candidates = append(candidates, card)
	}
	if len(candidates) == 0 {
		return AgentCard{}, errors.New("a2a router has no available cards")
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Priority == candidates[j].Priority {
			if candidates[i].PeerID == candidates[j].PeerID {
				return candidates[i].AgentID < candidates[j].AgentID
			}
			return candidates[i].PeerID < candidates[j].PeerID
		}
		return candidates[i].Priority > candidates[j].Priority
	})
	if maxCandidates < len(candidates) {
		candidates = candidates[:maxCandidates]
	}

	type scored struct {
		card  AgentCard
		score int
	}
	scoredCandidates := make([]scored, 0, len(candidates))
	for _, card := range candidates {
		score := capabilityScore(card.Capabilities, normalizedRequired, r.RequireAll)
		if score < 0 {
			continue
		}
		scoredCandidates = append(scoredCandidates, scored{card: card, score: score})
	}
	if len(scoredCandidates) == 0 {
		return AgentCard{}, errors.New("a2a router found no peer matching required capabilities")
	}
	sort.Slice(scoredCandidates, func(i, j int) bool {
		if scoredCandidates[i].score == scoredCandidates[j].score {
			if scoredCandidates[i].card.PeerID == scoredCandidates[j].card.PeerID {
				if scoredCandidates[i].card.Priority == scoredCandidates[j].card.Priority {
					return scoredCandidates[i].card.AgentID < scoredCandidates[j].card.AgentID
				}
				return scoredCandidates[i].card.Priority > scoredCandidates[j].card.Priority
			}
			return scoredCandidates[i].card.PeerID < scoredCandidates[j].card.PeerID
		}
		return scoredCandidates[i].score > scoredCandidates[j].score
	})
	return scoredCandidates[0].card, nil
}

func capabilityScore(capabilities, required []string, requireAll bool) int {
	if len(required) == 0 {
		return 0
	}
	set := map[string]struct{}{}
	for _, capability := range capabilities {
		set[capability] = struct{}{}
	}
	score := 0
	for _, requiredCapability := range required {
		if _, ok := set[requiredCapability]; ok {
			score++
			continue
		}
		if requireAll {
			return -1
		}
	}
	if !requireAll && score == 0 {
		return -1
	}
	return score
}

type Handler interface {
	Handle(ctx context.Context, req TaskRequest) (map[string]any, error)
}

type HandlerFunc func(ctx context.Context, req TaskRequest) (map[string]any, error)

func (f HandlerFunc) Handle(ctx context.Context, req TaskRequest) (map[string]any, error) {
	return f(ctx, req)
}

type Server interface {
	Submit(ctx context.Context, req TaskRequest) (TaskRecord, error)
	Status(ctx context.Context, taskID string) (TaskRecord, error)
	Result(ctx context.Context, taskID string) (TaskRecord, error)
}

type InMemoryServer struct {
	mu      sync.Mutex
	tasks   map[string]TaskRecord
	handler Handler
	now     func() time.Time
	seq     int64

	timeline types.EventHandler
}

func NewInMemoryServer(handler Handler, timeline types.EventHandler) *InMemoryServer {
	return &InMemoryServer{
		tasks:    map[string]TaskRecord{},
		handler:  handler,
		now:      time.Now,
		timeline: timeline,
	}
}

func (s *InMemoryServer) Submit(ctx context.Context, req TaskRequest) (TaskRecord, error) {
	s.mu.Lock()
	req.TaskID = strings.TrimSpace(req.TaskID)
	req.AgentID = strings.TrimSpace(req.AgentID)
	req.PeerID = strings.TrimSpace(req.PeerID)
	req.Method = strings.TrimSpace(req.Method)
	if req.TaskID == "" {
		s.seq++
		req.TaskID = fmt.Sprintf("a2a-task-%06d", s.seq)
	}
	if req.AgentID == "" {
		return TaskRecord{}, errors.New("a2a submit requires agent_id")
	}
	if req.PeerID == "" {
		return TaskRecord{}, errors.New("a2a submit requires peer_id")
	}
	if _, ok := s.tasks[req.TaskID]; ok {
		s.mu.Unlock()
		return TaskRecord{}, fmt.Errorf("a2a task_id %q already exists", req.TaskID)
	}
	record := TaskRecord{
		TaskID:    req.TaskID,
		AgentID:   req.AgentID,
		PeerID:    req.PeerID,
		Status:    StatusSubmitted,
		Progress:  map[string]any{},
		Result:    map[string]any{},
		UpdatedAt: s.nowTime(),
	}
	s.tasks[record.TaskID] = record
	s.mu.Unlock()

	s.emitTimeline(ctx, record, ReasonSubmit)
	go s.execute(req)
	return record, nil
}

func (s *InMemoryServer) execute(req TaskRequest) {
	ctx := context.Background()
	s.updateTask(ctx, req.TaskID, func(rec *TaskRecord) {
		rec.Status = StatusRunning
		rec.UpdatedAt = s.nowTime()
	})
	if s.handler == nil {
		s.updateTask(ctx, req.TaskID, func(rec *TaskRecord) {
			rec.Status = StatusFailed
			rec.ErrorClass = types.ErrMCP
			rec.A2AErrorLayer = string(ErrorLayerProtocol)
			rec.ErrorCode = "handler_missing"
			rec.ErrorMessage = "a2a handler is not configured"
			rec.UpdatedAt = s.nowTime()
		})
		return
	}

	result, err := s.handler.Handle(ctx, req)
	if err == nil {
		s.updateTask(ctx, req.TaskID, func(rec *TaskRecord) {
			rec.Status = StatusSucceeded
			rec.Result = copyMap(result)
			rec.UpdatedAt = s.nowTime()
		})
		return
	}
	class, layer, code := ClassifyError(err)
	s.updateTask(ctx, req.TaskID, func(rec *TaskRecord) {
		rec.Status = StatusFailed
		rec.ErrorClass = class
		rec.A2AErrorLayer = string(layer)
		rec.ErrorCode = code
		rec.ErrorMessage = err.Error()
		rec.UpdatedAt = s.nowTime()
	})
}

func (s *InMemoryServer) Status(ctx context.Context, taskID string) (TaskRecord, error) {
	s.mu.Lock()
	record, ok := s.tasks[strings.TrimSpace(taskID)]
	if !ok {
		s.mu.Unlock()
		return TaskRecord{}, fmt.Errorf("a2a task %q not found", taskID)
	}
	s.mu.Unlock()

	s.emitTimeline(ctx, record, ReasonStatusPoll)
	return record, nil
}

func (s *InMemoryServer) Result(ctx context.Context, taskID string) (TaskRecord, error) {
	s.mu.Lock()
	record, ok := s.tasks[strings.TrimSpace(taskID)]
	if !ok {
		s.mu.Unlock()
		return TaskRecord{}, fmt.Errorf("a2a task %q not found", taskID)
	}
	if !isTerminal(record.Status) {
		s.mu.Unlock()
		return record, fmt.Errorf("a2a task %q is not terminal yet", taskID)
	}
	s.mu.Unlock()

	s.emitTimeline(ctx, record, ReasonResolve)
	return record, nil
}

func (s *InMemoryServer) updateTask(ctx context.Context, taskID string, apply func(*TaskRecord)) {
	s.mu.Lock()
	record, ok := s.tasks[strings.TrimSpace(taskID)]
	if !ok {
		s.mu.Unlock()
		return
	}
	apply(&record)
	s.tasks[record.TaskID] = record
	s.mu.Unlock()
	if isTerminal(record.Status) {
		s.emitTimeline(ctx, record, ReasonResolve)
		return
	}
	s.emitTimeline(ctx, record, ReasonSubmit)
}

func (s *InMemoryServer) emitTimeline(ctx context.Context, record TaskRecord, reason string) {
	if s == nil || s.timeline == nil {
		return
	}
	status := mapToSemanticStatus(record.Status)
	now := s.nowTime()
	payload := map[string]any{
		"phase":    string(types.ActionPhaseRun),
		"status":   string(status),
		"reason":   reason,
		"sequence": now.UnixNano(),
		"task_id":  record.TaskID,
		"agent_id": record.AgentID,
		"peer_id":  record.PeerID,
	}
	s.timeline.OnEvent(ctx, types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    types.EventTypeActionTimeline,
		RunID:   record.TaskID,
		Time:    now,
		Payload: payload,
	})
}

func (s *InMemoryServer) nowTime() time.Time {
	if s == nil || s.now == nil {
		return time.Now()
	}
	return s.now()
}

func isTerminal(status TaskStatus) bool {
	switch status {
	case StatusSucceeded, StatusFailed, StatusCanceled:
		return true
	default:
		return false
	}
}

func mapToSemanticStatus(status TaskStatus) types.ActionStatus {
	switch status {
	case StatusSubmitted:
		return types.ActionStatusPending
	case StatusRunning:
		return types.ActionStatusRunning
	case StatusSucceeded:
		return types.ActionStatusSucceeded
	case StatusFailed:
		return types.ActionStatusFailed
	case StatusCanceled:
		return types.ActionStatusCanceled
	default:
		return types.ActionStatusFailed
	}
}

type ClientPolicy struct {
	Timeout            time.Duration
	RequestMaxAttempts int
	RequestBackoff     time.Duration
	CallbackRetry      RetryPolicy
}

type RetryPolicy struct {
	MaxAttempts int
	Backoff     time.Duration
}

type Client struct {
	server   Server
	router   Router
	cards    []AgentCard
	policy   ClientPolicy
	now      func() time.Time
	timeline types.EventHandler
}

func NewClient(server Server, cards []AgentCard, router Router, policy ClientPolicy, timeline types.EventHandler) *Client {
	if router == nil {
		router = DeterministicRouter{MaxCandidates: 16, RequireAll: true}
	}
	if policy.Timeout <= 0 {
		policy.Timeout = 1500 * time.Millisecond
	}
	if policy.RequestMaxAttempts <= 0 {
		policy.RequestMaxAttempts = 3
	}
	if policy.RequestBackoff < 0 {
		policy.RequestBackoff = 0
	}
	if policy.CallbackRetry.MaxAttempts <= 0 {
		policy.CallbackRetry.MaxAttempts = 3
	}
	if policy.CallbackRetry.Backoff < 0 {
		policy.CallbackRetry.Backoff = 0
	}
	return &Client{
		server:   server,
		router:   router,
		cards:    append([]AgentCard(nil), cards...),
		policy:   policy,
		now:      time.Now,
		timeline: timeline,
	}
}

func (c *Client) Submit(ctx context.Context, req TaskRequest) (TaskRecord, error) {
	if c.server == nil {
		return TaskRecord{}, errors.New("a2a client server is not configured")
	}
	req.RequiredCapabilities = normalizeCapabilities(req.RequiredCapabilities)
	if strings.TrimSpace(req.PeerID) == "" {
		card, err := c.router.SelectPeer(c.cards, req.RequiredCapabilities)
		if err != nil {
			return TaskRecord{}, err
		}
		req.PeerID = card.PeerID
	}
	var lastErr error
	for attempt := 1; attempt <= c.policy.RequestMaxAttempts; attempt++ {
		record, err := c.withTimeout(ctx, func(callCtx context.Context) (TaskRecord, error) {
			return c.server.Submit(callCtx, req)
		})
		if err == nil {
			return record, nil
		}
		lastErr = err
		if attempt < c.policy.RequestMaxAttempts && isRetryableTransportError(err) {
			if c.policy.RequestBackoff > 0 {
				time.Sleep(c.policy.RequestBackoff)
			}
			continue
		}
		break
	}
	return TaskRecord{}, lastErr
}

func (c *Client) WaitResult(
	ctx context.Context,
	taskID string,
	pollInterval time.Duration,
	callback func(context.Context, TaskRecord) error,
) (TaskRecord, error) {
	if pollInterval <= 0 {
		pollInterval = 20 * time.Millisecond
	}
	for {
		record, err := c.withTimeout(ctx, func(callCtx context.Context) (TaskRecord, error) {
			return c.server.Status(callCtx, taskID)
		})
		if err != nil {
			return TaskRecord{}, err
		}
		if isTerminal(record.Status) {
			if callback != nil {
				if err := c.deliverCallback(ctx, record, callback); err != nil {
					return record, err
				}
			}
			result, err := c.withTimeout(ctx, func(callCtx context.Context) (TaskRecord, error) {
				return c.server.Result(callCtx, taskID)
			})
			if err != nil {
				return record, err
			}
			return result, nil
		}
		timer := time.NewTimer(pollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return TaskRecord{}, ctx.Err()
		case <-timer.C:
		}
	}
}

func (c *Client) deliverCallback(ctx context.Context, record TaskRecord, callback func(context.Context, TaskRecord) error) error {
	var lastErr error
	for attempt := 1; attempt <= c.policy.CallbackRetry.MaxAttempts; attempt++ {
		err := callback(ctx, record)
		if err == nil {
			return nil
		}
		lastErr = err
		if attempt >= c.policy.CallbackRetry.MaxAttempts {
			break
		}
		c.emitTimeline(ctx, record, ReasonCallbackRetry)
		if c.policy.CallbackRetry.Backoff > 0 {
			timer := time.NewTimer(c.policy.CallbackRetry.Backoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
	}
	return fmt.Errorf("a2a callback retry exhausted: %w", lastErr)
}

func (c *Client) emitTimeline(ctx context.Context, record TaskRecord, reason string) {
	if c == nil || c.timeline == nil {
		return
	}
	payload := map[string]any{
		"phase":    string(types.ActionPhaseRun),
		"status":   string(mapToSemanticStatus(record.Status)),
		"reason":   reason,
		"sequence": c.nowTime().UnixNano(),
		"task_id":  record.TaskID,
		"agent_id": record.AgentID,
		"peer_id":  record.PeerID,
	}
	c.timeline.OnEvent(ctx, types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    types.EventTypeActionTimeline,
		RunID:   record.TaskID,
		Time:    c.nowTime(),
		Payload: payload,
	})
}

func (c *Client) nowTime() time.Time {
	if c == nil || c.now == nil {
		return time.Now()
	}
	return c.now()
}

func (c *Client) withTimeout(ctx context.Context, call func(context.Context) (TaskRecord, error)) (TaskRecord, error) {
	timeout := c.policy.Timeout
	if timeout <= 0 {
		timeout = 1500 * time.Millisecond
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return call(callCtx)
}

func normalizeCapabilities(in []string) []string {
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, item := range in {
		key := strings.ToLower(strings.TrimSpace(item))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func isRetryableTransportError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(msg, "timeout") || strings.Contains(msg, "connection") || strings.Contains(msg, "transport")
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

func ClassifyError(err error) (types.ErrorClass, ErrorLayer, string) {
	if err == nil {
		return "", "", ""
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return types.ErrPolicyTimeout, ErrorLayerTransport, "timeout"
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case strings.Contains(message, "unsupported method"), strings.Contains(message, "protocol"):
		return types.ErrMCP, ErrorLayerProtocol, "unsupported_method"
	case strings.Contains(message, "invalid"), strings.Contains(message, "semantic"), strings.Contains(message, "schema"):
		return types.ErrContext, ErrorLayerSemantic, "invalid_payload"
	case strings.Contains(message, "connection"), strings.Contains(message, "refused"), strings.Contains(message, "transport"):
		return types.ErrMCP, ErrorLayerTransport, "transport_failure"
	default:
		return types.ErrMCP, ErrorLayerProtocol, "unknown"
	}
}

type RunSummary struct {
	A2ATaskTotal  int    `json:"a2a_task_total"`
	A2ATaskFailed int    `json:"a2a_task_failed"`
	PeerID        string `json:"peer_id,omitempty"`
	A2AErrorLayer string `json:"a2a_error_layer,omitempty"`
}

func BuildRunSummary(tasks []TaskRecord) RunSummary {
	out := RunSummary{}
	seen := map[string]TaskRecord{}
	for _, task := range tasks {
		key := strings.TrimSpace(task.TaskID)
		if key == "" {
			key = fmt.Sprintf("__anon__:%s:%s", strings.TrimSpace(task.AgentID), strings.TrimSpace(task.PeerID))
		}
		prev, ok := seen[key]
		if !ok {
			seen[key] = task
			continue
		}
		// Prefer newer records by UpdatedAt; for same timestamp prefer terminal states.
		if task.UpdatedAt.After(prev.UpdatedAt) || (task.UpdatedAt.Equal(prev.UpdatedAt) && isTerminal(task.Status) && !isTerminal(prev.Status)) {
			seen[key] = task
		}
	}

	out.A2ATaskTotal = len(seen)
	for _, task := range seen {
		if out.PeerID == "" && strings.TrimSpace(task.PeerID) != "" {
			out.PeerID = strings.TrimSpace(task.PeerID)
		}
		if task.Status == StatusFailed {
			out.A2ATaskFailed++
			if out.A2AErrorLayer == "" && strings.TrimSpace(task.A2AErrorLayer) != "" {
				out.A2AErrorLayer = strings.TrimSpace(task.A2AErrorLayer)
			}
		}
	}
	return out
}
