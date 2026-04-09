# context-governed-reference-first (minimal)

## Purpose
reference-first assembly path with bounded context selection.

## Run
go run ./examples/agent-modes/context-governed-reference-first/minimal

## Prerequisites
- Go 1.22+ and module dependencies resolved (go mod tidy).
- Writable local cache for Go build artifacts (for deterministic smoke runs).
- No external network service is required; execution is fully local.

## Real Runtime Path
- core/runner: executes model/tool loop and returns final run result.
- tool/local: dispatches local.mode_step deterministic tool calls.
- runtime/config: runtime manager wiring for policy/config runtime path.

## Contract Mapping
- contracts: `jit-context-organization-and-reference-first-assembly-contract` + `context-compression-production-hardening-contract`
- gates: `check-context-jit-organization-contract.*` + `check-context-compression-production-contract.*`
- replay: `context_reference_first.v1` + `context_compression_production.v1`

## Diagnostics And Tracing Signals
- diagnostics marker: `agent_mode.context_governed_reference_first.minimal`
- tracing marker: `agent_mode.context_governed_reference_first.minimal`

## Expected Output/Verification
- Output must include verification.mainline_runtime_path=ok.
- Output must include result.final_answer= and result.signature= markers.
- Verify with smoke gate: pwsh -File scripts/check-agent-mode-examples-smoke.ps1.

