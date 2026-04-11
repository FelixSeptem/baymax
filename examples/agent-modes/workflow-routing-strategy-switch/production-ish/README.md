# workflow-routing-strategy-switch (production-ish)

## Purpose
Real runtime semantic example for `workflow-routing-strategy-switch` with `production-ish` evidence profile.

## Variant Delta (vs minimal)
- Reuses the same semantic anchor and runtime path baseline as minimal.
- Uses stricter switch threshold and different selection policy for spiky mixed traffic profile.
- Adds governance branch (`governance_routing_gate_enforced`, `governance_routing_replay_bound`) that may enforce canary-based decision.
- Requires verification.semantic.governance=enforced and a different `result.signature` from minimal.

## Run
go run ./examples/agent-modes/workflow-routing-strategy-switch/production-ish

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `routing.strategy_switch_confidence`.
- Classification: `workflow.strategy_switch`.
- Runtime path evidence: `core/runner,tool/local,runtime/config,orchestration/workflow`.
- Related contracts: `workflow-graph-composability-contract`.
- Required gates: `check-multi-agent-shared-contract.*`.
- Replay fixtures: `cross-domain-primary-reason-arbitration-contract.v1`.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P2`
- `verification.semantic.anchor=routing.strategy_switch_confidence`
- `verification.semantic.classification=workflow.strategy_switch`
- `verification.semantic.runtime_path=core/runner,tool/local,runtime/config,orchestration/workflow`
- `verification.semantic.governance=enforced`
- `verification.semantic.expected_markers=routing_strategy_selected,routing_confidence_evaluated,routing_switch_committed,governance_routing_gate_enforced,governance_routing_replay_bound`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`
- `result.final_answer` includes governance fields: `governance`, `ticket`, `replay`.

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If governance decision/replay fields are missing, inspect marker handlers for `governance_routing_gate_enforced` and `governance_routing_replay_bound`.
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.


