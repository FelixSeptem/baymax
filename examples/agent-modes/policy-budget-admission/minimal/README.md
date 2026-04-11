# policy-budget-admission (minimal)

## Purpose
用真实语义链路演示 `policy-budget-admission` 的最小闭环：策略优先级仲裁、预算准入决策、决策追踪落盘。

## Run
go run ./examples/agent-modes/policy-budget-admission/minimal

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `policy.precedence_budget_admission_trace`.
- Classification: `policy.budget_admission`.
- Runtime path evidence: `core/runner,tool/local,runtime/config,runtime/diagnostics`.
- Semantic flow:
  - `policy_precedence_applied`: 按优先级栈选出 winning policy。
  - `budget_admission_decided`: 结合预算余量与 projected cost 给出 `admit/defer/reject`。
  - `decision_trace_recorded`: 输出可复放的 `trace_id/trace_hash`。
- Related contracts: `policy-precedence-and-decision-trace-contract; runtime-cost-latency-budget-and-admission-contract`.
- Required gates: `check-policy-precedence-contract.*; check-runtime-budget-admission-contract.*`.
- Replay fixtures: `policy_stack.v1; budget_admission.v1`.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P1`
- `verification.semantic.anchor=policy.precedence_budget_admission_trace`
- `verification.semantic.classification=policy.budget_admission`
- `verification.semantic.runtime_path=core/runner,tool/local,runtime/config,runtime/diagnostics`
- `verification.semantic.governance=baseline`
- `verification.semantic.expected_markers=policy_precedence_applied,budget_admission_decided,decision_trace_recorded`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`
- `result.final_answer` 包含 `policy/precedence/budget/admission/trace` 等真实仲裁字段。

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If admission result is unexpected,优先核对 `winning_policy` 与 `budget_headroom` 的一致性。
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.
