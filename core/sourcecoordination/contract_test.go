package sourcecoordination

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// contractOptions scopes the per-backend guarantees. blockingAcquire is an
// optional extra guarantee: Memory and Redis refuse a foreign Acquire while a
// live lease exists, while File follows sequential-reopen takeover (the new
// writer wins Acquire; the previous fence is rejected at CAS). CAS-level
// fencing, revision conflicts and foreign/stale rejection are universal.
type contractOptions struct {
	blockingAcquire bool
}

// runCoordinatorContract runs the backend-agnostic fencing/CAS semantics every
// Coordinator implementation must satisfy.
func runCoordinatorContract(t *testing.T, make func(t *testing.T) Coordinator, opts contractOptions) {
	t.Helper()
	ctx := context.Background()

	coordinator := make(t)

	empty, err := coordinator.Read(ctx)
	if err != nil {
		t.Fatalf("read empty: %v", err)
	}
	if empty.Revision != 0 || len(empty.Payload) != 0 {
		t.Fatalf("empty snapshot = %+v", empty)
	}

	first, err := coordinator.Acquire(ctx, "writer-a", time.Hour)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	if first.Version != ContractVersion || first.WriterID != "writer-a" || first.Fence == "" || first.ExpiresAt.IsZero() {
		t.Fatalf("lease = %+v", first)
	}

	if opts.blockingAcquire {
		if _, err := coordinator.Acquire(ctx, "writer-b", time.Hour); !errors.Is(err, types.ErrOwnerUnavailable) {
			t.Fatalf("blocked acquire = %v", err)
		}
	}

	committed, err := coordinator.CompareAndSwap(ctx, first, 0, []byte(`{"state":"one"}`))
	if err != nil {
		t.Fatalf("cas: %v", err)
	}
	if committed.Revision != 1 || string(committed.Payload) != `{"state":"one"}` {
		t.Fatalf("committed = %+v", committed)
	}

	if _, err := coordinator.CompareAndSwap(ctx, first, 0, []byte(`{"state":"stale-rev"}`)); !errors.Is(err, types.ErrDeploymentConflict) {
		t.Fatalf("revision conflict = %v", err)
	}

	if validator, ok := coordinator.(LeaseValidator); ok {
		if err := validator.ValidateLease(ctx, first); err != nil {
			t.Fatalf("validator interface: %v", err)
		}
	}

	foreign := first
	foreign.OwnerNamespace = "foreign-owner"
	if _, err := coordinator.CompareAndSwap(ctx, foreign, 1, []byte(`{}`)); !errors.Is(err, types.ErrForeignWriter) {
		t.Fatalf("foreign CAS = %v", err)
	}
	if validator, ok := coordinator.(LeaseValidator); ok {
		if err := validator.ValidateLease(ctx, foreign); !errors.Is(err, types.ErrForeignWriter) {
			t.Fatalf("foreign validate = %v", err)
		}
	}

	tampered := first
	tampered.Fence = "fence:tampered"
	if _, err := coordinator.CompareAndSwap(ctx, tampered, 1, []byte(`{}`)); !errors.Is(err, types.ErrStaleWriter) {
		t.Fatalf("stale CAS = %v", err)
	}

	current, err := coordinator.Read(ctx)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if current.Revision != 1 || string(current.Payload) != `{"state":"one"}` {
		t.Fatalf("current = %+v", current)
	}
}

func TestCoordinatorContractMemory(t *testing.T) {
	runCoordinatorContract(t, func(t *testing.T) Coordinator {
		c, err := NewMemory(Options{OwnerNamespace: "contract"})
		if err != nil {
			t.Fatal(err)
		}
		return c
	}, contractOptions{blockingAcquire: true})
}

func TestCoordinatorContractFile(t *testing.T) {
	runCoordinatorContract(t, func(t *testing.T) Coordinator {
		c, err := OpenFile(t.TempDir(), Options{OwnerNamespace: "contract"})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = c.Close() })
		return c
	}, contractOptions{})
}

func TestCoordinatorContractRedis(t *testing.T) {
	runCoordinatorContract(t, func(t *testing.T) Coordinator {
		mr := miniredis.RunT(t)
		client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		t.Cleanup(func() { _ = client.Close() })
		c, err := NewRedis(client, RedisOptions{Options: Options{OwnerNamespace: "contract"}})
		if err != nil {
			t.Fatal(err)
		}
		return c
	}, contractOptions{blockingAcquire: true})
}
