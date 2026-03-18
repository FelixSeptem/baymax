# Multi-Agent Identifier Model

更新时间：2026-03-18

## 目标

为 Teams / Workflows / A2A 提供统一的标识字段与映射规则，避免跨模块重复定义和语义漂移。本文档是 R4-T0 阶段的字段单一事实源，后续实现与文档更新应以本文件为基线。

## 字段定义

| 字段 | 含义 | 作用域 | 是否必填 |
| --- | --- | --- | --- |
| `run_id` | 单次 runner 执行标识 | run | 是（现有字段） |
| `session_id` | 会话标识 | session | 否（现有字段） |
| `team_id` | 一次团队协作执行标识 | team | Teams 路径必填 |
| `workflow_id` | 一次工作流实例标识 | workflow | Workflow 路径必填 |
| `task_id` | 一个协作任务实例标识 | team/workflow/a2a | 任务路径必填 |
| `attempt_id` | 调度尝试实例标识 | scheduler/subagent | scheduler 路径必填 |
| `step_id` | workflow 内步骤标识 | workflow step | Workflow 路径必填 |
| `agent_id` | 参与协作的 agent 标识 | team/a2a | 多 agent 路径必填 |
| `peer_id` | A2A 对端 agent 标识 | a2a | A2A 路径必填 |

## 关联关系

- `run_id` 是执行主键，其他标识均应可回溯到 `run_id`。
- `team_id` 与 `workflow_id` 在同一 run 内可同时存在，但语义不同：
  - `team_id` 表示“谁协作”
  - `workflow_id` 表示“按什么流程协作”
- `task_id` 是跨 Teams/Workflow/A2A/Scheduler 的最小可追踪单元。
- `attempt_id` 是 scheduler lease 周期内的唯一尝试标识，用于幂等 terminal commit 与跨进程接管追踪。
- `step_id` 仅用于 workflow 内部有序节点，不用于 Teams 并行 worker 编号替代。

## 生成与稳定性规则

- 标识生成应确定性可重放，禁止依赖非稳定随机源导致同输入不可复现。
- 在 replay 场景中，事件重复写入不得生成新的业务标识。
- 推荐规则：
  - `team_id` / `workflow_id`：由宿主或编排器在入口创建并透传。
  - `task_id`：由编排层在 dispatch 时创建并保持到 terminal。
  - `step_id`：来自 DSL 声明键，不在运行时重写。

## 状态语义层与映射

多代理路径统一使用 timeline 语义层状态：
- `pending`
- `running`
- `succeeded`
- `failed`
- `skipped`
- `canceled`

子域可以保留 raw 状态，但必须可确定性映射到统一语义层。

| 子域 | raw 状态 | 统一语义层状态 |
| --- | --- | --- |
| teams | `pending` | `pending` |
| teams | `running` | `running` |
| teams | `succeeded` | `succeeded` |
| teams | `failed` | `failed` |
| teams | `skipped` | `skipped` |
| teams | `canceled` | `canceled` |
| workflow | `pending` | `pending` |
| workflow | `running` | `running` |
| workflow | `succeeded` | `succeeded` |
| workflow | `failed` | `failed` |
| workflow | `skipped` | `skipped` |
| workflow | `canceled` | `canceled` |
| a2a | `submitted` | `pending` |
| a2a | `running` | `running` |
| a2a | `succeeded` | `succeeded` |
| a2a | `failed` | `failed` |
| a2a | `canceled` | `canceled` |

约束：
- 映射规则必须稳定可回放。
- 同一事件重放不得改变映射结果。
- 任何新增 raw 状态必须在本文件补齐映射后才能进入实现。

## 事件映射规则

- Action Timeline 事件在相关路径中应携带：
  - Teams：`team_id`, `agent_id`, `task_id`
  - Workflow：`workflow_id`, `step_id`, `task_id`（A2A remote step 额外携带 `team_id/agent_id/peer_id`）
  - A2A：`workflow_id`, `team_id`, `step_id`, `agent_id`, `task_id`, `peer_id`
  - Scheduler/Subagent：`run_id`, `workflow_id`, `team_id`, `step_id`, `task_id`, `attempt_id`, `agent_id`, `peer_id`
- reason code 需要保持“路径 + 动作”可判别，例如：
  - `team.dispatch`, `team.collect`, `team.resolve`, `team.dispatch_remote`, `team.collect_remote`
  - `workflow.schedule`, `workflow.retry`, `workflow.resume`, `workflow.dispatch_a2a`
  - `a2a.submit`, `a2a.status_poll`, `a2a.sse_subscribe`, `a2a.sse_reconnect`, `a2a.delivery_fallback`, `a2a.version_mismatch`, `a2a.callback_retry`, `a2a.resolve`
  - `scheduler.enqueue`, `scheduler.claim`, `scheduler.heartbeat`, `scheduler.lease_expired`, `scheduler.requeue`
  - `subagent.spawn`, `subagent.join`, `subagent.budget_reject`

## Reason 命名空间规范

多代理新增 reason code 仅允许以下前缀：
- `team.*`
- `workflow.*`
- `a2a.*`
- `scheduler.*`
- `subagent.*`

约束：
- 禁止无前缀 reason（例如 `dispatch`、`retry`）。
- 禁止跨域复用前缀表达其他子域语义。
- 未通过前缀规范检查的变更视为阻断级失败。

## 诊断映射规则

- run 级摘要字段采用 additive 扩展，不破坏既有消费者：
  - Teams：`team_id`, `team_strategy`, `team_task_total`, `team_task_failed`, `team_task_canceled`, `team_remote_task_total`, `team_remote_task_failed`
  - Workflow：`workflow_id`, `workflow_status`, `workflow_step_total`, `workflow_step_failed`, `workflow_resume_count`, `workflow_remote_step_total`, `workflow_remote_step_failed`
  - A2A：`a2a_task_total`, `a2a_task_failed`, `peer_id`, `a2a_error_layer`, `a2a_delivery_mode`, `a2a_delivery_fallback_used`, `a2a_delivery_fallback_reason`, `a2a_version_local`, `a2a_version_peer`, `a2a_version_negotiation_result`
  - Scheduler/Subagent：`scheduler_backend`, `scheduler_queue_total`, `scheduler_claim_total`, `scheduler_reclaim_total`, `subagent_child_total`, `subagent_child_failed`, `subagent_budget_reject_total`
- 所有新增字段遵循 single-writer + idempotent replay 约束。

## 兼容性要求

- 对外保持 additive：新增字段可空、可缺省、默认不影响旧逻辑。
- 字段命名统一使用 snake_case，避免同义多名（例如 `agentId` 与 `agent_id` 并存）。
- 任何字段重命名必须经过 OpenSpec 变更并提供迁移窗口。
