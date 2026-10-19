## Context

See `proposal.md` for motivation and the three delta specs for observable behavior. Today `orchestration/planbridge` has a bounded `host_execution_plan.v1` compiler, a detached compiled snapshot and one explicit native binding per shape, but `compile.go` rejects every non-inline delivery before source activation. `execute.go` delegates each supported shape to its owner, yet its local-child path is deliberately exceptional and must not become the model for mailbox execution.

`orchestration/mailbox` already owns durable message identity, publish/claim, lease, ack/nack/requeue, retry and DLQ state. The bridge must therefore depend on a narrow public mailbox binding/admission/outcome surface rather than instantiate storage, worker or lifecycle control. The existing native event adapter remains a realtime callback projection only; it cannot become durable history, an event cursor owner or a replacement outcome store.

## Goals / Non-Goals

**Goals:**

- Publish a bounded manifest that makes form-level support and rejection discoverable before a host dispatches.
- Extend planbridge with a `mailbox_async` path that validates a full host/source binding, delegates exactly once to the existing mailbox owner and exposes source-owned admission/lifecycle/terminal facts.
- Preserve logical-message idempotency and all authority, occurrence, scope and causation facts across duplicate dispatch, lease loss, retry and DLQ.
- Keep equivalent Run/Stream views of an admitted continuation source-terminal-equivalent without creating a new execution.

**Non-Goals:**

- No durable event-history/cursor implementation, checkpoint store, cross-process restore, or composite continuation. Spec 2 and Spec 3 own those changes.
- No host outbox/effect commit ownership, new queue/scheduler/worker, mailbox store migration, global event ordering, retry policy, direct-run fallback, Provider SDK dependency, configuration key, or diagnostic writer.
- No assertion that mailbox delivery accepted, ack, nack, retry or DLQ is a business commit or that a metadata reference can resume a parent run.

## Decisions

### 1. Expose one immutable capability manifest before admitting mailbox plans

Add a small public contract package or source-neutral public type that produces a detached, bounded capability document. Its manifest identity includes contract version, public API version and build/module revision; each form entry includes the required boolean/enum capability dimensions, retention policy and a bounded evidence reference. The initial document is conservative: it reports existing unsupported capabilities as unsupported rather than leaving fields absent.

Planbridge receives either this manifest or a narrow manifest reader at construction/compile time. It verifies the requested form/delivery/required capabilities during compile and verifies compatible version plus current mailbox availability at admission. This separates static eligibility from current availability while avoiding a global capability registry.

Alternative considered: infer support from `Binding` variants or `nil` fields. Rejected because a caller cannot distinguish an unimplemented binding from temporary unavailability, and package/DTO presence would become an accidental public compatibility promise.

### 2. Add a mailbox binding variant; delegate lifecycle through a public owner interface

Extend `planbridge.Binding` with exactly one mailbox-specific variant, mutually exclusive with direct/team/workflow/child variants. Its descriptor projection carries only bounded identity/digest/scope/limit facts, while the bound mailbox service is an opaque interface supplied by the caller from the existing `orchestration/mailbox` owner.

The mailbox owner interface admits one full logical binding and returns a source-owned admission/outcome reference. It is responsible for duplicate convergence, publish, delivery, leases, retry and DLQ. Planbridge validates descriptor-to-binding consistency, calls host admission, then performs one owner admission; it does not call a handler, drive a worker or invoke a new runner. DTOs returned to callers are snapshots, not mutable mailbox records.

Alternative considered: have planbridge convert a descriptor into `mailbox.Envelope` and call a new worker. Rejected because it duplicates lifecycle ownership and cannot correctly preserve lease/retry/DLQ/source terminal facts. Alternative considered: route mailbox plans through `composer.DispatchChild`; rejected because scheduler task/attempt identity differs from mailbox message/delivery/lease identity.

### 3. Define full logical admission identity and fail closed on conflict

The owner normalizes an idempotency key from source message identity plus parent run, causation, authority digest, plan digest, write scope, effect namespace and occurrence. Every component is bounded and included in compile/admission consistency checks; no business payload enters the canonical plan digest. An exact repeat returns the original admission projection. Any non-equivalent reuse returns a typed conflict before publication.

The bridge preserves source identifiers rather than minting a bridge execution ID. A plan digest proves descriptor/binding consistency only; current host authorization remains an input to `Binding.Admit`, and mailbox lease/current availability remains an input to owner admission.

Alternative considered: deduplicate only on message ID or plan digest. Rejected because either permits authority/scope/occurrence drift to join an unrelated logical execution.

### 4. Model lifecycle and terminal evidence as source projections, not bridge transitions

The mailbox outcome has explicit delivery lifecycle and attempt evidence: ack/nack/retry/DLQ result, message ID, delivery/lease ID and retry causation. It separately carries the source terminal outcome when the source has one. Run and Stream project the same logical mailbox admission/outcome through their respective supported source paths and normalize only explicitly allowed stream fragmentation; neither creates a parent Run, retries a delivery or synthesizes a terminal.

Lease loss and timeout are forwarded from mailbox owner evidence. A failed lease, retry or DLQ cannot trigger recovery by calling `Run` again with the parent identifier. Host business commit remains a separate external fact.

Alternative considered: let a bridge-level outcome reducer translate every mailbox state into a terminal result. Rejected because it would conflate delivery lifecycle with source terminal/business commit and create a competing state machine.

### 5. Preserve bounded validation and no-side-effect ordering

`Compile` continues to snapshot and validate all descriptor/binding references before source activation. The mailbox branch adds bounded checks for contract version, message/parent/causation identity, authority and plan digest, scope/namespace and resource limits. `execute` rechecks current host admission and manifest/source availability immediately before owner admission. Failures return stable typed categories without handler/model/tool invocation, publish, claim, acknowledgement, retry, DLQ or diagnostic write.

The existing `DeliveryInline` and unsupported resumed path retain their behavior. A known mailbox binding does not imply source history, checkpoint creation, cursor replay, cross-process restore or pause support; the manifest reports those dimensions independently.

## Risks / Trade-offs

- **A manifest becomes stale between compile and dispatch** → compile validates compatibility, execute rechecks current manifest/source availability, and stale decisions require explicit recompilation rather than fallback.
- **Duplicate rules diverge from existing mailbox semantics** → define the logical-admission normalizer beside mailbox owner tests, expose only its owner result to planbridge, and test conflicting duplicates at both boundaries.
- **An outcome DTO leaks business details or mutates after return** → use reference-only, fixed-size fields; copy public maps/slices; reject bodies, credentials and raw callback errors before serialization.
- **Run/Stream adaptation falsely suggests a source continuation is complete** → retain delivery/lifecycle/terminal as distinct fields and compare normalized source facts in integration/replay fixtures.
- **Manifest grows into a runtime registry** → manifest is a local snapshot/read interface with no remote discovery, mutable global state, scheduling or authorization capability.

## Migration Plan

1. Add manifest types/read path with all existing unsupported forms explicitly represented, leaving existing planbridge behavior unchanged by default.
2. Add mailbox public binding/outcome DTOs and owner admission interface behind constructor-only opt-in; retain direct mailbox APIs and existing `Binding` paths unchanged.
3. Extend compiler, snapshot/canonicalization and execute path only after negative tests demonstrate no publish/Run on absent capability, version mismatch or binding conflict.
4. Enable `mailbox_async` only when the supplied manifest and mailbox owner both declare the compatible profile; otherwise preserve typed rejection.
5. Update docs/example/gates, run full verification and record actual evidence. Rollback removes construction of the opt-in bridge or marks the form unavailable; it leaves existing mailbox records and host business facts intact.

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`docs/development-roadmap.md`
- 维护 owner：orchestration、mailbox、host-integration 与 documentation maintainers
- 验证证据：OpenSpec strict validation、documentation impact gate、roadmap consistency 与 mailbox plan bridge contract gate

`docs/development-roadmap.md` is the currently synchronized planning baseline. Before implementation tasks are marked complete, the final public API, limits, typed errors, support matrix, rollback and example runtime path must be synchronized in the affected maintained documentation named by `proposal.md`.

## Documentation Impact Assessment

| Area | Outcome | Affected paths | Owner | Verification |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/development-roadmap.md` | orchestration maintainers | 复核 owner 分层及禁止第二 queue/worker/scheduler/state machine |
| components | 修改文档 | `orchestration/planbridge/README.md`、`orchestration/mailbox/README.md` | orchestration/mailbox maintainers | manifest、binding/outcome、owner delegation 与 support matrix 对照 source tests |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime configuration maintainers | mailbox availability/version 的 constructor/admission 语义；无新增配置时明确保持 `env > file > default` |
| contract/API | 修改文档 | `docs/guides/integration.md`、`docs/mainline-contract-test-index.md` | contract and host-integration maintainers | spec requirements、public DTO、Run/Stream integration、fixture/replay/gate 对照 |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md` | observability maintainers | source-owned bounded correlation/outcome；RuntimeRecorder 单写且无新 QueryRuns contract |
| examples | 修改文档 | `examples/host-execution-plan/README.md` | examples maintainers | README 文档先行、offline marker、no-fallback 与 rollback 路径验证 |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | host-integration maintainers | compile/admit/dispatch、typed rejection、duplicate convergence 与 host boundary |
| best practices | 修改文档 | `docs/guides/best-practices.md` | documentation maintainers | digest/ack/DLQ 不等于 business commit 或 restore 的反例核对 |
| roadmap | 修改文档 | `docs/development-roadmap.md` | roadmap maintainers | `openspec list --json` 与 roadmap status consistency |
