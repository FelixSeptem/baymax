## Context

Baymax has several mature but separately evolved contracts: `ClassifiedError` in `core/types`, ReAct termination reasons, provider streaming error taxonomy, protocol Run states, scheduler/mailbox recovery outcomes, and additive diagnostics through `RuntimeRecorder`. Existing contracts already preserve source ownership and Run/Stream parity, but they do not yet provide one cross-domain projection for the difference between an operation that was rejected before execution, work that started and then failed, work that was canceled, and work that reached a terminal result.

The design is intentionally additive. It extends existing protocol and execution contracts; it does not create a second state machine, replace source-specific reason codes, or require every failure to become a model-visible tool result. The owning subsystem remains authoritative for retry, queueing, policy, and recovery decisions.

## Goals / Non-Goals

**Goals:**

- Define a finite, versioned failure family and terminal outcome projection usable by Run, Stream, provider, tool, orchestration, and recovery mappings.
- Preserve the distinction between pre-execution rejection and post-start runtime failure.
- Make equivalent Run and Stream executions expose equivalent terminal class, retry/resume eligibility, causal correlation, and bounded attempt metadata.
- Preserve valid partial facts and source-owned diagnostics when failure occurs after work has started.
- Make terminal outcome publication idempotent and prevent terminal Runs from being mutated back to `working`.
- Provide deterministic replay fixtures and cross-domain integration coverage.
- Keep all new fields additive, nullable, or defaultable and route diagnostics through `RuntimeRecorder`.

**Non-Goals:**

- Replacing existing `ErrorClass` values, ReAct reason codes, provider reason categories, scheduler taxonomy, or recovery owners.
- Converting all failures into successful Go returns, model messages, or tool results.
- Introducing a protocol-owned queue, scheduler, session store, event store, hosted gateway, or control plane.
- Changing retry counts, timeout values, policy precedence, sandbox behavior, or recovery algorithms in this change.
- Defining a new transport or requiring a particular SSE/JSON-RPC implementation.

## Decisions

### 1. Use a small normalized failure family plus source detail

The projection uses the canonical families `none`, `rejected`, `policy_denied`, `runtime_failed`, `timed_out`, `canceled`, `retry_exhausted`, and `recovery_conflict`. A successful terminal outcome uses `none` with terminal state `completed`. Existing `ErrorClass`, source reason codes, and detailed messages remain nested/additive metadata.

**Why:** callers need stable cross-domain grouping without losing provider/tool/scheduler detail.

**Alternatives considered:**

- Reuse only existing `ErrorClass`: rejected because it conflates domain and lifecycle phase.
- Create a separate taxonomy for every subsystem: rejected because cross-domain consumers would still need another mapping layer.

### 2. Keep source-owned decisions and project, never synthesize

Each owner emits or maps a terminal outcome containing phase (`pre_execution` or `post_start`), normalized family, terminal Run state, source reason, retryable/resumable flags, and bounded causal/attempt metadata. The protocol layer validates and projects this data; it does not enqueue, retry, cancel, resume, or branch work.

**Why:** this preserves the existing library-first ownership rule and prevents a projection from becoming a hidden scheduler.

**Alternatives considered:**

- Let the protocol infer retry/resume from error text: rejected because inference is non-deterministic and bypasses policy owners.
- Make every runtime error an error Event only: rejected because some failures must remain fail-fast or terminal Go errors.

### 3. Treat terminal publication as a single-assignment operation

The first valid terminal outcome wins. Repeated identical publication is an idempotent no-op. A later conflicting terminal outcome is retained as a conflict diagnostic/event and never overwrites the business terminal state. A terminal Run cannot transition back to `working`; retry creates a new causal attempt or source-owned Run according to the existing recovery contract.

**Why:** this matches existing replay/idempotency governance and prevents late callbacks from corrupting state.

**Alternatives considered:**

- Last writer wins: rejected because delayed provider or scheduler reports can overwrite a valid result.
- Silently ignore conflicts: rejected because operators need evidence of divergence.

### 4. Preserve partial facts after post-start failure

If a Provider stream or tool has emitted valid text deltas, complete tool calls, artifacts, or events before failing, those facts remain in the event/replay stream. The terminal projection adds the failure classification; it does not erase prior facts or re-execute consumed work.

**Why:** stream consumers need a truthful account of what happened, even when the final outcome is failed.

**Alternatives considered:**

- Retry the entire consumed stream: rejected because it can duplicate output or side effects.
- Return only the final error: rejected because it loses observable work.

### 5. Scope Run/Stream parity to semantic fields

Run and Stream may differ in event delivery and ordering, but equivalent requests must normalize to the same terminal family/state, source reason, retry/resume eligibility, causal relation, and aggregate attempt counts. Event order remains governed by existing realtime/source contracts.

**Why:** this extends the repository's existing parity rule without requiring identical transport behavior.

### 6. Roll out with additive projections and compatibility defaults

New terminal fields are omitted/defaulted for legacy fixtures. Existing consumers continue reading `RunResult.Error`, existing protocol states, and source reason codes. A profile/version marker identifies the normalized projection when present.

**Why:** the repository is in the 0.x additive compatibility model and already uses versioned replay fixtures.

## Risks / Trade-offs

- [Risk] A taxonomy that is too broad hides useful domain distinctions → Mitigation: retain source `ErrorClass`, reason code, phase, and bounded details alongside the normalized family.
- [Risk] Mapping code accidentally becomes a second scheduler or retry engine → Mitigation: require source-owned decision flags, reject synthesized retry/resume, and add a control-plane/source-ownership gate.
- [Risk] Different paths publish conflicting terminal outcomes → Mitigation: single-assignment arbiter, first-terminal-wins, conflict diagnostics, and replay tests.
- [Risk] Partial output retention exposes sensitive or high-cardinality data → Mitigation: reuse existing payload truncation, cardinality budgets, policy filters, and `RuntimeRecorder` constraints.
- [Risk] New additive fields drift between Run and Stream → Mitigation: shared normalization helper plus explicit parity fixtures for success, rejection, timeout, cancel, provider failure, tool failure, and recovery conflict.
- [Risk] Existing fixtures interpret absent fields inconsistently → Mitigation: define explicit default values and keep legacy `agent_runtime_protocol.v1` fixtures valid.

## Migration Plan

1. Add normalized types and validation/projection helpers in the existing protocol/types layer without changing current source state transitions.
2. Add provider, runner/tool, and recovery adapters that populate the projection only when source outcomes are already known.
3. Add additive diagnostics and OTel fields through `RuntimeRecorder`; keep high-cardinality payloads out of attributes.
4. Add replay fixtures and integration cases for each failure family, first-terminal-wins, conflict recording, partial facts, and Run/Stream parity.
5. Update examples and contract-test index with additive assertions.
6. Roll back by disabling projection emission and ignoring the additive fields; existing source errors and lifecycle states remain usable.

## Open Questions

- Should the normalized family be exposed as a new typed field on `RunResult`, a protocol-only field first, or both in the first rollout?
- Which existing diagnostics field names best fit the additive projection without duplicating `primary_reason`/`reason_code` fields?
- Does the current replay fixture format need a new version, or can the projection remain optional within existing fixture versions?

## Example Impact Assessment

修改示例

Existing Run/Stream, realtime, tool-loop, and recovery examples should assert additive terminal family/state fields; a new example is not required unless an existing fixture cannot express a boundary case.
