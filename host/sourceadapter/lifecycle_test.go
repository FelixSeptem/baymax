package sourceadapter_test

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/host/sourceadapter"
)

func TestAbortAndConcurrentAdmission(t *testing.T) {
	s := memory(t)
	entered := make(chan struct{})
	a := newAdapter(t, s, func(ctx context.Context, _ sourceadapter.Fact) (sourceadapter.Disposition, error) {
		close(entered)
		<-ctx.Done()
		return sourceadapter.Sent, ctx.Err()
	})
	done := make(chan error, 1)
	go func() {
		done <- a.Offer(context.Background(), sourceadapter.Page{Metadata: metadata(), Facts: []sourceadapter.Fact{fact("one", "1", "body")}})
	}()
	<-entered
	if err := a.RunOnce(context.Background()); !errors.Is(err, sourceadapter.ErrBusy) {
		t.Fatalf("concurrent admission=%v", err)
	}
	a.Abort()
	a.Abort()
	a.Dispose()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("abort=%v", err)
	}
	if err := a.RunOnce(context.Background()); !errors.Is(err, sourceadapter.ErrClosed) {
		t.Fatalf("closed=%v", err)
	}
	if load(t, s).Confirmed != nil || a.Health().Class != sourceadapter.Closed {
		t.Fatal("abort confirmed work or lost closed state")
	}
}

func TestOperationTimeoutPreservesPending(t *testing.T) {
	s := memory(t)
	a, err := sourceadapter.New(metadata(), sourceFunc(func(context.Context, *sourceadapter.Cursor, int) (sourceadapter.Page, error) {
		return sourceadapter.Page{Metadata: metadata()}, nil
	}), consumerFunc(func(ctx context.Context, _ sourceadapter.Fact) (sourceadapter.Disposition, error) {
		<-ctx.Done()
		return sourceadapter.Sent, nil
	}), s, sourceadapter.Options{OperationTimeout: time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	defer a.Dispose()
	err = a.Offer(context.Background(), sourceadapter.Page{Metadata: metadata(), Facts: []sourceadapter.Fact{fact("one", "1", "body")}})
	if !errors.Is(err, context.DeadlineExceeded) || load(t, s).Confirmed != nil || load(t, s).Pending == nil {
		t.Fatalf("timeout=%v", err)
	}
	if a.Health().Class != sourceadapter.Interrupted {
		t.Fatal(a.Health())
	}
}

type failingConfirmStore struct {
	sourceadapter.Store
	fail bool
}

func (s *failingConfirmStore) CompareAndSwap(ctx context.Context, expected uint64, next sourceadapter.State) (sourceadapter.State, error) {
	if next.Pending == nil && s.fail {
		return sourceadapter.State{}, errors.New("private failure payload")
	}
	return s.Store.CompareAndSwap(ctx, expected, next)
}

func TestHostEffectCASAfterUnconfirmedEffect(t *testing.T) {
	store := &failingConfirmStore{Store: memory(t), fail: true}
	effects := make(map[string]bool)
	effectCount, deliveries := 0, 0
	consume := consumerFunc(func(_ context.Context, f sourceadapter.Fact) (sourceadapter.Disposition, error) {
		deliveries++
		if !effects[f.ExternalID] {
			effects[f.ExternalID] = true
			effectCount++
		}
		return sourceadapter.Sent, nil
	})
	a := newAdapter(t, store, consume)
	p := sourceadapter.Page{Metadata: metadata(), Facts: []sourceadapter.Fact{fact("one", "1", "body")}}
	if err := a.Offer(context.Background(), p); !errors.Is(err, sourceadapter.ErrDependency) || strings.Contains(err.Error(), "private") {
		t.Fatalf("store failure=%v", err)
	}
	if load(t, store).Confirmed != nil || load(t, store).Pending == nil {
		t.Fatal("lost pending evidence")
	}
	a.Dispose()
	store.fail = false
	b := newAdapter(t, store, consume)
	if err := b.Offer(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if effectCount != 1 || deliveries != 2 || load(t, store).Confirmed.Token != "1" {
		t.Fatalf("effects=%d deliveries=%d", effectCount, deliveries)
	}
}

func TestBoundsAdmissionAndDetachedState(t *testing.T) {
	s := memory(t)
	consumer := consumerFunc(func(context.Context, sourceadapter.Fact) (sourceadapter.Disposition, error) {
		return sourceadapter.Sent, nil
	})
	source := sourceFunc(func(context.Context, *sourceadapter.Cursor, int) (sourceadapter.Page, error) {
		return sourceadapter.Page{}, nil
	})
	for _, options := range []sourceadapter.Options{{PageLimit: -1}, {PageLimit: 201}, {OperationTimeout: -1}} {
		if _, err := sourceadapter.New(metadata(), source, consumer, s, options); !errors.Is(err, sourceadapter.ErrInvalid) {
			t.Fatal("invalid options admitted")
		}
	}
	var absent *sourceadapter.MemoryStore
	if _, err := sourceadapter.New(metadata(), source, consumer, absent, sourceadapter.Options{}); err == nil {
		t.Fatal("typed nil admitted")
	}
	a := newAdapter(t, s, consumer)
	f := fact("one", "1", strings.Repeat("x", sourceadapter.MaxPayloadBytes))
	if err := a.Offer(context.Background(), sourceadapter.Page{Metadata: metadata(), Facts: []sourceadapter.Fact{f}}); err != nil {
		t.Fatal(err)
	}
	state := load(t, s)
	state.Confirmed.Token = "mutated"
	state.Recent[0].Reference = "mutated"
	if load(t, s).Confirmed.Token != "1" || load(t, s).Recent[0].Reference == "mutated" {
		t.Fatal("store snapshot aliases state")
	}
	for _, name := range []string{"payload", "id", "token", "utf8"} {
		t.Run(name, func(t *testing.T) {
			f := fact("two", "2", "body")
			switch name {
			case "payload":
				f = fact("two", "2", strings.Repeat("x", sourceadapter.MaxPayloadBytes+1))
			case "id":
				f.ExternalID = "white space"
			case "token":
				f.Next.Token = strings.Repeat("x", 4097)
			case "utf8":
				f.ExternalID = string([]byte{0xff})
			}
			if err := a.Offer(context.Background(), sourceadapter.Page{Metadata: metadata(), After: load(t, s).Confirmed, Facts: []sourceadapter.Fact{f}}); err == nil {
				t.Fatal("invalid fact admitted")
			}
		})
	}
}

func TestRecentWindowAndDuplicateNeverRegress(t *testing.T) {
	s := memory(t)
	a := newAdapter(t, s, func(context.Context, sourceadapter.Fact) (sourceadapter.Disposition, error) {
		return sourceadapter.Sent, nil
	})
	for i := range 202 {
		id := fmt.Sprint(i)
		if err := a.Offer(context.Background(), sourceadapter.Page{Metadata: metadata(), After: load(t, s).Confirmed, Facts: []sourceadapter.Fact{fact(id, id, id)}}); err != nil {
			t.Fatal(err)
		}
	}
	state := load(t, s)
	if len(state.Recent) != 200 || state.Recent[0].ExternalID != "2" {
		t.Fatal("receipt evidence unbounded")
	}
	if err := a.Offer(context.Background(), sourceadapter.Page{Metadata: metadata(), After: state.Confirmed, Facts: []sourceadapter.Fact{fact("2", "2", "2")}}); err != nil {
		t.Fatal(err)
	}
	if load(t, s).Confirmed.Token != "201" {
		t.Fatal("duplicate regressed cursor")
	}
}

func TestSourceAdapterReservesBothStateRevisionsBeforeEffect(t *testing.T) {
	s := memory(t)
	state := load(t, s)
	state.Revision = math.MaxUint64 - 1
	if _, err := sourceadapter.Prepare(state, fact("one", "1", "body")); !errors.Is(err, sourceadapter.ErrConflict) {
		t.Fatalf("exhausted revision accepted: %v", err)
	}
}

func TestSourceAdapterStoreCASHasOneWinner(t *testing.T) {
	s := memory(t)
	initial := load(t, s)
	next, err := sourceadapter.Prepare(initial, fact("one", "1", "body"))
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	for range 2 {
		go func() { <-start; _, err := s.CompareAndSwap(context.Background(), 0, next); results <- err }()
	}
	close(start)
	wins, conflicts := 0, 0
	for range 2 {
		err := <-results
		switch {
		case err == nil:
			wins++
		case errors.Is(err, sourceadapter.ErrConflict):
			conflicts++
		default:
			t.Fatal(err)
		}
	}
	if wins != 1 || conflicts != 1 {
		t.Fatalf("wins=%d conflicts=%d", wins, conflicts)
	}
}

func TestSourceAdapterInvalidDispositionAndConsumerErrors(t *testing.T) {
	for _, name := range []string{"invalid", "error"} {
		t.Run(name, func(t *testing.T) {
			s := memory(t)
			a := newAdapter(t, s, func(context.Context, sourceadapter.Fact) (sourceadapter.Disposition, error) {
				if name == "error" {
					return sourceadapter.Sent, errors.New("private consumer body")
				}
				return "unknown", nil
			})
			err := a.Offer(context.Background(), sourceadapter.Page{Metadata: metadata(), Facts: []sourceadapter.Fact{fact("one", "1", "body")}})
			switch {
			case err == nil, strings.Contains(err.Error(), "private"), load(t, s).Confirmed != nil, load(t, s).Pending == nil:
				t.Fatalf("bad disposition handling: %v", err)
			}
		})
	}
}
