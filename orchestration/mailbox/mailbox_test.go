package mailbox

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestMailboxEnvelopeValidationFailFast(t *testing.T) {
	mb := newTestMailbox(t, Policy{})
	ctx := context.Background()
	if _, err := mb.Publish(ctx, Envelope{
		IdempotencyKey: "idem-1",
		Kind:           KindCommand,
	}); err == nil {
		t.Fatal("expected validation error for missing message_id")
	}
	if _, err := mb.Publish(ctx, Envelope{
		MessageID: "msg-1",
		Kind:      KindCommand,
	}); err == nil {
		t.Fatal("expected validation error for missing idempotency_key")
	}
	if _, err := mb.Publish(ctx, Envelope{
		MessageID:      "msg-1",
		IdempotencyKey: "idem-1",
		Kind:           "unknown",
	}); err == nil {
		t.Fatal("expected validation error for invalid kind")
	}
}

func TestMailboxDuplicatePublishConvergesByIdempotencyKey(t *testing.T) {
	mb := newTestMailbox(t, Policy{})
	ctx := context.Background()
	first, err := mb.Publish(ctx, Envelope{
		MessageID:      "msg-1",
		IdempotencyKey: "idem-same",
		Kind:           KindCommand,
		Payload:        map[string]any{"x": 1},
	})
	if err != nil {
		t.Fatalf("first publish failed: %v", err)
	}
	second, err := mb.Publish(ctx, Envelope{
		MessageID:      "msg-2",
		IdempotencyKey: "idem-same",
		Kind:           KindCommand,
		Payload:        map[string]any{"x": 2},
	})
	if err != nil {
		t.Fatalf("second publish failed: %v", err)
	}
	if second.Duplicate != true {
		t.Fatalf("duplicate publish should be marked duplicate: %#v", second)
	}
	if second.Record.Envelope.MessageID != first.Record.Envelope.MessageID {
		t.Fatalf("duplicate publish should converge to first message, got %#v vs %#v", second.Record, first.Record)
	}
	stats, err := mb.Stats(ctx)
	if err != nil {
		t.Fatalf("stats failed: %v", err)
	}
	if stats.PublishedTotal != 1 || stats.DuplicatePublishTotal != 1 {
		t.Fatalf("publish dedup stats mismatch: %#v", stats)
	}
}

func TestMailboxLifecycleAckNackRetryAndDLQ(t *testing.T) {
	now := time.Now().UTC()
	clock := now
	mb := newTestMailboxWithClock(t, Policy{
		MaxAttempts:    2,
		BackoffInitial: 20 * time.Millisecond,
		BackoffMax:     20 * time.Millisecond,
		JitterRatio:    0,
		DLQEnabled:     true,
	}, func() time.Time { return clock })
	ctx := context.Background()

	if _, err := mb.Publish(ctx, Envelope{
		MessageID:      "msg-lifecycle",
		IdempotencyKey: "idem-lifecycle",
		Kind:           KindCommand,
	}); err != nil {
		t.Fatalf("publish failed: %v", err)
	}
	claimed1, ok, err := mb.Consume(ctx, "worker-1")
	if err != nil || !ok {
		t.Fatalf("first consume failed: ok=%v err=%v", ok, err)
	}
	if claimed1.DeliveryAttempt != 1 || claimed1.State != StateInFlight {
		t.Fatalf("first consume mismatch: %#v", claimed1)
	}
	if _, err := mb.Nack(ctx, claimed1.Envelope.MessageID, "worker-1", "transient"); err != nil {
		t.Fatalf("nack failed: %v", err)
	}
	requeued1, err := mb.Requeue(ctx, claimed1.Envelope.MessageID, "worker-1", "transient")
	if err != nil {
		t.Fatalf("requeue #1 failed: %v", err)
	}
	if requeued1.State != StateQueued {
		t.Fatalf("requeue #1 should return queued state: %#v", requeued1)
	}
	if _, ok, err := mb.Consume(ctx, "worker-1"); err != nil || ok {
		t.Fatalf("consume before retry boundary should be blocked: ok=%v err=%v", ok, err)
	}
	clock = clock.Add(30 * time.Millisecond)
	claimed2, ok, err := mb.Consume(ctx, "worker-1")
	if err != nil || !ok {
		t.Fatalf("second consume failed: ok=%v err=%v", ok, err)
	}
	if claimed2.DeliveryAttempt != 2 {
		t.Fatalf("second consume attempt mismatch: %#v", claimed2)
	}
	requeued2, err := mb.Requeue(ctx, claimed2.Envelope.MessageID, "worker-1", "retry_exhausted")
	if err != nil {
		t.Fatalf("requeue #2 failed: %v", err)
	}
	if requeued2.State != StateDeadLetter {
		t.Fatalf("requeue #2 should enter dead_letter: %#v", requeued2)
	}
	stats, err := mb.Stats(ctx)
	if err != nil {
		t.Fatalf("stats failed: %v", err)
	}
	if stats.NackTotal != 2 || stats.RequeueTotal != 1 || stats.DeadLetterTotal != 1 {
		t.Fatalf("lifecycle stats mismatch: %#v", stats)
	}
}

func TestMailboxDelayedAndExpirySemantics(t *testing.T) {
	now := time.Now().UTC()
	clock := now
	mb := newTestMailboxWithClock(t, Policy{
		MaxAttempts: 3,
		DLQEnabled:  true,
	}, func() time.Time { return clock })
	ctx := context.Background()

	if _, err := mb.Publish(ctx, Envelope{
		MessageID:      "msg-delayed-expired",
		IdempotencyKey: "idem-delayed-expired",
		Kind:           KindCommand,
		NotBefore:      now.Add(300 * time.Millisecond),
		ExpireAt:       now.Add(100 * time.Millisecond),
	}); err != nil {
		t.Fatalf("publish delayed-expired failed: %v", err)
	}
	if _, ok, err := mb.Consume(ctx, "worker-delay"); err != nil || ok {
		t.Fatalf("consume before not_before should be blocked: ok=%v err=%v", ok, err)
	}
	clock = clock.Add(120 * time.Millisecond)
	if _, ok, err := mb.Consume(ctx, "worker-delay"); err != nil || ok {
		t.Fatalf("expired message should not be consumable: ok=%v err=%v", ok, err)
	}
	page, err := mb.Query(ctx, QueryRequest{MessageID: "msg-delayed-expired"})
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].State != StateDeadLetter {
		t.Fatalf("expired message state mismatch: %#v", page.Items)
	}
	stats, err := mb.Stats(ctx)
	if err != nil {
		t.Fatalf("stats failed: %v", err)
	}
	if stats.ExpiredTotal != 1 || stats.DeadLetterTotal != 1 {
		t.Fatalf("expiry stats mismatch: %#v", stats)
	}
}

func TestMailboxSnapshotRestoreMemoryFileParity(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	clock := now
	policy := Policy{
		MaxAttempts:    3,
		BackoffInitial: 10 * time.Millisecond,
		BackoffMax:     10 * time.Millisecond,
		JitterRatio:    0,
		DLQEnabled:     true,
	}

	mem := newTestMailboxWithClock(t, policy, func() time.Time { return clock })
	if _, err := mem.Publish(ctx, Envelope{
		MessageID:      "msg-a",
		IdempotencyKey: "idem-a",
		Kind:           KindCommand,
		RunID:          "run-1",
		TaskID:         "task-1",
	}); err != nil {
		t.Fatalf("publish msg-a failed: %v", err)
	}
	if _, err := mem.Publish(ctx, Envelope{
		MessageID:      "msg-b",
		IdempotencyKey: "idem-b",
		Kind:           KindResult,
		RunID:          "run-1",
		TaskID:         "task-1",
	}); err != nil {
		t.Fatalf("publish msg-b failed: %v", err)
	}
	claimed, ok, err := mem.Consume(ctx, "worker-a")
	if err != nil || !ok {
		t.Fatalf("consume msg-a failed: ok=%v err=%v", ok, err)
	}
	if _, err := mem.Ack(ctx, claimed.Envelope.MessageID, "worker-a"); err != nil {
		t.Fatalf("ack msg-a failed: %v", err)
	}

	snapshot, err := mem.Snapshot(ctx)
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}
	fileStore, err := NewFileStore(filepath.Join(t.TempDir(), "mailbox-state.json"), policy)
	if err != nil {
		t.Fatalf("new file store failed: %v", err)
	}
	fileMB, err := New(fileStore, WithClock(func() time.Time { return clock }))
	if err != nil {
		t.Fatalf("new file mailbox failed: %v", err)
	}
	if err := fileMB.Restore(ctx, snapshot); err != nil {
		t.Fatalf("restore snapshot to file failed: %v", err)
	}
	pageSize := 1
	req := QueryRequest{
		RunID:    "run-1",
		PageSize: &pageSize,
		Sort: QuerySort{
			Field: "updated_at",
			Order: "desc",
		},
	}
	memPages := collectQueryPages(t, mem, req)
	filePages := collectQueryPages(t, fileMB, req)
	if !reflect.DeepEqual(memPages, filePages) {
		t.Fatalf("memory/file parity mismatch: memory=%#v file=%#v", memPages, filePages)
	}
}

func TestNewStoreWithFallbackToMemory(t *testing.T) {
	result, err := NewStoreWithFallback("file", "", Policy{})
	if err != nil {
		t.Fatalf("new store with fallback returned error: %v", err)
	}
	if !result.Fallback || result.Backend != "memory" {
		t.Fatalf("fallback result mismatch: %#v", result)
	}
	mb, err := New(result.Store)
	if err != nil {
		t.Fatalf("new mailbox failed: %v", err)
	}
	stats, err := mb.Stats(context.Background())
	if err != nil {
		t.Fatalf("stats failed: %v", err)
	}
	if !stats.BackendFallback || stats.BackendFallbackReason == "" {
		t.Fatalf("fallback marker missing: %#v", stats)
	}
}

func TestLifecycleReasonTaxonomyFrozen(t *testing.T) {
	expected := []string{
		LifecycleReasonRetryExhausted,
		LifecycleReasonExpired,
		LifecycleReasonConsumerMismatch,
		LifecycleReasonMessageNotFound,
		LifecycleReasonHandlerError,
	}
	if got := LifecycleCanonicalReasons(); !reflect.DeepEqual(got, expected) {
		t.Fatalf("canonical reasons mismatch: got=%#v want=%#v", got, expected)
	}
	for _, reason := range expected {
		if !IsCanonicalLifecycleReason(reason) {
			t.Fatalf("reason %q should be canonical", reason)
		}
	}
	if IsCanonicalLifecycleReason("transient") {
		t.Fatal("unexpected canonical reason: transient")
	}
}

func TestNormalizeWorkerConfigDefaultsAndValidation(t *testing.T) {
	cfg, err := NormalizeWorkerConfig(WorkerConfig{})
	if err != nil {
		t.Fatalf("NormalizeWorkerConfig default failed: %v", err)
	}
	if cfg.PollInterval != DefaultWorkerPollInterval {
		t.Fatalf("default poll_interval=%v, want %v", cfg.PollInterval, DefaultWorkerPollInterval)
	}
	if cfg.HandlerErrorPolicy != WorkerHandlerErrorPolicyRequeue {
		t.Fatalf("default handler_error_policy=%q, want %q", cfg.HandlerErrorPolicy, WorkerHandlerErrorPolicyRequeue)
	}

	if _, err := NormalizeWorkerConfig(WorkerConfig{PollInterval: 0}); err != nil {
		t.Fatalf("zero poll_interval should resolve to default: %v", err)
	}
	if _, err := NormalizeWorkerConfig(WorkerConfig{PollInterval: -1 * time.Millisecond}); err == nil {
		t.Fatal("expected validation error for poll_interval < 0")
	}
	if _, err := NormalizeWorkerConfig(WorkerConfig{PollInterval: 10 * time.Millisecond, HandlerErrorPolicy: "drop"}); err == nil {
		t.Fatal("expected validation error for unsupported handler_error_policy")
	}
}

func TestMailboxWorkerRunOnceDefaultRequeuePolicy(t *testing.T) {
	now := time.Now().UTC()
	clock := now
	events := make([]LifecycleEvent, 0, 8)
	mb, err := New(
		NewMemoryStore(Policy{
			MaxAttempts:    3,
			BackoffInitial: 10 * time.Millisecond,
			BackoffMax:     10 * time.Millisecond,
			JitterRatio:    0,
		}),
		WithClock(func() time.Time { return clock }),
		WithLifecycleObserver(func(_ context.Context, event LifecycleEvent) {
			events = append(events, event)
		}),
	)
	if err != nil {
		t.Fatalf("new mailbox failed: %v", err)
	}
	ctx := context.Background()
	if _, err := mb.Publish(ctx, Envelope{
		MessageID:      "msg-worker-default",
		IdempotencyKey: "idem-worker-default",
		Kind:           KindCommand,
	}); err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	calls := 0
	worker, err := NewWorker(mb, WorkerConfig{Enabled: true}, func(_ context.Context, _ Record) error {
		calls++
		if calls == 1 {
			return errors.New("temporary")
		}
		return nil
	}, "worker-default")
	if err != nil {
		t.Fatalf("new worker failed: %v", err)
	}
	processed, err := worker.RunOnce(ctx)
	if err != nil || !processed {
		t.Fatalf("first RunOnce failed: processed=%v err=%v", processed, err)
	}
	clock = clock.Add(20 * time.Millisecond)
	processed, err = worker.RunOnce(ctx)
	if err != nil || !processed {
		t.Fatalf("second RunOnce failed: processed=%v err=%v", processed, err)
	}

	snapshot, err := mb.Snapshot(ctx)
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}
	rec := snapshot.Records[0]
	if rec.State != StateAcked {
		t.Fatalf("worker terminal state=%q, want acked", rec.State)
	}

	got := make([]LifecycleTransition, 0, len(events))
	for _, event := range events {
		got = append(got, event.Transition)
		if strings.TrimSpace(event.ReasonCode) != "" && !IsCanonicalLifecycleReason(event.ReasonCode) {
			t.Fatalf("event reason must be canonical: %#v", event)
		}
	}
	want := []LifecycleTransition{
		TransitionConsume,
		TransitionNack,
		TransitionRequeue,
		TransitionConsume,
		TransitionAck,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("worker lifecycle transitions mismatch: got=%#v want=%#v", got, want)
	}
}

func TestMailboxWorkerDisabledNoop(t *testing.T) {
	mb := newTestMailbox(t, Policy{})
	ctx := context.Background()
	if _, err := mb.Publish(ctx, Envelope{
		MessageID:      "msg-worker-disabled",
		IdempotencyKey: "idem-worker-disabled",
		Kind:           KindCommand,
	}); err != nil {
		t.Fatalf("publish failed: %v", err)
	}
	worker, err := NewWorker(mb, WorkerConfig{Enabled: false}, func(context.Context, Record) error {
		return nil
	}, "worker-disabled")
	if err != nil {
		t.Fatalf("new worker failed: %v", err)
	}
	if err := worker.Run(ctx); err != nil {
		t.Fatalf("disabled worker run failed: %v", err)
	}
	claimed, ok, err := mb.Consume(ctx, "worker-disabled-check")
	if err != nil || !ok {
		t.Fatalf("message should remain queued when worker disabled: ok=%v err=%v", ok, err)
	}
	if claimed.State != StateInFlight {
		t.Fatalf("claimed state=%q, want in_flight", claimed.State)
	}
}

func TestMailboxLifecycleObserverExpiredAndDeadLetter(t *testing.T) {
	now := time.Now().UTC()
	clock := now
	events := make([]LifecycleEvent, 0, 4)
	mb, err := New(
		NewMemoryStore(Policy{
			MaxAttempts: 3,
			DLQEnabled:  true,
		}),
		WithClock(func() time.Time { return clock }),
		WithLifecycleObserver(func(_ context.Context, event LifecycleEvent) {
			events = append(events, event)
		}),
	)
	if err != nil {
		t.Fatalf("new mailbox failed: %v", err)
	}
	ctx := context.Background()
	if _, err := mb.Publish(ctx, Envelope{
		MessageID:      "msg-expire-dlq",
		IdempotencyKey: "idem-expire-dlq",
		Kind:           KindCommand,
		NotBefore:      now.Add(300 * time.Millisecond),
		ExpireAt:       now.Add(80 * time.Millisecond),
	}); err != nil {
		t.Fatalf("publish failed: %v", err)
	}
	clock = clock.Add(120 * time.Millisecond)
	if _, ok, err := mb.Consume(ctx, "worker-expire"); err != nil || ok {
		t.Fatalf("consume should not return expired message: ok=%v err=%v", ok, err)
	}

	if len(events) < 2 {
		t.Fatalf("lifecycle observer should emit expired/dead_letter events, got=%#v", events)
	}
	transitions := []LifecycleTransition{events[0].Transition, events[1].Transition}
	want := []LifecycleTransition{TransitionDeadLetter, TransitionExpired}
	if !reflect.DeepEqual(transitions, want) {
		t.Fatalf("expiry transitions mismatch: got=%#v want=%#v", transitions, want)
	}
	for _, event := range events {
		if event.Transition == TransitionDeadLetter || event.Transition == TransitionExpired {
			if event.ReasonCode != LifecycleReasonExpired {
				t.Fatalf("expiry reason_code=%q, want %q", event.ReasonCode, LifecycleReasonExpired)
			}
		}
	}
}

func newTestMailbox(t *testing.T, policy Policy) *Mailbox {
	t.Helper()
	mb, err := New(NewMemoryStore(policy))
	if err != nil {
		t.Fatalf("new mailbox failed: %v", err)
	}
	return mb
}

func newTestMailboxWithClock(t *testing.T, policy Policy, now func() time.Time) *Mailbox {
	t.Helper()
	mb, err := New(NewMemoryStore(policy), WithClock(now))
	if err != nil {
		t.Fatalf("new mailbox failed: %v", err)
	}
	return mb
}

func collectQueryPages(t *testing.T, mb *Mailbox, req QueryRequest) []QueryResult {
	t.Helper()
	ctx := context.Background()
	pages := make([]QueryResult, 0)
	seen := map[string]struct{}{}
	current := req
	for {
		page, err := mb.Query(ctx, current)
		if err != nil {
			t.Fatalf("query failed: %v", err)
		}
		pages = append(pages, page)
		if page.NextCursor == "" {
			break
		}
		if _, ok := seen[page.NextCursor]; ok {
			t.Fatalf("cursor should advance deterministically without loops: %q", page.NextCursor)
		}
		seen[page.NextCursor] = struct{}{}
		current.Cursor = page.NextCursor
	}
	return pages
}
