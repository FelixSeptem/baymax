package runner

import "testing"

func TestReactPlanNotebookLifecycleVersionAndFreeze(t *testing.T) {
	n := newReactPlanNotebook("plan-a67", 16)
	if err := n.apply(reactPlanActionCreate, "initial", "k1"); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if n.Version != 1 || n.Status != reactPlanStatusActive || n.LastAction != reactPlanActionCreate {
		t.Fatalf("unexpected state after create: %#v", n)
	}
	if err := n.apply(reactPlanActionRevise, "tool-outcome", "k2"); err != nil {
		t.Fatalf("revise failed: %v", err)
	}
	if n.Version != 2 || n.LastAction != reactPlanActionRevise {
		t.Fatalf("unexpected state after revise: %#v", n)
	}
	if err := n.apply(reactPlanActionComplete, "final-answer", "k3"); err != nil {
		t.Fatalf("complete failed: %v", err)
	}
	if n.Version != 3 || n.Status != reactPlanStatusCompleted || n.LastAction != reactPlanActionComplete {
		t.Fatalf("unexpected state after complete: %#v", n)
	}
	if err := n.apply(reactPlanActionRevise, "should-block", "k4"); err == nil {
		t.Fatal("expected revise to fail after completed freeze")
	}
}

func TestReactPlanNotebookRecoverAndIdempotency(t *testing.T) {
	n := newReactPlanNotebook("plan-a67", 16)
	if err := n.apply(reactPlanActionCreate, "initial", "k1"); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if err := n.apply(reactPlanActionRecover, "resume", "recover-op"); err != nil {
		t.Fatalf("recover failed: %v", err)
	}
	if n.Version != 2 || n.RecoverCount != 1 || n.ChangeTotal != 2 {
		t.Fatalf("unexpected state after recover: %#v", n)
	}
	if err := n.apply(reactPlanActionRecover, "resume", "recover-op"); err != nil {
		t.Fatalf("idempotent recover should not error, got %v", err)
	}
	if n.Version != 2 || n.RecoverCount != 1 || n.ChangeTotal != 2 {
		t.Fatalf("replayed recover should not inflate counters: %#v", n)
	}
}

func TestReactPlanNotebookRecoverCanBootstrapFromPending(t *testing.T) {
	n := newReactPlanNotebook("plan-a67", 16)
	if err := n.apply(reactPlanActionRecover, "resume", "recover-bootstrap"); err != nil {
		t.Fatalf("recover bootstrap failed: %v", err)
	}
	if n.Version != 1 || n.Status != reactPlanStatusActive || n.LastAction != reactPlanActionRecover {
		t.Fatalf("unexpected state after recover bootstrap: %#v", n)
	}
}

func TestReactPlanNotebookReviseReplayIdempotency(t *testing.T) {
	n := newReactPlanNotebook("plan-a67", 16)
	if err := n.apply(reactPlanActionCreate, "initial", "k1"); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if err := n.apply(reactPlanActionRevise, "r1", "revise-op"); err != nil {
		t.Fatalf("revise failed: %v", err)
	}
	if err := n.apply(reactPlanActionRevise, "r1", "revise-op"); err != nil {
		t.Fatalf("replayed revise should not error: %v", err)
	}
	if n.Version != 2 || n.ChangeTotal != 2 || n.LastAction != reactPlanActionRevise {
		t.Fatalf("replayed revise should not inflate state: %#v", n)
	}
}

func TestReactPlanNotebookRejectsInvalidAction(t *testing.T) {
	n := newReactPlanNotebook("plan-a67", 4)
	if err := n.apply("mutate", "x", "k1"); err == nil {
		t.Fatal("expected invalid action to fail")
	}
}

func TestReactPlanNotebookHistoryBounded(t *testing.T) {
	n := newReactPlanNotebook("plan-a67", 2)
	if err := n.apply(reactPlanActionCreate, "initial", "k1"); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if err := n.apply(reactPlanActionRevise, "r1", "k2"); err != nil {
		t.Fatalf("revise1 failed: %v", err)
	}
	if err := n.apply(reactPlanActionRevise, "r2", "k3"); err != nil {
		t.Fatalf("revise2 failed: %v", err)
	}
	if len(n.History) != 2 {
		t.Fatalf("history length=%d, want 2", len(n.History))
	}
	if n.History[0].Reason != "r1" || n.History[1].Reason != "r2" {
		t.Fatalf("history tail mismatch: %#v", n.History)
	}
}
