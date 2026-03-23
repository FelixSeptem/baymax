## Context

A29 已提供 Task Board 查询能力，但当前主线仍缺少库级任务控制入口。调用方虽然可以读取 `queued|running|awaiting_report|failed|dead_letter`，却不能通过统一契约执行受约束的人工恢复动作（例如取消队列任务、重试 dead-letter 任务），导致以下问题：

- 任务恢复依赖调用方自定义流程，跨模块语义不一致；
- 控制请求缺少统一幂等键，重复请求可能造成计数膨胀；
- 手工干预事件缺少 canonical reason，timeline/diagnostics 对账困难。

当前仓库定位仍为 `library-first + contract-first`，目标是在不引入平台化控制面的前提下，补齐“可查 + 可控”的最小闭环。

## Goals / Non-Goals

**Goals:**
- 在 `orchestration/scheduler` 提供库级 Task Board control API，支持 `cancel` 与 `retry_terminal`。
- 固化状态约束矩阵与 fail-fast 语义：
  - `cancel` 仅支持 `queued|awaiting_report`；
  - `running` 默认 fail-fast，不做强制中断；
  - `retry_terminal` 仅支持 `failed|dead_letter -> queued`。
- 引入 `operation_id` 幂等语义，保证 replay/重复请求不膨胀 logical outcome。
- 扩展 runtime 配置与 diagnostics，并纳入 shared/quality gate 阻断路径。

**Non-Goals:**
- 不引入平台化控制面（UI/RBAC/多租户运维面）。
- 不新增任务迁移/重分配（reassign）与优先级在线改写。
- 不改变 Task Board query 的只读契约语义。
- 不引入对 in-flight `running` 任务的强制 kill/interrupt 机制。

## Decisions

### 1) 控制面落在 scheduler 领域，采用显式 action API
- 决策：在 `orchestration/scheduler` 增加 Task Board control API（例如 `ControlTask` 或等价入口），统一承载 manual control 语义。
- 原因：状态机归属在 scheduler，避免把写路径扩散到 diagnostics 或 query 域。
- 备选：在 `runtime/diagnostics` 暴露写操作。拒绝原因：职责越界，破坏单写可观测边界。

### 2) `running` 不支持 manual cancel，保持 fail-fast
- 决策：`cancel` 仅允许 `queued|awaiting_report`，对 `running` 返回明确 fail-fast 错误。
- 原因：当前缺少跨模块安全中断协议，强制终止会引入不可控副作用与语义漂移。
- 备选：支持 best-effort kill。拒绝原因：难以保证一致性与回滚边界。

### 3) 手工重试仅允许 terminal 重入，并设置预算
- 决策：`retry_terminal` 仅允许 `failed|dead_letter -> queued`，并由
  `scheduler.task_board.control.max_manual_retry_per_task=3` 约束预算。
- 原因：限制人工重试风暴，保持与现有 retry/backoff 治理协同。
- 备选：无限制重试。拒绝原因：容易造成抖动与统计污染。

### 4) 幂等键采用 `operation_id` 必填
- 决策：每次 control 请求必须携带 `operation_id`，重复请求返回幂等结果，不重复改变计数。
- 原因：与 recovery/replay 语义对齐，保障跨会话重放可预测。
- 备选：隐式幂等（由 task+action 推导）。拒绝原因：无法覆盖同一任务多次合法控制请求场景。

### 5) 新增 canonical reason，不引入新顶级 namespace
- 决策：新增 `scheduler.manual_cancel` 与 `scheduler.manual_retry`，沿用既有 `scheduler.*` 命名空间。
- 原因：与 A16/A33 的“复用既有 namespace”原则一致，降低 taxonomy 扩散。
- 备选：引入 `taskboard.*`。拒绝原因：跨域 reason 增长，治理成本高。

## Risks / Trade-offs

- [Risk] 手工控制接口被滥用导致状态抖动  
  -> Mitigation: 默认关闭 + 明确状态矩阵 + manual retry 预算上限 + gate 覆盖。

- [Risk] 幂等实现不完整导致重复请求计数膨胀  
  -> Mitigation: `operation_id` 必填，补 replay/idempotency contract 套件并纳入阻断。

- [Risk] 新增 reason 与现有 taxonomy 漂移  
  -> Mitigation: 更新 action-timeline 与 shared contract snapshot，加入 drift guard。

- [Risk] memory/file backend 行为分叉  
  -> Mitigation: 增加 backend parity + restore/replay 一致性测试并接入 shared gate。

## Migration Plan

1. 在 scheduler 领域引入 control action 请求/响应模型，补齐参数校验与状态机入口。
2. 实现 `cancel` 与 `retry_terminal` 两类动作及状态约束；接入 `operation_id` 幂等表。
3. 扩展 timeline reason 与 diagnostics additive 字段，确保可观测一致。
4. 扩展 `runtime/config`：新增 `scheduler.task_board.control.*`，纳入启动/热更新 fail-fast + 原子回滚。
5. 补齐 unit/integration 合同测试：状态矩阵、幂等、Run/Stream 等价、memory/file parity、replay 稳定性。
6. 将 suites 接入 `check-multi-agent-shared-contract.*` 与 `check-quality-gate.*`。
7. 同步 README / roadmap / runtime-config-diagnostics / mainline-contract-test-index。

回滚策略：
- 通过 `scheduler.task_board.control.enabled=false` 关闭能力；
- 如需快速回退实现，可仅保留 query 只读路径，不影响既有运行主链路。

## Open Questions

无阻塞项，按推荐值推进：
- `enabled=false`
- `max_manual_retry_per_task=3`
- `running` 上 `cancel` 继续 fail-fast
