## Why

Baymax already has source-owned continuation implementations for selected direct, child, team, workflow, and mailbox paths, but the public capability contract still treats the three pause reasons as a single source-level set. `planbridge` can therefore promote all three cells when a source only exposes a DTO or lifecycle projection, and mailbox lifecycle continuation can be mistaken for host-handler execution restore. Danny's planner needs a capability decision that is independently true for each execution form and pause reason before admission.

## What Changes

- Replace all-or-nothing pause-reason promotion with an explicit per-form, per-reason evidence matrix.
- Require source-owned producer, safe-point material, durable checkpoint/restore, race handling, terminal evidence, and Run/Stream parity before a cell is `supported`.
- Keep missing or mismatched cells as typed `unsupported_capability` or `unavailable` before source activation; never borrow evidence from another form or fall back to a new Run, enqueue, retry, or callback replay.
- Distinguish mailbox lifecycle continuation/projection from host-handler execution continuation; lifecycle-only evidence cannot promote handler restore or host context safe points.
- Preserve the existing owner APIs and deployment scope. This change does not add a scheduler, queue, mailbox owner, checkpoint store, graph restore engine, or host-side resume loop.
- Add contract fixtures, negative admission tests, Run/Stream parity tests, and a dedicated cross-form pause-matrix gate.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `composite-continuation-restore`: make pause-reason support independently evidenced by execution form and define lifecycle-only mailbox rejection.
- `host-runtime-capability-manifest`: publish and validate per-reason evidence and stable unsupported reasons without inferring support from package/API presence.
- `host-execution-plan-bridge`: gate continuation admission on the requested form/reason cell and matching owner evidence.

## Impact

- Public Go capability DTOs and validation in `core/types`.
- Capability manifest construction and continuation admission in `orchestration/planbridge`.
- Owner capability projections/tests for runner, child, team, workflow, and mailbox; no new owner is introduced.
- Integration and package documentation, capability fixtures, replay/contract tests, and shell/PowerShell quality gates.

## Documentation Impact Assessment

| Area | Outcome | Maintenance docs | Owner | Verification evidence |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | runtime architecture | dependency and owner-boundary checks |
| components | 修改文档 | `orchestration/planbridge/README.md`, `orchestration/mailbox/README.md` | planbridge/mailbox owners | capability and lifecycle boundary tests |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime architecture | capability matrix semantics documented; no new configuration keys or defaults |
| contract/API | 修改文档 | `docs/guides/integration.md`, `docs/mainline-contract-test-index.md` | integration/contracts | public matrix fixture and gate |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md` | diagnostics owner | typed unsupported/unavailable evidence |
| examples | 修改文档 | `examples/host-execution-plan/README.md` | examples owner | positive and rejected matrix examples |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | integration docs | Run/Stream admission parity |
| best practices | 修改文档 | `docs/guides/integration.md` | integration docs | no fallback/replay guidance |
| roadmap | 修改文档 | `docs/development-roadmap.md` | runtime architecture | roadmap status gate |

## Example Impact Assessment

- 修改示例

The host execution plan example must show an owner-bound supported cell and a lifecycle-only mailbox rejection; it remains offline and provider-free.

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`, `docs/development-roadmap.md`, `docs/guides/integration.md`, `docs/runtime-config-diagnostics.md`, `docs/runtime-module-boundaries.md`, `docs/mainline-contract-test-index.md`, `orchestration/planbridge/README.md`, `orchestration/mailbox/README.md`, `examples/host-execution-plan/README.md`
- 维护 owner：runtime architecture、planbridge、mailbox、integration、diagnostics、contracts、examples
- 验证证据：`openspec validate --all`, `pwsh -File scripts/check-openspec-documentation-impact.ps1`, `pwsh -File scripts/check-openspec-example-impact-declaration.ps1`, `pwsh -File scripts/check-openspec-roadmap-status-consistency.ps1`, `pwsh -File scripts/check-docs-consistency.ps1`
