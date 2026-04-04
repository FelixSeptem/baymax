## Why

当前 A68 正在实施 realtime protocol，但 ReAct 场景的上下文组织仍缺统一合同：引用注入与正文展开混在同一路径、子代理回传缺少标准结构、激进清理缺少收益门槛、swap-back 仍偏 run 级粗粒度，导致上下文膨胀、噪声注入与回放漂移风险持续存在。roadmap 已将 A67-CTX 定位为 A68 之后的下一顺位收口项，此时启动提案可以把 context 组织同域需求一次性并入单一合同，避免再拆平行提案。

## What Changes

- 新增 A67-CTX 主合同：JIT context organization + reference-first assembly。
- 新增 reference-first 两段式注入合同：
  - `discover_refs -> resolve_selected_refs`；
  - 默认优先注入引用（path/id/type/locator），按需展开正文。
- 新增 isolate handoff 合同：
  - 子代理回传固定结构 `summary`、`artifacts[]`、`evidence_refs[]`、`confidence`、`ttl`；
  - 主代理默认消费摘要与引用，正文按策略延后解析。
- 新增 context edit gate 合同：
  - `clear_at_least` 收益阈值（预计释放 token 与稳定性收益比）；
  - 未达阈值禁止激进编辑。
- 新增 relevance swap-back 与 lifecycle tiering 合同：
  - swap-back 从 run 级回填升级为 query + evidence tag 相关性回填；
  - 统一 `hot|warm|cold` 分层与 TTL/淘汰治理。
- 新增 task-aware recap 合同：
  - recap 必须基于本轮实际选择/剪裁/外化动作生成结构化摘要。
- 新增配置域：
  - `runtime.context.jit.reference_first.*`
  - `runtime.context.jit.isolate_handoff.*`
  - `runtime.context.jit.edit_gate.*`
  - `runtime.context.jit.swap_back.*`
  - `runtime.context.jit.lifecycle_tiering.*`
- 新增 QueryRuns additive 字段（最小集）：
  - `context_ref_discover_count`
  - `context_ref_resolve_count`
  - `context_edit_estimated_saved_tokens`
  - `context_edit_gate_decision`
  - `context_swapback_relevance_score`
  - `context_lifecycle_tier_stats`
  - `context_recap_source`
- 新增 replay fixtures：
  - `context_reference_first.v1`
  - `context_isolate_handoff.v1`
  - `context_edit_gate.v1`
  - `context_relevance_swapback.v1`
  - `context_lifecycle_tiering.v1`
- 新增 drift taxonomy：
  - `reference_resolution_drift`
  - `isolate_handoff_drift`
  - `edit_gate_threshold_drift`
  - `swapback_relevance_drift`
  - `lifecycle_tiering_drift`
  - `recap_semantic_drift`
- 新增 gate：`check-context-jit-organization-contract.sh/.ps1`，并接入 `check-quality-gate.*`。
- 一次性收口约束：A67-CTX 同域需求仅允许在本提案 tasks 内增量吸收，不再新增平行 context 组织提案。

## Capabilities

### New Capabilities
- `jit-context-organization-and-reference-first-assembly-contract`: 定义 reference-first、isolate handoff、edit gate、relevance swap-back、lifecycle tiering 与 task-aware recap 的统一合同。

### Modified Capabilities
- `runtime-config-and-diagnostics-api`: 增加 `runtime.context.jit.*` 配置域与 A67-CTX additive 诊断字段。
- `react-loop-and-tool-calling-parity-contract`: 扩展 ReAct parity 到 JIT context organization 的 Run/Stream 语义等价。
- `diagnostics-replay-tooling`: 增加 A67-CTX fixtures 与 drift 分类断言。
- `go-quality-gate`: 增加 context-jit contract gate、impacted suites 阻断与边界断言。

## Impact

- 代码：
  - `context/*`（reference-first、isolate handoff、edit gate、swap-back、tiering 与 recap 接线）
  - `core/runner`（ReAct 流程中 JIT context 调度边界）
  - `runtime/config`（A67-CTX 配置解析、校验、热更新回滚）
  - `runtime/diagnostics`、`observability/event`（A67-CTX additive 字段）
  - `tool/diagnosticsreplay`、`integration/*`（fixtures + drift tests）
  - `scripts/check-context-jit-organization-contract.*` + `scripts/check-quality-gate.*`
- 文档：
  - `docs/runtime-config-diagnostics.md`
  - `docs/mainline-contract-test-index.md`
  - `docs/development-roadmap.md`
  - `README.md`
- 兼容性与边界：
  - 对外 API 不引入 breaking 变更；新增字段遵循 `additive + nullable + default`。
  - 不改变 A56 ReAct loop 终止 taxonomy 与 A58 决策解释语义，不新增平行 loop 或平行决策链。
  - `context/*` 保持不直接依赖 provider 官方 SDK；运行态写入保持 `RuntimeRecorder` 单写入口。
