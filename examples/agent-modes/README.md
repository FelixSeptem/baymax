# Agent Modes Example Pack

## Purpose
runtime demonstration for examples in agent-modes mode.

## Run
go run ./examples/agent-modes/examples/agent-modes

## Prerequisites
- Go 1.22+ and module dependencies resolved (go mod tidy).
- Writable local cache for Go build artifacts (for deterministic smoke runs).
- No external network service is required; execution is fully local.

## Real Runtime Path
- core/runner: executes model/tool loop and returns final run result.
- tool/local: dispatches local.mode_step deterministic tool calls.
- runtime/config: runtime manager wiring for policy/config runtime path.

## Contract Mapping
- contracts: see examples/agent-modes/MATRIX.md
- gates: see examples/agent-modes/MATRIX.md
- replay: see examples/agent-modes/MATRIX.md

## Diagnostics And Tracing Signals
- diagnostics marker: agent_mode.examples.agent_modes
- tracing marker: agent_mode.examples.agent_modes

## Expected Output/Verification
- Output must include verification.mainline_runtime_path=ok.
- Output must include result.final_answer= and result.signature= markers.
- Verify with smoke gate: pwsh -File scripts/check-agent-mode-examples-smoke.ps1.

