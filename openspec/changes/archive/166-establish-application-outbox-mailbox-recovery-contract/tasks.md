## 1. Documentation baseline and contract preparation

- [x] 1.1 Update roadmap/README/current-status documentation to move `establish-application-outbox-mailbox-recovery-contract` from candidate to active, distinguish planned semantics from delivered behavior, and verify `pwsh -File scripts/check-openspec-roadmap-status-consistency.ps1` passes.
- [x] 1.2 Establish `examples/application-outbox-recovery/README.md` before example code with semantic anchors, runtime paths, mailbox/scheduler support matrix, expected markers, crash-window cases, no-live-dependency rule, and rollback notes; verify documentation impact and example-impact declaration gates pass.
- [x] 1.3 Update module-boundary, integration, best-practices, diagnostics, orchestration/mailbox/scheduler and contract-index documentation with the opt-in/reference-only/owner boundaries; verify proposal/design/tasks commitments name identical changed paths and `pwsh -File scripts/check-openspec-documentation-impact.ps1` passes.
- [x] 1.4 Define the bounded `application_outbox_dispatch.v1` profile, finite error taxonomy, full duplicate identity and target/result support matrix in the new package/spec-facing documentation; verify malformed/oversize/unknown-target examples have an unambiguous fail-closed classification.

## 2. Bounded bridge API and admission

- [x] 2.1 Add a focused orchestration bridge package with constructor-only opt-in public DTOs for common outbox correlation, tagged mailbox/scheduler target bindings, bounded validation/canonical digest and function adapters; verify package unit tests cover valid, missing, malformed, oversized, unsupported-version and conflicting identity cases.
- [x] 2.2 Add trusted current host admission/effect-idempotency interfaces that accept reference-only facts and fail closed on missing, stale, invalid, canceled, errored or panicking callbacks; verify no callback holds a source lock and failed admission performs no target mutation.
- [x] 2.3 Implement complete logical dispatch duplicate handling across profile, target, outbox/occurrence, revision/effect references, source correlation and idempotency key; verify exact duplicates converge while key reuse with changed target/revision/effect binding rejects.

## 3. Native target delegation and recovery boundaries

- [x] 3.1 Implement the mailbox target adapter using existing mailbox publish/claim/ack/nack/requeue/delay/lease-reclaim/DLQ behavior without a parallel worker, retry, lease or terminal state; verify target lifecycle classifications and duplicate publish convergence stay source-owned.
- [x] 3.2 Implement the scheduler target adapter using existing task/claim/attempt/lease/requeue/retry/terminal/DLQ behavior without a parallel task, attempt, retry or terminal state; verify stale attempts and lease rollover cannot confirm a newer occurrence/effect binding.
- [x] 3.3 Represent dispatch accepted, target lifecycle, host effect evidence and host outbox confirmation as distinct reference-only result facts; verify mailbox acknowledgement, scheduler terminal, Run/Stream terminal, digest and replay never imply business commit or delete/confirm an outbox record.
- [x] 3.4 Implement recovery reconciliation for publish/enqueue-before-outbox-confirmation and effect-committed-before-acknowledgement, preserving at-least-once behavior and non-composed retry ownership; verify mailbox delivery retry, scheduler execution retry and host business-commit retry cannot amplify one another.
- [x] 3.5 Add cancellation, delayed delivery, DLQ, current-binding drift and target-incompatibility handling with no-side-effect-before-rejection evidence; verify unsupported paths create no message, task, attempt, Run, effect or synthetic terminal outcome.

## 4. Conformance, replay and example

- [x] 4.1 Add focused bridge/mailbox/scheduler/composer-recovery unit and integration tests with fake outbox/effect collaborators for both targets, two crash windows, duplicates, mismatches, stale lease, delay, cancel and DLQ; verify the affected package suites pass with `go test -count=1`.
- [x] 4.2 Add supported source Run/Stream conformance tests where dispatch activates native execution; verify normalized source outcome, correlation and target facts agree and delivery acceptance never becomes a business terminal result.
- [x] 4.3 Add versioned reference-only diagnostics replay fixtures and drift classifications; verify replay never invokes host callbacks, executes effects, publishes/enqueues targets, mutates outbox state or restores runtime state.
- [x] 4.4 Implement the documented offline example using fake outbox/effect store and verify it emits the documented mailbox/scheduler, crash-window, duplicate and rollback markers without a live provider or business database.
- [x] 4.5 Add `scripts/check-application-outbox-mailbox-recovery-contract.sh` and `.ps1`, contribution checks, quality-path wiring and contract-index mapping; verify both platform gates exercise equivalent suites and fail on target/commit/replay drift.

## 5. Completion verification and delivery lifecycle

- [x] 5.1 Run and record focused unit/integration/replay/example/contribution suites, dedicated Shell and PowerShell gates, `openspec validate establish-application-outbox-mailbox-recovery-contract --strict`, documentation/example/roadmap gates and `git diff --check`; verify all actual exit codes and environment limitations are recorded in `verification.md`.
- [x] 5.2 Run and record `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`, `pwsh -File scripts/check-quality-gate.ps1`, and `pwsh -File scripts/check-docs-consistency.ps1`; verify generated baseline artifacts are handled deliberately and no task is checked without code/test/document/gate evidence.
- [x] 5.3 Before archive, recheck public API/limits/errors/support matrix/rollback documentation and all three identical Public Semantic Documentation Commitments; verify every declared documentation path is present in the branch diff and no planned wording claims unimplemented delivery.

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/development-roadmap.md`、`docs/runtime-module-boundaries.md`、`docs/guides/integration.md`、`docs/guides/best-practices.md`、`docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`orchestration/README.md`、`orchestration/outboxbridge/README.md`、`orchestration/mailbox/README.md`、`orchestration/scheduler/README.md`、`examples/application-outbox-recovery/README.md`
- 维护 owner：orchestration、mailbox、scheduler、host integration、observability 与 documentation maintainers
- 验证证据：proposal/design/tasks 的一致声明；实现文档与代码/测试/示例/replay/gate 的分支差异；OpenSpec strict validation；documentation/example impact 与 roadmap/docs consistency gates；`verification.md` 记录真实 unit/integration/replay/example/gate 与完整质量门禁的实际退出结果

本 change 的公开语义包括 opt-in API、descriptor limits、有限失败分类、mailbox/scheduler support matrix、delivery/execution/business-commit 三者的非推导关系、at-least-once 恢复模型以及 rollback。上述路径已在本功能分支同步维护；当前实现已交付 `orchestration/outboxbridge`、mailbox/scheduler binding、integration/replay/example 与双平台 contract gate，真实验证证据集中记录在 `verification.md`。归档前仍需保留这些代码、测试、示例、文档和门禁证据。

## Documentation Impact Assessment

| Area | Outcome | Affected paths | Owner | Verification |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md`, `orchestration/README.md` | orchestration maintainers | bridge 不拥有 outbox、effect、retry、lease、terminal 或 diagnostics state |
| components | 修改文档 | `orchestration/mailbox/README.md`, `orchestration/scheduler/README.md` | mailbox/scheduler maintainers | 两类 target 保留 source-owned lifecycle 与 support matrix |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime config maintainers | constructor-only opt-in；无 env/file/hot-update 键，保持 `env > file > default` |
| contract/API | 修改文档 | `docs/guides/integration.md`, `docs/mainline-contract-test-index.md` | contract/host integration maintainers | descriptor/admission/recovery、replay 和 gate 对照 delta requirements |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md` | observability maintainers | 仅 additive + nullable + default；RuntimeRecorder 单写且无业务 body |
| examples | 新增文档 | `examples/application-outbox-recovery/README.md`, `README.md` | examples maintainers | README 先于 example code，覆盖双 target、两类 crash window、duplicate 与 rollback markers |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | host integration maintainers | 已交付 constructor-only 接入、双 target 委托、recovery/rollback 边界与可运行示例入口 |
| best practices | 修改文档 | `docs/guides/best-practices.md` | documentation maintainers | delivery/terminal/digest 不等于 business commit；effect CAS 仍由宿主负责 |
| roadmap | 修改文档 | `docs/development-roadmap.md`, `README.md` | roadmap maintainers | active slug 与 `openspec list --json` 一致，archive 后再更新归档状态 |
