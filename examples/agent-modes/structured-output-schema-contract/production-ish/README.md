# structured-output-schema-contract (production-ish)

## Purpose
Real runtime semantic example for `structured-output-schema-contract` with `production-ish` evidence profile.

## Run
go run ./examples/agent-modes/structured-output-schema-contract/production-ish

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `schema.validate_compat_drift`.
- Classification: `structured_output.schema_contract`.
- Runtime path evidence: `core/runner,tool/local,runtime/config,core/types,runtime/diagnostics`.
- Related contracts: `diagnostics-replay-tooling`.
- Required gates: `check-diagnostics-replay-contract.*`.
- Replay fixtures: `cross-domain-primary-reason-arbitration-contract.v1`.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P0`
- `verification.semantic.anchor=schema.validate_compat_drift`
- `verification.semantic.classification=structured_output.schema_contract`
- `verification.semantic.runtime_path=core/runner,tool/local,runtime/config,core/types,runtime/diagnostics`
- `verification.semantic.governance=enforced`
- `verification.semantic.expected_markers=schema_contract_loaded,schema_compat_window_checked,schema_drift_signal_emitted,governance_schema_gate_enforced,governance_schema_replay_bound`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.
