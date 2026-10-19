package mailbox

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

func TestFileMailboxHistoryReplaysAfterRestartWithoutPayload(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "mailbox.json")
	store, err := NewFileStore(path, Policy{})
	if err != nil {
		t.Fatal(err)
	}
	mb, err := New(store)
	if err != nil {
		t.Fatal(err)
	}
	_, err = mb.Publish(ctx, Envelope{MessageID: "history-message", IdempotencyKey: "history-idem", Kind: KindCommand, Payload: map[string]any{"secret": "must-not-persist"}})
	if err != nil {
		t.Fatal(err)
	}
	source, err := mb.HistorySource(ctx)
	if err != nil {
		t.Fatal(err)
	}
	page, err := mb.ReadAfter(ctx, source, nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Events) != 1 || page.Events[0].NativeType != "mailbox.admit" {
		t.Fatalf("admission history: %+v", page)
	}
	if page.NextCursor == nil {
		t.Fatal("missing cursor")
	}
	if _, ok := page.Events[0].Event.Payload["secret"]; ok {
		t.Fatal("payload retained")
	}
	store2, err := NewFileStore(path, Policy{})
	if err != nil {
		t.Fatal(err)
	}
	mb2, err := New(store2)
	if err != nil {
		t.Fatal(err)
	}
	page2, err := mb2.ReadAfter(ctx, source, page.NextCursor, 10)
	if err != nil || len(page2.Events) != 0 || !page2.Complete {
		t.Fatalf("restart replay: %+v %v", page2, err)
	}
	if _, err = mb2.ReadAfter(ctx, source, &types.OpaqueCursor{Value: "not-a-cursor"}, 10); !errors.Is(err, types.ErrCursorMalformed) {
		t.Fatalf("malformed cursor: %v", err)
	}
}

func TestMailboxHistoryLifecycleAndCursorErrorsAreReadOnly(t *testing.T) {
	ctx := context.Background()
	mb, err := New(NewMemoryStore(Policy{MaxAttempts: 1, DLQEnabled: true}))
	if err != nil {
		t.Fatal(err)
	}
	_, err = mb.Publish(ctx, Envelope{MessageID: "lifecycle", IdempotencyKey: "idem", Kind: KindCommand})
	if err != nil {
		t.Fatal(err)
	}
	leased, ok, err := mb.Consume(ctx, "worker")
	if err != nil || !ok {
		t.Fatalf("consume: %v %v", ok, err)
	}
	if _, err = mb.Nack(ctx, leased.Envelope.MessageID, "worker", "handler_error"); err != nil {
		t.Fatal(err)
	}
	source, err := mb.HistorySource(ctx)
	if err != nil {
		t.Fatal(err)
	}
	caps, err := mb.Capabilities(ctx, source)
	if err != nil || caps.Durable {
		t.Fatalf("memory durability declaration: %+v %v", caps, err)
	}
	page, err := mb.ReadAfter(ctx, source, nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Events) != 4 || page.Events[0].NativeType != "mailbox.admit" || page.Events[1].NativeType != "mailbox.consume" || page.Events[2].NativeType != "mailbox.nack" || page.Events[3].NativeType != "mailbox.dead_letter" {
		t.Fatalf("lifecycle order: %+v", page.Events)
	}
	before, err := mb.Stats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = mb.ReadAfter(ctx, types.SourceRef{Kind: types.HistorySourceMailbox, Namespace: source.Namespace, ID: "foreign"}, page.NextCursor, 10); !errors.Is(err, types.ErrCursorForeign) {
		t.Fatalf("foreign cursor: %v", err)
	}
	bad := encodeHistoryCursor(source, 100)
	if _, err = mb.ReadAfter(ctx, source, bad, 10); !errors.Is(err, types.ErrHistoryGap) {
		t.Fatalf("future cursor: %v", err)
	}
	after, err := mb.Stats(ctx)
	if err != nil || before != after {
		t.Fatalf("history read changed mailbox stats: before=%+v after=%+v err=%v", before, after, err)
	}
	for i := 0; i < HistoryRetentionEntries; i++ {
		id := fmt.Sprintf("retained-%d", i)
		if _, err := mb.Publish(ctx, Envelope{MessageID: id, IdempotencyKey: id, Kind: KindCommand}); err != nil {
			t.Fatal(err)
		}
	}
	if floor, err := mb.ReadAfter(ctx, source, encodeHistoryCursor(source, 0), 10); !errors.Is(err, types.ErrCursorExpired) || floor.RetentionFloor == nil || len(floor.Events) != 0 {
		t.Fatalf("expiry must fail with floor only: %+v %v", floor, err)
	}
}

func TestMailboxHistoryTerminalRestartProcess(t *testing.T) {
	ctx := context.Background()
	if path := os.Getenv("BAYMAX_HISTORY_TEST_PATH"); path != "" {
		store, err := NewFileStore(path, Policy{})
		if err != nil {
			t.Fatal(err)
		}
		mb, err := New(store)
		if err != nil {
			t.Fatal(err)
		}
		source, err := mb.HistorySource(ctx)
		if err != nil {
			t.Fatal(err)
		}
		before, err := mb.Snapshot(ctx)
		if err != nil {
			t.Fatal(err)
		}
		page, err := mb.ReadAfter(ctx, source, &types.OpaqueCursor{Value: os.Getenv("BAYMAX_HISTORY_TEST_CURSOR")}, 2)
		if err != nil || len(page.Events) != 2 || page.Events[1].History.TerminalOutcome != "acked" || page.Events[1].History.Sequence != 3 {
			t.Fatalf("terminal restart: %+v %v", page, err)
		}
		after, err := mb.Snapshot(ctx)
		if err != nil || !reflect.DeepEqual(before, after) {
			t.Fatal("replay changed source snapshot")
		}
		return
	}
	path := filepath.Join(t.TempDir(), "retained.json")
	store, err := NewFileStore(path, Policy{})
	if err != nil {
		t.Fatal(err)
	}
	mb, err := New(store)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = mb.Publish(ctx, Envelope{MessageID: "terminal", IdempotencyKey: "terminal", Kind: KindCommand, RunID: "same-run"}); err != nil {
		t.Fatal(err)
	}
	source, err := mb.HistorySource(ctx)
	if err != nil {
		t.Fatal(err)
	}
	page, err := mb.ReadAfter(ctx, source, nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok, err := mb.Consume(ctx, "owner"); err != nil || !ok {
		t.Fatalf("consume %v %v", ok, err)
	}
	if _, err = mb.Ack(ctx, "terminal", "owner"); err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestMailboxHistoryTerminalRestartProcess$", "-test.count=1")
	cmd.Env = append(os.Environ(), "BAYMAX_HISTORY_TEST_PATH="+path, "BAYMAX_HISTORY_TEST_CURSOR="+page.NextCursor.Value)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("new process: %v %s", err, out)
	}
}

func TestMailboxHistoryLeaseRetryTimeoutAndIsolation(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	store := NewMemoryStore(Policy{MaxAttempts: 2, BackoffInitial: time.Millisecond, BackoffMax: time.Millisecond, DLQEnabled: true})
	mb, err := New(store, WithClock(func() time.Time { return now }))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = mb.Publish(ctx, Envelope{MessageID: "lease", IdempotencyKey: "lease", Kind: KindCommand}); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := mb.ConsumeWithLease(ctx, "first", time.Millisecond, true); err != nil || !ok {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Millisecond)
	if _, ok, err := mb.ConsumeWithLease(ctx, "second", time.Millisecond, true); err != nil || ok {
		t.Fatal("expected backoff after reclaim", err)
	}
	now = now.Add(2 * time.Millisecond)
	if _, ok, err := mb.ConsumeWithLease(ctx, "second", time.Millisecond, true); err != nil || !ok {
		t.Fatal(err)
	}
	if _, err = mb.Requeue(ctx, "lease", "second", "raw credentials must not leak"); err != nil {
		t.Fatal(err)
	}
	if _, err = mb.Publish(ctx, Envelope{MessageID: "timeout", IdempotencyKey: "timeout", Kind: KindCommand, ExpireAt: now.Add(-time.Second)}); err != nil {
		t.Fatal(err)
	}
	if _, _, err = mb.Consume(ctx, "none"); err != nil {
		t.Fatal(err)
	}
	source, err := mb.HistorySource(ctx)
	if err != nil {
		t.Fatal(err)
	}
	page, err := mb.ReadAfter(ctx, source, nil, types.MaxHistoryPageSize)
	if err != nil {
		t.Fatal(err)
	}
	var leaseLost, retry, dlq, timeout bool
	for _, ev := range page.Events {
		if ev.Event.Payload["lifecycle_result"] == "lease_lost" {
			leaseLost = true
		}
		if ev.NativeType == "mailbox.requeue" {
			retry = true
		}
		if ev.History.TerminalOutcome == "dead_letter" {
			dlq = true
		}
		if ev.NativeType == "mailbox.expired" {
			timeout = true
		}
		if ev.History.Sequence == 0 || ev.Event.Sequence != 0 {
			t.Fatal("invalid source-local order")
		}
	}
	if !leaseLost || !retry || !dlq || !timeout {
		t.Fatalf("missing lifecycle facts: %t %t %t %t", leaseLost, retry, dlq, timeout)
	}
	page.Events[0].History.Source.ID = "mutated"
	page.Events[0].Event.Payload["message_id"] = "mutated"
	again, err := mb.ReadAfter(ctx, source, nil, 1)
	if err != nil || again.Events[0].Event.Payload["message_id"] != "lease" {
		t.Fatal("caller mutation escaped", err)
	}
	store.mu.Lock()
	store.state.History.Entries = append(store.state.History.Entries[:2], store.state.History.Entries[3:]...)
	store.mu.Unlock()
	if _, err = mb.ReadAfter(ctx, source, nil, 1); !errors.Is(err, types.ErrHistoryGap) {
		t.Fatal("missing gap error", err)
	}
}

func TestMailboxHistoryUnavailableUntilBatchCommit(t *testing.T) {
	ctx := context.Background()
	store, err := NewFileStore(filepath.Join(t.TempDir(), "batched.json"), Policy{}, WithPersistBatchSize(10))
	if err != nil {
		t.Fatal(err)
	}
	mb, err := New(store)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = mb.Publish(ctx, Envelope{MessageID: "pending", IdempotencyKey: "pending", Kind: KindCommand}); err != nil {
		t.Fatal(err)
	}
	if _, err = mb.HistorySource(ctx); !errors.Is(err, types.ErrSourceUnavailable) {
		t.Fatalf("pending history exposed: %v", err)
	}
	if err = store.Flush(); err != nil {
		t.Fatal(err)
	}
	if _, err = mb.HistorySource(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestMailboxHistoryConcurrentPages(t *testing.T) {
	ctx := context.Background()
	mb, err := New(NewMemoryStore(Policy{}))
	if err != nil {
		t.Fatal(err)
	}
	source, err := mb.HistorySource(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for worker := 0; worker < 4; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				id := fmt.Sprintf("%d-%d", worker, i)
				if _, err := mb.Publish(ctx, Envelope{MessageID: id, IdempotencyKey: id, Kind: KindCommand}); err != nil {
					t.Error(err)
					return
				}
				page, err := mb.ReadAfter(ctx, source, nil, types.MaxHistoryPageSize)
				if err != nil {
					t.Error(err)
					return
				}
				for j, ev := range page.Events {
					if ev.History.Source != source || ev.History.Sequence != uint64(j+1) {
						t.Error("source order drift")
						return
					}
				}
			}
		}(worker)
	}
	wg.Wait()
}
