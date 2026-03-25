## Why

A40/A43 已建立 readiness 预检与 adapter 健康可观测语义，但当前主链路仍以“查询与透传”为主，缺少统一执行前准入护栏。调用方无法在 Run/Stream 入口以一致策略处理 `blocked` 与 `degraded`，导致运行前决策分散、回归难以阻断。

## What Changes

- 新增 runtime readiness admission guard 契约：在执行入口统一进行 readiness 准入判定（Run/Stream 语义一致）。
- 新增 `runtime.readiness.admission.*` 配置域，保持 `env > file > default` 与启动/热更新 fail-fast + 原子回滚。
- 固化 admission 默认行为：
  - `blocked` 默认 fail-fast 拒绝执行；
  - `degraded` 按策略 `allow_and_record` 或 `fail_fast` 处理。
- 扩展 diagnostics additive 字段，记录 admission 总量、阻断量、降级放行量、策略模式与主阻断原因。
- 将 admission contract suites 接入 shared gate / quality gate 阻断路径（read-only 准入检查、Run/Stream 等价、replay idempotency）。

## Capabilities

### New Capabilities
- `runtime-readiness-admission-guard-contract`: 定义执行前 readiness 准入检查、阻断/降级放行策略与无副作用保证。

### Modified Capabilities
- `runtime-readiness-preflight-contract`: 增加 preflight 结果到 admission 决策的稳定映射约束。
- `runtime-config-and-diagnostics-api`: 增加 `runtime.readiness.admission.*` 配置域与 admission additive 诊断字段契约。
- `multi-agent-lib-first-composer`: 增加 composer managed Run/Stream 入口的 readiness admission 执行语义与无副作用约束。
- `go-quality-gate`: 增加 readiness admission contract suites 与阻断映射。

## Impact

- 代码：
  - `runtime/config/*`（admission 配置解析/校验/热更新回滚）
  - `runtime/config/readiness*.go`（admission 决策器与 preflight 结果映射）
  - `orchestration/composer/*`（Run/Stream 入口 admission guard 接线）
  - `runtime/diagnostics/*`（admission additive 字段与查询输出）
  - `integration/*`（admission 契约测试矩阵）
  - `scripts/check-quality-gate.*`
- 文档：
  - `README.md`
  - `docs/runtime-config-diagnostics.md`
  - `docs/development-roadmap.md`
  - `docs/mainline-contract-test-index.md`
- 兼容性：
  - 默认 `runtime.readiness.admission.enabled=false`，保持现有行为；
  - 新字段遵循 `additive + nullable + default`；
  - 不引入平台化控制面与外部依赖。
