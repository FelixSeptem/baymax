# runtime/diagnostics 组件说明

## 功能域

`runtime/diagnostics` 提供统一诊断数据模型与查询能力：

- 调用记录：`CallRecord`
- 运行摘要：`RunRecord`
- 技能记录：`SkillRecord`
- 热更新记录：`ReloadRecord`

当前稳定查询接口：
- `RecentCalls`
- `RecentRuns`
- `RecentSkills`
- `TimelineTrends`
- `CA2ExternalTrends`

当前进度（2026-03-19）：
- A16 协作原语 additive 字段已归档稳定。
- A17 recovery boundary additive 字段已在模型中落位并持续收敛。
- A18 统一 run/team/workflow/task 查询契约正在实施中，完成前以 `Recent* + Trends` 作为稳定读接口。

## 架构设计

`Store` 使用有界内存结构维护多类记录，并支持：

- 容量动态调整（Resize）
- 幂等去重（run/skill idempotency key）
- 统一 run 查询（`QueryRuns`：多维过滤 + 分页 + 排序 + opaque cursor）
- timeline 聚合（phase 统计、P95 延迟）
- 趋势查询（`TimelineTrends`、`CA2ExternalTrends`）

该包只负责数据模型和聚合算法，不负责事件订阅。

## 关键入口

- `store.go`

## 边界与依赖

- 诊断写入统一由 `observability/event.RuntimeRecorder` 驱动。
- 业务模块不应直接构造跨域写路径绕开 RuntimeRecorder。
- 新增诊断字段时，需要同步代码、文档与契约测试索引。
