# workflow-branch-retry-failfast (production-ish)

## Purpose
Real runtime semantic example for `workflow-branch-retry-failfast` with `production-ish` evidence profile.

## Run
go run ./examples/agent-modes/workflow-branch-retry-failfast/production-ish

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `workflow.branch_retry_failfast`.
- Classification: `workflow.retry_failfast`.
- Runtime path evidence: `core/runner,tool/local,runtime/config,orchestration/workflow`.
- Related contracts: `workflow-graph-composability-contract`.
- Required gates: `check-multi-agent-shared-contract.*`.
- Replay fixtures: `cross-domain-primary-reason-arbitration-contract.v1`.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P1`
- `verification.semantic.anchor=workflow.branch_retry_failfast`
- `verification.semantic.classification=workflow.retry_failfast`
- `verification.semantic.runtime_path=core/runner,tool/local,runtime/config,orchestration/workflow`
- `verification.semantic.governance=enforced`
- `verification.semantic.expected_markers=workflow_branch_routed,workflow_retry_budgeted,workflow_failfast_classified,governance_workflow_gate_enforced,governance_workflow_replay_bound`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.
