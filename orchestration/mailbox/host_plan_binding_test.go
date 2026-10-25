package mailbox

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

// blockingSnapshotStore exposes the stale snapshot/restore interleaving that
// continuation mutations must not permit against the mailbox lifecycle owner.
type blockingSnapshotStore struct {
	Store
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func (s *blockingSnapshotStore) Snapshot(ctx context.Context) (Snapshot, error) {
	snapshot, err := s.Store.Snapshot(ctx)
	s.once.Do(func() {
		close(s.entered)
		<-s.release
	})
	return snapshot, err
}

func (s *blockingSnapshotStore) continuationSnapshot(ctx context.Context, fn func(Snapshot) error) error {
	return s.Store.(continuationStore).continuationSnapshot(ctx, fn)
}

func (s *blockingSnapshotStore) continuationMutate(ctx context.Context, fn func(*mailboxState) error) error {
	s.once.Do(func() {
		close(s.entered)
		<-s.release
	})
	return s.Store.(continuationStore).continuationMutate(ctx, fn)
}

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

func TestMailboxContinuationReopensAndRejectsLeaseLossWithoutRepublish(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "mailbox.json")
	store, err := NewFileStore(path, Policy{MaxAttempts: 2, DLQEnabled: true})
	if err != nil {
		t.Fatal(err)
	}
	mb, err := New(store)
	if err != nil {
		t.Fatal(err)
	}
	binding := HostPlanBinding{ContractVersion: HostPlanBindingContractVersion, MessageID: "continuation-message", IdempotencyKey: "continuation-logical", RunID: "run", ParentRunID: "parent", CausationID: "cause", AuthorityDigest: "authority", PlanDigest: "plan", WriteScope: "scope", EffectNamespace: "effects", Occurrence: "occurrence", HostCorrelation: "host", ResourceLimit: 1}
	if _, err := mb.AdmitHostPlan(ctx, binding); err != nil {
		t.Fatal(err)
	}
	claimed, ok, err := mb.ConsumeWithLease(ctx, "worker", time.Minute, false)
	if err != nil || !ok {
		t.Fatalf("consume: ok=%v err=%v", ok, err)
	}
	now := time.Now().UTC()
	continuation, err := SealMailboxContinuation(MailboxContinuation{Ref: MailboxContinuationRef{OwnerNamespace: "mailbox"}, Binding: binding, MessageID: binding.MessageID, DeliveryID: claimed.DeliveryID, LeaseID: claimed.LeaseID, AttemptCausation: claimed.AttemptCausation, DeliveryAttempt: claimed.DeliveryAttempt, LifecycleState: claimed.State, ParentRunID: binding.ParentRunID, RunID: binding.RunID, HostCorrelation: binding.HostCorrelation, AuthorityDigest: binding.AuthorityDigest, PlanDigest: binding.PlanDigest, WriteScope: binding.WriteScope, EffectNamespace: binding.EffectNamespace, Occurrence: binding.Occurrence, PauseReason: MailboxPauseInputRequired, Generation: 1, CreatedAt: now, ExpiresAt: now.Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if err := mb.SaveHostPlanContinuation(ctx, continuation); err != nil {
		t.Fatal(err)
	}
	reopenedStore, err := NewFileStore(path, Policy{})
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := New(reopenedStore)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := reopened.LoadHostPlanContinuation(ctx, continuation.Ref)
	if err != nil || loaded.Ref != continuation.Ref {
		t.Fatalf("load: %#v err=%v", loaded, err)
	}
	req := MailboxResumeRequest{Ref: continuation.Ref, Binding: binding, IdempotencyKey: "resume", Generation: 1, PauseReason: MailboxPauseInputRequired, Deployment: types.DeploymentRequest{OwnerNamespace: "mailbox", WriterFence: "fence:mailbox"}}
	claim, err := reopened.ClaimHostPlanResume(ctx, req)
	if err != nil || claim.Duplicate {
		t.Fatalf("claim: %#v err=%v", claim, err)
	}
	dup, err := reopened.ClaimHostPlanResume(ctx, req)
	if !errors.Is(err, ErrMailboxContinuationInProgress) || !dup.Duplicate || dup.AttemptID != claim.AttemptID {
		t.Fatalf("duplicate claim: %#v err=%v", dup, err)
	}
	if _, err := reopened.Requeue(ctx, binding.MessageID, "worker", LifecycleReasonLeaseExpired); err != nil {
		t.Fatal(err)
	}
	if _, err := reopened.ClaimHostPlanResume(ctx, req); !errors.Is(err, ErrMailboxContinuationLeaseLost) {
		t.Fatalf("lease loss error = %v", err)
	}
}

func TestMailboxContinuationFreshProcessDuplicateResumeReturnsRetainedProjection(t *testing.T) {
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
	binding := HostPlanBinding{ContractVersion: HostPlanBindingContractVersion, MessageID: "duplicate-projection-message", IdempotencyKey: "duplicate-projection-logical", RunID: "run", ParentRunID: "parent", CausationID: "cause", AuthorityDigest: "authority", PlanDigest: "plan", WriteScope: "scope", EffectNamespace: "effects", Occurrence: "occurrence", HostCorrelation: "host", ResourceLimit: 1}
	if _, err := mb.AdmitHostPlan(ctx, binding); err != nil {
		t.Fatal(err)
	}
	claimed, ok, err := mb.ConsumeWithLease(ctx, "worker", time.Hour, false)
	if err != nil || !ok {
		t.Fatalf("consume: ok=%v err=%v", ok, err)
	}
	now := time.Now().UTC()
	continuation, err := SealMailboxContinuation(MailboxContinuation{Ref: MailboxContinuationRef{OwnerNamespace: "mailbox"}, Binding: binding, MessageID: binding.MessageID, DeliveryID: claimed.DeliveryID, LeaseID: claimed.LeaseID, AttemptCausation: claimed.AttemptCausation, DeliveryAttempt: claimed.DeliveryAttempt, LifecycleState: claimed.State, ParentRunID: binding.ParentRunID, RunID: binding.RunID, HostCorrelation: binding.HostCorrelation, AuthorityDigest: binding.AuthorityDigest, PlanDigest: binding.PlanDigest, WriteScope: binding.WriteScope, EffectNamespace: binding.EffectNamespace, Occurrence: binding.Occurrence, PauseReason: MailboxPauseInputRequired, Generation: 1, CreatedAt: now, ExpiresAt: now.Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if err := mb.SaveHostPlanContinuation(ctx, continuation); err != nil {
		t.Fatal(err)
	}
	req := MailboxResumeRequest{Ref: continuation.Ref, Binding: binding, IdempotencyKey: "resume", Generation: 1, PauseReason: MailboxPauseInputRequired, Deployment: types.DeploymentRequest{OwnerNamespace: "mailbox", WriterFence: "fence:mailbox"}}
	first, err := mb.ResumeHostPlanContinuation(ctx, req)
	if err != nil {
		t.Fatalf("first resume: %v", err)
	}
	reopenedStore, err := NewFileStore(path, Policy{})
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := New(reopenedStore)
	if err != nil {
		t.Fatal(err)
	}
	duplicate, err := reopened.ResumeHostPlanContinuation(ctx, req)
	if err != nil {
		t.Fatalf("duplicate resume after reopen: %v", err)
	}
	if duplicate != first {
		t.Fatalf("duplicate projection = %#v, want %#v", duplicate, first)
	}
	page, err := reopened.Query(ctx, QueryRequest{MessageID: binding.MessageID})
	if err != nil || len(page.Items) != 1 || page.Items[0].State != StateInFlight || page.Items[0].DeliveryAttempt != 1 {
		t.Fatalf("resume mutated lifecycle: %#v err=%v", page.Items, err)
	}
}

func TestMailboxContinuationRejectsExpiredOriginalDeadlineWithoutLifecycleMutation(t *testing.T) {
	ctx := context.Background()
	mb := newTestMailbox(t, Policy{})
	binding := HostPlanBinding{ContractVersion: HostPlanBindingContractVersion, MessageID: "expired-deadline-message", IdempotencyKey: "expired-deadline-logical", RunID: "run", ParentRunID: "parent", CausationID: "cause", AuthorityDigest: "authority", PlanDigest: "plan", WriteScope: "scope", EffectNamespace: "effects", Occurrence: "occurrence", HostCorrelation: "host", ResourceLimit: 1}
	if _, err := mb.AdmitHostPlan(ctx, binding); err != nil {
		t.Fatal(err)
	}
	claimed, ok, err := mb.ConsumeWithLease(ctx, "worker", time.Hour, false)
	if err != nil || !ok {
		t.Fatalf("consume: ok=%v err=%v", ok, err)
	}
	now := time.Now().UTC()
	continuation, err := SealMailboxContinuation(MailboxContinuation{Ref: MailboxContinuationRef{OwnerNamespace: "mailbox"}, Binding: binding, MessageID: binding.MessageID, DeliveryID: claimed.DeliveryID, LeaseID: claimed.LeaseID, AttemptCausation: claimed.AttemptCausation, DeliveryAttempt: claimed.DeliveryAttempt, LifecycleState: claimed.State, ParentRunID: binding.ParentRunID, RunID: binding.RunID, HostCorrelation: binding.HostCorrelation, AuthorityDigest: binding.AuthorityDigest, PlanDigest: binding.PlanDigest, WriteScope: binding.WriteScope, EffectNamespace: binding.EffectNamespace, Occurrence: binding.Occurrence, PauseReason: MailboxPauseBudgetExhausted, OriginalDeadline: now.Add(-time.Minute), Generation: 1, CreatedAt: now.Add(-2 * time.Minute), ExpiresAt: now.Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if err := mb.SaveHostPlanContinuation(ctx, continuation); err != nil {
		t.Fatal(err)
	}
	req := MailboxResumeRequest{Ref: continuation.Ref, Binding: binding, IdempotencyKey: "resume", Generation: 1, PauseReason: MailboxPauseBudgetExhausted, Deployment: types.DeploymentRequest{OwnerNamespace: "mailbox", WriterFence: "fence:mailbox"}}
	if _, err := mb.ResumeHostPlanContinuation(ctx, req); !errors.Is(err, ErrMailboxContinuationRecoveryRequired) {
		t.Fatalf("ResumeHostPlanContinuation() error = %v, want recovery required", err)
	}
	page, err := mb.Query(ctx, QueryRequest{MessageID: binding.MessageID})
	if err != nil || len(page.Items) != 1 || page.Items[0].State != StateInFlight || page.Items[0].DeliveryAttempt != 1 {
		t.Fatalf("expired continuation mutated lifecycle: %#v err=%v", page.Items, err)
	}
}

func TestSaveHostPlanContinuationDoesNotOverwriteConcurrentAck(t *testing.T) {
	ctx := context.Background()
	base := NewMemoryStore(Policy{})
	store := &blockingSnapshotStore{Store: base, entered: make(chan struct{}), release: make(chan struct{})}
	mb, err := New(store)
	if err != nil {
		t.Fatal(err)
	}
	binding := HostPlanBinding{ContractVersion: HostPlanBindingContractVersion, MessageID: "atomic-continuation", IdempotencyKey: "atomic-logical", RunID: "run", ParentRunID: "parent", CausationID: "cause", AuthorityDigest: "authority", PlanDigest: "plan", WriteScope: "scope", EffectNamespace: "effects", Occurrence: "occurrence", HostCorrelation: "host", ResourceLimit: 1}
	if _, err := mb.AdmitHostPlan(ctx, binding); err != nil {
		t.Fatal(err)
	}
	claimed, ok, err := mb.Consume(ctx, "worker")
	if err != nil || !ok {
		t.Fatalf("consume: ok=%v err=%v", ok, err)
	}
	now := time.Now().UTC()
	continuation, err := SealMailboxContinuation(MailboxContinuation{Ref: MailboxContinuationRef{OwnerNamespace: "mailbox"}, Binding: binding, MessageID: binding.MessageID, DeliveryID: claimed.DeliveryID, LeaseID: claimed.LeaseID, AttemptCausation: claimed.AttemptCausation, DeliveryAttempt: claimed.DeliveryAttempt, LifecycleState: claimed.State, ParentRunID: binding.ParentRunID, RunID: binding.RunID, HostCorrelation: binding.HostCorrelation, AuthorityDigest: binding.AuthorityDigest, PlanDigest: binding.PlanDigest, WriteScope: binding.WriteScope, EffectNamespace: binding.EffectNamespace, Occurrence: binding.Occurrence, PauseReason: MailboxPauseInputRequired, Generation: 1, CreatedAt: now, ExpiresAt: now.Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	saved := make(chan error, 1)
	go func() { saved <- mb.SaveHostPlanContinuation(ctx, continuation) }()
	select {
	case <-store.entered:
	case <-time.After(time.Second):
		t.Fatal("continuation save did not read its owner state")
	}
	if _, err := mb.Ack(ctx, binding.MessageID, "worker"); err != nil {
		t.Fatal(err)
	}
	close(store.release)
	if err := <-saved; !errors.Is(err, ErrMailboxContinuationLeaseLost) {
		t.Fatalf("stale continuation save error = %v, want ErrMailboxContinuationLeaseLost", err)
	}
	snapshot, err := mb.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Records) != 1 || snapshot.Records[0].State != StateAcked {
		t.Fatalf("continuation write overwrote acked lifecycle state: %#v", snapshot.Records)
	}
	if len(snapshot.Continuations) != 0 {
		t.Fatalf("stale continuation was persisted after ack: %#v", snapshot.Continuations)
	}
}

func TestMailboxPauseReasonsReopenAndClaimSameDelivery(t *testing.T) {
	for _, reason := range []MailboxPauseReason{MailboxPauseInputRequired, MailboxPauseApprovalRequired, MailboxPauseBudgetExhausted} {
		t.Run(string(reason), func(t *testing.T) {
			ctx := context.Background()
			path := filepath.Join(t.TempDir(), string(reason)+".json")
			store, err := NewFileStore(path, Policy{})
			if err != nil {
				t.Fatal(err)
			}
			mb, err := New(store)
			if err != nil {
				t.Fatal(err)
			}
			binding := HostPlanBinding{ContractVersion: HostPlanBindingContractVersion, MessageID: "reason-" + string(reason), IdempotencyKey: "logical-" + string(reason), RunID: "run", ParentRunID: "parent", CausationID: "cause", AuthorityDigest: "authority", PlanDigest: "plan", WriteScope: "scope", EffectNamespace: "effects", Occurrence: "occurrence", HostCorrelation: "host", ResourceLimit: 1}
			if _, err := mb.AdmitHostPlan(ctx, binding); err != nil {
				t.Fatal(err)
			}
			claimed, ok, err := mb.ConsumeWithLease(ctx, "worker", time.Hour, false)
			if err != nil || !ok {
				t.Fatalf("consume: ok=%v err=%v", ok, err)
			}
			now := time.Now().UTC()
			continuation, err := SealMailboxContinuation(MailboxContinuation{Ref: MailboxContinuationRef{OwnerNamespace: "mailbox"}, Binding: binding, MessageID: binding.MessageID, DeliveryID: claimed.DeliveryID, LeaseID: claimed.LeaseID, AttemptCausation: claimed.AttemptCausation, DeliveryAttempt: claimed.DeliveryAttempt, LifecycleState: claimed.State, ParentRunID: binding.ParentRunID, RunID: binding.RunID, HostCorrelation: binding.HostCorrelation, AuthorityDigest: binding.AuthorityDigest, PlanDigest: binding.PlanDigest, WriteScope: binding.WriteScope, EffectNamespace: binding.EffectNamespace, Occurrence: binding.Occurrence, PauseReason: reason, PendingActionReference: string(reason), ConsumedBudget: 3, OriginalDeadline: now.Add(time.Hour), Generation: 1, CreatedAt: now, ExpiresAt: now.Add(time.Hour)})
			if err != nil {
				t.Fatal(err)
			}
			if err := mb.SaveHostPlanContinuation(ctx, continuation); err != nil {
				t.Fatal(err)
			}
			reopenedStore, err := NewFileStore(path, Policy{})
			if err != nil {
				t.Fatal(err)
			}
			reopened, err := New(reopenedStore)
			if err != nil {
				t.Fatal(err)
			}
			req := MailboxResumeRequest{Ref: continuation.Ref, Binding: binding, IdempotencyKey: "resume", Generation: 1, PauseReason: reason, Deployment: types.DeploymentRequest{OwnerNamespace: "mailbox", WriterFence: "fence:mailbox"}}
			claim, err := reopened.ClaimHostPlanResume(ctx, req)
			if err != nil || claim.Duplicate || claim.Request.PauseReason != reason {
				t.Fatalf("claim=%#v err=%v", claim, err)
			}
		})
	}
}
