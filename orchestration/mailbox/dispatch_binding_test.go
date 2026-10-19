package mailbox

import (
	"context"
	"errors"
	"testing"
)

func TestDispatchBindingRejectsMismatchWithoutMutation(t *testing.T) {
	ctx := context.Background()
	m, err := New(NewMemoryStore(Policy{}))
	if err != nil {
		t.Fatal(err)
	}
	e := Envelope{MessageID: "bound", IdempotencyKey: "key", Kind: KindCommand, DispatchBinding: "first"}
	if _, err = m.Publish(ctx, e); err != nil {
		t.Fatal(err)
	}
	before, _ := m.Stats(ctx)
	e.DispatchBinding = "second"
	if _, err = m.Publish(ctx, e); !errors.Is(err, ErrDispatchBindingMismatch) {
		t.Fatalf("expected binding mismatch, got %v", err)
	}
	after, _ := m.Stats(ctx)
	if before != after {
		t.Fatal("rejection mutated stats")
	}
	e.DispatchBinding = "first"
	got, err := m.Publish(ctx, e)
	if err != nil || !got.Duplicate {
		t.Fatalf("duplicate: %+v %v", got, err)
	}
	e.DispatchBinding = ""
	if _, err = m.Publish(ctx, e); !errors.Is(err, ErrDispatchBindingMismatch) {
		t.Fatal("unbound caller replaced bound identity")
	}
}
