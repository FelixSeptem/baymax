// Package host provides a transport-neutral embedded host coordinator.
package host

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

const (
	defaultPendingLimit      = 128
	defaultCorrelationLimit  = 4096
	defaultDeliveryQueueSize = 32
	defaultDeliveryTimeout   = 5 * time.Second

	HostRequestTypeClarification       = "clarification"
	HostRequestTypeActionGate          = "action_gate"
	HostReasonCorrelationLimit         = "host.correlation_limit"
	HostReasonDeliverySourcePolicyDrop = "host.delivery.source_policy_drop_with_record"
)

var (
	ErrConnectionClosed   = errors.New("host: connection closed")
	ErrPendingNotFound    = errors.New("host: pending request not found")
	ErrPendingCorrelation = errors.New("host: pending response correlation mismatch")
	ErrPendingLimit       = errors.New("host: pending request limit reached")
	ErrDeliveryTimeout    = errors.New("host: delivery timed out")
	ErrDeliveryFull       = errors.New("host: delivery queue full")
)

// SourceControl delegates lifecycle and Realtime mutation to the source.
type SourceControl interface {
	ExecuteAction(context.Context, types.HostCommandEnvelope, types.ProtocolAction) (types.HostAdmissionStatus, string, error)
	IngestRealtime(context.Context, types.HostCommandEnvelope, types.RealtimeEventEnvelope) (types.HostAdmissionStatus, string, error)
}

// EventSubscriber delegates cursor recovery and live-tail ownership.
type EventSubscriber interface {
	SubscribeHostEvents(context.Context, types.EventStreamSubscription) (types.EventStreamBindingProjection, error)
}

// TerminalQuery delegates authoritative terminal recovery.
type TerminalQuery interface {
	QueryTerminal(context.Context, string, string) (*types.TerminalOutcome, error)
}

// ReadinessChecker is an admission hook. It must not mutate source state.
type ReadinessChecker interface {
	CheckHostReadiness(context.Context, types.HostCommandEnvelope) error
}

// Authorizer is an admission hook distinct from advertised availability.
type Authorizer interface {
	AuthorizeHostCommand(context.Context, types.HostCommandEnvelope) error
}

// EventSink is the optional standard event sink used for bounded host facts.
// Implementations may be types.EventHandler or observability/event.RuntimeRecorder.
type EventSink interface {
	OnEvent(context.Context, types.Event)
}

// ControlSource retains the original Engine active-control shape.
type ControlSource interface {
	CancelRun(runID, sessionID string) (types.HostAdmissionStatus, error)
	IngestRealtime(runID, sessionID string, ev types.RealtimeEventEnvelope) (types.HostAdmissionStatus, error)
}

type Coordinator struct {
	runner           types.Runner
	starter          types.HostRunStarter
	control          SourceControl
	subscriber       EventSubscriber
	terminal         TerminalQuery
	readiness        ReadinessChecker
	authorization    Authorizer
	sink             EventSink
	now              func() time.Time
	pendingLimit     int
	correlationLimit int
	deliverySize     int
	deliveryWait     time.Duration
	nextMessageID    atomic.Uint64
}

type Option func(*Coordinator)

func WithClock(now func() time.Time) Option {
	return func(c *Coordinator) {
		if now != nil {
			c.now = now
		}
	}
}
func WithRunStarter(source types.HostRunStarter) Option {
	return func(c *Coordinator) { c.starter = source }
}
func WithSourceControl(source SourceControl) Option {
	return func(c *Coordinator) { c.control = source }
}
func WithEventSubscription(source EventSubscriber) Option {
	return func(c *Coordinator) { c.subscriber = source }
}
func WithTerminalQuery(source TerminalQuery) Option {
	return func(c *Coordinator) { c.terminal = source }
}
func WithReadiness(source ReadinessChecker) Option {
	return func(c *Coordinator) { c.readiness = source }
}
func WithAuthorization(source Authorizer) Option {
	return func(c *Coordinator) { c.authorization = source }
}
func WithEventSink(sink EventSink) Option {
	return func(c *Coordinator) { c.sink = sink }
}
func WithPendingLimit(limit int) Option {
	return func(c *Coordinator) {
		if limit > 0 {
			c.pendingLimit = limit
		}
	}
}
func WithCommandCorrelationLimit(limit int) Option {
	return func(c *Coordinator) {
		if limit > 0 {
			c.correlationLimit = limit
		}
	}
}
func WithDeliveryQueueSize(size int) Option {
	return func(c *Coordinator) {
		if size > 0 {
			c.deliverySize = size
		}
	}
}
func WithDeliveryTimeout(timeout time.Duration) Option {
	return func(c *Coordinator) {
		if timeout > 0 {
			c.deliveryWait = timeout
		}
	}
}

func New(runner types.Runner, opts ...Option) *Coordinator {
	c := &Coordinator{runner: runner, now: time.Now, pendingLimit: defaultPendingLimit, correlationLimit: defaultCorrelationLimit, deliverySize: defaultDeliveryQueueSize, deliveryWait: defaultDeliveryTimeout}
	if starter, ok := runner.(types.HostRunStarter); ok {
		c.starter = starter
	}
	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}
	return c
}

type pendingRequest struct {
	response  chan types.HostResponseEnvelope
	settled   chan pendingOutcome
	sessionID string
	runID     string
}

type pendingOutcome struct {
	response types.HostResponseEnvelope
	err      error
}

type delivery struct {
	frame  any
	result chan error
	ctx    context.Context
	cancel context.CancelFunc
}

// FrameWriter is the production transport boundary. WriteFrame must stop and
// return when ctx is canceled, and Close must unblock any active write.
// A nil error means the complete frame is externally visible.
type FrameWriter interface {
	WriteFrame(context.Context, any) error
	Close() error
}

type FrameWriterFunc func(context.Context, any) error

func (f FrameWriterFunc) WriteFrame(ctx context.Context, frame any) error {
	return f(ctx, frame)
}

func (FrameWriterFunc) Close() error { return nil }

type legacyFrameWriter struct{ write func(any) error }

func (w legacyFrameWriter) WriteFrame(_ context.Context, frame any) error {
	if w.write == nil {
		return ErrConnectionClosed
	}
	return w.write(frame)
}

func (legacyFrameWriter) Close() error { return nil }

// commandResponseBarrier keeps source events produced during command
// admission behind that command's response. It is connection-local delivery
// ordering only; it does not own source event or terminal state.
type commandResponseBarrier struct {
	mu      sync.Mutex
	open    bool
	limit   int
	pending []deferredCommandDelivery
}

type deferredCommandDelivery struct {
	commit func() error
	abort  func()
}

func (b *commandResponseBarrier) deferDelivery(item deferredCommandDelivery) (bool, error) {
	if b == nil {
		return false, nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.open {
		return false, nil
	}
	if b.limit > 0 && len(b.pending) >= b.limit {
		return false, ErrDeliveryFull
	}
	b.pending = append(b.pending, item)
	return true, nil
}

func (b *commandResponseBarrier) commit() {
	if b == nil {
		return
	}
	for {
		b.mu.Lock()
		if len(b.pending) == 0 {
			b.open = true
			b.mu.Unlock()
			return
		}
		item := b.pending[0]
		b.pending[0] = deferredCommandDelivery{}
		b.pending = b.pending[1:]
		b.mu.Unlock()
		if item.commit != nil {
			if err := item.commit(); err == nil {
				continue
			}
		}
		if item.abort != nil {
			item.abort()
		}
		if item.commit != nil {
			b.abort()
			return
		}
	}
}

func (b *commandResponseBarrier) abort() {
	if b == nil {
		return
	}
	b.mu.Lock()
	pending := b.pending
	b.pending = nil
	b.open = true
	b.mu.Unlock()
	for _, item := range pending {
		if item.abort != nil {
			item.abort()
		}
	}
}

type Connection struct {
	coord               *Coordinator
	writer              FrameWriter
	interruptibleWriter bool
	writeCtx            context.Context
	cancelWrite         context.CancelCauseFunc
	mu                  sync.Mutex
	closed              bool
	pending             map[string]*pendingRequest
	seen                map[string]struct{}
	barriers            map[string]*commandResponseBarrier
	done                chan struct{}
	queue               chan delivery
	close               sync.Once
}

// Connect is the legacy callback adapter. The callback cannot be forcibly
// interrupted, so production bindings that require deterministic timeout and
// Close behavior must use ConnectFrameWriter.
func (c *Coordinator) Connect(write func(any) error) *Connection {
	return c.connectFrameWriter(legacyFrameWriter{write: write}, false)
}

func (c *Coordinator) ConnectFrameWriter(writer FrameWriter) *Connection {
	return c.connectFrameWriter(writer, true)
}

func (c *Coordinator) connectFrameWriter(writer FrameWriter, interruptible bool) *Connection {
	writeCtx, cancelWrite := context.WithCancelCause(context.Background())
	conn := &Connection{coord: c, writer: writer, interruptibleWriter: interruptible, writeCtx: writeCtx, cancelWrite: cancelWrite, pending: make(map[string]*pendingRequest), seen: make(map[string]struct{}), barriers: make(map[string]*commandResponseBarrier), done: make(chan struct{}), queue: make(chan delivery, c.deliverySize)}
	go conn.writeLoop()
	return conn
}

func (c *Connection) PendingCount() int {
	if c == nil {
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.pending)
}

func (c *Connection) Close() {
	c.closeWithCause(ErrConnectionClosed)
}

func boundedFact(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 96 {
		return value[:96]
	}
	return value
}

// emitFact sends only bounded, host-owned classification fields. In
// particular it never forwards command payloads, cursors, frame bodies, or
// transport errors beyond a bounded reason string.
func (c *Connection) emitFact(ctx context.Context, eventType, runID string, payload map[string]any) {
	if c == nil || c.coord == nil || c.coord.sink == nil {
		return
	}
	safe := map[string]any{}
	for _, key := range []string{"fact", "session_id", "request_id", "message_id", "admission_status", "reason_code", "source_control", "pending_outcome", "delivery_status"} {
		if value, ok := payload[key]; ok {
			switch typed := value.(type) {
			case string:
				safe[key] = boundedFact(typed)
			case bool:
				safe[key] = typed
			case int, int64:
				safe[key] = typed
			}
		}
	}
	fact := types.Event{Version: types.EventSchemaVersionV1, Type: eventType, RunID: boundedFact(runID), Time: c.coord.now().UTC(), Payload: safe}
	c.emitFactEvent(ctx, fact)
}

func (c *Connection) emitFactEvent(ctx context.Context, fact types.Event) {
	if c == nil || c.coord == nil || c.coord.sink == nil {
		return
	}
	defer func() { _ = recover() }()
	c.coord.sink.OnEvent(ctx, fact)
}

func (c *Connection) emitAdmission(cmd types.HostCommandEnvelope, response types.HostCommandResponse) {
	c.emitFact(context.Background(), types.EventTypeHostAdmission, cmd.RunID, map[string]any{
		"fact": "admission", "session_id": cmd.SessionID, "request_id": cmd.RequestID,
		"message_id": cmd.MessageID, "admission_status": string(response.Status), "reason_code": response.ReasonCode,
	})
}

func (c *Connection) emitAdmissionFromResponse(response types.HostCommandResponse) {
	c.emitFact(context.Background(), types.EventTypeHostAdmission, response.RunID, map[string]any{
		"fact": "admission", "session_id": response.SessionID, "request_id": response.RequestID,
		"message_id": response.MessageID, "admission_status": string(response.Status), "reason_code": response.ReasonCode,
	})
}

func (c *Connection) closeWithCause(cause error) {
	if c == nil {
		return
	}
	if cause == nil {
		cause = ErrConnectionClosed
	}
	c.close.Do(func() {
		c.mu.Lock()
		pendingFacts := make([]types.Event, 0, len(c.pending))
		c.closed = true
		for requestID, entry := range c.pending {
			delete(c.pending, requestID)
			entry.publish(pendingOutcome{err: cause})
			pendingFacts = append(pendingFacts, types.Event{Version: types.EventSchemaVersionV1, Type: types.EventTypeHostPendingClose, RunID: boundedFact(entry.runID), Time: c.coord.now().UTC(), Payload: map[string]any{
				"fact": "disconnected", "session_id": boundedFact(entry.sessionID), "reason_code": boundedFact(cause.Error()),
			}})
		}
		close(c.done)
		c.mu.Unlock()
		for _, fact := range pendingFacts {
			c.emitFactEvent(context.Background(), fact)
		}
		c.cancelWrite(cause)
		if c.writer != nil {
			_ = c.writer.Close()
		}
	})
}

func (p *pendingRequest) publish(outcome pendingOutcome) {
	if outcome.err == nil {
		p.response <- outcome.response
	}
	close(p.response)
	p.settled <- outcome
}

func (c *Connection) isClosed() bool {
	if c == nil {
		return true
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

// HandleCommand is the sole command-response transport owner: every
// classifiable command outcome is written exactly once before this method
// returns. Executable adapters must not write the returned response again;
// the return value exists for in-process inspection and tests.
func (c *Connection) HandleCommand(ctx context.Context, cmd types.HostCommandEnvelope) (types.HostCommandResponse, error) {
	if c == nil || c.coord == nil || c.isClosed() {
		return types.HostCommandResponse{}, ErrConnectionClosed
	}
	if err := cmd.Validate(); err != nil {
		response, classifyErr := c.malformedCommandResponse(cmd, err)
		if classifyErr != nil {
			return types.HostCommandResponse{}, err
		}
		deliveryErr := c.deliver(ctx, response, true)
		c.emitAdmissionFromResponse(response)
		return response, deliveryErr
	}
	c.mu.Lock()
	if _, exists := c.seen[cmd.MessageID]; exists {
		c.mu.Unlock()
		response, err := types.NormalizeHostCommandAdmission(cmd, types.HostAdmissionStatusDuplicate, types.HostReasonDuplicate)
		return c.deliverCommandResponse(ctx, response, err)
	}
	if len(c.seen) >= c.coord.correlationLimit {
		c.mu.Unlock()
		response, err := types.NormalizeHostCommandAdmission(cmd, types.HostAdmissionStatusRejected, HostReasonCorrelationLimit)
		return c.deliverCommandResponse(ctx, response, err)
	}
	c.seen[cmd.MessageID] = struct{}{}
	c.mu.Unlock()
	if c.coord.readiness != nil {
		if err := c.coord.readiness.CheckHostReadiness(ctx, cmd); err != nil {
			response, normalizeErr := reject(cmd, err)
			return c.deliverCommandResponse(ctx, response, normalizeErr)
		}
	}
	if c.coord.authorization != nil {
		if err := c.coord.authorization.AuthorizeHostCommand(ctx, cmd); err != nil {
			response, normalizeErr := reject(cmd, err)
			return c.deliverCommandResponse(ctx, response, normalizeErr)
		}
	}
	barrier := c.beginCommandResponseBarrier(cmd.MessageID)
	defer c.endCommandResponseBarrier(cmd.MessageID, barrier)
	var (
		response types.HostCommandResponse
		after    func()
		abandon  func()
		err      error
	)
	switch cmd.Kind {
	case types.HostCommandKindRunStart:
		response, after, abandon, err = c.startRun(ctx, cmd)
	case types.HostCommandKindAction:
		response, err = c.executeAction(ctx, cmd)
	case types.HostCommandKindRealtimeInterrupt, types.HostCommandKindRealtimeResume:
		response, err = c.ingestRealtime(ctx, cmd)
	case types.HostCommandKindHITLRespond:
		response, err = c.handleHITLResponse(cmd)
	case types.HostCommandKindEventsSubscribe:
		response, after, err = c.subscribe(ctx, cmd)
	case types.HostCommandKindRunGet:
		response, after, err = c.queryTerminal(ctx, cmd)
	default:
		response, err = types.NormalizeHostCommandAdmission(cmd, types.HostAdmissionStatusRejected, types.HostReasonUnknownKind)
	}
	if err != nil {
		barrier.abort()
		if abandon != nil {
			abandon()
		}
		return response, err
	}
	c.emitAdmission(cmd, response)
	if err := c.deliver(ctx, response, true); err != nil {
		barrier.abort()
		if abandon != nil {
			abandon()
		}
		return response, err
	}
	barrier.commit()
	if after != nil {
		after()
	}
	return response, nil
}

func (c *Connection) deliverCommandResponse(ctx context.Context, response types.HostCommandResponse, normalizeErr error) (types.HostCommandResponse, error) {
	if normalizeErr != nil {
		return response, normalizeErr
	}
	err := c.deliver(ctx, response, true)
	c.emitAdmissionFromResponse(response)
	return response, err
}

func (c *Connection) malformedCommandResponse(cmd types.HostCommandEnvelope, validationErr error) (types.HostCommandResponse, error) {
	reason := types.HostReasonInvalidEnvelope
	if text := strings.TrimSpace(validationErr.Error()); text != "" {
		if prefix, _, found := strings.Cut(text, ":"); found && strings.TrimSpace(prefix) != "" {
			reason = strings.TrimSpace(prefix)
		}
	}
	response := types.HostCommandResponse{
		Version: types.HostProtocolVersionV1, MessageID: cmd.MessageID,
		Kind: types.HostEnvelopeKindCommandResponse, Time: c.coord.now().UTC(),
		RequestID: cmd.RequestID, SessionID: cmd.SessionID, RunID: cmd.RunID,
		Status: types.HostAdmissionStatusRejected, ReasonCode: reason,
		SourceCorrelation: cmd.SourceCorrelation,
	}
	if err := response.Validate(); err != nil {
		return types.HostCommandResponse{}, err
	}
	return response, nil
}

func (c *Connection) beginCommandResponseBarrier(messageID string) *commandResponseBarrier {
	barrier := &commandResponseBarrier{limit: c.coord.deliverySize}
	c.mu.Lock()
	c.barriers[messageID] = barrier
	c.mu.Unlock()
	return barrier
}

func (c *Connection) endCommandResponseBarrier(messageID string, barrier *commandResponseBarrier) {
	c.mu.Lock()
	if c.barriers[messageID] == barrier {
		delete(c.barriers, messageID)
	}
	c.mu.Unlock()
}

func (c *Connection) commandResponseBarrier(messageID string) *commandResponseBarrier {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.barriers[messageID]
}

func reject(cmd types.HostCommandEnvelope, err error) (types.HostCommandResponse, error) {
	reason := types.HostReasonInvalidEnvelope
	if err != nil && strings.TrimSpace(err.Error()) != "" {
		reason = err.Error()
	}
	return types.NormalizeHostCommandAdmission(cmd, types.HostAdmissionStatusRejected, reason)
}

func (c *Connection) startRun(ctx context.Context, cmd types.HostCommandEnvelope) (types.HostCommandResponse, func(), func(), error) {
	if c.coord.starter == nil {
		response, err := types.NormalizeHostCommandAdmission(cmd, types.HostAdmissionStatusRejected, "host.run_starter_unavailable")
		return response, nil, nil, err
	}
	req := types.RunRequest{RunID: cmd.RunID, SessionID: cmd.SessionID}
	if raw, ok := cmd.Payload["input"].(string); ok {
		req.Input = raw
	}
	mode := types.HostRunExecutionModeRun
	if stream, _ := cmd.Payload["stream"].(bool); stream {
		mode = types.HostRunExecutionModeStream
	}
	admission, err := c.coord.starter.AdmitHostRun(context.WithoutCancel(ctx), req, mode)
	if err != nil {
		response, normalizeErr := reject(cmd, err)
		return response, nil, nil, normalizeErr
	}
	if admission.RunID != "" {
		cmd.RunID = admission.RunID
	}
	response, err := types.NormalizeHostCommandAdmission(cmd, admission.Status, admission.ReasonCode)
	if err != nil {
		if admission.Execution != nil {
			admission.Execution.Release()
		}
		return response, nil, nil, err
	}
	if admission.Status != types.HostAdmissionStatusAccepted {
		if admission.Execution != nil {
			admission.Execution.Release()
		}
		return response, nil, nil, nil
	}
	if admission.Execution == nil {
		response, err = types.NormalizeHostCommandAdmission(cmd, types.HostAdmissionStatusRejected, "host.run_execution_unavailable")
		return response, nil, nil, err
	}
	execution := admission.Execution
	after := func() { go c.executeRun(cmd, execution) }
	return response, after, execution.Release, nil
}

func (c *Connection) executeAction(ctx context.Context, cmd types.HostCommandEnvelope) (types.HostCommandResponse, error) {
	action, _ := cmd.Payload["action"].(string)
	normalized := types.ProtocolAction(strings.ToLower(strings.TrimSpace(action)))
	if normalized != types.ProtocolActionCancel && normalized != types.ProtocolActionResume && normalized != types.ProtocolActionRetry {
		return types.NormalizeHostCommandAdmission(cmd, types.HostAdmissionStatusRejected, types.ProtocolReasonActionUnsupported)
	}
	if c.coord.control != nil {
		status, reason, err := c.coord.control.ExecuteAction(ctx, cmd, normalized)
		if err != nil {
			c.emitFact(ctx, types.EventTypeHostSourceControl, cmd.RunID, map[string]any{"fact": "rejected", "session_id": cmd.SessionID, "request_id": cmd.RequestID, "source_control": string(normalized), "reason_code": err.Error()})
			return reject(cmd, err)
		}
		c.emitFact(ctx, types.EventTypeHostSourceControl, cmd.RunID, map[string]any{"fact": string(status), "session_id": cmd.SessionID, "request_id": cmd.RequestID, "source_control": string(normalized), "reason_code": reason})
		return types.NormalizeHostCommandAdmission(cmd, status, reason)
	}
	if source, ok := c.coord.runner.(ControlSource); ok && normalized == types.ProtocolActionCancel {
		status, err := source.CancelRun(cmd.RunID, cmd.SessionID)
		if err != nil {
			c.emitFact(ctx, types.EventTypeHostSourceControl, cmd.RunID, map[string]any{"fact": "rejected", "session_id": cmd.SessionID, "request_id": cmd.RequestID, "source_control": string(normalized), "reason_code": err.Error()})
			return reject(cmd, err)
		}
		c.emitFact(ctx, types.EventTypeHostSourceControl, cmd.RunID, map[string]any{"fact": string(status), "session_id": cmd.SessionID, "request_id": cmd.RequestID, "source_control": string(normalized)})
		return types.NormalizeHostCommandAdmission(cmd, status, "")
	}
	return types.NormalizeHostCommandAdmission(cmd, types.HostAdmissionStatusRejected, types.ProtocolReasonActionUnsupported)
}

func (c *Connection) ingestRealtime(ctx context.Context, cmd types.HostCommandEnvelope) (types.HostCommandResponse, error) {
	ev := types.RealtimeEventEnvelope{EventID: cmd.MessageID, SessionID: cmd.SessionID, RunID: cmd.RunID, Seq: int64FromPayload(cmd.Payload, "seq"), Type: types.RealtimeEventTypeInterrupt, TS: c.coord.now().UTC(), Payload: cmd.Payload}
	if cmd.Kind == types.HostCommandKindRealtimeResume {
		ev.Type = types.RealtimeEventTypeResume
	}
	if c.coord.control != nil {
		status, reason, err := c.coord.control.IngestRealtime(ctx, cmd, ev)
		if err != nil {
			c.emitFact(ctx, types.EventTypeHostSourceControl, cmd.RunID, map[string]any{"fact": "rejected", "session_id": cmd.SessionID, "request_id": cmd.RequestID, "source_control": string(cmd.Kind), "reason_code": err.Error()})
			return reject(cmd, err)
		}
		c.emitFact(ctx, types.EventTypeHostSourceControl, cmd.RunID, map[string]any{"fact": string(status), "session_id": cmd.SessionID, "request_id": cmd.RequestID, "source_control": string(cmd.Kind), "reason_code": reason})
		return types.NormalizeHostCommandAdmission(cmd, status, reason)
	}
	if source, ok := c.coord.runner.(ControlSource); ok {
		status, err := source.IngestRealtime(cmd.RunID, cmd.SessionID, ev)
		if err != nil {
			c.emitFact(ctx, types.EventTypeHostSourceControl, cmd.RunID, map[string]any{"fact": "rejected", "session_id": cmd.SessionID, "request_id": cmd.RequestID, "source_control": string(cmd.Kind), "reason_code": err.Error()})
			return reject(cmd, err)
		}
		c.emitFact(ctx, types.EventTypeHostSourceControl, cmd.RunID, map[string]any{"fact": string(status), "session_id": cmd.SessionID, "request_id": cmd.RequestID, "source_control": string(cmd.Kind)})
		return types.NormalizeHostCommandAdmission(cmd, status, "")
	}
	return types.NormalizeHostCommandAdmission(cmd, types.HostAdmissionStatusRejected, "host.source_control_unavailable")
}

func (c *Connection) handleHITLResponse(cmd types.HostCommandEnvelope) (types.HostCommandResponse, error) {
	requestID, _ := cmd.Payload["request_id"].(string)
	if requestID == "" {
		requestID = cmd.RequestID
	}
	accepted, _ := cmd.Payload["accepted"].(bool)
	resp := types.HostResponseEnvelope{Version: cmd.Version, MessageID: cmd.MessageID, Kind: types.HostEnvelopeKindHostResponse, Time: cmd.Time, RequestID: requestID, SessionID: cmd.SessionID, RunID: cmd.RunID, Payload: clonePayload(cmd.Payload), Accepted: accepted}
	delete(resp.Payload, "request_id")
	delete(resp.Payload, "accepted")
	commit, err := c.preparePendingResponse(resp)
	if err != nil {
		return reject(cmd, err)
	}
	response, err := types.NormalizeHostCommandAdmission(cmd, types.HostAdmissionStatusAccepted, "")
	if err != nil {
		commit(pendingOutcome{err: err})
		return response, err
	}
	barrier := c.commandResponseBarrier(cmd.MessageID)
	if barrier == nil {
		commit(pendingOutcome{err: ErrConnectionClosed})
		return types.HostCommandResponse{}, ErrConnectionClosed
	}
	deferred, deferErr := barrier.deferDelivery(deferredCommandDelivery{
		commit: func() error {
			commit(pendingOutcome{response: resp})
			return nil
		},
		abort: func() { commit(pendingOutcome{err: ErrConnectionClosed}) },
	})
	if deferErr != nil || !deferred {
		commit(pendingOutcome{err: deferErr})
		if deferErr != nil {
			c.closeWithCause(deferErr)
		}
		return types.HostCommandResponse{}, deferErr
	}
	return response, nil
}

func (c *Connection) subscribe(ctx context.Context, cmd types.HostCommandEnvelope) (types.HostCommandResponse, func(), error) {
	if c.coord.subscriber == nil {
		response, err := types.NormalizeHostCommandAdmission(cmd, types.HostAdmissionStatusRejected, "host.event_subscription_unavailable")
		return response, nil, err
	}
	var sub types.EventStreamSubscription
	if err := decodePayload(cmd.Payload, &sub); err != nil {
		response, rejectErr := reject(cmd, err)
		return response, nil, rejectErr
	}
	if sub.SessionID == "" {
		sub.SessionID = cmd.SessionID
	}
	if sub.RunID == "" {
		sub.RunID = cmd.RunID
	}
	projection, err := c.coord.subscriber.SubscribeHostEvents(ctx, sub)
	if err != nil {
		response, rejectErr := reject(cmd, err)
		return response, nil, rejectErr
	}
	resp, err := types.NormalizeHostCommandAdmission(cmd, types.HostAdmissionStatusAccepted, projection.Outcome.ReasonCode)
	return resp, func() { go c.emitProjection(cmd, projection) }, err
}

func (c *Connection) queryTerminal(ctx context.Context, cmd types.HostCommandEnvelope) (types.HostCommandResponse, func(), error) {
	if c.coord.terminal == nil {
		response, err := types.NormalizeHostCommandAdmission(cmd, types.HostAdmissionStatusRejected, "host.terminal_query_unavailable")
		return response, nil, err
	}
	terminal, err := c.coord.terminal.QueryTerminal(ctx, cmd.SessionID, cmd.RunID)
	if err != nil {
		response, rejectErr := reject(cmd, err)
		return response, nil, rejectErr
	}
	resp, err := types.NormalizeHostCommandAdmission(cmd, types.HostAdmissionStatusAccepted, "")
	if err != nil || terminal == nil {
		return resp, nil, err
	}
	return resp, func() { go c.emitTerminal(cmd, *terminal) }, nil
}

func (c *Connection) emitTerminal(cmd types.HostCommandEnvelope, terminal types.TerminalOutcome) {
	when := c.coord.now().UTC()
	event := types.EventEnvelope{
		EventID:   "terminal-" + cmd.MessageID,
		RunID:     terminal.RunID,
		SessionID: terminal.SessionID,
		Source:    types.ProtocolSourceRunner,
		Kind:      types.ProtocolEventKindState,
		Time:      when,
		Payload:   map[string]any{"terminal_outcome": terminal},
	}
	envelope := types.HostRuntimeEventEnvelope{
		Version: types.HostProtocolVersionV1, MessageID: event.EventID,
		Kind: types.HostEnvelopeKindRuntimeEvent, Time: when, Event: event,
		RequestID: cmd.RequestID, CausationID: cmd.CausationID,
		SourceCorrelation: cmd.SourceCorrelation,
	}
	_ = c.deliver(context.Background(), envelope, true)
}

func int64FromPayload(payload map[string]any, key string) int64 {
	switch n := payload[key].(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case float64:
		return int64(n)
	default:
		return 0
	}
}

func (c *Connection) executeRun(cmd types.HostCommandEnvelope, execution types.HostRunExecution) {
	h := EventHandlerFunc(func(eventCtx context.Context, ev types.Event) { _ = c.emitEvent(eventCtx, cmd, ev) })
	_, _ = execution.Execute(h)
}

func (c *Connection) emitEvent(ctx context.Context, cmd types.HostCommandEnvelope, ev types.Event) error {
	mapped, err := types.MapEventToProtocol(ev, types.ProtocolSourceRunner)
	if err != nil {
		return err
	}
	if mapped.SessionID == "" {
		mapped.SessionID = cmd.SessionID
	}
	env := types.HostRuntimeEventEnvelope{Version: types.HostProtocolVersionV1, MessageID: mapped.EventID, Kind: types.HostEnvelopeKindRuntimeEvent, Time: mapped.Time, RequestID: cmd.RequestID, CausationID: cmd.CausationID, SourceCorrelation: cmd.SourceCorrelation, Event: mapped}
	if barrier := c.commandResponseBarrier(cmd.MessageID); barrier != nil {
		deferred, err := barrier.deferDelivery(deferredCommandDelivery{
			commit: func() error { return c.deliver(context.Background(), env, true) },
		})
		if err != nil {
			c.emitFact(ctx, types.EventTypeHostDelivery, cmd.RunID, map[string]any{"fact": "failed", "session_id": cmd.SessionID, "request_id": cmd.RequestID, "delivery_status": "queue_full", "reason_code": err.Error()})
			barrier.abort()
			c.closeWithCause(err)
			return err
		}
		if deferred {
			return nil
		}
	}
	// Direct Runner callbacks carry no source-owned drop policy. Treat every
	// such frame as non-droppable, but cap the callback wait by deliveryWait.
	return c.deliver(ctx, env, true)
}

func (c *Connection) emitProjection(cmd types.HostCommandEnvelope, projection types.EventStreamBindingProjection) {
	events, err := types.MapEventStreamBindingToProtocol(projection)
	if err != nil {
		return
	}
	for i, event := range events {
		env := types.HostRuntimeEventEnvelope{Version: types.HostProtocolVersionV1, MessageID: event.EventID, Kind: types.HostEnvelopeKindRuntimeEvent, Time: event.Time, RequestID: cmd.RequestID, CausationID: cmd.CausationID, SourceCorrelation: cmd.SourceCorrelation, Event: event}
		critical := !c.projectionAllowsRecordedDrop(projection, i)
		if err := c.deliver(context.Background(), env, critical); err != nil {
			if !critical && errors.Is(err, ErrDeliveryFull) {
				c.emitFact(context.Background(), types.EventTypeHostDelivery, cmd.RunID, map[string]any{
					"fact": "dropped", "session_id": cmd.SessionID, "request_id": cmd.RequestID,
					"message_id": event.EventID, "delivery_status": "dropped", "reason_code": HostReasonDeliverySourcePolicyDrop,
				})
				continue
			}
			c.emitFact(context.Background(), types.EventTypeHostDelivery, cmd.RunID, map[string]any{"fact": "failed", "session_id": cmd.SessionID, "request_id": cmd.RequestID, "delivery_status": "failed", "reason_code": err.Error()})
			return
		}
	}
}

func (c *Connection) projectionAllowsRecordedDrop(projection types.EventStreamBindingProjection, index int) bool {
	if c == nil || c.coord == nil || c.coord.sink == nil ||
		projection.Subscription.DeliveryPolicy != types.EventStreamDeliveryPolicyDropWithRecord ||
		!projection.Outcome.SourceOutcomeDeclared || index < 0 || index >= len(projection.Events) {
		return false
	}
	eventType := strings.TrimSpace(string(projection.Events[index].Type))
	return strings.EqualFold(eventType, string(types.RealtimeEventTypeDelta))
}

type EventHandlerFunc func(context.Context, types.Event)

func (f EventHandlerFunc) OnEvent(ctx context.Context, ev types.Event) {
	if f != nil {
		f(ctx, ev)
	}
}

func (c *Connection) RegisterPending(requestID string) (<-chan types.HostResponseEnvelope, error) {
	entry, err := c.registerPending(requestID, "", "")
	if err != nil {
		return nil, err
	}
	return entry.response, nil
}

func (c *Connection) registerPending(requestID, sessionID, runID string) (*pendingRequest, error) {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return nil, errors.New("host: request id required")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil, ErrConnectionClosed
	}
	if _, exists := c.pending[requestID]; exists {
		return nil, errors.New("host: duplicate request")
	}
	if len(c.pending) >= c.coord.pendingLimit {
		return nil, ErrPendingLimit
	}
	entry := &pendingRequest{
		response:  make(chan types.HostResponseEnvelope, 1),
		settled:   make(chan pendingOutcome, 1),
		sessionID: strings.TrimSpace(sessionID),
		runID:     strings.TrimSpace(runID),
	}
	c.pending[requestID] = entry
	return entry, nil
}

func (c *Connection) Respond(resp types.HostResponseEnvelope) error {
	if err := resp.Validate(); err != nil {
		return err
	}
	commit, err := c.preparePendingResponse(resp)
	if err != nil {
		return err
	}
	commit(pendingOutcome{response: resp})
	return nil
}

// preparePendingResponse atomically claims the source-owned pending wait but
// deliberately delays publishing it. HITL commands use this two-step form so
// continuation cannot race ahead of the command_response transport frame.
func (c *Connection) preparePendingResponse(resp types.HostResponseEnvelope) (func(pendingOutcome), error) {
	if err := resp.Validate(); err != nil {
		return nil, err
	}
	c.mu.Lock()
	entry, ok := c.pending[resp.RequestID]
	if ok && ((entry.sessionID != "" && entry.sessionID != strings.TrimSpace(resp.SessionID)) || (entry.runID != "" && entry.runID != strings.TrimSpace(resp.RunID))) {
		c.mu.Unlock()
		c.emitFact(context.Background(), types.EventTypeHostCorrelation, resp.RunID, map[string]any{"fact": "mismatch", "session_id": resp.SessionID, "request_id": resp.RequestID, "message_id": resp.MessageID, "reason_code": "session_or_run_mismatch"})
		return nil, ErrPendingCorrelation
	}
	if ok {
		delete(c.pending, resp.RequestID)
	}
	c.mu.Unlock()
	if !ok {
		c.emitFact(context.Background(), types.EventTypeHostCorrelation, resp.RunID, map[string]any{"fact": "unknown", "session_id": resp.SessionID, "request_id": resp.RequestID, "message_id": resp.MessageID, "reason_code": ErrPendingNotFound.Error()})
		return nil, fmt.Errorf("%w: %s", ErrPendingNotFound, resp.RequestID)
	}
	var once sync.Once
	return func(outcome pendingOutcome) {
		once.Do(func() {
			entry.publish(outcome)
			fact := "completed"
			if outcome.err != nil {
				fact = "rejected"
			}
			c.emitFact(context.Background(), types.EventTypeHostPendingClose, entry.runID, map[string]any{
				"fact": fact, "session_id": entry.sessionID, "request_id": resp.RequestID, "reason_code": pendingReason(outcome.err),
			})
		})
	}, nil
}

func pendingReason(err error) string {
	if err == nil {
		return ""
	}
	return boundedFact(err.Error())
}

func (c *Connection) settlePending(requestID string, entry *pendingRequest, outcome pendingOutcome) pendingOutcome {
	c.mu.Lock()
	if current, ok := c.pending[requestID]; ok && current == entry {
		delete(c.pending, requestID)
		entry.publish(outcome)
		c.mu.Unlock()
		fact := "completed"
		if outcome.err != nil {
			fact = "rejected"
		}
		c.emitFact(context.Background(), types.EventTypeHostPendingClose, entry.runID, map[string]any{
			"fact": fact, "session_id": entry.sessionID, "request_id": requestID, "reason_code": pendingReason(outcome.err),
		})
		return outcome
	}
	c.mu.Unlock()
	return <-entry.settled
}

// Resolve implements types.ClarificationResolver through a reverse request.
func (c *Connection) Resolve(ctx context.Context, req types.ClarificationResolveRequest) (types.ClarificationResponse, error) {
	requestID := strings.TrimSpace(req.Request.RequestID)
	entry, err := c.registerPending(requestID, req.SessionID, req.RunID)
	if err != nil {
		return types.ClarificationResponse{}, err
	}
	if err := c.deliver(ctx, c.newHostRequest(requestID, req.SessionID, req.RunID, HostRequestTypeClarification, req), true); err != nil {
		outcome := c.settlePending(requestID, entry, pendingOutcome{err: err})
		if outcome.err != nil {
			return types.ClarificationResponse{}, outcome.err
		}
		return clarificationResponse(requestID, outcome.response)
	}
	resp, err := c.awaitPending(ctx, req.Timeout, requestID, entry)
	if err != nil {
		return types.ClarificationResponse{}, err
	}
	return clarificationResponse(requestID, resp)
}

func clarificationResponse(requestID string, resp types.HostResponseEnvelope) (types.ClarificationResponse, error) {
	if !resp.Accepted {
		return types.ClarificationResponse{}, errors.New("host: clarification canceled_by_user")
	}
	return types.ClarificationResponse{RequestID: requestID, Answers: stringSlice(resp.Payload["answers"]), Meta: mapPayload(resp.Payload["meta"])}, nil
}

// Confirm implements types.ActionGateResolver; every bridge failure is closed.
func (c *Connection) Confirm(ctx context.Context, req types.ActionGateConfirmRequest) (bool, error) {
	requestID := strings.TrimSpace(req.Check.CallID)
	if requestID == "" {
		return false, errors.New("host: action gate call_id required")
	}
	entry, err := c.registerPending(requestID, req.Check.SessionID, req.Check.RunID)
	if err != nil {
		return false, err
	}
	if err := c.deliver(ctx, c.newHostRequest(requestID, req.Check.SessionID, req.Check.RunID, HostRequestTypeActionGate, req), true); err != nil {
		outcome := c.settlePending(requestID, entry, pendingOutcome{err: err})
		if outcome.err != nil {
			return false, outcome.err
		}
		return outcome.response.Accepted, nil
	}
	resp, err := c.awaitPending(ctx, req.Timeout, requestID, entry)
	if err != nil {
		return false, err
	}
	return resp.Accepted, nil
}

func (c *Connection) newHostRequest(requestID, sessionID, runID, requestType string, payload any) types.HostRequestEnvelope {
	return types.HostRequestEnvelope{Version: types.HostProtocolVersionV1, MessageID: fmt.Sprintf("host-request-%d", c.coord.nextMessageID.Add(1)), Kind: types.HostEnvelopeKindHostRequest, Time: c.coord.now().UTC(), RequestID: requestID, SessionID: sessionID, RunID: runID, RequestType: requestType, Payload: structPayload(payload)}
}

func (c *Connection) awaitPending(ctx context.Context, timeout time.Duration, requestID string, entry *pendingRequest) (types.HostResponseEnvelope, error) {
	if timeout <= 0 {
		timeout = c.coord.deliveryWait
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case outcome := <-entry.settled:
		return outcome.response, outcome.err
	case <-ctx.Done():
		outcome := c.settlePending(requestID, entry, pendingOutcome{err: ctx.Err()})
		return outcome.response, outcome.err
	case <-timer.C:
		outcome := c.settlePending(requestID, entry, pendingOutcome{err: context.DeadlineExceeded})
		return outcome.response, outcome.err
	case <-c.done:
		outcome := <-entry.settled
		return outcome.response, outcome.err
	}
}

func (c *Connection) deliver(ctx context.Context, frame any, critical bool) error {
	if c == nil || c.isClosed() {
		return ErrConnectionClosed
	}
	writeCtx, cancelWrite := context.WithTimeoutCause(c.writeCtx, c.coord.deliveryWait, ErrDeliveryTimeout)
	item := delivery{frame: frame, ctx: writeCtx, cancel: cancelWrite}
	if critical {
		item.result = make(chan error, 1)
	}
	if !critical {
		select {
		case c.queue <- item:
			return nil
		case <-c.done:
			cancelWrite()
			return ErrConnectionClosed
		default:
			cancelWrite()
			return ErrDeliveryFull
		}
	}
	select {
	case c.queue <- item:
	case <-writeCtx.Done():
		cause := context.Cause(writeCtx)
		c.closeWithCause(cause)
		return cause
	case <-c.done:
		cancelWrite()
		return ErrConnectionClosed
	}
	// Once a critical frame is accepted by the connection queue, its delivery
	// result—not caller cancellation—owns commit versus rollback. Otherwise a
	// writer may make an accepted response visible after its reservation was
	// already released.
	select {
	case err := <-item.result:
		cancelWrite()
		return err
	case <-writeCtx.Done():
		cause := context.Cause(writeCtx)
		if !c.interruptibleWriter {
			c.closeWithCause(cause)
			return cause
		}
		// A compliant FrameWriter returns after cancellation. Wait for that
		// deterministic outcome before the caller rolls back any reservation.
		err := <-item.result
		if err == nil {
			return nil
		}
		c.closeWithCause(cause)
		return cause
	case <-c.done:
		if c.interruptibleWriter {
			// The item was already accepted by the production writer loop.
			// Close cancels that write, but rollback waits until the writer has
			// returned and can no longer make the frame visible.
			return <-item.result
		}
		return ErrConnectionClosed
	}
}

func (c *Connection) writeLoop() {
	for {
		select {
		case <-c.done:
			c.failQueuedDeliveries(context.Cause(c.writeCtx))
			return
		case item := <-c.queue:
			err := c.safeWrite(item.ctx, item.frame)
			if err != nil {
				if cause := context.Cause(item.ctx); cause != nil {
					err = cause
				}
			}
			if item.result == nil {
				item.cancel()
			}
			if err != nil {
				c.closeWithCause(err)
			}
			if item.result != nil {
				item.result <- err
			}
			if err != nil {
				c.failQueuedDeliveries(err)
				return
			}
		}
	}
}

func (c *Connection) failQueuedDeliveries(cause error) {
	if cause == nil {
		cause = ErrConnectionClosed
	}
	for {
		select {
		case item := <-c.queue:
			item.cancel()
			if item.result != nil {
				item.result <- cause
			}
		default:
			return
		}
	}
}

func (c *Connection) safeWrite(ctx context.Context, frame any) (err error) {
	if c.writer == nil {
		return ErrConnectionClosed
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("host: writer panic: %v", recovered)
		}
	}()
	err = c.writer.WriteFrame(ctx, frame)
	if err != nil {
		c.emitFact(context.Background(), types.EventTypeHostDelivery, "", map[string]any{"fact": "failed", "delivery_status": "write_error", "reason_code": err.Error()})
	}
	return err
}

func decodePayload(payload map[string]any, target any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, target)
}
func structPayload(value any) map[string]any {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	var payload map[string]any
	if json.Unmarshal(raw, &payload) != nil {
		return nil
	}
	return payload
}
func clonePayload(payload map[string]any) map[string]any {
	if payload == nil {
		return nil
	}
	out := make(map[string]any, len(payload))
	for key, value := range payload {
		out[key] = value
	}
	return out
}
func stringSlice(value any) []string {
	switch values := value.(type) {
	case []string:
		return append([]string(nil), values...)
	case []any:
		out := make([]string, 0, len(values))
		for _, value := range values {
			if text, ok := value.(string); ok {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}
func mapPayload(value any) map[string]any {
	if payload, ok := value.(map[string]any); ok {
		return clonePayload(payload)
	}
	return nil
}
