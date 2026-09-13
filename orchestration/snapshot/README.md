# orchestration/snapshot 组件说明

## 功能域

`orchestration/snapshot` 提供统一 state/session snapshot 合同层，覆盖导出、导入、版本兼容与幂等恢复语义。

本提案的 durable task/attempt workspace binding 与 completion safe-point 只增加可选的引用和恢复校验：snapshot 仍是 manifest/schema/digest/restore-policy 的事实源，不解析或保存 workspace 内容，也不拥有 scheduler、mailbox 或 Runner safe-point 状态。

## 架构设计

- `Manifest`：定义 `state_session_snapshot.v1` 的 schema、segment、digest。
- `Export/ExportManifest`：对输入 payload 做规范化并生成稳定 digest。
- `Importer.Import`：执行 `strict|compatible` 恢复策略、兼容窗口判定与 operation 级幂等收敛。
- `ImportError`：输出稳定 `conflict_code`，用于回放与门禁分类。

## 关键入口

- `manifest.go`
- `contract.go`

## 边界与依赖

- 该包只负责 snapshot 合同，不重写既有 checkpoint/snapshot 存储事实源。
- 该包不直接写入 `runtime/diagnostics`，观测写入仍经标准事件单写路径收口。
- 兼容性判定以版本/窗口规则为准，冲突场景必须 fail-fast。
- scheduler task/attempt、lease、mailbox completion 与 runtime-input safe point 的 source owner 保持不变；snapshot 只做 reference-only association validation 和 pre-mutation reconciliation。
- workspace provenance 使用有界 `workspace_id`、`change_set_id`、checkpoint/reference 与完整性分类；不读取 workspace、Git、artifact 或 completion body。

## 配置与默认值

- 默认配置口径：
  - `runtime.state.snapshot.enabled=false`
  - `runtime.state.snapshot.restore_mode=strict`
  - `runtime.state.snapshot.compat_window=1`
  - `runtime.state.snapshot.schema_version=state_session_snapshot.v1`
- 未显式传入 `operation_id` 时，导入幂等键默认使用 `manifest.digest`。

## Compatibility, privacy, and rollback

新增 binding/completion references 遵循 `additive + nullable + default`。历史 snapshot 缺少这些字段时按 `binding absent` 处理；未知字段安全忽略；显式提供的 invalid/mismatched/stale/dirty/conflict/drift association 在 strict 模式下于任何 mutation 前拒绝。compatible 模式只能在既有 `compat_window` 内执行 bounded downgrade，并保留稳定 reason/classification。

回滚只需移除新增引用投影、fixture/replay/gate 与对应恢复适配，不需要持久化迁移。既有 `state_session_snapshot.v1`、checkpoint provenance、scheduler lifecycle、mailbox delivery、Run/Stream safe-point 和 terminal outcome 继续有效。

本提案不新增 runtime configuration key、环境变量或 hot-update 分支；仍使用现有 `runtime.state.snapshot.*` 和固定 `env > file > default` 语义。也不引入 workspace store、hosted artifact service、Git/worktree manager、第二套恢复状态机或 completion queue。

## 可观测性与验证

- `go test ./orchestration/snapshot -count=1`
- `go test ./integration -run '^TestUnifiedSnapshot' -count=1`
- 门禁脚本：`scripts/check-state-snapshot-contract.sh` / `scripts/check-state-snapshot-contract.ps1`

## 扩展点与常见误用

- 扩展点：新增 segment 版本时保持兼容窗口治理与冲突码稳定。
- 常见误用：在 compatible 模式下放宽窗口但不补 drift 回归，导致跨版本恢复不可控。
- 常见误用：把 `operation_id` 幂等语义替换为随机值，破坏重复导入 no-op 保证。

## Verification ownership

- Scheduler association and stale-attempt checks: `orchestration/scheduler`.
- Snapshot import/recovery and compatibility: `orchestration/snapshot` plus `orchestration/composer`.
- Completion correlation and safe-point application: `orchestration/mailbox` and `core/runner`; snapshot stores only bounded pending references when explicitly supported.
- Offline normalization and privacy rejection: `tool/diagnosticsreplay` fixtures `durable_attempt_workspace_binding.v1` and `completion_safe_point_ownership.v1`.
