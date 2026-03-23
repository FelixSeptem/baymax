## Why

当前主线能力已经覆盖运行、编排、诊断与回放，但缺少统一的“运行前准备度”契约。调用方难以在启动前或热更新后快速判断系统是否处于 `ready/degraded/blocked`，只能在运行后通过分散日志排障，影响可用性收敛效率。

## What Changes

- 新增库级 runtime readiness preflight 能力：
  - 提供统一 `Readiness` API，输出 `ready|degraded|blocked` 状态与结构化 findings。
  - findings 统一字段：`code/domain/severity/message/metadata`。
- 预检覆盖关键域（最小闭环）：
  - 配置加载与校验状态；
  - scheduler/mailbox/recovery backend 初始化与 fallback 状态；
  - 关键运行依赖可用性（在不依赖平台控制面的前提下）。
- 新增 readiness 配置域：
  - `runtime.readiness.enabled=true`
  - `runtime.readiness.strict=false`
  - `runtime.readiness.remote_probe_enabled=false`
- 新增 strict 策略：
  - `strict=false` 时 `degraded` 可运行但必须可观测；
  - `strict=true` 时 `degraded` 升级为阻断（`blocked`）。
- 扩展 diagnostics additive 字段，记录 readiness 判定与主要退化原因。
- 将 readiness contract suites 纳入 quality gate 阻断路径（含 replay idempotency 与文档映射）。

## Capabilities

### New Capabilities

- `runtime-readiness-preflight-contract`: 定义 runtime readiness 预检 API、状态分级（ready/degraded/blocked）、strict 策略与结果模型。

### Modified Capabilities

- `runtime-config-and-diagnostics-api`: 增加 `runtime.readiness.*` 配置字段与 readiness additive 诊断字段语义。
- `multi-agent-lib-first-composer`: 增加 composer 对 readiness 结果的库级透传/聚合语义。
- `go-quality-gate`: 增加 readiness contract suites 阻断映射与 drift 校验要求。

## Impact

- 代码：
  - `runtime/config/*`（readiness 配置解析/校验/热更新回滚）
  - `runtime/diagnostics/*`（readiness additive 字段与查询）
  - `orchestration/composer/*`（readiness 透传入口与摘要接线）
  - `integration/*`（readiness contract tests：strict matrix、fallback visibility、replay idempotency）
  - `scripts/check-quality-gate.*`（readiness 套件接入）
- 文档：
  - `README.md`
  - `docs/runtime-config-diagnostics.md`
  - `docs/development-roadmap.md`
  - `docs/mainline-contract-test-index.md`
- 兼容性：
  - 新字段均为 additive；
  - 默认 `strict=false`，保持既有运行路径兼容。
