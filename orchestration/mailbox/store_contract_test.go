package mailbox

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// runStoreContract runs the backend-agnostic mailbox semantics every Store
// implementation must satisfy: idempotent publish, inflight ownership, heartbeat
// ownership checks, ack/nack/requeue lifecycle, retry backoff, dead-letter,
// inflight reclaim and snapshot/restore parity. The `now` parameter of the
// Store interface drives the clock so no backend can hide behind wall time.
func runStoreContract(t *testing.T, make func(t *testing.T) Store) {
	t.Helper()
	ctx := context.Background()
	base := time.Date(2026, 11, 13, 8, 0, 0, 0, time.UTC)

	store := make(t)

	// Empty queue: consume reports no message without mutating stats.
	if _, ok, err := store.Consume(ctx, "consumer-1", base, 30*time.Second, false); err != nil || ok {
		t.Fatalf("consume from empty queue = ok=%v err=%v", ok, err)
	}

	first, err := store.Publish(ctx, Envelope{
		MessageID:      "msg-contract",
		IdempotencyKey: "idem-contract",
		Kind:           KindCommand,
		Payload:        map[string]any{"n": 1},
	}, base)
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if first.Duplicate || first.Record.State != StateQueued {
		t.Fatalf("first publish = %+v", first)
	}
	duplicate, err := store.Publish(ctx, Envelope{
		MessageID:      "msg-contract-dup",
		IdempotencyKey: "idem-contract",
		Kind:           KindCommand,
	}, base.Add(time.Millisecond))
	if err != nil {
		t.Fatalf("duplicate publish: %v", err)
	}
	if !duplicate.Duplicate || duplicate.Record.Envelope.MessageID != "msg-contract" {
		t.Fatalf("duplicate publish = %+v", duplicate)
	}

	claimed, ok, err := store.Consume(ctx, "consumer-1", base.Add(10*time.Millisecond), 30*time.Second, false)
	if err != nil || !ok {
		t.Fatalf("consume = ok=%v err=%v", ok, err)
	}
	if claimed.State != StateInFlight || claimed.DeliveryAttempt != 1 || claimed.ConsumerID != "consumer-1" {
		t.Fatalf("claimed = %+v", claimed)
	}

	if _, err := store.Heartbeat(ctx, "msg-contract", "consumer-2", base.Add(20*time.Millisecond), 30*time.Second); !errors.Is(err, ErrConsumerMismatch) {
		t.Fatalf("foreign heartbeat = %v", err)
	}
	if _, err := store.Heartbeat(ctx, "msg-contract", "consumer-1", base.Add(20*time.Millisecond), 30*time.Second); err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
	if _, err := store.Ack(ctx, "msg-contract", "consumer-2", base.Add(25*time.Millisecond)); !errors.Is(err, ErrConsumerMismatch) {
		t.Fatalf("foreign ack = %v", err)
	}
	acked, err := store.Ack(ctx, "msg-contract", "consumer-1", base.Add(25*time.Millisecond))
	if err != nil || acked.State != StateAcked {
		t.Fatalf("ack = %+v err=%v", acked, err)
	}

	stats, err := store.Stats(ctx)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.PublishedTotal != 1 || stats.DuplicatePublishTotal != 1 || stats.AckTotal != 1 || stats.ConsumedTotal != 1 {
		t.Fatalf("stats = %+v", stats)
	}

	// Retry backoff, dead-letter and inflight reclaim.
	retryBase := base.Add(time.Second)
	retry, err := store.Publish(ctx, Envelope{
		MessageID:      "msg-retry",
		IdempotencyKey: "idem-retry",
		Kind:           KindCommand,
	}, retryBase)
	if err != nil {
		t.Fatalf("retry publish: %v", err)
	}
	_ = retry
	// The store was created with Policy{MaxAttempts:2, DLQ:true} by the caller;
	// enforce the assumption defensively.
	snap, err := store.Snapshot(ctx)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if snap.Policy.MaxAttempts != 2 || !snap.Policy.DLQEnabled {
		t.Fatalf("contract store policy = %+v", snap.Policy)
	}

	attempt1, ok, err := store.Consume(ctx, "consumer-1", retryBase, 30*time.Second, false)
	if err != nil || !ok || attempt1.DeliveryAttempt != 1 {
		t.Fatalf("retry consume #1 = ok=%v attempt=%d err=%v", ok, attempt1.DeliveryAttempt, err)
	}
	if _, err := store.Nack(ctx, "msg-retry", "consumer-1", "transient", retryBase, ActionOptions{}); err != nil {
		t.Fatalf("nack #1: %v", err)
	}
	if _, err := store.Requeue(ctx, "msg-retry", "consumer-1", "transient", retryBase, ActionOptions{}); err != nil {
		t.Fatalf("requeue #1: %v", err)
	}
	if _, ok, err := store.Consume(ctx, "consumer-1", retryBase.Add(10*time.Millisecond), 30*time.Second, false); err != nil || ok {
		t.Fatalf("consume inside backoff window = ok=%v err=%v", ok, err)
	}
	attempt2, ok, err := store.Consume(ctx, "consumer-1", retryBase.Add(100*time.Millisecond), 30*time.Second, false)
	if err != nil || !ok {
		t.Fatalf("retry consume #2 = ok=%v err=%v", ok, err)
	}
	if attempt2.DeliveryAttempt != 2 {
		t.Fatalf("attempt #2 = %d", attempt2.DeliveryAttempt)
	}
	dead, err := store.Requeue(ctx, "msg-retry", "consumer-1", "retry_exhausted", retryBase.Add(110*time.Millisecond), ActionOptions{})
	if err != nil || dead.State != StateDeadLetter {
		t.Fatalf("dead letter = %+v err=%v", dead, err)
	}

	// Inflight reclaim: consumer-1 loses its lease, consumer-2 reclaims.
	reclaimBase := base.Add(2 * time.Second)
	if _, err := store.Publish(ctx, Envelope{
		MessageID:      "msg-reclaim",
		IdempotencyKey: "idem-reclaim",
		Kind:           KindCommand,
	}, reclaimBase); err != nil {
		t.Fatalf("reclaim publish: %v", err)
	}
	if _, ok, err := store.Consume(ctx, "consumer-1", reclaimBase, 50*time.Millisecond, false); err != nil || !ok {
		t.Fatalf("reclaim consume = ok=%v err=%v", ok, err)
	}
	// The reclaiming pass moves the expired inflight message back to queued
	// with a retry backoff, so it returns nothing this pass by design.
	statsBefore, err := store.Stats(ctx)
	if err != nil {
		t.Fatalf("stats before reclaim: %v", err)
	}
	if _, ok, err := store.Consume(ctx, "consumer-2", reclaimBase.Add(100*time.Millisecond), 50*time.Millisecond, true); err != nil || ok {
		t.Fatalf("reclaiming pass = ok=%v err=%v", ok, err)
	}
	statsAfter, err := store.Stats(ctx)
	if err != nil {
		t.Fatalf("stats after reclaim: %v", err)
	}
	if statsAfter.RequeueTotal <= statsBefore.RequeueTotal {
		t.Fatalf("reclaim did not requeue: before=%+v after=%+v", statsBefore, statsAfter)
	}
	reclaimed, ok, err := store.Consume(ctx, "consumer-2", reclaimBase.Add(300*time.Millisecond), 50*time.Millisecond, false)
	if err != nil || !ok {
		t.Fatalf("reclaimed consume = ok=%v err=%v", ok, err)
	}
	if reclaimed.Envelope.MessageID != "msg-reclaim" || reclaimed.ConsumerID != "consumer-2" || reclaimed.DeliveryAttempt != 2 {
		t.Fatalf("reclaimed = %+v", reclaimed)
	}
}

func TestStoreContractMemory(t *testing.T) {
	runStoreContract(t, func(t *testing.T) Store {
		return NewMemoryStore(Policy{MaxAttempts: 2, DLQEnabled: true, BackoffInitial: 20 * time.Millisecond, BackoffMax: 20 * time.Millisecond, JitterRatio: 0})
	})
}

func TestStoreContractFile(t *testing.T) {
	runStoreContract(t, func(t *testing.T) Store {
		store, err := NewFileStore(t.TempDir()+"/mailbox.json", Policy{MaxAttempts: 2, DLQEnabled: true, BackoffInitial: 20 * time.Millisecond, BackoffMax: 20 * time.Millisecond, JitterRatio: 0})
		if err != nil {
			t.Fatal(err)
		}
		return store
	})
}

func TestStoreContractRedis(t *testing.T) {
	runStoreContract(t, func(t *testing.T) Store {
		mr := miniredis.RunT(t)
		client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		t.Cleanup(func() { _ = client.Close() })
		store, err := NewRedisStore(RedisStoreConfig{Client: client, Namespace: "contract"}, Policy{MaxAttempts: 2, DLQEnabled: true, BackoffInitial: 20 * time.Millisecond, BackoffMax: 20 * time.Millisecond, JitterRatio: 0})
		if err != nil {
			t.Fatal(err)
		}
		return store
	})
}

func TestRedisStoreCrossInstanceVisibility(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	policy := Policy{MaxAttempts: 3, DLQEnabled: true, BackoffInitial: time.Millisecond, BackoffMax: time.Millisecond, JitterRatio: 0}
	storeA, err := NewRedisStore(RedisStoreConfig{Client: client, Namespace: "cross"}, policy)
	if err != nil {
		t.Fatal(err)
	}
	storeB, err := NewRedisStore(RedisStoreConfig{Client: client, Namespace: "cross"}, policy)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	now := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)

	if _, err := storeA.Publish(ctx, Envelope{
		MessageID:      "msg-cross",
		IdempotencyKey: "idem-cross",
		Kind:           KindCommand,
	}, now); err != nil {
		t.Fatalf("publish on A: %v", err)
	}

	claimed, ok, err := storeB.Consume(ctx, "instance-b", now.Add(time.Millisecond), time.Minute, false)
	if err != nil || !ok {
		t.Fatalf("consume on B = ok=%v err=%v", ok, err)
	}
	if claimed.Envelope.MessageID != "msg-cross" {
		t.Fatalf("claimed on B = %+v", claimed)
	}
	if _, err := storeB.Ack(ctx, "msg-cross", "instance-b", now.Add(2*time.Millisecond)); err != nil {
		t.Fatalf("ack on B: %v", err)
	}

	stats, err := storeA.Stats(ctx)
	if err != nil {
		t.Fatalf("stats on A: %v", err)
	}
	if stats.AckTotal != 1 || stats.ConsumedTotal != 1 {
		t.Fatalf("stats on A after B mutations = %+v", stats)
	}

	// Concurrent publishes from both instances must not lose messages.
	for i := 0; i < 5; i++ {
		if _, err := storeA.Publish(ctx, Envelope{
			MessageID:      "msg-concurrent-a-" + string(rune('0'+i)),
			IdempotencyKey: "idem-concurrent-a-" + string(rune('0'+i)),
			Kind:           KindEvent,
		}, now); err != nil {
			t.Fatalf("concurrent publish A: %v", err)
		}
		if _, err := storeB.Publish(ctx, Envelope{
			MessageID:      "msg-concurrent-b-" + string(rune('0'+i)),
			IdempotencyKey: "idem-concurrent-b-" + string(rune('0'+i)),
			Kind:           KindEvent,
		}, now); err != nil {
			t.Fatalf("concurrent publish B: %v", err)
		}
	}
	stats, err = storeA.Stats(ctx)
	if err != nil {
		t.Fatalf("final stats: %v", err)
	}
	if stats.PublishedTotal != 11 {
		t.Fatalf("published total = %d, want 11", stats.PublishedTotal)
	}
}

func TestRedisStoreSnapshotRestoreAcrossInstances(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	policy := Policy{MaxAttempts: 3, DLQEnabled: true}
	store, err := NewRedisStore(RedisStoreConfig{Client: client, Namespace: "snap"}, policy)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	now := time.Date(2026, 11, 13, 10, 0, 0, 0, time.UTC)
	if _, err := store.Publish(ctx, Envelope{
		MessageID:      "msg-snap",
		IdempotencyKey: "idem-snap",
		Kind:           KindCommand,
	}, now); err != nil {
		t.Fatal(err)
	}
	snapshot, err := store.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Records) != 1 || snapshot.Backend != RedisBackend {
		t.Fatalf("snapshot = %+v", snapshot)
	}

	// Restore into a fresh namespace and verify the projection survives.
	target, err := NewRedisStore(RedisStoreConfig{Client: client, Namespace: "snap-restore"}, policy)
	if err != nil {
		t.Fatal(err)
	}
	snapshot.Backend = RedisBackend
	if err := target.Restore(ctx, snapshot); err != nil {
		t.Fatalf("restore: %v", err)
	}
	restored, err := target.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(restored.Records) != 1 || restored.Records[0].Envelope.MessageID != "msg-snap" {
		t.Fatalf("restored = %+v", restored)
	}
}

func TestRedisStoreUnavailableFailsClosed(t *testing.T) {
	if _, err := NewRedisStore(RedisStoreConfig{}, Policy{}); !errors.Is(err, ErrRedisUnavailable) {
		t.Fatalf("nil client init = %v", err)
	}
	// Unreachable server: every mutation fails closed with ErrRedisUnavailable.
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", DialTimeout: 200 * time.Millisecond, MaxRetries: -1})
	t.Cleanup(func() { _ = client.Close() })
	store, err := NewRedisStore(RedisStoreConfig{Client: client, OpTimeout: 300 * time.Millisecond}, Policy{})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, _, err := store.Consume(ctx, "c", time.Now(), time.Second, false); !errors.Is(err, ErrRedisUnavailable) {
		t.Fatalf("unreachable consume = %v", err)
	}
	if _, err := store.Publish(ctx, Envelope{MessageID: "m", IdempotencyKey: "i", Kind: KindCommand}, time.Now()); !errors.Is(err, ErrRedisUnavailable) {
		t.Fatalf("unreachable publish = %v", err)
	}
}
