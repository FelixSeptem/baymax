## Why

A57 在实施期间会同时触发 `action gate`、`security s2`、`sandbox action/egress`、`adapter allowlist`、`readiness/admission` 多层判定。当前这些判定链路缺少统一 precedence 合同，存在以下风险：
- 同一请求在 Run/Stream、preflight/admission、不同入口路径下出现不同阻断结论；
- deny 来源不可解释，导致排障与审计链路割裂；
- replay 与 gate 无法稳定拦截“策略顺序漂移”类回归。

为避免在 A57 之后继续拆分同主题提案，需要用 A58 一次性冻结策略优先级与决策解释链。

## What Changes

- 新增 policy precedence + decision trace 主合同（A58）。
- 冻结跨策略层 canonical precedence matrix：`action_gate -> security_s2 -> sandbox_action -> sandbox_egress -> adapter_allowlist -> readiness_admission`。
- 冻结 deterministic tie-break 规则（同层冲突时 lexical code + stable source 顺序）。
- 新增 `runtime.policy.precedence.*`、`runtime.policy.tie_breaker.*`、`runtime.policy.explainability.*` 配置域并保持 `env > file > default`。
- 新增 QueryRuns additive 字段：`policy_decision_path`、`deny_source`、`winner_stage`、`tie_break_reason`。
- 将 precedence 输出接入 readiness preflight + admission guard，保持 deny side-effect-free。
- 新增 `policy_stack.v1` replay fixture 与 drift taxonomy（`precedence_conflict`、`tie_break_drift`、`deny_source_mismatch`）。
- 新增 `check-policy-precedence-contract.sh/.ps1` 并接入 `check-quality-gate.*`，暴露独立 required-check 候选。
- 同步 roadmap、README、runtime config/diagnostics 与主线 contract index。

## Capabilities

### New Capabilities
- `policy-precedence-and-decision-trace-contract`: 冻结跨策略层 precedence 与决策解释链语义。

### Modified Capabilities
- `runtime-config-and-diagnostics-api`: 增加 `runtime.policy.*` 配置域与 policy additive 诊断字段。
- `runtime-readiness-preflight-contract`: 增加 policy stack finding 聚合与 winner-stage 映射。
- `runtime-readiness-admission-guard-contract`: 增加 precedence-driven deny/allow 映射与 explainability 透传。
- `cross-domain-primary-reason-arbitration`: 增加 policy winner 与 primary-reason 输出对齐约束。
- `diagnostics-replay-tooling`: 增加 `policy_stack.v1` fixture 与 drift 分类断言。
- `go-quality-gate`: 增加 A58 contract gate 与独立 required-check 暴露。

## Impact

- 代码：
  - `runtime/config`、`runtime/config/readiness`（`runtime.policy.*` 配置与 preflight 映射）
  - `runtime/diagnostics`、`observability/event`（policy additive 字段）
  - `runtime/security`、`core/runner`（policy stack precedence 评估与 canonical 输出）
  - `tool/diagnosticsreplay`、`integration/*`（`policy_stack.v1` fixtures + drift tests）
  - `scripts/check-quality-gate.*` + `scripts/check-policy-precedence-contract.*`
- 文档：
  - `docs/development-roadmap.md`
  - `docs/mainline-contract-test-index.md`
  - `docs/runtime-config-diagnostics.md`
  - `README.md`
- 兼容性：
  - 外部 API 保持兼容；新增字段遵循 `additive + nullable + default`。
  - 不改变既有 deny/allow 语义，仅冻结跨策略层裁决顺序与解释字段。
