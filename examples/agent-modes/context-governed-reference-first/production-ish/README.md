# context-governed-reference-first (production-ish)

## Purpose
Real runtime semantic example for `context-governed-reference-first` with `production-ish` evidence profile.

## Run
go run ./examples/agent-modes/context-governed-reference-first/production-ish

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `context.reference_first_isolate_edit_tiering`.
- Classification: `context.reference_first_governance`.
- Runtime path evidence: `core/runner,tool/local,runtime/config,context/assembler,context/guard,context/journal`.
- Related contracts: `jit-context-organization-and-reference-first-assembly-contract; context-compression-production-hardening-contract`.
- Required gates: `check-context-jit-organization-contract.*; check-context-compression-production-contract.*`.
- Replay fixtures: `context_reference_first.v1; context_compression_production.v1`.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P0`
- `verification.semantic.anchor=context.reference_first_isolate_edit_tiering`
- `verification.semantic.classification=context.reference_first_governance`
- `verification.semantic.runtime_path=core/runner,tool/local,runtime/config,context/assembler,context/guard,context/journal`
- `verification.semantic.governance=enforced`
- `verification.semantic.expected_markers=context_reference_first_selected,context_isolate_handoff_applied,context_edit_gate_evaluated,governance_context_tiering_enforced,governance_context_replay_bound`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.
