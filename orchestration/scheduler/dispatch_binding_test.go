package scheduler

import (
	"context"
	"errors"
	"testing"
)

func TestDispatchBindingRejectsMismatchWithoutMutation(t *testing.T) {
	ctx := context.Background()
	s, err := New(NewMemoryStore())
	if err != nil {
		t.Fatal(err)
	}
	task := Task{TaskID: "bound", DispatchBinding: "first"}
	if _, err = s.Enqueue(ctx, task); err != nil {
		t.Fatal(err)
	}
	task.DispatchBinding = "second"
	if _, err = s.Enqueue(ctx, task); !errors.Is(err, ErrDispatchBindingMismatch) {
		t.Fatalf("expected binding mismatch: %v", err)
	}
	task.DispatchBinding = "first"
	got, err := s.Enqueue(ctx, task)
	if err != nil || got.Task.DispatchBinding != "first" {
		t.Fatalf("duplicate: %+v %v", got, err)
	}
	stats, _ := s.Stats(ctx)
	if stats.QueueTotal != 1 {
		t.Fatal("duplicate created another task")
	}
	task.DispatchBinding = ""
	if _, err = s.Enqueue(ctx, task); !errors.Is(err, ErrDispatchBindingMismatch) {
		t.Fatal("unbound caller replaced bound identity")
	}
}
