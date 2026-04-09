# workflow-branch-retry-failfast (minimal)

## Purpose
branch execution with deterministic retry budget and fail-fast terminal.

## Run
go run ./examples/agent-modes/workflow-branch-retry-failfast/minimal

## Prerequisites
- Go 1.22+ and module dependencies resolved (go mod tidy).
- Writable local cache for Go build artifacts (for deterministic smoke runs).
- No external network service is required; execution is fully local.

## Real Runtime Path
- core/runner: executes model/tool loop and returns final run result.
- tool/local: dispatches local.mode_step deterministic tool calls.
- runtime/config: runtime manager wiring for policy/config runtime path.

## Contract Mapping
- contracts: `workflow-graph-composability-contract`
- gates: `check-multi-agent-shared-contract.*`
- replay: `cross-domain primary reason arbitration contract.v1`

## Diagnostics And Tracing Signals
- diagnostics marker: `agent_mode.workflow_branch_retry_failfast.minimal`
- tracing marker: `agent_mode.workflow_branch_retry_failfast.minimal`

## Expected Output/Verification
- Output must include verification.mainline_runtime_path=ok.
- Output must include result.final_answer= and result.signature= markers.
- Verify with smoke gate: pwsh -File scripts/check-agent-mode-examples-smoke.ps1.

