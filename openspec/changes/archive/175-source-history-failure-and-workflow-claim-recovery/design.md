## Context

See `proposal.md` for motivation. Native history is explicitly injected into source owners; current event callbacks are projections, not a transactional source journal. Workflow continuation persists claims and results in a file-backed store, but restart handling must distinguish a completed result from an execution attempt whose outcome is unknown.

## Goals / Non-Goals

**Goals:**
- Make history writer failure visible at the owner boundary without changing opt-in behavior.
- Identify the source whose callback-history append failed and avoid claiming complete retained history.
- Preserve fail-closed `recovery_required` semantics at the workflow continuation claim/result boundary when effects may have happened but the source result is not committed.
- Make abandoned workflow claims deterministic after reopen and enforce the declared sequential single-writer scope.
- Keep the required strict security scan executable by pinning its validation toolchain to the patched Go `1.26.10` release.

**Non-Goals:**
- Atomic transactions between Baymax source facts and Danny/application projections.
- A second runtime journal, scheduler, retry loop, or automatic execution recovery mechanism.
- Multi-writer, multi-instance, or cross-machine FileStore support.
- Enabling child/team/mailbox/resumed-plan continuation beyond existing capabilities.

## Decisions

1. **Separate observation health from execution arbitration.** Runner/child/team/workflow callback append failures are visible through a typed source-identified history-health path. These callbacks observe source transitions after or alongside execution and cannot establish a pre-effect gate. Their failure therefore marks retained history degraded/unavailable; it does not retroactively stop or relabel the underlying Run result. Workflow continuation store transitions remain the source-owned fail-closed boundary for resume claims and committed results.

2. **Do not make generic callback projection the transaction boundary.** Runner/composer/team/workflow callbacks can report facts, but they cannot prove that an external effect and journal append committed atomically. A callback append failure does not imply the source effect was prevented. Already committed facts remain replayable; consumer projection remains independently idempotent. Any recovery decision must use the source continuation/checkpoint owner, not callback history.

3. **Reclassify incomplete claims on store reopen.** A newly opened workflow store treats both `claimed` and `running` records without a committed result as `recovery_required`. It does not inspect timestamps and guess that an old worker is dead, nor does it rerun pending steps. Completed result records remain duplicate-readable.

4. **Fence workflow FileStore ownership at process scope.** Acquire a path-scoped OS lock when opening the store, retain it for the store lifetime, release on `Close`, and reject a second owner before reading/mutating records. This implements only same-host sequential reopen. If the platform cannot enforce the lock, capability is unavailable rather than optimistic.

5. **Keep existing typed errors and add only the minimum missing classification.** Reuse source unavailable/recovery-required types where semantically correct; introduce a dedicated history append error or health DTO only if existing contracts cannot distinguish degraded observation from execution recovery. Capability lookup must not report support when its required writer/fence is unavailable.

6. **Treat the Go patch floor as a strict verification dependency.** `GO-2026-6809` is reported reachable in the configured Go `1.26.9` standard library, and the scanner reports Go `1.26.10` as the fixed version. Keep the language line at `go 1.26.0`, update only the `toolchain` floor and its public prerequisite documentation, and retain strict scan behavior without suppression or downgrade.

## Risks / Trade-offs

- [A journal append can fail after an effect has executed] -> Surface recovery-required and prohibit automatic rerun; require source-specific evidence or operator/host resolution.
- [Callback append failure can leave retained history incomplete after an effect] -> Expose source-identified degraded/unavailable history health; do not claim this callback journal gates execution or repairs uncertain effects.
- [OS locking semantics vary by platform/filesystem] -> Test Windows and Unix process contention; return typed unavailable where the filesystem cannot provide the declared local fence.
- [A crashed `running` attempt cannot safely be resumed from status alone] -> Keep fail-closed recovery-required; do not infer that no effect occurred.

## Migration Plan

1. Add owner-level typed append result/health and failure-injection tests while preserving existing no-history behavior.
2. Update workflow reopen classification and FileStore fencing, then add subprocess crash/contention tests.
3. Update capability checks and package/API documentation after tests prove the behavior.
4. Roll back by disabling the capability advertisement and returning typed unavailable; never fall back to a new Run.

## Example Impact Assessment

无需示例变更（附理由）：本 change 不新增正常执行形态或配置入口；故障注入和进程崩溃测试比正常路径示例更适合验证这些错误语义。

## Documentation Impact Assessment

| Area | Outcome | Document | Owner | Evidence |
|---|---|---|---|---|
| architecture | 修改文档 | `orchestration/workflow/README.md` | Workflow owner | 明确 incomplete claim、recovery-required 与单写者 fence 行为 |
| components | 修改文档 | `core/types/README.md` | Runtime contracts owner | 明确显式 history 注入、append failure 与 capability health 边界 |
| configuration | 修改文档 | `core/types/README.md` | Runtime contracts owner | 明确本 change 不新增配置键或默认值 |
| validation toolchain | 修改文档 | `CONTRIBUTING.md`, `docs/versioning-and-compatibility.md`, `docs/runtime-config-diagnostics.md` | Runtime contracts owner | 将 strict security validation toolchain floor 同步为 Go 1.26.10 |
| contract/API | 修改文档 | `core/types/README.md` | Runtime contracts owner | 记录 typed history failure 和 workflow recovery 语义 |
| diagnostics | 修改文档 | `core/types/README.md` | Runtime contracts owner | 明确不增加诊断 schema 或 recorder 字段 |
| examples | 修改文档 | `core/types/README.md` | Runtime contracts owner | 说明负向/崩溃窗口由测试覆盖，不增加示例 |
| CLI/integration | 无需文档变更（附理由） | — | Runtime contracts owner | 不改变 CLI 或 integration 调用流程 |
| best practices | 无需文档变更（附理由） | — | Runtime contracts owner | 由 owner package 文档说明 failure/recovery 约束，不引入新惯例 |
| roadmap | 修改文档 | `docs/development-roadmap.md` | Runtime roadmap owner | 标记本 change 为 active 并保留剩余 source-owner gaps |
| release status snapshot | 修改文档 | `README.md` | Runtime contracts owner | 将 active OpenSpec 状态与 `openspec list --json` 对齐 |

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`, `docs/development-roadmap.md`, `core/types/README.md`, `orchestration/workflow/README.md`, `CONTRIBUTING.md`, `docs/versioning-and-compatibility.md`, `docs/runtime-config-diagnostics.md`
- 维护 owner：Runtime contracts owner and Workflow owner
- 验证证据：`openspec validate --all`; `pwsh -File scripts/check-docs-consistency.ps1`; targeted history/workflow tests
