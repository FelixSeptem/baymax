## Why

The existing mailbox continuation is deliberately lifecycle-only: it seals and
claims source facts, but `ResumeHostPlanContinuation` returns a projection and
never resumes a worker handler or business effect. A host that needs
`input_required`, `approval_required`, or `budget_exhausted` inside a mailbox
handler therefore has no source-owned safe point. Re-enqueueing, consuming a
new delivery, replaying lifecycle history, or re-running a parent would lose
the original delivery/lease/attempt boundary and can duplicate effects.

This change gives the existing mailbox owner an opt-in handler continuation
path. It makes that path observable and admissible only when a configured
owner can provide the required durable material and handler resumer; unconfigured
mailbox workers remain explicitly unsupported.

## What Changes

- Add an owner-native worker handler result protocol for a bounded pause at an
  effect-safe boundary. It represents `input_required`, `approval_required`,
  or `budget_exhausted` with pending/completed effect references, budget and
  deadline facts, and an opaque bounded handler checkpoint reference.
- Persist the handler continuation atomically in the existing mailbox state,
  bound to the original message, delivery, lease, attempt causation,
  host-plan identity, authority, occurrence, and effect namespace. A pause
  retains the original in-flight delivery and performs no Ack, Nack, Requeue,
  retry, publish, or replacement consume.
- Add a registered source-owned mailbox handler resumer and resume entry point.
  It validates the continuation, live delivery/lease/attempt state, deadline,
  deployment scope, resolver reauthorization, and atomic duplicate claim before
  invoking the resumer once. It delegates final Ack, recoverable failure,
  retry/DLQ, and terminal projection to the existing mailbox owner.
- Promote only evidenced `mailbox_async × pause reason` manifest cells when
  the durable owner and handler-resume configuration are present. Keep default
  lifecycle-only mailbox continuation and missing/invalid configurations typed
  unsupported, unavailable, stale, conflict, or recovery-required before any
  handler/effect fallback.
- Extend planbridge admission so a handler continuation requires the matching
  owner-native evidence and still delegates all lifecycle mutation to mailbox;
  it does not create a worker, queue, executor, scheduler, or parent Run.
- Add focused owner, fresh-process, duplicate claim, lease loss, failure
  lifecycle, no-side-effect, Run/Stream parity, capability fixture, and
  PowerShell/shell contract-gate coverage; update public docs and examples to
  distinguish handler restore from lifecycle replay.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `multi-agent-mailbox-contract`: add explicit owner-native worker handler
  pause/resume behavior and mailbox-owned terminal lifecycle arbitration.
- `composite-continuation-restore`: distinguish mailbox handler execution
  continuation from lifecycle projection, and require its source-owned safe
  boundary for any supported mailbox pause cell.
- `host-runtime-capability-manifest`: publish mailbox handler safe-point and
  per-reason support only from explicit owner configuration/evidence.
- `host-execution-plan-bridge`: require matching mailbox handler continuation
  evidence at admission and preserve owner terminal semantics for Run/Stream.

## Impact

- **Code/API:** `orchestration/mailbox` gains opt-in worker handler pause and
  resumer APIs, persisted continuation material, and typed outcomes;
  `orchestration/planbridge` gains evidence-gated admission only. Existing
  `WorkerHandler`, lifecycle APIs, and lifecycle-only continuation remain
  compatible.
- **Ownership:** mailbox remains the only delivery/lease/retry/DLQ owner.
  Planbridge never executes, resumes, consumes, acknowledges, requeues, or
  creates a second mailbox owner. The host retains business authorization and
  effect idempotency.
- **Deployment:** File-backed configured owners support the repository's
  documented scope; memory owners remain process-local. A missing coordinator
  or unsupported scope fails closed rather than claiming distributed restore.
- **Documentation and examples:** update the mailbox and planbridge public
  boundaries, integration/capability guidance, test index, roadmap, and an
  offline example marker to make the opt-in support and rollback path explicit.

## Example Impact Assessment

修改示例

理由：更新现有 host execution plan / native event example 的语义锚点，展示显式
配置的 mailbox handler continuation、其 admission evidence 和默认拒绝边界；不
新增业务 payload、平行 worker 或自动重跑示例。

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
