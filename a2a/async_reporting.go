package a2a

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

const (
	AsyncReportSinkCallback = "callback"
	AsyncReportSinkChannel  = "channel"
)

type AsyncReportingPolicy struct {
	Enabled     bool
	Sink        string
	Retry       AsyncReportingRetryPolicy
	JitterRatio float64
	SinkImpl    ReportSink
}

type AsyncReportingRetryPolicy struct {
	MaxAttempts    int
	BackoffInitial time.Duration
	BackoffMax     time.Duration
}

type AsyncSubmitAck struct {
	TaskID     string    `json:"task_id"`
	WorkflowID string    `json:"workflow_id,omitempty"`
	TeamID     string    `json:"team_id,omitempty"`
	StepID     string    `json:"step_id,omitempty"`
	AgentID    string    `json:"agent_id,omitempty"`
	PeerID     string    `json:"peer_id,omitempty"`
	AcceptedAt time.Time `json:"accepted_at"`
}

type AsyncReport struct {
	ReportKey        string           `json:"report_key"`
	OutcomeKey       string           `json:"outcome_key,omitempty"`
	WorkflowID       string           `json:"workflow_id,omitempty"`
	TeamID           string           `json:"team_id,omitempty"`
	StepID           string           `json:"step_id,omitempty"`
	TaskID           string           `json:"task_id"`
	AttemptID        string           `json:"attempt_id,omitempty"`
	AgentID          string           `json:"agent_id,omitempty"`
	PeerID           string           `json:"peer_id,omitempty"`
	Status           TaskStatus       `json:"status"`
	Result           map[string]any   `json:"result,omitempty"`
	ErrorClass       types.ErrorClass `json:"error_class,omitempty"`
	ErrorLayer       string           `json:"error_layer,omitempty"`
	ErrorCode        string           `json:"error_code,omitempty"`
	ErrorMessage     string           `json:"error_message,omitempty"`
	DeliveryAttempt  int              `json:"delivery_attempt,omitempty"`
	DeliveryError    string           `json:"delivery_error,omitempty"`
	BusinessTerminal bool             `json:"business_terminal"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

type ReportSink interface {
	Deliver(ctx context.Context, report AsyncReport) error
}

type ReportSinkFunc func(context.Context, AsyncReport) error

func (f ReportSinkFunc) Deliver(ctx context.Context, report AsyncReport) error {
	if f == nil {
		return errors.New("a2a async report sink is nil")
	}
	return f(ctx, report)
}

type ChannelReportSink struct {
	ch chan AsyncReport
}

func NewChannelReportSink(buffer int) *ChannelReportSink {
	if buffer <= 0 {
		buffer = 16
	}
	return &ChannelReportSink{ch: make(chan AsyncReport, buffer)}
}

func (s *ChannelReportSink) Deliver(ctx context.Context, report AsyncReport) error {
	if s == nil || s.ch == nil {
		return errors.New("a2a channel report sink is nil")
	}
	select {
	case s.ch <- report:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *ChannelReportSink) Channel() <-chan AsyncReport {
	if s == nil {
		return nil
	}
	return s.ch
}

type CallbackReportSink struct {
	callback func(context.Context, AsyncReport) error
}

func NewCallbackReportSink(callback func(context.Context, AsyncReport) error) *CallbackReportSink {
	return &CallbackReportSink{callback: callback}
}

func (s *CallbackReportSink) Deliver(ctx context.Context, report AsyncReport) error {
	if s == nil || s.callback == nil {
		return errors.New("a2a callback report sink is nil")
	}
	return s.callback(ctx, report)
}

type AsyncReportDeliveryError struct {
	Cause     error
	Retryable bool
}

func (e *AsyncReportDeliveryError) Error() string {
	if e == nil || e.Cause == nil {
		return "a2a async report delivery error"
	}
	return strings.TrimSpace(e.Cause.Error())
}

func (e *AsyncReportDeliveryError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func normalizeAsyncReportingPolicy(policy AsyncReportingPolicy) AsyncReportingPolicy {
	out := policy
	out.Sink = strings.ToLower(strings.TrimSpace(out.Sink))
	if out.Sink == "" {
		out.Sink = AsyncReportSinkCallback
	}
	switch out.Sink {
	case AsyncReportSinkCallback, AsyncReportSinkChannel:
	default:
		out.Sink = AsyncReportSinkCallback
	}
	if out.Retry.MaxAttempts <= 0 {
		out.Retry.MaxAttempts = 3
	}
	if out.Retry.BackoffInitial < 0 {
		out.Retry.BackoffInitial = 0
	}
	if out.Retry.BackoffInitial == 0 {
		out.Retry.BackoffInitial = 50 * time.Millisecond
	}
	if out.Retry.BackoffMax < out.Retry.BackoffInitial {
		out.Retry.BackoffMax = out.Retry.BackoffInitial
	}
	if out.Retry.BackoffMax == 0 {
		out.Retry.BackoffMax = 500 * time.Millisecond
	}
	if out.JitterRatio < 0 {
		out.JitterRatio = 0
	}
	if out.JitterRatio > 1 {
		out.JitterRatio = 1
	}
	if out.JitterRatio == 0 {
		out.JitterRatio = 0.2
	}
	return out
}

func IsRetryableReportDeliveryError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var asyncErr *AsyncReportDeliveryError
	if errors.As(err, &asyncErr) {
		return asyncErr.Retryable
	}
	return true
}

func (c *Client) SubmitAsync(ctx context.Context, req TaskRequest, sink ReportSink) (AsyncSubmitAck, error) {
	if c == nil {
		return AsyncSubmitAck{}, errors.New("a2a client is nil")
	}
	if c.server == nil {
		return AsyncSubmitAck{}, errors.New("a2a client server is not configured")
	}
	effectiveSink, err := c.resolveAsyncReportSink(sink)
	if err != nil {
		return AsyncSubmitAck{}, err
	}
	submitted, err := c.Submit(ctx, req)
	if err != nil {
		return AsyncSubmitAck{}, err
	}
	ack := AsyncSubmitAck{
		TaskID:     strings.TrimSpace(submitted.TaskID),
		WorkflowID: strings.TrimSpace(submitted.WorkflowID),
		TeamID:     strings.TrimSpace(submitted.TeamID),
		StepID:     strings.TrimSpace(submitted.StepID),
		AgentID:    strings.TrimSpace(submitted.AgentID),
		PeerID:     strings.TrimSpace(submitted.PeerID),
		AcceptedAt: c.nowTime(),
	}
	c.emitTimelineWithExtras(ctx, submitted, ReasonAsyncSubmit, map[string]any{
		"report_sink": c.policy.AsyncReporting.Sink,
	})
	go c.awaitAndDeliverAsyncReport(context.Background(), strings.TrimSpace(submitted.TaskID), effectiveSink)
	return ack, nil
}

func (c *Client) resolveAsyncReportSink(sink ReportSink) (ReportSink, error) {
	if sink != nil {
		return sink, nil
	}
	if c != nil && c.policy.AsyncReporting.SinkImpl != nil {
		return c.policy.AsyncReporting.SinkImpl, nil
	}
	return nil, errors.New("a2a async reporting requires report sink")
}

func (c *Client) awaitAndDeliverAsyncReport(ctx context.Context, taskID string, sink ReportSink) {
	if c == nil || strings.TrimSpace(taskID) == "" || sink == nil {
		return
	}
	record, err := c.WaitResult(ctx, taskID, 20*time.Millisecond, nil)
	if err != nil {
		class, layer, code := ClassifyError(err)
		record = TaskRecord{
			TaskID:        strings.TrimSpace(taskID),
			Status:        StatusFailed,
			ErrorClass:    class,
			A2AErrorLayer: strings.TrimSpace(string(layer)),
			ErrorCode:     strings.TrimSpace(code),
			ErrorMessage:  strings.TrimSpace(err.Error()),
			UpdatedAt:     c.nowTime(),
		}
	}
	report := BuildAsyncReport(record)
	c.deliverAsyncReport(ctx, sink, report)
}

func BuildAsyncReport(record TaskRecord) AsyncReport {
	out := AsyncReport{
		WorkflowID:       strings.TrimSpace(record.WorkflowID),
		TeamID:           strings.TrimSpace(record.TeamID),
		StepID:           strings.TrimSpace(record.StepID),
		TaskID:           strings.TrimSpace(record.TaskID),
		AttemptID:        strings.TrimSpace(extractAttemptID(record)),
		AgentID:          strings.TrimSpace(record.AgentID),
		PeerID:           strings.TrimSpace(record.PeerID),
		Status:           record.Status,
		Result:           copyMap(record.Result),
		ErrorClass:       record.ErrorClass,
		ErrorLayer:       strings.TrimSpace(record.A2AErrorLayer),
		ErrorCode:        strings.TrimSpace(record.ErrorCode),
		ErrorMessage:     strings.TrimSpace(record.ErrorMessage),
		BusinessTerminal: isTerminal(record.Status),
		UpdatedAt:        record.UpdatedAt,
		OutcomeKey:       deriveAsyncOutcomeKey(record),
	}
	if out.UpdatedAt.IsZero() {
		out.UpdatedAt = time.Now()
	}
	out.ReportKey = BuildAsyncReportKey(out)
	return out
}

func BuildAsyncReportKey(report AsyncReport) string {
	parts := []string{
		strings.TrimSpace(report.WorkflowID),
		strings.TrimSpace(report.TaskID),
		strings.TrimSpace(report.AttemptID),
		strings.TrimSpace(string(report.Status)),
		strings.TrimSpace(report.OutcomeKey),
	}
	return strings.Join(parts, "|")
}

func (c *Client) deliverAsyncReport(ctx context.Context, sink ReportSink, report AsyncReport) {
	if c == nil || sink == nil {
		return
	}
	key := strings.TrimSpace(report.ReportKey)
	if key == "" {
		key = BuildAsyncReportKey(report)
	}
	if _, ok := c.asyncDelivered.Load(key); ok {
		c.emitAsyncReportTimeline(ctx, report, ReasonAsyncReportDedup, map[string]any{
			"report_key": key,
		})
		return
	}
	retryPolicy := c.policy.AsyncReporting.Retry
	if retryPolicy.MaxAttempts <= 0 {
		retryPolicy.MaxAttempts = 1
	}
	lastErr := error(nil)
	for attempt := 1; attempt <= retryPolicy.MaxAttempts; attempt++ {
		reportAttempt := report
		reportAttempt.DeliveryAttempt = attempt
		err := sink.Deliver(ctx, reportAttempt)
		if err == nil {
			c.asyncDelivered.Store(key, struct{}{})
			c.emitAsyncReportTimeline(ctx, reportAttempt, ReasonAsyncReportDeliver, map[string]any{
				"report_key": key,
			})
			return
		}
		lastErr = err
		reportAttempt.DeliveryError = strings.TrimSpace(err.Error())
		if attempt >= retryPolicy.MaxAttempts || !IsRetryableReportDeliveryError(err) {
			c.emitAsyncReportTimeline(ctx, reportAttempt, ReasonAsyncReportDrop, map[string]any{
				"report_key":          key,
				"report_drop_reason":  classifyAsyncDropReason(err),
				"report_delivery_err": strings.TrimSpace(err.Error()),
			})
			return
		}
		c.emitAsyncReportTimeline(ctx, reportAttempt, ReasonAsyncReportRetry, map[string]any{
			"report_key": key,
		})
		delay := c.retryDelayForAsyncReport(attempt)
		if delay <= 0 {
			continue
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			c.emitAsyncReportTimeline(ctx, reportAttempt, ReasonAsyncReportDrop, map[string]any{
				"report_key":          key,
				"report_drop_reason":  "context_canceled",
				"report_delivery_err": ctx.Err().Error(),
			})
			return
		case <-timer.C:
		}
	}
	if lastErr != nil {
		report.DeliveryError = strings.TrimSpace(lastErr.Error())
	}
	c.emitAsyncReportTimeline(ctx, report, ReasonAsyncReportDrop, map[string]any{
		"report_key":          key,
		"report_drop_reason":  "retry_exhausted",
		"report_delivery_err": report.DeliveryError,
	})
}

func (c *Client) retryDelayForAsyncReport(attempt int) time.Duration {
	if c == nil {
		return 0
	}
	policy := normalizeAsyncReportingPolicy(c.policy.AsyncReporting)
	if attempt <= 1 {
		return policy.Retry.BackoffInitial
	}
	delay := policy.Retry.BackoffInitial
	for i := 1; i < attempt; i++ {
		delay *= 2
		if delay >= policy.Retry.BackoffMax {
			delay = policy.Retry.BackoffMax
			break
		}
	}
	if policy.JitterRatio <= 0 {
		return delay
	}
	jitterWindow := int64(float64(delay) * policy.JitterRatio)
	if jitterWindow <= 0 {
		return delay
	}
	seed := time.Now().UnixNano() % (2*jitterWindow + 1)
	shifted := int64(delay) + (seed - jitterWindow)
	if shifted < 0 {
		shifted = 0
	}
	if shifted > int64(policy.Retry.BackoffMax) {
		shifted = int64(policy.Retry.BackoffMax)
	}
	return time.Duration(shifted)
}

func (c *Client) emitAsyncReportTimeline(ctx context.Context, report AsyncReport, reason string, extras map[string]any) {
	record := TaskRecord{
		TaskID:        strings.TrimSpace(report.TaskID),
		WorkflowID:    strings.TrimSpace(report.WorkflowID),
		TeamID:        strings.TrimSpace(report.TeamID),
		StepID:        strings.TrimSpace(report.StepID),
		AgentID:       strings.TrimSpace(report.AgentID),
		PeerID:        strings.TrimSpace(report.PeerID),
		Status:        report.Status,
		Result:        copyMap(report.Result),
		ErrorClass:    report.ErrorClass,
		A2AErrorLayer: strings.TrimSpace(report.ErrorLayer),
		ErrorCode:     strings.TrimSpace(report.ErrorCode),
		ErrorMessage:  strings.TrimSpace(report.ErrorMessage),
		UpdatedAt:     c.nowTime(),
	}
	merged := map[string]any{
		"report_key":              strings.TrimSpace(report.ReportKey),
		"report_delivery_attempt": report.DeliveryAttempt,
	}
	if attemptID := strings.TrimSpace(report.AttemptID); attemptID != "" {
		merged["attempt_id"] = attemptID
	}
	if outcomeKey := strings.TrimSpace(report.OutcomeKey); outcomeKey != "" {
		merged["outcome_key"] = outcomeKey
	}
	for key, value := range extras {
		merged[key] = value
	}
	c.emitTimelineWithExtras(ctx, record, reason, merged)
}

func deriveAsyncOutcomeKey(record TaskRecord) string {
	if record.Status == StatusSucceeded {
		keys := make([]string, 0, len(record.Result))
		for key := range record.Result {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		builder := strings.Builder{}
		builder.WriteString("succeeded")
		for _, key := range keys {
			builder.WriteString("|")
			builder.WriteString(key)
		}
		return builder.String()
	}
	if strings.TrimSpace(record.ErrorMessage) == "" {
		return strings.TrimSpace(string(record.Status))
	}
	return strings.TrimSpace(string(record.Status)) + "|" + strings.TrimSpace(record.ErrorMessage)
}

func extractAttemptID(record TaskRecord) string {
	if id := strings.TrimSpace(record.AttemptID); id != "" {
		return id
	}
	if value, ok := record.Progress["attempt_id"]; ok {
		if id, ok := value.(string); ok {
			return strings.TrimSpace(id)
		}
	}
	if value, ok := record.Result["attempt_id"]; ok {
		if id, ok := value.(string); ok {
			return strings.TrimSpace(id)
		}
	}
	return ""
}

func classifyAsyncDropReason(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.Canceled) {
		return "context_canceled"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	var asyncErr *AsyncReportDeliveryError
	if errors.As(err, &asyncErr) && !asyncErr.Retryable {
		return "non_retryable"
	}
	return "retry_exhausted"
}
