# tool/local 组件说明

## 功能域

`tool/local` 提供本地工具运行时能力：

- 工具注册与命名规范（`local.*`）
- 参数 schema 校验
- 并发分发、重试、背压治理

## 架构设计

核心由 `Registry + Dispatcher` 组成：

- `Registry` 负责工具注册、命名标准化、去重
- `Dispatcher` 负责调度执行，区分读写调用路径
- 调度策略由 `DispatchConfig` 控制：
  - 并发度与队列长度
  - 背压模式（`block|reject|drop_low_priority`）
  - fail-fast 与 retry 次数

运行时默认值可由 `runtime/config.Manager` 注入覆盖。

## 关键入口

- `registry.go`
- `schema.go`

## 边界与依赖

- 只处理本地工具生命周期，不承载 MCP 远端传输语义。
- 诊断通过 `runtime/config.Manager.RecordCall` 记录，不自行维护独立存储。
- 调度阶段会补充 `dispatch_phase`、`queue_full` 等细粒度错误详情，供上层 timeline 聚合。
