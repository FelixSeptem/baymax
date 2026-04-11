# mainline-readiness-admission-degradation (minimal)

## Purpose
Real runtime semantic example for `mainline-readiness-admission-degradation` with `minimal` evidence profile.
This variant demonstrates readiness preflight evaluation and admission/rollback guard classification.

## Run
go run ./examples/agent-modes/mainline-readiness-admission-degradation/minimal

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
- `verification.semantic.governance=baseline`
- `verification.semantic.expected_markers=readiness_preflight_evaluated,admission_degradation_classified,readiness_rollback_guarded`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`
- `result.final_answer` includes readiness fields: `latency_p95_ms`, `error_rate_pct`, `readiness`, `admission`, `degradation`, `rollback_guard`, `rollback_plan`.

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If admission/rollback behavior diverges, inspect marker handlers for `admission_degradation_classified` and `readiness_rollback_guarded`.
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.
