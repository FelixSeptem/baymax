# policy-budget-admission (production-ish)

## Purpose
Real runtime semantic example for `policy-budget-admission` with `production-ish` evidence profile.

## Run
go run ./examples/agent-modes/policy-budget-admission/production-ish

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `policy.precedence_budget_admission_trace`.
- Classification: `policy.budget_admission`.
- Runtime path evidence: `core/runner,tool/local,runtime/config,runtime/diagnostics`.
- Related contracts: `policy-precedence-and-decision-trace-contract; runtime-cost-latency-budget-and-admission-contract`.
- Required gates: `check-policy-precedence-contract.*; check-runtime-budget-admission-contract.*`.
- Replay fixtures: `policy_stack.v1; budget_admission.v1`.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P1`
- `verification.semantic.anchor=policy.precedence_budget_admission_trace`
- `verification.semantic.classification=policy.budget_admission`
- `verification.semantic.runtime_path=core/runner,tool/local,runtime/config,runtime/diagnostics`
- `verification.semantic.governance=enforced`
- `verification.semantic.expected_markers=policy_precedence_applied,budget_admission_decided,decision_trace_recorded,governance_policy_gate_enforced,governance_policy_replay_bound`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.
