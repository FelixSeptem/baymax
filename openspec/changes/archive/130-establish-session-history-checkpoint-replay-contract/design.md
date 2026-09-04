## Context

Baymax's library-first Agent Runtime Protocol already projects Session, Run, Step, Event, Artifact, and Checkpoint references. Unified snapshots own state export/import, checkpoint/workspace provenance owns lineage references, durable event binding owns stream recovery, and diagnostics replay validates historical fixtures. These contracts intentionally avoid a hosted session store, but they do not yet define how message history, checkpoint lineage, restore operations, and replay records relate to one another.

The change therefore adds a cross-boundary contract and adapters while preserving existing source ownership. The main consumers are embedded hosts that need to branch from a session leaf, restore a checkpoint safely, inspect lineage, or replay a run offline without re-running side effects.

Source-of-truth mapping:

| Surface | Owner | This change's role |
| --- | --- | --- |
| Session message history and leaf chain | host/session owner | expose bounded references and validate continuity |
| Run/Step execution and terminal outcomes | `core/runner` and owning orchestrators | retain causal correlation; never mutate terminal parents |
| Snapshot manifests and restore | `orchestration/snapshot` / composer recovery | validate history/checkpoint context before source mutation |
| Checkpoint/workspace lineage | protocol projection and provenance owner | add session association and branch/replay references |
| Runtime diagnostics | `observability/event.RuntimeRecorder` | record bounded additive decisions through the single writer |
| Offline replay fixtures | `tool/diagnosticsreplay` | normalize and validate without executing side effects |

Constraints:

- `library-first`: no remote gateway, hosted persistence service, tenant control plane, or second session repository.
- Existing snapshot manifests remain the state fact source; existing realtime, tool lifecycle, policy, sandbox, and terminal-outcome contracts remain authoritative.
- New protocol and diagnostics fields are additive, nullable/defaultable, and versioned.
- Diagnostics writes continue through `observability/event.RuntimeRecorder`.
- Equivalent Run and Stream inputs must normalize to equivalent history/checkpoint/replay outcomes.

## Goals / Non-Goals

**Goals:**

- Define canonical ownership and correlation between session history, runs, steps, events, checkpoints, snapshots, and replay records.
- Validate history-chain continuity and leaf-based branch/fork relationships without copying unbounded message content into protocol objects.
- Define restore compatibility, atomicity, idempotency, and deterministic conflict classifications.
- Make offline replay read-only and side-effect-free while preserving source-owned facts and terminal outcomes.
- Add valid, negative, drift, integration, replay, gate, and example coverage with shell/PowerShell parity.

**Non-Goals:**

- Creating a new session/history database, message broker, artifact content service, or global history index.
- Replacing unified snapshot storage, checkpoint manifests, realtime cursor semantics, tool lifecycle semantics, or existing recovery owners.
- Re-executing tools/providers during replay, automatically resolving branch conflicts, or exposing a hosted API.
- Defining authorization, RBAC, multi-tenant isolation, or product-level conversation retention policy.

## Decisions

### 1. Add a reference-only history boundary instead of a second store

Introduce a bounded projection containing session identifier, history root/leaf references, parent message/event references, producing Run/Step correlation, and branch/fork metadata. Message bodies remain owned by the existing session/history source. This keeps the protocol embeddable and avoids competing persistence facts.

Alternative considered: copy complete message trees into protocol snapshots. Rejected because it duplicates state, expands payloads without a bounded contract, and makes restore ownership ambiguous.

### 2. Treat branch/fork as a new Run lineage, not a mutation of a terminal Run

A fork references an existing history leaf and parent Run/Checkpoint, then creates a distinct Run correlation. The original Run and checkpoint remain immutable. Branch validation requires a resolvable parent and monotonic history position; conflicts are reported rather than silently merged.

Alternative considered: mutate the existing Run to point at a new leaf. Rejected because it breaks terminal idempotency and makes replay non-deterministic.

### 3. Keep restore source-owned and validate before mutation

History/checkpoint validation runs before snapshot restore mutation. `strict` rejects incompatible schema, broken lineage, or unresolved required references. `compatible` may proceed only within the existing compatibility window and records a bounded downgrade classification. Repeated imports with the same operation identity are idempotent.

Alternative considered: let the protocol layer perform restore and conflict resolution. Rejected because snapshot/composer owners already define storage and restore semantics, and a second restore state machine would violate the architecture boundary.

### 4. Make replay a deterministic read-only projection

The replay fixture supplies normalized history, checkpoint, restore, and expected terminal data. Replay validates schema, lineage, conflict taxonomy, and Run/Stream parity without provider/tool execution, workspace mutation, or diagnostics side effects. Replay identity and normalized digest detect duplicate/conflicting inputs.

Alternative considered: replay by re-running the runtime. Rejected because it would introduce nondeterministic external effects and conflate diagnostic replay with business recovery.

### 5. Extend existing contracts additively

The new capability gets its own spec. Existing protocol, snapshot, provenance, and diagnostics-replay specs receive only the requirement deltas needed to reference history and replay boundaries. All new fields carry an explicit profile/fixture version and preserve historical fixture parsing defaults.

Alternative considered: create a replacement protocol version with breaking schemas. Rejected because current consumers depend on additive compatibility and the repository's 0.x governance requires explicit migration for incompatible inputs.

## Risks / Trade-offs

- [Risk] Different source runtimes expose different amounts of message history → Mitigation: require only bounded references and classify unavailable optional data as omitted; required lineage gaps fail deterministically.
- [Risk] Branch metadata could be mistaken for an authorization decision → Mitigation: keep authorization in existing policy/HITL owners and label branch/fork as source-owned lineage outcomes only.
- [Risk] Compatible restore could hide meaningful drift → Mitigation: restrict it to the existing compatibility window, emit a bounded downgrade classification, and retain strict mode for validation-sensitive hosts.
- [Risk] Replay fixtures may grow unbounded → Mitigation: use reference/digest fields, size limits, canonical normalization, and explicit fixture validation before processing.
- [Risk] Run/Stream mapping diverges on branch or restore outcomes → Mitigation: require normalized parity fixtures and a blocking shell/PowerShell gate.

## Migration Plan

1. Add the new reference-oriented types, validation rules, and capability spec without changing existing manifests or stores.
2. Add protocol/snapshot/provenance/replay adapters that default new fields to absent and preserve historical fixture behavior.
3. Add canonical valid, negative, and drift fixtures, integration tests, and parity gates.
4. Add a documented session/checkpoint branch example with side-effect-free offline replay assertions.
5. Run the full Go, race, lint, quality, docs, and OpenSpec validation suites before enabling the contract for consumers.

Rollback is additive: disable the new projection/gate integration and ignore the new optional fields. Existing snapshot restore, runtime execution, and historical replay paths remain unchanged. Inputs that require the new contract continue to fail fast rather than mutating source state.

## Open Questions

- Which existing embedded session/history owner will provide the first concrete history-leaf adapter? The contract should remain owner-neutral until an implementation path is selected.
- Should the first fixture carry message identifiers only, or also bounded content digests? Recommendation: identifiers plus optional digests, never full message bodies.
- Which retention/compaction policy applies to history references? This proposal intentionally leaves retention to the source owner; only reference validity and replay safety are standardized.

## Example Impact Assessment

新增示例

Add a documented session/checkpoint branch and offline replay example; no existing example semantics are changed.
