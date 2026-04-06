# orchestration/snapshot 组件说明

## 功能域

`orchestration/snapshot` 提供统一 state/session snapshot 合同层，覆盖导出、导入、版本兼容与幂等恢复语义。

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

## 配置与默认值

- 默认配置口径：
  - `runtime.state.snapshot.enabled=false`
  - `runtime.state.snapshot.restore_mode=strict`
  - `runtime.state.snapshot.compat_window=1`
  - `runtime.state.snapshot.schema_version=state_session_snapshot.v1`
- 未显式传入 `operation_id` 时，导入幂等键默认使用 `manifest.digest`。

## 可观测性与验证

- `go test ./orchestration/snapshot -count=1`
- `go test ./integration -run '^TestUnifiedSnapshot' -count=1`
- 门禁脚本：`scripts/check-state-snapshot-contract.sh` / `scripts/check-state-snapshot-contract.ps1`

## 扩展点与常见误用

- 扩展点：新增 segment 版本时保持兼容窗口治理与冲突码稳定。
- 常见误用：在 compatible 模式下放宽窗口但不补 drift 回归，导致跨版本恢复不可控。
- 常见误用：把 `operation_id` 幂等语义替换为随机值，破坏重复导入 no-op 保证。
