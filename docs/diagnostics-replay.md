# Diagnostics JSON Replay（精简 + A47/A48 组合模式）

更新时间：2026-03-26

## 适用场景

- 复盘单次 run 的 Action Timeline 最小执行路径。
- 离线排障（无需连接运行时 API）。
- 回归校验（固定输入/输出契约）。

## 输入契约

### 1) 精简模式（D1）

当前仅支持 JSON 输入，支持两种形态：

1. `timeline_events` 数组（已归一化）
2. `events` 数组（原始事件，自动提取 `type=action.timeline`）

最小必填字段：
- `run_id`
- `sequence`（`> 0`）
- `phase`
- `status`
- `timestamp`（或 `time`，RFC3339）

### 2) A47 组合模式（readiness-timeout-health）

组合模式用于跨域语义回放门禁（quality-gate blocking check），输入为版本化 fixture：

- `version`：当前固定 `a47.v1`
- `cases[]`：场景矩阵项，必须覆盖最小轴：
  - readiness：`ready|degraded|blocked` + `readiness_strict=true|false`
  - timeout：`profile|domain|request` + `none|clamped|rejected`
  - adapter health：`healthy|degraded|unavailable` + `closed|open|half_open` + `adapter_required=true|false`
- 每个 case 包含：`run`、`stream`、`expected`、`idempotency`
  - `run/stream/expected` 强约束字段：`status/primary_code/reason_taxonomy/timeout source + budget outcome + trace/circuit_state`
  - `idempotency`：`first_logical_ingest_total` 与 `replay_logical_ingest_total` 必须保持稳定

### 3) A48 组合模式（cross-domain primary reason arbitration）

跨域 primary reason 模式用于验证 timeout/readiness/adapter-health 竞争下的固定裁决语义。

- `version`：当前固定 `a48.v1`
- `cases[]`：每个 case 必须包含 `run`、`stream`、`expected`、`idempotency`
- 强约束字段：
  - `runtime_primary_domain`
  - `runtime_primary_code`
  - `runtime_primary_source`
  - `runtime_primary_conflict_total`
- `idempotency`：`first_logical_ingest_total == replay_logical_ingest_total`

## 使用方式

### 精简模式 CLI

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

### 组合模式（Go API）

```go
raw, _ := os.ReadFile("integration/testdata/diagnostics-replay/a47/v1/composite-success.json")
out, err := diagnosticsreplay.EvaluateCompositeFixtureJSON(raw)
if err != nil {
    // err.(*diagnosticsreplay.ValidationError).Code
}
_ = out // deterministic normalized output
```

```go
raw, _ := os.ReadFile("integration/testdata/diagnostics-replay/a48/v1/success.json")
out, err := diagnosticsreplay.EvaluateArbitrationFixtureJSON(raw)
if err != nil {
    // err.(*diagnosticsreplay.ValidationError).Code
}
_ = out // deterministic normalized output
```

## 稳定错误码

- `invalid_json`
- `invalid_json_shape`
- `missing_timeline_events`
- `missing_required_field`
- `invalid_field_type`
- `invalid_timestamp`
- `schema_mismatch`（fixture 结构/版本/矩阵覆盖缺失）
- `semantic_drift`（taxonomy/source/state/idempotency 漂移）
- `ordering_drift`（ordering 非确定性漂移）
- `precedence_drift`（A48：timeout/reject 与 blocked/required/degraded 层级漂移）
- `tie_break_drift`（A48：同层 lexical tie-break 或 conflict_total 漂移）
- `taxonomy_drift`（A48：primary code/source/domain taxonomy 漂移）

这些错误码用于 CI 契约回归和脚本自动判定，除非显式版本化，不应随意变更。

## CI 门禁

- Linux: `bash scripts/check-diagnostics-replay-contract.sh`
- PowerShell: `pwsh -File scripts/check-diagnostics-replay-contract.ps1`
- A47 组合回放（blocking）：
  - Linux/PowerShell 统一由 `scripts/check-quality-gate.sh` / `scripts/check-quality-gate.ps1` 执行
  - 目标套件：`go test ./tool/diagnosticsreplay ./integration -run 'Test(ReplayContractCompositeFixture|ReplayContractPrimaryReasonArbitrationFixture|ReadinessTimeoutHealthReplayContract|PrimaryReasonArbitrationReplayContract)' -count=1`

建议在分支保护中将 `diagnostics-replay-gate` 设置为 required status check。
