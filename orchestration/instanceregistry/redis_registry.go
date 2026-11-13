package instanceregistry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	defaultRedisInstancePrefix = "baymax:registry"
	instanceKeyPrefix          = "inst"
	sessionKeyPrefix           = "sess"
)

// RedisRegistry stores instances and session affinity with server-side TTLs.
// Expired keys disappear from Redis, so liveness never depends on a sweep.
type RedisRegistry struct {
	client redis.UniversalClient
	prefix string
}

// RedisRegistryConfig wires the injected client. The client is owned by the
// caller so one pool can serve the coordinator, mailbox and registry.
type RedisRegistryConfig struct {
	Client    redis.UniversalClient
	KeyPrefix string
}

func NewRedisRegistry(cfg RedisRegistryConfig) (*RedisRegistry, error) {
	if cfg.Client == nil {
		return nil, fmt.Errorf("%w: redis client is required", ErrAffinityUnavailable)
	}
	prefix := strings.TrimSpace(cfg.KeyPrefix)
	if prefix == "" {
		prefix = defaultRedisInstancePrefix
	}
	return &RedisRegistry{client: cfg.Client, prefix: prefix}, nil
}

func (r *RedisRegistry) instanceKey(instanceID string) string {
	return r.prefix + ":" + instanceKeyPrefix + ":" + instanceID
}

func (r *RedisRegistry) sessionKey(sessionID string) string {
	return r.prefix + ":" + sessionKeyPrefix + ":" + sessionID
}

func (r *RedisRegistry) Register(ctx context.Context, instanceID, endpoint string, ttl time.Duration) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("%w: registry is unavailable", ErrAffinityUnavailable)
	}
	id, err := normalizeInstanceID(instanceID)
	if err != nil {
		return err
	}
	if ttl <= 0 {
		return errors.New("instance registry: heartbeat ttl must be > 0")
	}
	now := time.Now().UTC()
	info := InstanceInfo{InstanceID: id, Endpoint: strings.TrimSpace(endpoint), StartedAt: now, LastSeenAt: now}
	if existing, err := r.client.Get(ctx, r.instanceKey(id)).Result(); err == nil {
		var previous InstanceInfo
		if json.Unmarshal([]byte(existing), &previous) == nil && !previous.StartedAt.IsZero() {
			info.StartedAt = previous.StartedAt
			info.LastSeenAt = now
		}
	} else if !errors.Is(err, redis.Nil) {
		return fmt.Errorf("%w: read instance heartbeat: %v", ErrAffinityUnavailable, err)
	}
	raw, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("%w: encode instance: %v", ErrAffinityUnavailable, err)
	}
	if err := r.client.Set(ctx, r.instanceKey(id), raw, ttl).Err(); err != nil {
		return fmt.Errorf("%w: write instance heartbeat: %v", ErrAffinityUnavailable, err)
	}
	return nil
}

func (r *RedisRegistry) List(ctx context.Context) ([]InstanceInfo, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("%w: registry is unavailable", ErrAffinityUnavailable)
	}
	pattern := r.prefix + ":" + instanceKeyPrefix + ":*"
	instances := []InstanceInfo{}
	var cursor uint64
	for {
		keys, next, err := r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, fmt.Errorf("%w: scan instances: %v", ErrAffinityUnavailable, err)
		}
		for _, key := range keys {
			raw, err := r.client.Get(ctx, key).Result()
			if err != nil {
				continue
			}
			var info InstanceInfo
			if json.Unmarshal([]byte(raw), &info) != nil {
				continue
			}
			instances = append(instances, info)
		}
		if next == 0 {
			break
		}
		cursor = next
	}
	return instances, nil
}

func (r *RedisRegistry) Unregister(ctx context.Context, instanceID string) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("%w: registry is unavailable", ErrAffinityUnavailable)
	}
	id, err := normalizeInstanceID(instanceID)
	if err != nil {
		return err
	}
	if _, err := r.client.Del(ctx, r.instanceKey(id)).Result(); err != nil {
		return fmt.Errorf("%w: unregister instance: %v", ErrAffinityUnavailable, err)
	}
	pattern := r.prefix + ":" + sessionKeyPrefix + ":*"
	var cursor uint64
	for {
		keys, next, err := r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("%w: scan session bindings: %v", ErrAffinityUnavailable, err)
		}
		for _, key := range keys {
			owner, err := r.client.Get(ctx, key).Result()
			if err != nil || owner != id {
				continue
			}
			_ = r.client.Del(ctx, key).Err()
		}
		if next == 0 {
			break
		}
		cursor = next
	}
	return nil
}

func (r *RedisRegistry) BindSession(ctx context.Context, sessionID, instanceID string, ttl time.Duration) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("%w: registry is unavailable", ErrAffinityUnavailable)
	}
	session, err := normalizeInstanceID(sessionID)
	if err != nil {
		return errors.New("instance registry: session id must be a bounded single token")
	}
	id, err := normalizeInstanceID(instanceID)
	if err != nil {
		return err
	}
	if ttl <= 0 {
		return errors.New("instance registry: affinity ttl must be > 0")
	}
	if _, err := r.client.Get(ctx, r.instanceKey(id)).Result(); errors.Is(err, redis.Nil) {
		return fmt.Errorf("%w: %s", ErrUnknownInstance, id)
	} else if err != nil {
		return fmt.Errorf("%w: read instance: %v", ErrAffinityUnavailable, err)
	}
	if err := r.client.Set(ctx, r.sessionKey(session), id, ttl).Err(); err != nil {
		return fmt.Errorf("%w: write session affinity: %v", ErrAffinityUnavailable, err)
	}
	return nil
}

func (r *RedisRegistry) ResolveSession(ctx context.Context, sessionID string) (string, error) {
	if r == nil || r.client == nil {
		return "", fmt.Errorf("%w: registry is unavailable", ErrAffinityUnavailable)
	}
	session, err := normalizeInstanceID(sessionID)
	if err != nil {
		return "", errors.New("instance registry: session id must be a bounded single token")
	}
	owner, err := r.client.Get(ctx, r.sessionKey(session)).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrSessionUnbound
	}
	if err != nil {
		return "", fmt.Errorf("%w: read session affinity: %v", ErrAffinityUnavailable, err)
	}
	id, err := normalizeInstanceID(owner)
	if err != nil {
		return "", fmt.Errorf("%w: session affinity references an invalid instance", ErrAffinityLost)
	}
	if _, err := r.client.Get(ctx, r.instanceKey(id)).Result(); errors.Is(err, redis.Nil) {
		return "", fmt.Errorf("%w: owning instance %s is gone", ErrAffinityLost, id)
	} else if err != nil {
		return "", fmt.Errorf("%w: read owning instance: %v", ErrAffinityUnavailable, err)
	}
	return id, nil
}

// Compile-time interface check.
var _ Registry = (*RedisRegistry)(nil)
