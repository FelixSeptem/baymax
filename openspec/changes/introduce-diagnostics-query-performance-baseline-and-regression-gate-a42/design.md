## Context

`runtime/diagnostics` 已经提供统一查询入口（`QueryRuns`、`QueryMailbox`、`MailboxAggregates`），并被 A18/A29/A30-A41 的功能持续加字段使用。当前质量门禁中，性能回归阻断重点是 multi-agent mainline 与 CA4，但 diagnostics 查询路径尚无独立 benchmark 基线与阈值治理。

在查询记录规模增长时，查询路径的过滤、排序、分页与聚合容易出现尾延迟和分配回归；如果缺少专项 gate，这类回归会在功能测试全部通过时被遗漏。

## Goals / Non-Goals

**Goals:**
- 为 diagnostics 查询路径建立独立 benchmark 矩阵与固定数据集生成口径。
- 新增 shell/PowerShell 双脚本回归 gate，使用相对阈值评估 `ns/op`、`p95-ns/op`、`allocs/op`。
- 将 diagnostics-query perf gate 接入 `check-quality-gate.*`，默认阻断。
- 固化 fail-fast 语义：baseline 缺失/非法、参数非法、benchmark 输出不可解析均返回非零。
- 保持 lib-first 边界，不引入外部依赖和平台化组件。

**Non-Goals:**
- 不改变 `QueryRuns`、`QueryMailbox`、`MailboxAggregates` 的 API 语义与过滤规则。
- 不引入外部数据库或索引服务。
- 不与 A19 主链路 benchmark 合并为单脚本，保持职责分离。

## Decisions

### Decision 1: 建立 diagnostics-query 专项 benchmark，而不是复用 A19 主链路 benchmark

- 方案：新增独立脚本与 baseline 文件，覆盖查询三件套。
- 原因：A19 关注 orchestrator mainline，指标波动来源不同；单独治理更易定位回归根因。
- 备选：
  - 扩展 A19 脚本：职责混杂，基线更新与告警解释成本高。

### Decision 2: 固定数据集生成口径，避免环境噪声放大

- 方案：在 benchmark 内固定生成 run/mailbox 数据规模与分布（包含多 team/workflow/task 状态混合）。
- 原因：同一仓库、同一脚本下可重复比较，降低偶发波动。
- 备选：
  - 使用实时随机数据：不稳定，基线难维护。

### Decision 3: 采用相对阈值 + 严格 fail-fast

- 方案：默认阈值（建议值）：
  - `ns/op` 退化上限 `12%`
  - `p95-ns/op` 退化上限 `15%`
  - `allocs/op` 退化上限 `12%`
- 原因：查询路径受字段增长影响更敏感，阈值略宽于 A19 主链路但仍具阻断价值。
- 备选：
  - 仅记录不阻断：无法形成治理闭环。
  - 绝对阈值：跨机器可移植性差。

### Decision 4: 保持 shell / PowerShell 语义等价

- 方案：脚本输入变量、解析逻辑、失败条件保持一一对应。
- 原因：仓库已有跨平台 gate parity 要求，避免平台侧“假通过”。
- 备选：
  - 仅保留单平台脚本：违背现有治理基线。

## Risks / Trade-offs

- [Risk] 基准数据量偏大导致本地执行时间增长  
  -> Mitigation: 提供默认参数并支持环境变量调小规模用于本地快速回归。

- [Risk] 查询实现优化后，历史 baseline 很快过时  
  -> Mitigation: 保留可控 re-baseline 流程与文档，要求 PR 中同步说明更新原因。

- [Risk] 与 A19 并行执行时总体 benchmark 时间上升  
  -> Mitigation: 将 diagnostics-query 脚本放在质量门禁后段，并支持按需分段执行。

## Migration Plan

1. 在 `integration/benchmark_test.go` 增加 diagnostics query benchmark 框架与固定数据集构造。
2. 新增 `check-diagnostics-query-performance-regression.{sh,ps1}` 与 baseline env。
3. 将新 gate 挂接至 `check-quality-gate.*` 并保持日志标签清晰。
4. 更新 `docs/performance-policy.md`、`docs/mainline-contract-test-index.md`、`README.md`、`docs/development-roadmap.md`。
5. 初次合入时写入基线值并在 PR 记录测试环境与重基线原因。

## Open Questions

- 默认数据规模是否采用单一固定值（如 runs=5000、mailbox=20000）或按 `small/large` 双档执行（当前建议先单档固定）。
- 是否将 `TimelineTrends`/`CA2ExternalTrends` 纳入同一脚本（当前建议先聚焦 QueryRuns/QueryMailbox/MailboxAggregates）。
