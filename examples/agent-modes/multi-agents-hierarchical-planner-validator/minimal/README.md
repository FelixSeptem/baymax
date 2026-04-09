# multi-agents-hierarchical-planner-validator (minimal)

## Purpose
Real runtime semantic example for `multi-agents-hierarchical-planner-validator` with `minimal` evidence profile.

## Run
go run ./examples/agent-modes/multi-agents-hierarchical-planner-validator/minimal

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `hierarchy.planner_validator_correction`.
- Classification: `multi_agents.hierarchy`.
- Runtime path evidence: `core/runner,tool/local,runtime/config,orchestration/teams,orchestration/workflow`.
- Related contracts: `multi-agent-collaboration-primitives`.
- Required gates: `check-multi-agent-shared-contract.*`.
- Replay fixtures: `cross-domain-primary-reason-arbitration-contract.v1`.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P2`
- `verification.semantic.anchor=hierarchy.planner_validator_correction`
- `verification.semantic.classification=multi_agents.hierarchy`
- `verification.semantic.runtime_path=core/runner,tool/local,runtime/config,orchestration/teams,orchestration/workflow`
- `verification.semantic.governance=baseline`
- `verification.semantic.expected_markers=hierarchy_plan_decomposed,hierarchy_validator_feedback_applied,hierarchy_correction_loop_closed`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.
