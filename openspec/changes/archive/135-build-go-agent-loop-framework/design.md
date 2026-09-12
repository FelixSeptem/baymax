## Context

当前项目处于蓝图阶段，尚未形成可执行的规格合同。目标是交付一个 Go `library-first` Agent Loop 框架，统一模型调用、工具调度、MCP 双传输、技能加载和可观测能力。主要约束包括：
- 对外 API 需要稳定，避免后续适配层变更引发破坏性升级。
- 行为需可验证，必须通过结构化事件和测试场景定义行为边界。
- 需要兼容 OpenAI Responses 与 MCP 两类生态接口。
- v1 不做分布式和持久化恢复，仅在单进程会话内保证正确性与可观测。

## Goals / Non-Goals

**Goals:**
- 建立可分阶段交付的技术设计，支撑 M0/M1/M2/M3 roadmap。
- 在 Runner 层定义统一状态机和策略参数，统一错误分类与终止语义。
- 在工具层实现本地工具与 MCP 工具统一调度路径与命名空间。
- 在可观测层提供统一事件模型与 OTel trace/log 关联。
- 在技能层定义 AGENTS/SKILL 发现、编译、生效与冲突优先级。

**Non-Goals:**
- 不实现分布式任务编排、跨会话持久化恢复。
- 不在 v1 内置复杂 RBAC、审计平台或多租户控制面。
- 不引入 Go 动态插件机制作为技能扩展方式。

## Decisions

### Decision 1: `core/runner` 采用显式状态机循环
- Choice: 采用 `Init -> LoadSkills -> ModelStep -> DispatchToolCalls -> MergeToolResults -> DecideNext -> Finalize/Abort` 的显式状态推进。
- Rationale: 便于测试每个状态分支和故障转移路径，减少隐式条件分支导致的不可预测行为。
- Alternative considered: 使用递归或单函数流程驱动。该方式实现更快，但可观测粒度与可测试性较差。

### Decision 2: 统一 `LoopPolicy` 作为运行时行为总开关
- Choice: 默认 `MaxIterations=12`、`MaxToolCallsPerIteration=8`、`StepTimeout=60s`、`ModelRetry=2`、`ToolRetry=1`。
- Rationale: 将运行时控制集中化，便于调用方按场景覆盖，同时保证默认行为有边界。
- Alternative considered: 分散配置到各模块。分散配置会导致行为组合难以推理。

### Decision 3: 工具调用统一结果模型
- Choice: 工具返回 `ToolResult{content, structured, error?}`，错误作为数据回灌模型。
- Rationale: 保持 loop 连续性，让模型参与恢复决策；避免工具失败直接中断整轮。
- Alternative considered: 工具失败即 fail-fast。适合强一致场景，但会降低复杂任务完成率。

### Decision 4: MCP 双传输保持语义一致，差异封装在适配器内部
- Choice: `mcp/stdio` 与 `mcp/http` 必须对齐超时、重试、取消与事件语义。
- Rationale: 上层 Runner 不关心传输细节，避免“同一工具不同传输不同行为”。
- Alternative considered: 暴露传输差异到上层。将导致调用方逻辑分叉和维护成本提升。

### Decision 5: 技能系统采用“系统内建 > AGENTS > SKILL”优先级
- Choice: 冲突时按固定优先级处理，工具冲突仅保留显式启用项。
- Rationale: 可解释、可审计，便于定位指令来源与冲突原因。
- Alternative considered: 最近匹配优先。行为更灵活但不可预测。

### Decision 6: 可观测采用“事件 + trace + JSON 日志”三位一体
- Choice: 事件流作为外部消费契约，OTel span 作为内部性能与链路视图，日志承担审计和排障。
- Rationale: 单一观测通道不足以覆盖 CLI/UI、性能分析和线上排障三类需求。
- Alternative considered: 只保留日志。无法可靠支持实时 UI 和跨组件链路追踪。

## Risks / Trade-offs

- [Risk] 状态机过早固化导致扩展困难 → Mitigation: 在 `core/types` 保留可扩展字段，新增状态需向后兼容。
- [Risk] 工具错误回灌可能增加无效迭代 → Mitigation: 用 `MaxIterations` 与重试阈值限制，记录 `warning` 并可切换 fail-fast。
- [Risk] MCP HTTP 重连导致重复调用或乱序事件 → Mitigation: 为每次调用分配稳定 call-id，事件携带序号并在恢复后校验。
- [Risk] 技能语义触发过宽导致误触发 → Mitigation: 语义触发仅作为候选，默认以显式提及优先。
- [Risk] 观测数据过多增加性能开销 → Mitigation: 默认事件最小集，trace 采样率可配置。

## Migration Plan

1. M0 建立无工具闭环与基础事件，验证 Runner 合同可用。
2. M1 引入本地工具闭环与策略控制，补齐错误回灌路径。
3. M2 接入 MCP 双传输并统一语义，完成连接与重试机制。
4. M3 接入技能系统与 OTel，完成集成测试与基准测试。
5. 每阶段通过后才进入下一阶段，若未达验收标准则不推进。

## Open Questions

- OpenAI Responses 在流式 tool call 的边界事件是否需要额外中间态映射？
- MCP HTTP 在长连接环境下的默认心跳和重连上限应如何按部署环境分层配置？
- 事件版本化策略（v1/v2）是否要在 M3 前冻结，避免消费端后续迁移成本？
