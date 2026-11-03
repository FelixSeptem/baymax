## 1. Host resolver bounded admission

- [x] 1.1 Write focused failing `context/hostprojection` tests for exact role/content/aggregate limits, whole-resolution rejection, and no partial material; verify each new test fails for the missing material-bound behavior.
- [x] 1.2 Implement fixed all-or-nothing resolver material validation and a stable bounded classification without configuration, diagnostics, persistence, truncation, or owner-boundary changes; verify `go test ./context/hostprojection -count=1` and race coverage pass.
- [x] 1.3 Extend existing assembly/Run/Stream contract coverage for a material-bound rejection before provider/tool/effect execution; verify the focused assembler/runner suites and `scripts/check-host-context-assembly-handoff-contract.ps1` pass.

## 2. Model-surface request evidence

- [x] 2.1 Write failing `model/conformance` and replay tests for explicit configuration/prompt/tool-schema safe facts, unavailable defaults, invalid/oversize rejection, privacy, expected/observed drift, historical fixture compatibility, and Run/Stream parity; verify the tests fail because model-surface evidence is absent.
- [x] 2.2 Implement the bounded opt-in provider-neutral model-surface carrier, canonical validation/defaulting, stable drift classification, and projection comparison without deriving facts from roles, ToolOrder, capabilities, cache controls, or provider SDK values; verify `go test ./model/conformance ./tool/diagnosticsreplay -count=1` passes.
- [x] 2.3 Update provider audit fixtures and dual-platform request-projection gate for deterministic replay, privacy, overflow, drift, and parity evidence; verify `scripts/check-provider-request-projection-contract.ps1` passes and deliberate fixture drift is rejected.

## 3. Public documentation, quality, and delivery

- [x] 3.1 Synchronize `README.md`, `context/hostprojection/README.md`, `model/README.md`, `docs/runtime-module-boundaries.md`, `docs/runtime-config-diagnostics.md`, `docs/mainline-contract-test-index.md`, and `docs/development-roadmap.md` with exact bounds, opt-in/default behavior, privacy, owner boundaries, replay/gate evidence, rollback, and active status; verify documentation-impact, example-impact, and roadmap-status checks pass.
- [x] 3.2 Apply the two delta specs to their main capability specs using the repository OpenSpec sync workflow; verify `openspec validate --all` and the checked-in main specs carry the delivered requirements.
- [x] 3.3 Run focused suites, all required repository gates, and independent implementation review; record fresh commands and results in `verification.md`, then verify every task has concrete evidence before archival.
- [ ] 3.4 Archive the completed change only with `pwsh -File scripts/openspec-archive-seq.ps1 -ChangeName "model-surface-revision-and-bounded-context-contract"`; verify archive index/roadmap synchronization, commit the scoped change, merge into latest `master`, push, verify `origin/master`, back up only runtime artifacts if produced, and remove the merged worktree and branch.

## Example Impact Assessment

无需示例变更（附理由）：本 change 不改变 `examples/agent-modes` 的配置、
runtime path、semantic anchor、expected markers 或 rollback notes。host material
与 request projection 的可执行证据由 package tests、offline replay fixtures
和 dedicated gates 提供，不引入新的 agent-mode 示例。

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`context/hostprojection/README.md`、`model/README.md`、`docs/runtime-module-boundaries.md`、`docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`docs/development-roadmap.md`
- 维护 owner：`context/hostprojection`、`model/conformance` 和 documentation maintainers
- 验证证据：`go test ./context/hostprojection ./model/conformance ./tool/diagnosticsreplay`、dedicated shell/PowerShell gate、`pwsh -File scripts/check-openspec-documentation-impact.ps1`、`pwsh -File scripts/check-docs-consistency.ps1`、`openspec validate --all`

## Documentation Impact Assessment

| area | outcome | affected paths | owner | evidence |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | architecture maintainers | package boundary and gate checks |
| host context API | 修改文档 | `context/hostprojection/README.md` | context maintainers | exact-bound resolver tests |
| model request evidence | 修改文档 | `model/README.md` | model maintainers | canonical/replay/parity tests |
| configuration and diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime maintainers | no new key/writer assertions |
| contract index | 修改文档 | `docs/mainline-contract-test-index.md` | quality maintainers | fixture and dedicated gate |
| public entry point | 修改文档 | `README.md` | documentation maintainers | public capability summary |
| roadmap | 修改文档 | `docs/development-roadmap.md` | documentation maintainers | status consistency check |
| components | 修改文档 | `context/hostprojection/README.md`、`model/README.md` | context/model maintainers | focused package tests |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime maintainers | no-config assertion and docs consistency gate |
| contract/API | 修改文档 | `README.md`、`docs/runtime-module-boundaries.md` | API/architecture maintainers | OpenSpec validation and focused contract gates |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime maintainers | RuntimeRecorder/QueryRuns exclusion assertion |
| examples | 修改文档 | `context/hostprojection/README.md`、`model/README.md` | context/model maintainers | Contract documentation records the behavior; existing examples/agent-modes paths and markers remain unchanged. |
| CLI/integration | 修改文档 | `docs/mainline-contract-test-index.md` | quality maintainers | dual-platform dedicated gates |
| best practices | 修改文档 | `README.md`、`model/README.md` | documentation maintainers | privacy/default behavior tests |
