## Context

仓库当前已有并发、HITL、Action Timeline、diagnostics 回放等基础能力，但流程编排仍停留在“代码驱动”的应用层，缺少统一 DSL 契约和确定性执行模型。R4 路线图中 Workflow 已作为平台化方向，当前需要先交付最小可执行、可回归的 workflow baseline。

本设计覆盖 DSL 语法、计划校验、执行语义、恢复点与观测字段，不覆盖可视化编排器与 control plane。统一标识规则复用 `docs/multi-agent-identifier-model.md`。

## Goals / Non-Goals

**Goals:**
- 提供最小 workflow DSL（YAML/JSON）与静态校验能力。
- 提供确定性执行语义（稳定拓扑排序、分支、重试、超时）。
- 提供最小 checkpoint/resume 契约，支持故障后恢复与回放。
- 保持与 runner/tool/mcp/skill 路径可组合，且 Run/Stream 语义等价。
- 保持边界清晰：workflow 编排模块独立于 `core/runner`。

**Non-Goals:**
- 不提供可视化流程设计器。
- 不实现跨节点分布式调度与全局事务。
- 不引入多租户治理与 RBAC。
- 不在本期实现 A2A 协议互联（仅预留接入点）。

## Decisions

### 1) DSL 先最小可用，再增量扩展语法
- 方案：首期仅支持 `step`、`depends_on`、`condition`、`retry`、`timeout`，拒绝复杂脚本语法。
- 原因：控制复杂度，先保证可验证与可回放。
- 备选：一次性引入完整表达式 DSL。拒绝原因：实现与安全风险高。

### 2) 调度采用稳定拓扑排序保证确定性
- 方案：对同层可执行 step 按稳定键排序调度，保证等价输入下执行序一致。
- 原因：降低回放漂移，便于诊断与 benchmark 对比。
- 备选：纯并发抢占式执行。拒绝原因：不可重现。

### 3) checkpoint 采用最小快照模型
- 方案：仅持久化 workflow 级进度与 step 终态，不持久化 provider 内部细节。
- 原因：先满足恢复最小闭环，避免过早绑定存储模型。
- 备选：全量事件 sourcing。拒绝原因：超出本期范围。

### 4) workflow 与 runtime 观测统一字段
- 方案：workflow 输出统一 `workflow_id`、`step_id`、`step_status`、`step_attempt`，并映射到 timeline/diagnostics。
- 原因：与 Teams/A2A 字段对齐，减少后续迁移成本。
- 备选：workflow 自定义独立日志体系。拒绝原因：破坏 single-writer 口径。

## Risks / Trade-offs

- [Risk] DSL 过于简化导致业务覆盖不足  
  → Mitigation: 先确保扩展点（condition/retry）可插拔，按场景增量补充语法。

- [Risk] checkpoint 粒度不足影响恢复质量  
  → Mitigation: 首期明确恢复边界并补齐回放差异告警。

- [Risk] 与现有并发背压逻辑发生冲突  
  → Mitigation: workflow 只编排执行顺序，不重写底层 dispatch/backpressure 语义。

## Migration Plan

1. 引入 workflow schema 与静态校验器，默认仅在显式调用时生效。  
2. 引入 workflow 执行器并打通最小 step 执行路径。  
3. 增量接入 checkpoint/resume 与 timeline/diagnostics 字段。  
4. 补齐契约测试与文档，再开放示例入口。  

回滚策略：
- 关闭 workflow 入口，回退到应用侧手工流程编排；
- 保持 runner 主路径与现有 API 不变。

## Open Questions

- `condition` 表达式是否允许宿主回调，还是仅支持内建操作符？
- checkpoint 默认存储后端是否先限定 file，实现后再扩展到 db？
