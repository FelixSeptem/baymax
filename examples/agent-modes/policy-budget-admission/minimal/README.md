# policy-budget-admission (minimal)

## Purpose
policy precedence evaluation with budget admission allow/deny.

## Run
go run ./examples/agent-modes/policy-budget-admission/minimal

## Prerequisites
- Go 1.22+ and module dependencies resolved (go mod tidy).
- Writable local cache for Go build artifacts (for deterministic smoke runs).
- No external network service is required; execution is fully local.

## Real Runtime Path
- core/runner: executes model/tool loop and returns final run result.
- tool/local: dispatches local.mode_step deterministic tool calls.
- runtime/config: runtime manager wiring for policy/config runtime path.

## Contract Mapping
- contracts: `policy-precedence-and-decision-trace-contract` + `runtime-cost-latency-budget-and-admission-contract`
- gates: `check-policy-precedence-contract.*` + `check-runtime-budget-admission-contract.*`
- replay: `policy_stack.v1` + `budget_admission.v1`

## Diagnostics And Tracing Signals
- diagnostics marker: `agent_mode.policy_budget_admission.minimal`
- tracing marker: `agent_mode.policy_budget_admission.minimal`

## Expected Output/Verification
- Output must include verification.mainline_runtime_path=ok.
- Output must include result.final_answer= and result.signature= markers.
- Verify with smoke gate: pwsh -File scripts/check-agent-mode-examples-smoke.ps1.

