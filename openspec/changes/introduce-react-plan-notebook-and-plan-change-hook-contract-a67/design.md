## Context

现有 ReAct 主链路已具备：
- A56：Run/Stream parity、工具调用闭环、预算与终止 taxonomy；
- A58：跨域决策解释链与 precedence；
- A61：可观测回放与跨后端互操作。

但“计划”仍缺少统一合同层：同一任务在不同入口下计划修订策略不一致、计划变更原因不稳定、恢复后计划历史缺失，导致复杂任务场景可解释性与回放阻断能力不足。roadmap 已明确 A67 需要“复用 A56 终止 taxonomy 与 A65 hook 合同，不新增平行 ReAct 主循环”，因此本设计以“计划治理合同层”收口，不改写既有 ReAct loop 主语义。

## Goals / Non-Goals

**Goals:**
- 定义 ReAct plan notebook 生命周期：`create|revise|complete|recover`。
- 定义 plan-change hook 合同：before/after 时序、上下文透传、失败策略。
- 定义配置治理：`runtime.react.plan_notebook.*`、`runtime.react.plan_change_hook.*`。
- 定义可观测与回放：A67 additive 字段、`react_plan_notebook.v1` fixture、drift taxonomy。
- 保证 Run/Stream、memory/file 后端下的计划生命周期语义等价。

**Non-Goals:**
- 不新增平行 ReAct loop，不改变 A56 已冻结终止 taxonomy。
- 不引入托管计划控制面、远程计划服务或平台化 UI/RBAC。
- 不在 A67 内推进性能专项（A64 负责）或交付示例收口（A62 负责）。

## Decisions

### Decision 1: 引入“Notebook 状态机 + 版本化历史”作为计划事实层

- 方案：计划实体包含 `plan_id`、`version`、`status`、`history[]`，动作仅允许 `create|revise|complete|recover`，`version` 单调递增。
- 备选：仅记录最新计划文本，不保留历史。
- 取舍：历史版控是 drift 分析与恢复复放的前提，且不改变既有 loop 控制流。

### Decision 2: 计划变更 hook 固定为 before/after 双相，复用现有 hook 执行链

- 方案：在计划变更边界发出 `before_plan_change` / `after_plan_change` 两类 hook 事件，遵循既有 middleware/hook 顺序与错误冒泡规则。
- 备选：在每轮 reasoning 末尾隐式回调一次统一 hook。
- 取舍：双相 hook 更适合表达“变更前决策 + 变更后审计”，且可精确做到 deterministic 回放。

### Decision 3: 失败语义采用 `fail_fast|degrade`，并固定降级可观测字段

- 方案：
  - `fail_fast`：hook 或 notebook 校验失败即终止本次计划变更；
  - `degrade`：跳过当前计划变更并记录 `react_plan_hook_status=degraded`。
- 备选：统一强制 fail-fast。
- 取舍：保留 degrade 以适应线上弹性，但通过 additive 字段确保可观测可审计。

### Decision 4: 计划恢复语义复用现有恢复事实源，A67 只定义合同层

- 方案：A67 仅定义 notebook 恢复接口与幂等语义，不新建独立存储事实源；恢复入口复用现有 session/recovery 接缝。
- 备选：A67 单独引入计划专用持久化引擎。
- 取舍：避免与 A66 状态统一合同重复建设，减少跨提案冲突。

### Decision 5: 回放与门禁采用 fixture-first + 独立 gate

- 方案：新增 `react_plan_notebook.v1`，覆盖 create/revise/complete/recover、hook fail_fast/degrade、Run/Stream parity，新增独立 gate 并接入质量门禁。
- 备选：仅依赖 integration 用例，不做独立 replay fixture。
- 取舍：独立 fixture + gate 对计划语义漂移阻断更稳定，也符合现有 contract-first 治理方式。

## Risks / Trade-offs

- [Risk] 计划状态机接线点增加，可能引入 loop 行为回归。  
  → Mitigation: 限定 A67 只在计划变更边界插桩，不改 loop 终止与预算判断顺序。

- [Risk] degrade 策略可能掩盖真实计划漂移。  
  → Mitigation: 冻结 `react_plan_hook_status` 与 drift 分类；degrade 必须可观测且可回放。

- [Risk] 与 A66 恢复相关实现产生边界重叠。  
  → Mitigation: A67 明确“不新增事实源”，仅消费既有恢复能力并定义计划层合同。

- [Risk] Run/Stream 计划事件顺序在并发下出现分叉。  
  → Mitigation: 固定 before/after hook 的 step-boundary 触发点并增加 parity contract suites。

## Migration Plan

1. 配置层：在 `runtime/config` 增加 `runtime.react.plan_notebook.*` 与 `runtime.react.plan_change_hook.*`，实现 fail-fast 与热更新回滚。
2. 模型层：定义 notebook 数据结构与动作状态机（create/revise/complete/recover + version/history）。
3. 接线层：在 ReAct loop 的计划变更边界接入 notebook 与 before/after hook。
4. 观测层：在 `runtime/diagnostics` 与 `RuntimeRecorder` 增加 A67 additive 字段并保持单写幂等。
5. 回放层：新增 `react_plan_notebook.v1` fixture、drift 分类与 mixed fixture 兼容测试。
6. 门禁层：新增 `check-react-plan-notebook-contract.sh/.ps1` 并接入 `check-quality-gate.*`。
7. 文档层：同步 runtime config/diagnostics、contract index、roadmap 与 README。

回滚策略：
- 配置回滚：热更新非法配置自动回滚到上一个有效快照；
- 功能回滚：关闭 `runtime.react.plan_notebook.enabled` 与 `runtime.react.plan_change_hook.enabled` 即恢复到 A67 前行为；
- 数据兼容：新增字段保持 additive，旧消费者可安全忽略。

## Open Questions

- None. A67 按 roadmap 一次性收口 ReAct 计划治理同域需求，不再拆平行提案。
