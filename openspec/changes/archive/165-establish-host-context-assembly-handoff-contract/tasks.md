## 1. Admission audit and documentation baseline

- [x] 1.1 Audit existing `context/assembler`, `context/handoff`, `context/budgetprojection`, runner request flow, planbridge identity and host-resume revalidation; add a minimal synthetic valid/invalid projection fixture that proves the required gap or records why no runtime bridge change is needed, and verify no source resolver/provider/tool/diagnostic side effect occurs before validation.
- [x] 1.2 Move this candidate from the roadmap pool to the active-change status, create `examples/host-context-assembly/README.md` before example code with semantic anchor/runtime path/expected markers/rollback notes, and verify roadmap/example documentation gates pass.
- [x] 1.3 Define the supported direct Run/Stream and legal handoff/restore matrix, explicitly reject resolver-independent restore, cross-process continuation, composed mailbox/workflow/team recovery and dynamic-action continuation, and verify the matrix is consistent with existing planbridge and host-resume support boundaries.

## 2. Bounded host projection contract

- [x] 2.1 Add the provider-neutral `host_context_projection.v1` descriptor, immutable detached snapshot, canonical serializer/digest and constructor-only bounds (32 KiB envelope, 64 references, 256-byte references, 128-byte digests); verify valid vectors, unknown version, missing identity, duplicate/oversized references, invalid capabilities, exact bounds and caller mutation in focused unit tests.
- [x] 2.2 Add the host-owned reference resolver and finite `host_context.*` classification boundary; verify malformed/drifted descriptors never invoke it, a valid descriptor resolves only current authorized material, resolver denial/missing source fails closed, and no body/callback error enters the contract result.
- [x] 2.3 Add parent/child capability-subset validation and optional immutable host-plan identity/version/digest correlation; verify narrowing succeeds, widening/missing parent evidence fails before resolution, and correlation cannot dispatch, recompile, resume or authorize a plan.

## 3. Existing assembler and budget integration

- [x] 3.1 Thread the optional detached projection through the minimal existing context-assembly entry point while preserving legacy callers when absent; verify the validation order is structure/bounds → digest → correlation/subset → resolver → existing assembler and that no host code runs under an assembler/source lock.
- [x] 3.2 Reuse `budget_projection.v1`, prefix/journal and context-pressure/compaction behavior without write-back or new budget state; verify incomplete/saturated budgets retain existing derived semantics and cannot change admission or terminal outcomes.
- [x] 3.3 Cover direct real-runner Run and Stream assembly with fake model/tool using the same valid projection; verify normalized assembled projection, budget facts and bounded classifications are semantically equivalent without fabricated stream/terminal behavior.

## 4. Handoff and restore revalidation

- [x] 4.1 Add additive, nullable, bounded host projection correlation to legal existing `handoff.v1` cuts without resolved bodies or business authorization payloads; verify legacy handoff consumers remain compatible and finalized/checkpoint/tool-finalized/flushed-stream cuts retain only opaque evidence.
- [x] 4.2 Require existing handoff/source validation followed by projection correlation validation and current host resolver reauthorization before restore injection; verify reference/digest drift, revoked access, resolver failure and capability expansion fail before context mutation, continuation, effect or terminal emission.
- [x] 4.3 Verify repeated valid restore remains idempotent through the existing durable restore identity path and equivalent Run/Stream handoffs restore to semantically equal normalized outcomes; verify a handoff/replay digest never acts as current authorization.

## 5. Replay, contract gates, and regression evidence

- [x] 5.1 Add versioned reference-only `host_context_projection.v1` golden, malformed, boundary, privacy, authorization, digest/reference drift, capability-expansion, restore and Run/Stream fixtures; verify deterministic repeated replay invokes no resolver/provider/tool/network/business store/RuntimeRecorder and classifies every required drift.
- [x] 5.2 Add equivalent shell and PowerShell host-context-assembly-handoff contract gates plus deliberate-drift tests; verify both fail for privacy leakage, unauthorized restore, Run/Stream mismatch, source side effect before rejection, Provider SDK in `context/*`, budget write-back, direct diagnostics writing and parallel execution/terminal/recovery ownership.
- [x] 5.3 Wire the dedicated gate into contribution/quality verification only after its suite exists, update the mainline contract index with actual suite paths, and verify the gate reports unavailable/skipped checks explicitly rather than as success.

## 6. Public documentation, example, and release verification

- [x] 6.1 Implement the offline host-context-assembly example after its README baseline, using synthetic opaque references and fake model/tool to demonstrate valid pre-filtered assembly, capability-widening rejection, restore reauthorization and rollback; verify expected markers without a live provider or business database.
- [x] 6.2 Synchronize `README.md`, `docs/development-roadmap.md`, `context/README.md`, `context/handoff/README.md`, `docs/runtime-module-boundaries.md`, `docs/runtime-config-diagnostics.md`, `docs/guides/integration.md`, `docs/mainline-contract-test-index.md` and example/package README paths with final API, bounds, errors, owner boundaries, support matrix, compatibility and rollback; verify all declared paths are in the same branch diff.
- [x] 6.3 Run focused unit/integration/replay/gate suites, `openspec validate establish-host-context-assembly-handoff-contract --strict`, `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml ./...`, `pwsh -File scripts/check-quality-gate.ps1`, `pwsh -File scripts/check-docs-consistency.ps1` and `git diff --check`; record actual commands, exit codes, risks, rollback point and justified non-executed items in `verification.md` before any task is marked complete or archive is attempted.

## Example Impact Assessment

新增示例

新增 `examples/host-context-assembly`。README 将在任何示例代码之前定义 semantic anchor、runtime path、支持/拒绝边界、expected markers 与 rollback notes；示例采用离线 fake model/tool 和 synthetic opaque references，不接入 agent-modes、live provider 或业务数据库。

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/development-roadmap.md`、`context/README.md`、`context/handoff/README.md`、`docs/runtime-module-boundaries.md`、`docs/runtime-config-diagnostics.md`、`docs/guides/integration.md`、`docs/mainline-contract-test-index.md`、`examples/host-context-assembly/README.md`
- 维护 owner：context、runner、host integration 与 documentation maintainers
- 验证证据：OpenSpec strict validation、documentation/example impact、roadmap/docs consistency；实施后补齐 context contract/replay、真实 runner Run/Stream parity、双平台 gate 与完整 Go/race/lint/quality/docs 门禁的实际命令和退出结果

“已同步”指本 change 已在同一功能分支建立 active proposal 的公开语义、owner、范围、文档与示例基线；它不表示运行时代码已经交付。最终 API、limits、错误、支持矩阵与实现证据仍须在任务 6 中更新并以 `verification.md` 的实际命令结果为准。没有上述维护路径的实际 Git diff、对应测试与门禁证据，任务不得勾选、change 不得归档。

## Documentation Impact Assessment

| Area | Outcome | Affected paths | Owner | Verification |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md`、`context/README.md` | context maintainers | bridge 不拥有 source body、业务授权、预算或恢复状态；`context/*` 不引入 Provider SDK |
| components | 修改文档 | `context/handoff/README.md`、实施新增 bridge README | context/runner maintainers | projection → assembler → handoff/restore owner 映射与实际 API 对照 |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime config maintainers | 明确 explicit opt-in、constructor-only limits、无新 env/file/hot-update 键与现有优先级不变 |
| contract/API | 修改文档 | `docs/guides/integration.md`、`docs/mainline-contract-test-index.md` | host integration/contract maintainers | spec requirement、正负边界/integration/replay/gate 逐项映射 |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md` | observability maintainers | bounded opaque reference/digest、RuntimeRecorder 单写、不新增 QueryRuns 字段或业务正文 |
| examples | 新增文档 | `examples/host-context-assembly/README.md`、`README.md` | examples maintainers | 文档基线先行、离线 smoke、expected markers 与 rollback 验证 |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | host integration maintainers | explicit opt-in、宿主 resolver 与无新增 wire command 的 active-proposal 边界 |
| best practices | 无需文档变更（附理由） | `docs/guides/best-practices.md` | documentation maintainers | 该 contract 的集成 guardrails 已由 integration/context 文档覆盖；尚未增加通用操作建议或独立 best-practice 语义 |
| roadmap | 修改文档 | `docs/development-roadmap.md` | roadmap maintainers | active slug 与 `openspec list --json` 状态一致，归档后同步索引 |
