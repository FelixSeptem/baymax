## Why

A53（sandbox adapter）与 A54（memory SPI）正在推进，运行时新增语义将显著增加跨环境排障与外部观测集成成本。当前缺少统一的观测出口与诊断取证包 contract，导致接入 OTLP/Langfuse 等系统和事故复盘依赖分散脚本，容易出现字段漂移与不可回放问题。

## What Changes

- 新增可观测出口 contract：定义 `observability.export.profile`（`none|otlp|langfuse|custom`）与统一 exporter SPI，保持 `RuntimeRecorder` 单写入口不变。
- 新增诊断取证包（diagnostics bundle）contract：冻结 bundle 结构（timeline、diagnostics window、redacted effective config、replay hints、gate fingerprint、schema version）。
- 新增 `runtime.observability.export.*` 与 `runtime.diagnostics.bundle.*` 配置域，纳入 `env > file > default`、启动 fail-fast、热更新原子回滚。
- 扩展 runtime diagnostics additive 字段，覆盖 export/bundle 执行状态、失败分类、队列/丢弃计数与 bundle 生成结果，保持 bounded-cardinality 与 replay idempotency。
- 扩展 readiness preflight：新增 `observability.export.*` 与 `diagnostics.bundle.*` findings，并保持 strict/non-strict 映射语义。
- 扩展 diagnostics replay：新增 `observability.v1` fixture，校验 export/bundle 语义稳定性与 drift 分类。
- 扩展 quality gate：新增 `check-observability-export-and-bundle-contract.sh/.ps1` 并作为独立 required-check 候选。
- 同步模板/文档索引与 roadmap，明确 A55 与 A53/A54 的依赖边界与迁移路径。

## Capabilities

### New Capabilities
- `observability-export-and-diagnostics-bundle-contract`: 定义运行时观测出口 profile、diagnostics bundle schema、导出与取证的一致性 contract。

### Modified Capabilities
- `runtime-config-and-diagnostics-api`: 增加 observability export/bundle 配置域、校验规则与 additive 诊断字段。
- `runtime-readiness-preflight-contract`: 增加 export sink / bundle storage 可用性 finding 与 strict/non-strict 映射。
- `diagnostics-replay-tooling`: 增加 `observability.v1` fixture 及 drift 分类断言。
- `go-quality-gate`: 增加 observability export + diagnostics bundle contract gate 与独立 required-check 暴露。

## Impact

- 代码：
  - `runtime/config`（`runtime.observability.export.*`、`runtime.diagnostics.bundle.*`）
  - `runtime/config/readiness`（observability/bundle findings）
  - `observability/event`、`runtime/diagnostics`（additive 字段与单写映射）
  - `tool/diagnosticsreplay`、`integration/*`（`observability.v1` fixture 与回放契约）
  - `scripts/check-quality-gate.*` + `check-observability-export-and-bundle-contract.*`
- 文档：
  - `docs/runtime-config-diagnostics.md`
  - `docs/mainline-contract-test-index.md`
  - `docs/development-roadmap.md`
  - `README.md`
- 兼容性：
  - 不改变 Run/Stream 对外行为；新增字段遵循 `additive + nullable + default`。
  - 导出器不可用时按配置策略 fail-fast 或 degrade，并保证可审计 reason code。
