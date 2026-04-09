# react-plan-notebook-loop (minimal)

## Purpose
react loop with plan-notebook synchronization in local run path.

## Run
go run ./examples/agent-modes/react-plan-notebook-loop/minimal

## Prerequisites
- Go 1.22+ and module dependencies resolved (go mod tidy).
- Writable local cache for Go build artifacts (for deterministic smoke runs).
- No external network service is required; execution is fully local.

## Real Runtime Path
- core/runner: executes model/tool loop and returns final run result.
- tool/local: dispatches local.mode_step deterministic tool calls.
- runtime/config: runtime manager wiring for policy/config runtime path.

## Contract Mapping
- contracts: `react-plan-notebook-and-plan-change-hook-contract`
- gates: `check-react-plan-notebook-contract.*`
- replay: `react_plan_notebook.v1`

## Diagnostics And Tracing Signals
- diagnostics marker: `agent_mode.react_plan_notebook_loop.minimal`
- tracing marker: `agent_mode.react_plan_notebook_loop.minimal`

## Expected Output/Verification
- Output must include verification.mainline_runtime_path=ok.
- Output must include result.final_answer= and result.signature= markers.
- Verify with smoke gate: pwsh -File scripts/check-agent-mode-examples-smoke.ps1.

