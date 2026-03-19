# runtime/config 组件说明

## 功能域

`runtime/config` 提供统一运行时配置能力：

- 默认配置建模（`DefaultConfig`）
- 配置加载与校验（`env > file > default`）
- 热更新监听、原子切换与失败回滚
- 对外提供策略解析与脱敏辅助接口

当前进度（2026-03-19）：
- A16 已归档：`composer.collab.*` 配置已稳定。
- A17 进行中：`recovery.resume_boundary/inflight_policy/timeout_reentry_*` 已进入配置契约。
- A18 进行中：统一诊断检索 API 不新增 feature flag，沿用 diagnostics 标准能力入口。

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
- `workflow.graph_composability.enabled=false`
- `a2a.async_reporting.enabled=false`
- `scheduler.qos.mode=fifo`
- `scheduler.dlq.enabled=false`
- `recovery.enabled=false`
- `recovery.conflict_policy=fail_fast`

## 关键入口

- `config.go`
- `manager.go`

## 边界与依赖

- `runtime/config` 不依赖 `mcp/http` 或 `mcp/stdio` 传输实现。
- 非法配置和非法热更新必须 fail-fast，并保持旧快照可回滚。
- 配置字段变更需要同步更新 `docs/runtime-config-diagnostics.md` 与契约测试。
