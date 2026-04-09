# adapter-onboarding-manifest-capability (minimal)

## Purpose
adapter manifest parse and capability declaration validation path.

## Run
go run ./examples/agent-modes/adapter-onboarding-manifest-capability/minimal

## Prerequisites
- Go 1.22+ and module dependencies resolved (go mod tidy).
- Writable local cache for Go build artifacts (for deterministic smoke runs).
- No external network service is required; execution is fully local.

## Real Runtime Path
- core/runner: executes model/tool loop and returns final run result.
- tool/local: dispatches local.mode_step deterministic tool calls.
- runtime/config: runtime manager wiring for policy/config runtime path.

## Contract Mapping
- contracts: `adapter-manifest-and-runtime-compatibility` + `adapter-capability-negotiation-and-fallback` + `adapter-contract-profile-versioning-and-replay`
- gates: `check-adapter-manifest-contract.*` + `check-adapter-capability-contract.*` + `check-adapter-contract-replay.*`
- replay: `adapter_contract_profile.v1`

## Diagnostics And Tracing Signals
- diagnostics marker: `agent_mode.adapter_onboarding_manifest_capability.minimal`
- tracing marker: `agent_mode.adapter_onboarding_manifest_capability.minimal`

## Expected Output/Verification
- Output must include verification.mainline_runtime_path=ok.
- Output must include result.final_answer= and result.signature= markers.
- Verify with smoke gate: pwsh -File scripts/check-agent-mode-examples-smoke.ps1.

