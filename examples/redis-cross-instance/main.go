// Command redis-cross-instance demonstrates the Redis-backed coordination
// stack introduced by the cross-instance proposal:
//
//   - sourcecoordination.Redis: lease contention, fence rotation and CAS
//     revision conflicts across two writers sharing one Redis keyspace
//   - mailbox.RedisStore: publish on one instance, consume and ack on another
//   - instanceregistry.RedisRegistry: session affinity binding and the lost-
//     affinity signal when the owning instance disappears
//
// It runs fully offline on an in-process Redis substitute (miniredis). Against
// a real Redis, swap the client address via BAYMAX_EXAMPLE_REDIS_ADDR and the
// same sequence executes unchanged.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/FelixSeptem/baymax/core/sourcecoordination"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/orchestration/instanceregistry"
	"github.com/FelixSeptem/baymax/orchestration/mailbox"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func main() {
	fmt.Println("== redis cross-instance coordination ==")

	var client *redis.Client
	if addr := strings.TrimSpace(os.Getenv("BAYMAX_EXAMPLE_REDIS_ADDR")); addr != "" {
		client = redis.NewClient(&redis.Options{Addr: addr})
		fmt.Printf("REDIS addr=%s mode=real\n", addr)
	} else {
		server, err := miniredis.Run()
		if err != nil {
			fail("miniredis: " + err.Error())
		}
		defer server.Close()
		client = redis.NewClient(&redis.Options{Addr: server.Addr()})
		fmt.Printf("REDIS addr=%s mode=miniredis-offline\n", server.Addr())
	}
	defer func() { _ = client.Close() }()

	ctx := context.Background()

	demonstrateCoordinator(ctx, client)
	demonstrateMailbox(ctx, client)
	demonstrateAffinity(ctx, client)

	fmt.Println("redis cross-instance demo complete")
}

// demonstrateCoordinator shows two writers contending for one source-owner
// lease, fence rotation on refresh, and the CAS revision conflict contract.
func demonstrateCoordinator(ctx context.Context, client *redis.Client) {
	writerA, err := sourcecoordination.NewRedis(client, sourcecoordination.RedisOptions{
		Options:   sourcecoordination.Options{OwnerNamespace: "demo:history"},
		KeyPrefix: "demo",
	})
	if err != nil {
		fail("coordinator A: " + err.Error())
	}
	writerB, err := sourcecoordination.NewRedis(client, sourcecoordination.RedisOptions{
		Options:   sourcecoordination.Options{OwnerNamespace: "demo:history"},
		KeyPrefix: "demo",
	})
	if err != nil {
		fail("coordinator B: " + err.Error())
	}

	scope := writerA.DeploymentScope()
	fmt.Printf(
		"COORD_SCOPE profile=%s fencing=%s transaction=%s evidence=%s\n",
		scope.Profile, scope.FencingScope, scope.TransactionScope, scope.EvidenceReference,
	)
	if scope.Profile != types.DeploymentProfileCrossMachine {
		fail("redis coordinator must advertise cross_machine")
	}
	fmt.Println("MARKER coordinator-cross-machine-scope")

	leaseA, err := writerA.Acquire(ctx, "writer-a", time.Minute)
	if err != nil {
		fail("acquire A: " + err.Error())
	}
	if _, err := writerB.Acquire(ctx, "writer-b", time.Minute); !errors.Is(err, types.ErrOwnerUnavailable) {
		fail("writer-b should be blocked while writer-a holds the lease")
	}
	fmt.Println("MARKER coordinator-lease-contended")

	committed, err := writerA.CompareAndSwap(ctx, leaseA, 0, []byte(`{"revision":"first"}`))
	if err != nil {
		fail("cas A: " + err.Error())
	}
	if _, err := writerA.CompareAndSwap(ctx, leaseA, 0, []byte(`{"revision":"stale"}`)); !errors.Is(err, types.ErrDeploymentConflict) {
		fail("stale revision should conflict")
	}
	fmt.Printf("MARKER cas-revision-conflict committed_revision=%d\n", committed.Revision)
}

// demonstrateMailbox shows cross-instance delivery: instance A publishes, and
// instance B consumes and acks through a separate store bound to the same
// Redis namespace.
func demonstrateMailbox(ctx context.Context, client *redis.Client) {
	policy := mailbox.Policy{MaxAttempts: 3, DLQEnabled: true, BackoffInitial: time.Millisecond, BackoffMax: time.Millisecond, JitterRatio: 0}
	instanceA, err := mailbox.NewRedisStore(mailbox.RedisStoreConfig{Client: client, Namespace: "demo:tasks"}, policy)
	if err != nil {
		fail("mailbox A: " + err.Error())
	}
	instanceB, err := mailbox.NewRedisStore(mailbox.RedisStoreConfig{Client: client, Namespace: "demo:tasks"}, policy)
	if err != nil {
		fail("mailbox B: " + err.Error())
	}

	now := time.Now().UTC()
	published, err := instanceA.Publish(ctx, mailbox.Envelope{
		MessageID:      "msg-demo",
		IdempotencyKey: "idem-demo",
		Kind:           mailbox.KindCommand,
		Payload:        map[string]any{"origin": "instance-a"},
	}, now)
	if err != nil {
		fail("publish: " + err.Error())
	}
	fmt.Printf("MAILBOX_PUBLISH instance=A message=%s state=%s\n", published.Record.Envelope.MessageID, published.Record.State)

	claimed, ok, err := instanceB.Consume(ctx, "instance-b", now.Add(time.Millisecond), time.Minute, false)
	if err != nil || !ok {
		fail(fmt.Sprintf("cross-instance consume ok=%v err=%v", ok, err))
	}
	fmt.Printf(
		"MARKER mailbox-cross-instance-delivery instance=B attempt=%d consumer=%s\n",
		claimed.DeliveryAttempt, claimed.ConsumerID,
	)

	if _, err := instanceB.Ack(ctx, "msg-demo", "instance-b", now.Add(2*time.Millisecond)); err != nil {
		fail("ack: " + err.Error())
	}
	stats, err := instanceA.Stats(ctx)
	if err != nil || stats.AckTotal != 1 {
		fail(fmt.Sprintf("instance A must observe B's ack: %+v err=%v", stats, err))
	}
	fmt.Println("MARKER mailbox-cross-instance-ack-visible")
}

// demonstrateAffinity shows session affinity binding and the lost-affinity
// signal when the owning instance disappears from the registry.
func demonstrateAffinity(ctx context.Context, client *redis.Client) {
	registry, err := instanceregistry.NewRedisRegistry(instanceregistry.RedisRegistryConfig{Client: client, KeyPrefix: "demo:registry"})
	if err != nil {
		fail("registry: " + err.Error())
	}
	if err := registry.Register(ctx, "pod-1", "10.1.0.5:9090", time.Minute); err != nil {
		fail("register: " + err.Error())
	}
	if err := registry.BindSession(ctx, "session-demo", "pod-1", time.Minute); err != nil {
		fail("bind: " + err.Error())
	}
	owner, err := registry.ResolveSession(ctx, "session-demo")
	if err != nil || owner != "pod-1" {
		fail(fmt.Sprintf("resolve owner=%q err=%v", owner, err))
	}
	fmt.Printf("MARKER affinity-bound session=session-demo owner=%s\n", owner)

	if err := registry.Unregister(ctx, "pod-1"); err != nil {
		fail("unregister: " + err.Error())
	}
	if _, err := registry.ResolveSession(ctx, "session-demo"); !errors.Is(err, instanceregistry.ErrSessionUnbound) {
		fail(fmt.Sprintf("resolve after graceful unregister = %v", err))
	}
	fmt.Println("MARKER affinity-released-on-graceful-leave")
}

func fail(message string) {
	fmt.Printf("FAIL %s\n", message)
	os.Exit(1)
}
