# mainline-readiness-admission-degradation (production-ish)

## Purpose
readiness plus admission degradation path with strict classification and rollback-safe behavior.

## Run
go run ./examples/agent-modes/mainline-readiness-admission-degradation/production-ish

## Prerequisites
- Go 1.22+ and module dependencies resolved (go mod tidy).
- Writable local cache for Go build artifacts (for deterministic smoke runs).
- No external network service is required; execution is fully local.

## Real Runtime Path
- core/runner: executes model/tool loop and returns final run result.
- tool/local: dispatches local.mode_step deterministic tool calls.
- runtime/config: runtime manager wiring for policy/config runtime path.

## Contract Mapping
- contracts: `runtime-readiness-preflight-contract` + `runtime-readiness-admission-guard-contract`
- gates: `check-quality-gate.*`
- replay: `readiness-timeout-health replay fixture gate.v1`

## Diagnostics And Tracing Signals
- diagnostics marker: `agent_mode.mainline_readiness_admission_degradation.production_ish`
- tracing marker: `agent_mode.mainline_readiness_admission_degradation.production_ish`

## Expected Output/Verification
- Output must include verification.mainline_runtime_path=ok.
- Output must include result.final_answer= and result.signature= markers.
- Verify with smoke gate: pwsh -File scripts/check-agent-mode-examples-smoke.ps1.

## Prod Delta Checklist
- [ ] config: validate env > file > default precedence and invalid reload rollback path.
- [ ] permissions: confirm sandbox and allowlist policy matrix for target environment.
- [ ] capacity: define concurrency, retry budget, and timeout envelope for sustained load.
- [ ] observability: wire diagnostics and tracing export into runtime recorder pathways.
- [ ] replay: maintain replay fixture compatibility and additive parser expectations.
- [ ] gates: run mapped contract gates plus quality gate path before promotion.

