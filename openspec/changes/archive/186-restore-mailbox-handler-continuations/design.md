## Context

See [proposal.md](proposal.md) for motivation and the delta specs for observable
behavior. Today `Worker.RunOnce` maps every handler return to Ack or configured
Nack/Requeue. `MailboxContinuation` is already stored atomically with the
mailbox owner, but its resume method intentionally returns a source projection
without invoking a handler. The mailbox owns all delivery/lease/retry/DLQ
transitions and the plan bridge is only an admission/delegation layer.

The implementation must preserve this ownership: an execution-safe pause is
part of one existing leased delivery, rather than a new queue, worker or
planbridge execution loop. A FileStore can provide the configured deployment
profile; memory-backed owner state is necessarily process local.

## Goals / Non-Goals

**Goals:**

- Define a bounded, explicit pause result that a worker handler can return at a
  pre-effect safe point for each canonical pause reason.
- Seal it using the existing mailbox continuation identity and atomically retain
  it before `RunOnce` returns, with no lifecycle mutation.
- Provide a single registered resumer path that validates current source state,
  revalidates host authorization, claims atomically, invokes once, and maps its
  terminal result through the original owner lifecycle operations.
- Publish continuation capability only when the exact owner has both durable
  state and handler continuation configuration, with pause reason evidence.

**Non-Goals:**

- No generic ExecutionGraph, second mailbox store, scheduler, worker, queue,
  planbridge executor, distributed transaction, or cross-machine assertion.
- No conversion of journal replay, lifecycle projection, application snapshot,
  or reused run ID into handler restore material.
- No fallback to parent `Run`, re-enqueue, re-consume, automatic restart, or
  terminal fabrication after ambiguous persistence/lease state.
- No change to the behavior of existing workers which use only `WorkerHandler`.

## Decisions

### 1. Add a result-capable handler beside the compatible legacy handler

`WorkerHandler` remains unchanged. An opt-in `ContinuationWorkerHandler` returns
a bounded `HandlerResult`; its optional `HandlerPause` carries the canonical
reason plus bounded checkpoint-reference fields. A
`ContinuationWorkerResumer` and `HandlerResumeRevalidator` are supplied at
construction rather than interpreting a special `error`.
This is selected with a worker option/configuration rather than interpreting a
special `error`; it prevents arbitrary errors from being converted into pauses.

Alternatives considered:

- **Encode pause in `error`**: rejected because a handler error would gain
  accidental lifecycle semantics and make untrusted payload/error strings part
  of the pause protocol.
- **Teach planbridge to execute the handler**: rejected because it would make
  the bridge a second worker and lease owner.

### 2. Reuse `MailboxContinuation`; add handler material and execution state

The existing envelope is already digest-bound to host-plan identity and delivery
facts. Add a versioned bounded opaque handler checkpoint reference and a handler
continuation marker, retaining the existing checkpoint digest,
generation, completed-effect, budget/deadline and claim fields. Store these in
the same mailbox snapshot transaction as the in-flight record check.

A checkpoint reference does not persist a Go closure or business material. The
new process supplies the registered `ContinuationWorkerResumer` and required
`HandlerResumeRevalidator`; revalidation happens before the execution callback.
This allows a newly opened FileStore to resume only when it can supply the
matching resumer and current authorization; absence is a typed unavailable
outcome.

### 3. Pause stops heartbeat; resume owns a new bounded heartbeat lifetime

`RunOnce` heartbeats while calling the handler. On successful sealing it stops
its goroutine and returns as processed without terminal mutation. The persisted
record retains the same current lease. `ResumeHandlerContinuation` verifies
that the delivery/lease/attempt is still the live in-flight record before
claim/start and creates a heartbeat loop for its resumer invocation. If a
heartbeat or source state is lost, it never executes again and returns typed
lease-loss/recovery-required evidence.

This deliberately requires resume before the original lease expires. A
reclaimed lease has been assigned a new source delivery by normal mailbox
policy, so trying to resume the old handler would violate the effect boundary.

### 4. Terminal work routes through existing mailbox lifecycle APIs

The resumer returns `HandlerResult`. Completion calls `Ack`; a recoverable
failure uses the worker's configured Nack/Requeue policy; source errors retain
the claim as recovery-required. Claim terminal projection is persisted after
the owner lifecycle call so duplicate resumes return source facts. An owner
crash in an ambiguous commit window never invokes the resumer automatically.

### 5. Capability promotion is configuration- and reason-specific

`ContinuationCapabilities` gets all three evidence cells only when the
mailbox's declared handler continuation configuration is available. The API
does not infer support from a FileStore alone. Manifest promotion continues to
require the durable owner capability and handler safe point, so default
mailbox/lifecycle owners remain conservative.

## Risks / Trade-offs

- [Handler creates an irreversible effect before returning pause] → API
  documents/validates only pre-effect safe-point intent; effect idempotency
  remains host responsibility, and resumed owner never restarts an ambiguous
  claim.
- [Lease expires while paused] → original delivery validation returns
  `ErrMailboxContinuationLeaseLost`; no restart/requeue is synthesized.
- [Process crashes after lifecycle mutation but before claim projection] → a
  subsequent same request receives recovery-required rather than executing.
- [Resolver configuration changes after restart] → reauthorization/resolver
  failure occurs before resumer execution and remains unavailable/denied.
- [Long resumer outlives lease] → owner heartbeat is maintained only during
  active invocation; heartbeat failure prevents terminal fallback.

## Migration Plan

1. Add result/resumer DTOs and strict bounded validation while keeping current
   worker constructor compatible.
2. Add failing owner tests, then worker pause/save and restore/terminal code.
3. Add manifest/planbridge evidence promotion and negative admission tests.
4. Update fixtures, dedicated gates, docs, example markers, test index and
   roadmap state.
5. Rollback disables/removes the explicit continuation configuration. Existing
   persisted handler checkpoints remain source records and fail closed rather
   than being reinterpreted by a legacy worker.

## Example Impact Assessment

修改示例

理由：现有示例将明确展示配置门槛、Run/Stream parity 和 rollback 标记，而不是把
mailbox lifecycle replay 误写成 handler restore。

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/development-roadmap.md`、`docs/guides/integration.md`、`docs/runtime-module-boundaries.md`、`docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`orchestration/mailbox/README.md`、`orchestration/planbridge/README.md`、`examples/host-execution-plan/README.md`、`examples/native-execution-events/README.md`
- 维护 owner：runtime contracts owner、mailbox owner maintainer、planbridge maintainer 与 examples maintainer
- 验证证据：mailbox owner focused/fresh-process tests、Run/Stream parity、dedicated PowerShell/shell gate、race/lint/quality/docs gates 和 OpenSpec validation

## Documentation Impact Assessment

| area | outcome | affected paths | owner | evidence |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | runtime contracts owner | source-owner boundary review |
| components | 修改文档 | `orchestration/mailbox/README.md`、`orchestration/planbridge/README.md` | mailbox/planbridge maintainers | focused owner and bridge tests |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime contracts owner | no-new-config regression and documentation gate |
| contract/API | 修改文档 | `docs/guides/integration.md` | runtime contracts owner | strict OpenSpec validation |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md` | diagnostics maintainer | dedicated contract gate |
| examples | 修改文档 | `examples/host-execution-plan/README.md`、`examples/native-execution-events/README.md` | examples maintainer | expected-marker review |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | runtime contracts owner | pre-admission rejection tests |
| best practices | 修改文档 | `README.md` | documentation maintainer | rollback boundary review |
| roadmap | 修改文档 | `docs/development-roadmap.md` | documentation maintainer | roadmap consistency gate |
