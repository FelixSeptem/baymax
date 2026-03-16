## Context

现有架构已具备：
- H1/H1.5 Action Timeline（结构化事件与 phase 聚合）
- H2 Action Gate（执行前确认，timeout-deny）
- 多 agent 示例（07/08）与统一 runtime config/diagnostics

缺口在于：当 agent 运行中发现信息不足时，缺乏统一的“等待用户澄清并恢复”语义，导致多 agent 场景需自行拼装状态机，观测口径也不一致。

本提案实现 H3 最小闭环：单进程原生 `await_user/resume/cancel_by_user`，不引入持久化恢复。

## Goals / Non-Goals

**Goals**
- 在 runner 内提供 H3 原生生命周期状态，支持澄清请求后继续执行。
- 定义结构化 `clarification_request` payload，供调用方直接渲染与交互。
- 提供库接口回调，不绑定 CLI。
- 默认超时为 `cancel_by_user` 并 fail-fast 结束本次 run。
- Run/Stream 语义保持一致。
- 增补最小诊断计数与示例演示。

**Non-Goals**
- 不实现跨进程持久化恢复（checkpoint/store/replay）。
- 不引入 A2A/MCP 新协议扩展承载 H3。
- 不引入参数 schema 级风险判定（仍属于 H2/H4 后续）。

## Decisions

### 1) 采用 Runner 内“阻塞等待 + 回调返回”的最小模型
- 决策：定义 `ClarificationResolver`（library interface），runner 在需要澄清时阻塞等待回调结果。
- 原因：不改变现有主流程调用方式，最小入侵实现 H3。

### 2) 统一结构化 payload：`clarification_request`
- 决策：事件中增加 `clarification_request` 对象，最小字段包括：
  - `request_id`
  - `questions[]`
  - `context_summary`
  - `timeout_ms`
- 原因：前端/SDK 可直接消费，避免解析自由文本。

### 3) 默认超时策略为 `cancel_by_user`
- 决策：澄清等待超时直接终止 run，返回标准化错误分类。
- 原因：与 fail-fast 基线一致，避免不确定挂起占用。

### 4) 诊断先采用最小计数器
- 决策：新增 `await_count`、`resume_count`、`cancel_by_user_count`。
- 原因：先满足可观测与验收闭环，后续再扩展细粒度延迟分布。

### 5) 多 agent 示例采用“增量改造现有示例”
- 决策：优先在 `examples/07-multi-agent-async-channel` 增加 clarification-agent 流程。
- 原因：进程内语义与本期单进程边界一致，调试成本最低。

## Risks / Trade-offs

- [Risk] 回调实现阻塞导致 run 长时间占用资源  
  → Mitigation: 强制 timeout + `cancel_by_user` 默认策略。

- [Risk] Run/Stream 行为分叉  
  → Mitigation: 增加 Run/Stream 语义等价契约测试。

- [Risk] 结构化 payload 字段不足导致前端改造二次返工  
  → Mitigation: 固定最小字段 + 明确可扩展字段策略（向后兼容追加）。

## Migration Plan

1. `core/types` 增加 clarification 相关接口与 DTO。
2. `runtime/config` 增加 H3 配置（enabled/timeout/timeout_policy）。
3. `core/runner` 接入 await/resume/cancel 生命周期，统一 Run/Stream 路径。
4. `observability/event` 与 `runtime/diagnostics` 增加最小计数与 payload 映射。
5. 增改 `examples/07` 演示 clarification-agent。
6. 补齐契约测试与文档更新。

## Open Questions

- 本期是否需要将 `clarification_request.context_summary` 长度限制纳入配置（建议后续补充）。
