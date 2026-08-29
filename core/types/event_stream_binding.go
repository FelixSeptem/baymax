package types

import "strings"

const DurableEventStreamBindingVersionV1 = "durable_runtime_event_stream_binding.v1"

const (
	ProtocolReasonEventStreamInvalidSubscription         = "protocol.event_stream.invalid_subscription"
	ProtocolReasonEventStreamIncompatibleDeliveryOutcome = "protocol.event_stream.incompatible_delivery_outcome"
	ProtocolReasonEventStreamSequenceGap                 = "protocol.event_stream.sequence_gap"
)

const (
	eventStreamSubscriptionIDMaxBytes = 128
	eventStreamCursorMaxBytes         = 512
	eventStreamMaxBatchSize           = 256
)

// EventStreamStartMode determines whether an embedded host consumes only
// future events or asks the source to resume after its opaque cursor.
type EventStreamStartMode string

const (
	EventStreamStartLatest      EventStreamStartMode = "latest"
	EventStreamStartAfterCursor EventStreamStartMode = "after_cursor"
)

// EventStreamDeliveryPolicy declares a source-owned slow-consumer policy. It
// is an availability/result projection and never gives the binding queue or
// pause authority over a Runtime source.
type EventStreamDeliveryPolicy string

const (
	EventStreamDeliveryPolicyReject         EventStreamDeliveryPolicy = "reject"
	EventStreamDeliveryPolicyDropWithRecord EventStreamDeliveryPolicy = "drop_with_record"
	EventStreamDeliveryPolicyPauseSource    EventStreamDeliveryPolicy = "pause_source"
	EventStreamDeliveryPolicyUnknown        EventStreamDeliveryPolicy = "unknown"
)

// EventStreamBindingPhase is a finite source-owned subscription result.
type EventStreamBindingPhase string

const (
	EventStreamBindingPhaseAccepted      EventStreamBindingPhase = "accepted"
	EventStreamBindingPhaseCatchingUp    EventStreamBindingPhase = "catching_up"
	EventStreamBindingPhaseLive          EventStreamBindingPhase = "live"
	EventStreamBindingPhaseExpired       EventStreamBindingPhase = "expired"
	EventStreamBindingPhaseUnresolved    EventStreamBindingPhase = "unresolved"
	EventStreamBindingPhaseGap           EventStreamBindingPhase = "gap"
	EventStreamBindingPhaseBackpressured EventStreamBindingPhase = "backpressured"
	EventStreamBindingPhaseDisconnected  EventStreamBindingPhase = "disconnected"
	EventStreamBindingPhaseClosed        EventStreamBindingPhase = "closed"
)

// EventStreamCursor is an opaque source cursor. Sequence is optional source
// correlation, not a binding-owned sequence allocator.
type EventStreamCursor struct {
	Value    string `json:"value,omitempty"`
	Sequence int64  `json:"sequence,omitempty"`
}

// EventStreamSubscription is an immutable host request projected to a source.
// It cannot create a listener, retain history, or mutate the source runtime.
type EventStreamSubscription struct {
	Version        string                    `json:"version"`
	SubscriptionID string                    `json:"subscription_id"`
	Source         ProtocolSource            `json:"source"`
	SessionID      string                    `json:"session_id"`
	RunID          string                    `json:"run_id"`
	StartMode      EventStreamStartMode      `json:"start_mode"`
	Cursor         EventStreamCursor         `json:"cursor,omitempty"`
	DeliveryPolicy EventStreamDeliveryPolicy `json:"delivery_policy,omitempty"`
	MaxBatchSize   int                       `json:"max_batch_size"`
}

func (s EventStreamSubscription) Validate() error {
	if strings.TrimSpace(s.Version) != DurableEventStreamBindingVersionV1 {
		return eventStreamValidationError("version must be %q", DurableEventStreamBindingVersionV1)
	}
	if id := strings.TrimSpace(s.SubscriptionID); id == "" || len(id) > eventStreamSubscriptionIDMaxBytes {
		return eventStreamValidationError("subscription_id is required and must not exceed %d bytes", eventStreamSubscriptionIDMaxBytes)
	}
	if s.Source != ProtocolSourceRealtime {
		return eventStreamValidationError("source must be %q", ProtocolSourceRealtime)
	}
	if strings.TrimSpace(s.SessionID) == "" || strings.TrimSpace(s.RunID) == "" {
		return eventStreamValidationError("session_id and run_id are required")
	}
	if s.MaxBatchSize <= 0 || s.MaxBatchSize > eventStreamMaxBatchSize {
		return eventStreamValidationError("max_batch_size must be between 1 and %d", eventStreamMaxBatchSize)
	}
	if !isValidEventStreamDeliveryPolicy(s.DeliveryPolicy) {
		return eventStreamValidationError("unsupported delivery_policy %q", s.DeliveryPolicy)
	}
	switch s.StartMode {
	case EventStreamStartLatest:
		if strings.TrimSpace(s.Cursor.Value) != "" || s.Cursor.Sequence != 0 {
			return eventStreamValidationError("latest start_mode must not include cursor")
		}
	case EventStreamStartAfterCursor:
		if strings.TrimSpace(s.Cursor.Value) == "" || len(strings.TrimSpace(s.Cursor.Value)) > eventStreamCursorMaxBytes || s.Cursor.Sequence < 0 {
			return eventStreamValidationError("after_cursor start_mode requires bounded cursor value and non-negative sequence")
		}
	default:
		return eventStreamValidationError("unsupported start_mode %q", s.StartMode)
	}
	return nil
}

// EventStreamBindingOutcome is supplied by the source. The binding validates
// that the result can be represented, but it never executes the result.
type EventStreamBindingOutcome struct {
	SubscriptionID        string                  `json:"subscription_id"`
	Phase                 EventStreamBindingPhase `json:"phase"`
	ReasonCode            string                  `json:"reason_code"`
	LastSequence          int64                   `json:"last_sequence,omitempty"`
	SourceOutcomeDeclared bool                    `json:"source_outcome_declared,omitempty"`
}

func (o EventStreamBindingOutcome) Validate(subscription EventStreamSubscription) error {
	if err := subscription.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(o.SubscriptionID) != strings.TrimSpace(subscription.SubscriptionID) {
		return eventStreamValidationError("outcome subscription_id does not match subscription")
	}
	if !isValidEventStreamBindingPhase(o.Phase) {
		return eventStreamValidationError("unsupported binding phase %q", o.Phase)
	}
	if strings.TrimSpace(o.ReasonCode) == "" {
		return eventStreamValidationError("binding reason_code is required")
	}
	if o.LastSequence < 0 {
		return eventStreamValidationError("last_sequence must not be negative")
	}
	if o.Phase == EventStreamBindingPhaseBackpressured && subscription.DeliveryPolicy == EventStreamDeliveryPolicyReject {
		return eventStreamIncompatibleOutcomeError("policy %q cannot emit phase %q", subscription.DeliveryPolicy, o.Phase)
	}
	if o.Phase == EventStreamBindingPhaseBackpressured && subscription.DeliveryPolicy == EventStreamDeliveryPolicyPauseSource && !o.SourceOutcomeDeclared {
		return eventStreamIncompatibleOutcomeError("policy %q requires source-declared outcome", subscription.DeliveryPolicy)
	}
	return nil
}

// EventStreamBindingProjection is a side-effect-free normalized view of
// source-owned catch-up and live event slices.
type EventStreamBindingProjection struct {
	Subscription EventStreamSubscription   `json:"subscription"`
	Outcome      EventStreamBindingOutcome `json:"outcome"`
	Events       []RealtimeEventEnvelope   `json:"events,omitempty"`
}

// ProjectEventStreamBinding validates and clones source data, deduplicating
// one bounded catch-up/live overlap with the canonical Realtime dedup key. It
// cannot create a transport, history store, cursor, global queue, or source
// state mutation.
func ProjectEventStreamBinding(subscription EventStreamSubscription, outcome EventStreamBindingOutcome, history, live []RealtimeEventEnvelope) (EventStreamBindingProjection, error) {
	if err := outcome.Validate(subscription); err != nil {
		return EventStreamBindingProjection{}, err
	}
	if len(history) > subscription.MaxBatchSize || len(live) > subscription.MaxBatchSize {
		return EventStreamBindingProjection{}, eventStreamValidationError("source event batch exceeds max_batch_size")
	}
	if !eventStreamPhaseAllowsEvents(outcome.Phase) && (len(history) != 0 || len(live) != 0) {
		return EventStreamBindingProjection{}, eventStreamValidationError("phase %q must not include source events", outcome.Phase)
	}

	lastSequence := int64(0)
	if subscription.StartMode == EventStreamStartAfterCursor {
		lastSequence = subscription.Cursor.Sequence
	}
	seen := make(map[string]struct{}, len(history)+len(live))
	normalized := make([]RealtimeEventEnvelope, 0, len(history)+len(live))
	for _, event := range append(append([]RealtimeEventEnvelope(nil), history...), live...) {
		if err := ValidateRealtimeEventEnvelope(event); err != nil {
			return EventStreamBindingProjection{}, err
		}
		if strings.TrimSpace(event.SessionID) != strings.TrimSpace(subscription.SessionID) || strings.TrimSpace(event.RunID) != strings.TrimSpace(subscription.RunID) {
			return EventStreamBindingProjection{}, eventStreamValidationError("source event correlation does not match subscription")
		}
		key := event.DedupKey()
		if _, exists := seen[key]; exists {
			continue
		}
		if lastSequence > 0 && event.Seq != lastSequence+1 {
			return EventStreamBindingProjection{}, eventStreamSequenceGapError("expected sequence %d, got %d", lastSequence+1, event.Seq)
		}
		seen[key] = struct{}{}
		normalized = append(normalized, cloneEventStreamEnvelope(event))
		lastSequence = event.Seq
	}
	if outcome.LastSequence > 0 && len(normalized) > 0 && outcome.LastSequence != lastSequence {
		return EventStreamBindingProjection{}, eventStreamValidationError("outcome last_sequence %d does not match normalized sequence %d", outcome.LastSequence, lastSequence)
	}
	projectedOutcome := outcome
	if len(normalized) > 0 {
		projectedOutcome.LastSequence = lastSequence
	}
	return EventStreamBindingProjection{
		Subscription: subscription,
		Outcome:      projectedOutcome,
		Events:       normalized,
	}, nil
}

func isValidEventStreamDeliveryPolicy(policy EventStreamDeliveryPolicy) bool {
	return policy == EventStreamDeliveryPolicyReject ||
		policy == EventStreamDeliveryPolicyDropWithRecord ||
		policy == EventStreamDeliveryPolicyPauseSource ||
		policy == EventStreamDeliveryPolicyUnknown
}

func isValidEventStreamBindingPhase(phase EventStreamBindingPhase) bool {
	switch phase {
	case EventStreamBindingPhaseAccepted,
		EventStreamBindingPhaseCatchingUp,
		EventStreamBindingPhaseLive,
		EventStreamBindingPhaseExpired,
		EventStreamBindingPhaseUnresolved,
		EventStreamBindingPhaseGap,
		EventStreamBindingPhaseBackpressured,
		EventStreamBindingPhaseDisconnected,
		EventStreamBindingPhaseClosed:
		return true
	default:
		return false
	}
}

func eventStreamPhaseAllowsEvents(phase EventStreamBindingPhase) bool {
	return phase == EventStreamBindingPhaseCatchingUp || phase == EventStreamBindingPhaseLive
}

func cloneEventStreamEnvelope(event RealtimeEventEnvelope) RealtimeEventEnvelope {
	return RealtimeEventEnvelope{
		EventID:   event.EventID,
		SessionID: event.SessionID,
		RunID:     event.RunID,
		Seq:       event.Seq,
		Type:      event.Type,
		TS:        event.TS,
		Payload:   cloneAnyPayloadMap(event.Payload),
	}
}

func eventStreamValidationError(format string, args ...any) error {
	return protocolValidationError(ProtocolReasonEventStreamInvalidSubscription+": "+format, args...)
}

func eventStreamIncompatibleOutcomeError(format string, args ...any) error {
	return protocolValidationError(ProtocolReasonEventStreamIncompatibleDeliveryOutcome+": "+format, args...)
}

func eventStreamSequenceGapError(format string, args ...any) error {
	return protocolValidationError(ProtocolReasonEventStreamSequenceGap+": "+format, args...)
}
