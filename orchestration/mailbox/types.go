package mailbox

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

type EnvelopeKind string

const (
	KindCommand EnvelopeKind = "command"
	KindEvent   EnvelopeKind = "event"
	KindResult  EnvelopeKind = "result"
)

const (
	WorkerHandlerErrorPolicyRequeue = "requeue"
	WorkerHandlerErrorPolicyNack    = "nack"
	DefaultWorkerPollInterval       = 100 * time.Millisecond
	DefaultWorkerConsumerID         = "mailbox-worker"
)

const (
	LifecycleReasonRetryExhausted   = "retry_exhausted"
	LifecycleReasonExpired          = "expired"
	LifecycleReasonConsumerMismatch = "consumer_mismatch"
	LifecycleReasonMessageNotFound  = "message_not_found"
	LifecycleReasonHandlerError     = "handler_error"
)

type MessageState string

const (
	StateQueued     MessageState = "queued"
	StateInFlight   MessageState = "in_flight"
	StateAcked      MessageState = "acked"
	StateNacked     MessageState = "nacked"
	StateDeadLetter MessageState = "dead_letter"
	StateExpired    MessageState = "expired"
)

type Envelope struct {
	MessageID      string         `json:"message_id"`
	IdempotencyKey string         `json:"idempotency_key"`
	CorrelationID  string         `json:"correlation_id,omitempty"`
	Kind           EnvelopeKind   `json:"kind"`
	FromAgent      string         `json:"from_agent,omitempty"`
	ToAgent        string         `json:"to_agent,omitempty"`
	TaskID         string         `json:"task_id,omitempty"`
	RunID          string         `json:"run_id,omitempty"`
	WorkflowID     string         `json:"workflow_id,omitempty"`
	TeamID         string         `json:"team_id,omitempty"`
	Payload        map[string]any `json:"payload,omitempty"`
	NotBefore      time.Time      `json:"not_before,omitempty"`
	ExpireAt       time.Time      `json:"expire_at,omitempty"`
	Attempt        int            `json:"attempt,omitempty"`
}

type Record struct {
	Envelope         Envelope     `json:"envelope"`
	State            MessageState `json:"state"`
	ConsumerID       string       `json:"consumer_id,omitempty"`
	DeliveryAttempt  int          `json:"delivery_attempt,omitempty"`
	NextEligibleAt   time.Time    `json:"next_eligible_at,omitempty"`
	LastError        string       `json:"last_error,omitempty"`
	DeadLetterReason string       `json:"dead_letter_reason,omitempty"`
	CreatedAt        time.Time    `json:"created_at"`
	UpdatedAt        time.Time    `json:"updated_at"`
}

type Stats struct {
	Backend               string `json:"backend"`
	BackendFallback       bool   `json:"backend_fallback,omitempty"`
	BackendFallbackReason string `json:"backend_fallback_reason,omitempty"`
	QueueTotal            int    `json:"queue_total"`
	InFlightTotal         int    `json:"in_flight_total"`
	PublishedTotal        int    `json:"published_total"`
	DuplicatePublishTotal int    `json:"duplicate_publish_total"`
	ConsumedTotal         int    `json:"consumed_total"`
	AckTotal              int    `json:"ack_total"`
	NackTotal             int    `json:"nack_total"`
	RequeueTotal          int    `json:"requeue_total"`
	ExpiredTotal          int    `json:"expired_total"`
	DeadLetterTotal       int    `json:"dead_letter_total"`
}

type Policy struct {
	MaxAttempts    int           `json:"max_attempts"`
	BackoffInitial time.Duration `json:"backoff_initial"`
	BackoffMax     time.Duration `json:"backoff_max"`
	JitterRatio    float64       `json:"jitter_ratio"`
	TTL            time.Duration `json:"ttl"`
	DLQEnabled     bool          `json:"dlq_enabled"`
}

type Snapshot struct {
	Backend     string            `json:"backend"`
	Records     []Record          `json:"records"`
	Queue       []string          `json:"queue"`
	Idempotency map[string]string `json:"idempotency,omitempty"`
	Stats       Stats             `json:"stats"`
	Policy      Policy            `json:"policy"`
}

type PublishResult struct {
	Record    Record `json:"record"`
	Duplicate bool   `json:"duplicate"`
}

type WorkerConfig struct {
	Enabled            bool          `json:"enabled"`
	PollInterval       time.Duration `json:"poll_interval"`
	HandlerErrorPolicy string        `json:"handler_error_policy"`
}

type LifecycleTransition string

const (
	TransitionConsume    LifecycleTransition = "consume"
	TransitionAck        LifecycleTransition = "ack"
	TransitionNack       LifecycleTransition = "nack"
	TransitionRequeue    LifecycleTransition = "requeue"
	TransitionDeadLetter LifecycleTransition = "dead_letter"
	TransitionExpired    LifecycleTransition = "expired"
)

type LifecycleEvent struct {
	Time       time.Time           `json:"time"`
	Transition LifecycleTransition `json:"transition"`
	Record     Record              `json:"record"`
	ReasonCode string              `json:"reason_code,omitempty"`
}

type LifecycleObserver func(ctx context.Context, event LifecycleEvent)

type Store interface {
	Backend() string
	Publish(ctx context.Context, envelope Envelope, now time.Time) (PublishResult, error)
	Consume(ctx context.Context, consumerID string, now time.Time) (Record, bool, error)
	Ack(ctx context.Context, messageID, consumerID string, now time.Time) (Record, error)
	Nack(ctx context.Context, messageID, consumerID, reason string, now time.Time) (Record, error)
	Requeue(ctx context.Context, messageID, consumerID, reason string, now time.Time) (Record, error)
	Stats(ctx context.Context) (Stats, error)
	Snapshot(ctx context.Context) (Snapshot, error)
	Restore(ctx context.Context, snapshot Snapshot) error
}

var (
	ErrMessageNotFound    = errors.New("mailbox message not found")
	ErrMessageNotInflight = errors.New("mailbox message is not in-flight")
	ErrConsumerMismatch   = errors.New("mailbox consumer mismatch")
	ErrSnapshotCorrupt    = errors.New("mailbox snapshot is corrupt")
)

func normalizeEnvelope(in Envelope) (Envelope, error) {
	out := in
	out.MessageID = strings.TrimSpace(out.MessageID)
	out.IdempotencyKey = strings.TrimSpace(out.IdempotencyKey)
	out.CorrelationID = strings.TrimSpace(out.CorrelationID)
	out.FromAgent = strings.TrimSpace(out.FromAgent)
	out.ToAgent = strings.TrimSpace(out.ToAgent)
	out.TaskID = strings.TrimSpace(out.TaskID)
	out.RunID = strings.TrimSpace(out.RunID)
	out.WorkflowID = strings.TrimSpace(out.WorkflowID)
	out.TeamID = strings.TrimSpace(out.TeamID)
	out.Attempt = maxInt(out.Attempt, 0)
	out.Payload = copyMap(out.Payload)
	if !out.NotBefore.IsZero() {
		out.NotBefore = out.NotBefore.UTC()
	}
	if !out.ExpireAt.IsZero() {
		out.ExpireAt = out.ExpireAt.UTC()
	}
	if out.MessageID == "" {
		return Envelope{}, errors.New("message_id is required")
	}
	if out.IdempotencyKey == "" {
		return Envelope{}, errors.New("idempotency_key is required")
	}
	switch out.Kind {
	case KindCommand, KindEvent, KindResult:
	default:
		return Envelope{}, fmt.Errorf("kind must be one of [%s,%s,%s], got %q", KindCommand, KindEvent, KindResult, out.Kind)
	}
	return out, nil
}

func normalizePolicy(in Policy) Policy {
	out := in
	if out.MaxAttempts <= 0 {
		out.MaxAttempts = 3
	}
	if out.BackoffInitial < 0 {
		out.BackoffInitial = 0
	}
	if out.BackoffInitial == 0 {
		out.BackoffInitial = 50 * time.Millisecond
	}
	if out.BackoffMax < out.BackoffInitial {
		out.BackoffMax = out.BackoffInitial
	}
	if out.BackoffMax == 0 {
		out.BackoffMax = 500 * time.Millisecond
	}
	if out.JitterRatio < 0 {
		out.JitterRatio = 0
	}
	if out.JitterRatio > 1 {
		out.JitterRatio = 1
	}
	if out.TTL < 0 {
		out.TTL = 0
	}
	return out
}

func NormalizeWorkerConfig(in WorkerConfig) (WorkerConfig, error) {
	out := in
	if out.PollInterval == 0 {
		out.PollInterval = DefaultWorkerPollInterval
	}
	if out.PollInterval < 0 {
		return WorkerConfig{}, errors.New("worker.poll_interval must be > 0")
	}
	if out.PollInterval <= 0 {
		return WorkerConfig{}, errors.New("worker.poll_interval must be > 0")
	}
	out.HandlerErrorPolicy = strings.ToLower(strings.TrimSpace(out.HandlerErrorPolicy))
	if out.HandlerErrorPolicy == "" {
		out.HandlerErrorPolicy = WorkerHandlerErrorPolicyRequeue
	}
	switch out.HandlerErrorPolicy {
	case WorkerHandlerErrorPolicyRequeue, WorkerHandlerErrorPolicyNack:
	default:
		return WorkerConfig{}, fmt.Errorf(
			"worker.handler_error_policy must be one of [%s,%s], got %q",
			WorkerHandlerErrorPolicyRequeue,
			WorkerHandlerErrorPolicyNack,
			in.HandlerErrorPolicy,
		)
	}
	return out, nil
}

func LifecycleCanonicalReasons() []string {
	return []string{
		LifecycleReasonRetryExhausted,
		LifecycleReasonExpired,
		LifecycleReasonConsumerMismatch,
		LifecycleReasonMessageNotFound,
		LifecycleReasonHandlerError,
	}
}

func NormalizeLifecycleReason(in string) string {
	reason := strings.ToLower(strings.TrimSpace(in))
	if reason == "" {
		return ""
	}
	if cut := strings.Index(reason, ":"); cut >= 0 {
		reason = strings.TrimSpace(reason[:cut])
	}
	return reason
}

func IsCanonicalLifecycleReason(in string) bool {
	reason := NormalizeLifecycleReason(in)
	switch reason {
	case LifecycleReasonRetryExhausted,
		LifecycleReasonExpired,
		LifecycleReasonConsumerMismatch,
		LifecycleReasonMessageNotFound,
		LifecycleReasonHandlerError:
		return true
	default:
		return false
	}
}

func LifecycleReasonFromError(err error) string {
	switch {
	case errors.Is(err, ErrConsumerMismatch):
		return LifecycleReasonConsumerMismatch
	case errors.Is(err, ErrMessageNotFound):
		return LifecycleReasonMessageNotFound
	default:
		return LifecycleReasonHandlerError
	}
}

func CanonicalizeLifecycleReason(in, fallback string) string {
	if reason := NormalizeLifecycleReason(in); IsCanonicalLifecycleReason(reason) {
		return reason
	}
	if reason := NormalizeLifecycleReason(fallback); IsCanonicalLifecycleReason(reason) {
		return reason
	}
	return LifecycleReasonHandlerError
}

func cloneRecord(in Record) Record {
	out := in
	out.Envelope = in.Envelope
	out.Envelope.Payload = copyMap(in.Envelope.Payload)
	return out
}

func normalizeSnapshot(in Snapshot, fallbackBackend string) (Snapshot, error) {
	out := in
	backend := strings.TrimSpace(out.Backend)
	if backend == "" {
		backend = strings.TrimSpace(fallbackBackend)
	}
	if backend == "" {
		backend = "memory"
	}
	out.Backend = backend
	out.Policy = normalizePolicy(out.Policy)
	if out.Idempotency == nil {
		out.Idempotency = map[string]string{}
	}
	records := make([]Record, 0, len(out.Records))
	seenMessage := make(map[string]struct{}, len(out.Records))
	for i := range out.Records {
		rec := out.Records[i]
		env, err := normalizeEnvelope(rec.Envelope)
		if err != nil {
			return Snapshot{}, fmt.Errorf("%w: records[%d] invalid envelope: %v", ErrSnapshotCorrupt, i, err)
		}
		rec.Envelope = env
		rec.ConsumerID = strings.TrimSpace(rec.ConsumerID)
		rec.LastError = strings.TrimSpace(rec.LastError)
		rec.DeadLetterReason = strings.TrimSpace(rec.DeadLetterReason)
		switch rec.State {
		case StateQueued, StateInFlight, StateAcked, StateNacked, StateDeadLetter, StateExpired:
		default:
			return Snapshot{}, fmt.Errorf("%w: records[%d] has unsupported state %q", ErrSnapshotCorrupt, i, rec.State)
		}
		if rec.DeliveryAttempt < 0 {
			return Snapshot{}, fmt.Errorf("%w: records[%d].delivery_attempt must be >= 0", ErrSnapshotCorrupt, i)
		}
		if rec.CreatedAt.IsZero() {
			return Snapshot{}, fmt.Errorf("%w: records[%d].created_at is required", ErrSnapshotCorrupt, i)
		}
		if rec.UpdatedAt.IsZero() {
			rec.UpdatedAt = rec.CreatedAt
		}
		if _, ok := seenMessage[rec.Envelope.MessageID]; ok {
			return Snapshot{}, fmt.Errorf("%w: duplicate message_id %q", ErrSnapshotCorrupt, rec.Envelope.MessageID)
		}
		seenMessage[rec.Envelope.MessageID] = struct{}{}
		records = append(records, cloneRecord(rec))
	}

	queue := make([]string, 0, len(out.Queue))
	queuedSet := map[string]struct{}{}
	for i := range out.Queue {
		id := strings.TrimSpace(out.Queue[i])
		if id == "" {
			continue
		}
		if _, ok := queuedSet[id]; ok {
			return Snapshot{}, fmt.Errorf("%w: duplicate queue message_id %q", ErrSnapshotCorrupt, id)
		}
		queuedSet[id] = struct{}{}
		queue = append(queue, id)
	}
	idSet := map[string]MessageState{}
	for i := range records {
		idSet[records[i].Envelope.MessageID] = records[i].State
	}
	for id := range queuedSet {
		state, ok := idSet[id]
		if !ok {
			return Snapshot{}, fmt.Errorf("%w: queue references unknown message_id %q", ErrSnapshotCorrupt, id)
		}
		if state != StateQueued {
			return Snapshot{}, fmt.Errorf("%w: queue message %q must be queued", ErrSnapshotCorrupt, id)
		}
	}

	for key, messageID := range out.Idempotency {
		trimmedKey := strings.TrimSpace(key)
		trimmedMessageID := strings.TrimSpace(messageID)
		if trimmedKey == "" || trimmedMessageID == "" {
			return Snapshot{}, fmt.Errorf("%w: idempotency map contains empty key/value", ErrSnapshotCorrupt)
		}
		if _, ok := idSet[trimmedMessageID]; !ok {
			return Snapshot{}, fmt.Errorf("%w: idempotency key %q references unknown message_id %q", ErrSnapshotCorrupt, trimmedKey, trimmedMessageID)
		}
		if trimmedKey != key || trimmedMessageID != messageID {
			delete(out.Idempotency, key)
			out.Idempotency[trimmedKey] = trimmedMessageID
		}
	}

	out.Records = records
	out.Queue = queue
	out.Stats.Backend = backend
	return out, nil
}

func sortRecordsDeterministic(records []Record) {
	sort.SliceStable(records, func(i, j int) bool {
		left := records[i]
		right := records[j]
		if left.UpdatedAt.Equal(right.UpdatedAt) {
			if left.CreatedAt.Equal(right.CreatedAt) {
				return strings.TrimSpace(left.Envelope.MessageID) < strings.TrimSpace(right.Envelope.MessageID)
			}
			return left.CreatedAt.Before(right.CreatedAt)
		}
		return left.UpdatedAt.Before(right.UpdatedAt)
	})
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

func maxInt(a, b int) int {
	if a >= b {
		return a
	}
	return b
}
