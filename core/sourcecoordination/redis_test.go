package sourcecoordination

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestRedis(t *testing.T, namespace string) (*Redis, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	coordinator, err := NewRedis(client, RedisOptions{Options: Options{OwnerNamespace: namespace}, KeyPrefix: "test"})
	if err != nil {
		t.Fatal(err)
	}
	return coordinator, mr
}

func TestRedisCoordinatorRequiresNamespaceAndClient(t *testing.T) {
	if _, err := NewRedis(nil, RedisOptions{Options: Options{OwnerNamespace: "runner"}}); !errors.Is(err, types.ErrOwnerUnavailable) {
		t.Fatalf("nil client = %v", err)
	}
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	if _, err := NewRedis(client, RedisOptions{}); !errors.Is(err, types.ErrOwnerUnavailable) {
		t.Fatalf("empty namespace = %v", err)
	}
}

func TestRedisCoordinatorAcquireBlocksForeignWriterAndRefreshesOwnLease(t *testing.T) {
	coordinator, _ := newTestRedis(t, "runner")
	ctx := context.Background()

	first, err := coordinator.Acquire(ctx, "writer-a", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if first.Version != ContractVersion || first.OwnerNamespace != "runner" || first.WriterID != "writer-a" || first.Fence == "" || first.ExpiresAt.IsZero() {
		t.Fatalf("lease = %+v", first)
	}

	if _, err := coordinator.Acquire(ctx, "writer-b", time.Hour); !errors.Is(err, types.ErrOwnerUnavailable) {
		t.Fatalf("foreign acquire = %v", err)
	}

	second, err := coordinator.Acquire(ctx, "writer-a", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if second.Fence == first.Fence {
		t.Fatal("refresh must rotate the fence")
	}

	if _, err := coordinator.CompareAndSwap(ctx, first, 0, []byte(`{}`)); !errors.Is(err, types.ErrStaleWriter) {
		t.Fatalf("prior fence accepted = %v", err)
	}
}

func TestRedisCoordinatorCompareAndSwapRevisions(t *testing.T) {
	coordinator, _ := newTestRedis(t, "mailbox")
	ctx := context.Background()

	empty, err := coordinator.Read(ctx)
	if err != nil || empty.Revision != 0 || len(empty.Payload) != 0 {
		t.Fatalf("empty read = %+v %v", empty, err)
	}

	lease, err := coordinator.Acquire(ctx, "writer-a", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := coordinator.CompareAndSwap(ctx, lease, 0, []byte(`{"delivery":1}`)); err != nil {
		t.Fatal(err)
	}
	if _, err := coordinator.CompareAndSwap(ctx, lease, 0, []byte(`{"delivery":"stale"}`)); !errors.Is(err, types.ErrDeploymentConflict) {
		t.Fatalf("revision conflict = %v", err)
	}
	current, err := coordinator.Read(ctx)
	if err != nil || current.Revision != 1 || string(current.Payload) != `{"delivery":1}` {
		t.Fatalf("current = %+v %v", current, err)
	}
}

func TestRedisCoordinatorRejectsForeignAndExpiredLease(t *testing.T) {
	coordinator, mr := newTestRedis(t, "workflow")
	ctx := context.Background()

	lease, err := coordinator.Acquire(ctx, "writer", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	foreign := lease
	foreign.OwnerNamespace = "other"
	if _, err := coordinator.CompareAndSwap(ctx, foreign, 0, []byte(`{}`)); !errors.Is(err, types.ErrForeignWriter) {
		t.Fatalf("foreign CAS = %v", err)
	}
	if err := coordinator.ValidateLease(ctx, foreign); !errors.Is(err, types.ErrForeignWriter) {
		t.Fatalf("foreign validate = %v", err)
	}

	// Redis TTL is authoritative for expiry: once the lease key expires the
	// writer loses its fence and any further mutation fails closed.
	mr.FastForward(2 * time.Minute)
	if _, err := coordinator.CompareAndSwap(ctx, lease, 0, []byte(`{}`)); !errors.Is(err, types.ErrOwnerUnavailable) {
		t.Fatalf("expired CAS = %v", err)
	}
	if err := coordinator.ValidateLease(ctx, lease); !errors.Is(err, types.ErrOwnerUnavailable) {
		t.Fatalf("expired validate = %v", err)
	}
}

func TestRedisCoordinatorValidateLeaseDetectsTamperedFence(t *testing.T) {
	coordinator, _ := newTestRedis(t, "runner")
	ctx := context.Background()

	lease, err := coordinator.Acquire(ctx, "writer-a", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if err := coordinator.ValidateLease(ctx, lease); err != nil {
		t.Fatalf("valid lease = %v", err)
	}
	tampered := lease
	tampered.Fence = "fence:deadbeef"
	if err := coordinator.ValidateLease(ctx, tampered); !errors.Is(err, types.ErrStaleWriter) {
		t.Fatalf("tampered fence = %v", err)
	}
	tamperedWriter := lease
	tamperedWriter.WriterID = "writer-b"
	if err := coordinator.ValidateLease(ctx, tamperedWriter); !errors.Is(err, types.ErrStaleWriter) {
		t.Fatalf("tampered writer = %v", err)
	}
}

func TestRedisCoordinatorDeploymentScopeFailClosed(t *testing.T) {
	coordinator, mr := newTestRedis(t, "runner")

	scope := coordinator.DeploymentScope()
	if scope.Profile != types.DeploymentProfileCrossMachine {
		t.Fatalf("profile = %q", scope.Profile)
	}
	if scope.FencingScope != types.FencingScopeDistributed || scope.TransactionScope != types.TransactionScopeDistributed {
		t.Fatalf("scope = %+v", scope)
	}
	if scope.EvidenceReference == "" || !scope.Supported {
		t.Fatalf("scope = %+v", scope)
	}
	if err := scope.Validate(); err != nil {
		t.Fatalf("scope validate = %v", err)
	}

	mr.Close()
	degraded := coordinator.DeploymentScope()
	if degraded.Supported {
		t.Fatalf("unreachable redis must not advertise cross-machine: %+v", degraded)
	}
	if degraded.Profile == types.DeploymentProfileCrossMachine {
		t.Fatalf("unreachable redis degraded profile = %q", degraded.Profile)
	}

	unreachable := &Redis{}
	if scope := unreachable.DeploymentScope(); scope.Supported {
		t.Fatalf("nil coordinator scope = %+v", scope)
	}
}

func TestRedisCoordinatorKeyPrefixIsolation(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	alpha, err := NewRedis(client, RedisOptions{Options: Options{OwnerNamespace: "alpha"}, KeyPrefix: "team-a"})
	if err != nil {
		t.Fatal(err)
	}
	beta, err := NewRedis(client, RedisOptions{Options: Options{OwnerNamespace: "beta"}, KeyPrefix: "team-b"})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	lease, err := alpha.Acquire(ctx, "writer-a", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := beta.Acquire(ctx, "writer-a", time.Hour); err != nil {
		t.Fatalf("namespaced lease leaked = %v", err)
	}
	if _, err := beta.CompareAndSwap(ctx, lease, 0, []byte(`{}`)); !errors.Is(err, types.ErrForeignWriter) {
		t.Fatalf("cross-namespace CAS = %v", err)
	}
}

func TestRedisCoordinatorRealRedisIntegration(t *testing.T) {
	addr := os.Getenv("BAYMAX_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("skipped-unconfigured: BAYMAX_TEST_REDIS_ADDR is not set")
	}
	client := redis.NewClient(&redis.Options{Addr: addr})
	t.Cleanup(func() { _ = client.Close() })
	coordinator, err := NewRedis(client, RedisOptions{Options: Options{OwnerNamespace: "integration"}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	lease, err := coordinator.Acquire(ctx, "integration-writer", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := coordinator.CompareAndSwap(ctx, lease, 0, []byte(`{"source":"real-redis"}`)); err != nil {
		t.Fatal(err)
	}
}
