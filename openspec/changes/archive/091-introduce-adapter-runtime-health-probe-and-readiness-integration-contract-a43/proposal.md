## Why

当前 adapter 域已具备 manifest/capability/conformance 的静态契约，但缺少运行期健康探测与 readiness 集成语义。外部 adapter 在运行中发生不可用或退化时，调用方难以通过统一库级接口得到可观测、可阻断、可降级的一致决策。

## What Changes

- 新增 adapter runtime health probe 契约，统一探测模型与状态分级（`healthy|degraded|unavailable`）。
- 新增 adapter health 配置域并纳入 `env > file > default`、启动 fail-fast、热更新原子回滚。
- 将 adapter health 结果纳入 runtime readiness（A40）判定：
  - required adapter 异常在 strict 模式下阻断；
  - optional adapter 异常允许降级并保留可观测 findings。
- 扩展 diagnostics additive 字段，记录 adapter health 状态、探测计数、主原因码与降级次数。
- 在 adapter conformance harness 与 quality gate 中增加 health matrix 阻断套件（memory/offline deterministic、Run/Stream 等价、replay idempotency）。

## Capabilities

### New Capabilities
- `adapter-runtime-health-probe-contract`: 定义 adapter 运行期健康探测、状态分级、错误分类与降级判定语义。

### Modified Capabilities
- `runtime-readiness-preflight-contract`: 增加 adapter 健康结果对 `ready|degraded|blocked` 的映射规则。
- `runtime-config-and-diagnostics-api`: 增加 `adapter.health.*` 配置域与 adapter health additive 诊断字段。
- `external-adapter-conformance-harness`: 增加 adapter health 探测矩阵与 required/optional 降级断言。
- `go-quality-gate`: 增加 adapter health contract suites 的阻断映射与跨平台 parity 约束。

## Impact

- 代码：
  - `adapter/*`（health probe 接口、结果模型、默认探测实现）
  - `runtime/config/*`（`adapter.health.*` 配置解析/校验/热更新回滚）
  - `runtime/diagnostics/*`（adapter health additive 字段与聚合）
  - `runtime/readiness` / `runtime/config.Manager`（preflight 集成）
  - `integration/adapterconformance/*`（health contract suites）
  - `scripts/check-adapter-conformance.*`
  - `scripts/check-quality-gate.*`
- 文档：
  - `README.md`
  - `docs/runtime-config-diagnostics.md`
  - `docs/development-roadmap.md`
  - `docs/mainline-contract-test-index.md`
- 兼容性：
  - 默认 `adapter.health.enabled=false` 保持保守行为；
  - 新字段遵循 `additive + nullable + default`；
  - 不引入平台化控制面或外部托管依赖。
