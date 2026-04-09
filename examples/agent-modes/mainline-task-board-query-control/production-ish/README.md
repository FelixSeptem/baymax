# mainline-task-board-query-control (production-ish)

## Purpose
Real runtime semantic example for `mainline-task-board-query-control` with `production-ish` evidence profile.

## Run
go run ./examples/agent-modes/mainline-task-board-query-control/production-ish

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `taskboard.query_control_idempotency`.
- Classification: `mainline.taskboard_control`.
- Runtime path evidence: `core/runner,tool/local,runtime/config,orchestration/scheduler,runtime/diagnostics`.
- Related contracts: `multi-agent-task-board-control-contract`.
- Required gates: `check-multi-agent-shared-contract.*`.
- Replay fixtures: `cross-domain-primary-reason-arbitration-contract.v1`.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P2`
- `verification.semantic.anchor=taskboard.query_control_idempotency`
- `verification.semantic.classification=mainline.taskboard_control`
- `verification.semantic.runtime_path=core/runner,tool/local,runtime/config,orchestration/scheduler,runtime/diagnostics`
- `verification.semantic.governance=enforced`
- `verification.semantic.expected_markers=taskboard_query_filtered,taskboard_control_validated,taskboard_operation_idempotent,governance_taskboard_gate_enforced,governance_taskboard_replay_bound`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.
