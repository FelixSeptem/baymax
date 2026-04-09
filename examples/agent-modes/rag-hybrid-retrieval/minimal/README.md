# rag-hybrid-retrieval (minimal)

## Purpose
memory retrieval with deterministic local corpus filtering.

## Run
go run ./examples/agent-modes/rag-hybrid-retrieval/minimal

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
- diagnostics marker: `agent_mode.rag_hybrid_retrieval.minimal`
- tracing marker: `agent_mode.rag_hybrid_retrieval.minimal`

## Expected Output/Verification
- Output must include verification.mainline_runtime_path=ok.
- Output must include result.final_answer= and result.signature= markers.
- Verify with smoke gate: pwsh -File scripts/check-agent-mode-examples-smoke.ps1.

