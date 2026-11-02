## 1. Owner-native handler continuation contract

- [x] 1.1 Add bounded paused/completed/failed handler result and registered resumer/revalidation DTOs, keeping the legacy worker handler API compatible; verify malformed/oversized/foreign material is rejected by focused mailbox tests.
- [x] 1.2 Add a failing then passing worker safe-point path that atomically seals an original in-flight handler continuation without Ack/Nack/Requeue/retry/publish/re-consume; verify `go test ./orchestration/mailbox -run HandlerPause`.
- [x] 1.3 Add a failing then passing atomic resume path that validates live source lineage and invokes a configured resumer once; verify duplicate, foreign, expired, deadline, unavailable and lease-loss tests pass, including FileStore fresh reopen.
- [x] 1.4 Route resumed success/failure through the existing mailbox Ack/Nack/Requeue/DLQ owner paths and retain duplicate terminal projections; verify retry/DLQ, crash/recovery-required and no-parent-rerun tests pass.

## 2. Capability and bridge admission

- [x] 2.1 Publish mailbox continuation capability cells from explicit durable handler continuation configuration only; verify default lifecycle-only and partial per-reason matrices remain rejected in `go test ./orchestration/planbridge ./core/types`.
- [x] 2.2 Extend planbridge mailbox continuation admission/binding only as needed to validate the handler-safe-point source evidence and preserve Run/Stream parity; verify unsupported admission has no mailbox mutation and configured parity tests pass.
- [x] 2.3 Add/update versioned fixtures and paired PowerShell/shell contract gates for handler pause, lease loss, duplicate restore, retry/DLQ and unsupported fallback; verify both dedicated gates pass.

## 3. Public semantics and examples

- [x] 3.1 Update mailbox, planbridge, integration, module-boundary, diagnostics, README, example, capability-test-index and roadmap docs to distinguish lifecycle projection from configured handler restore; verify documentation-impact and roadmap consistency checks pass.
- [x] 3.2 Record public API/typed error, deployment-scope, rollback and no-second-owner semantics in all required OpenSpec artifacts; verify `openspec validate --all` passes.

## 4. Complete validation and archive readiness

- [x] 4.1 Run affected package/unit/integration and race coverage for mailbox and planbridge; verify focused and broader `go test` commands pass without fixture drift.
- [x] 4.2 Run repository gates: `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`, `pwsh -File scripts/check-quality-gate.ps1`, `pwsh -File scripts/check-docs-consistency.ps1`, and `git diff --check`; record only successful evidence before archive.

## Example Impact Assessment

修改示例

理由：更新现有 host execution plan / native event example 的语义锚点，说明显式
配置的 mailbox handler continuation、admission evidence 与默认拒绝边界；不新增
业务 payload、平行 worker 或自动重跑示例。

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/development-roadmap.md`、`docs/guides/integration.md`、`docs/runtime-module-boundaries.md`、`docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`orchestration/mailbox/README.md`、`orchestration/planbridge/README.md`、`examples/host-execution-plan/README.md`、`examples/native-execution-events/README.md`
- 维护 owner：runtime contracts owner、mailbox owner maintainer、planbridge maintainer 与 examples maintainer
- 验证证据：mailbox owner focused/fresh-process tests、Run/Stream parity、dedicated PowerShell/shell gate、race/lint/quality/docs gates 和 OpenSpec validation

## Documentation Impact Assessment

| area | outcome | affected paths | owner | evidence |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | runtime contracts owner | source-owner boundary review |
| components | 修改文档 | `orchestration/mailbox/README.md`、`orchestration/planbridge/README.md` | mailbox/planbridge maintainers | focused owner and bridge tests |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime contracts owner | no-new-config regression and documentation gate |
| contract/API | 修改文档 | `docs/guides/integration.md` | runtime contracts owner | strict OpenSpec validation |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md` | diagnostics maintainer | dedicated contract gate |
| examples | 修改文档 | `examples/host-execution-plan/README.md`、`examples/native-execution-events/README.md` | examples maintainer | expected-marker review |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | runtime contracts owner | pre-admission rejection tests |
| best practices | 修改文档 | `README.md` | documentation maintainer | rollback boundary review |
| roadmap | 修改文档 | `docs/development-roadmap.md` | documentation maintainer | roadmap consistency gate |
