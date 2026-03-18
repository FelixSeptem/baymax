## Why

Teams、Workflow、A2A 当前已经分别具备最小可用能力，但缺少跨模块组合契约，导致“workflow 编排远端 agent 任务”与“teams 本地+远端混编”仍需业务侧拼装。A4 正在加固 A2A delivery/version，本提案顺势补齐编排层与契约层连接，把 R4 从“模块并列可用”推进到“主链路可组合可回归”。

## What Changes

- 为 Workflow 增加 A2A 步骤类型与执行语义（在既有 adapter 架构下接入，不破坏现有 `runner|tool|mcp|skill` 语义）。
- 为 Teams 增加本地 worker 与 A2A remote worker 的混编执行语义，收敛并行/取消/失败策略。
- 统一跨域关联字段透传与映射（`team_id/workflow_id/task_id/step_id/agent_id/peer_id`），并保持 replay 幂等。
- 扩展 timeline reason 与 run diagnostics 聚合，覆盖组合编排关键路径与降级原因。
- 增加跨域契约测试矩阵（Teams + Workflow + A2A + MCP 组合场景），并收敛主干索引文档。

## Capabilities

### New Capabilities
- `multi-agent-composed-orchestration`: 定义 Teams、Workflow、A2A 组合编排的主链路契约、字段映射与失败收敛语义。

### Modified Capabilities
- `workflow-deterministic-dsl`: 扩展 workflow step 执行适配，支持 A2A 远端步骤并保持确定性与 Run/Stream 等价。
- `teams-collaboration-runtime`: 扩展 Teams 任务执行模型，支持 local/remote worker 混编与统一失败/取消语义。
- `a2a-minimal-interoperability`: 将 A2A 从独立互联能力扩展为可被编排层复用的标准执行域（保留与 MCP 边界）。
- `runtime-config-and-diagnostics-api`: 增加组合编排所需配置项与 additive 诊断字段契约。
- `action-timeline-events`: 增加组合编排 reason 与关联字段要求，确保跨域可观测性一致。
- `runtime-module-boundaries`: 强化 orchestration 与 a2a/mcp 责任边界及单写入口约束。

## Impact

- 影响代码：`orchestration/workflow/*`、`orchestration/teams/*`、`a2a/*`、`runtime/config/*`、`runtime/diagnostics/*`、`observability/event/*`、`integration/*`。
- 影响测试：新增/更新 workflow A2A step、teams local+remote 混编、A2A+MCP 组合边界、Run/Stream 等价与 replay 幂等契约测试。
- 影响文档：`docs/runtime-config-diagnostics.md`、`docs/runtime-module-boundaries.md`、`docs/v1-acceptance.md`、`docs/mainline-contract-test-index.md`、`docs/development-roadmap.md`。
- 兼容性：对外字段与行为采用 additive 扩展，不移除既有路径；如存在默认策略变化需在文档和迁移说明中显式标注。
