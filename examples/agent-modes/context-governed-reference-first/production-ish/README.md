# context-governed-reference-first (production-ish)

## Purpose
reference-first, isolate handoff, edit gate, and tiering path aligned to context compression convergence evidence.

## Run
go run ./examples/agent-modes/context-governed-reference-first/production-ish

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
- diagnostics marker: `agent_mode.context_governed_reference_first.production_ish`
- tracing marker: `agent_mode.context_governed_reference_first.production_ish`

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
- [ ] context compression dependency: confirm production outputs are green before completion.

