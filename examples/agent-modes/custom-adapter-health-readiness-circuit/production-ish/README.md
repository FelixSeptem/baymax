# custom-adapter-health-readiness-circuit (production-ish)

## Purpose
Real runtime semantic example for `custom-adapter-health-readiness-circuit` with `production-ish` evidence profile.

## Variant Delta (vs minimal)
- Reuses the same semantic anchor and runtime path baseline as minimal.
- Uses degraded health probe signals and circuit-guarded recovery/backoff path.
- Adds governance branch (`governance_adapter_health_gate_enforced`, `governance_adapter_health_replay_bound`) for health gate decision and replay trace.
- Requires verification.semantic.governance=enforced and a different `result.signature` from minimal.

## Run
go run ./examples/agent-modes/custom-adapter-health-readiness-circuit/production-ish

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `adapterhealth.readiness_backoff_circuit`.
- Classification: `adapter.health_readiness`.
- Runtime path evidence: `core/runner,tool/local,runtime/config,adapter/health`.
- Related contracts: `adapter-runtime-health-probe-contract; adapter-health-backoff-and-circuit-governance-contract`.
- Required gates: `check-adapter-conformance.*`.
- Replay fixtures: `readiness-timeout-health-replay-fixture-gate.v1`.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P2`
- `verification.semantic.anchor=adapterhealth.readiness_backoff_circuit`
- `verification.semantic.classification=adapter.health_readiness`
- `verification.semantic.runtime_path=core/runner,tool/local,runtime/config,adapter/health`
- `verification.semantic.governance=enforced`
- `verification.semantic.expected_markers=adapter_health_probe_sampled,adapter_readiness_circuit_transitioned,adapter_backoff_recovery_classified,governance_adapter_health_gate_enforced,governance_adapter_health_replay_bound`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`
- `result.final_answer` includes governance fields: `governance`, `ticket`, `replay`.

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If governance fields are missing, inspect marker handlers for `governance_adapter_health_gate_enforced` and `governance_adapter_health_replay_bound`.
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.


