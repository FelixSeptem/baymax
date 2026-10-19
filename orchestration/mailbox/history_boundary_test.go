package mailbox

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

func TestMailboxHistoryNeverRetainsLegacyCallbackError(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore(Policy{})
	mb, err := New(store)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = mb.Publish(ctx, Envelope{MessageID: "legacy", IdempotencyKey: "legacy", Kind: KindCommand}); err != nil {
		t.Fatal(err)
	}
	if _, _, err = mb.Consume(ctx, "worker"); err != nil {
		t.Fatal(err)
	}
	// Legacy snapshots may contain a LastError predating reason canonicalization.
	store.state.Records["legacy"].LastError = "credential=raw-callback-error"
	if _, err = mb.Ack(ctx, "legacy", "worker"); err != nil {
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
	raw, err := json.Marshal(page)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "credential=") {
		t.Fatal("raw legacy callback error escaped into source journal")
	}
}

func TestMailboxHistoryCursorShapeAndLimitsFailClosed(t *testing.T) {
	ctx := context.Background()
	mb, err := New(NewMemoryStore(Policy{}))
	if err != nil {
		t.Fatal(err)
	}
	source, err := mb.HistorySource(ctx)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := base64.RawURLEncoding.DecodeString(encodeHistoryCursor(source, 0).Value)
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{
		string(canonical) + " ",
		strings.Replace(string(canonical), HistoryVersion, "mailbox.history.v999", 1),
		strings.Replace(string(canonical), `"after":0`, `"after":null`, 1),
		strings.Replace(string(canonical), `"after":0`, `"after":0,"after":0`, 1),
		strings.Replace(string(canonical), `,"after":0`, "", 1),
	} {
		cursor := &types.OpaqueCursor{Value: base64.RawURLEncoding.EncodeToString([]byte(raw))}
		if _, err := mb.ReadAfter(ctx, source, cursor, 1); !errors.Is(err, types.ErrCursorMalformed) {
			t.Fatalf("accepted noncanonical cursor: %s: %v", raw, err)
		}
	}
	for _, limit := range []int{-1, 0, types.MaxHistoryPageSize + 1} {
		if _, err := mb.ReadAfter(ctx, source, nil, limit); !errors.Is(err, types.ErrHistoryInvalid) {
			t.Fatal(err)
		}
	}
	for _, limit := range []int{1, types.MaxHistoryPageSize} {
		page, err := mb.ReadAfter(ctx, source, nil, limit)
		if err != nil || !page.Complete || page.NextCursor == nil || len(page.Events) != 0 {
			t.Fatalf("empty bounded page: %+v %v", page, err)
		}
	}
}

func TestMailboxHistoryPersistFailureCannotFlushOnRead(t *testing.T) {
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
	if _, err = mb.Publish(ctx, Envelope{MessageID: "committed", IdempotencyKey: "committed", Kind: KindCommand}); err != nil {
		t.Fatal(err)
	}
	source, err := mb.HistorySource(ctx)
	if err != nil {
		t.Fatal(err)
	}
	store.path = t.TempDir() // rename onto an existing directory fails on both platforms.
	if _, err = mb.Publish(ctx, Envelope{MessageID: "failed", IdempotencyKey: "failed", Kind: KindCommand}); err == nil {
		t.Fatal("expected persistence failure")
	}
	store.path = path
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, read := range []func() error{
		func() error { _, err := mb.Capabilities(ctx, source); return err },
		func() error { _, err := mb.ReadAfter(ctx, source, nil, 10); return err },
	} {
		if err := read(); !errors.Is(err, types.ErrSourceUnavailable) {
			t.Fatalf("uncommitted history exposed: %v", err)
		}
	}
	after, err := os.ReadFile(path)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("read flushed pending source mutation: %v", err)
	}
	if err = store.Flush(); err != nil {
		t.Fatal(err)
	}
	if _, err = mb.ReadAfter(ctx, source, nil, 10); err != nil {
		t.Fatal(err)
	}
}

func TestMailboxHistoryRetentionFloorIsValidAfterPosition(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore(Policy{})
	mb, err := New(store)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < HistoryRetentionEntries+1; i++ {
		// Exercise the owner append without spending time normalizing unrelated envelopes.
		store.state.appendHistory("admit", Record{Envelope: Envelope{MessageID: "message"}, UpdatedAt: time.Now().UTC()}, "", ActionOptions{})
	}
	source, err := mb.HistorySource(ctx)
	if err != nil {
		t.Fatal(err)
	}
	page, err := mb.ReadAfter(ctx, source, encodeHistoryCursor(source, 1), 1)
	if err != nil || len(page.Events) != 1 || page.Events[0].History.Sequence != 2 || page.Complete {
		t.Fatalf("floor boundary lost first retained event: %+v %v", page, err)
	}
	if _, err = mb.ReadAfter(ctx, source, encodeHistoryCursor(source, 0), 1); !errors.Is(err, types.ErrCursorExpired) {
		t.Fatal(err)
	}
}

func TestMailboxHistoryExpiryPreservesTransitionTerminal(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	mb, err := New(NewMemoryStore(Policy{DLQEnabled: true}), WithClock(func() time.Time { return now }))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = mb.Publish(ctx, Envelope{MessageID: "expired-terminal", IdempotencyKey: "expired-terminal", Kind: KindCommand, ExpireAt: now.Add(-time.Second)}); err != nil {
		t.Fatal(err)
	}
	if _, _, err = mb.Consume(ctx, "expiry-worker"); err != nil {
		t.Fatal(err)
	}
	source, err := mb.HistorySource(ctx)
	if err != nil {
		t.Fatal(err)
	}
	page, err := mb.ReadAfter(ctx, source, nil, 10)
	if err != nil || len(page.Events) != 3 {
		t.Fatalf("expiry history: %+v %v", page, err)
	}
	if page.Events[1].History.TerminalOutcome != "dead_letter" || page.Events[2].History.TerminalOutcome != "dead_letter" {
		t.Fatalf("transition terminal evidence drifted: %+v", page.Events)
	}
}
