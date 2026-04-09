# config-hot-reload-rollback (production-ish)

## Purpose
Real runtime semantic example for `config-hot-reload-rollback` with `production-ish` evidence profile.

## Run
go run ./examples/agent-modes/config-hot-reload-rollback/production-ish

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `config.reload_failfast_rollback`.
- Classification: `runtime.config_rollback`.
- Runtime path evidence: `core/runner,tool/local,runtime/config,runtime/diagnostics`.
- Related contracts: `runtime-config-and-diagnostics-api`.
- Required gates: `check-quality-gate.*`.
- Replay fixtures: `cross-domain-primary-reason-arbitration-contract.v1`.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P2`
- `verification.semantic.anchor=config.reload_failfast_rollback`
- `verification.semantic.classification=runtime.config_rollback`
- `verification.semantic.runtime_path=core/runner,tool/local,runtime/config,runtime/diagnostics`
- `verification.semantic.governance=enforced`
- `verification.semantic.expected_markers=config_reload_attempted,config_invalid_failfast,config_atomic_rollback_verified,governance_config_gate_enforced,governance_config_replay_bound`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.
