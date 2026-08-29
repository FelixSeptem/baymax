## Context

The archived Agent Runtime Protocol exposes a reference-only `CheckpointRef`, while the unified snapshot contract owns manifest schema, digest validation, strict/compatible restore, and idempotent import. Source runtimes also own workspace and sandbox side effects. Hosts need a bounded projection of checkpoint history, lineage, branch/replay identity, and workspace change-set integrity to audit long-running and cross-session recovery without introducing a second repository.

## Example Impact Assessment

新增示例

The checkpoint/workspace provenance example will be documented in `examples/agent-modes/MATRIX.md` and its mode README before executable smoke coverage is added.

## Goals / Non-Goals

**Goals:**

- Add nullable, reference-only checkpoint lineage/history/branch/replay fields.
- Add bounded workspace change-set and before/after integrity provenance with Run/Step/Checkpoint correlation.
- Validate lineage continuity, branch conflicts, replay idempotency, schema compatibility, and workspace drift deterministically.
- Reuse snapshot manifest and restore owners, preserve strict/compatible semantics, and keep Run/Stream projections equivalent.
- Extend diagnostics, OTel, replay fixtures, gates, and documentation through existing ownership boundaries.

**Non-Goals:**

- No second state/checkpoint repository, workspace filesystem, artifact content service, ACL, garbage collector, hosted control plane, or global queue.
- No mutation of snapshot manifests, restore state, workspace files, sandbox decisions, or source scheduler state by protocol mapping.
- No embedded workspace contents, prompts, provider objects, or unbounded metadata.

## Decisions

### 1. Extend the protocol projection, not the storage owner

Add optional lineage fields to `CheckpointRef` and a `WorkspaceProvenance` reference in `core/types`. Snapshot adapters provide pure manifest-to-reference helpers and accept caller-supplied recovery context. The manifest remains the sole fact source. A snapshot-store history API was rejected because it would widen storage ownership and create a competing repository contract.

### 2. Use explicit finite relationship and restore vocabularies

Checkpoint relation values are `root`, `derived`, `branch`, and `replay`; restore sources are `fresh`, `resume`, and `cross_session`. Validators reject missing parents for derived/branch/replay references, branch conflicts, disconnected history, schema incompatibility, and inconsistent Run/Step/Session associations. Free-form relationship strings were rejected because replay could not classify drift deterministically.

### 3. Treat workspace provenance as integrity references

`WorkspaceProvenance` carries workspace/change-set identifiers, before/after integrity references, producing Run/Step, and optional checkpoint correlation. It never copies content or makes policy decisions. A digest mismatch is classified as `workspace.integrity_drift` and returned to the source policy owner. A richer workspace service or ACL model is explicitly deferred.

### 4. Make replay identity idempotent

`replay_key` identifies a logical replay projection. Repeating the same key with equivalent normalized data returns the same reference; conflicting data fails with `checkpoint.replay_conflict`. History is validated by ordered index and parent links, with cycles and gaps rejected before source mutation.

### 5. Preserve restore, Run/Stream, and observability contracts

Strict and compatible restore continue to execute in snapshot/composer owners. Protocol helpers are side-effect free and shared by Run and Stream. New diagnostics and OTel fields are nullable, bounded, low-cardinality, and written through `RuntimeRecorder`; workspace content and full digest lists are excluded.

## Risks / Trade-offs

- [Callers provide incomplete history] -> Require explicit root/parent semantics and deterministic `checkpoint.history_disconnected` validation.
- [Provenance is mistaken for workspace authorization] -> Document it as an integrity reference and route policy decisions to existing sandbox/readiness owners.
- [Digest cardinality leaks into telemetry] -> Bound identifiers, keep OTel to state/reason values, and omit raw integrity values from high-cardinality attributes.
- [Run and Stream projections diverge] -> Use one normalization helper and parity fixtures for every restore/branch/replay outcome.
- [Schema evolution breaks old consumers] -> Keep all additions optional/defaultable and preserve existing `agent_runtime_protocol.v1` fixtures unchanged.

## Migration Plan

1. Add DTOs, enums, validators, and pure snapshot projection helpers with empty optional fields for existing callers.
2. Add source-owned recovery/workspace context adapters without changing manifest serialization or restore behavior.
3. Add replay fixtures, diagnostics/OTel nullable correlation, and shell/PowerShell contract gates.
4. Update the example documentation baseline before executable smoke assertions, then update repository docs and roadmap.
5. Roll back by disabling provenance projection and omitting optional fields; existing checkpoint, snapshot, sandbox, and Run/Stream paths remain unchanged.

## Open Questions

- Which existing source adapters can supply a stable workspace identifier in the first implementation; the default may be an omitted optional field.
- Whether a future profile needs bounded history pagination; this proposal intentionally leaves history as caller-provided references.
