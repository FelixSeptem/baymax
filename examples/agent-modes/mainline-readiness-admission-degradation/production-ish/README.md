# mainline-readiness-admission-degradation (production-ish)

## Purpose
Real runtime semantic example for `mainline-readiness-admission-degradation` with `production-ish` evidence profile.

## Variant Delta (vs minimal)
- Reuses the same semantic anchor and runtime path baseline as minimal.
- Uses degraded preflight metrics and admission-with-degradation path with rollback guard enabled.
- Adds governance branch (`governance_readiness_gate_enforced`, `governance_readiness_replay_bound`) for gate decision and replay trace.
- Requires verification.semantic.governance=enforced and a different `result.signature` from minimal.

## Run
go run ./examples/agent-modes/mainline-readiness-admission-degradation/production-ish

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `readiness.admission_degradation`.
- Classification: `mainline.readiness_admission`.
- Runtime path evidence: `core/runner,tool/local,runtime/config,runtime/diagnostics,orchestration/composer`.
- Related contracts: `runtime-readiness-preflight-contract; runtime-readiness-admission-guard-contract`.
- Required gates: `check-quality-gate.*`.
- Replay fixtures: `readiness-timeout-health-replay-fixture-gate.v1`.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P2`
- `verification.semantic.anchor=readiness.admission_degradation`
- `verification.semantic.classification=mainline.readiness_admission`
- `verification.semantic.runtime_path=core/runner,tool/local,runtime/config,runtime/diagnostics,orchestration/composer`
- `verification.semantic.governance=enforced`
- `verification.semantic.expected_markers=readiness_preflight_evaluated,admission_degradation_classified,readiness_rollback_guarded,governance_readiness_gate_enforced,governance_readiness_replay_bound`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`
- `result.final_answer` includes governance fields: `governance`, `ticket`, `replay`.

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If governance fields are missing, inspect marker handlers for `governance_readiness_gate_enforced` and `governance_readiness_replay_bound`.
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.


