## Context

The archived `agent-runtime-protocol-contract` provides reference-only lifecycle DTOs and a minimal Run state machine. `SessionRef` currently contains only an identifier and source, while `RunRef` does not expose what protocol actions or optional runtime capabilities are available. The source runtimes already own adapter negotiation, readiness admission, scheduler queueing, snapshot persistence, and conflict handling; this change must make those facts discoverable without moving ownership into the protocol projection layer.

The design is constrained by the repository's library-first boundary, `env > file > default` configuration precedence, fail-fast plus atomic rollback for invalid configuration, RuntimeRecorder's single diagnostics write path, additive nullable default compatibility, and Run/Stream semantic equivalence.

## Example Impact Assessment

修改示例

The existing agent-runtime-protocol-projection example must expose the new descriptor, bounded context, and admission outcomes while retaining the v1 lifecycle projection.

## Goals / Non-Goals

**Goals:**

- Provide a versioned, transport-neutral protocol descriptor for runtime identity, supported protocol profiles, required/optional capabilities, and host-visible actions.
- Make Session context explicit through bounded participants, agent identity, scope, and metadata while keeping conversation bodies and provider SDK objects out of the protocol DTO.
- Express same-Session concurrent-Run policy and deterministic admission outcomes without implementing a second scheduler, lock, or persistence store.
- Reuse adapter capability negotiation vocabulary and reason taxonomy, including required/optional capability handling, `fail_fast|best_effort`, profile versioning, downgrade, and replay semantics.
- Preserve existing lifecycle, Realtime interrupt/resume, policy precedence, readiness admission, snapshot, diagnostics, and source-runtime ownership.

**Non-Goals:**

- No hosted Session service, remote scheduler, global Session lock, task queue, artifact store, or control plane.
- No mandatory concurrency policy for source runtimes that do not expose one; an unknown policy remains explicit rather than silently becoming serialization or cancellation.
- No provider-specific context schema, message history replacement, memory store, workspace store, RBAC, tenant model, or transport binding.
- No new terminal state, parallel interrupt/resume semantics, or mutation of a terminal Run into another attempt.

## Decisions

### 1. Add a protocol descriptor as a projection, not a registry

Add a `ProtocolDescriptor` containing protocol name/profile version, runtime identity, declared capabilities, supported actions, and compatibility metadata. Descriptor values are produced by the owning Runtime adapter or host configuration and are immutable for one effective profile.

Capability entries use the existing adapter negotiation shape: a stable capability name, optional version/profile, and required/optional request classification. Required capability absence returns deterministic fail-fast classification; optional absence can return a deterministic downgrade only when `best_effort` was explicitly selected. The protocol must not invent a registry or infer capability support from framework names.

Alternative considered: expose a free-form `map[string]any` metadata blob. Rejected because clients could not validate compatibility or distinguish discovery metadata from executable capability declarations.

### 2. Keep Session context bounded and reference-oriented

Add a protocol context projection associated with `SessionRef`. It contains participants/agent identities, declared context scope, and bounded scalar metadata. Participant and agent identifiers are references with source and role, not embedded prompts, files, provider objects, or full message history. Metadata limits and key normalization are deterministic so the projection cannot become an unbounded diagnostics surface.

Alternative considered: expose the complete Thread/message history. Rejected because history ownership belongs to the source Session/memory contract and would couple this protocol to storage, retention, and provider-specific message formats.

### 3. Represent host actions as an explicit allowlisted set

Expose a canonical action set for `cancel`, `resume`, `retry`, and capability-specific action names. A descriptor reports actions available under the current profile; it does not grant authorization. Authorization and policy precedence remain owned by the existing readiness, sandbox, policy, and HITL paths. Unsupported actions fail protocol validation before source state mutation.

Alternative considered: let clients assume all lifecycle operations exist because the minimal state machine names them. Rejected because `resume`, `retry`, and capability actions depend on source support and current state.

### 4. Model concurrent Run admission as an outcome projection

Define a finite policy vocabulary: `reject`, `serialize`, `branch`, and `optimistic`. Define deterministic outcomes: `admitted`, `queued`, `branched`, and `rejected`, with Session/Run correlation, conflicting Run references when available, a stable reason code, and optional branch/queue reference. An adapter may report `unknown` when the source does not expose a policy; the protocol must never synthesize queueing, cancellation, or a global lock.

The protocol validates that an outcome is compatible with the declared policy (for example, `queued` is not emitted for `reject`) and preserves causal relationships through existing Run references. It does not perform admission or change source scheduler state.

Alternative considered: make `reject` the universal default. Rejected because it would silently change existing source behavior and make an absent source contract look like a deliberate concurrency decision.

### 5. Preserve Run/Stream and source ownership

Descriptor, context, action, capability, and admission mapping helpers are pure or side-effect-free. They consume source-owned outcomes from Runner, Scheduler, Composer, A2A, Realtime, Snapshot, and policy/readiness paths. Equivalent Run and Stream inputs must normalize to equivalent descriptor negotiation and admission classifications after permitted event-order normalization.

### 6. Compatibility and observability are additive

New fields are nullable/defaultable and carry a protocol profile version. Existing `agent_runtime_protocol.v1` fixtures remain valid; new fixtures cover descriptor negotiation, bounded context validation, action support, admission policy/outcome compatibility, unknown policy, and Run/Stream parity. Diagnostics and OTel receive additive correlation fields only, and any diagnostics write continues through `RuntimeRecorder`.

## Risks / Trade-offs

- [Capability taxonomy drifts from adapter negotiation] -> Reuse adapter names, strategy, profile version, and reason codes; add cross-contract replay fixtures and a drift assertion.
- [Context metadata becomes an unbounded data or PII channel] -> Allow only bounded scalar metadata and references; enforce count/length limits and redact/deny unsupported values deterministically.
- [Admission projection is mistaken for a scheduler] -> Keep APIs side-effect-free, require source outcome input, and add a `concurrency_admission_owner`/control-plane-absence gate assertion.
- [Different source policies normalize inconsistently] -> Validate policy/outcome pairs, preserve source and reason metadata, and require Run/Stream equivalence fixtures.
- [New action fields are interpreted as authorization] -> Document action availability separately from permission; route decisions through existing policy precedence and readiness admission.

## Migration Plan

1. Add additive DTOs, enums, validation helpers, and profile fixtures without changing existing request/result or snapshot schemas.
2. Add opt-in projection adapters for Runner/Stream and source-owned concurrent admission results; hosts that do not request the descriptor continue using the existing protocol surface.
3. Add diagnostics/OTel correlation fields as nullable/default values and update replay/gate expectations.
4. Update the protocol example documentation before adding executable descriptor/context/admission assertions.
5. Roll back by disabling descriptor/context/admission projection wiring; existing lifecycle, scheduler, realtime, snapshot, and policy behavior remains unchanged.

## Open Questions

- Which source owners can provide a stable conflict Run list in the first implementation, versus only a reason code and policy outcome?
- Should the descriptor expose a single effective profile version or a bounded list of compatible versions for negotiation? The initial implementation should prefer a single effective profile plus supported range if existing adapter metadata permits it.
- Which context metadata keys are safe and useful for the first profile? The default should be an empty allowlist until the contract fixtures define bounded keys.
