## Why

当前仓库只有一份架构蓝图，缺少可执行的规格与任务拆解，团队无法并行开发、验收标准不统一、实现顺序容易漂移。现在将蓝图固化为 OpenSpec 变更，以便按里程碑推进并确保 API、行为与可观测性在实现前达成一致。

## What Changes

- 建立 Go `library-first` Agent Loop 的规范化交付路径，从骨架到可观测与稳定性分阶段落地。
- 定义统一 Runner 合同：单次会话的循环执行、工具调度、终止条件、错误语义与输出结构。
- 引入本地工具与 MCP 工具统一调度能力，覆盖 `stdio` 与 `SSE/HTTP` 双传输。
- 引入 `AGENTS.md + SKILL.md` 技能发现与冲突优先级策略，支持显式触发与语义触发。
- 定义事件流、OTel trace 与 JSON 日志对齐机制，确保 `run_id/trace_id` 可关联。
- 按 M0/M1/M2/M3 形成可验收的 roadmap 与测试计划。

## Capabilities

### New Capabilities
- `agent-runner-loop`: 统一定义 Runner 状态机、循环策略、终止条件与标准结果输出。
- `tool-dispatch-runtime`: 统一定义本地工具注册、参数校验、并发调度与错误回灌行为。
- `mcp-unified-transport`: 统一定义 MCP `stdio` 与 `SSE/HTTP` 调用语义、超时重试与事件一致性。
- `skill-loading-resolution`: 统一定义 AGENTS/SKILL 发现、触发、编译与冲突优先级。
- `observability-event-trace`: 统一定义 callback 事件、OTel span 结构与结构化日志关联字段。

### Modified Capabilities
- 无

## Impact

- Affected code: `core/runner`, `core/types`, `model/openai`, `tool/local`, `mcp/stdio`, `mcp/http`, `skill/loader`, `observability/*`。
- API impact: 新增公共接口 `Runner`, `ModelClient`, `Tool`, `MCPClient`, `SkillLoader`, `EventHandler` 及相关数据结构。
- External dependencies: OpenAI Responses 兼容客户端、MCP 客户端实现、OpenTelemetry SDK、JSON schema 校验组件。
- Delivery impact: 需要新增单元测试、Fake 集成测试、流式事件顺序校验和基准测试。
