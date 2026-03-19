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

## 配置与默认值

- 关键配置来自 `runtime/config` 中本地 dispatch 策略（并发、队列、背压、重试）。
- 默认背压模式为 `block`；可切换 `reject` 或 `drop_low_priority`。
- 工具命名默认归一到 `local.*` 命名空间。

## 可观测性与验证

- 关键验证：`go test ./tool/local -count=1`。
- 与 runner 集成语义可通过 `go test ./core/runner -count=1` 联动覆盖。
- 调用事件与错误详情会进入统一 diagnostics 聚合链路。

## 扩展点与常见误用

- 扩展点：自定义 schema 校验规则、定制 dispatch 配置、扩展工具注册治理。
- 常见误用：工具实现忽略必填参数 fail-fast，导致上层重试与错误归类失真。
- 常见误用：直接在业务层拼接工具名，绕过 registry 标准化。
