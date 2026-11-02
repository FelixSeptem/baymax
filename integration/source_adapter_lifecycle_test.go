package integration

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/host/eventadapter"
	"github.com/FelixSeptem/baymax/host/sourceadapter"
)

type lifecyclePoll func(context.Context, *sourceadapter.Cursor, int) (sourceadapter.Page, error)

func (f lifecyclePoll) Poll(ctx context.Context, cursor *sourceadapter.Cursor, limit int) (sourceadapter.Page, error) {
	return f(ctx, cursor, limit)
}

type lifecycleConsume func(context.Context, sourceadapter.Fact) (sourceadapter.Disposition, error)

func (f lifecycleConsume) Consume(ctx context.Context, fact sourceadapter.Fact) (sourceadapter.Disposition, error) {
	return f(ctx, fact)
}

type lifecycleUnconfirmedStore struct{ sourceadapter.Store }

func (s lifecycleUnconfirmedStore) CompareAndSwap(ctx context.Context, revision uint64, next sourceadapter.State) (sourceadapter.State, error) {
	if next.Pending == nil {
		return sourceadapter.State{}, errors.New("injected confirmation failure")
	}
	return s.Store.CompareAndSwap(ctx, revision, next)
}

func TestSourceAdapterIntegrationObserverCannotConfirmHostEffect(t *testing.T) {
	ctx := context.Background()
	metadata := sourceadapter.Metadata{Source: sourceadapter.SourceIdentity{Kind: "offline", Namespace: "host", ID: "inbox"}, IdentityNamespace: "messages", CursorVersion: "v1"}
	path := filepath.Join(t.TempDir(), "source-state.json")
	store, err := sourceadapter.OpenFileStore(path, metadata)
	if err != nil {
		t.Fatal(err)
	}
	observer := nativeObserver(t, types.ProtocolSourceRunner, "native-run", eventadapter.SinkFunc(func(context.Context, types.NativeExecutionEvent) error { return nil }))
	for _, capability := range []any{any(observer)} {
		if _, ok := capability.(sourceadapter.Source); ok {
			t.Fatal("Observer unexpectedly owns polling")
		}
		if _, ok := capability.(sourceadapter.Store); ok {
			t.Fatal("Observer unexpectedly owns durable cursor")
		}
	}
	engine := nativeEngine(t, func(context.Context, types.ModelRequest) (types.ModelResponse, error) {
		return types.ModelResponse{FinalAnswer: "observed"}, nil
	}, nil)
	if _, err := nativeExecute(ctx, engine, false, observer); err != nil {
		t.Fatal(err)
	}
	if observer.Status().TerminalDelivered != 1 {
		t.Fatal("native terminal was not observed")
	}
	state, err := store.Load(ctx)
	if err != nil || state.Confirmed != nil || state.Revision != 0 {
		t.Fatalf("observation advanced source state: %+v %v", state, err)
	}
	observer.Close()
	observer.Close()
	body := []byte("offline source fact")
	fact := sourceadapter.Fact{ExternalID: "message-1", Reference: "message/ref-1", Payload: body, PayloadDigest: fmt.Sprintf("sha256:%x", sha256.Sum256(body)), Next: sourceadapter.Cursor{Metadata: metadata, Token: "cursor-1"}}
	polls, deliveries, effects := 0, 0, 0
	committed := make(map[string]bool)
	source := lifecyclePoll(func(_ context.Context, cursor *sourceadapter.Cursor, _ int) (sourceadapter.Page, error) {
		polls++
		if cursor != nil {
			t.Fatal("unconfirmed cursor escaped to polling")
		}
		return sourceadapter.Page{Metadata: metadata, Facts: []sourceadapter.Fact{fact}}, nil
	})
	consumer := lifecycleConsume(func(_ context.Context, f sourceadapter.Fact) (sourceadapter.Disposition, error) {
		deliveries++
		// This host-owned ledger stands in for the business transaction's CAS.
		if !committed[f.ExternalID] {
			committed[f.ExternalID] = true
			effects++
		}
		return sourceadapter.Sent, nil
	})
	first, err := sourceadapter.New(metadata, source, consumer, lifecycleUnconfirmedStore{store}, sourceadapter.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer first.Dispose()
	if err := first.RunOnce(ctx); !errors.Is(err, sourceadapter.ErrDependency) {
		t.Fatalf("confirmation failure=%v", err)
	}
	first.Dispose()
	reopened, err := sourceadapter.OpenFileStore(path, metadata)
	if err != nil {
		t.Fatal(err)
	}
	state, err = reopened.Load(ctx)
	if err != nil || state.Confirmed != nil || state.Pending == nil || state.Pending.ExternalID != fact.ExternalID {
		t.Fatalf("lost unconfirmed source evidence: %+v %v", state, err)
	}
	second, err := sourceadapter.New(metadata, source, consumer, reopened, sourceadapter.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer second.Dispose()
	if err := second.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	state, err = reopened.Load(ctx)
	if err != nil || state.Confirmed == nil || state.Confirmed.Token != "cursor-1" || state.Pending != nil || effects != 1 || deliveries != 2 || polls != 2 {
		t.Fatalf("retry did not respect host CAS/source cursor: %+v effects=%d deliveries=%d polls=%d err=%v", state, effects, deliveries, polls, err)
	}
}
