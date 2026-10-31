## Context

`Engine.ResumeContinuation` executes the restored graph. If execution pauses again, `persistContinuation` calls `WorkflowContinuationStore.Save` for the successor; only after execution returns does the engine call `CommitResumeResult` for the predecessor. The file store persists each record independently. After a crash in that interval, reopen classifies the predecessor's running claim as recovery-required while the successor record has no committed predecessor projection that exposes its reference.

The runtime supports same-host, single-writer sequential reopen only. The fix must preserve that declared scope and cannot claim a transaction with scheduler state or another source owner.

## Goals / Non-Goals

**Goals:**

- Atomically persist the predecessor claim/result and successor checkpoint in the workflow continuation store.
- Validate successor identity and causal lineage against the claimed predecessor before publishing either record.
- Ensure duplicates return the same retained result and successor reference after reopen.
- Ensure a failed commit leaves neither a published successor nor a completed predecessor projection.

**Non-Goals:**

- Multi-writer, cross-machine or distributed transactions.
- Automatic recovery by rerunning a workflow after an uncertain dispatch.
- Changes to child, team, mailbox, or planbridge continuation support.

## Decisions

- Add a successor-aware owner-store commit operation, keeping the existing no-successor commit path available for terminal resumes. The engine uses the successor-aware operation whenever the returned resume result carries a successor checkpoint.
- Add an optional embedded successor continuation to the predecessor's file record. The file store validates both and atomically replaces only the predecessor record with its completed claim/result and successor material together. When a caller later loads the successor ref, the store resolves it from that committed predecessor record and indexes it as the next owner record; subsequent generations repeat the same transition.
- The memory store applies predecessor claim/result and successor record under the same lock after validating both. On persistence failure, neither the predecessor projection nor the in-memory successor remains published.
- The predecessor's retained result is the only admission path to the successor. Successor validation checks owner namespace, original graph/run/session/causal/authority bindings, immediate predecessor ID, and next generation.
- The owner persists an incomplete running claim before dispatch as it does today. A crash before the atomic completion transition remains `recovery-required`; this change does not claim to resolve uncertain external effects.

Alternatives considered:

- Keep separate writes and add orphan scanning: rejected because it cannot prove which claim owns an orphan or safely recover execution state.
- Re-run the workflow to reconstruct a successor: rejected because completed effects may repeat and it violates source-owned restore semantics.

## Risks / Trade-offs

- [File snapshot publication fails] → Return typed unavailable/recovery-required and preserve the last committed snapshot; do not dispatch or retry execution.
- [Older/custom stores lack successor commit] → A resumed workflow that pauses again fails closed before exposing the successor as committed; the capability must not silently use two writes.
- [Process crashes during atomic replace] → Reopen observes either the previous committed snapshot or the complete successor transition, never a mixed projection.

## Migration Plan

The file record envelope gains an optional successor material field; older records without it remain readable. Rollback keeps the additive field readable and must not restore the prior two-write behavior. Stores that cannot implement the successor-aware commit fail closed when a resumed workflow pauses again.

## Example Impact Assessment

无需示例变更（附理由）：this changes owner-store publication semantics only; no sample configuration or execution flow changes.

## Documentation Impact Assessment

| Area | Outcome | Maintainer | Verification | Evidence |
| --- | --- | --- | --- | --- |
| architecture | 无需文档变更（附理由） | — | — | No module boundary changes. |
| components | 修改文档 | workflow owner | `go test ./orchestration/workflow -count=1` | `orchestration/workflow/README.md` records atomic publication. |
| configuration | 修改文档 | runtime configuration owner | `pwsh -File scripts/check-docs-consistency.ps1` | `docs/runtime-config-diagnostics.md` records that no new key is introduced. |
| contract/API | 修改文档 | contract-test index owner | `pwsh -File scripts/check-quality-gate.ps1` | `docs/mainline-contract-test-index.md` maps the new gate. |
| diagnostics | 修改文档 | runtime diagnostics owner | `pwsh -File scripts/check-docs-consistency.ps1` | `docs/runtime-config-diagnostics.md` records that no diagnostic field is introduced. |
| examples | 修改文档 | native-events example owner | `go test ./orchestration/workflow -count=1` | `examples/native-execution-events/README.md` states the example remains outside owner commit. |
| CLI/integration | 无需文档变更（附理由） | — | — | No CLI or integration surface changes. |
| best practices | 无需文档变更（附理由） | — | — | No usage guidance changes beyond the workflow owner README. |
| roadmap | 修改文档 | roadmap maintainer | `pwsh -File scripts/check-docs-consistency.ps1` | `docs/development-roadmap.md` records the audited implementation direction. |

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`orchestration/workflow/README.md`, `docs/development-roadmap.md`, `docs/mainline-contract-test-index.md`, `docs/runtime-config-diagnostics.md`, `examples/native-execution-events/README.md`
- 维护 owner：workflow owner, roadmap maintainer, contract-test index owner, runtime configuration/diagnostics owner, native-events example owner
- 验证证据：`go test ./orchestration/workflow -count=1`; `pwsh -File scripts/check-docs-consistency.ps1`; `pwsh -File scripts/check-quality-gate.ps1`
