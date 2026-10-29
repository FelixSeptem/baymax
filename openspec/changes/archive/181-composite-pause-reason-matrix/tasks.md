## 1. Contract and evidence model

- [x] 1.1 Add the additive per-pause-reason capability DTO/normalization path in `core/types`; preserve legacy complete-source decoding and reject malformed/ambiguous evidence.
- [x] 1.2 Add table-driven validation tests for complete, partial, missing, package-only, foreign, and bounded evidence.
- [x] 1.3 Define fixture keys and stable unsupported reasons for form/reason cells, including mailbox lifecycle-only limitation.

## 2. Manifest and admission

- [x] 2.1 Change `orchestration/planbridge` manifest promotion to copy only supported cells from the matching owner and form.
- [x] 2.2 Add compile/admission checks for requested pause reason, owner namespace, capability/API version, deployment scope, and stale evidence.
- [x] 2.3 Add negative tests proving sibling owner evidence, DTO presence, and legacy pause lists cannot promote a cell.
- [x] 2.4 Add mailbox-specific rejection tests proving lifecycle continuation does not advertise handler execution or context restore.

## 3. Owner conformance and parity

- [x] 3.1 Audit runner, child, team, workflow, and mailbox owner capability projections and mark only cells backed by source safe-point evidence.
- [x] 3.2 Add positive tests for supported input/approval/budget cells, including pending state, consumed budget/deadline, deny/timeout terminal evidence, and completed-effect preservation.
- [x] 3.3 Add duplicate/conflicting resume, cancellation, timeout, retry, lease-loss, and terminal persistence tests for applicable owners.
- [x] 3.4 Add Run/Stream parity tests for supported and rejected cells.

## 4. Fixtures, docs, and gates

- [x] 4.1 Add versioned manifest/replay fixtures for partial matrix, lifecycle-only mailbox, sibling mismatch, and exact owner evidence.
- [x] 4.2 Add `scripts/check-composite-pause-reason-matrix-contract.ps1` and `.sh`, and wire them into the quality/mainline contract checks.
- [x] 4.3 Update `README.md`, roadmap, integration guide, runtime diagnostics/module-boundary docs, contract index, planbridge/mailbox READMEs, and host-plan example.
- [x] 4.4 Run focused tests, `go test ./...`, race tests, lint, OpenSpec validation, documentation/example/roadmap gates, quality gate, and `git diff --check`.

## Documentation Impact Assessment

| Area | Outcome | Maintenance docs | Owner | Verification evidence |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | runtime architecture | owner-boundary checks |
| components | 修改文档 | `orchestration/planbridge/README.md`, `orchestration/mailbox/README.md` | planbridge/mailbox owners | capability and lifecycle tests |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime architecture | capability semantics and gate |
| contract/API | 修改文档 | `docs/guides/integration.md`, `docs/mainline-contract-test-index.md` | integration/contracts | matrix fixtures and gate |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md` | diagnostics owner | typed outcome checks |
| examples | 修改文档 | `examples/host-execution-plan/README.md` | examples owner | matrix example |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | integration docs | Run/Stream parity |
| best practices | 修改文档 | `docs/guides/integration.md` | integration docs | fail-closed guidance |
| roadmap | 修改文档 | `docs/development-roadmap.md` | runtime architecture | roadmap gate |

## Example Impact Assessment

- 修改示例：the host-plan example and manifest fixtures must show supported and rejected matrix cells.

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`, `docs/development-roadmap.md`, `docs/guides/integration.md`, `docs/runtime-config-diagnostics.md`, `docs/runtime-module-boundaries.md`, `docs/mainline-contract-test-index.md`, `orchestration/planbridge/README.md`, `orchestration/mailbox/README.md`, `examples/host-execution-plan/README.md`
- 维护 owner：runtime architecture、planbridge、mailbox、integration、diagnostics、contracts、examples
- 验证证据：`openspec validate --all`, `pwsh -File scripts/check-openspec-documentation-impact.ps1`, `pwsh -File scripts/check-openspec-example-impact-declaration.ps1`, `pwsh -File scripts/check-openspec-roadmap-status-consistency.ps1`, `pwsh -File scripts/check-docs-consistency.ps1`
