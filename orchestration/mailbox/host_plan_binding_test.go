package mailbox

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestAdmitHostPlanConvergesEquivalentBindingAndRejectsConflict(t *testing.T) {
	mb := newTestMailbox(t, Policy{})
	binding := HostPlanBinding{
		ContractVersion: "mailbox_host_plan_binding.v1",
		MessageID:       "message-1",
		IdempotencyKey:  "logical-1",
		RunID:           "run-1",
		ParentRunID:     "parent-1",
		CausationID:     "cause-1",
		AuthorityDigest: "authority",
		PlanDigest:      "sha256:plan",
		WriteScope:      "scope-1",
		EffectNamespace: "effect-1",
		Occurrence:      "occurrence-1",
		HostCorrelation: "host-1",
		ResourceLimit:   1,
	}
	first, err := mb.AdmitHostPlan(context.Background(), binding)
	if err != nil {
		t.Fatalf("AdmitHostPlan(first) error = %v", err)
	}
	if first.Duplicate || first.MessageID != binding.MessageID || first.RunID != binding.RunID || first.ParentRunID != binding.ParentRunID || first.CausationID != binding.CausationID {
		t.Fatalf("first outcome = %#v", first)
	}
	second, err := mb.AdmitHostPlan(context.Background(), binding)
	if err != nil {
		t.Fatalf("AdmitHostPlan(duplicate) error = %v", err)
	}
	if !second.Duplicate || second.MessageID != binding.MessageID || second.AuthorityDigest != binding.AuthorityDigest || second.EffectNamespace != binding.EffectNamespace {
		t.Fatalf("duplicate outcome = %#v", second)
	}
	conflict := binding
	conflict.WriteScope = "scope-2"
	if _, err := mb.AdmitHostPlan(context.Background(), conflict); !errors.Is(err, ErrHostPlanBindingConflict) {
		t.Fatalf("AdmitHostPlan(conflict) error = %v, want ErrHostPlanBindingConflict", err)
	}
	snapshot, err := mb.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	if len(snapshot.Records) != 1 || snapshot.Records[0].Envelope.DispatchBinding == "" {
		t.Fatalf("conflict mutated mailbox records: %#v", snapshot.Records)
	}
	claimed, ok, err := mb.Consume(context.Background(), "consumer-1")
	if err != nil || !ok || claimed.DeliveryID == "" || claimed.LeaseID == "" {
		t.Fatalf("source delivery identity missing: ok=%v err=%v record=%#v", ok, err, claimed)
	}
	projected, err := HostPlanOutcomeFromRecord(claimed)
	if err != nil || projected.DeliveryLeaseID == "" || projected.CausationID != binding.CausationID {
		t.Fatalf("source outcome projection = %#v err=%v", projected, err)
	}
}

func TestHostPlanOutcomeProjectsLeaseLossRetryDLQAndTimeout(t *testing.T) {
	ctx := context.Background()
	clock := time.Now().UTC()
	binding := HostPlanBinding{
		ContractVersion: HostPlanBindingContractVersion,
		MessageID:       "lifecycle-message", IdempotencyKey: "lifecycle-logical", RunID: "run-lifecycle", ParentRunID: "parent-lifecycle",
		CausationID: "cause-lifecycle", AuthorityDigest: "authority", PlanDigest: "sha256:plan",
		WriteScope: "scope", EffectNamespace: "effects", Occurrence: "occurrence", HostCorrelation: "host",
		ResourceLimit: 1,
	}
	mb := newTestMailboxWithClock(t, Policy{MaxAttempts: 2, BackoffInitial: time.Millisecond, BackoffMax: time.Millisecond, DLQEnabled: true, TTL: time.Second}, func() time.Time { return clock })
	if _, err := mb.AdmitHostPlan(ctx, binding); err != nil {
		t.Fatal(err)
	}
	claimed, ok, err := mb.ConsumeWithLease(ctx, "worker", time.Millisecond, false)
	if err != nil || !ok {
		t.Fatalf("consume: ok=%v err=%v", ok, err)
	}
	out, err := HostPlanOutcomeFromRecord(claimed)
	if err != nil || out.LifecycleResult != "leased" || out.DeliveryID == "" || out.LeaseID == "" || out.AttemptCausation == "" {
		t.Fatalf("leased outcome = %#v err=%v", out, err)
	}
	clock = clock.Add(5 * time.Millisecond)
	_, ok, err = mb.ConsumeWithLease(ctx, "worker-2", time.Millisecond, true)
	if err != nil || ok {
		t.Fatalf("reclaim should only requeue: ok=%v err=%v", ok, err)
	}
	page, err := mb.Query(ctx, QueryRequest{MessageID: binding.MessageID})
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("lease-loss query: items=%d err=%v", len(page.Items), err)
	}
	out, err = HostPlanOutcomeFromRecord(page.Items[0])
	if err != nil || out.LifecycleResult != "lease_lost" || out.LifecycleReason != LifecycleReasonLeaseExpired || out.ParentRunID != binding.ParentRunID {
		t.Fatalf("lease-loss outcome = %#v err=%v", out, err)
	}
	clock = clock.Add(2 * time.Millisecond)
	retried, ok, err := mb.ConsumeWithLease(ctx, "worker-3", time.Millisecond, false)
	if err != nil || !ok {
		t.Fatalf("retry consume: ok=%v err=%v", ok, err)
	}
	if retried.DeliveryAttempt != 2 || retried.AttemptCausation == claimed.AttemptCausation {
		t.Fatalf("retry causation did not advance: first=%#v retry=%#v", claimed, retried)
	}
	dlq, err := mb.Requeue(ctx, retried.Envelope.MessageID, "worker-3", LifecycleReasonRetryExhausted)
	if err != nil {
		t.Fatal(err)
	}
	out, err = HostPlanOutcomeFromRecord(dlq)
	if err != nil || out.LifecycleResult != "dlq" || out.TerminalOutcome != "dead_letter" || out.LifecycleReason == "" {
		t.Fatalf("dlq outcome = %#v err=%v", out, err)
	}

	timeoutMB := newTestMailboxWithClock(t, Policy{MaxAttempts: 1, DLQEnabled: false, TTL: 5 * time.Millisecond}, func() time.Time { return clock })
	timeoutBinding := binding
	timeoutBinding.MessageID = "timeout-message"
	timeoutBinding.IdempotencyKey = "timeout-logical"
	if _, err := timeoutMB.AdmitHostPlan(ctx, timeoutBinding); err != nil {
		t.Fatal(err)
	}
	clock = clock.Add(10 * time.Millisecond)
	if _, ok, err := timeoutMB.Consume(ctx, "timeout-worker"); err != nil || ok {
		t.Fatalf("expired consume should not deliver: ok=%v err=%v", ok, err)
	}
	page, err = timeoutMB.Query(ctx, QueryRequest{MessageID: timeoutBinding.MessageID})
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("timeout query: items=%d err=%v", len(page.Items), err)
	}
	out, err = HostPlanOutcomeFromRecord(page.Items[0])
	if err != nil || out.LifecycleResult != "timeout" || out.TerminalOutcome != "expired" {
		t.Fatalf("timeout outcome = %#v err=%v", out, err)
	}
}
