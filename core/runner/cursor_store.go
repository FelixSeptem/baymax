package runner

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// CursorStore persists realtime resume cursors keyed by session ID. The
// default engine wiring keeps cursors in process memory; a cross-instance
// deployment injects a distributed store (for example Redis) so a resumed
// stream can be served by a different pod than the one that stored the cursor.
type CursorStore interface {
	Load(ctx context.Context, sessionID string, now time.Time) (string, bool, error)
	Save(ctx context.Context, sessionID, cursor string, expiresAt time.Time) error
	Delete(ctx context.Context, sessionID string) error
}

// memoryCursorStore is the default in-process store. It preserves the
// historical map+TTL behavior of engine.realtimeCursors exactly.
type memoryCursorStore struct {
	mu      *sync.Mutex
	cursors map[string]realtimeCursorRecord
}

func newMemoryCursorStore(cursors map[string]realtimeCursorRecord, mu *sync.Mutex) *memoryCursorStore {
	if mu == nil {
		mu = &sync.Mutex{}
	}
	if cursors == nil {
		cursors = map[string]realtimeCursorRecord{}
	}
	return &memoryCursorStore{mu: mu, cursors: cursors}
}

func (m *memoryCursorStore) Load(_ context.Context, sessionID string, now time.Time) (string, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	record, ok := m.cursors[sessionID]
	if !ok {
		return "", false, nil
	}
	if !record.ExpiresAt.IsZero() && !record.ExpiresAt.After(now) {
		delete(m.cursors, sessionID)
		return "", false, nil
	}
	cursor := strings.TrimSpace(record.Cursor)
	return cursor, cursor != "", nil
}

func (m *memoryCursorStore) Save(_ context.Context, sessionID, cursor string, expiresAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cursors == nil {
		m.cursors = map[string]realtimeCursorRecord{}
	}
	m.cursors[sessionID] = realtimeCursorRecord{Cursor: cursor, ExpiresAt: expiresAt}
	return nil
}

func (m *memoryCursorStore) Delete(_ context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.cursors, sessionID)
	return nil
}

// redisCursorStore stores one cursor per session with a server-side TTL so a
// different instance can resume the same session. Every error is returned to
// the caller, which falls back to the in-process store and keeps Run/Stream
// alive (fail-open for resume cursors, never for admission decisions).
type redisCursorStore struct {
	client redis.UniversalClient
	prefix string
}

const defaultRedisCursorPrefix = "baymax:cursor"

func NewRedisCursorStore(client redis.UniversalClient, keyPrefix string) (*redisCursorStore, error) {
	if client == nil {
		return nil, errors.New("redis cursor store requires a client")
	}
	prefix := strings.TrimSpace(keyPrefix)
	if prefix == "" {
		prefix = defaultRedisCursorPrefix
	}
	return &redisCursorStore{client: client, prefix: prefix}, nil
}

func (r *redisCursorStore) key(sessionID string) string {
	return r.prefix + ":" + strings.TrimSpace(sessionID)
}

func (r *redisCursorStore) Load(ctx context.Context, sessionID string, _ time.Time) (string, bool, error) {
	if r == nil || r.client == nil {
		return "", false, errors.New("redis cursor store is unavailable")
	}
	cursor, err := r.client.Get(ctx, r.key(sessionID)).Result()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	cursor = strings.TrimSpace(cursor)
	return cursor, cursor != "", nil
}

func (r *redisCursorStore) Save(ctx context.Context, sessionID, cursor string, expiresAt time.Time) error {
	if r == nil || r.client == nil {
		return errors.New("redis cursor store is unavailable")
	}
	if strings.TrimSpace(sessionID) == "" {
		return errors.New("session id is required")
	}
	if expiresAt.IsZero() {
		return r.client.Set(ctx, r.key(sessionID), cursor, 0).Err()
	}
	return r.client.Set(ctx, r.key(sessionID), cursor, time.Until(expiresAt)).Err()
}

func (r *redisCursorStore) Delete(ctx context.Context, sessionID string) error {
	if r == nil || r.client == nil {
		return errors.New("redis cursor store is unavailable")
	}
	return r.client.Del(ctx, r.key(sessionID)).Err()
}

// Compile-time interface checks.
var (
	_ CursorStore = (*memoryCursorStore)(nil)
	_ CursorStore = (*redisCursorStore)(nil)
)
