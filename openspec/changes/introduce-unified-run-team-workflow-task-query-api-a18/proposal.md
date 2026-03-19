## Why

当前 runtime diagnostics 主要提供 `RecentRuns/RecentCalls/RecentSkills` 与趋势查询能力，但缺少统一的多维检索契约，调用方难以按 `run/team/workflow/task` 一致方式查询历史执行状态。A18 目标是在 `library-first` 前提下补齐统一查询 API，降低接入成本并稳定多代理观测消费语义。

## What Changes

- 新增统一检索能力：按 `run_id`、`team_id`、`workflow_id`、`task_id`、`status`、`time_range` 组合过滤。
- 新增分页与排序契约：默认分页 `50`、最大分页 `200`、默认 `time desc`。
- 新增游标分页契约：使用 opaque cursor，不暴露内部存储结构。
- 固化过滤组合语义：多条件按 `AND` 组合。
- 固化 `task_id` 缺失语义：当目标记录不存在时返回空集，不返回错误。
- 固化参数校验语义：非法过滤参数 fail-fast。
- 保持兼容：保留 `Recent*` API，不破坏现有调用；新增字段遵循 `additive + nullable + default`。
- 将查询契约测试纳入 shared quality gate 与主干索引映射。

## Capabilities

### New Capabilities
- `multi-agent-unified-query-api`: 定义统一多维检索、分页、排序与游标语义契约。

### Modified Capabilities
- `runtime-config-and-diagnostics-api`: 增加统一查询 API 与参数校验、兼容窗口约束。
- `go-quality-gate`: 增加统一查询 API 合同测试与 gate 阻断规则。

## Impact

- 代码：
  - `runtime/diagnostics/*`
  - `runtime/config/manager.go`
  - `tool/contributioncheck/*`
  - `scripts/check-multi-agent-shared-contract.*`
- 测试：
  - `runtime/diagnostics/*_test.go` 查询矩阵
  - `integration/*` 多维检索 + replay-idempotent 场景
- 文档：
  - `README.md`
  - `docs/runtime-config-diagnostics.md`
  - `docs/mainline-contract-test-index.md`
  - `docs/development-roadmap.md`
- 兼容性：
  - 不引入平台化查询服务；
  - 默认行为兼容现有 `Recent*` API；
  - 查询新增字段保持 `additive + nullable + default`。
