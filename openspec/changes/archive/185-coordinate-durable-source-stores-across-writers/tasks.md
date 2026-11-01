## 1. Contract and failing evidence

- [x] 1.1 Add source-coordination DTOs/interfaces, bounds and typed errors; write failing tests for valid lease/CAS, foreign/stale fence and revision conflict before implementation.
- [x] 1.2 Add deterministic coordinator test adapter and durable shared-directory adapter; prove reopen, handover, corruption and unavailable behavior.
- [x] 1.3 Update capability/plan deployment validation fixtures so local-only owners reject coordinated requests before source activation.

## 2. Checkpoint coordination

- [x] 2.1 Add explicit checkpoint store coordinator option; keep default FileStore single-writer semantics unchanged.
- [x] 2.2 Route save, claim, running/terminal/successor transitions through fenced source snapshot CAS; prove competing writers converge without duplicate logical attempt.
- [x] 2.3 Prove stale writer, CAS conflict, crash-before-publication and coordinator outage fail closed with no Run/retry/effect fallback.

## 3. History coordination

- [x] 3.1 Add explicit journal coordinator option and source snapshot encoding; retain default file journal behavior.
- [x] 3.2 Route append/retention floor through CAS and prove restart/concurrent writers maintain contiguous source-local identity.
- [x] 3.3 Prove ReadAfter remains read-only across cursor, expiry, conflict and unavailable cases.

## 4. Capability, bridge, replay and example

- [x] 4.1 Project live coordinator evidence into continuation/history capabilities and manifest cells.
- [x] 4.2 Extend planbridge admission and tests for matching/mismatched coordinated requests with no-side-effect rejection.
- [x] 4.3 Add read-only diagnostics replay facts/fixtures and paired shell/PowerShell contract gates.
- [x] 4.4 Add an offline coordinated-source-store example and update README, integration, module boundaries, diagnostics, index and roadmap.

## 5. Verification and archive

- [x] 5.1 Run focused positive/negative/concurrent/restart tests plus relevant dedicated gates.
- [x] 5.2 Run `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`, quality/docs/roadmap gates and `openspec validate --all`.
- [x] 5.3 Archive through `scripts/openspec-archive-seq.ps1`, review the archive index, commit, merge latest `master`, push and verify `master == origin/master`.

## Example Impact Assessment

新增示例

说明：add a documented offline coordinated-source-store example proving explicit configuration and both CAS success and local-only rejection.

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/development-roadmap.md`、`docs/guides/integration.md`、`docs/runtime-module-boundaries.md`、`docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`core/history/README.md`、`orchestration/README.md`、`examples/coordinated-source-store/README.md`
- 维护 owner：core source-coordination, checkpoint/history and planbridge maintainers
- 验证证据：focused Go tests、durable-store shell/PowerShell gate、docs consistency gate、OpenSpec validation and full quality verification

## Documentation Impact Assessment

| 表面 | 结果 | 文档路径 | Owner | 验证证据 |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | architecture maintainers | boundary review and docs consistency gate |
| components | 修改文档 | `core/history/README.md`, `orchestration/README.md` | source-owner maintainers | focused coordinator/history/checkpoint tests |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime configuration maintainers | constructor-only/no-config-key documentation review |
| contract/API | 修改文档 | `README.md`, `docs/guides/integration.md` | core and integration maintainers | deployment, checkpoint, history and planbridge tests |
| diagnostics | 修改文档 | `docs/mainline-contract-test-index.md`, `docs/runtime-config-diagnostics.md` | diagnostics maintainers | diagnostics replay tests and dedicated gate |
| examples | 新增文档 | `examples/coordinated-source-store/README.md` | example maintainers | `go run ./examples/coordinated-source-store` |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | integration maintainers | integration documentation review and dedicated gate |
| best practices | 修改文档 | `docs/guides/integration.md` | integration maintainers | explicit coordinator/rollback guidance review |
| roadmap | 修改文档 | `docs/development-roadmap.md` | runtime maintainers | roadmap consistency gate |
