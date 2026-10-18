# context 组件说明

## 功能域

`context` 提供上下文装配链路（语义阶段）所需能力：

- 装配编排：`context/assembler`
- journal 存储：`context/journal`
- guard 校验：`context/guard`
- stage2 provider 适配：`context/provider`
- 派生预算投影契约：`context/budgetprojection`

Canonical 架构入口：`docs/runtime-harness-architecture.md`

## 架构设计

当前实现由 `Assembler` 统一编排多阶段流程：

- context-prefix-and-journal-baseline：prefix hash + guard + journal intent/commit
- context-stage2-routing-and-disclosure：stage2 规则/agentic 路由 + provider 拉取 + tail recap
- context-pressure-compaction-and-swapback：压力分区、压缩、prune、spill、semantic compaction
- context-production-hardening-and-threshold-governance：阈值覆盖与触发来源追踪（通过压力阶段统计字段体现）
- `context-compression-runtime-handoff`：压缩后的有界事实交接与引用校验（`context/handoff`），不拥有 session/checkpoint/artifact 正文。
- `context-budget-aware-derived-projection`：由既有预算 owner 事实派生的有界只读剩余预算投影（`context/budgetprojection`），不拥有预算事实、不建立第二本账本。

归档 165（`establish-host-context-assembly-handoff-contract`）已交付 `host_context_projection.v1` direct bridge：`context/hostprojection` 以 constructor/API 明确 opt-in 验证 detached、reference-only descriptor（32 KiB、64 references、256 UTF-8 bytes/reference、128 UTF-8 bytes/digest），且仅在结构/limits → canonical digest → correlation/capability subset 校验后调用宿主 fresh current-authority resolver。direct Run/Stream 仅在既有 Context Assembler 启用时复用该接缝；禁用时保留 assembler bypass 语义，projection 不会被 admit 或 resolve。`handoff.v1` 只可附带 additive、nullable 的 reference-only correlation；关联 restore 必须使用显式 current resolver，且只返回 caller-owned injection 所需的 transient material，不自行注入、resume 或 run。有限 `host_context.*` 错误 fail-closed，且不暴露 body 或 callback error。bridge 不持久化 resolver body，也不得把 source body、业务授权、private memory/knowledge、预算事实或业务 commit 迁入 `context/*`；dedicated replay、双平台 gate、标准 quality-path 接线、验证记录与[离线示例](../examples/host-context-assembly/README.md)已交付。边界与回滚见 [hostprojection README](hostprojection/README.md) 和[归档 proposal](../openspec/changes/archive/165-establish-host-context-assembly-handoff-contract/proposal.md)。

`provider` 子域负责 file/http/rag/db/elasticsearch 的检索适配和错误分层（`transport|protocol|semantic`）。

## 关键入口

- `assembler/assembler.go`
- `assembler/handoff.go`（可选 `AssembleWithHandoff` / `RestoreHandoff` 接缝）
- `assembler/context_pressure_recovery.go`
- `journal/storage.go`
- `guard/guard.go`
- `provider/provider.go`
- `budgetprojection/projection.go`（`budget_projection.v1` 派生投影契约）
- `budgetprojection/benchmark.go`（离线预算利用 benchmark）

## 边界与依赖

- `context/*` 不直接依赖 provider 官方 SDK；模型能力应通过 `model/*` 间接复用。
- stage2 错误层、reason code、hint 元数据需保持契约稳定，供 diagnostics 聚合。
- 该域只生成标准结果与事件，不直接写 `runtime/diagnostics` 存储。
- handoff 仅保存 `handoff.v1` 结构化事实、推断、待确认事项与 source references；恢复通过既有 owner 和 reference-first 接缝完成。
- `context/budgetprojection` 只依赖标准库，不 import 任何 Baymax 包；预算事实仍由 `runtime/config`（ReAct iteration/tool-call limit、`budget_admission.v1`）、`orchestration/scheduler`（`ParentRemainingBudget`）与 `core/runner`（React Plan Notebook）各自拥有。投影是纯派生函数，不写回事实源、不参与终态判定。
- active host-context bridge 采用 constructor/API 明确 opt-in，不新增 env/file/hot-update 键、RuntimeRecorder payload 或 QueryRuns 字段；descriptor/handoff digest 是一致性证据而非当前 source access 或业务授权。resolver 每次仅在验证后由宿主按当前授权解析，解析的 body 是当次 assembly/restore 的 transient material，不会持久化，也不得直接写 diagnostics。bridge 不写回 `budget_projection.v1` 的 source facts、不建立预算状态或 admission cache；incomplete/saturated 的既有派生语义和 terminal owner 保持不变。

## 派生预算投影契约（`budget_projection.v1`）

版本化、有界、只读的派生预算投影契约，把既有预算 owner 的事实快照归一化为可投影的剩余预算视图，并配套确定性的离线预算利用 benchmark。任一维度事实缺失时按 `additive + nullable + default` 表达（`available=false`，不伪造剩余值）。

稳定分类码（与 `tool/diagnosticsreplay` 及 gate 脚本共享同一词表）：

- `budget_projection_schema_drift`
- `budget_projection_unknown_version`
- `budget_facts_missing_iteration_limit`
- `budget_facts_missing_tool_call_limit`
- `budget_facts_missing_time_budget`
- `budget_facts_missing_cost_threshold`
- `budget_projection_negative_remaining`
- `budget_projection_ratio_out_of_range`
- `budget_projection_pressure_level_mismatch`
- `budget_projection_note_unbounded`
- `budget_projection_digest_mismatch`
- `budget_projection_overflow_drift`
- `budget_projection_writeback_shape_detected`
- `budget_benchmark_schema_drift`
- `budget_benchmark_metric_mismatch`
- `budget_benchmark_recovery_recompute_drift`
- `budget_benchmark_overflow_drift`
- `budget_projection_replay_not_idempotent`

门禁：`scripts/check-budget-aware-context-projection-contract.sh` / `.ps1`。

## 配置与默认值

- 语义阶段阈值与策略默认值由 `runtime/config` 提供，`context/*` 只消费快照。
- Stage2 外部检索默认采用 best-effort，可按治理策略切换 fail-fast。
- 压力压缩与阈值治理默认走保守参数，避免过度剪裁。

## 可观测性与验证

- 关键验证：`go test ./context/assembler ./context/guard ./context/journal ./context/provider ./context/budgetprojection -count=1`。
- 可观测指标包括 phase 延迟、pressure level、stage2 provider 错误分层。
- 回归重点是 run/stream 语义一致与 reason taxonomy 稳定。

## 扩展点与常见误用

- 扩展点：新增 stage2 provider adapter、扩展压力压缩 reranker/scorer、接入新阈值治理策略。
- 常见误用：在 assembler 内直接做 provider SDK 调用，绕过 provider 抽象层。
- 常见误用：无诊断标注地引入压缩策略变更，导致线上排障困难。
- 常见误用：把 `context/budgetprojection` 当成预算事实源（新增账本字段、回写剩余预算、或用 pressure 触发终态）。它只是 source-owned facts 的只读派生投影。
