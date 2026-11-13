package mailbox

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore keeps the mailboxState machine byte-identical to the memory and
// file backends and uses Redis for two things only: durable persistence of the
// serialized state document and a short-lived distributed lock that serializes
// mutations across instances. Inflight ownership, reclaim, retry backoff and
// dead-letter semantics therefore cannot drift between backends by
// construction; runtime unavailability fails closed through ErrRedisUnavailable.
type RedisStore struct {
	mu sync.Mutex

	client   redis.UniversalClient
	policy   Policy
	backend  string
	stateKey string
	lockKey  string

	lockTTL       time.Duration
	opTimeout     time.Duration
	lockPoll      time.Duration
	traceEvents   bool
	pendingEvents []LifecycleEvent
}

// RedisStoreConfig wires the injected Redis client. The client is owned by the
// caller so one connection pool can serve the coordinator, mailbox and
// instance registry without cross-package lifecycle coupling.
type RedisStoreConfig struct {
	Client      redis.UniversalClient
	KeyPrefix   string
	Namespace   string
	LockTTL     time.Duration
	OpTimeout   time.Duration
	LockPoll    time.Duration
	CloseClient bool
}

const (
	RedisBackend = "redis"

	defaultRedisMailboxPrefix = "baymax:mailbox"
	defaultRedisNamespace     = "default"
	defaultRedisLockTTL       = 10 * time.Second
	defaultRedisOpTimeout     = 5 * time.Second
	defaultRedisLockPoll      = 10 * time.Millisecond
)

var ErrRedisUnavailable = errors.New("mailbox redis backend unavailable")

const releaseRedisLockScript = `
if redis.call('GET', KEYS[1]) == ARGV[1] then
  return redis.call('DEL', KEYS[1])
end
return 0
`

func NewRedisStore(cfg RedisStoreConfig, policy Policy) (*RedisStore, error) {
	if cfg.Client == nil {
		return nil, fmt.Errorf("%w: redis client is required", ErrRedisUnavailable)
	}
	namespace := strings.TrimSpace(cfg.Namespace)
	if namespace == "" {
		namespace = defaultRedisNamespace
	}
	prefix := strings.TrimSpace(cfg.KeyPrefix)
	if prefix == "" {
		prefix = defaultRedisMailboxPrefix
	}
	lockTTL := cfg.LockTTL
	if lockTTL <= 0 {
		lockTTL = defaultRedisLockTTL
	}
	opTimeout := cfg.OpTimeout
	if opTimeout <= 0 {
		opTimeout = defaultRedisOpTimeout
	}
	poll := cfg.LockPoll
	if poll <= 0 {
		poll = defaultRedisLockPoll
	}
	base := prefix + ":" + namespace
	return &RedisStore{
		client:    cfg.Client,
		policy:    normalizePolicy(policy),
		backend:   RedisBackend,
		stateKey:  base + ":state",
		lockKey:   base + ":lock",
		lockTTL:   lockTTL,
		opTimeout: opTimeout,
		lockPoll:  poll,
	}, nil
}

func (s *RedisStore) Backend() string {
	if s == nil {
		return RedisBackend
	}
	return s.backend
}

func (s *RedisStore) MarkFallback(reason string) {
	if s == nil {
		return
	}
	_ = s.mutate(context.Background(), func(state *mailboxState) error {
		state.setFallback(reason)
		return nil
	})
}

func (s *RedisStore) Publish(ctx context.Context, envelope Envelope, now time.Time) (PublishResult, error) {
	if s == nil {
		return PublishResult{}, ErrRedisUnavailable
	}
	var result PublishResult
	err := s.mutate(ctx, func(state *mailboxState) error {
		out, err := state.publish(envelope, now)
		result = out
		return err
	})
	if err != nil {
		return PublishResult{}, err
	}
	return result, nil
}

func (s *RedisStore) Consume(
	ctx context.Context,
	consumerID string,
	now time.Time,
	inflightTimeout time.Duration,
	reclaimOnConsume bool,
) (Record, bool, error) {
	if s == nil {
		return Record{}, false, ErrRedisUnavailable
	}
	var (
		record Record
		ok     bool
	)
	err := s.mutate(ctx, func(state *mailboxState) error {
		out, consumed, _, err := state.consume(consumerID, now, inflightTimeout, reclaimOnConsume)
		record = out
		ok = consumed
		return err
	})
	if err != nil {
		return Record{}, false, err
	}
	return record, ok, nil
}

func (s *RedisStore) Heartbeat(
	ctx context.Context,
	messageID, consumerID string,
	now time.Time,
	inflightTimeout time.Duration,
) (Record, error) {
	if s == nil {
		return Record{}, ErrRedisUnavailable
	}
	var record Record
	err := s.mutate(ctx, func(state *mailboxState) error {
		out, err := state.heartbeat(messageID, consumerID, now, inflightTimeout)
		record = out
		return err
	})
	if err != nil {
		return Record{}, err
	}
	return record, nil
}

func (s *RedisStore) Ack(ctx context.Context, messageID, consumerID string, now time.Time) (Record, error) {
	if s == nil {
		return Record{}, ErrRedisUnavailable
	}
	var record Record
	err := s.mutate(ctx, func(state *mailboxState) error {
		out, err := state.ack(messageID, consumerID, now)
		record = out
		return err
	})
	if err != nil {
		return Record{}, err
	}
	return record, nil
}

func (s *RedisStore) Nack(
	ctx context.Context,
	messageID, consumerID, reason string,
	now time.Time,
	opts ActionOptions,
) (Record, error) {
	if s == nil {
		return Record{}, ErrRedisUnavailable
	}
	var record Record
	err := s.mutate(ctx, func(state *mailboxState) error {
		out, err := state.nack(messageID, consumerID, reason, now, opts)
		record = out
		return err
	})
	if err != nil {
		return Record{}, err
	}
	return record, nil
}

func (s *RedisStore) Requeue(
	ctx context.Context,
	messageID, consumerID, reason string,
	now time.Time,
	opts ActionOptions,
) (Record, error) {
	if s == nil {
		return Record{}, ErrRedisUnavailable
	}
	var record Record
	err := s.mutate(ctx, func(state *mailboxState) error {
		out, err := state.requeue(messageID, consumerID, reason, now, opts)
		record = out
		return err
	})
	if err != nil {
		return Record{}, err
	}
	return record, nil
}

// Stats reads the persisted state without taking the mutation lock; a single
// key read is atomic, so the projection is always one consistent revision.
func (s *RedisStore) Stats(ctx context.Context) (Stats, error) {
	if s == nil {
		return Stats{}, ErrRedisUnavailable
	}
	state, err := s.loadShared(ctx)
	if err != nil {
		return Stats{}, err
	}
	return state.Stats, nil
}

// Snapshot reads the persisted state without mutating it.
func (s *RedisStore) Snapshot(ctx context.Context) (Snapshot, error) {
	if s == nil {
		return Snapshot{}, ErrRedisUnavailable
	}
	state, err := s.loadShared(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	return state.snapshot(), nil
}

// Restore rewrites the durable state inside the mutation lock.
func (s *RedisStore) Restore(ctx context.Context, snapshot Snapshot) error {
	if s == nil {
		return ErrRedisUnavailable
	}
	return s.mutate(ctx, func(state *mailboxState) error {
		return state.restore(snapshot)
	})
}

func (s *RedisStore) continuationSnapshot(ctx context.Context, fn func(Snapshot) error) error {
	if s == nil {
		return ErrRedisUnavailable
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	state, err := s.loadShared(ctx)
	if err != nil {
		return err
	}
	return fn(state.snapshot())
}

func (s *RedisStore) continuationMutate(ctx context.Context, fn func(*mailboxState) error) error {
	if s == nil {
		return ErrRedisUnavailable
	}
	return s.mutate(ctx, fn)
}

func (s *RedisStore) DrainLifecycleEvents() []LifecycleEvent {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	events := s.pendingEvents
	s.pendingEvents = nil
	return events
}

func (s *RedisStore) SetLifecycleTracing(enabled bool) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.traceEvents = enabled
	if !enabled {
		s.pendingEvents = nil
	}
}

// mutate serializes one state-machine mutation across instances: acquire the
// distributed lock, load the durable state, apply fn, persist, release.
func (s *RedisStore) mutate(ctx context.Context, fn func(*mailboxState) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	unlock, err := s.acquireLock(ctx)
	if err != nil {
		return err
	}
	state, err := s.loadLocked(ctx)
	if err != nil {
		_ = unlock(ctx)
		return err
	}
	state.traceEvents = s.currentTraceEvents()
	if err := fn(state); err != nil {
		_ = unlock(ctx)
		return err
	}
	if err := s.persistLocked(ctx, state); err != nil {
		_ = unlock(ctx)
		return err
	}
	s.collectEvents(state)
	return unlock(ctx)
}

// loadShared decodes the durable state without the mutation lock.
func (s *RedisStore) loadShared(ctx context.Context) (mailboxState, error) {
	if err := ctx.Err(); err != nil {
		return mailboxState{}, err
	}
	raw, err := s.client.Get(ctx, s.stateKey).Result()
	if errors.Is(err, redis.Nil) {
		return newMailboxState(s.backend, s.policy), nil
	}
	if err != nil {
		return mailboxState{}, fmt.Errorf("%w: load mailbox state: %v", ErrRedisUnavailable, err)
	}
	return decodeMailboxState(raw, s.backend, s.policy)
}

func (s *RedisStore) loadLocked(ctx context.Context) (*mailboxState, error) {
	state, err := s.loadShared(ctx)
	if err != nil {
		return nil, err
	}
	return &state, nil
}

func (s *RedisStore) persistLocked(ctx context.Context, state *mailboxState) error {
	snapshot := state.snapshot()
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("%w: encode mailbox state: %v", ErrRedisUnavailable, err)
	}
	if err := s.client.Set(ctx, s.stateKey, raw, 0).Err(); err != nil {
		return fmt.Errorf("%w: persist mailbox state: %v", ErrRedisUnavailable, err)
	}
	return nil
}

func decodeMailboxState(raw, backend string, policy Policy) (mailboxState, error) {
	var snapshot Snapshot
	if err := json.Unmarshal([]byte(raw), &snapshot); err != nil {
		return mailboxState{}, fmt.Errorf("%w: decode mailbox state: %v", ErrRedisUnavailable, err)
	}
	state := newMailboxState(backend, policy)
	if err := state.restore(snapshot); err != nil {
		return mailboxState{}, fmt.Errorf("%w: restore mailbox state: %v", ErrRedisUnavailable, err)
	}
	return state, nil
}

func (s *RedisStore) currentTraceEvents() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.traceEvents
}

func (s *RedisStore) collectEvents(state *mailboxState) {
	events := state.drainLifecycleEvents()
	if len(events) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pendingEvents = append(s.pendingEvents, events...)
}

func (s *RedisStore) acquireLock(ctx context.Context) (func(context.Context) error, error) {
	token, err := randomRedisToken()
	if err != nil {
		return nil, fmt.Errorf("%w: lock token: %v", ErrRedisUnavailable, err)
	}
	deadline := s.opTimeout
	if existing, ok := ctx.Deadline(); ok {
		if remaining := time.Until(existing); remaining < deadline {
			deadline = remaining
		}
	}
	if deadline <= 0 {
		deadline = time.Millisecond
	}
	expires := time.Now().Add(deadline)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		ok, err := s.client.SetNX(ctx, s.lockKey, token, s.lockTTL).Result()
		if err == nil && ok {
			return s.releaseLockFunc(token), nil
		}
		if err != nil && ctx.Err() == nil {
			return nil, fmt.Errorf("%w: acquire mailbox lock: %v", ErrRedisUnavailable, err)
		}
		if !time.Now().Before(expires) {
			return nil, fmt.Errorf("%w: mailbox lock timeout", ErrRedisUnavailable)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(s.lockPoll):
		}
	}
}

func (s *RedisStore) releaseLockFunc(token string) func(context.Context) error {
	return func(ctx context.Context) error {
		if s == nil || s.client == nil {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		return s.client.Eval(ctx, releaseRedisLockScript, []string{s.lockKey}, token).Err()
	}
}

func randomRedisToken() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}

// Compile-time interface checks.
var (
	_ Store = (*RedisStore)(nil)
	_ interface {
		continuationSnapshot(context.Context, func(Snapshot) error) error
		continuationMutate(context.Context, func(*mailboxState) error) error
	} = (*RedisStore)(nil)
)
