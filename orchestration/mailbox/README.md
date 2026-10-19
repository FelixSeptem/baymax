# orchestration/mailbox

`orchestration/mailbox` owns the canonical mailbox envelope and its delivery lifecycle: publish, claim/consume, acknowledgement, nack/requeue, delayed delivery, lease reclaim, expiry and DLQ. It is not an application outbox, business-effect database, scheduler, worker pool, authorization service, terminal reducer or diagnostics writer. Mailbox diagnostics continue to use the existing configuration/recorder path; new diagnostics must respect the repository's single-writer constraint.

## Ownership and recovery boundary

Mailbox delivery is at-least-once. A publish acceptance or `Ack` only describes mailbox lifecycle; neither proves a host business effect committed nor confirms a host-owned application outbox record. The host keeps business authorization, effect key/idempotency/CAS and final outbox confirmation. Delivery retry/requeue/lease/DLQ remain mailbox-owned and must not be compounded with scheduler execution retry or host business-commit retry.

## Host execution plan mailbox binding

`Mailbox.AdmitHostPlan` is the source-owned admission seam used by the
manifest-approved `mailbox_async` planbridge path. It accepts bounded
`HostPlanBinding` references (message/idempotency identity, original run,
parent/causation,
authority/plan digests, write scope, effect namespace, occurrence and host
correlation) and returns a detached `HostPlanOutcome`. It never runs a consumer,
creates a second queue, owns a scheduler or asserts a business commit.

Equivalent full logical identities converge to one mailbox message; conflicting
authority, occurrence, parent/causation, scope or effect namespace fails closed.
`Record` retains source delivery and lease IDs plus attempt causation for each
attempt. `HostPlanOutcomeFromRecord` returns a detached projection containing
those identities, `LifecycleResult`/`LifecycleReason`, and a separate
`TerminalOutcome` after consume, ack, nack, retry, lease reclaim, timeout or
DLQ. A queued lease reclaim is explicitly `lease_lost`; it is neither a new
parent Run nor a fabricated business completion. Mailbox remains the only owner
of delivery retry, lease and DLQ semantics.

## Application outbox dispatch proposal

`establish-application-outbox-mailbox-recovery-contract` is an active proposal, not a delivered mailbox API. It plans a separate constructor-only, reference-only bridge that may delegate a bounded descriptor to this existing owner or to scheduler. It must not introduce a parallel mailbox store, worker, lease/retry/DLQ state, acknowledgement rule or terminal state. Before implementation, no bridge type, example, replay fixture or dedicated gate exists.

Any future bridge must retain distinct facts for dispatch accepted, mailbox lifecycle, host effect evidence and host outbox confirmation. It must reject invalid/unknown/stale/non-equivalent duplicate requests before mailbox mutation; exact duplicates must converge through source-owned identity. Rollback stops new bridge calls and preserves pending message/outbox/effect facts for host reconciliation.

## Verification

Existing mailbox behavior is covered by the mailbox lifecycle and integration
suites listed in `docs/mainline-contract-test-index.md`. The delivered bridge
evidence is `scripts/check-mailbox-plan-bridge-contract.sh/.ps1`: it runs the
real-owner lifecycle/parity tests and reference-only replay fixture. Rollback
stops new `AdmitHostPlan` calls; it does not delete or reinterpret retained
mailbox records.
