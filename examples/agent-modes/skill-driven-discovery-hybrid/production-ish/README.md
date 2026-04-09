# skill-driven-discovery-hybrid (production-ish)

## Purpose
Real runtime semantic example for `skill-driven-discovery-hybrid` with `production-ish` evidence profile.

## Run
go run ./examples/agent-modes/skill-driven-discovery-hybrid/production-ish

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `discovery.source_priority_score_mapping`.
- Classification: `skill.hybrid_discovery`.
- Runtime path evidence: `core/runner,tool/local,runtime/config,skill/loader,context/assembler`.
- Related contracts: `skill-trigger-scoring`.
- Required gates: `check-react-contract.*`.
- Replay fixtures: `react.v1`.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P0`
- `verification.semantic.anchor=discovery.source_priority_score_mapping`
- `verification.semantic.classification=skill.hybrid_discovery`
- `verification.semantic.runtime_path=core/runner,tool/local,runtime/config,skill/loader,context/assembler`
- `verification.semantic.governance=enforced`
- `verification.semantic.expected_markers=discovery_sources_prioritized,discovery_score_reconciled,discovery_mapping_emitted,governance_skill_gate_enforced,governance_skill_replay_bound`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.
