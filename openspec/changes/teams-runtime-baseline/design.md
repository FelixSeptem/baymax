## Context

现状是“有 multi-agent 示例、无 Teams runtime 抽象”。`examples/07` 与 `examples/08` 演示了模式，但业务可复用的角色定义、协作策略、生命周期状态机、诊断字段尚未沉淀为平台契约。与此同时，路线图已经把 Teams/Workflow/A2A 列入 R4 方向，需要先交付一层可回归的 Teams 基线，避免后续每个能力分别发明字段和状态语义。

本设计聚焦 T1 的单进程 Teams MVP，不覆盖分布式调度和 control plane。统一标识模型采用 `docs/multi-agent-identifier-model.md` 作为单一事实源。

## Goals / Non-Goals

**Goals:**
- 在 library-first 前提下提供 Teams 一等抽象和可组合执行接口。
- 覆盖三类协作策略：`serial`、`parallel`、`vote`，且结果确定性可回放。
- 与现有 Run/Stream 语义保持等价，不破坏现有 tool/mcp/skill 路径。
- 扩展最小配置与诊断字段，并保持 single-writer + idempotency 口径。
- 明确 Teams 模块边界，避免直接膨胀 `core/runner` 主状态机。

**Non-Goals:**
- 不实现跨进程 Teams 调度或远程 worker 池。
- 不引入租户/RBAC/control plane。
- 不引入新的 provider SDK 依赖或改写 CA1-CA4 策略。
- 不在本期实现 Workflow DSL 与 A2A 协议层（仅预留接口）。

## Decisions

### 1) Teams 采用独立编排模块承载，不直接嵌入 runner 状态机
- 方案：新增编排模块（建议 `orchestration/teams`），通过现有 runner 接口执行 agent step。
- 原因：保持 `core/runner` 单 run 循环职责稳定，降低跨能力耦合。
- 备选：直接扩展 `core/runner` 为多代理状态机。拒绝原因：风险高，且会放大回归面。

### 2) 策略接口最小化并要求确定性收敛
- 方案：统一策略接口 `Plan -> Dispatch -> Collect -> Resolve`，`vote` 默认固定 tie-break 规则。
- 原因：保证 replay 可复现，避免“同输入不同收敛”。
- 备选：先做启发式动态策略。拒绝原因：可观测与测试成本高。

### 3) 统一标识采用层次化映射并贯穿事件/诊断
- 方案：在 run 语义上引入 `team_id/agent_id/task_id/role/strategy`，并映射到 timeline/diagnostics。
- 原因：为 Workflow/A2A 复用字段打底，避免二次迁移。
- 备选：仅在示例层打印字段。拒绝原因：无法形成平台契约。

### 4) 失败与取消语义与现有治理保持一致
- 方案：延续 `fail_fast/best_effort`、`cancel.propagated`、背压 reason 口径，Teams 不定义第二套错误语义。
- 原因：降低用户心智负担，保持 Run/Stream 可比性。
- 备选：Teams 定义独立错误类。拒绝原因：会割裂主干 taxonomy。

## Risks / Trade-offs

- [Risk] 并行与投票策略会提高并发复杂度  
  → Mitigation: 先限制策略参数面，优先补齐 integration + race + benchmark 回归。

- [Risk] 字段扩展可能导致历史消费者解析分叉  
  → Mitigation: 全部新增字段保持 additive，旧字段语义不变。

- [Risk] 模块边界漂移，Teams 逻辑反向侵入 runner  
  → Mitigation: 增加边界检查与评审清单，约束责任归属。

## Migration Plan

1. 新增 Teams 配置与字段模型（默认关闭或不激活，不影响现网路径）。  
2. 落地串行策略，再增量开启并行/投票策略。  
3. 补齐 timeline/diagnostics 字段并完成 replay 对齐。  
4. 通过契约测试后再开放示例与文档入口。  

回滚策略：
- 关闭 Teams 开关或回退调用入口到单 agent run；
- 保留既有 runner 路径，不触发兼容性破坏。

## Open Questions

- `vote` 策略中“权重投票”是否进入 T1，还是仅支持等权投票？
- Teams API 对外是否直接暴露泛型任务上下文，还是通过固定 DTO 收敛？
