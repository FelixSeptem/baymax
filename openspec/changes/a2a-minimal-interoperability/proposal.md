## Why

仓库已经具备单进程多代理示例与网络桥接示例，但缺少正式的 A2A 协议层能力。当前跨 Agent 协作依赖业务自定义协议和临时适配，导致能力发现、任务生命周期、状态查询与错误语义无法稳定复用，也难以和现有 observability/diagnostics 契约对齐。

引入最小 A2A 互联基线，可以在保持 library-first 的同时，为跨进程协作提供可验证的 client/server 契约，并明确与 MCP 的互补边界。

## What Changes

- 新增 A2A 最小互联能力：Client/Server 基线、任务提交、状态查询、结果回传。
- 新增 Agent Card 能力发现与最小路由决策输入。
- 新增 A2A 错误归一化与 runtime `types.ErrorClass` 映射规则。
- 扩展 A2A 配置、timeline 与 diagnostics 字段，保证与现有 single-writer 管道兼容。
- 明确 A2A 与 MCP 边界：A2A 负责 Agent 协作，MCP 负责工具集成。

## Capabilities

### New Capabilities
- `a2a-minimal-interoperability`: 提供最小 A2A Client/Server 互联、能力发现与生命周期语义。

### Modified Capabilities
- `runtime-config-and-diagnostics-api`: 增加 A2A 配置与观测字段契约。
- `action-timeline-events`: 增加 A2A 路径关联字段与 reason 语义。
- `runtime-module-boundaries`: 增加 A2A 与 MCP 的职责边界约束。

## Impact

- 影响代码：
  - 新增 `a2a/*` 模块（client/server/card/router）
  - `core/types`（A2A 请求/响应 DTO 与错误映射）
  - `runtime/config`、`runtime/diagnostics`、`observability/event`
- 影响测试：
  - A2A 协议契约测试（submit/status/result）
  - A2A+MCP 组合场景测试
  - 事件与诊断字段回放一致性测试
- 影响文档：
  - `docs/runtime-config-diagnostics.md`
  - `docs/runtime-module-boundaries.md`
  - `docs/development-roadmap.md`
  - `docs/multi-agent-identifier-model.md`
