# multi-agents-collab-recovery (minimal)

## Purpose
coordinator plus worker collaboration lifecycle with mailbox sync.

## Run
go run ./examples/agent-modes/multi-agents-collab-recovery/minimal

## Prerequisites
- Go 1.22+ and module dependencies resolved (go mod tidy).
- Writable local cache for Go build artifacts (for deterministic smoke runs).
- No external network service is required; execution is fully local.

## Real Runtime Path
- core/runner: executes model/tool loop and returns final run result.
- tool/local: dispatches local.mode_step deterministic tool calls.
- runtime/config: runtime manager wiring for policy/config runtime path.

## Contract Mapping
- contracts: `multi-agent-collaboration-primitives` + `long-running-recovery-boundary`
- gates: `check-multi-agent-shared-contract.*`
- replay: `cross-domain primary reason arbitration contract.v1`

## Diagnostics And Tracing Signals
- diagnostics marker: `agent_mode.multi_agents_collab_recovery.minimal`
- tracing marker: `agent_mode.multi_agents_collab_recovery.minimal`

## Expected Output/Verification
- Output must include verification.mainline_runtime_path=ok.
- Output must include result.final_answer= and result.signature= markers.
- Verify with smoke gate: pwsh -File scripts/check-agent-mode-examples-smoke.ps1.

