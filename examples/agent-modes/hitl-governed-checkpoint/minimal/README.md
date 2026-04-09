# hitl-governed-checkpoint (minimal)

## Purpose
checkpoint await and explicit resume path with deterministic state.

## Run
go run ./examples/agent-modes/hitl-governed-checkpoint/minimal

## Prerequisites
- Go 1.22+ and module dependencies resolved (go mod tidy).
- Writable local cache for Go build artifacts (for deterministic smoke runs).
- No external network service is required; execution is fully local.

## Real Runtime Path
- core/runner: executes model/tool loop and returns final run result.
- tool/local: dispatches local.mode_step deterministic tool calls.
- runtime/config: runtime manager wiring for policy/config runtime path.

## Contract Mapping
- contracts: `react-loop-and-tool-calling-parity-contract`
- gates: `check-react-contract.*`
- replay: `react.v1`

## Diagnostics And Tracing Signals
- diagnostics marker: `agent_mode.hitl_governed_checkpoint.minimal`
- tracing marker: `agent_mode.hitl_governed_checkpoint.minimal`

## Expected Output/Verification
- Output must include verification.mainline_runtime_path=ok.
- Output must include result.final_answer= and result.signature= markers.
- Verify with smoke gate: pwsh -File scripts/check-agent-mode-examples-smoke.ps1.

