# Minimal Session History Checkpoint Replay

文档基线：先验证 history root/leaf 连续性，再从 leaf 创建独立 branch Run，最后以 fixture-only replay 验证不执行工具/provider。

semantic anchor：`session.history_leaf_branch_restore_replay`

runtime path：`core/types,orchestration/snapshot,tool/diagnosticsreplay,runtime/diagnostics`

expected markers：`history_chain_validated`、`branch_run_distinct`、`replay_side_effect_free`

rollback notes：仅移除示例目录和 smoke 断言；不修改 snapshot manifest、checkpoint store 或 replay owner。

## Run

`go run ./examples/agent-modes/session-history-checkpoint-replay/minimal`

## Prerequisites

Go 工具链；仅使用本地核心类型和 replay fixture。

## Real Runtime Path

`core/types, orchestration/snapshot, tool/diagnosticsreplay, runtime/diagnostics`

## Expected Output/Verification

应看到 `history_chain_validated`、`branch_run_distinct`、`replay_side_effect_free` 三个 marker。

## Failure/Rollback Notes

校验失败时不产生 restore、branch 或 replay 副作用；删除该示例即可回滚。
