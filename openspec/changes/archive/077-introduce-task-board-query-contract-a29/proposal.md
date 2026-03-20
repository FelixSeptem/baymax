## Why

A18 已补齐 run 维度统一查询，但当前仍缺少面向 scheduler 任务生命周期的只读看板契约。调用方无法稳定查询 `queued/running/dead_letter` 等任务态与 attempt 视图，导致排障和协作编排观测需要依赖临时拼接逻辑。

在保持 `library-first` 边界前提下，需要补齐 Task Board 查询契约，让主机应用可通过库接口稳定获取任务视图，并与既有 run 诊断查询形成可组合观测路径。

## What Changes

- 新增 Task Board 只读查询契约：支持 `task_id/run_id/workflow_id/team_id/state/priority/agent_id/peer_id/parent_run_id/time_range` 过滤。
- 固化组合过滤语义：多条件按 `AND` 执行。
- 固化分页/排序语义：默认 `page_size=50`、上限 `200`、默认排序 `updated_at desc`。
- 固化游标语义：使用 opaque cursor，游标与查询边界绑定，不暴露内部 offset/index 细节。
- 固化错误与空集边界：非法参数 fail-fast；合法但无匹配返回空集。
- 通过 scheduler 快照路径提供后端一致行为（memory/file 语义一致）。
- 将 Task Board 合同测试接入 shared multi-agent gate，并更新主干契约索引与文档。

## Capabilities

### New Capabilities
- `multi-agent-task-board-query-contract`: 定义 scheduler 任务看板只读查询、分页排序、游标和错误边界契约。

### Modified Capabilities
- `distributed-subagent-scheduler`: 增加 scheduler 任务看板查询入口与后端一致性约束。
- `go-quality-gate`: 增加 task board query contract suites 作为阻断项。

## Impact

- 代码：
  - `orchestration/scheduler/*`（查询模型、查询执行、游标与校验）
  - `integration/*`（memory/file 一致性与恢复后查询语义）
- 测试：
  - `orchestration/scheduler/*_test.go`（过滤、分页、游标、fail-fast）
  - `integration/*`（跨后端一致性、replay/restore 后语义稳定）
- 文档：
  - `README.md`
  - `docs/runtime-config-diagnostics.md`
  - `docs/mainline-contract-test-index.md`
  - `docs/development-roadmap.md`
- 兼容性：
  - 不移除既有 `QueryRuns` 与 `Recent*` 能力；
  - 不引入平台化控制台、任务写操作（cancel/retry/reassign）。
