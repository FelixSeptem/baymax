# rag-hybrid-retrieval (production-ish)

## Purpose
Real runtime semantic example for `rag-hybrid-retrieval` with `production-ish` evidence profile.

## Variant Delta (vs minimal)
- Reuses the same semantic anchor and runtime path baseline as minimal.
- Adds `governance_retrieval_budget_gate`: apply candidate budget limit before answer synthesis.
- Adds `governance_retrieval_replay_bound`: emit replay signature derived from ranked IDs + fallback route.
- Requires verification.semantic.governance=enforced.
- Requires verification.semantic.expected_markers and result.signature to differ from minimal.

## Run
go run ./examples/agent-modes/rag-hybrid-retrieval/production-ish

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `retrieval.candidate_rerank_fallback`.
- Classification: `rag.hybrid_retrieval`.
- Runtime path evidence: `core/runner,tool/local,runtime/config,memory,context/assembler`.
- Related contracts: `memory-scope-and-builtin-filesystem-v2-governance-contract`.
- Required gates: `check-memory-scope-and-search-contract.*`.
- Replay fixtures: `memory_scope.v1`.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P0`
- `verification.semantic.anchor=retrieval.candidate_rerank_fallback`
- `verification.semantic.classification=rag.hybrid_retrieval`
- `verification.semantic.runtime_path=core/runner,tool/local,runtime/config,memory,context/assembler`
- `verification.semantic.governance=enforced`
- `verification.semantic.expected_markers=retrieval_candidates_built,retrieval_rerank_applied,retrieval_fallback_classified,governance_retrieval_budget_gate,governance_retrieval_replay_bound`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If governance markers are missing, verify budget/replay branches in `semantic_example.go`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.


