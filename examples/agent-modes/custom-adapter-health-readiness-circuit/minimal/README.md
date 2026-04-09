# custom-adapter-health-readiness-circuit (minimal)

## Purpose
health probe and readiness baseline for custom adapter.

## Run
go run ./examples/agent-modes/custom-adapter-health-readiness-circuit/minimal

## Prerequisites
- Go 1.22+ and module dependencies resolved (go mod tidy).
- Writable local cache for Go build artifacts (for deterministic smoke runs).
- No external network service is required; execution is fully local.

## Real Runtime Path
- core/runner: executes model/tool loop and returns final run result.
- tool/local: dispatches local.mode_step deterministic tool calls.
- runtime/config: runtime manager wiring for policy/config runtime path.

## Contract Mapping
- contracts: `adapter-runtime-health-probe-contract` + `adapter-health-backoff-and-circuit-governance-contract`
- gates: `check-adapter-conformance.*`
- replay: `readiness-timeout-health replay fixture gate.v1`

## Diagnostics And Tracing Signals
- diagnostics marker: `agent_mode.custom_adapter_health_readiness_circuit.minimal`
- tracing marker: `agent_mode.custom_adapter_health_readiness_circuit.minimal`

## Expected Output/Verification
- Output must include verification.mainline_runtime_path=ok.
- Output must include result.final_answer= and result.signature= markers.
- Verify with smoke gate: pwsh -File scripts/check-agent-mode-examples-smoke.ps1.

