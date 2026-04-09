# rag-hybrid-retrieval (production-ish)

## Purpose
hybrid retrieval with memory primary, mcp fallback, and fallback classification output.

## Run
go run ./examples/agent-modes/rag-hybrid-retrieval/production-ish

## Prerequisites
- Go 1.22+ and module dependencies resolved (go mod tidy).
- Writable local cache for Go build artifacts (for deterministic smoke runs).
- No external network service is required; execution is fully local.

## Real Runtime Path
- core/runner: executes model/tool loop and returns final run result.
- tool/local: dispatches local.mode_step deterministic tool calls.
- runtime/config: runtime manager wiring for policy/config runtime path.

## Contract Mapping
- contracts: `memory-scope-and-builtin-filesystem-v2-governance-contract`
- gates: `check-memory-scope-and-search-contract.*`
- replay: `memory_scope.v1`

## Diagnostics And Tracing Signals
- diagnostics marker: `agent_mode.rag_hybrid_retrieval.production_ish`
- tracing marker: `agent_mode.rag_hybrid_retrieval.production_ish`

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

