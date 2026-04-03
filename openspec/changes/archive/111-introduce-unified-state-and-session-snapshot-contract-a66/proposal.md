## Why

当前主干的恢复与状态持久化能力分散在 scheduler/composer/memory 等模块，虽然各自可用，但缺少统一 state/session snapshot 合同，导致跨模块恢复、迁移、回放和外部接入时需要拼装多套语义，成本高且容易漂移。A65 在实施中后，按顺序推进 A66，可一次性冻结统一快照导入导出合同，减少后续同域重复提案。

## What Changes

- 新增 A66 主合同：unified state/session snapshot contract。
- 新增统一快照描述面：覆盖 `runner/session`、`scheduler/mailbox`、`composer recovery`、`memory` 的版本化 state descriptor。
- 冻结快照导入导出语义：
  - snapshot manifest（版本、来源、时间戳、模块分段、校验摘要）；
  - restore policy（strict|compatible）、冲突策略、部分恢复规则；
  - 幂等导入与回放一致性。
- 新增配置域：
  - `runtime.state.snapshot.*`
  - `runtime.session.state.*`
- 冻结兼容窗口与升级策略：支持 schema versioned compatibility window，不允许 silent downgrade。
- 新增 QueryRuns additive 字段：`state_snapshot_version`、`state_restore_action`、`state_restore_conflict_code`、`state_restore_source`。
- 新增 replay fixture：`state_session_snapshot.v1`，并冻结 drift taxonomy：
  - `snapshot_schema_drift`
  - `state_restore_semantic_drift`
  - `snapshot_compat_window_drift`
  - `partial_restore_policy_drift`
- 新增 gate：`check-state-snapshot-contract.sh/.ps1`，并接入 `check-quality-gate.*`。
- 一次性收口约束：A66 同域需求（状态导入导出、兼容窗口、恢复冲突仲裁、回放门禁）仅允许在 A66 增量吸收，不再拆平行提案。

## Capabilities

### New Capabilities
- `unified-state-and-session-snapshot-contract`: 统一 state/session snapshot 的导入导出、兼容与恢复治理合同。

### Modified Capabilities
- `runtime-config-and-diagnostics-api`: 增加 `runtime.state.snapshot.*` / `runtime.session.state.*` 配置与 A66 additive 诊断字段。
- `multi-agent-session-recovery`: 扩展恢复路径以消费统一 snapshot manifest 与冲突策略。
- `memory-scope-and-builtin-filesystem-v2-governance-contract`: 将 memory lifecycle 与 unified snapshot 导入导出对齐，保持既有事实源语义不变。
- `diagnostics-replay-tooling`: 增加 `state_session_snapshot.v1` fixture 与 A66 drift 分类断言。
- `go-quality-gate`: 增加 state/session snapshot contract gate 与 required-check 候选。

## Impact

- 代码：
  - `runtime/config`（snapshot/session 配置解析、校验、热更新回滚）
  - `orchestration/composer`、`orchestration/scheduler`（统一 snapshot import/export 接缝）
  - `memory/*`（与统一 snapshot 生命周期对齐，不重写事实源）
  - `runtime/diagnostics`、`observability/event`（A66 additive 字段）
  - `tool/diagnosticsreplay`、`integration/*`（A66 fixtures + drift tests）
  - `scripts/check-state-snapshot-contract.*` + `scripts/check-quality-gate.*`
- 文档：
  - `docs/runtime-config-diagnostics.md`
  - `docs/mainline-contract-test-index.md`
  - `docs/development-roadmap.md`
  - `README.md`
- 兼容性与边界：
  - 对外 API 不引入 breaking 变更；新增字段遵循 `additive + nullable + default`。
  - A66 必须复用现有 checkpoint/snapshot 语义与 A59 memory lifecycle，不得重写存储层事实源。
  - 保持 `library-first`：不引入平台化状态控制面或托管恢复服务。
