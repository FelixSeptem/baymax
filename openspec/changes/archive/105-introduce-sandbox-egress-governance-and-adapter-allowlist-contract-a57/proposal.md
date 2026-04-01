## Why

A56 正在收敛 ReAct 工具闭环，但在安全治理侧仍存在两个主线缺口：
- sandbox 执行隔离已覆盖进程与资源边界，尚未统一网络外呼（egress）治理契约；
- adapter 接入有 manifest/conformance 基线，尚缺供应链 allowlist 激活约束与准入联动。

若不在同一提案内收口 egress + allowlist，将导致：
- 合规审计无法对“工具执行 + 网络出口 + adapter 激活”形成一体化可验证链路；
- readiness/admission 在安全维度出现执行前后语义断层；
- 质量门禁难以前置阻断 egress 漂移与未授权 adapter 激活。

## What Changes

- 新增 sandbox egress 治理与 adapter allowlist 一体化 contract（A57）。
- 在 `security.sandbox` 下新增 egress 策略配置域，冻结默认 deny-first 与显式 allowlist 语义。
- 在 adapter manifest/runtime activation 新增 allowlist 契约（publisher/id/version/signature/status）与 fail-fast 激活边界。
- 将 egress/allowlist finding 接入 readiness preflight + admission guard，保持 side-effect-free deny 语义。
- 扩展 diagnostics/replay/gate，新增 `sandbox_egress.v1` fixture 与 drift taxonomy。
- 扩展 conformance harness，覆盖 egress policy matrix + allowlist activation matrix。
- 增加 `check-sandbox-egress-allowlist-contract.sh/.ps1`，并暴露独立 required-check 候选。
- 同步 README/roadmap/mainline index/runtime-config 文档，保证状态与契约索引一致。

## Capabilities

### New Capabilities
- `sandbox-egress-governance-and-adapter-allowlist-contract`: 冻结 sandbox egress + adapter allowlist 的统一治理合同。

### Modified Capabilities
- `sandbox-execution-isolation`: 增加网络外呼决策与执行期 egress 违规 taxonomy。
- `sandbox-adapter-conformance-and-migration-pack`: 增加 egress/allowlist conformance matrix 与迁移映射。
- `adapter-manifest-and-runtime-compatibility`: 增加 allowlist 元数据与激活前 fail-fast 校验。
- `runtime-config-and-diagnostics-api`: 增加 `security.sandbox.egress.*` 与 `adapter.allowlist.*` 配置域及 additive 诊断字段。
- `runtime-readiness-preflight-contract`: 增加 egress/allowlist readiness finding 与 strict/non-strict 映射。
- `runtime-readiness-admission-guard-contract`: 增加 egress/allowlist admission deny 语义与 explainability 透传。
- `diagnostics-replay-tooling`: 增加 `sandbox_egress.v1` fixture 与 drift 分类断言。
- `go-quality-gate`: 增加 A57 contract gate 与 CI 独立 required-check 暴露。

## Impact

- 代码：
  - `runtime/config`、`runtime/config/readiness`（egress/allowlist 配置与 preflight）
  - `runtime/diagnostics`、`observability/event`（A57 additive 字段）
  - `runtime/security`、`core/runner`（egress 决策执行与 violation taxonomy）
  - `adapter/manifest`、`adapter/capability`、`integration/adapterconformance`（allowlist 激活边界与 conformance）
  - `tool/diagnosticsreplay`、`integration/*`（`sandbox_egress.v1` fixtures + drift tests）
  - `scripts/check-quality-gate.*` + `check-sandbox-egress-allowlist-contract.*`
- 文档：
  - `docs/development-roadmap.md`
  - `docs/mainline-contract-test-index.md`
  - `docs/runtime-config-diagnostics.md`
  - `docs/adapter-migration-mapping.md`
  - `README.md`
- 兼容性：
  - 外部 API 保持兼容；新增字段遵循 `additive + nullable + default`。
  - 默认策略为 deny-first；通过配置显式放开，不引入隐式 allow 行为。
