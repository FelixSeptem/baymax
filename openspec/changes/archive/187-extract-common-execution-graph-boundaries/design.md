## Context

See `proposal.md` for the motivation. The current owners have intentionally different execution models: workflow owns an arbitrary dependency DAG, teams own strategy-specific task membership, and composer owns individual child dispatch and its scheduler lifecycle. Their continuation records also bind owner-specific digests. The shared layer must expose topology facts without becoming a new execution or restore owner.

## Goals / Non-Goals

**Goals:**

- Provide a versioned, bounded, deterministic topology compiler usable by all three existing owners.
- Project workflow DAGs, team strategy ordering, and one composer child dispatch into that shared form.
- Give callers stable normalized nodes, deterministic topological order, and a topology-only SHA-256 digest suitable for comparison and contract evidence.
- Keep owner-specific validation, digests, effects, execution ordering where already defined, continuation stores, claims, restore, and terminal outcomes authoritative.

**Non-Goals:**

- A generic graph executor, scheduler, retry engine, cross-owner state store, checkpoint format, or terminal arbiter.
- Making composer child dispatch represent or restore an entire parent fanout graph.
- Replacing owner-specific plan/checkpoint digests or changing their persisted schema and restore semantics.
- Claiming a global event order across parallel members or source forms.

## Decisions

### Shared topology package

Add `orchestration/graph` with standard-library-only `Definition`, `Node`, `Compiled`, `Identity`, and typed validation errors. `Definition` carries `GraphID`, `Version`, and nodes with `ID` plus `DependsOn`. Compilation trims identifiers, sorts nodes and dependency lists, normalizes duplicate dependency declarations, validates identity and structural bounds, rejects missing/self/cyclic edges, and computes a deterministic Kahn topological order with lexicographic tie-breaking. The topology digest covers only the normalized graph identity, node IDs, and dependency edges; it excludes task payloads, strategy policy, callbacks, authority, and owner checkpoint material.

Choose one shared compiler instead of three independently maintained graph validators. Keep owner-specific field validation and translate common structural errors into each owner's existing stable error taxonomy so current callers retain their error contract.

### Owner-local adapters

Workflow projects its fully normalized/expanded definition into nodes and dependency edges; workflow retains its condition/retry/timeout checks and remains the only owner of arbitrary workflow DAG execution. Team projects serial strategy as an ordered chain and parallel/vote strategies as independent nodes; the team engine retains aggregation, failure, and vote behavior. Composer projects one child dispatch as a single-node graph and explicitly does not infer sibling nodes or parent topology.

Adapters live in their owner packages and depend on `orchestration/graph`; the shared graph package has no imports from owner packages. This avoids cycles and keeps source interpretation with each owner.

### Digest and compatibility boundary

Expose `TopologyDigest` separately from existing `GraphDigest` and owner plan digests. Do not substitute it into continuation source bindings or checkpoint claim requests. Existing checkpoint bytes and their schema versions remain valid; the new digest is additive projection evidence only. A caller changing topology must receive a different topology digest, while an execution payload-only change may retain the same topology digest and still be detected by the owner's existing execution digest.

### Bounds and error behavior

Use documented fixed bounds for graph ID/version/node IDs, total nodes, total edges, and encoded canonical bytes. Return typed errors with stable codes and field/node context; do not return a partial `Compiled` value on failure. Owner adapters preserve their existing preflight-before-effect boundary and return before dispatch on projection error.

### Verification and documentation

Use unit tables for valid/invalid topology, owner tests for strategy-to-edge mapping, and a cross-form contract fixture proving equivalent workflow/team (and single-node child) projections have identical topology digests. Add a dedicated deterministic gate and a small offline example. The example must not execute a model, tool, scheduler, or owner restore. Run/Stream behavior is checked through existing owner tests plus projection parity; no new terminal semantics are introduced.

## Risks / Trade-offs

- [Topology digest could be mistaken for execution authorization or checkpoint integrity] -> Use a distinct field/type and document that all authority and restore validation remains owner-specific; include a negative restore-binding test.
- [Workflow error ordering could drift when validation is extracted] -> Preserve workflow-specific checks and map graph errors deterministically; add compatibility assertions for existing malformed fixtures.
- [Team serial input order is semantically meaningful while parallel/vote order is not] -> Encode serial order as dependency edges; canonicalize independent nodes without changing team execution input order.
- [Composer callers may treat a child graph as the full parent graph] -> Name the projection as child-scoped, bind only one node, and test that it never synthesizes sibling edges.
- [Changing owner continuation digests could invalidate persisted checkpoints] -> Do not modify current owner digests or checkpoint schemas; treat topology digest as additive only.

## Migration Plan

The new projection APIs are additive. Existing Run/Stream calls and persisted checkpoints continue using current owner contracts. Hosts may adopt topology projection for comparison or diagnostics; rollback consists of ceasing to call the new projection API, with no data migration or checkpoint rewrite required.

## Example Impact Assessment

新增示例：`examples/execution-graph` 展示 workflow、team、single-child projection 的规范化结果与 topology digest，并断言 topology digest 不可替代 source checkpoint/authority 校验。

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/README.md`、`docs/guides/execution-graph.md`、`docs/runtime-module-boundaries.md`、`docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`docs/development-roadmap.md`、`examples/execution-graph/README.md`
- 维护 owner：`orchestration/graph` maintainers、runtime contract maintainers、documentation maintainers
- 验证证据：`pwsh -File scripts/check-execution-graph-contract.ps1`、`pwsh -File scripts/check-docs-consistency.ps1`、`openspec validate --all`

## Documentation Impact Assessment

| area | outcome | affected paths | owner | evidence |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | runtime contract maintainers | source-owner boundary review and docs consistency gate |
| components | 修改文档 | `docs/README.md`、`docs/guides/execution-graph.md` | orchestration and documentation maintainers | owner projection contract tests and documentation link validation |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime contract maintainers | explicitly documents that this contract adds no configuration keys or reload semantics |
| contract/API | 修改文档 | `docs/guides/execution-graph.md` | orchestration maintainers | graph package and cross-form contract gate |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime contract maintainers | explicitly documents that this contract adds no diagnostics or RuntimeRecorder fields |
| examples | 新增文档 | `examples/execution-graph/README.md` | examples maintainers | offline example marker test |
| CLI/integration | 无需文档变更（附理由） | — | — | no CLI, wire command, or transport integration changes |
| best practices | 修改文档 | `README.md` | documentation maintainers | active milestone snapshot and public capability boundary |
| roadmap | 修改文档 | `docs/development-roadmap.md` | documentation maintainers | roadmap status consistency gate |
