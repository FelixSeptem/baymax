# react-plan-notebook-loop (production-ish)

## Purpose
Real runtime semantic example for `react-plan-notebook-loop` with `production-ish` evidence profile.

## Run
go run ./examples/agent-modes/react-plan-notebook-loop/production-ish

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `react.plan_notebook_change_hooks`.
- Classification: `react.plan_notebook_loop`.
- Runtime path evidence: `core/runner,tool/local,runtime/config,runtime/diagnostics`.
- Related contracts: `react-plan-notebook-and-plan-change-hook-contract`.
- Required gates: `check-react-plan-notebook-contract.*`.
- Replay fixtures: `react_plan_notebook.v1`.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P1`
- `verification.semantic.anchor=react.plan_notebook_change_hooks`
- `verification.semantic.classification=react.plan_notebook_loop`
- `verification.semantic.runtime_path=core/runner,tool/local,runtime/config,runtime/diagnostics`
- `verification.semantic.governance=enforced`
- `verification.semantic.expected_markers=react_plan_notebook_synced,react_change_hook_emitted,react_tool_loop_closed,governance_react_gate_enforced,governance_react_replay_bound`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.
