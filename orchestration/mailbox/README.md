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

## Native retained history

The existing mailbox owner also implements `types.NativeEventHistoryReader`.
`HistorySource` returns a source-instance reference; `Capabilities` declares
`mailbox.history.v1`, source-local sequence ordering, a 1024-entry retention
window, cursor scope/version, durability and the limit of terminal recovery
(mailbox lifecycle only, never host business commit). `ReadAfter` returns a
bounded detached page and opaque versioned cursor. A nil cursor starts at the
current retained floor; a cursor before that floor returns
`types.ErrCursorExpired` with floor evidence, a discontinuity returns
`types.ErrHistoryGap`, malformed/foreign cursors return their distinct typed
errors, and pending/failed FileStore persistence returns
`types.ErrSourceUnavailable`. It never falls back to latest.

The floor is the after-position immediately before the first retained sequence;
the retention policy is a count bound (last 1024 events), not a time guarantee.
FileStore persists journal and mailbox snapshot atomically, so a newly opened
FileStore can continue from a host-confirmed cursor. MemoryStore supports only
process-local history and must not be advertised as durable. Legacy snapshots
without a journal do not claim retained history. Journal entries contain bounded
reference-only lifecycle/terminal facts and omit envelope payload, credentials,
callback errors and business-commit assertions.

Replay is structurally read-only: it does not publish, consume/claim, renew,
ack, nack, requeue, retry, reclaim, dead-letter, invoke a handler, Run/Stream,
resume a checkpoint or write diagnostics. `Complete` means the current retained
tail is reached, not that the source can never emit another lifecycle event.
Runner, child, team and workflow have no mailbox journal and remain explicitly
unsupported. Cursor replay is not execution recovery. Contract coverage is
listed in [the mainline contract index](../../docs/mainline-contract-test-index.md).

## Host-plan continuation

Mailbox host-plan continuation is an explicit operation of the existing mailbox
owner. It seals reference-only checkpoint material bound to the original
message, delivery, lease, attempt causation, host/run/parent correlation and
authority/plan/write-scope/effect/occurrence identities. It excludes business
payloads and business-commit assertions. `FileStore` supports same-host
single-writer sequential reopen; `MemoryStore` is process-local and does not
advertise cross-process restore.

`SaveHostPlanContinuation`, claim, running and terminal projection mutations
are atomic with the mailbox lifecycle owner. A stale save after ack, requeue,
lease loss or another terminal transition fails closed and cannot overwrite the
source record. Resume never publishes, consumes, re-enqueues, acknowledges,
nacks, retries or creates a second queue/worker/parent Run. Lease loss, retry,
DLQ and timeout stay source lifecycle facts, not host business commits.
`input_required`, `approval_required` and `budget_exhausted` have separate
checkpoint reason evidence; unavailable, corrupt, foreign, expired and
conflicting states remain typed failures.

Mailbox continuation is a lifecycle projection, not a host handler execution
safe point. `ResumeHostPlanContinuation` claims and returns the current source
outcome but does not invoke a plan handler or business effect. Consequently
`host_context_projection.v1` cannot be injected through mailbox continuation:
the manifest/context admission must remain `unsupported` until the mailbox
owner exposes a message/delivery/lease/attempt-bound handler safe point with
its own lease, retry, DLQ and duplicate-delivery evidence. Planbridge must not
fill this gap with a wrapper executor or a second consume loop.

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

The mailbox history capability also declares `deployment_scope.v1`. FileStore
reopen is same-host single-writer sequential evidence only; MemoryStore is
process-local. Mailbox does not claim multi-writer CAS, cross-machine
coordination or multi-instance cursor/claim fencing.
