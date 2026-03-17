## Context

当前系统在 R3 提供了网络桥接示例，但没有可复用的 A2A 协议抽象层。跨 Agent 调用仍依赖示例代码或业务自定义封装，不利于稳定扩展和组合测试。路线图已明确 A2A 是 R4 平台化方向，本设计聚焦最小互联能力与边界治理，不覆盖控制平面和多租户运营面。

本设计遵循现有架构约束：A2A 作为独立模块，复用现有 runtime config、event、diagnostics 能力，不绕过 single-writer 写入口。统一标识规则复用 `docs/multi-agent-identifier-model.md`。

## Goals / Non-Goals

**Goals:**
- 交付 A2A 最小 Client/Server 能力：提交任务、查询状态、回传结果。
- 交付 Agent Card 能力发现与最小路由决策输入。
- 交付 A2A 错误归一化并映射到 `types.ErrorClass`。
- 扩展 timeline/diagnostics 字段以支持 A2A 链路追踪。
- 明确 A2A 与 MCP 边界并形成可执行治理规则。

**Non-Goals:**
- 不实现控制平面、租户治理、RBAC、审计门户。
- 不覆盖复杂流控（跨区域调度、全局一致性事务）。
- 不在本期交付完整协议生态兼容（仅最小互联能力）。
- 不重构现有 MCP 传输栈。

## Decisions

### 1) A2A 采用独立模块而非复用 MCP 传输语义
- 方案：新增 `a2a` 模块承载任务生命周期与能力发现语义，MCP 保持工具调用语义。
- 原因：避免协作语义与工具语义耦合，保持扩展清晰。
- 备选：直接在 MCP 上扩展 Agent-to-Agent 方法。拒绝原因：职责混淆、迁移成本高。

### 2) 生命周期模型先收敛最小状态机
- 方案：最小状态集合 `submitted/running/succeeded/failed/canceled`，优先可观测与可回放。
- 原因：减少协议表面积，先建立稳定契约。
- 备选：一次性引入复杂长任务状态。拒绝原因：实现和测试成本过高。

### 3) Agent Card 采用静态发现 + 能力匹配
- 方案：基线支持静态 card 注册和能力字段匹配路由，不引入动态控制平面注册。
- 原因：满足最小互联需求并降低外部依赖。
- 备选：中心注册中心。拒绝原因：超出 library-first 基线范围。

### 4) A2A 链路观测复用现有事件与诊断管道
- 方案：A2A 事件进入 `observability/event.RuntimeRecorder`，run 级摘要新增 A2A 字段。
- 原因：保持 single-writer 与 replay 幂等语义一致。
- 备选：A2A 独立观测存储。拒绝原因：会制造并行事实源。

## Risks / Trade-offs

- [Risk] 协议边界定义不清导致 A2A 与 MCP 重叠  
  → Mitigation: 在 spec 与边界文档中固化职责划分，并补组合场景契约测试。

- [Risk] 跨进程故障路径复杂，错误映射不一致  
  → Mitigation: 定义统一错误映射表并纳入主干测试。

- [Risk] A2A 事件字段扩展导致消费者兼容风险  
  → Mitigation: additive 字段策略，保持旧字段不变且可空。

## Migration Plan

1. 增量加入 A2A DTO、配置与空实现接口（默认不启用）。  
2. 实现最小 Client/Server 生命周期路径与能力发现。  
3. 接入 timeline/diagnostics 字段并补齐 replay/幂等测试。  
4. 增加 A2A+MCP 组合契约测试并更新文档口径。  

回滚策略：
- 关闭 A2A 入口，退回现有单进程或示例网络桥接实现；
- 不影响 MCP 与 runner 现有路径语义。

## Open Questions

- A2A 推送通道首期是否只支持回调，还是同时支持 SSE？
- Agent Card 能力字段是否在首期支持版本协商，还是固定 schema 版本？
