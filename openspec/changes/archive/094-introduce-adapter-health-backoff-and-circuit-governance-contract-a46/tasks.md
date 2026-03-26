## 1. Runtime Config Governance Fields

- [x] 1.1 在 `runtime/config` 增加 `adapter.health.backoff.*` 与 `adapter.health.circuit.*` 配置结构、默认值与 env 映射。
- [x] 1.2 补齐 startup 校验（duration/range/enum/bool）与非法配置 fail-fast 行为。
- [x] 1.3 补齐 hot reload 非法更新回滚测试，确保保留上一份有效快照。

## 2. Adapter Health Backoff and Circuit Core

- [x] 2.1 在 `adapter/health` 实现指数退避 + 抖动调度策略（initial/max/multiplier/jitter）。
- [x] 2.2 实现 `closed|open|half_open` 状态机与 canonical 转移规则。
- [x] 2.3 实现 half-open 探测预算与恢复判定（`half_open_max_probe`、`half_open_success_threshold`）。

## 3. Readiness Integration and Canonical Taxonomy

- [x] 3.1 在 `runtime/config/readiness` 接入 circuit/backoff 结果映射，输出 canonical `adapter.health.*` findings。
- [x] 3.2 补齐 strict/non-strict 下 required/optional adapter 的分类测试。
- [x] 3.3 确保 Run/Stream admission 依赖的 readiness 语义保持一致且无行为回归。

## 4. Diagnostics, Conformance, and Gates

- [x] 4.1 在 `runtime/diagnostics` 增加 adapter-health governance additive 字段与聚合逻辑。
- [x] 4.2 补齐 replay idempotency 测试，确保重复事件不膨胀治理计数。
- [x] 4.3 在 `integration/adapterconformance` 增加 backoff/circuit matrix suites（状态转移、半开恢复、taxonomy drift guard）。
- [x] 4.4 更新 `scripts/check-adapter-conformance.*` 与 `scripts/check-quality-gate.*`，将治理 suites 纳入阻断步骤并保持 shell/PowerShell parity。

## 5. Documentation and Acceptance

- [x] 5.1 更新 `docs/runtime-config-diagnostics.md`，补充 `adapter.health.backoff.*`、`adapter.health.circuit.*` 字段与诊断映射。
- [x] 5.2 更新 `docs/mainline-contract-test-index.md`、`docs/development-roadmap.md` 与 `README.md`，同步 A46 状态与门禁映射。
- [x] 5.3 执行并记录最小验收命令：`go test ./...`、`go test -race ./...`、`pwsh -File scripts/check-docs-consistency.ps1`、`pwsh -File scripts/check-quality-gate.ps1`。
