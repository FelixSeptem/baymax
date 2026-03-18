# runtime/diagnostics 组件说明

## 功能域

`runtime/diagnostics` 提供统一诊断数据模型与查询能力：

- 调用记录：`CallRecord`
- 运行摘要：`RunRecord`
- 技能记录：`SkillRecord`
- 热更新记录：`ReloadRecord`

## 架构设计

`Store` 使用有界内存结构维护多类记录，并支持：

- 容量动态调整（Resize）
- 幂等去重（run/skill idempotency key）
- timeline 聚合（phase 统计、P95 延迟）
- 趋势查询（`TimelineTrends`、`CA2ExternalTrends`）

该包只负责数据模型和聚合算法，不负责事件订阅。

## 关键入口

- `store.go`

## 边界与依赖

- 诊断写入统一由 `observability/event.RuntimeRecorder` 驱动。
- 业务模块不应直接构造跨域写路径绕开 RuntimeRecorder。
- 新增诊断字段时，需要同步代码、文档与契约测试索引。
