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
	ReasonSubmit           = "a2a.submit"
	ReasonStatusPoll       = "a2a.status_poll"
	ReasonCallbackRetry    = "a2a.callback_retry"
	ReasonResolve          = "a2a.resolve"
	ReasonSSESubscribe     = "a2a.sse_subscribe"
	ReasonSSEReconnect     = "a2a.sse_reconnect"
	ReasonDeliveryFallback = "a2a.delivery_fallback"
	ReasonVersionMismatch  = "a2a.version_mismatch"
)

const (
	DeliveryModeCallback = "callback"
	DeliveryModeSSE      = "sse"
)

const (
	VersionPolicyStrictMajor = "strict_major"
)

const (
	DeliveryErrorUnsupported           = "a2a.delivery_unsupported"
	DeliveryErrorRetryExhausted        = "a2a.delivery_retry_exhausted"
	DeliveryErrorSSEReconnectExhausted = "a2a.sse_reconnect_exhausted"
	DeliveryErrorVersionMismatch       = "a2a.version_mismatch"
)

const (
	VersionNegotiationCompatible = "compatible"
	VersionNegotiationMismatch   = "mismatch"
	VersionNegotiationUnknown    = "unknown"
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
	TaskID                   string         `json:"task_id,omitempty"`
	WorkflowID               string         `json:"workflow_id,omitempty"`
	TeamID                   string         `json:"team_id,omitempty"`
	StepID                   string         `json:"step_id,omitempty"`
	AgentID                  string         `json:"agent_id"`
	PeerID                   string         `json:"peer_id,omitempty"`
	Method                   string         `json:"method,omitempty"`
	RequiredCapabilities     []string       `json:"required_capabilities,omitempty"`
	Payload                  map[string]any `json:"payload,omitempty"`
	DeliveryMode             string         `json:"delivery_mode,omitempty"`
	DeliveryFallbackUsed     bool           `json:"delivery_fallback_used,omitempty"`
	DeliveryFallbackReason   string         `json:"delivery_fallback_reason,omitempty"`
	VersionLocal             string         `json:"version_local,omitempty"`
	VersionPeer              string         `json:"version_peer,omitempty"`
	VersionNegotiationResult string         `json:"version_negotiation_result,omitempty"`
}

type TaskRecord struct {
	TaskID                   string           `json:"task_id"`
	WorkflowID               string           `json:"workflow_id,omitempty"`
	TeamID                   string           `json:"team_id,omitempty"`
	StepID                   string           `json:"step_id,omitempty"`
	AgentID                  string           `json:"agent_id"`
	PeerID                   string           `json:"peer_id"`
	Status                   TaskStatus       `json:"status"`
	Progress                 map[string]any   `json:"progress,omitempty"`
	Result                   map[string]any   `json:"result,omitempty"`
	ErrorClass               types.ErrorClass `json:"error_class,omitempty"`
	A2AErrorLayer            string           `json:"a2a_error_layer,omitempty"`
	ErrorCode                string           `json:"error_code,omitempty"`
	ErrorMessage             string           `json:"error_message,omitempty"`
	UpdatedAt                time.Time        `json:"updated_at"`
	DeliveryMode             string           `json:"a2a_delivery_mode,omitempty"`
	DeliveryFallbackUsed     bool             `json:"a2a_delivery_fallback_used,omitempty"`
	DeliveryFallbackReason   string           `json:"a2a_delivery_fallback_reason,omitempty"`
	VersionLocal             string           `json:"a2a_version_local,omitempty"`
	VersionPeer              string           `json:"a2a_version_peer,omitempty"`
	VersionNegotiationResult string           `json:"a2a_version_negotiation_result,omitempty"`
}

type AgentCard struct {
	AgentID                string   `json:"agent_id"`
	PeerID                 string   `json:"peer_id"`
	SchemaVersion          string   `json:"schema_version"`
	Endpoint               string   `json:"endpoint,omitempty"`
	Capabilities           []string `json:"capabilities,omitempty"`
	Priority               int      `json:"priority,omitempty"`
	SupportedDeliveryModes []string `json:"supported_delivery_modes,omitempty"`
}

type DeliveryPolicy struct {
	Mode          string
	FallbackMode  string
	CallbackRetry RetryPolicy
	SSEReconnect  RetryPolicy
}

type CardVersionPolicy struct {
	Mode              string
	LocalVersion      string
	MinSupportedMinor int
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
			card.SchemaVersion = "a2a.v1.0"
		}
		card.SchemaVersion = normalizeVersionString(card.SchemaVersion)
		card.Capabilities = normalizeCapabilities(card.Capabilities)
		card.SupportedDeliveryModes = normalizeDeliveryModes(card.SupportedDeliveryModes)
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
	req.WorkflowID = strings.TrimSpace(req.WorkflowID)
	req.TeamID = strings.TrimSpace(req.TeamID)
	req.StepID = strings.TrimSpace(req.StepID)
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
		TaskID:                   req.TaskID,
		WorkflowID:               req.WorkflowID,
		TeamID:                   req.TeamID,
		StepID:                   req.StepID,
		AgentID:                  req.AgentID,
		PeerID:                   req.PeerID,
		Status:                   StatusSubmitted,
		Progress:                 map[string]any{},
		Result:                   map[string]any{},
		UpdatedAt:                s.nowTime(),
		DeliveryMode:             normalizeDeliveryMode(req.DeliveryMode),
		DeliveryFallbackUsed:     req.DeliveryFallbackUsed,
		DeliveryFallbackReason:   strings.TrimSpace(req.DeliveryFallbackReason),
		VersionLocal:             strings.TrimSpace(req.VersionLocal),
		VersionPeer:              strings.TrimSpace(req.VersionPeer),
		VersionNegotiationResult: strings.TrimSpace(req.VersionNegotiationResult),
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
		"phase":         string(types.ActionPhaseRun),
		"status":        string(status),
		"reason":        reason,
		"sequence":      now.UnixNano(),
		"task_id":       record.TaskID,
		"agent_id":      record.AgentID,
		"peer_id":       record.PeerID,
		"delivery_mode": record.DeliveryMode,
		"version_local": record.VersionLocal,
		"version_peer":  record.VersionPeer,
	}
	if strings.TrimSpace(record.WorkflowID) != "" {
		payload["workflow_id"] = strings.TrimSpace(record.WorkflowID)
	}
	if strings.TrimSpace(record.TeamID) != "" {
		payload["team_id"] = strings.TrimSpace(record.TeamID)
	}
	if strings.TrimSpace(record.StepID) != "" {
		payload["step_id"] = strings.TrimSpace(record.StepID)
	}
	if record.DeliveryFallbackUsed {
		payload["delivery_fallback_used"] = true
	}
	if strings.TrimSpace(record.DeliveryFallbackReason) != "" {
		payload["delivery_fallback_reason"] = strings.TrimSpace(record.DeliveryFallbackReason)
	}
	if strings.TrimSpace(record.VersionNegotiationResult) != "" {
		payload["version_negotiation_result"] = strings.TrimSpace(record.VersionNegotiationResult)
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
	Delivery           DeliveryPolicy
	CardVersion        CardVersionPolicy
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
	pending  sync.Map
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
	if strings.TrimSpace(policy.Delivery.Mode) == "" {
		policy.Delivery.Mode = DeliveryModeCallback
	}
	if strings.TrimSpace(policy.Delivery.FallbackMode) == "" {
		policy.Delivery.FallbackMode = DeliveryModeCallback
	}
	if policy.Delivery.CallbackRetry.MaxAttempts <= 0 {
		policy.Delivery.CallbackRetry.MaxAttempts = policy.CallbackRetry.MaxAttempts
	}
	if policy.Delivery.CallbackRetry.Backoff < 0 {
		policy.Delivery.CallbackRetry.Backoff = 0
	}
	if policy.Delivery.SSEReconnect.MaxAttempts <= 0 {
		policy.Delivery.SSEReconnect.MaxAttempts = 3
	}
	if policy.Delivery.SSEReconnect.Backoff < 0 {
		policy.Delivery.SSEReconnect.Backoff = 0
	}
	if strings.TrimSpace(policy.CardVersion.Mode) == "" {
		policy.CardVersion.Mode = VersionPolicyStrictMajor
	}
	if strings.TrimSpace(policy.CardVersion.LocalVersion) == "" {
		policy.CardVersion.LocalVersion = "a2a.v1.0"
	}
	if policy.CardVersion.MinSupportedMinor < 0 {
		policy.CardVersion.MinSupportedMinor = 0
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
	req.TaskID = strings.TrimSpace(req.TaskID)
	req.WorkflowID = strings.TrimSpace(req.WorkflowID)
	req.TeamID = strings.TrimSpace(req.TeamID)
	req.StepID = strings.TrimSpace(req.StepID)
	req.AgentID = strings.TrimSpace(req.AgentID)
	req.PeerID = strings.TrimSpace(req.PeerID)
	if req.TaskID == "" {
		req.TaskID = fmt.Sprintf("a2a-client-%d", c.nowTime().UnixNano())
	}
	req.RequiredCapabilities = normalizeCapabilities(req.RequiredCapabilities)
	card, err := c.resolvePeerCard(req)
	if err != nil {
		return TaskRecord{}, err
	}
	req.PeerID = card.PeerID

	versionLocal, versionPeer, versionResult, err := negotiateCardVersion(
		c.policy.CardVersion,
		card.SchemaVersion,
	)
	if err != nil {
		record := TaskRecord{
			TaskID:                   req.TaskID,
			WorkflowID:               req.WorkflowID,
			TeamID:                   req.TeamID,
			StepID:                   req.StepID,
			AgentID:                  req.AgentID,
			PeerID:                   req.PeerID,
			Status:                   StatusFailed,
			ErrorClass:               types.ErrContext,
			A2AErrorLayer:            string(ErrorLayerSemantic),
			ErrorCode:                DeliveryErrorVersionMismatch,
			ErrorMessage:             err.Error(),
			UpdatedAt:                c.nowTime(),
			VersionLocal:             versionLocal,
			VersionPeer:              versionPeer,
			VersionNegotiationResult: VersionNegotiationMismatch,
		}
		c.emitTimeline(ctx, record, ReasonVersionMismatch)
		return TaskRecord{}, err
	}
	req.VersionLocal = versionLocal
	req.VersionPeer = versionPeer
	req.VersionNegotiationResult = versionResult

	selectedMode, fallbackUsed, fallbackReason, err := negotiateDeliveryMode(
		c.policy.Delivery,
		req.DeliveryMode,
		card.SupportedDeliveryModes,
	)
	if err != nil {
		record := TaskRecord{
			TaskID:                   req.TaskID,
			WorkflowID:               req.WorkflowID,
			TeamID:                   req.TeamID,
			StepID:                   req.StepID,
			AgentID:                  req.AgentID,
			PeerID:                   req.PeerID,
			Status:                   StatusFailed,
			ErrorClass:               types.ErrMCP,
			A2AErrorLayer:            string(ErrorLayerProtocol),
			ErrorCode:                DeliveryErrorUnsupported,
			ErrorMessage:             err.Error(),
			UpdatedAt:                c.nowTime(),
			DeliveryMode:             selectedMode,
			DeliveryFallbackUsed:     fallbackUsed,
			DeliveryFallbackReason:   fallbackReason,
			VersionLocal:             req.VersionLocal,
			VersionPeer:              req.VersionPeer,
			VersionNegotiationResult: req.VersionNegotiationResult,
		}
		if fallbackUsed {
			c.emitTimeline(ctx, record, ReasonDeliveryFallback)
		}
		return TaskRecord{}, err
	}
	req.DeliveryMode = selectedMode
	req.DeliveryFallbackUsed = fallbackUsed
	req.DeliveryFallbackReason = fallbackReason
	if fallbackUsed {
		record := TaskRecord{
			TaskID:                   req.TaskID,
			WorkflowID:               req.WorkflowID,
			TeamID:                   req.TeamID,
			StepID:                   req.StepID,
			AgentID:                  req.AgentID,
			PeerID:                   req.PeerID,
			Status:                   StatusSubmitted,
			UpdatedAt:                c.nowTime(),
			DeliveryMode:             req.DeliveryMode,
			DeliveryFallbackUsed:     req.DeliveryFallbackUsed,
			DeliveryFallbackReason:   req.DeliveryFallbackReason,
			VersionLocal:             req.VersionLocal,
			VersionPeer:              req.VersionPeer,
			VersionNegotiationResult: req.VersionNegotiationResult,
		}
		c.emitTimeline(ctx, record, ReasonDeliveryFallback)
	}

	var lastErr error
	for attempt := 1; attempt <= c.policy.RequestMaxAttempts; attempt++ {
		record, err := c.withTimeout(ctx, func(callCtx context.Context) (TaskRecord, error) {
			return c.server.Submit(callCtx, req)
		})
		if err != nil {
			lastErr = err
			if attempt < c.policy.RequestMaxAttempts && isRetryableTransportError(err) {
				if c.policy.RequestBackoff > 0 {
					time.Sleep(c.policy.RequestBackoff)
				}
				continue
			}
			break
		}
		if strings.TrimSpace(record.DeliveryMode) == "" {
			record.DeliveryMode = req.DeliveryMode
		}
		if strings.TrimSpace(record.WorkflowID) == "" {
			record.WorkflowID = req.WorkflowID
		}
		if strings.TrimSpace(record.TeamID) == "" {
			record.TeamID = req.TeamID
		}
		if strings.TrimSpace(record.StepID) == "" {
			record.StepID = req.StepID
		}
		if strings.TrimSpace(record.VersionLocal) == "" {
			record.VersionLocal = req.VersionLocal
		}
		if strings.TrimSpace(record.VersionPeer) == "" {
			record.VersionPeer = req.VersionPeer
		}
		if strings.TrimSpace(record.VersionNegotiationResult) == "" {
			record.VersionNegotiationResult = req.VersionNegotiationResult
		}
		record.DeliveryFallbackUsed = record.DeliveryFallbackUsed || req.DeliveryFallbackUsed
		if strings.TrimSpace(record.DeliveryFallbackReason) == "" {
			record.DeliveryFallbackReason = req.DeliveryFallbackReason
		}
		c.pending.Store(strings.TrimSpace(record.TaskID), record)
		return record, nil
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
	var (
		sseMode             bool
		sseSubscribed       bool
		sseReconnectAttempt int
		lastRecord          TaskRecord
	)
	if pendingRecord, ok := c.loadPendingTask(taskID); ok {
		lastRecord = pendingRecord
		mode := normalizeDeliveryMode(lastRecord.DeliveryMode)
		sseMode = mode == DeliveryModeSSE
		if sseMode {
			c.emitTimeline(ctx, lastRecord, ReasonSSESubscribe)
			sseSubscribed = true
		}
	}
	for {
		record, err := c.withTimeout(ctx, func(callCtx context.Context) (TaskRecord, error) {
			return c.server.Status(callCtx, taskID)
		})
		if err != nil {
			if sseMode && isRetryableTransportError(err) {
				if sseReconnectAttempt < c.policy.Delivery.SSEReconnect.MaxAttempts {
					sseReconnectAttempt++
					retryRecord := TaskRecord{
						TaskID:                   strings.TrimSpace(taskID),
						WorkflowID:               lastRecord.WorkflowID,
						TeamID:                   lastRecord.TeamID,
						StepID:                   lastRecord.StepID,
						AgentID:                  lastRecord.AgentID,
						PeerID:                   lastRecord.PeerID,
						Status:                   StatusRunning,
						UpdatedAt:                c.nowTime(),
						DeliveryMode:             DeliveryModeSSE,
						VersionLocal:             lastRecord.VersionLocal,
						VersionPeer:              lastRecord.VersionPeer,
						VersionNegotiationResult: lastRecord.VersionNegotiationResult,
					}
					c.emitTimeline(ctx, retryRecord, ReasonSSEReconnect)
					if c.policy.Delivery.SSEReconnect.Backoff > 0 {
						timer := time.NewTimer(c.policy.Delivery.SSEReconnect.Backoff)
						select {
						case <-ctx.Done():
							timer.Stop()
							return TaskRecord{}, ctx.Err()
						case <-timer.C:
						}
					}
					continue
				}
				return TaskRecord{}, newA2AReasonError(
					DeliveryErrorSSEReconnectExhausted,
					ErrorLayerTransport,
					err,
					"a2a sse reconnect exhausted for task %q",
					taskID,
				)
			}
			return TaskRecord{}, err
		}
		lastRecord = record
		c.pending.Store(strings.TrimSpace(lastRecord.TaskID), lastRecord)
		mode := normalizeDeliveryMode(record.DeliveryMode)
		if mode == "" {
			mode = DeliveryModeCallback
		}
		if mode == DeliveryModeSSE {
			sseMode = true
			if !sseSubscribed {
				c.emitTimeline(ctx, record, ReasonSSESubscribe)
				sseSubscribed = true
			}
		}
		if isTerminal(record.Status) {
			if mode == DeliveryModeCallback && callback != nil {
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
			if strings.TrimSpace(result.DeliveryMode) == "" {
				result.DeliveryMode = mode
			}
			if strings.TrimSpace(result.WorkflowID) == "" {
				result.WorkflowID = record.WorkflowID
			}
			if strings.TrimSpace(result.TeamID) == "" {
				result.TeamID = record.TeamID
			}
			if strings.TrimSpace(result.StepID) == "" {
				result.StepID = record.StepID
			}
			if strings.TrimSpace(result.VersionLocal) == "" {
				result.VersionLocal = record.VersionLocal
			}
			if strings.TrimSpace(result.VersionPeer) == "" {
				result.VersionPeer = record.VersionPeer
			}
			if strings.TrimSpace(result.VersionNegotiationResult) == "" {
				result.VersionNegotiationResult = record.VersionNegotiationResult
			}
			c.pending.Delete(strings.TrimSpace(result.TaskID))
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

func (c *Client) loadPendingTask(taskID string) (TaskRecord, bool) {
	if c == nil {
		return TaskRecord{}, false
	}
	value, ok := c.pending.Load(strings.TrimSpace(taskID))
	if !ok {
		return TaskRecord{}, false
	}
	record, ok := value.(TaskRecord)
	if !ok {
		return TaskRecord{}, false
	}
	return record, true
}

func (c *Client) deliverCallback(ctx context.Context, record TaskRecord, callback func(context.Context, TaskRecord) error) error {
	var lastErr error
	maxAttempts := c.policy.Delivery.CallbackRetry.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = c.policy.CallbackRetry.MaxAttempts
	}
	if maxAttempts <= 0 {
		maxAttempts = 3
	}
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := callback(ctx, record)
		if err == nil {
			return nil
		}
		lastErr = err
		if attempt >= maxAttempts {
			break
		}
		c.emitTimeline(ctx, record, ReasonCallbackRetry)
		backoff := c.policy.Delivery.CallbackRetry.Backoff
		if backoff < 0 {
			backoff = 0
		}
		if backoff > 0 {
			timer := time.NewTimer(backoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
	}
	return newA2AReasonError(
		DeliveryErrorRetryExhausted,
		ErrorLayerTransport,
		lastErr,
		"a2a callback retry exhausted for task %q",
		record.TaskID,
	)
}

func (c *Client) emitTimeline(ctx context.Context, record TaskRecord, reason string) {
	if c == nil || c.timeline == nil {
		return
	}
	payload := map[string]any{
		"phase":         string(types.ActionPhaseRun),
		"status":        string(mapToSemanticStatus(record.Status)),
		"reason":        reason,
		"sequence":      c.nowTime().UnixNano(),
		"task_id":       record.TaskID,
		"agent_id":      record.AgentID,
		"peer_id":       record.PeerID,
		"delivery_mode": record.DeliveryMode,
		"version_local": record.VersionLocal,
		"version_peer":  record.VersionPeer,
	}
	if strings.TrimSpace(record.WorkflowID) != "" {
		payload["workflow_id"] = strings.TrimSpace(record.WorkflowID)
	}
	if strings.TrimSpace(record.TeamID) != "" {
		payload["team_id"] = strings.TrimSpace(record.TeamID)
	}
	if strings.TrimSpace(record.StepID) != "" {
		payload["step_id"] = strings.TrimSpace(record.StepID)
	}
	if record.DeliveryFallbackUsed {
		payload["delivery_fallback_used"] = true
	}
	if strings.TrimSpace(record.DeliveryFallbackReason) != "" {
		payload["delivery_fallback_reason"] = strings.TrimSpace(record.DeliveryFallbackReason)
	}
	if strings.TrimSpace(record.VersionNegotiationResult) != "" {
		payload["version_negotiation_result"] = strings.TrimSpace(record.VersionNegotiationResult)
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

func (c *Client) resolvePeerCard(req TaskRequest) (AgentCard, error) {
	if strings.TrimSpace(req.PeerID) == "" {
		card, err := c.router.SelectPeer(c.cards, req.RequiredCapabilities)
		if err != nil {
			return AgentCard{}, err
		}
		return normalizeCard(card), nil
	}
	peerID := strings.TrimSpace(req.PeerID)
	for _, card := range c.cards {
		if strings.EqualFold(strings.TrimSpace(card.PeerID), peerID) {
			return normalizeCard(card), nil
		}
	}
	return normalizeCard(AgentCard{
		AgentID:       strings.TrimSpace(req.AgentID),
		PeerID:        peerID,
		SchemaVersion: c.policy.CardVersion.LocalVersion,
		SupportedDeliveryModes: []string{
			normalizeDeliveryMode(c.policy.Delivery.Mode),
			normalizeDeliveryMode(c.policy.Delivery.FallbackMode),
		},
	}), nil
}

func normalizeCard(card AgentCard) AgentCard {
	card.AgentID = strings.TrimSpace(card.AgentID)
	card.PeerID = strings.TrimSpace(card.PeerID)
	card.SchemaVersion = normalizeVersionString(card.SchemaVersion)
	if card.SchemaVersion == "" {
		card.SchemaVersion = "a2a.v1.0"
	}
	card.Capabilities = normalizeCapabilities(card.Capabilities)
	card.SupportedDeliveryModes = normalizeDeliveryModes(card.SupportedDeliveryModes)
	if len(card.SupportedDeliveryModes) == 0 {
		card.SupportedDeliveryModes = []string{DeliveryModeCallback}
	}
	return card
}

func negotiateDeliveryMode(policy DeliveryPolicy, requested string, peerSupported []string) (string, bool, string, error) {
	preferred := normalizeDeliveryMode(requested)
	if preferred == "" {
		preferred = normalizeDeliveryMode(policy.Mode)
	}
	fallback := normalizeDeliveryMode(policy.FallbackMode)
	if preferred == "" {
		preferred = DeliveryModeCallback
	}
	if fallback == "" {
		fallback = DeliveryModeCallback
	}
	peerModes := normalizeDeliveryModes(peerSupported)
	if len(peerModes) == 0 {
		peerModes = []string{DeliveryModeCallback}
	}
	if contains(peerModes, preferred) {
		return preferred, false, "", nil
	}
	if contains(peerModes, fallback) {
		return fallback, true, DeliveryErrorUnsupported, nil
	}
	return preferred, false, "", newA2AReasonError(
		DeliveryErrorUnsupported,
		ErrorLayerProtocol,
		nil,
		"a2a delivery unsupported: preferred=%q fallback=%q peer_supported=%v",
		preferred,
		fallback,
		peerModes,
	)
}

func negotiateCardVersion(policy CardVersionPolicy, peerVersionRaw string) (string, string, string, error) {
	mode := strings.ToLower(strings.TrimSpace(policy.Mode))
	if mode == "" {
		mode = VersionPolicyStrictMajor
	}
	if mode != VersionPolicyStrictMajor {
		return "", "", VersionNegotiationMismatch, newA2AReasonError(
			DeliveryErrorVersionMismatch,
			ErrorLayerSemantic,
			nil,
			"a2a card version policy %q is not supported",
			policy.Mode,
		)
	}
	localRaw := normalizeVersionString(policy.LocalVersion)
	if localRaw == "" {
		localRaw = "a2a.v1.0"
	}
	peerRaw := normalizeVersionString(peerVersionRaw)
	if peerRaw == "" {
		peerRaw = "a2a.v1.0"
	}
	local, err := parseCardVersion(localRaw)
	if err != nil {
		return localRaw, peerRaw, VersionNegotiationMismatch, newA2AReasonError(
			DeliveryErrorVersionMismatch,
			ErrorLayerSemantic,
			err,
			"a2a local card version %q is invalid",
			localRaw,
		)
	}
	peer, err := parseCardVersion(peerRaw)
	if err != nil {
		return localRaw, peerRaw, VersionNegotiationMismatch, newA2AReasonError(
			DeliveryErrorVersionMismatch,
			ErrorLayerSemantic,
			err,
			"a2a peer card version %q is invalid",
			peerRaw,
		)
	}
	if local.major != peer.major {
		return localRaw, peerRaw, VersionNegotiationMismatch, newA2AReasonError(
			DeliveryErrorVersionMismatch,
			ErrorLayerSemantic,
			nil,
			"a2a card major version mismatch local=%s peer=%s",
			localRaw,
			peerRaw,
		)
	}
	if peer.minor < policy.MinSupportedMinor {
		return localRaw, peerRaw, VersionNegotiationMismatch, newA2AReasonError(
			DeliveryErrorVersionMismatch,
			ErrorLayerSemantic,
			nil,
			"a2a peer card minor version %d is below min_supported_minor %d",
			peer.minor,
			policy.MinSupportedMinor,
		)
	}
	return localRaw, peerRaw, VersionNegotiationCompatible, nil
}

type parsedCardVersion struct {
	major int
	minor int
}

func parseCardVersion(raw string) (parsedCardVersion, error) {
	in := normalizeVersionString(raw)
	if in == "" {
		return parsedCardVersion{}, errors.New("empty version")
	}
	in = strings.TrimPrefix(in, "a2a.")
	in = strings.TrimPrefix(in, "v")
	parts := strings.Split(in, ".")
	if len(parts) == 1 {
		var major int
		if _, err := fmt.Sscanf(parts[0], "%d", &major); err != nil {
			return parsedCardVersion{}, err
		}
		return parsedCardVersion{major: major, minor: 0}, nil
	}
	var major, minor int
	if _, err := fmt.Sscanf(parts[0], "%d", &major); err != nil {
		return parsedCardVersion{}, err
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &minor); err != nil {
		return parsedCardVersion{}, err
	}
	return parsedCardVersion{major: major, minor: minor}, nil
}

type a2aReasonError struct {
	Code    string
	Layer   ErrorLayer
	Message string
	Cause   error
}

func (e *a2aReasonError) Error() string {
	if e == nil {
		return ""
	}
	msg := strings.TrimSpace(e.Message)
	if msg == "" && e.Cause != nil {
		msg = e.Cause.Error()
	}
	if msg == "" {
		return strings.TrimSpace(e.Code)
	}
	return msg
}

func (e *a2aReasonError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func newA2AReasonError(code string, layer ErrorLayer, cause error, format string, args ...any) error {
	return &a2aReasonError{
		Code:    strings.TrimSpace(code),
		Layer:   layer,
		Message: fmt.Sprintf(format, args...),
		Cause:   cause,
	}
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

func normalizeVersionString(in string) string {
	raw := strings.ToLower(strings.TrimSpace(in))
	if raw == "" {
		return ""
	}
	if !strings.HasPrefix(raw, "a2a.") && !strings.HasPrefix(raw, "v") {
		raw = "v" + raw
	}
	if strings.HasPrefix(raw, "v") {
		raw = "a2a." + raw
	}
	return raw
}

func normalizeDeliveryMode(in string) string {
	mode := strings.ToLower(strings.TrimSpace(in))
	switch mode {
	case DeliveryModeCallback, DeliveryModeSSE:
		return mode
	default:
		return ""
	}
}

func normalizeDeliveryModes(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, raw := range in {
		mode := normalizeDeliveryMode(raw)
		if mode == "" {
			continue
		}
		if _, ok := seen[mode]; ok {
			continue
		}
		seen[mode] = struct{}{}
		out = append(out, mode)
	}
	sort.Strings(out)
	return out
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item), strings.TrimSpace(target)) {
			return true
		}
	}
	return false
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
	var reasonErr *a2aReasonError
	if errors.As(err, &reasonErr) {
		code := strings.TrimSpace(reasonErr.Code)
		layer := reasonErr.Layer
		switch code {
		case DeliveryErrorVersionMismatch:
			return types.ErrContext, ErrorLayerSemantic, code
		case DeliveryErrorUnsupported:
			return types.ErrMCP, ErrorLayerProtocol, code
		case DeliveryErrorRetryExhausted, DeliveryErrorSSEReconnectExhausted:
			return types.ErrMCP, ErrorLayerTransport, code
		}
		if layer == "" {
			layer = ErrorLayerProtocol
		}
		return types.ErrMCP, layer, code
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return types.ErrPolicyTimeout, ErrorLayerTransport, "timeout"
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case strings.Contains(message, "a2a.version_mismatch"), strings.Contains(message, "version mismatch"):
		return types.ErrContext, ErrorLayerSemantic, DeliveryErrorVersionMismatch
	case strings.Contains(message, "a2a.delivery_unsupported"), strings.Contains(message, "delivery unsupported"):
		return types.ErrMCP, ErrorLayerProtocol, DeliveryErrorUnsupported
	case strings.Contains(message, "a2a.delivery_retry_exhausted"), strings.Contains(message, "callback retry exhausted"):
		return types.ErrMCP, ErrorLayerTransport, DeliveryErrorRetryExhausted
	case strings.Contains(message, "a2a.sse_reconnect_exhausted"), strings.Contains(message, "sse reconnect exhausted"):
		return types.ErrMCP, ErrorLayerTransport, DeliveryErrorSSEReconnectExhausted
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
	A2ATaskTotal                int    `json:"a2a_task_total"`
	A2ATaskFailed               int    `json:"a2a_task_failed"`
	PeerID                      string `json:"peer_id,omitempty"`
	A2AErrorLayer               string `json:"a2a_error_layer,omitempty"`
	A2ADeliveryMode             string `json:"a2a_delivery_mode,omitempty"`
	A2ADeliveryFallbackUsed     bool   `json:"a2a_delivery_fallback_used,omitempty"`
	A2ADeliveryFallbackReason   string `json:"a2a_delivery_fallback_reason,omitempty"`
	A2AVersionLocal             string `json:"a2a_version_local,omitempty"`
	A2AVersionPeer              string `json:"a2a_version_peer,omitempty"`
	A2AVersionNegotiationResult string `json:"a2a_version_negotiation_result,omitempty"`
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
		if out.A2ADeliveryMode == "" && strings.TrimSpace(task.DeliveryMode) != "" {
			out.A2ADeliveryMode = strings.TrimSpace(task.DeliveryMode)
		}
		if task.DeliveryFallbackUsed {
			out.A2ADeliveryFallbackUsed = true
			if out.A2ADeliveryFallbackReason == "" && strings.TrimSpace(task.DeliveryFallbackReason) != "" {
				out.A2ADeliveryFallbackReason = strings.TrimSpace(task.DeliveryFallbackReason)
			}
		}
		if out.A2AVersionLocal == "" && strings.TrimSpace(task.VersionLocal) != "" {
			out.A2AVersionLocal = strings.TrimSpace(task.VersionLocal)
		}
		if out.A2AVersionPeer == "" && strings.TrimSpace(task.VersionPeer) != "" {
			out.A2AVersionPeer = strings.TrimSpace(task.VersionPeer)
		}
		if out.A2AVersionNegotiationResult == "" && strings.TrimSpace(task.VersionNegotiationResult) != "" {
			out.A2AVersionNegotiationResult = strings.TrimSpace(task.VersionNegotiationResult)
		}
	}
	return out
}
