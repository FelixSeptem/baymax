## Context

Baymax's `tool/local.Dispatcher` already owns lookup, JSON Schema validation, middleware chaining, retry, panic recovery, sandbox dispatch, timeout handling, input-order result placement, and per-call diagnostics. ReAct and terminal-outcome contracts consume those results, while policy, allowlist, egress, and sandbox contracts retain authority over admission and side effects. The gap is a cross-layer contract that makes the logical lifecycle and failure isolation observable and replayable without moving ownership into a new coordinator.

The design therefore treats lifecycle stages as a normalized projection over one existing invocation, not as a second execution state machine. A call keeps its original `call_id`, name, source, input index, run/step correlation, and existing `ClassifiedError`/reason data. Stage facts are additive and nullable so older consumers can ignore them.

## Goals / Non-Goals

**Goals:**

- Define deterministic `prepare -> validate -> authorize -> execute -> finalize` stage outcomes for source-owned tool calls.
- Distinguish pre-execution rejection from execution failure, cancellation, timeout, retry exhaustion, middleware short-circuit, and panic while reusing existing error families.
- Guarantee an idempotent finalize projection for every call that enters dispatch processing, including abnormal exits, without claiming that cleanup can undo an already-owned side effect.
- Preserve input ordering, `call_id` correlation, partial valid results, tool retry metadata, terminal outcome ownership, and Run/Stream semantic equivalence.
- Make lifecycle facts available through `RuntimeRecorder`, diagnostics replay, focused integration tests, and a deterministic shell/PowerShell gate.
- Provide an additive path that can be enabled or observed without requiring a hosted service or external dependency.

**Non-Goals:**

- No new Tool, MCP, middleware, sandbox, policy, or terminal state machine.
- No transport listener, hosted tool executor, global invocation queue, external event store, or platform control plane.
- No change to existing ReAct termination names, policy precedence, sandbox/egress upper bounds, retry policy defaults, or Go error compatibility.
- No exactly-once side-effect guarantee; the contract records source-owned attempts and outcomes rather than manufacturing transactional semantics.
- No requirement that every internal implementation detail become public API; only normalized, bounded contract fields are exposed.

## Decisions

### 1. Project stages over the existing dispatcher path

The normalized lifecycle is derived at the existing boundaries:

```text
prepare -> validate -> authorize -> execute -> finalize
                              \-> rejected result
```

`prepare` records identity and ordering. `validate` covers lookup and argument/schema checks. `authorize` covers policy, allowlist, sandbox capability, and egress decisions before side effects. `execute` covers middleware, sandbox or local invocation, retry, timeout, cancellation, and panic recovery. `finalize` normalizes the result, records bounded diagnostics, places the outcome at the original input index, and exposes any terminal projection.

Alternative considered: introduce explicit stage objects that own execution transitions. Rejected because it would duplicate dispatcher and terminal ownership and create a second state machine.

### 2. Reuse existing failure and terminal taxonomies

Each stage maps to existing `ClassifiedError`, source reason codes, ReAct/tool termination reasons, and the shared terminal arbiter. The new contract adds a bounded `lifecycle_stage` and `failure_origin`/`isolation_action` projection only where needed to distinguish rejection, failure, and cleanup. Unknown or legacy values remain nullable/defaultable.

Alternative considered: define a new universal tool error enum. Rejected because it would conflict with provider, security, and terminal taxonomies already used by callers and replay fixtures.

### 3. Finalization is idempotent observation, not compensation

The dispatcher MUST emit at most one normalized finalization record per `(run_id, step_id, call_id, attempt_scope)` projection. Repeated callbacks or recovery observations deduplicate; late conflicting observations are diagnostic-only and cannot rewrite a previously accepted business result. Finalization records cleanup status and retained facts but does not promise rollback of external tool side effects.

Alternative considered: retry or compensate automatically from the lifecycle layer. Rejected because retry belongs to the existing dispatcher policy and compensation is tool/domain-specific.

### 4. Keep ordering and concurrency source-owned

Parallel calls may finish in any order, but normalized outcomes and model feedback remain ordered by the original input slice and `call_id`. The lifecycle contract records completion timing only as bounded metadata; it does not serialize calls or introduce a global queue.

Alternative considered: expose completion order as the canonical feedback order. Rejected because it would break existing ReAct parity and make model input nondeterministic.

### 5. Diagnostics and replay are the compatibility surface

New fields flow through `RuntimeRecorder` and `runtime/diagnostics` using additive nullable/default semantics. Replay fixtures use stable, bounded values and classify drift for stage order, failure origin, finalize idempotency, retained facts, call ordering, and Run/Stream parity. High-cardinality arguments, stack traces, and raw payloads remain outside OTel metric dimensions.

Alternative considered: make lifecycle metadata a mandatory public API before evidence exists. Rejected to preserve compatibility and allow source owners to opt into richer details incrementally.

### 6. Gate ownership remains library-first

The dedicated gate checks source ownership and dependency boundaries, canonical fixture replay, negative classifications, and shell/PowerShell parity. It rejects transport listeners, hosted execution/session stores, global lifecycle queues, and a second terminal state machine in the recovery path. Existing quality and docs gates remain the final integration path.

## Risks / Trade-offs

- [Risk] Different source owners may omit a stage that is not applicable to their path. → Require explicit `skipped`/`not_applicable` projection with a stable reason rather than silently fabricating execution.
- [Risk] Panic or cancellation can occur after a side effect begins. → Record `started` and `finalize` facts separately, preserve the source result, and explicitly avoid exactly-once or compensation claims.
- [Risk] Adding per-call metadata can increase diagnostics volume. → Bound strings/lists, keep high-cardinality values in structured payloads only, and add cardinality/replay assertions.
- [Risk] Middleware and sandbox may both report an error for one call. → Preserve the first authoritative business result, record later observations as secondary diagnostics, and reuse terminal conflict rules.
- [Risk] Run and Stream implementations may evolve independently. → Require shared normalized fixture comparisons and parity tests in the dedicated gate before task completion.
- [Risk] The contract could become an alternate Tool API. → Keep all stage projection adapters behind existing dispatcher/middleware interfaces and reject public API duplication in the boundary gate.

## Migration Plan

1. Freeze the stage vocabulary, failure-origin mapping, field bounds, and example impact in the new spec and docs.
2. Add projection types/helpers and tests without changing existing dispatch behavior or defaults.
3. Thread additive recorder/diagnostic fields through both Run and Stream paths, then add canonical and negative replay fixtures.
4. Add integration and example assertions, followed by shell/PowerShell contract gates and quality-gate wiring.
5. Roll back by disabling/removing the projection and gate wiring; existing ToolCall/ToolResult, ClassifiedError, and terminal fields remain usable because no breaking field or owner migration is required.

## Open Questions

- Whether the first implementation should expose stage facts only in diagnostics/replay or also attach a bounded lifecycle summary to `ToolCallOutcome`; the spec keeps both compatible by making any public projection additive.
- Whether sandbox-hosted calls need a distinct `authorize` subreason or can reuse existing capability/egress reason codes; the initial mapping should prefer existing canonical reasons and add a subreason only where replay cannot otherwise distinguish the boundary.

## Example Impact Assessment

修改示例
