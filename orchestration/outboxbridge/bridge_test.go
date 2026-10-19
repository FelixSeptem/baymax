package outboxbridge

import (
	"context"
	"errors"
	"testing"

	"github.com/FelixSeptem/baymax/orchestration/mailbox"
	"github.com/FelixSeptem/baymax/orchestration/scheduler"
)

func descriptor(target Target) Descriptor {
	d := Descriptor{Target: target, OutboxID: "order-1", Revision: "r1", AuthorityRef: "auth-1", EffectRef: "effect-1", EffectKey: "effect-key-1", RunID: "run-1", IdempotencyKey: "idem-1"}
	if target == TargetMailbox {
		d.Mailbox = &MailboxBinding{ToAgent: "worker"}
	} else {
		d.Scheduler = &SchedulerBinding{AgentID: "worker", MaxAttempts: 2}
	}
	return d
}
func admission(d Decision) Admission {
	return func(context.Context, Descriptor, string) (Decision, error) { return d, nil }
}

func TestDispatchMailboxAndReconstructedDuplicate(t *testing.T) {
	ctx := context.Background()
	store := mailbox.NewMemoryStore(mailbox.Policy{})
	owner, _ := mailbox.New(store)
	d := descriptor(TargetMailbox)
	_, digest, _ := Canonical(d)
	bridge, err := New(Config{Mailbox: owner, Admission: admission(Decision{Eligible: true, Digest: digest})})
	if err != nil {
		t.Fatal(err)
	}
	first, err := bridge.Dispatch(ctx, d)
	if err != nil || !first.Facts.DispatchAccepted {
		t.Fatalf("first: %+v %v", first, err)
	}
	second, err := bridge.Dispatch(ctx, d)
	if err != nil || !second.Duplicate {
		t.Fatalf("duplicate: %+v %v", second, err)
	}
	if _, err = bridge.Inspect(ctx, d); err != nil {
		t.Fatal(err)
	}
	d.Revision = "r2"
	if _, err = bridge.Dispatch(ctx, d); err == nil {
		t.Fatal("admission must reject changed digest")
	}
}

func TestDispatchSchedulerAndFailClosedAdmission(t *testing.T) {
	ctx := context.Background()
	owner, _ := scheduler.New(scheduler.NewMemoryStore())
	d := descriptor(TargetScheduler)
	_, digest, _ := Canonical(d)
	bridge, err := New(Config{Scheduler: owner, Admission: admission(Decision{Eligible: true, Digest: digest})})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = bridge.Dispatch(ctx, d); err != nil {
		t.Fatal(err)
	}
	if _, err = bridge.Inspect(ctx, d); err != nil {
		t.Fatal(err)
	}
	denied, _ := New(Config{Scheduler: owner, Admission: admission(Decision{})})
	if _, err = denied.Dispatch(ctx, d); err == nil {
		t.Fatal("denied admission mutated")
	}
	panicBridge, _ := New(Config{Scheduler: owner, Admission: func(context.Context, Descriptor, string) (Decision, error) { panic("boom") }})
	if _, err = panicBridge.Dispatch(ctx, d); err == nil {
		t.Fatal("panic must fail closed")
	}
	var be *Error
	if !errors.As(err, &be) || be.Code != ErrAdmissionFailed {
		t.Fatalf("wrong error: %v", err)
	}
}

func TestDispatchMapsNativeBindingMismatch(t *testing.T) {
	for _, target := range []Target{TargetMailbox, TargetScheduler} {
		t.Run(string(target), func(t *testing.T) {
			ctx := context.Background()
			d := descriptor(target)
			var mailboxTarget *mailbox.Mailbox
			var schedulerOwner *scheduler.Scheduler
			config := Config{Admission: func(_ context.Context, _ Descriptor, digest string) (Decision, error) {
				return Decision{Eligible: true, Digest: digest}, nil
			}}
			if target == TargetMailbox {
				mailboxTarget = mailboxOwner(t)
				config.Mailbox = mailboxTarget
			} else {
				var err error
				schedulerOwner, err = scheduler.New(scheduler.NewMemoryStore())
				if err != nil {
					t.Fatal(err)
				}
				config.Scheduler = schedulerOwner
			}
			bridge, err := New(config)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = bridge.Dispatch(ctx, d); err != nil {
				t.Fatal(err)
			}
			d.Revision = "r2"
			_, err = bridge.Dispatch(ctx, d)
			var bridgeErr *Error
			if !errors.As(err, &bridgeErr) || bridgeErr.Code != ErrBindingMismatch {
				t.Fatalf("expected %s, got %v", ErrBindingMismatch, err)
			}
			if !errors.Is(err, mailbox.ErrDispatchBindingMismatch) && !errors.Is(err, scheduler.ErrDispatchBindingMismatch) {
				t.Fatalf("native mismatch was not preserved: %v", err)
			}
		})
	}
}

func TestValidationRejectsConflictingTargetBeforeCallback(t *testing.T) {
	called := false
	b, _ := New(Config{Mailbox: mailboxOwner(t), Admission: func(context.Context, Descriptor, string) (Decision, error) { called = true; return Decision{}, nil }})
	d := descriptor(TargetMailbox)
	d.Scheduler = &SchedulerBinding{AgentID: "x"}
	if _, err := b.Dispatch(context.Background(), d); err == nil {
		t.Fatal("conflicting target accepted")
	}
	if called {
		t.Fatal("invalid descriptor called admission")
	}
}

func TestFileOwnerRetainsBindingAcrossBridgeReconstruction(t *testing.T) {
	ctx := context.Background()
	path := t.TempDir() + "/mailbox.json"
	d := descriptor(TargetMailbox)
	store, err := mailbox.NewFileStore(path, mailbox.Policy{})
	if err != nil {
		t.Fatal(err)
	}
	owner, _ := mailbox.New(store)
	_, digest, _ := Canonical(d)
	bridge, _ := New(Config{Mailbox: owner, Admission: admission(Decision{Eligible: true, Digest: digest})})
	if _, err = bridge.Dispatch(ctx, d); err != nil {
		t.Fatal(err)
	}
	reopened, err := mailbox.NewFileStore(path, mailbox.Policy{})
	if err != nil {
		t.Fatal(err)
	}
	owner2, _ := mailbox.New(reopened)
	bridge2, _ := New(Config{Mailbox: owner2, Admission: admission(Decision{Eligible: true, Digest: digest})})
	got, err := bridge2.Dispatch(ctx, d)
	if err != nil || !got.Duplicate {
		t.Fatalf("recovered duplicate: %+v %v", got, err)
	}
	d.EffectRef = "different"
	if _, err = bridge2.Dispatch(ctx, d); err == nil {
		t.Fatal("changed binding accepted after reconstruction")
	}
}

func TestReconcileKeepsCommittedEffectSeparateFromTarget(t *testing.T) {
	ctx := context.Background()
	owner := mailboxOwner(t)
	d := descriptor(TargetMailbox)
	_, digest, _ := Canonical(d)
	bridge, _ := New(Config{Mailbox: owner, Admission: admission(Decision{Eligible: true, Digest: digest, Evidence: Evidence{EffectCommitted: true, Digest: digest}})})
	if _, err := bridge.Dispatch(ctx, d); err != nil {
		t.Fatal(err)
	}
	got, err := bridge.Reconcile(ctx, d)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Facts.Effect.EffectCommitted || got.Facts.OutboxConfirmed || got.Facts.TargetState == "" {
		t.Fatalf("facts conflated: %+v", got.Facts)
	}
}

func TestReconcileDoesNotOwnMailboxRetryOrAcknowledgement(t *testing.T) {
	ctx := context.Background()
	owner := mailboxOwner(t)
	d := descriptor(TargetMailbox)
	_, digest, _ := Canonical(d)
	bridge, _ := New(Config{Mailbox: owner, Admission: admission(Decision{Eligible: true, Digest: digest})})
	dispatched, err := bridge.Dispatch(ctx, d)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok, err := owner.Consume(ctx, "worker"); err != nil || !ok {
		t.Fatalf("consume failed: ok=%v err=%v", ok, err)
	}
	before, err := owner.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := bridge.Reconcile(ctx, d); err != nil {
		t.Fatal(err)
	} else if got.Facts.TargetState != string(mailbox.StateInFlight) {
		t.Fatalf("target state=%q, want %q", got.Facts.TargetState, mailbox.StateInFlight)
	}
	after, err := owner.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(before.Records) != len(after.Records) || before.Records[0].State != after.Records[0].State || after.Records[0].Envelope.MessageID != dispatched.Facts.MessageID {
		t.Fatalf("reconcile mutated mailbox lifecycle: before=%+v after=%+v", before.Records, after.Records)
	}
}

func TestReconcileDoesNotOwnSchedulerRetryOrTerminal(t *testing.T) {
	ctx := context.Background()
	owner, err := scheduler.New(scheduler.NewMemoryStore())
	if err != nil {
		t.Fatal(err)
	}
	d := descriptor(TargetScheduler)
	_, digest, _ := Canonical(d)
	bridge, _ := New(Config{Scheduler: owner, Admission: admission(Decision{Eligible: true, Digest: digest})})
	if _, err := bridge.Dispatch(ctx, d); err != nil {
		t.Fatal(err)
	}
	claimed, ok, err := owner.Claim(ctx, "worker")
	if err != nil || !ok {
		t.Fatalf("claim failed: ok=%v err=%v", ok, err)
	}
	before, found, err := owner.Get(ctx, claimed.Record.Task.TaskID)
	if err != nil || !found {
		t.Fatalf("get before reconcile failed: found=%v err=%v", found, err)
	}
	if _, err := bridge.Reconcile(ctx, d); err != nil {
		t.Fatal(err)
	}
	after, found, err := owner.Get(ctx, claimed.Record.Task.TaskID)
	if err != nil || !found {
		t.Fatalf("get after reconcile failed: found=%v err=%v", found, err)
	}
	if after.State != before.State || after.CurrentAttempt != before.CurrentAttempt || len(after.Attempts) != len(before.Attempts) {
		t.Fatalf("reconcile mutated scheduler lifecycle: before=%+v after=%+v", before, after)
	}
}

func mailboxOwner(t *testing.T) *mailbox.Mailbox {
	t.Helper()
	m, e := mailbox.New(mailbox.NewMemoryStore(mailbox.Policy{}))
	if e != nil {
		t.Fatal(e)
	}
	return m
}
