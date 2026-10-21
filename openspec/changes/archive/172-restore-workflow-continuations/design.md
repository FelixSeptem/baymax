## Context

See `proposal.md` for motivation and public scope. Direct native continuation already owns a bounded checkpoint envelope and atomic resume claim, while workflow owns deterministic graph expansion, dependency scheduling and existing minimal checkpoint semantics. The design extends the workflow source owner without creating a second executor, queue, journal, mailbox owner or business-state store. The first deployment profile remains same-host single-writer with sequential reopen.

## Goals / Non-Goals

**Goals:**

- Persist enough source-owned workflow graph state to restore the same native graph after a new Engine opens the configured store.
- Make the three pause reasons observable at real workflow safe points, with duplicate convergence, fail-closed binding validation and Run/Stream parity.
- Keep capability cells and replay evidence honest, especially while child/team/mailbox/planbridge forms remain unsupported.
- Preserve existing deterministic expanded-graph and A2A scheduling semantics without rerunning terminal steps or effects.

**Non-Goals:**

- No distributed or multi-writer store, cross-machine fencing, global event journal, hosted control plane or unified composite executor.
- No restore from callback history, Danny projection, mailbox lifecycle history, application checkpoint, reused Run ID or a fresh parent Run.
- No child/fanout, team parallel/vote, mailbox host-plan or planbridge resumed support in this change.
- No new runtime configuration keys, business authorization owner, budget ledger, provider dependency or direct diagnostics writer.

## Decisions

### Source-owner boundary

Workflow remains authoritative for graph state, step lifecycle, dependency readiness, retry/timeout facts, pause evidence and aggregate terminal arbitration. The generic checkpoint DTO carries bounded references and digests; workflow source material carries the normalized graph continuation required to resume. Planbridge, history replay and host callbacks can validate or project facts but cannot construct continuation state.

### Graph continuation envelope

The persisted envelope contains graph ID/version/digest; parent/run/session/host correlation and causation; canonical expanded step IDs and dependency completion; pending and completed step outcomes; per-step consumed budget and original deadline; pending action and completed effect references; pause reason/boundary; terminal arbitration evidence; checkpoint generation and predecessor/successor relation. Save is immutable; successor pause and predecessor completion are committed as one source transition. Payloads, credentials and arbitrary callbacks remain outside the public reference projection.

### Pause safe points

`input_required` is emitted after source-owned completed outcomes and before the next eligible model/action boundary. `approval_required` is emitted before dispatch of the pending high-risk effect and retains the action plus prior outcomes. `budget_exhausted` is emitted at a source-owned resource boundary and retains consumed counters and the original deadline. Deny/timeout is a source terminal decision, not a host projection. Each cell is advertised only after producer, restore, race and parity fixtures pass.

### Restore and claim state machine

Admission loads and validates schema, digest, graph binding, authority/plan/scope/effect namespace, expiry, ownership, current revision/budget facts and predecessor relation. Host revalidation runs outside source locks; the source generation and pause state are checked again before an atomic claim. Equivalent normalized resume identity converges to the retained claim/result projection. Conflicting, foreign, stale, expired, corrupt or already-running requests return typed failure or recovery-required and never reclaim or start another continuation.

### Run/Stream and scheduler parity

Run and Stream invoke the same workflow source claim and graph continuation path; only transport projection differs. Restored scheduling uses the original normalized expanded graph and deterministic ready-step ordering. Completed steps, retries, timeouts, A2A results and effects are reused by reference. Cancellation, timeout, retry and resume precedence is decided from source generation, not request arrival order or replay order.

### Capability and storage advertisement

The manifest has independent workflow cells for checkpoint creation, cross-process restore and each pause reason, with contract/API version, build revision, evidence reference, retention/expiry and deployment limitation. Other composite cells remain explicit unsupported. A file-backed store advertises only same-host single-writer sequential reopen; lock, sync, checksum, version or availability failure is typed unavailable and cannot downgrade to memory, latest replay or a new Run.

### Replay and gates

Replay consumes versioned, bounded, reference-only workflow fixtures. It checks graph/identity drift, duplicate and conflict convergence, crash windows, terminal evidence, unsupported cells and Run/Stream parity without constructing a workflow owner or invoking any execution dependency. Dedicated shell and PowerShell gates share the same fixtures and taxonomy; replay success never advertises runtime restore capability.

## Risks / Trade-offs

- [Incomplete graph material] -> Require source-owned schema/digest and reject a checkpoint that cannot reconstruct dependency and pending-action state.
- [Duplicate effects after restart] -> Persist completed effect references and gate scheduling on terminal step state before dispatch.
- [False capability advertisement] -> Generate workflow cells only from passing source-process fixtures; leave every other form unsupported.
- [Crash after claim] -> Retain attempt/generation and project `in-progress` or `recovery-required`; never auto-retry or create a new graph.
- [Sensitive checkpoint data] -> Bound and detach public references, route diagnostics through `RuntimeRecorder`, and keep business payloads out of replay fixtures.

## Migration Plan

1. Add workflow continuation DTOs, source transitions and tests behind the existing constructor-injected store boundary.
2. Add producer/restore evidence for each pause reason and enable only the verified manifest cells.
3. Update integration guidance and the existing durable continuation example to state workflow-first scope and unsupported forms.
4. Roll back by removing workflow capability advertisement and retaining persisted records for inspection; do not reinterpret them as fresh Runs.

## Open Questions

None. The source owner, pause matrix, storage scope, claim identity and unsupported boundaries are fixed for implementation.

## Example Impact Assessment

修改示例

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`docs/development-roadmap.md`、`docs/guides/integration.md`、`docs/runtime-module-boundaries.md`、`docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`orchestration/workflow/README.md`、`examples/durable-continuation-restore/README.md`
- 维护 owner：workflow/runtime、diagnostics/replay、documentation maintainers
- 验证证据：`openspec validate restore-workflow-continuations --strict`；`pwsh -File scripts/check-openspec-documentation-impact.ps1`；`pwsh -File scripts/check-docs-consistency.ps1`

## Documentation Impact Assessment

| area | outcome | affected paths | owner | evidence |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | workflow/runtime | source-owner boundary review |
| components | 修改文档 | `orchestration/workflow/README.md` | workflow/runtime | package contract review |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | workflow/runtime | configuration boundary review |
| contract/API | 修改文档 | `docs/guides/integration.md` | workflow/runtime | OpenSpec strict validation |
| diagnostics | 修改文档 | `docs/mainline-contract-test-index.md` | diagnostics/replay | replay gate mapping review |
| examples | 修改文档 | `examples/durable-continuation-restore/README.md` | documentation maintainers | expected-marker review |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | workflow/runtime | admission boundary review |
| best practices | 修改文档 | `docs/guides/integration.md` | documentation maintainers | restore safety review |
| roadmap | 修改文档 | `docs/development-roadmap.md` | documentation maintainers | roadmap status gate |
