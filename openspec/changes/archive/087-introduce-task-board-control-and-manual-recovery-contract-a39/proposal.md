## Why

A29 已补齐 Task Board 只读查询，但主线仍缺少库级“任务控制”契约。当前调用方无法通过统一 API 对 `queued|awaiting_report|failed|dead_letter` 任务执行可回归的人工干预（取消、重试），排障与恢复仍依赖临时代码路径，难以保证幂等与 reason taxonomy 一致性。

## What Changes

- 新增 Task Board control 契约能力，提供库级控制入口（非平台化控制面）：
  - `cancel`：仅支持 `queued|awaiting_report`，`running` 默认 fail-fast（不做强杀）。
  - `retry_terminal`：仅支持 `failed|dead_letter -> queued`。
- 控制请求新增幂等键 `operation_id`，重复提交必须幂等收敛，不重复膨胀计数。
- 扩展 scheduler canonical reason taxonomy，新增：
  - `scheduler.manual_cancel`
  - `scheduler.manual_retry`
- 扩展 runtime 配置域：
  - `scheduler.task_board.control.enabled=false`（默认关闭）
  - `scheduler.task_board.control.max_manual_retry_per_task=3`
- 扩展 diagnostics additive 字段，覆盖 manual control 的请求、成功、拒绝与幂等去重观测。
- 将 manual control contract suites 纳入 shared multi-agent gate 与 quality gate 阻断路径（含 Run/Stream 等价、memory/file parity、replay idempotency）。

## Capabilities

### New Capabilities

- `multi-agent-task-board-control-contract`: 定义 Task Board 控制动作（cancel/retry_terminal）、状态约束、幂等语义与失败分类。

### Modified Capabilities

- `distributed-subagent-scheduler`: 增加 scheduler 任务控制入口与状态迁移约束，补齐 manual cancel/retry reason taxonomy。
- `runtime-config-and-diagnostics-api`: 增加 `scheduler.task_board.control.*` 配置字段与 manual control 诊断汇总字段语义。
- `action-timeline-events`: 将 `scheduler.manual_cancel`、`scheduler.manual_retry` 纳入 scheduler canonical reason 语义集合。
- `go-quality-gate`: 增加 task-board-control 契约套件与 taxonomy drift 阻断映射。

## Impact

- 代码：
  - `orchestration/scheduler/*`（control API、状态机迁移、幂等处理、reason 事件）
  - `orchestration/composer/*`（可选：统一托管路径透传控制入口）
  - `runtime/config/*`（control 配置加载/校验/热更新回滚）
  - `runtime/diagnostics/*`（manual control additive 字段与查询/聚合）
  - `integration/*`（control contract tests：cancel/retry/idempotency/parity）
  - `scripts/check-multi-agent-shared-contract.*`
  - `scripts/check-quality-gate.*`
- 文档：
  - `README.md`
  - `docs/development-roadmap.md`
  - `docs/runtime-config-diagnostics.md`
  - `docs/mainline-contract-test-index.md`
- 兼容性：
  - 新能力默认关闭，保持保守行为；
  - 不引入平台化控制面，不改变既有 query 只读语义。
