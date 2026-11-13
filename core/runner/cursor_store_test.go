package runner

import (
	"context"
	"errors"
	"testing"
	"time"

	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestMemoryCursorStoreTTLExpiry(t *testing.T) {
	store := newMemoryCursorStore(nil, nil)
	ctx := context.Background()
	now := time.Date(2026, 11, 13, 11, 0, 0, 0, time.UTC)

	if err := store.Save(ctx, "session-1", "cursor:1", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	cursor, ok, err := store.Load(ctx, "session-1", now)
	if err != nil || !ok || cursor != "cursor:1" {
		t.Fatalf("load = %q %v %v", cursor, ok, err)
	}
	cursor, ok, err = store.Load(ctx, "session-1", now.Add(2*time.Minute))
	if err != nil || ok {
		t.Fatalf("expired load = %q %v %v", cursor, ok, err)
	}
	if err := store.Delete(ctx, "session-1"); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := store.Load(ctx, "session-1", now.Add(2*time.Minute)); ok {
		t.Fatal("deleted cursor still loadable")
	}
}

func TestRedisCursorStoreRoundTrip(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	store, err := NewRedisCursorStore(client, "")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)

	// The Redis store derives TTL from the wall clock, so the expiry passed to
	// Save must be wall-clock based even though Load ignores its now argument.
	if err := store.Save(ctx, "session-r", "cursor:r", time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	cursor, ok, err := store.Load(ctx, "session-r", now)
	if err != nil || !ok || cursor != "cursor:r" {
		t.Fatalf("load = %q %v %v", cursor, ok, err)
	}

	// Server-side TTL is authoritative.
	mr.FastForward(2 * time.Minute)
	if _, ok, err := store.Load(ctx, "session-r", now); err != nil || ok {
		t.Fatalf("expired load = ok=%v err=%v", ok, err)
	}

	if err := store.Save(ctx, "session-r", "cursor:r2", time.Time{}); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(ctx, "session-r"); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := store.Load(ctx, "session-r", now); err != nil || ok {
		t.Fatalf("deleted load = ok=%v err=%v", ok, err)
	}

	if _, err := NewRedisCursorStore(nil, ""); err == nil {
		t.Fatal("nil client accepted")
	}
}

type failingCursorStore struct{}

func (failingCursorStore) Load(context.Context, string, time.Time) (string, bool, error) {
	return "", false, errors.New("redis down")
}
func (failingCursorStore) Save(context.Context, string, string, time.Time) error {
	return errors.New("redis down")
}
func (failingCursorStore) Delete(context.Context, string) error {
	return errors.New("redis down")
}

func engineCursorTestRuntime(e *Engine, sessionID string) *realtimeSessionRuntime {
	cfg := runtimeconfig.DefaultConfig().Runtime.Realtime
	cfg.InterruptResume.Enabled = true
	cfg.InterruptResume.ResumeCursorTTLMS = 60000
	return &realtimeSessionRuntime{
		engine:    e,
		sessionID: sessionID,
		cfg:       cfg,
	}
}

func TestEngineCursorStoreFallbackToMemory(t *testing.T) {
	engine := New(nil, WithCursorStore(failingCursorStore{}))
	if engine.cursorStore == nil {
		t.Fatal("cursor store injection lost")
	}
	runtime := engineCursorTestRuntime(engine, "session-fallback")
	engine.now = func() time.Time { return time.Date(2026, 11, 13, 13, 0, 0, 0, time.UTC) }
	runtime.storeCursor("cursor:fallback")
	if got := len(engine.realtimeCursors); got != 1 {
		t.Fatalf("fallback memory store has %d cursors, want 1", got)
	}
	cursor, ok := runtime.loadCursor()
	if !ok || cursor != "cursor:fallback" {
		t.Fatalf("fallback load = %q %v", cursor, ok)
	}
}

func TestEngineCursorStoreUsesInjectedStore(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	redisStore, err := NewRedisCursorStore(client, "test:cursor")
	if err != nil {
		t.Fatal(err)
	}
	engine := New(nil, WithCursorStore(redisStore))
	runtime := engineCursorTestRuntime(engine, "session-redis")
	engine.now = func() time.Time { return time.Date(2026, 11, 13, 14, 0, 0, 0, time.UTC) }
	runtime.storeCursor("cursor:redis")
	if got := len(engine.realtimeCursors); got != 0 {
		t.Fatalf("injected store must not touch the in-process map: %d", got)
	}
	cursor, ok := runtime.loadCursor()
	if !ok || cursor != "cursor:redis" {
		t.Fatalf("injected load = %q %v", cursor, ok)
	}
}
