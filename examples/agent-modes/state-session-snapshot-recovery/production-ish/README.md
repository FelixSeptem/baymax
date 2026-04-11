# state-session-snapshot-recovery (production-ish)

## Purpose
用真实语义链路演示 `state-session-snapshot-recovery` 的生产治理闭环：在最小链路基础上增加 gate 与回放绑定。

## Variant Delta (vs minimal)
- 生产链路引入额外回放帧，故意放大漂移检测，路径差异来自行为而非 marker 文本。
- 在幂等判定后增加治理决策：`allow / hold_for_review / deny`。
- 将治理决策与快照/回放摘要绑定到 `replay_binding`，用于审计回放。

## Run
go run ./examples/agent-modes/state-session-snapshot-recovery/production-ish

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `snapshot.export_restore_replay`.
- Classification: `state.session_snapshot_recovery`.
- Runtime path evidence: `core/runner,tool/local,runtime/config,orchestration/snapshot,runtime/diagnostics`.
- Semantic flow:
  - minimal 的 3 步快照链路；
  - 追加 `governance_snapshot_gate_enforced` 与 `governance_snapshot_replay_bound` 两步治理链路。
- Related contracts: `unified-state-and-session-snapshot-contract`.
- Required gates: `check-state-snapshot-contract.*`.
- Replay fixtures: `state_session_snapshot.v1`.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P1`
- `verification.semantic.anchor=snapshot.export_restore_replay`
- `verification.semantic.classification=state.session_snapshot_recovery`
- `verification.semantic.runtime_path=core/runner,tool/local,runtime/config,orchestration/snapshot,runtime/diagnostics`
- `verification.semantic.governance=enforced`
- `verification.semantic.expected_markers=snapshot_export_emitted,snapshot_restore_verified,snapshot_replay_idempotent,governance_snapshot_gate_enforced,governance_snapshot_replay_bound`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`
- `result.final_answer` 包含 `governance/ticket/replay_binding`，且签名必须与 minimal 不同。

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If governance result is unexpected,检查 `replay_drift` 与 gate 决策是否一致。
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.
