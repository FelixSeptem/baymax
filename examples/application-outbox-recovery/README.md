# Application outbox recovery

> Status: offline executable example for `application_outbox_dispatch.v1`. It uses only fake in-memory native owners and an in-process admission callback.

## Semantic anchor

The `application_outbox_dispatch.v1` bridge is explicit opt-in and reference-only. It lets a host select exactly one native target per descriptor:

| Target | Existing owner retained | Bridge does not own |
| --- | --- | --- |
| `mailbox` | publish/claim/ack/nack/requeue/delay/lease/DLQ | mailbox worker, delivery retry, lease, DLQ or terminal state |
| `scheduler` | task/attempt/claim/lease/retry/terminal/DLQ | task store, execution retry, attempt lifecycle or terminal state |

The host retains application outbox storage, occurrence/cron/DST/misfire policy, authority, business revision, effect idempotency/CAS and final outbox confirmation. Dispatch accepted, target lifecycle, host effect committed and host outbox confirmed are four distinct facts. In particular, an ack, scheduler terminal, Run/Stream terminal, descriptor digest, or replay result is not a business commit.

## Runtime path exercised by the example

`fake outbox/effect store -> constructor-only bridge -> mailbox OR scheduler native owner -> host idempotent reconciliation`

The example stays offline and deterministic: fake collaborators only, no provider/network/credential/business database. It does not execute an effect from replay or use replay as runtime recovery.

## Expected markers

The example emits these markers:

- `mailbox_dispatch_accepted` and `scheduler_dispatch_accepted` show target-specific acceptance without business commit.
- `publish_before_outbox_confirmation_reconciled` covers the first crash window.
- `effect_committed_before_target_reconciliation` covers the second crash window.
- `duplicate_converged` distinguishes exact duplicate convergence from non-equivalent key reuse rejection.
- `rollback_preserves_pending_facts` documents that rollback stops new bridge use without deleting outbox/message/task/attempt/effect facts.

## Recovery and rollback notes

Recovery is at-least-once. Mailbox delivery retry, scheduler execution retry and host business-commit retry remain separate and are not auto-composed. Rollback stops constructing the bridge; the host reconciles retained facts with its effect idempotency/CAS. Run with `go run ./examples/application-outbox-recovery`; it must not contact a provider, network, credential store or business database. The executable does not call ack, complete, retry or an effect.
