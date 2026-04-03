## Why

当前主线已经通过 A56 固化了 ReAct loop 的 Run/Stream parity 与工具调用闭环，但对“计划”本身仍缺少统一合同：计划创建、修订、恢复、完成的生命周期分散在业务侧提示词或外层编排中，导致复杂任务下 plan drift 难解释、恢复难复现、回放难阻断。A66 进入实施后，下一顺位推进 A67 可一次性收口 ReAct 计划治理与计划变更 hook，避免继续拆分同域提案。

## What Changes

- 新增 A67 主合同：ReAct plan notebook + plan-change hook。
- 新增 plan notebook 生命周期合同：
  - `create|revise|complete|recover` 四类动作；
  - 计划版本单调递增、历史留痕、终态冻结规则。
- 新增 plan-change hook 合同：
  - 计划变更前后回调（before/after）；
  - 上下文快照、变更原因、失败策略与 deterministic 顺序。
- 新增配置域：
  - `runtime.react.plan_notebook.*`
  - `runtime.react.plan_change_hook.*`
- 新增 QueryRuns additive 字段：
  - `react_plan_id`
  - `react_plan_version`
  - `react_plan_change_total`
  - `react_plan_last_action`
  - `react_plan_change_reason`
  - `react_plan_recover_count`
  - `react_plan_hook_status`
- 新增 replay fixture：`react_plan_notebook.v1`，并冻结 drift taxonomy：
  - `react_plan_version_drift`
  - `react_plan_change_reason_drift`
  - `react_plan_hook_semantic_drift`
  - `react_plan_recover_drift`
- 新增 gate：`check-react-plan-notebook-contract.sh/.ps1`，并接入 `check-quality-gate.*`。
- 一次性收口约束：A67 同域需求（计划生命周期、计划变更 hooks、计划回放与门禁）仅允许在 A67 增量吸收，不再新增平行 ReAct 计划治理提案。

## Capabilities

### New Capabilities
- `react-plan-notebook-and-plan-change-hook-contract`: ReAct 计划生命周期与计划变更 hook 的统一合同。

### Modified Capabilities
- `react-loop-and-tool-calling-parity-contract`: 扩展 ReAct parity 到计划生命周期语义，保持 Run/Stream 等价。
- `runtime-config-and-diagnostics-api`: 增加 `runtime.react.plan_notebook.*` / `runtime.react.plan_change_hook.*` 与 A67 additive 字段。
- `diagnostics-replay-tooling`: 增加 `react_plan_notebook.v1` fixture 与 A67 drift 分类断言。
- `go-quality-gate`: 增加 plan notebook contract gate 与 impacted suites 阻断。

## Impact

- 代码：
  - `core/runner`（ReAct plan notebook 生命周期接线）
  - `runtime/config`（A67 配置解析、校验、热更新回滚）
  - `runtime/diagnostics`、`observability/event`（A67 additive 字段）
  - `tool/diagnosticsreplay`、`integration/*`（A67 fixtures + drift tests）
  - `scripts/check-react-plan-notebook-contract.*` + `scripts/check-quality-gate.*`
- 文档：
  - `docs/runtime-config-diagnostics.md`
  - `docs/mainline-contract-test-index.md`
  - `docs/development-roadmap.md`
  - `README.md`
- 兼容性与边界：
  - 对外 API 不引入 breaking 变更；新增字段遵循 `additive + nullable + default`。
  - A67 必须复用 A56 ReAct 终止 taxonomy 与 A58 决策解释链，不新增平行 loop 或平行决策语义。
  - 保持 `library-first`：不引入托管计划控制面、远程计划服务或平台化编排中心。
