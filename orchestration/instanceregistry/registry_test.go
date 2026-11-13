package instanceregistry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// runRegistryContract exercises the shared registry semantics on both backends.
func runRegistryContract(t *testing.T, make func(t *testing.T) Registry) {
	t.Helper()
	ctx := context.Background()

	registry := make(t)

	if err := registry.Register(ctx, "inst-a", "10.0.0.1:8080", time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(ctx, "inst-b", "10.0.0.2:8080", time.Minute); err != nil {
		t.Fatal(err)
	}
	instances, err := registry.List(ctx)
	if err != nil || len(instances) != 2 {
		t.Fatalf("list = %+v err=%v", instances, err)
	}

	// Binding to an unregistered instance is rejected.
	if err := registry.BindSession(ctx, "session-1", "inst-z", time.Minute); !errors.Is(err, ErrUnknownInstance) {
		t.Fatalf("bind unknown = %v", err)
	}
	if err := registry.BindSession(ctx, "session-1", "inst-a", time.Minute); err != nil {
		t.Fatal(err)
	}
	owner, err := registry.ResolveSession(ctx, "session-1")
	if err != nil || owner != "inst-a" {
		t.Fatalf("resolve = %q err=%v", owner, err)
	}

	// Unbound sessions are reported distinctly from lost affinity.
	if _, err := registry.ResolveSession(ctx, "session-2"); !errors.Is(err, ErrSessionUnbound) {
		t.Fatalf("unbound resolve = %v", err)
	}

	// An explicit unregister drops the session bindings gracefully (the host
	// chose to leave), so the session reports unbound rather than lost. Lost
	// affinity is the crash path — the TTL expiry test below covers it.
	if err := registry.Unregister(ctx, "inst-a"); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.ResolveSession(ctx, "session-1"); !errors.Is(err, ErrSessionUnbound) {
		t.Fatalf("resolve after unregister = %v", err)
	}

	if err := registry.Register(ctx, "", "x", time.Minute); err == nil {
		t.Fatal("empty instance id accepted")
	}
	if err := registry.Register(ctx, "inst-c", "x", 0); err == nil {
		t.Fatal("zero ttl accepted")
	}
}

func TestRegistryContractMemory(t *testing.T) {
	runRegistryContract(t, func(t *testing.T) Registry {
		return NewMemoryRegistry()
	})
}

func TestRegistryContractRedis(t *testing.T) {
	runRegistryContract(t, func(t *testing.T) Registry {
		mr := miniredis.RunT(t)
		client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		t.Cleanup(func() { _ = client.Close() })
		registry, err := NewRedisRegistry(RedisRegistryConfig{Client: client})
		if err != nil {
			t.Fatal(err)
		}
		return registry
	})
}

func TestRedisRegistryTTLExpiryRemovesInstance(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	registry, err := NewRedisRegistry(RedisRegistryConfig{Client: client, KeyPrefix: "test:registry"})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	if err := registry.Register(ctx, "inst-a", "ep", 30*time.Second); err != nil {
		t.Fatal(err)
	}
	if err := registry.BindSession(ctx, "session-a", "inst-a", time.Minute); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.ResolveSession(ctx, "session-a"); err != nil {
		t.Fatalf("resolve before expiry = %v", err)
	}

	mr.FastForward(45 * time.Second)

	instances, err := registry.List(ctx)
	if err != nil || len(instances) != 0 {
		t.Fatalf("list after expiry = %+v err=%v", instances, err)
	}
	if _, err := registry.ResolveSession(ctx, "session-a"); !errors.Is(err, ErrAffinityLost) {
		t.Fatalf("resolve after instance expiry = %v", err)
	}
}

func TestRedisRegistryHeartbeatRefreshesTTL(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	registry, err := NewRedisRegistry(RedisRegistryConfig{Client: client})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	if err := registry.Register(ctx, "inst-a", "ep-old", 10*time.Second); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(ctx, "inst-a", "ep-new", 10*time.Second); err != nil {
		t.Fatal(err)
	}
	instances, err := registry.List(ctx)
	if err != nil || len(instances) != 1 {
		t.Fatalf("list = %+v err=%v", instances, err)
	}
	if instances[0].Endpoint != "ep-new" {
		t.Fatalf("heartbeat did not update endpoint: %+v", instances[0])
	}
	if instances[0].StartedAt.IsZero() {
		t.Fatal("started_at lost on heartbeat")
	}
}

func TestRedisRegistryUnavailableFailsClosed(t *testing.T) {
	if _, err := NewRedisRegistry(RedisRegistryConfig{}); !errors.Is(err, ErrAffinityUnavailable) {
		t.Fatalf("nil client init = %v", err)
	}
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", DialTimeout: 200 * time.Millisecond, MaxRetries: -1})
	t.Cleanup(func() { _ = client.Close() })
	registry, err := NewRedisRegistry(RedisRegistryConfig{Client: client})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := registry.ResolveSession(ctx, "session-x"); !errors.Is(err, ErrAffinityUnavailable) {
		t.Fatalf("unreachable resolve = %v", err)
	}
}
