# Production-ish Session History Checkpoint Replay

在 minimal 路径上增加 strict/compatible restore 分类、conflicting operation identity 和 Run/Stream normalized parity 断言。

semantic anchor：`session.history_leaf_branch_restore_replay`

runtime path：`core/types,orchestration/snapshot,tool/diagnosticsreplay,runtime/diagnostics`

expected markers：`history_chain_validated`、`branch_run_distinct`、`restore_conflict_classified`、`replay_side_effect_free`、`run_stream_history_replay_parity`

rollback notes：回退只关闭该模式的 production-ish smoke，不影响既有 state snapshot、checkpoint provenance 或 diagnostics replay 合同。

## Run

`go run ./examples/agent-modes/session-history-checkpoint-replay/production-ish`

## Prerequisites

Go 工具链；不需要外部服务或 Provider 凭据。

## Real Runtime Path

`core/types, orchestration/snapshot, tool/diagnosticsreplay, runtime/diagnostics`

## Expected Output/Verification

除基础 markers 外，应看到 `restore_conflict_classified` 和 `run_stream_history_replay_parity`。

## Contract Gates

gate mapping：`check-session-history-checkpoint-replay-contract.*`。

PowerShell：`pwsh -File scripts/check-session-history-checkpoint-replay-contract.ps1`；等价 shell gate：`bash scripts/check-session-history-checkpoint-replay-contract.sh`。

## Variant Delta (vs minimal)

production-ish 增加 strict/compatible restore 冲突、operation identity 冲突和 Run/Stream parity 验证。

## Failure/Rollback Notes

冲突分类或 parity 不满足时立即失败；回退只移除 production-ish 示例和对应 smoke 配置。
