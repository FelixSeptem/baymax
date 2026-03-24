# runtime/config 组件说明

## 功能域

`runtime/config` 提供统一运行时配置能力：

- 默认配置建模（`DefaultConfig`）
- 配置加载与校验（`env > file > default`）
- 热更新监听、原子切换与失败回滚
- 对外提供策略解析与脱敏辅助接口

## 架构设计

核心对象是 `Manager`：

- 启动阶段：解析并校验配置，构建首个不可变快照
- 运行阶段：通过 `atomic.Value` 暴露 `EffectiveConfig` / `CurrentSnapshot`
- 热更新阶段：`parse -> validate -> swap`，失败仅记录 reload 诊断，不污染现网快照

该包同时维护配置与诊断的连接点：

- 为 MCP/并发/编排模块提供策略解析
- 管理 `runtime/diagnostics.Store` 容量与趋势配置
- 暴露统一脱敏输出（`EffectiveConfigSanitized`）

关键默认值（多代理相关）：
- `composer.collab.enabled=false`
- `composer.collab.retry.enabled=false`（当前保持 primitive retry 默认关闭）
- `workflow.graph_composability.enabled=false`
- `a2a.async_reporting.enabled=false`
- `mailbox.backend=memory`
- `mailbox.retry.max_attempts=3`
- `mailbox.ttl=15m`
- `scheduler.qos.mode=fifo`
- `scheduler.dlq.enabled=false`
- `scheduler.async_await.report_timeout=15m`
- `scheduler.async_await.late_report_policy=drop_and_record`
- `scheduler.async_await.timeout_terminal=failed`
- `scheduler.async_await.reconcile.enabled=false`
- `scheduler.async_await.reconcile.interval=5s`
- `scheduler.async_await.reconcile.batch_size=64`
- `scheduler.async_await.reconcile.jitter_ratio=0.2`
- `scheduler.async_await.reconcile.not_found_policy=keep_until_timeout`
- `recovery.enabled=false`
- `recovery.conflict_policy=fail_fast`

## 关键入口

- `config.go`
- `manager.go`

## 边界与依赖

- `runtime/config` 不依赖 `mcp/http` 或 `mcp/stdio` 传输实现。
- 非法配置和非法热更新必须 fail-fast，并保持旧快照可回滚。
- 配置字段变更需要同步更新 `docs/runtime-config-diagnostics.md` 与契约测试。
- A41 新增 operation profile 配置域：`runtime.operation_profiles.default_profile` 与四个 profile timeout（`legacy|interactive|background|batch`）。
- A41 timeout 解析器固定三层优先级：`profile -> domain -> request`，并输出来源标签与 trace（`v1`）。

## 配置与默认值

- 默认值入口：`DefaultConfig`。
- 优先级固定：`env > file > default`，并在 `EffectiveConfig` 中体现最终快照。
- 热更新默认允许监听，但非法更新会阻断并回滚到上一个有效快照。
- operation profile 默认值：
  - `default_profile=legacy`
  - `legacy.timeout=3s`
  - `interactive.timeout=10s`
  - `background.timeout=30s`
  - `batch.timeout=2m`

## 可观测性与验证

- 关键验证：`go test ./runtime/config -count=1`。
- 关键观测字段：reload 成功/失败计数、回滚标记、配置来源优先级。
- 与 diagnostics 的联动验证需覆盖 QueryRuns 与趋势查询兼容语义。

## 扩展点与常见误用

- 扩展点：新增配置子域时同步 parser、validator、docs、契约测试。
- 常见误用：只更新默认值不更新文档与测试，导致 gate 漂移。
- 常见误用：热更新失败后继续使用脏快照，破坏 fail-fast + rollback 语义。
