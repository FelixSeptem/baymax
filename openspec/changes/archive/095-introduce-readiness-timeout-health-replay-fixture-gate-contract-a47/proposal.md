## Why

A40/A41/A43/A44/A46 已分别收敛 readiness、timeout resolution、adapter health 与 admission 语义，但当前主线仍缺少跨能力组合场景的统一 replay fixture 契约。没有固定夹具与阻断门禁时，跨提案演进容易出现 taxonomy 漂移、策略映射回归和回放结果不一致。

## What Changes

- 新增 `readiness + timeout + adapter health` 的交叉 replay fixture 契约，固化输入、输出与断言口径。
- 定义 fixture 分层矩阵（strict/non-strict、profile/domain/request override、required/optional adapter、circuit state）并固定最小覆盖集。
- 扩展 replay tooling 对组合场景的规范化输出校验，保证 deterministic ordering 与 replay idempotency。
- 将 A47 交叉语义回放套件接入 quality gate 阻断路径（shell/PowerShell parity）。
- 同步更新主线契约索引、runtime 诊断文档与 roadmap 状态口径。

## Capabilities

### New Capabilities
- `readiness-timeout-health-replay-fixture-gate`: 定义跨 readiness/timeout/adapter-health 的 replay fixture、断言模型与阻断门禁契约。

### Modified Capabilities
- `diagnostics-replay-tooling`: 扩展组合语义回放夹具的输入/输出稳定性与错误分类约束。
- `runtime-readiness-preflight-contract`: 增加 readiness finding 与 replay fixture 对账一致性要求。
- `runtime-operation-profiles-and-timeout-resolution-contract`: 增加 timeout resolution trace 与 replay fixture 一致性要求。
- `adapter-runtime-health-probe-contract`: 增加 adapter health governance 输出与 replay fixture 一致性要求。
- `go-quality-gate`: 增加 A47 replay fixture suites 阻断映射与 shell/PowerShell parity 要求。

## Impact

- 代码：
  - `tool/diagnosticsreplay/*`（fixture 装载、组合语义断言、错误分层）
  - `integration/*`（readiness-timeout-health replay suites）
  - `scripts/check-quality-gate.*`（A47 阻断步骤）
- 文档：
  - `docs/mainline-contract-test-index.md`
  - `docs/runtime-config-diagnostics.md`
  - `docs/development-roadmap.md`
  - `README.md`
- 兼容性：
  - 不引入平台化控制面与外部依赖；
  - 仅新增 additive/fixture 约束，不破坏既有 Run/Stream 业务语义；
  - gate 检测到语义漂移时 fail-fast 阻断。
