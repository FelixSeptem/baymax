# state-session-snapshot-recovery (minimal)

## Purpose
用真实语义链路演示 `state-session-snapshot-recovery` 的最小闭环：快照导出、恢复校验、回放幂等判定。

## Run
go run ./examples/agent-modes/state-session-snapshot-recovery/minimal

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `snapshot.export_restore_replay`.
- Classification: `state.session_snapshot_recovery`.
- Runtime path evidence: `core/runner,tool/local,runtime/config,orchestration/snapshot,runtime/diagnostics`.
- Semantic flow:
  - `snapshot_export_emitted`: 按 chunk 导出会话快照并产出 `snapshot_digest`。
  - `snapshot_restore_verified`: 恢复快照并校验 checksum / cursor。
  - `snapshot_replay_idempotent`: 回放事件帧并给出幂等与漂移结论。
- Related contracts: `unified-state-and-session-snapshot-contract`.
- Required gates: `check-state-snapshot-contract.*`.
- Replay fixtures: `state_session_snapshot.v1`.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P1`
- `verification.semantic.anchor=snapshot.export_restore_replay`
- `verification.semantic.classification=state.session_snapshot_recovery`
- `verification.semantic.runtime_path=core/runner,tool/local,runtime/config,orchestration/snapshot,runtime/diagnostics`
- `verification.semantic.governance=baseline`
- `verification.semantic.expected_markers=snapshot_export_emitted,snapshot_restore_verified,snapshot_replay_idempotent`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`
- `result.final_answer` 包含 `snapshot_version/chunks/digest/replay_idempotent/replay_drift` 等状态字段。

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If replay result is unexpected,优先核对 `restore_checksum_ok` 与 `replay_idempotent` 是否一致。
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.
