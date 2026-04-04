package runner

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

const (
	realtimeRunnerEventType = "realtime.event"

	realtimeErrorLayerTransport = "transport"
	realtimeErrorLayerProtocol  = "protocol"
	realtimeErrorLayerSemantic  = "semantic"

	realtimeReasonUnsupportedEventType = "realtime.unsupported_event_type"
	realtimeReasonSequenceGap          = "realtime.sequence_gap"
	realtimeReasonEventOrderDrift      = "realtime.event_order_drift"
	realtimeReasonInvalidResumeCursor  = "realtime.resume.invalid_cursor"
	realtimeReasonInterruptedFreeze    = "realtime.interrupt.freeze"
	realtimeReasonSchemaInvalid        = "realtime.schema_invalid"
	realtimeReasonBufferOverflow       = "realtime.buffer_overflow"
)

type realtimeProtocolError struct {
	Code    string
	Layer   string
	Message string
}

func (e *realtimeProtocolError) Error() string {
	if e == nil {
		return ""
	}
	return strings.TrimSpace(e.Message)
}

type realtimeCursorRecord struct {
	Cursor    string
	ExpiresAt time.Time
}

type realtimeSessionRuntime struct {
	engine    *Engine
	cfg       runtimeconfig.RuntimeRealtimeConfig
	runID     string
	sessionID string

	seqMax      int64
	lastInSeq   int64
	seenDedup   map[string]struct{}
	interrupted bool
	cursor      string

	interruptTotal int
	resumeTotal    int
	dedupTotal     int
	resumeSource   string

	lastErrorCode  string
	lastErrorLayer string
}

func (e *Engine) runtimeRealtimeConfigSnapshot() runtimeconfig.RuntimeRealtimeConfig {
	cfg := runtimeconfig.DefaultConfig().Runtime.Realtime
	if e != nil && e.runtimeMgr != nil {
		cfg = e.runtimeMgr.EffectiveConfig().Runtime.Realtime
	}
	return cfg
}

func (e *Engine) newRealtimeSessionRuntime(runID string, req types.RunRequest) *realtimeSessionRuntime {
	cfg := e.runtimeRealtimeConfigSnapshot()
	if !cfg.Protocol.Enabled {
		return nil
	}
	sessionID := strings.TrimSpace(req.SessionID)
	if req.Realtime != nil && strings.TrimSpace(req.Realtime.SessionID) != "" {
		sessionID = strings.TrimSpace(req.Realtime.SessionID)
	}
	if sessionID == "" {
		sessionID = "session:" + strings.TrimSpace(runID)
	}
	return &realtimeSessionRuntime{
		engine:    e,
		cfg:       cfg,
		runID:     strings.TrimSpace(runID),
		sessionID: sessionID,
		seenDedup: map[string]struct{}{},
	}
}

func (s *realtimeSessionRuntime) fillRunFinishMeta(meta *runFinishMeta) {
	if s == nil || meta == nil {
		return
	}
	meta.RealtimeProtocolVersion = strings.TrimSpace(s.cfg.Protocol.Version)
	meta.RealtimeEventSeqMax = s.seqMax
	meta.RealtimeInterruptTotal = s.interruptTotal
	meta.RealtimeResumeTotal = s.resumeTotal
	meta.RealtimeResumeSource = strings.TrimSpace(s.resumeSource)
	meta.RealtimeIdempotencyDedupTotal = s.dedupTotal
	meta.RealtimeLastErrorCode = strings.TrimSpace(s.lastErrorCode)
	meta.RealtimeErrorLayer = strings.TrimSpace(s.lastErrorLayer)
}

func (s *realtimeSessionRuntime) ingestControlEvents(
	ctx context.Context,
	h types.EventHandler,
	iteration int,
	events []types.RealtimeEventEnvelope,
) error {
	if s == nil || len(events) == 0 {
		return nil
	}
	if s.cfg.Protocol.MaxBufferedEvents > 0 && len(events) > s.cfg.Protocol.MaxBufferedEvents {
		return s.fail(
			realtimeReasonBufferOverflow,
			realtimeErrorLayerProtocol,
			fmt.Sprintf(
				"realtime control events overflow: buffered=%d limit=%d",
				len(events),
				s.cfg.Protocol.MaxBufferedEvents,
			),
		)
	}
	for i := range events {
		ev := events[i]
		ev.Type = types.RealtimeEventType(strings.ToLower(strings.TrimSpace(string(ev.Type))))
		if strings.TrimSpace(ev.SessionID) == "" {
			ev.SessionID = s.sessionID
		}
		if strings.TrimSpace(ev.RunID) == "" {
			ev.RunID = s.runID
		}
		if ev.Payload == nil {
			ev.Payload = map[string]any{}
		}
		if ev.TS.IsZero() {
			ev.TS = s.now()
		}
		switch ev.Type {
		case types.RealtimeEventTypeRequest,
			types.RealtimeEventTypeDelta,
			types.RealtimeEventTypeInterrupt,
			types.RealtimeEventTypeResume,
			types.RealtimeEventTypeAck,
			types.RealtimeEventTypeError,
			types.RealtimeEventTypeComplete:
		default:
			return s.fail(
				realtimeReasonUnsupportedEventType,
				realtimeErrorLayerProtocol,
				fmt.Sprintf("unsupported realtime event type %q", strings.TrimSpace(string(ev.Type))),
			)
		}
		if err := types.ValidateRealtimeEventEnvelope(ev); err != nil {
			return s.fail(realtimeReasonSchemaInvalid, realtimeErrorLayerProtocol, err.Error())
		}
		if key := strings.TrimSpace(ev.DedupKey()); key != "" {
			if _, seen := s.seenDedup[key]; seen {
				s.dedupTotal++
				continue
			}
			s.seenDedup[key] = struct{}{}
		}
		if err := s.acceptSequence(ev.Seq); err != nil {
			return err
		}
		s.observeSeq(ev.Seq)
		s.emitEnvelope(ctx, h, iteration, ev)
		if err := s.applyControlEvent(ev); err != nil {
			return err
		}
	}
	return nil
}

func (s *realtimeSessionRuntime) emitRequest(ctx context.Context, h types.EventHandler, iteration int, req types.RunRequest) {
	if s == nil {
		return
	}
	_ = s.emitSyntheticEvent(
		ctx,
		h,
		iteration,
		types.RealtimeEventTypeRequest,
		map[string]any{
			"input": strings.TrimSpace(req.Input),
		},
	)
}

func (s *realtimeSessionRuntime) emitDelta(ctx context.Context, h types.EventHandler, iteration int, delta string) {
	if s == nil {
		return
	}
	_ = s.emitSyntheticEvent(
		ctx,
		h,
		iteration,
		types.RealtimeEventTypeDelta,
		map[string]any{
			"delta": delta,
		},
	)
}

func (s *realtimeSessionRuntime) emitComplete(ctx context.Context, h types.EventHandler, iteration int, finalAnswer string) {
	if s == nil {
		return
	}
	_ = s.emitSyntheticEvent(
		ctx,
		h,
		iteration,
		types.RealtimeEventTypeComplete,
		map[string]any{
			"final_answer": finalAnswer,
		},
	)
}

func (s *realtimeSessionRuntime) emitError(
	ctx context.Context,
	h types.EventHandler,
	iteration int,
	reasonCode string,
	layer string,
	message string,
) {
	if s == nil {
		return
	}
	_ = s.emitSyntheticEvent(
		ctx,
		h,
		iteration,
		types.RealtimeEventTypeError,
		map[string]any{
			"reason_code": strings.TrimSpace(reasonCode),
			"error_layer": strings.TrimSpace(layer),
			"message":     strings.TrimSpace(message),
		},
	)
}

func (s *realtimeSessionRuntime) interruptTerminalError() *types.ClassifiedError {
	if s == nil {
		return nil
	}
	terminal := classified(types.ErrContext, "realtime interrupt accepted; output progression frozen", false)
	terminal.Details = map[string]any{
		"reason_code":            realtimeReasonInterruptedFreeze,
		"realtime_error_layer":   realtimeErrorLayerSemantic,
		"realtime_resume_cursor": strings.TrimSpace(s.cursor),
	}
	s.lastErrorCode = realtimeReasonInterruptedFreeze
	s.lastErrorLayer = realtimeErrorLayerSemantic
	return terminal
}

func classifyRealtimeError(err error) *types.ClassifiedError {
	var protoErr *realtimeProtocolError
	if !errors.As(err, &protoErr) {
		terminal := classified(types.ErrContext, strings.TrimSpace(err.Error()), false)
		terminal.Details = map[string]any{
			"reason_code":          realtimeReasonSchemaInvalid,
			"realtime_error_layer": realtimeErrorLayerProtocol,
		}
		return terminal
	}
	terminal := classified(types.ErrContext, strings.TrimSpace(protoErr.Message), false)
	terminal.Details = map[string]any{
		"reason_code":          strings.TrimSpace(protoErr.Code),
		"realtime_error_layer": strings.TrimSpace(protoErr.Layer),
	}
	return terminal
}

func realtimeErrorDetailsFromTerminal(terminal *types.ClassifiedError) (string, string) {
	if terminal == nil || len(terminal.Details) == 0 {
		return "", ""
	}
	reasonCode, _ := terminal.Details["reason_code"].(string)
	layer, _ := terminal.Details["realtime_error_layer"].(string)
	return strings.TrimSpace(reasonCode), strings.TrimSpace(layer)
}

func errorMessageForRealtime(runErr error, terminal *types.ClassifiedError) string {
	if runErr != nil {
		return strings.TrimSpace(runErr.Error())
	}
	if terminal != nil {
		return strings.TrimSpace(terminal.Message)
	}
	return ""
}

func (s *realtimeSessionRuntime) emitSyntheticEvent(
	ctx context.Context,
	h types.EventHandler,
	iteration int,
	typ types.RealtimeEventType,
	payload map[string]any,
) error {
	seq := s.nextSeq()
	event := types.RealtimeEventEnvelope{
		EventID:   fmt.Sprintf("rt-%s-%d", strings.TrimSpace(s.runID), seq),
		SessionID: s.sessionID,
		RunID:     s.runID,
		Seq:       seq,
		Type:      typ,
		TS:        s.now(),
		Payload:   payload,
	}
	if err := types.ValidateRealtimeEventEnvelope(event); err != nil {
		return s.fail(realtimeReasonSchemaInvalid, realtimeErrorLayerProtocol, err.Error())
	}
	s.emitEnvelope(ctx, h, iteration, event)
	return nil
}

func (s *realtimeSessionRuntime) applyControlEvent(ev types.RealtimeEventEnvelope) error {
	switch ev.Type {
	case types.RealtimeEventTypeInterrupt:
		if !s.interrupted {
			s.interrupted = true
			s.interruptTotal++
			s.cursor = s.buildCursor(ev)
			s.storeCursor(s.cursor)
		}
	case types.RealtimeEventTypeResume:
		cursor := strings.TrimSpace(ev.ResumeCursor())
		if !s.validResumeCursor(cursor) {
			return s.fail(
				realtimeReasonInvalidResumeCursor,
				realtimeErrorLayerSemantic,
				fmt.Sprintf("invalid resume cursor %q", cursor),
			)
		}
		if s.interrupted {
			s.interrupted = false
			s.resumeTotal++
			s.resumeSource = "cursor"
		}
	case types.RealtimeEventTypeError:
		if reasonCode := strings.TrimSpace(payloadString(ev.Payload, "reason_code")); reasonCode != "" {
			s.lastErrorCode = reasonCode
		}
	}
	return nil
}

func (s *realtimeSessionRuntime) acceptSequence(seq int64) error {
	if seq <= 0 {
		return s.fail(realtimeReasonSchemaInvalid, realtimeErrorLayerProtocol, "realtime seq must be > 0")
	}
	if s.lastInSeq == 0 {
		s.lastInSeq = seq
		return nil
	}
	if seq == s.lastInSeq+1 {
		s.lastInSeq = seq
		return nil
	}
	if seq <= s.lastInSeq {
		return s.fail(
			realtimeReasonEventOrderDrift,
			realtimeErrorLayerProtocol,
			fmt.Sprintf("realtime sequence out of order: last=%d incoming=%d", s.lastInSeq, seq),
		)
	}
	return s.fail(
		realtimeReasonSequenceGap,
		realtimeErrorLayerProtocol,
		fmt.Sprintf("realtime sequence gap: last=%d incoming=%d", s.lastInSeq, seq),
	)
}

func (s *realtimeSessionRuntime) observeSeq(seq int64) {
	if seq > s.seqMax {
		s.seqMax = seq
	}
}

func (s *realtimeSessionRuntime) nextSeq() int64 {
	s.seqMax++
	return s.seqMax
}

func (s *realtimeSessionRuntime) emitEnvelope(ctx context.Context, h types.EventHandler, iteration int, ev types.RealtimeEventEnvelope) {
	if s == nil || s.engine == nil {
		return
	}
	s.engine.emit(ctx, h, types.Event{
		Version:   types.EventSchemaVersionV1,
		Type:      realtimeRunnerEventType,
		RunID:     strings.TrimSpace(ev.RunID),
		Iteration: iteration,
		Time:      ev.TS,
		Payload:   ev.CanonicalPayload(),
	})
}

func (s *realtimeSessionRuntime) buildCursor(ev types.RealtimeEventEnvelope) string {
	return fmt.Sprintf(
		"%s:%s:%d",
		strings.TrimSpace(ev.SessionID),
		strings.TrimSpace(ev.RunID),
		ev.Seq,
	)
}

func (s *realtimeSessionRuntime) validResumeCursor(cursor string) bool {
	cursor = strings.TrimSpace(cursor)
	if cursor == "" {
		return false
	}
	if cursor == strings.TrimSpace(s.cursor) {
		return true
	}
	stored, ok := s.loadCursor()
	return ok && stored == cursor
}

func (s *realtimeSessionRuntime) storeCursor(cursor string) {
	if s == nil || s.engine == nil || strings.TrimSpace(s.sessionID) == "" {
		return
	}
	ttl := time.Duration(s.cfg.InterruptResume.ResumeCursorTTLMS) * time.Millisecond
	if ttl <= 0 {
		return
	}
	s.engine.realtimeCursorMu.Lock()
	defer s.engine.realtimeCursorMu.Unlock()
	if s.engine.realtimeCursors == nil {
		s.engine.realtimeCursors = map[string]realtimeCursorRecord{}
	}
	s.engine.realtimeCursors[s.sessionID] = realtimeCursorRecord{
		Cursor:    cursor,
		ExpiresAt: s.now().Add(ttl),
	}
}

func (s *realtimeSessionRuntime) loadCursor() (string, bool) {
	if s == nil || s.engine == nil || strings.TrimSpace(s.sessionID) == "" {
		return "", false
	}
	s.engine.realtimeCursorMu.Lock()
	defer s.engine.realtimeCursorMu.Unlock()
	record, ok := s.engine.realtimeCursors[s.sessionID]
	if !ok {
		return "", false
	}
	if !record.ExpiresAt.IsZero() && !record.ExpiresAt.After(s.now()) {
		delete(s.engine.realtimeCursors, s.sessionID)
		return "", false
	}
	return strings.TrimSpace(record.Cursor), strings.TrimSpace(record.Cursor) != ""
}

func (s *realtimeSessionRuntime) fail(code string, layer string, message string) error {
	s.lastErrorCode = strings.TrimSpace(code)
	s.lastErrorLayer = strings.TrimSpace(layer)
	return &realtimeProtocolError{
		Code:    strings.TrimSpace(code),
		Layer:   strings.TrimSpace(layer),
		Message: strings.TrimSpace(message),
	}
}

func (s *realtimeSessionRuntime) now() time.Time {
	if s == nil || s.engine == nil || s.engine.now == nil {
		return time.Now().UTC()
	}
	return s.engine.now().UTC()
}

func payloadString(payload map[string]any, key string) string {
	if len(payload) == 0 {
		return ""
	}
	value, ok := payload[strings.TrimSpace(key)]
	if !ok {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	default:
		return strings.TrimSpace(fmt.Sprint(value))
	}
}
