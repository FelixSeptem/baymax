# orchestration/mailbox

`orchestration/mailbox` owns the canonical mailbox envelope and its delivery lifecycle: publish, claim/consume, acknowledgement, nack/requeue, delayed delivery, lease reclaim, expiry and DLQ. It is not an application outbox, business-effect database, scheduler, worker pool, authorization service, terminal reducer or diagnostics writer. Mailbox diagnostics continue to use the existing configuration/recorder path; new diagnostics must respect the repository's single-writer constraint.

## Ownership and recovery boundary

Mailbox delivery is at-least-once. A publish acceptance or `Ack` only describes mailbox lifecycle; neither proves a host business effect committed nor confirms a host-owned application outbox record. The host keeps business authorization, effect key/idempotency/CAS and final outbox confirmation. Delivery retry/requeue/lease/DLQ remain mailbox-owned and must not be compounded with scheduler execution retry or host business-commit retry.

## Application outbox dispatch proposal

`establish-application-outbox-mailbox-recovery-contract` is an active proposal, not a delivered mailbox API. It plans a separate constructor-only, reference-only bridge that may delegate a bounded descriptor to this existing owner or to scheduler. It must not introduce a parallel mailbox store, worker, lease/retry/DLQ state, acknowledgement rule or terminal state. Before implementation, no bridge type, example, replay fixture or dedicated gate exists.

Any future bridge must retain distinct facts for dispatch accepted, mailbox lifecycle, host effect evidence and host outbox confirmation. It must reject invalid/unknown/stale/non-equivalent duplicate requests before mailbox mutation; exact duplicates must converge through source-owned identity. Rollback stops new bridge calls and preserves pending message/outbox/effect facts for host reconciliation.

## Verification

Existing mailbox behavior is covered by the mailbox lifecycle and integration suites listed in `docs/mainline-contract-test-index.md`. Future bridge-specific tests and gates must be added before the proposal can be implemented or archived.
