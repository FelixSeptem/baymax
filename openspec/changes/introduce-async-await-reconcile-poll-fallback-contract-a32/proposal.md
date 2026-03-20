## Why

A31 引入了 `awaiting_report` 生命周期与 timeout 收敛，但当前异步终态主要依赖回报回调链路。  
当 callback 丢失、回调通道抖动或短暂不可达时，任务可能被误判为 timeout，缺少“主动对账”的自愈路径。

## What Changes

- 新增 async-await 对账能力：对处于 `awaiting_report` 的任务执行周期性 poll reconcile（status/result），作为 callback 之外的收敛后备路径。
- 在 async accepted 路径固化远端关联键（如 `remote_task_id`）持久化，确保跨重启和恢复场景仍可执行对账。
- 固化双来源终态收敛规则：`first_terminal_wins + record_conflict`，避免 callback/poll 互相覆盖导致终态翻转。
- 新增 `scheduler.async_await.reconcile.*` 配置域（默认关闭）并纳入 `env > file > default`、fail-fast 与热更新回滚语义。
- 扩展 Task Board 查询与任务可观测字段，支持查看终态来源与远端关联信息（只读，不引入控制面写操作）。
- 扩展 async-await 诊断聚合字段（additive），覆盖 reconcile 执行量、poll 收敛量、错误量与冲突量。
- 将 async-await reconcile contract suites 纳入 shared multi-agent gate 作为阻断项（Run/Stream 等价、memory/file parity、replay idempotency）。

## Capabilities

### New Capabilities
- `multi-agent-async-await-reconcile-contract`: 定义 `awaiting_report` 任务的 poll fallback、双来源终态收敛、冲突处理与幂等回放语义。

### Modified Capabilities
- `distributed-subagent-scheduler`: 扩展 scheduler 状态机与存储字段，支持 awaiting-report 对账调度与确定性收敛。
- `multi-agent-async-reporting`: 从“回报驱动”扩展为“回报 + 对账”双路径终态收敛契约。
- `multi-agent-task-board-query-contract`: 扩展任务查询可观测字段（终态来源/远端关联）并保持分页与游标契约不变。
- `runtime-config-and-diagnostics-api`: 新增 reconcile 配置域和 async-await 对账诊断聚合字段，保持兼容窗口语义。
- `go-quality-gate`: 纳入 async-await reconcile suites 到 shared contract gate 阻断路径。

## Impact

- 代码：
  - `orchestration/scheduler/*`（reconcile loop、状态收敛、存储/快照字段）
  - `orchestration/composer/*`（async accepted 远端关联键持久化、poll 结果提交）
  - `a2a/*`、`orchestration/invoke/*`（status/result poll 路径复用与错误分层归一）
  - `runtime/config/*`、`runtime/diagnostics/*`（配置与诊断字段扩展）
  - `integration/*`（reconcile fallback、冲突、重放、后端 parity 契约）
- 文档：
  - `README.md`
  - `docs/runtime-config-diagnostics.md`
  - `docs/mainline-contract-test-index.md`
  - `docs/development-roadmap.md`
- 兼容性：
  - 保持 `0.x` 阶段的 `additive + nullable + default` 兼容窗口，不引入平台化依赖，不承诺 exactly-once。
