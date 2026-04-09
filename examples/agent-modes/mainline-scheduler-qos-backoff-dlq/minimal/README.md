# mainline-scheduler-qos-backoff-dlq (minimal)

## Purpose
Real runtime semantic example for `mainline-scheduler-qos-backoff-dlq` with `minimal` evidence profile.

## Run
go run ./examples/agent-modes/mainline-scheduler-qos-backoff-dlq/minimal

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `scheduler.qos_backoff_dlq`.
- Classification: `mainline.scheduler_qos`.
- Runtime path evidence: `core/runner,tool/local,runtime/config,orchestration/scheduler,runtime/diagnostics`.
- Related contracts: `distributed-subagent-scheduler-qos`.
- Required gates: `check-multi-agent-shared-contract.*`.
- Replay fixtures: `cross-domain-primary-reason-arbitration-contract.v1`.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P2`
- `verification.semantic.anchor=scheduler.qos_backoff_dlq`
- `verification.semantic.classification=mainline.scheduler_qos`
- `verification.semantic.runtime_path=core/runner,tool/local,runtime/config,orchestration/scheduler,runtime/diagnostics`
- `verification.semantic.governance=baseline`
- `verification.semantic.expected_markers=scheduler_qos_fairness_applied,scheduler_backoff_budgeted,scheduler_dlq_classified`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.
