## Why

A52 正在实施 sandbox 运行治理（rollout/health/capacity），但生态侧仍缺少面向主流 sandbox 后端的标准化接入包。若没有统一的 adapter manifest、conformance matrix 与迁移映射，业务团队会继续以定制 glue code 接入，导致后端切换成本高、语义漂移难以被 gate 阻断。

## What Changes

- 新增主流 sandbox adapter 接入包 contract，统一 `linux_nsjail`、`linux_bwrap`、`oci_runtime`、`windows_job` 的 profile 语义。
- 扩展 adapter manifest compatibility 契约，增加 sandbox backend/profile/session 维度声明与 fail-fast 校验。
- 扩展 external adapter conformance harness，新增 sandbox backend matrix、能力协商、会话生命周期、drift 分类断言。
- 扩展 adapter template/migration mapping 文档契约，提供 sandbox adapter onboarding skeleton 与旧接入模式迁移映射。
- 扩展 adapter contract profile replay 契约，增加 sandbox profile fixture（建议 `sandbox.v1`）与兼容窗口语义。
- 扩展 readiness preflight 契约，新增 sandbox adapter profile 可用性/兼容性 finding 与 strict/non-strict 映射。
- 扩展 go quality gate 契约，新增 sandbox adapter conformance gate（shell/PowerShell parity + 独立 required-check 候选）。
- 本提案不重定义 A51/A52 sandbox 执行与运行治理语义，仅收口“外部接入 DX + conformance + migration”。

## Capabilities

### New Capabilities
- `sandbox-adapter-conformance-and-migration-pack`: 定义主流 sandbox 后端的 adapter profile pack、迁移映射与一致性收敛契约。

### Modified Capabilities
- `adapter-manifest-and-runtime-compatibility`: 增加 sandbox adapter metadata 与运行时兼容校验语义。
- `external-adapter-conformance-harness`: 增加 sandbox backend matrix、能力/会话 conformance 套件与 drift 分类。
- `external-adapter-template-and-migration-mapping`: 增加 sandbox adapter 模板与迁移映射契约。
- `adapter-contract-profile-versioning-and-replay`: 增加 sandbox adapter profile fixture 与 profile 兼容窗口语义。
- `runtime-readiness-preflight-contract`: 增加 sandbox adapter profile availability/compatibility 预检 finding。
- `go-quality-gate`: 增加 sandbox adapter conformance gate 及独立 required-check 暴露。

## Impact

- 代码：
  - `adapter/manifest`（sandbox adapter manifest 字段与校验）
  - `tool/*conformance*` / `integration/*`（backend matrix、session lifecycle、capability negotiation 套件）
  - `runtime/config/readiness*`（profile 可用性/兼容性预检）
  - `scripts/check-quality-gate.*` 与新增 sandbox adapter gate 脚本
- 文档：
  - `docs/external-adapter-template-index.md`
  - `docs/adapter-migration-mapping.md`
  - `docs/mainline-contract-test-index.md`
  - `docs/development-roadmap.md`
  - `README.md`
- 兼容性：
  - 仅新增或收敛 adapter 侧 contract，不移除既有接口。
  - 采用 `additive + nullable + default + fail-fast` 治理边界。
