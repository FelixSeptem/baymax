# multi-agents-hierarchical-planner-validator (production-ish)

## Purpose
Real runtime semantic example for `multi-agents-hierarchical-planner-validator` with `production-ish` evidence profile.

## Variant Delta (vs minimal)
- Reuses the same semantic anchor and runtime path baseline as minimal.
- Uses larger decomposition scope and additional validator/correction rounds before closing the loop.
- Adds governance branch (`governance_hierarchy_gate_enforced`, `governance_hierarchy_replay_bound`) to enforce execution decision and replay trace.
- Requires verification.semantic.governance=enforced and a different `result.signature` from minimal.

## Run
go run ./examples/agent-modes/multi-agents-hierarchical-planner-validator/production-ish

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
- `verification.semantic.governance=enforced`
- `verification.semantic.expected_markers=hierarchy_plan_decomposed,hierarchy_validator_feedback_applied,hierarchy_correction_loop_closed,governance_hierarchy_gate_enforced,governance_hierarchy_replay_bound`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`
- `result.final_answer` includes governance fields: `governance`, `ticket`, `replay`.

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If governance fields are missing, inspect marker handlers for `governance_hierarchy_gate_enforced` and `governance_hierarchy_replay_bound`.
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.


