# mcp-governed-stdio-http (minimal)

## Purpose
single transport execution path under deterministic transport selection.

## Run
go run ./examples/agent-modes/mcp-governed-stdio-http/minimal

## Prerequisites
- Go 1.22+ and module dependencies resolved (go mod tidy).
- Writable local cache for Go build artifacts (for deterministic smoke runs).
- No external network service is required; execution is fully local.

## Real Runtime Path
- core/runner: executes model/tool loop and returns final run result.
- tool/local: dispatches local.mode_step deterministic tool calls.
- runtime/config: runtime manager wiring for policy/config runtime path.

## Contract Mapping
- contracts: `mcp-runtime-reliability-profiles`
- gates: `check-quality-gate.*`
- replay: `cross-domain primary reason arbitration contract.v1`

## Diagnostics And Tracing Signals
- diagnostics marker: `agent_mode.mcp_governed_stdio_http.minimal`
- tracing marker: `agent_mode.mcp_governed_stdio_http.minimal`

## Expected Output/Verification
- Output must include verification.mainline_runtime_path=ok.
- Output must include result.final_answer= and result.signature= markers.
- Verify with smoke gate: pwsh -File scripts/check-agent-mode-examples-smoke.ps1.

