# Diagnostics JSON Replay（精简模式）

更新时间：2026-03-17

## 适用场景

- 复盘单次 run 的 Action Timeline 最小执行路径。
- 离线排障（无需连接运行时 API）。
- 回归校验（固定输入/输出契约）。

## 输入契约

当前仅支持 JSON 输入，支持两种形态：

1. `timeline_events` 数组（已归一化）
2. `events` 数组（原始事件，自动提取 `type=action.timeline`）

最小必填字段：
- `run_id`
- `sequence`（`> 0`）
- `phase`
- `status`
- `timestamp`（或 `time`，RFC3339）

## 使用方式

```bash
go run ./cmd/diagnostics-replay -input diagnostics.json
```

输出为精简 JSON：
- `run_id`
- `sequence`
- `phase`
- `status`
- `reason`（可选）
- `timestamp`

## 稳定错误码

- `invalid_json`
- `invalid_json_shape`
- `missing_timeline_events`
- `missing_required_field`
- `invalid_field_type`
- `invalid_timestamp`

这些错误码用于 CI 契约回归和脚本自动判定，除非显式版本化，不应随意变更。

## CI 门禁

- Linux: `bash scripts/check-diagnostics-replay-contract.sh`
- PowerShell: `pwsh -File scripts/check-diagnostics-replay-contract.ps1`

建议在分支保护中将 `diagnostics-replay-gate` 设置为 required status check。
