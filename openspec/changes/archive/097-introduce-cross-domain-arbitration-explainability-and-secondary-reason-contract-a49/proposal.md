## Why

A48 正在收敛 cross-domain primary reason 裁决规则，但当前主线仍缺少“secondary reasons + explainability”契约，导致排障与回放对账只能看到 primary 结果，无法稳定解释“为何其他候选未被选中”。在 A47/A48 并行收敛阶段补齐可解释性语义，能避免后续持续追加非标准字段与 drift 规则。

## What Changes

- 新增 cross-domain arbitration explainability 契约，定义 secondary reasons 的有界输出、稳定排序与去重规则。
- 固化 arbitration 解释字段：`rule_version`、`secondary_reason_codes`、`secondary_reason_count`、`remediation_hint_*`。
- 在 readiness preflight 与 admission guard 路径统一消费 explainability 输出，避免 per-path 解释漂移。
- 扩展 diagnostics replay tooling 的 explainability 夹具校验，检测 secondary order drift / hint taxonomy drift。
- 将 explainability drift 套件接入 quality gate 阻断流程（shell/PowerShell parity）。
- 同步更新 runtime diagnostics 文档、主线契约索引、roadmap 与 README 状态口径。

## Capabilities

### New Capabilities
- `cross-domain-arbitration-explainability-and-secondary-reason`: 定义 cross-domain 裁决可解释性、secondary reasons 有界输出和 remediation hint 契约。

### Modified Capabilities
- `runtime-config-and-diagnostics-api`: 新增 arbitration explainability 字段及 `additive + nullable + default` 兼容窗口约束。
- `runtime-readiness-preflight-contract`: 增加 preflight 输出对 explainability 字段的一致性要求。
- `runtime-readiness-admission-guard-contract`: 增加 admission 解释链路与 secondary reasons 对齐约束。
- `diagnostics-replay-tooling`: 增加 explainability fixture 对账能力与 drift 分类。
- `go-quality-gate`: 增加 explainability contract suites 阻断映射与 shell/PowerShell parity 要求。

## Impact

- 代码：
  - `runtime/config/readiness*`（explainability 输出装配）
  - `runtime/diagnostics/*` 与 `observability/event/*`（secondary reason/hint additive 字段）
  - `tool/diagnosticsreplay/*`（explainability fixture 规范化比较）
  - `integration/*`（Run/Stream/replay explainability parity suites）
  - `scripts/check-quality-gate.*`
- 文档：
  - `docs/runtime-config-diagnostics.md`
  - `docs/mainline-contract-test-index.md`
  - `docs/development-roadmap.md`
  - `README.md`
- 兼容性：
  - 不改变业务终态机，仅增强解释层契约；
  - 新增字段遵循 `additive + nullable + default`；
  - secondary reasons 输出保持 bounded-cardinality，防止诊断膨胀。
