# Session History / Checkpoint / Replay

该模式演示 Baymax 的 session history、checkpoint lineage、branch/fork 与离线 replay 边界。

- semantic anchor：`session.history_leaf_branch_restore_replay`
- runtime path：`core/types,orchestration/snapshot,tool/diagnosticsreplay,runtime/diagnostics`
- owner boundary：history/session source、snapshot restore source 和 diagnostics replay 各自保留事实所有权；模式只展示引用投影。
- 预期标记：`history_chain_validated`、`branch_run_distinct`、`restore_conflict_classified`、`replay_side_effect_free`
- rollback：删除该示例模式及对应 smoke 行不会改变 runtime、snapshot 或 replay 事实源。

## Minimal

见 `minimal/README.md`。覆盖连续 history leaf、独立 branch Run 和无副作用 replay。

## Production-ish

见 `production-ish/README.md`。额外覆盖 strict restore 冲突、replay identity 冲突和 Run/Stream parity。

## Run

在仓库根目录执行 `go run ./examples/agent-modes/session-history-checkpoint-replay/minimal` 或 production-ish 变体。

## Prerequisites

需要 Go 工具链和仓库依赖；示例使用本地 fixture，不需要 Provider、网络服务或 hosted session store。

## Real Runtime Path

`core/types -> orchestration/snapshot -> tool/diagnosticsreplay -> runtime/diagnostics`。

## Expected Output/Verification

输出包含 `verification.mainline_runtime_path=ok`、history/branch/replay markers 和稳定 `result.signature`。

## Failure/Rollback Notes

若 history continuity、branch lineage 或 replay side-effect 校验失败，示例立即退出；回滚只移除该模式目录和矩阵行。
