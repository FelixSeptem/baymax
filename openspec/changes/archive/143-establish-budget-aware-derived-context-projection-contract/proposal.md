## Why

Baymax 已有 ReAct iteration/tool-call limit、`runtime.react.on_budget_exhausted`、runtime budget admission、scheduler `ParentRemainingBudget`、ReAct Plan Notebook 与 task-aware tail recap，但**模型决策上下文里没有任何一条由这些 owner 计算出的剩余预算事实**。roadmap 把「预算感知派生上下文投影」列为观察候选，其启动条件是：先建立 benchmark、fixture 与 replay，比较 completion rate、iteration/tool-call usage、token/latency、repeated-loop rate、Run/Stream parity 与 snapshot/recovery 后的确定性重算；只有数据证明模型可见预算事实能稳定改善决策质量，才讨论策略提示或 tail projection 的最小 contract。当前缺口不是预算 source-of-truth（已存在且各有 owner），而是**缺少一份有界、只读、nullable 的派生投影契约与可回放证据基线**，因此无法回答「预算增加是否改善完成质量，还是只增加重复循环」。

现状审计（本提案 1.1 阶段实测，非推测）：

- `runtime/config/runtime_react.go` 拥有循环预算事实：`RuntimeReactConfig.MaxIterations`（L33）、`ToolCallLimit`（L34）、`OnBudgetExhausted`（L36，当前唯一合法值 `fail_fast`），终态常量 `react.max_iterations_exceeded` / `react.tool_call_limit_exceeded`（L24-25）。`normalizeRuntimeReactConfig`（L53-68）对非正值回落到默认值，因此**配置层不暴露"已用多少、还剩多少"**。
- `runtime/config/readiness.go` 的预算准入只覆盖 **cost 与 latency** 两个维度：`RuntimeAdmissionBudgetSnapshotVersionV1 = "budget_admission.v1"`（L113）、`RuntimeAdmissionBudgetDecision` 的 `allow`/`degrade`/`deny`（L110-112）、`RuntimeAdmissionBudgetCostEstimate{Token,Tool,Sandbox,Memory,Total}`（L267）与 `RuntimeAdmissionBudgetLatencyEstimate{TokenMs,ToolMs,SandboxMs,MemoryMs,TotalMs}`（L275）。这是**准入时点的一次性估算**，不是运行中的剩余预算，也不含 iteration/tool-call 维度。
- `orchestration/scheduler/types.go:509` 的 `SpawnRequest.ParentRemainingBudget time.Duration` 是唯一的父子预算传递字段；`Scheduler.SpawnChild`（`scheduler.go:153`）在 `<=0` 时返回 `BudgetRejectParentBudgetExhausted`，并在 `ChildTimeout > ParentRemainingBudget` 时裁剪并置 `TimeoutResolution.ParentBudgetClamped`（L170-172）。该字段**只被用于准入裁决，不派生出可投影的剩余时间事实**。
- `core/runner/react_plan_notebook.go:29` 的 `reactPlanNotebook` 拥有 plan lifecycle（plan id/version/status、create/revise/complete/recover 历史、`ChangeTotal`、`RecoverCount`，受 `maxHistory` 有界裁剪），但同样不向上下文投影。
- `context/assembler/assembler.go` 已有现成的「source-owned facts → 有界派生投影」模式：`buildTaskAwareTailRecap(cfg runtimeconfig.ContextAssemblerCA2Config, stage types.AssembleStage) (types.TailRecap, string)`（L1353）返回有界 `TailRecap{Status,Decisions,Todo,Risks}` 与来源标识 `task_aware.stage_actions.v1`（L41），并经 `sanitizeRecap`（L1480）脱敏。但 `AssembleStage`（`core/types/types.go:659`）只有 memory 侧预算字段（`MemoryBudgetUsed`，L681），**没有 iteration/tool/time/cost 的 remaining 事实**；`grep -rn "Remaining" context/` 返回空。
- 既有门禁 `scripts/check-runtime-budget-admission-contract.sh` 已强制两条不变量：`budget_control_plane_absent`（不得引入托管控制面）与 `budget_field_reuse_required`（禁止第二本账本、禁止复制既有字段别名，L130-132）。因此本提案不得新增预算账本字段、不得新增 `runtime.admission.*` 配置键、也不得改动准入阈值/维度/降级动作。

结论：事实源齐备且各有 owner，缺口是**派生投影层缺失 + 没有可回放的收益证据**。本提案只交付契约、benchmark、fixture、replay 与 gate，不把投影接进运行时上下文，也不引入 adaptive budget 策略。

## What Changes

- 新增版本化、provider-neutral、只读的派生预算投影契约 `budget_projection.v1`：有界 schema、canonical 归一化、确定性 SHA-256 digest、稳定 drift/gap 分类词表。
- 新增 `context/budgetprojection` 契约包（只依赖标准库，不依赖 `runtime/*`、`orchestration/*`、`model/*`、`observability/*`），把既有 owner 的预算事实**快照**归一化为 `BudgetFacts`，再派生出有界只读的 `BudgetProjection`（remaining iteration/tool/time/cost、pressure level、bounded notes、digest）。
- 明确 `additive + nullable + default` 口径：任一维度事实缺失时该维度 `available=false`，MUST NOT 以 0 / 默认值冒充"剩余为 0"或"充足"；未知字段安全忽略；历史 fixture 缺字段不得解析失败。
- 新增确定性、离线的预算利用 benchmark：由 fixture trace（每步预算事实 + `progress`/`repeat`/`terminal_success`/`terminal_exhausted` 结果）计算 completion rate、repeated-loop rate、budget utilization 与 saturation rate，并校验 snapshot/recovery 后的重算 digest 一致。**benchmark 不调用模型、不依赖时钟与网络**，只做计算型比较。
- 新增离线、只读、确定性的 replay：解析 `budget_projection.v1`，校验投影与 fixture 期望一致、分类码稳定、benchmark 指标可重算且幂等。
- 新增 shell/PowerShell 对等 gate：投影只读性、无第二本账本、无新配置键、有界性、taxonomy 稳定、replay 幂等、benchmark 确定性重算；失败分类码可审计。
- 本变更**不修改运行时可观测行为**：不修改 `runtime/config`、`orchestration/scheduler`、`core/runner`、`context/assembler` 的任何现有代码或配置语义，不把投影接入 tail recap，不新增 runtime 配置键，不引入 adaptive budget 策略，因此回滚只删除新增文件。
- `example impact`：`无需示例变更（附理由）`。

## Capabilities

### New Capabilities

- `budget-aware-derived-context-projection`: 由既有预算 owner 事实派生的有界只读剩余预算投影契约，覆盖 remaining iteration/tool/time/cost、pressure level、bounded notes、确定性 digest、nullable+default 兼容口径与离线 benchmark/replay/gate。

### Modified Capabilities

- `diagnostics-replay-tooling`: 扩展 replay 命名空间与确定性分类，新增 `budget_projection.v1` 与派生预算投影/benchmark 的 drift 码。

## Example Impact Assessment

`无需示例变更（附理由）`

理由：本变更只新增离线契约包、fixture、benchmark、replay 与 gate，不修改 `examples/agent-modes` 的配置键、runtime path、expected markers 或任何可观察语义；派生投影被显式排除在运行时接线之外（不进 tail recap、不改 ReAct 循环），因此示例无需跟随变更。若后续增量 change 决定把投影接入模型可见上下文并改变示例可观察输出，必须重新评估并遵循文档先行规则（先更新 `MATRIX.md` 与对应模式 README）。

## Impact

- 受影响实现 owner：`context/budgetprojection`（新增派生投影契约与 benchmark，只依赖标准库）、`tool/diagnosticsreplay`（新增 replay 命名空间）、`tool/contributioncheck`（新增边界守卫测试）。
- 受影响测试与证据：投影契约正向/负向/边界测试、benchmark 确定性与重算测试、replay fixture 与幂等测试、shell/PowerShell gate。
- 受影响文档：`docs/mainline-contract-test-index.md`、`context/README.md`、`docs/development-roadmap.md`、`README.md`、`docs/runtime-module-boundaries.md`。
- 明确不做：不新增第二本预算账本、不新增 runtime 配置键、不改 admission 阈值/维度/降级动作、不改 scheduler `ParentRemainingBudget` 语义、不改 Plan Notebook lifecycle、不把投影接入 `context/assembler` 或 tail recap、不修改 ReAct 循环策略、不引入 adaptive budget、不写 `runtime/diagnostics` 字段、不绕过 `RuntimeRecorder`、不落盘 raw reasoning/完整 transcript/无界 payload。
- 回滚：删除 `context/budgetprojection`、replay 命名空间、fixture、边界测试与两个 gate 脚本，并从 `scripts/check-quality-gate.*` 摘除接线；无持久化迁移、无配置变更、无运行时语义回退。
