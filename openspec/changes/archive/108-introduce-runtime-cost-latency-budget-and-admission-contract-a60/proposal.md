## Why

A59 完成后 memory 路径的行为可解释性会明显提升，但运行时仍缺少统一的“成本/时延预算 admission 合同”，导致 token/tool/sandbox/memory 的综合开销在高负载下只能被动失败，无法做 deterministic 的准入与降级决策。A60 的目标是在不引入平台控制面的前提下，一次性冻结预算快照、准入决策、降级动作与回放门禁口径。

## What Changes

- 新增 A60 主合同：runtime cost-latency budget and admission contract。
- 冻结预算输入面：统一 token/tool/sandbox/memory 成本与时延的预算快照模型。
- 冻结 admission 决策语义：`allow|degrade|deny` 的 deterministic 判定与优先级。
- 冻结降级策略合同：`runtime.admission.degrade_policy.*` 与 `degrade_action` 的可观测输出。
- 新增配置域：
  - `runtime.admission.budget.cost.*`
  - `runtime.admission.budget.latency.*`
  - `runtime.admission.degrade_policy.*`
- 新增 QueryRuns additive 字段：`budget_snapshot`、`budget_decision`、`degrade_action`。
- 新增 replay fixture：`budget_admission.v1`，并冻结 drift taxonomy：
  - `budget_threshold_drift`
  - `admission_decision_drift`
  - `degrade_policy_drift`
- 新增 gate：`check-runtime-budget-admission-contract.sh/.ps1`，并接入 `check-quality-gate.*`。
- 一次性收口约束：A60 吸收预算 admission 同域需求（阈值、维度、降级动作、回放与门禁），后续同域新增仅允许以 A60 增量任务吸收，不再拆分平行提案。
- 文档同步到 runtime config/diagnostics、mainline contract index、roadmap、README。

## Capabilities

### New Capabilities
- `runtime-cost-latency-budget-and-admission-contract`: 冻结统一预算快照、准入判定与降级动作的 contract。

### Modified Capabilities
- `runtime-config-and-diagnostics-api`: 增加 `runtime.admission.budget.*` / `runtime.admission.degrade_policy.*` 配置字段与 budget admission additive 诊断字段。
- `runtime-readiness-admission-guard-contract`: 增加预算 admission 判定与 side-effect-free deny/degrade 语义约束。
- `diagnostics-replay-tooling`: 增加 `budget_admission.v1` fixture 与预算准入 drift 分类断言。
- `go-quality-gate`: 增加 runtime budget admission contract gate 与 required-check 候选。

## Impact

- 代码：
  - `runtime/config`（预算与降级配置解析/校验）
  - `runtime/config/readiness` 与 admission 相关路径（预算判定接入）
  - `runtime/diagnostics`、`observability/event`（additive 字段写入）
  - `tool/diagnosticsreplay`、`integration/*`（budget fixtures + drift tests）
  - `scripts/check-runtime-budget-admission-contract.*` + `scripts/check-quality-gate.*`
- 文档：
  - `docs/runtime-config-diagnostics.md`
  - `docs/mainline-contract-test-index.md`
  - `docs/development-roadmap.md`
  - `README.md`
- 兼容性：
  - 对外 API 不引入 breaking 变更；新增字段遵循 `additive + nullable + default`。
  - 不重定义 A58 的 `policy_decision_path/deny_source`，A60 仅引用其结果作为预算准入解释上下文。
  - 不引入托管预算控制面或服务化准入调度平面，保持 `library-first`。
