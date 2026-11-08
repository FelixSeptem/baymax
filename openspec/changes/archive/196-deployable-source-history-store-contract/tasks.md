## 1. Owner-store binding and deployment validation

- [x] 1.1 Add a public `core/history` owner-store binding/factory that explicitly binds source identity, journal, optional projection ledger, coordinator lease, writer identity, retention, and deployment request; verify missing or mismatched components fail closed before capability publication.
- [x] 1.2 Revalidate deployment scope and coordinator lease on capability reads and mutations; verify same-host file profiles remain supported while cross-machine, memory-only, stale, foreign, and unavailable profiles return typed unsupported/unavailable outcomes.
- [x] 1.3 Add focused tests for factory construction, owner namespace/fence validation, deployment request rejection, fresh-process reopen, and concurrent writer stale/CAS conflict behavior.

## 2. Source history and projection semantics

- [x] 2.1 Make coordinated append identity-aware and idempotent for identical detached facts while rejecting conflicting event identity, digest, sequence, terminal evidence, or source metadata; verify sequence and retention-floor CAS atomicity.
- [x] 2.2 Route bounded `ReadAfter` and projection reconciliation through the explicit owner binding, preserving malformed/foreign/expired/gap/conflict/unavailable/recovery-required typed errors and read-only execution boundaries.
- [x] 2.3 Add crash-before-projection and fresh-process reconciliation tests proving committed facts replay without rerunning uncertain effects, with duplicate convergence and conflict rejection.

## 3. Capability evidence, fixtures, and gates

- [x] 3.1 Add source-owner capability fixtures covering unconfigured, same-host single-writer, same-host coordinated multi-writer, stale lease, coordinator outage, cross-machine rejection, and projection non-atomicity.
- [x] 3.2 Add equivalent `scripts/check-source-history-store-contract.ps1` and `.sh` gates covering factory binding, multi-process CAS/fencing, stable identity, bounded replay, reconciliation, and no-side-effect assertions; wire both into quality gates.
- [x] 3.3 Update `README.md`, roadmap, integration guide, runtime diagnostics/configuration, contract index, core types README, and host execution-plan example with the explicit owner-store profile and non-goals; verify docs consistency and example markers.

## 4. Verification and delivery

- [x] 4.1 Run focused history/sourcecoordination/types tests, contract gates, and `git diff --check`.
- [x] 4.2 Run `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`, quality/docs gates, OpenSpec validators, and all applicable replay/contract gates; record evidence in this file.
- [x] 4.3 Archive with `scripts/openspec-archive-seq.ps1`, explicitly stage only this change, commit, merge into latest `master`, push, verify merged state, and delete the merged branch/worktree.

Verification evidence for 4.2 (2026-11-08): `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`, `pwsh -File scripts/check-source-history-store-contract.ps1`, `pwsh -File scripts/check-durable-store-deployment-scope-contract.ps1`, `pwsh -File scripts/check-docs-consistency.ps1`, `pwsh -File scripts/check-openspec-documentation-impact.ps1`, `pwsh -File scripts/check-openspec-roadmap-status-consistency.ps1`, `pwsh -File scripts/check-openspec-example-impact-declaration.ps1`, `openspec validate --all` (154 passed, 0 failed), and `git diff --check` passed. `pwsh -File scripts/check-quality-gate.ps1` completed with exit code 0 after 95 steps in 1234.16 seconds; its full-chain example smoke and strict `govulncheck` completed successfully.

## Example Impact Assessment

修改示例

理由：host execution plan 示例需要展示显式 owner-store factory/deployment profile，以及未配置、跨机器和 projection 非原子边界的拒绝证据。

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/development-roadmap.md`、`docs/guides/integration.md`、`docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`core/types/README.md`、`examples/host-execution-plan/README.md`
- 维护 owner：runtime contracts owner、source history owner、examples/documentation maintainer
- 验证证据：`go test ./core/history ./core/sourcecoordination ./core/types`、`pwsh -File scripts/check-source-history-store-contract.ps1`、`pwsh -File scripts/check-quality-gate.ps1`、`pwsh -File scripts/check-docs-consistency.ps1`、`openspec validate --all`

## Documentation Impact Assessment

| area | outcome | affected paths | owner | evidence |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/guides/integration.md` | runtime contracts owner | owner-store deployment tests |
| components | 修改文档 | `core/types/README.md` | source history owner | capability/reconciliation tests |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime contracts owner | fail-closed profile tests |
| contract/API | 修改文档 | `docs/guides/integration.md` | runtime contracts owner | factory and typed error tests |
| diagnostics | 修改文档 | `docs/mainline-contract-test-index.md` | diagnostics/replay maintainer | dedicated gate |
| examples | 修改文档 | `examples/host-execution-plan/README.md` | examples/documentation maintainer | deployment matrix markers |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | runtime contracts owner | owner-store binding/reconciliation tests |
| best practices | 修改文档 | `README.md` | documentation maintainer | docs consistency |
| roadmap | 修改文档 | `docs/development-roadmap.md` | documentation maintainer | roadmap consistency |
