## Why

A42 已建立 diagnostics query 性能回归门禁，但主线仍缺少对诊断字段规模与高基数扩张的治理契约。随着 A41/A43/A44 持续新增 additive 字段，若没有统一 budget 与截断策略，查询性能和回放稳定性会在后续迭代中持续漂移。

## What Changes

- 新增 diagnostics cardinality budget 契约，定义 map/list/string 等可膨胀字段的统一预算约束。
- 新增 overflow 策略与默认行为：`truncate_and_record`（默认）与 `fail_fast`。
- 固化 deterministic 截断语义：同一输入与配置下，截断结果与标记字段稳定一致。
- 扩展 diagnostics additive 字段，记录预算命中、截断计数、截断字段摘要与策略信息。
- 新增 contract suites 与 quality gate 阻断映射，覆盖 Run/Stream 等价与 replay idempotency。
- 同步更新 runtime/config 文档、主线契约索引与 roadmap 状态口径。

## Capabilities

### New Capabilities
- `diagnostics-cardinality-budget-and-truncation-governance`: 定义 diagnostics 字段预算、截断策略与稳定序列化治理契约。

### Modified Capabilities
- `runtime-config-and-diagnostics-api`: 新增 `diagnostics.cardinality.*` 配置域与 cardinality additive 诊断字段语义。
- `go-quality-gate`: 新增 cardinality drift contract suites 的阻断映射与 shell/PowerShell parity 要求。

## Impact

- 代码：
  - `runtime/diagnostics/*`（预算检查、截断逻辑、字段摘要、稳定输出）
  - `runtime/config/*`（`diagnostics.cardinality.*` 解析/校验/热更新回滚）
  - `integration/*`（cardinality contract suites：budget、truncation、replay、Run/Stream parity）
  - `scripts/check-quality-gate.*`
- 文档：
  - `README.md`
  - `docs/runtime-config-diagnostics.md`
  - `docs/mainline-contract-test-index.md`
  - `docs/development-roadmap.md`
- 兼容性：
  - 默认策略 `truncate_and_record` 保持可运行并增强可观测；
  - 新字段保持 `additive + nullable + default` 兼容窗口；
  - 不引入平台化控制面或外部存储依赖。
