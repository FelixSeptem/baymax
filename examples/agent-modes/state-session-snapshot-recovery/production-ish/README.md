# state-session-snapshot-recovery (production-ish)

## Purpose
Real runtime semantic example for `state-session-snapshot-recovery` with `production-ish` evidence profile.

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

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.
