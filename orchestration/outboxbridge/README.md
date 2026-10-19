# Application outbox dispatch bridge

`New(Config)` explicitly binds a trusted current host `Admission` collaborator and
one or both native owners. There are no config keys, background workers or stores.
`Dispatch` publishes/enqueues once; `Inspect` only reads target facts. Neither
method executes effects, confirms outbox rows, acknowledges messages, completes
tasks, retries or restores execution. Stop bridge calls to roll back; preserve
pending native records and host facts.

## Bounded profile and identity

`application_outbox_dispatch.v1` accepts reference tokens only (ASCII letters,
digits and `._:/@+-`, 1–256 bytes), at most 16 KiB of canonical JSON. Required:
outbox ID, revision, authority reference, effect reference, effect key, idempotency
key, source run ID, target and exactly one target binding. Occurrence ID, source
message/task/attempt references are optional. A mailbox binding names an agent;
a scheduler binding names an agent and a maximum attempt count (1–100). Both
support `not_before`; mailbox also supports `expire_at` later than `not_before`.
Version and target are closed strings; unknown versions/targets fail closed.

All normalized descriptor fields enter the SHA-256 binding digest. Native IDs
are `outbox-` plus SHA-256 of the normalized idempotency key (independent of the
binding digest). The mailbox idempotency key is the same native ID. Payloads
contain only `outbox_binding` and `effect_ref`; no business body is accepted.
Native `DispatchBinding` fields persist the binding digest and reject unequal
reuse atomically under the source store lock, including collision with an
unbound native record. Existing unbound callers retain their duplicate semantics.

The host MUST durably bind each idempotency key to exactly one full descriptor,
including target, and compare current revision/authority/effect eligibility on
EVERY admission. This is essential across distinct native owners: they do not
share a key registry. Returning an echoed caller digest without checking trusted
host state is not admission. The bridge holds no callback/source lock, caches no
authority and rechecks cancellation after admission. Native operations then
recheck their own current identity/state atomically. Admission is a point-in-time
host decision, not an atomic transaction with host revocation or business commit;
the host must revalidate authority in its effect CAS transaction.

`Decision` must return the exact current digest and `Eligible=true`; missing,
stale, denied, errored, panicking or canceled decisions cannot mutate targets.
Optional `Evidence` is host-supplied and separately bound to that digest. Its
`EffectCommitted` and `OutboxConfirmed` booleans are never derived from target
state. Evidence is a point-in-time observation, not a permission to execute.

## Errors and support matrix

Errors are fixed strings and never include descriptor contents or callback
errors: `outbox.invalid_descriptor`, `outbox.unsupported_profile`,
`outbox.unsupported_target`, `outbox.unavailable_target`,
`outbox.admission_denied`, `outbox.admission_failed`, `outbox.binding_mismatch`,
`outbox.canceled`, `outbox.target_failed`, `outbox.not_found`.

| Target | Dispatch | Inspect facts | Lifecycle owner |
| --- | --- | --- | --- |
| mailbox | command publish; native duplicate flag | message ID, current state and delivery attempt | mailbox consume/ack/nack/requeue/delay/TTL/lease/DLQ |
| scheduler | task enqueue; convergent task identity | task ID, current state and current attempt ID | scheduler claim/attempt/heartbeat/requeue/retry/terminal/DLQ |
| resumed/shared queue/planbridge | no bridge execution or resume API | no synthetic execution outcome | existing owner remains unchanged |

Scheduler dispatch does not claim a new/duplicate flag: native `Enqueue` returns
an existing record without exposing that distinction. `Accepted` says only that
the native owner returned the exact binding. Inspect can find any native state,
including terminal or DLQ, without creating a new attempt or delivery. Native
delays and retry policies remain native. Source Run/Stream execution occurs only
through the existing supported integration; the bridge itself has neither mode.

## Crash recovery

After acceptance but before host outbox confirmation, repeat `Dispatch` with the
same descriptor, or `Inspect` to read the existing record. After host effect CAS
but before native ack/terminal, inspect trusted effect evidence and let the host
use native APIs to reconcile; do not repeat the effect without its idempotency
check. No result implies a business transaction, exactly-once execution or an
outbox confirmation. Durable recovery requires a durable native owner with
successful persistence (not memory or deferred unflushed file batching), plus
durable host key bindings/effect CAS. Recovery is at least once.
