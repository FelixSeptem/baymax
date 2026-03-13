## Why

当前运行时已经具备较完整的事件与诊断能力，但缺少对“执行路径”稳定、可消费的一致契约，导致用户侧很难在 Run/Stream 两条路径上统一渲染 Agent 行为时间线。随着后续 HITL、A2A、CA3/CA4 演进，先收敛 Action Timeline 语义可以降低后续改造成本并减少前端/编排侧适配分叉。

## What Changes

- 新增 Action Timeline 结构化事件能力，统一阶段与状态语义，覆盖 Run/Stream 双路径。
- 增加标准状态枚举：`pending`、`running`、`succeeded`、`failed`、`skipped`、`canceled`。
- 将 `context_assembler` 作为独立 phase 纳入 timeline 输出，便于后续 CA3/CA4 观测扩展。
- 默认启用 timeline 事件输出；本期不新增诊断聚合字段。
- 在文档与规格中补充“可观测性后续收敛”TODO，明确后续会把 timeline 聚合指标纳入 diagnostics。

## Capabilities

### New Capabilities
- `action-timeline-events`: 定义并输出可消费的 Action Timeline 结构化事件契约（phase/status/order/reason），并保证 Run/Stream 语义一致。

### Modified Capabilities
- `runtime-config-and-diagnostics-api`: 增补 timeline 事件启用与默认行为约束，以及 diagnostics 聚合字段后续收敛的 TODO 轨迹。

## Impact

- 影响模块：`core/runner`、`observability/event`、`runtime/diagnostics`（仅兼容性评估，不新增聚合字段）、`core/types`。
- 影响文档：`README.md`、`docs/runtime-config-diagnostics.md`、`docs/development-roadmap.md`。
- API 影响：新增 timeline 事件与状态枚举；保持现有事件与 diagnostics 字段向后兼容。
- 质量门禁：需补齐 Run/Stream 契约测试并保持 `go test ./...`、`go test -race ./...`、`golangci-lint` 全通过。
