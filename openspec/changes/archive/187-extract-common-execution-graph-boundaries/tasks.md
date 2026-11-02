## 1. Shared Graph Contract

- [x] 1.1 Add table-driven `orchestration/graph` tests for identity/bounds, normalization, deterministic order/digest, missing/duplicate/self/cyclic topology, and verify `go test ./orchestration/graph` fails before implementation.
- [x] 1.2 Implement bounded graph definition compilation and typed errors; verify all graph package tests pass and malformed input returns no partial projection.
- [x] 1.3 Add canonical digest fixtures and drift rejection; verify stable output across input permutations and changed edge sets.

## 2. Owner Projections

- [x] 2.1 Add workflow projection tests for ordinary and expanded definitions, workflow-specific error mapping, and digest stability; verify `go test ./orchestration/workflow`.
- [x] 2.2 Add team projection tests mapping serial task order to edges and parallel/vote membership to independent nodes without changing owner execution; verify `go test ./orchestration/teams`.
- [x] 2.3 Add composer projection tests proving a child dispatch projects only one node and preserves caller graph identity; verify `go test ./orchestration/composer`.
- [x] 2.4 Add cross-form fixtures for equivalent workflow/team serial and independent graphs plus workflow/single-child projection; verify same canonical topology digests and explicit non-equivalence for drift.
- [x] 2.5 Verify owner continuation digests, checkpoint fixtures, restore claims, and Run/Stream terminal behavior remain unchanged with focused owner tests.

## 3. Public Evidence and Gates

- [x] 3.1 Add a read-only offline example for topology projection and explicit topology-digest limitations; verify its expected markers with `go test ./examples/execution-graph -run TestOfflineExampleEmitsContractMarkers -count=1`.
- [x] 3.2 Add cross-platform execution-graph contract gates and ownership checks; verify both shell and PowerShell scripts execute focused suites, fixture drift checks, and the example.
- [x] 3.3 Wire the dedicated gate into standard quality/contribution paths and verify the gate contract test detects missing wiring.
- [x] 3.4 Update public guide, documentation index, module boundary, configuration/diagnostic no-change facts, mainline contract index, roadmap status, and example documentation; verify `pwsh -File scripts/check-docs-consistency.ps1` and `pwsh -File scripts/check-openspec-roadmap-status-consistency.ps1`.

## 4. Full Verification

- [x] 4.1 Run focused affected packages, full `go test ./...`, and `go test -race ./...`; record results and any environment limitation in verification.md.
- [x] 4.2 Run `golangci-lint run --config .golangci.yml`, `pwsh -File scripts/check-quality-gate.ps1`, `pwsh -File scripts/check-docs-consistency.ps1`, `openspec validate --all`, and required OpenSpec documentation/example impact gates; record exact results in verification.md.

## Example Impact Assessment

新增示例

完成 `examples/execution-graph` 离线投影示例，并在任务 3.1 验证其 markers。

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
