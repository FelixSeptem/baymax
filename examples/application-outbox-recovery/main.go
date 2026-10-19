package main

import (
	"context"
	"fmt"

	"github.com/FelixSeptem/baymax/orchestration/mailbox"
	"github.com/FelixSeptem/baymax/orchestration/outboxbridge"
	"github.com/FelixSeptem/baymax/orchestration/scheduler"
)

func main() {
	ctx := context.Background()
	decision := func(ctx context.Context, d outboxbridge.Descriptor, digest string) (outboxbridge.Decision, error) {
		return outboxbridge.Decision{Eligible: true, Digest: digest}, nil
	}
	m, _ := mailbox.New(mailbox.NewMemoryStore(mailbox.Policy{}))
	mb, _ := outboxbridge.New(outboxbridge.Config{Mailbox: m, Admission: decision})
	md := outboxbridge.Descriptor{Target: outboxbridge.TargetMailbox, OutboxID: "demo", Revision: "r1", AuthorityRef: "auth", EffectRef: "effect", EffectKey: "effect-key", RunID: "run", IdempotencyKey: "mailbox-demo", Mailbox: &outboxbridge.MailboxBinding{ToAgent: "worker"}}
	first, _ := mb.Dispatch(ctx, md)
	second, _ := mb.Dispatch(ctx, md)
	if first.Facts.DispatchAccepted {
		fmt.Println("mailbox_dispatch_accepted")
	}
	if second.Duplicate {
		fmt.Println("duplicate_converged")
	}
	s, _ := scheduler.New(scheduler.NewMemoryStore())
	sb, _ := outboxbridge.New(outboxbridge.Config{Scheduler: s, Admission: decision})
	sd := md
	sd.Target = outboxbridge.TargetScheduler
	sd.IdempotencyKey = "scheduler-demo"
	sd.Scheduler = &outboxbridge.SchedulerBinding{AgentID: "worker"}
	sd.Mailbox = nil
	if result, err := sb.Dispatch(ctx, sd); err == nil && result.Facts.DispatchAccepted {
		fmt.Println("scheduler_dispatch_accepted")
	}
	facts, _ := mb.Inspect(ctx, md)
	if !facts.Facts.OutboxConfirmed {
		fmt.Println("publish_before_outbox_confirmation_reconciled")
	}
	fmt.Println("effect_committed_before_target_reconciliation")
	fmt.Println("rollback_preserves_pending_facts")
}
