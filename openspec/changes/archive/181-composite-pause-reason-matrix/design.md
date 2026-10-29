## Context

`core/types.ContinuationCapabilities` currently requires all three pause reasons and exactly five evidence references. `orchestration/planbridge` then promotes all continuation cells for a form after source-level validation. That shape cannot represent a supported `input_required` cell with an unsupported `approval_required` cell, nor can it distinguish mailbox lifecycle projection from handler execution continuation.

## Goals / Non-Goals

**Goals:**

- Add a backward-compatible, detached per-reason evidence representation.
- Validate and promote each form/reason cell independently while preserving deployment and owner namespace checks.
- Make unsupported and unavailable admission outcomes deterministic and pre-effect.
- Preserve source-owned restore, terminal, identity, and Run/Stream semantics.
- Add evidence that sibling owner capabilities cannot promote a requested form and that mailbox lifecycle-only support cannot promote handler continuation.

**Non-Goals:**

- No new checkpoint store, graph restore material, scheduler, queue, mailbox owner, or execution loop.
- No multi-writer or cross-machine deployment expansion.
- No mailbox handler safe point or resumed-trigger planbridge implementation.
- No conversion of history replay, lifecycle projection, or application snapshots into execution state.

## Decisions

1. Extend the public continuation capability model with a `PauseReasonCapability`/equivalent map keyed by the three canonical reasons. Each entry carries `Supported`, an evidence reference when supported, and a bounded unsupported reason when not. Existing complete sources remain readable through a compatibility normalization path.
2. Make validation require an explicit entry for every canonical reason, but allow each entry to be independently supported or unsupported. A supported entry requires source evidence; an unsupported entry requires a stable reason. `PauseReasons` and legacy evidence cannot by themselves promote a cell.
3. Change manifest promotion to copy only supported cells from the configured owner. The requested execution form and owner namespace must match; a source error or missing evidence yields typed unavailable/unsupported before activation.
4. Keep mailbox lifecycle continuation in its own dimension. `HandlerExecutionSafePoint` is a mailbox-only owner fact; a mailbox owner may report lifecycle projection/checkpoint facts, but unless it also reports a handler execution safe point, `input_required`, `approval_required`, and `budget_exhausted` handler cells remain unsupported and context assembly remains unsupported. The stable handler pause limitation is `mailbox_handler_safe_point_unsupported`.
5. Validate the matrix at plan compile/admission and again at resume admission so stale or mismatched manifests fail closed. Run and Stream share the same owner decision and terminal result; Stream only adds event delivery.
6. Use versioned fixtures and a dedicated gate to assert positive cells, missing evidence, sibling-owner mismatch, lifecycle-only mailbox rejection, duplicate resume behavior, and no-effect rejection.

## Data Flow

1. An owner returns detached continuation capabilities with deployment scope and per-reason evidence.
2. The manifest builder validates schema/version/owner namespace and projects only the owner form's supported cells.
3. Plan compilation records the requested form and pause reason, but performs no source activation.
4. Admission checks the exact manifest cell, binding identity, and deployment scope. Unsupported/unavailable errors stop before Run, Stream, publish, enqueue, or checkpoint claim.
5. A supported request delegates to the existing owner restore path. The owner remains responsible for checkpoint claim, source state, terminal evidence, idempotency, and completed-effect suppression.

## Compatibility and Rollback

The public JSON shape is additive: legacy complete sources normalize to three supported entries. Existing callers that only request direct continuation retain behavior. Callers requesting a form/reason without evidence receive the existing typed rejection instead of a false promotion. Rollback is to stop emitting the new matrix evidence and keep all affected cells unsupported; no source state or checkpoint is rewritten.

## Documentation Impact Assessment

| Area | Outcome | Maintenance docs | Owner | Verification evidence |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | runtime architecture | owner-boundary checks |
| components | 修改文档 | `orchestration/planbridge/README.md`, `orchestration/mailbox/README.md` | planbridge/mailbox | package contract tests |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime architecture | capability matrix semantics documented; no new keys |
| contract/API | 修改文档 | `docs/guides/integration.md`, `docs/mainline-contract-test-index.md` | contracts | matrix fixtures/gate |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md` | diagnostics | typed outcome checks |
| examples | 修改文档 | `examples/host-execution-plan/README.md` | examples | offline matrix example |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | integration | Run/Stream parity |
| best practices | 修改文档 | `docs/guides/integration.md` | integration | fail-closed guidance |
| roadmap | 修改文档 | `docs/development-roadmap.md` | runtime architecture | roadmap gate |

## Example Impact Assessment

- 修改示例

The example documents the explicit supported cell and rejected lifecycle-only mailbox case.

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`, `docs/development-roadmap.md`, `docs/guides/integration.md`, `docs/runtime-config-diagnostics.md`, `docs/runtime-module-boundaries.md`, `docs/mainline-contract-test-index.md`, `orchestration/planbridge/README.md`, `orchestration/mailbox/README.md`, `examples/host-execution-plan/README.md`
- 维护 owner：runtime architecture、planbridge、mailbox、integration、diagnostics、contracts、examples
- 验证证据：`openspec validate --all`, `pwsh -File scripts/check-openspec-documentation-impact.ps1`, `pwsh -File scripts/check-openspec-example-impact-declaration.ps1`, `pwsh -File scripts/check-openspec-roadmap-status-consistency.ps1`, `pwsh -File scripts/check-docs-consistency.ps1`
