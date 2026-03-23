package mailbox

import (
	"fmt"
	"math"
	"strings"
	"time"
)

type mailboxState struct {
	Records     map[string]*Record  `json:"records"`
	Queue       []string            `json:"queue"`
	Idempotency map[string]string   `json:"idempotency"`
	Stats       Stats               `json:"stats"`
	Policy      Policy              `json:"policy"`
	sequence    map[string]struct{} `json:"-"`
	events      []LifecycleEvent    `json:"-"`
	traceEvents bool                `json:"-"`
}

func newMailboxState(backend string, policy Policy) mailboxState {
	return mailboxState{
		Records:     map[string]*Record{},
		Queue:       []string{},
		Idempotency: map[string]string{},
		Stats: Stats{
			Backend: strings.TrimSpace(backend),
		},
		Policy:      normalizePolicy(policy),
		sequence:    map[string]struct{}{},
		events:      []LifecycleEvent{},
		traceEvents: false,
	}
}

func (s *mailboxState) setFallback(reason string) {
	if s == nil {
		return
	}
	s.Stats.BackendFallback = true
	s.Stats.BackendFallbackReason = strings.TrimSpace(reason)
}

func (s *mailboxState) publish(envelope Envelope, now time.Time) (PublishResult, error) {
	normalized, err := normalizeEnvelope(envelope)
	if err != nil {
		return PublishResult{}, err
	}
	now = normalizeNow(now)
	if existingID, ok := s.Idempotency[normalized.IdempotencyKey]; ok {
		if existing, ok := s.Records[existingID]; ok && existing != nil {
			s.Stats.DuplicatePublishTotal++
			return PublishResult{Record: cloneRecord(*existing), Duplicate: true}, nil
		}
	}
	if existing, ok := s.Records[normalized.MessageID]; ok && existing != nil {
		s.Stats.DuplicatePublishTotal++
		return PublishResult{Record: cloneRecord(*existing), Duplicate: true}, nil
	}

	if normalized.ExpireAt.IsZero() && s.Policy.TTL > 0 {
		normalized.ExpireAt = now.Add(s.Policy.TTL).UTC()
	}
	rec := Record{
		Envelope:  normalized,
		State:     StateQueued,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.Records[normalized.MessageID] = &rec
	s.Queue = append(s.Queue, normalized.MessageID)
	s.Idempotency[normalized.IdempotencyKey] = normalized.MessageID
	s.Stats.PublishedTotal++
	s.Stats.QueueTotal++
	return PublishResult{Record: cloneRecord(rec)}, nil
}

func (s *mailboxState) consume(consumerID string, now time.Time) (Record, bool, bool, error) {
	consumerID = strings.TrimSpace(consumerID)
	if consumerID == "" {
		return Record{}, false, false, fmt.Errorf("consumer_id is required")
	}
	now = normalizeNow(now)
	mutated := s.expireQueued(now)

	selected := -1
	for i := range s.Queue {
		messageID := strings.TrimSpace(s.Queue[i])
		rec := s.Records[messageID]
		if rec == nil {
			continue
		}
		if rec.State != StateQueued {
			continue
		}
		if !rec.Envelope.NotBefore.IsZero() && rec.Envelope.NotBefore.After(now) {
			continue
		}
		if !rec.NextEligibleAt.IsZero() && rec.NextEligibleAt.After(now) {
			continue
		}
		selected = i
		break
	}
	if selected < 0 {
		return Record{}, false, mutated, nil
	}

	messageID := strings.TrimSpace(s.Queue[selected])
	s.Queue = append(s.Queue[:selected], s.Queue[selected+1:]...)
	rec := s.Records[messageID]
	if rec == nil {
		return Record{}, false, mutated, nil
	}
	rec.State = StateInFlight
	rec.ConsumerID = consumerID
	rec.DeliveryAttempt++
	rec.Envelope.Attempt = rec.DeliveryAttempt
	rec.NextEligibleAt = time.Time{}
	rec.UpdatedAt = now

	s.Stats.QueueTotal = maxInt(0, s.Stats.QueueTotal-1)
	s.Stats.InFlightTotal++
	s.Stats.ConsumedTotal++
	s.emitLifecycleEvent(TransitionConsume, *rec, "")
	return cloneRecord(*rec), true, true, nil
}

func (s *mailboxState) ack(messageID, consumerID string, now time.Time) (Record, error) {
	rec, err := s.requireInflightRecord(messageID, consumerID)
	if err != nil {
		if errorsIsAcked(rec) {
			return cloneRecord(*rec), nil
		}
		return Record{}, err
	}
	rec.State = StateAcked
	rec.UpdatedAt = normalizeNow(now)
	s.Stats.AckTotal++
	s.Stats.InFlightTotal = maxInt(0, s.Stats.InFlightTotal-1)
	s.emitLifecycleEvent(TransitionAck, *rec, "")
	return cloneRecord(*rec), nil
}

func (s *mailboxState) nack(messageID, consumerID, reason string, now time.Time) (Record, error) {
	rec, err := s.requireInflightRecord(messageID, consumerID)
	if err != nil {
		if rec != nil && (rec.State == StateNacked || rec.State == StateDeadLetter || rec.State == StateExpired) {
			return cloneRecord(*rec), nil
		}
		return Record{}, err
	}
	reasonCode := CanonicalizeLifecycleReason(reason, LifecycleReasonHandlerError)
	rec.State = StateNacked
	rec.LastError = reasonCode
	rec.UpdatedAt = normalizeNow(now)
	s.Stats.NackTotal++
	s.Stats.InFlightTotal = maxInt(0, s.Stats.InFlightTotal-1)
	s.emitLifecycleEvent(TransitionNack, *rec, reasonCode)
	if rec.DeliveryAttempt >= s.Policy.MaxAttempts && s.Policy.DLQEnabled {
		rec.State = StateDeadLetter
		rec.DeadLetterReason = deadLetterReason(LifecycleReasonRetryExhausted, reasonCode)
		s.Stats.DeadLetterTotal++
		s.emitLifecycleEvent(TransitionDeadLetter, *rec, LifecycleReasonRetryExhausted)
	}
	return cloneRecord(*rec), nil
}

func (s *mailboxState) requeue(messageID, consumerID, reason string, now time.Time) (Record, error) {
	messageID = strings.TrimSpace(messageID)
	consumerID = strings.TrimSpace(consumerID)
	rec := s.Records[messageID]
	if rec == nil {
		return Record{}, ErrMessageNotFound
	}
	reasonCode := CanonicalizeLifecycleReason(reason, LifecycleReasonHandlerError)
	switch rec.State {
	case StateInFlight:
		if consumerID != "" && rec.ConsumerID != consumerID {
			return Record{}, ErrConsumerMismatch
		}
		rec.State = StateNacked
		rec.LastError = reasonCode
		s.Stats.NackTotal++
		s.Stats.InFlightTotal = maxInt(0, s.Stats.InFlightTotal-1)
		s.emitLifecycleEvent(TransitionNack, *rec, reasonCode)
	case StateNacked:
	default:
		return Record{}, ErrMessageNotInflight
	}
	if rec.DeliveryAttempt >= s.Policy.MaxAttempts {
		if s.Policy.DLQEnabled {
			rec.State = StateDeadLetter
			rec.DeadLetterReason = deadLetterReason(LifecycleReasonRetryExhausted, reasonCode)
			rec.UpdatedAt = normalizeNow(now)
			s.Stats.DeadLetterTotal++
			s.emitLifecycleEvent(TransitionDeadLetter, *rec, LifecycleReasonRetryExhausted)
			return cloneRecord(*rec), nil
		}
		rec.State = StateExpired
		rec.DeadLetterReason = deadLetterReason(LifecycleReasonRetryExhausted, reasonCode)
		rec.UpdatedAt = normalizeNow(now)
		s.Stats.ExpiredTotal++
		s.emitLifecycleEvent(TransitionExpired, *rec, LifecycleReasonRetryExhausted)
		return cloneRecord(*rec), nil
	}
	rec.State = StateQueued
	rec.NextEligibleAt = normalizeNow(now).Add(s.retryDelay(rec.Envelope.MessageID, rec.DeliveryAttempt))
	rec.ConsumerID = ""
	rec.UpdatedAt = normalizeNow(now)
	s.Queue = append(s.Queue, rec.Envelope.MessageID)
	s.Stats.QueueTotal++
	s.Stats.RequeueTotal++
	s.emitLifecycleEvent(TransitionRequeue, *rec, reasonCode)
	return cloneRecord(*rec), nil
}

func (s *mailboxState) requireInflightRecord(messageID, consumerID string) (*Record, error) {
	messageID = strings.TrimSpace(messageID)
	consumerID = strings.TrimSpace(consumerID)
	rec := s.Records[messageID]
	if rec == nil {
		return nil, ErrMessageNotFound
	}
	if rec.State == StateAcked {
		return rec, ErrMessageNotInflight
	}
	if rec.State != StateInFlight {
		return rec, ErrMessageNotInflight
	}
	if consumerID != "" && rec.ConsumerID != consumerID {
		return rec, ErrConsumerMismatch
	}
	return rec, nil
}

func (s *mailboxState) retryDelay(messageID string, attempt int) time.Duration {
	policy := normalizePolicy(s.Policy)
	delay := policy.BackoffInitial
	if delay <= 0 {
		return 0
	}
	if attempt <= 1 {
		return delay
	}
	for i := 1; i < attempt; i++ {
		next := time.Duration(float64(delay) * 2.0)
		if next > policy.BackoffMax {
			delay = policy.BackoffMax
			break
		}
		delay = next
	}
	if policy.JitterRatio <= 0 {
		return delay
	}
	jitterWindow := int64(float64(delay) * policy.JitterRatio)
	if jitterWindow <= 0 {
		return delay
	}
	seed := stableJitterSeed(messageID, attempt)
	shifted := int64(delay) + (seed%(2*jitterWindow+1) - jitterWindow)
	if shifted < 0 {
		shifted = 0
	}
	if shifted > int64(policy.BackoffMax) {
		shifted = int64(policy.BackoffMax)
	}
	return time.Duration(shifted)
}

func (s *mailboxState) expireQueued(now time.Time) bool {
	if len(s.Queue) == 0 {
		return false
	}
	mutated := false
	filtered := make([]string, 0, len(s.Queue))
	for i := range s.Queue {
		id := strings.TrimSpace(s.Queue[i])
		rec := s.Records[id]
		if rec == nil {
			continue
		}
		if rec.State != StateQueued {
			continue
		}
		if !isRecordExpired(*rec, now) {
			filtered = append(filtered, id)
			continue
		}
		mutated = true
		rec.UpdatedAt = now
		s.Stats.QueueTotal = maxInt(0, s.Stats.QueueTotal-1)
		s.Stats.ExpiredTotal++
		if s.Policy.DLQEnabled {
			rec.State = StateDeadLetter
			rec.DeadLetterReason = deadLetterReason(LifecycleReasonExpired, "")
			s.Stats.DeadLetterTotal++
			s.emitLifecycleEvent(TransitionDeadLetter, *rec, LifecycleReasonExpired)
		} else {
			rec.State = StateExpired
		}
		s.emitLifecycleEvent(TransitionExpired, *rec, LifecycleReasonExpired)
	}
	s.Queue = filtered
	return mutated
}

func (s *mailboxState) snapshot() Snapshot {
	records := make([]Record, 0, len(s.Records))
	for _, rec := range s.Records {
		if rec == nil {
			continue
		}
		records = append(records, cloneRecord(*rec))
	}
	sortRecordsDeterministic(records)
	idempotency := make(map[string]string, len(s.Idempotency))
	for k, v := range s.Idempotency {
		idempotency[k] = v
	}
	queue := append([]string(nil), s.Queue...)
	return Snapshot{
		Backend:     s.Stats.Backend,
		Records:     records,
		Queue:       queue,
		Idempotency: idempotency,
		Stats:       s.Stats,
		Policy:      s.Policy,
	}
}

func (s *mailboxState) restore(snapshot Snapshot) error {
	normalized, err := normalizeSnapshot(snapshot, s.Stats.Backend)
	if err != nil {
		return err
	}
	records := make(map[string]*Record, len(normalized.Records))
	inFlightTotal := 0
	for i := range normalized.Records {
		rec := cloneRecord(normalized.Records[i])
		records[rec.Envelope.MessageID] = &rec
		if rec.State == StateInFlight {
			inFlightTotal++
		}
	}
	s.Records = records
	s.Queue = append([]string(nil), normalized.Queue...)
	s.Idempotency = make(map[string]string, len(normalized.Idempotency))
	for k, v := range normalized.Idempotency {
		s.Idempotency[k] = v
	}
	s.Policy = normalizePolicy(normalized.Policy)
	s.Stats = normalized.Stats
	s.Stats.QueueTotal = len(s.Queue)
	s.Stats.InFlightTotal = inFlightTotal
	return nil
}

func stableJitterSeed(messageID string, attempt int) int64 {
	raw := strings.TrimSpace(messageID) + "|" + fmt.Sprintf("%d", attempt)
	var h uint64 = 1469598103934665603
	const prime uint64 = 1099511628211
	for i := 0; i < len(raw); i++ {
		h ^= uint64(raw[i])
		h *= prime
	}
	return int64(h & math.MaxInt64)
}

func deadLetterReason(code, reason string) string {
	trimmedCode := strings.TrimSpace(code)
	trimmedReason := strings.TrimSpace(reason)
	if trimmedReason == "" {
		return trimmedCode
	}
	return trimmedCode + ":" + trimmedReason
}

func (s *mailboxState) emitLifecycleEvent(transition LifecycleTransition, rec Record, reason string) {
	if s == nil {
		return
	}
	if !s.traceEvents {
		return
	}
	event := LifecycleEvent{
		Time:       normalizeNow(rec.UpdatedAt),
		Transition: transition,
		Record:     cloneRecord(rec),
	}
	switch transition {
	case TransitionNack, TransitionRequeue, TransitionDeadLetter, TransitionExpired:
		event.ReasonCode = CanonicalizeLifecycleReason(reason, LifecycleReasonHandlerError)
	}
	if transition == TransitionDeadLetter && strings.TrimSpace(event.ReasonCode) == "" {
		event.ReasonCode = LifecycleReasonRetryExhausted
	}
	if transition == TransitionExpired && strings.TrimSpace(event.ReasonCode) == "" {
		event.ReasonCode = LifecycleReasonExpired
	}
	s.events = append(s.events, event)
}

func (s *mailboxState) drainLifecycleEvents() []LifecycleEvent {
	if s == nil || len(s.events) == 0 {
		return nil
	}
	out := make([]LifecycleEvent, len(s.events))
	copy(out, s.events)
	s.events = s.events[:0]
	return out
}

func (s *mailboxState) setTraceEvents(enabled bool) {
	if s == nil {
		return
	}
	s.traceEvents = enabled
	if !enabled {
		s.events = s.events[:0]
	}
}

func isRecordExpired(record Record, now time.Time) bool {
	if record.Envelope.ExpireAt.IsZero() {
		return false
	}
	return !record.Envelope.ExpireAt.After(now)
}

func normalizeNow(now time.Time) time.Time {
	if now.IsZero() {
		now = time.Now()
	}
	return now.UTC()
}

func errorsIsAcked(rec *Record) bool {
	return rec != nil && rec.State == StateAcked
}
