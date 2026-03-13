## Why

Action Timeline H1 已完成结构化事件发射，但当前 diagnostics 仍缺少 phase 级聚合视图，导致运行后分析只能依赖原始事件回放，难以稳定支持容量评估、失败分布诊断和趋势观测。现在收敛 H1.5 可以在不改变 runner 主状态机的前提下，补齐可观测闭环并为后续 HITL/CA3 提供统一基线。

## What Changes

- 在 `runtime/diagnostics` 扩展 run 级 Action Timeline 聚合字段，按 phase 输出计数与延迟统计。
- 聚合字段包含最小集：`count_total`、`failed_total`、`canceled_total`、`skipped_total`、`latency_ms`、`latency_p95_ms`。
- 聚合逻辑默认启用，不新增开关；保持 H1 结构化事件默认启用语义一致。
- 强化幂等语义：同一 run 的 timeline 重放不得重复计数。
- 新增 Run/Stream 语义一致性验收：同场景 phase 状态分布等价（不要求逐事件一一对应）。
- 同步文档：`README.md`、`docs/runtime-config-diagnostics.md`、`docs/development-roadmap.md`。
- 明确范围边界：本期不引入 Action Gate / pause-resume / HITL 状态机改造。

## Capabilities

### New Capabilities
- 无

### Modified Capabilities
- `action-timeline-events`: 增加 timeline 聚合可观测语义（phase 级计数与延迟分布）及 Run/Stream 等价约束。
- `runtime-config-and-diagnostics-api`: 扩展 diagnostics run 记录契约，纳入 timeline 聚合字段与重放幂等要求。

## Impact

- 影响模块：`runtime/diagnostics`、`observability/event`、`core/runner`（兼容性校验）、相关测试。
- API 影响：`RecentRuns` 返回结构新增 timeline 聚合字段（向后兼容新增字段）。
- 风险点：聚合计算与幂等去重错误可能引发统计偏差；需通过重放测试和并发测试兜底。
- 质量门禁：`go test ./...`、`go test -race ./...`、`golangci-lint` 与 docs consistency 全通过。
