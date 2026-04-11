# hitl-governed-checkpoint (production-ish)

## Purpose
Real runtime semantic example for `hitl-governed-checkpoint` with `production-ish` evidence profile.

## Variant Delta (vs minimal)
- Reuses the same semantic anchor and runtime path baseline as minimal.
- Adds `governance_hitl_gate_enforced`: classify `allow|allow_with_record|allow_with_recovery|block` from decision + timeout + recovery plan.
- Adds `governance_hitl_replay_bound`: bind replay signature from ticket version and governance result.
- Preserves minimal checkpoint flow and appends governance enforcement.
- Requires verification.semantic.governance=enforced.
- Requires verification.semantic.expected_markers and result.signature to differ from minimal.

## Run
go run ./examples/agent-modes/hitl-governed-checkpoint/production-ish

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `hitl.await_resume_reject_timeout_recover`.
- Classification: `hitl.checkpoint_governance`.
- Runtime path evidence: `core/runner,tool/local,runtime/config,orchestration/composer,runtime/diagnostics`.
- Related contracts: `react-loop-and-tool-calling-parity-contract`.
- Required gates: `check-react-contract.*`.
- Replay fixtures: `react.v1`.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P0`
- `verification.semantic.anchor=hitl.await_resume_reject_timeout_recover`
- `verification.semantic.classification=hitl.checkpoint_governance`
- `verification.semantic.runtime_path=core/runner,tool/local,runtime/config,orchestration/composer,runtime/diagnostics`
- `verification.semantic.governance=enforced`
- `verification.semantic.expected_markers=hitl_checkpoint_awaited,hitl_resume_reject_classified,hitl_timeout_recoverable,governance_hitl_gate_enforced,governance_hitl_replay_bound`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If governance or replay output is unexpected, inspect `governance_hitl_*` branches in `semantic_example.go`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.


