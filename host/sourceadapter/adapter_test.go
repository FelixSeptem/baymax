package sourceadapter_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/FelixSeptem/baymax/host/sourceadapter"
)

type sourceFunc func(context.Context, *sourceadapter.Cursor, int) (sourceadapter.Page, error)

func (f sourceFunc) Poll(ctx context.Context, after *sourceadapter.Cursor, limit int) (sourceadapter.Page, error) {
	return f(ctx, after, limit)
}

type consumerFunc func(context.Context, sourceadapter.Fact) (sourceadapter.Disposition, error)

func (f consumerFunc) Consume(ctx context.Context, fact sourceadapter.Fact) (sourceadapter.Disposition, error) {
	return f(ctx, fact)
}

func metadata() sourceadapter.Metadata {
	return sourceadapter.Metadata{Source: sourceadapter.SourceIdentity{Kind: "fixture", Namespace: "test", ID: "inbox"}, IdentityNamespace: "message", CursorVersion: "v1"}
}

func fact(id, token, body string) sourceadapter.Fact {
	m := metadata()
	digest := sha256.Sum256([]byte(body))
	return sourceadapter.Fact{ExternalID: id, Reference: "message/" + id, Payload: []byte(body), PayloadDigest: "sha256:" + hex.EncodeToString(digest[:]), Next: sourceadapter.Cursor{Metadata: m, Token: token}}
}

func newAdapter(t *testing.T, store sourceadapter.Store, consumer consumerFunc) *sourceadapter.Adapter {
	t.Helper()
	a, err := sourceadapter.New(metadata(), sourceFunc(func(context.Context, *sourceadapter.Cursor, int) (sourceadapter.Page, error) {
		return sourceadapter.Page{Metadata: metadata()}, nil
	}), consumer, store, sourceadapter.Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(a.Dispose)
	return a
}

func load(t *testing.T, store sourceadapter.Store) sourceadapter.State {
	t.Helper()
	s, err := store.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func memory(t *testing.T) *sourceadapter.MemoryStore {
	t.Helper()
	s, err := sourceadapter.NewMemoryStore(metadata())
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestDispositionConfirmsOnlyPrefix(t *testing.T) {
	for _, outcome := range []sourceadapter.Disposition{sourceadapter.Sent, sourceadapter.Gone} {
		t.Run(string(outcome), func(t *testing.T) {
			store := memory(t)
			var calls []string
			a := newAdapter(t, store, func(_ context.Context, f sourceadapter.Fact) (sourceadapter.Disposition, error) {
				calls = append(calls, f.ExternalID)
				if f.ExternalID == "second" {
					return sourceadapter.Retryable, nil
				}
				return outcome, nil
			})
			err := a.Offer(context.Background(), sourceadapter.Page{Metadata: metadata(), Facts: []sourceadapter.Fact{fact("first", "1", "one"), fact("second", "2", "two"), fact("third", "3", "three")}})
			if !errors.Is(err, sourceadapter.ErrRetryable) {
				t.Fatalf("want retryable, got %v", err)
			}
			s := load(t, store)
			if len(calls) != 2 || s.Confirmed == nil || s.Confirmed.Token != "1" || s.Pending == nil || s.Pending.ExternalID != "second" {
				t.Fatalf("calls=%v state=%+v", calls, s)
			}
		})
	}
}

func TestWholePageValidationPrecedesEffects(t *testing.T) {
	for _, change := range []string{"foreign", "digest", "page-conflict"} {
		t.Run(change, func(t *testing.T) {
			store := memory(t)
			calls := 0
			a := newAdapter(t, store, func(context.Context, sourceadapter.Fact) (sourceadapter.Disposition, error) {
				calls++
				return sourceadapter.Sent, nil
			})
			p := sourceadapter.Page{Metadata: metadata(), Facts: []sourceadapter.Fact{fact("one", "1", "one"), fact("two", "2", "two")}}
			switch change {
			case "foreign":
				p.Facts[1].Next.Metadata.Source.ID = "foreign"
			case "digest":
				p.Facts[1].Payload = []byte("changed")
			case "page-conflict":
				p.Facts[1] = fact("one", "2", "changed")
			}
			if err := a.Offer(context.Background(), p); err == nil {
				t.Fatal("invalid page accepted")
			}
			if calls != 0 || load(t, store).Revision != 0 {
				t.Fatal("validation had side effects")
			}
		})
	}
}

func TestPendingConflictSurvivesNewAdapter(t *testing.T) {
	store := memory(t)
	a := newAdapter(t, store, func(context.Context, sourceadapter.Fact) (sourceadapter.Disposition, error) {
		return sourceadapter.Retryable, nil
	})
	if err := a.Offer(context.Background(), sourceadapter.Page{Metadata: metadata(), Facts: []sourceadapter.Fact{fact("one", "1", "body")}}); !errors.Is(err, sourceadapter.ErrRetryable) {
		t.Fatal(err)
	}
	a.Dispose()
	calls := 0
	b := newAdapter(t, store, func(context.Context, sourceadapter.Fact) (sourceadapter.Disposition, error) {
		calls++
		return sourceadapter.Sent, nil
	})
	err := b.Offer(context.Background(), sourceadapter.Page{Metadata: metadata(), Facts: []sourceadapter.Fact{fact("one", "1", "changed")}})
	if !errors.Is(err, sourceadapter.ErrConflict) || calls != 0 || load(t, store).Confirmed != nil {
		t.Fatalf("conflict=%v calls=%d", err, calls)
	}
}

func TestPollAndOfferUseConfirmedCursor(t *testing.T) {
	store := memory(t)
	calls := 0
	a, err := sourceadapter.New(metadata(), sourceFunc(func(_ context.Context, after *sourceadapter.Cursor, limit int) (sourceadapter.Page, error) {
		calls++
		if limit != 200 {
			t.Fatalf("limit=%d", limit)
		}
		if calls == 2 && (after == nil || after.Token != "1") {
			t.Fatalf("after=%+v", after)
		}
		return sourceadapter.Page{Metadata: metadata(), After: after, Facts: []sourceadapter.Fact{fact("one", "1", "one")}}, nil
	}), consumerFunc(func(context.Context, sourceadapter.Fact) (sourceadapter.Disposition, error) {
		return sourceadapter.Sent, nil
	}), store, sourceadapter.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer a.Dispose()
	for range 2 {
		if err := a.RunOnce(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if len(load(t, store).Recent) != 1 {
		t.Fatal("duplicate expanded evidence")
	}
}
