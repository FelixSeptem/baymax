# policy-budget-admission (production-ish)

## Purpose
用真实语义链路演示 `policy-budget-admission` 的生产治理闭环：在最小链路上增加策略门控和 replay 绑定。

## Variant Delta (vs minimal)
- 生产输入会触发更高优先级策略（如 `deny_sensitive_model`），行为分歧来自真实仲裁结果。
- 在 admission 之后增加治理 gate，输出 `allow / allow_with_limit / deny`。
- 追加 replay 绑定，确保策略决策可审计复放。

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
- Semantic flow:
  - minimal 的 3 步仲裁链路；
  - 追加 `governance_policy_gate_enforced` 与 `governance_policy_replay_bound` 两步治理链路。
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
- `result.final_answer` 包含 `governance/ticket/replay`，且签名必须与 minimal 不同。

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If governance result is unexpected,检查 `admission_decision` 与 gate 决策是否匹配。
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.
