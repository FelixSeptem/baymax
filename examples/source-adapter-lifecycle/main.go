// Source adapter lifecycle is an offline integration example. Host scheduling
// and effect CAS stay explicit; no worker, network or model is started.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/FelixSeptem/baymax/host/sourceadapter"
)

type offlineSource struct {
	metadata sourceadapter.Metadata
	facts    []sourceadapter.Fact
}

func (s offlineSource) Poll(_ context.Context, after *sourceadapter.Cursor, _ int) (sourceadapter.Page, error) {
	start := 0
	if after != nil {
		for i, f := range s.facts {
			if f.Next == *after {
				start = i + 1
				break
			}
		}
	}
	return sourceadapter.Page{Metadata: s.metadata, After: after, Facts: s.facts[start:]}, nil
}

type consumerFunc func(context.Context, sourceadapter.Fact) (sourceadapter.Disposition, error)

func (f consumerFunc) Consume(ctx context.Context, fact sourceadapter.Fact) (sourceadapter.Disposition, error) {
	return f(ctx, fact)
}

func run() ([]string, error) {
	ctx := context.Background()
	m := sourceadapter.Metadata{Source: sourceadapter.SourceIdentity{Kind: "offline", Namespace: "example", ID: "inbox"}, IdentityNamespace: "message", CursorVersion: "v1"}
	facts := make([]sourceadapter.Fact, 0, 3)
	for _, id := range []string{"one", "gone", "retry"} {
		payload := []byte("offline-" + id)
		sum := sha256.Sum256(payload)
		facts = append(facts, sourceadapter.Fact{ExternalID: id, Reference: "message/" + id, Payload: payload, PayloadDigest: "sha256:" + hex.EncodeToString(sum[:]), Next: sourceadapter.Cursor{Metadata: m, Token: id}})
	}
	source := offlineSource{metadata: m, facts: facts}
	store, err := sourceadapter.NewMemoryStore(m)
	if err != nil {
		return nil, err
	}
	effects := map[string]bool{} // Host-owned effect CAS, not adapter state.
	effectCount := 0
	retry := true
	consume := consumerFunc(func(_ context.Context, f sourceadapter.Fact) (sourceadapter.Disposition, error) {
		if f.ExternalID == "gone" {
			return sourceadapter.Gone, nil
		}
		if f.ExternalID == "retry" && retry {
			return sourceadapter.Retryable, nil
		}
		if !effects[f.ExternalID] {
			effects[f.ExternalID] = true
			effectCount++
		}
		return sourceadapter.Sent, nil
	})
	a, err := sourceadapter.New(m, source, consume, store, sourceadapter.Options{})
	if err != nil {
		return nil, err
	}
	defer a.Dispose()
	if err := a.RunOnce(ctx); !errors.Is(err, sourceadapter.ErrRetryable) {
		return nil, fmt.Errorf("expected retryable: %w", err)
	}
	s, err := store.Load(ctx)
	if err != nil {
		return nil, err
	}
	if s.Confirmed == nil || s.Confirmed.Token != "gone" || s.Pending == nil || s.Pending.ExternalID != "retry" || effectCount != 1 {
		return nil, errors.New("confirmation prefix drift")
	}
	markers := []string{"sent_confirmed", "gone_confirmed", "retryable_preserved"}
	retry = false
	if err := a.RunOnce(ctx); err != nil {
		return nil, err
	}
	s, err = store.Load(ctx)
	if err != nil {
		return nil, err
	}
	if err := a.Offer(ctx, sourceadapter.Page{Metadata: m, After: s.Confirmed, Facts: facts[:1]}); err != nil {
		return nil, err
	}
	s, err = store.Load(ctx)
	if err != nil {
		return nil, err
	}
	if effectCount != 2 || s.Confirmed == nil || s.Confirmed.Token != "retry" {
		return nil, errors.New("duplicate effect or cursor regression")
	}
	markers = append(markers, "duplicate_no_effect")
	dir, err := os.MkdirTemp("", "baymax-source-adapter-")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	path := filepath.Join(dir, "state.json")
	file, err := sourceadapter.OpenFileStore(path, m)
	if err != nil {
		return nil, err
	}
	b, err := sourceadapter.New(m, source, consume, file, sourceadapter.Options{})
	if err != nil {
		return nil, err
	}
	err = b.Offer(ctx, sourceadapter.Page{Metadata: m, Facts: facts[:1]})
	b.Dispose()
	if err != nil {
		return nil, err
	}
	reopened, err := sourceadapter.OpenFileStore(path, m)
	if err != nil {
		return nil, err
	}
	s, err = reopened.Load(ctx)
	if err != nil {
		return nil, err
	}
	if s.Confirmed == nil || s.Confirmed.Token != "one" {
		return nil, errors.New("reopen lost cursor")
	}
	markers = append(markers, "restart_confirmed_cursor")
	abortStore, err := sourceadapter.NewMemoryStore(m)
	if err != nil {
		return nil, err
	}
	var aborted *sourceadapter.Adapter
	aborted, err = sourceadapter.New(m, source, consumerFunc(func(ctx context.Context, _ sourceadapter.Fact) (sourceadapter.Disposition, error) {
		aborted.Abort()
		<-ctx.Done()
		return sourceadapter.Sent, ctx.Err()
	}), abortStore, sourceadapter.Options{})
	if err != nil {
		return nil, err
	}
	defer aborted.Dispose()
	if err := aborted.Offer(ctx, sourceadapter.Page{Metadata: m, Facts: facts[:1]}); !errors.Is(err, context.Canceled) {
		return nil, errors.New("abort did not cancel delivery")
	}
	s, err = abortStore.Load(ctx)
	if err != nil {
		return nil, err
	}
	if s.Confirmed != nil || s.Pending == nil {
		return nil, errors.New("abort lost pending or advanced cursor")
	}
	return append(markers, "abort_preserved"), nil
}

func main() {
	markers, err := run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, marker := range markers {
		fmt.Println(marker)
	}
}
