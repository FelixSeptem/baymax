package mailbox

import (
	"context"
	"testing"
	"time"
)

func TestMailboxQueryDefaultsAndAndSemantics(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	clock := now
	mb := newTestMailboxWithClock(t, Policy{}, func() time.Time { return clock })

	mustPublish := func(env Envelope) {
		t.Helper()
		if _, err := mb.Publish(ctx, env); err != nil {
			t.Fatalf("publish failed: %v", err)
		}
	}
	mustPublish(Envelope{
		MessageID:      "msg-q-1",
		IdempotencyKey: "idem-q-1",
		Kind:           KindCommand,
		RunID:          "run-a",
		TaskID:         "task-a",
		WorkflowID:     "wf-a",
		TeamID:         "team-a",
	})
	clock = clock.Add(1 * time.Millisecond)
	mustPublish(Envelope{
		MessageID:      "msg-q-2",
		IdempotencyKey: "idem-q-2",
		Kind:           KindResult,
		RunID:          "run-a",
		TaskID:         "task-a",
		WorkflowID:     "wf-a",
		TeamID:         "team-a",
	})
	clock = clock.Add(1 * time.Millisecond)
	mustPublish(Envelope{
		MessageID:      "msg-q-3",
		IdempotencyKey: "idem-q-3",
		Kind:           KindCommand,
		RunID:          "run-b",
		TaskID:         "task-b",
		WorkflowID:     "wf-b",
		TeamID:         "team-b",
	})

	result, err := mb.Query(ctx, QueryRequest{
		RunID: "run-a",
		Kind:  "result",
	})
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if result.PageSize != DefaultQueryPageSize {
		t.Fatalf("default page_size = %d, want %d", result.PageSize, DefaultQueryPageSize)
	}
	if result.SortField != "updated_at" || result.SortOrder != "desc" {
		t.Fatalf("default sort mismatch: %#v", result)
	}
	if len(result.Items) != 1 || result.Items[0].Envelope.MessageID != "msg-q-2" {
		t.Fatalf("AND filter result mismatch: %#v", result.Items)
	}
}

func TestMailboxQueryValidationAndCursorDeterminism(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	clock := now
	mb := newTestMailboxWithClock(t, Policy{}, func() time.Time { return clock })

	for i := 0; i < 3; i++ {
		clock = clock.Add(1 * time.Millisecond)
		if _, err := mb.Publish(ctx, Envelope{
			MessageID:      "msg-cursor-" + string(rune('a'+i)),
			IdempotencyKey: "idem-cursor-" + string(rune('a'+i)),
			Kind:           KindCommand,
			RunID:          "run-cursor",
		}); err != nil {
			t.Fatalf("publish #%d failed: %v", i, err)
		}
	}

	pageSize := 1
	first, err := mb.Query(ctx, QueryRequest{
		RunID:    "run-cursor",
		PageSize: &pageSize,
	})
	if err != nil {
		t.Fatalf("first page query failed: %v", err)
	}
	if len(first.Items) != 1 || first.NextCursor == "" {
		t.Fatalf("first page mismatch: %#v", first)
	}
	second, err := mb.Query(ctx, QueryRequest{
		RunID:    "run-cursor",
		PageSize: &pageSize,
		Cursor:   first.NextCursor,
	})
	if err != nil {
		t.Fatalf("second page query failed: %v", err)
	}
	secondReplay, err := mb.Query(ctx, QueryRequest{
		RunID:    "run-cursor",
		PageSize: &pageSize,
		Cursor:   first.NextCursor,
	})
	if err != nil {
		t.Fatalf("second page replay failed: %v", err)
	}
	if len(second.Items) != 1 || len(secondReplay.Items) != 1 || second.Items[0].Envelope.MessageID != secondReplay.Items[0].Envelope.MessageID {
		t.Fatalf("cursor determinism mismatch: second=%#v replay=%#v", second, secondReplay)
	}

	tooLarge := 201
	if _, err := mb.Query(ctx, QueryRequest{PageSize: &tooLarge}); err == nil {
		t.Fatal("expected fail-fast for page_size > 200")
	}
	invalid := 0
	if _, err := mb.Query(ctx, QueryRequest{PageSize: &invalid}); err == nil {
		t.Fatal("expected fail-fast for page_size <= 0")
	}
	if _, err := mb.Query(ctx, QueryRequest{State: "running"}); err == nil {
		t.Fatal("expected fail-fast for invalid state")
	}
	if _, err := mb.Query(ctx, QueryRequest{
		Sort: QuerySort{Field: "run_id", Order: "desc"},
	}); err == nil {
		t.Fatal("expected fail-fast for unsupported sort field")
	}
	if _, err := mb.Query(ctx, QueryRequest{
		Sort: QuerySort{Field: "updated_at", Order: "down"},
	}); err == nil {
		t.Fatal("expected fail-fast for unsupported sort order")
	}
	if _, err := mb.Query(ctx, QueryRequest{
		TimeRange: &QueryTimeRange{
			Start: time.Now().Add(2 * time.Minute),
			End:   time.Now().Add(1 * time.Minute),
		},
	}); err == nil {
		t.Fatal("expected fail-fast for invalid time range")
	}
	if _, err := mb.Query(ctx, QueryRequest{
		RunID:    "run-cursor",
		PageSize: &pageSize,
		Cursor:   "not-a-valid-cursor",
	}); err == nil {
		t.Fatal("expected fail-fast for malformed cursor")
	}
	if _, err := mb.Query(ctx, QueryRequest{
		RunID:    "run-other",
		PageSize: &pageSize,
		Cursor:   first.NextCursor,
	}); err == nil {
		t.Fatal("expected fail-fast for query boundary mismatch cursor")
	}
}

func TestMailboxQueryCacheInvalidatesOnMutation(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	clock := now
	mb := newTestMailboxWithClock(t, Policy{}, func() time.Time { return clock })

	publish := func(id string) {
		t.Helper()
		if _, err := mb.Publish(ctx, Envelope{
			MessageID:      id,
			IdempotencyKey: "idem-" + id,
			Kind:           KindCommand,
			RunID:          "run-cache",
		}); err != nil {
			t.Fatalf("publish %q failed: %v", id, err)
		}
	}

	publish("msg-cache-1")
	clock = clock.Add(time.Millisecond)
	publish("msg-cache-2")
	first, err := mb.Query(ctx, QueryRequest{RunID: "run-cache"})
	if err != nil {
		t.Fatalf("first query failed: %v", err)
	}
	firstCount := len(first.Items)
	if firstCount != 2 {
		t.Fatalf("first query count=%d, want 2", firstCount)
	}

	clock = clock.Add(time.Millisecond)
	publish("msg-cache-3")
	second, err := mb.Query(ctx, QueryRequest{RunID: "run-cache"})
	if err != nil {
		t.Fatalf("second query failed: %v", err)
	}
	if len(second.Items) != firstCount+1 {
		t.Fatalf("query cache should invalidate after mutation: first=%d second=%d", firstCount, len(second.Items))
	}
}
