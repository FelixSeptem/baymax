## Why

A43 已建立 adapter runtime health probe 与 readiness 集成基线，但在 adapter 持续失败或网络抖动场景下，当前主线缺少统一的退避与熔断治理契约，容易产生探测风暴、readiness 抖动和噪声告警放大。A45 正在收敛 diagnostics cardinality，当前窗口适合同时补齐 health probing 的节流与状态收敛语义，避免后续重复改造。

## What Changes

- 新增 adapter health backoff + circuit 治理契约，统一定义 `closed|open|half_open` 状态机与状态转移条件。
- 为 adapter health probe 增加指数退避与抖动策略，固化默认值与 fail-fast 校验规则。
- 将 circuit/backoff 结果统一映射到 readiness findings，保证 `strict/non-strict` 路径下语义稳定。
- 扩展 diagnostics additive 字段，记录 backoff/circuit 计数、状态与主原因码，并保持 replay idempotency。
- 扩展 adapter conformance 与 quality gate 阻断套件，覆盖抖动抑制、半开探测、Run/Stream 等价与 memory/file parity。
- 同步更新 runtime/config 文档、主线契约索引与 roadmap 状态口径。

## Capabilities

### New Capabilities
- `adapter-health-backoff-and-circuit-governance`: 定义 adapter health probe 退避、熔断状态机、诊断字段与阻断门禁的统一契约。

### Modified Capabilities
- `adapter-runtime-health-probe-contract`: 在既有 health probe 三态语义上增加 backoff/circuit 运行治理约束。
- `runtime-config-and-diagnostics-api`: 新增 `adapter.health.backoff.*`、`adapter.health.circuit.*` 配置域与对应 additive 诊断字段。
- `runtime-readiness-preflight-contract`: 将 circuit/backoff 输出接入 readiness classification 与 canonical findings taxonomy。
- `external-adapter-conformance-harness`: 新增 adapter health backoff/circuit matrix 与一致性验收套件。
- `go-quality-gate`: 新增 adapter health backoff/circuit contract suites 的阻断映射与 shell/PowerShell parity 要求。

## Impact

- 代码：
  - `adapter/health/*`（退避、熔断状态机、状态转换与探测节流）
  - `runtime/config/*`（`adapter.health.backoff.*`、`adapter.health.circuit.*` 解析/校验/热更新回滚）
  - `runtime/config/readiness*`（readiness finding 映射与 strict/non-strict 收敛）
  - `runtime/diagnostics/*`（additive 字段与 replay idempotency 聚合）
  - `integration/adapterconformance/*` 与 `integration/*`（contract suites）
  - `scripts/check-adapter-conformance.*`、`scripts/check-quality-gate.*`
- 文档：
  - `README.md`
  - `docs/runtime-config-diagnostics.md`
  - `docs/mainline-contract-test-index.md`
  - `docs/development-roadmap.md`
- 兼容性：
  - 默认保持 lib-first 行为，不引入平台化控制面或外部协调存储；
  - 新增字段遵循 `additive + nullable + default`；
  - 非法配置与非法热更新保持 fail-fast + 原子回滚。
