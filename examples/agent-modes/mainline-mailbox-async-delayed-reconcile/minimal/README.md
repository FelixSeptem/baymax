# mainline-mailbox-async-delayed-reconcile (minimal)

## Purpose
Real runtime semantic example for `mainline-mailbox-async-delayed-reconcile` with `minimal` evidence profile.
This variant demonstrates delayed mailbox dispatch and reconcile with timeline reason emission.

## Run
go run ./examples/agent-modes/mainline-mailbox-async-delayed-reconcile/minimal

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `mailbox.async_delayed_reconcile`.
- Classification: `mainline.mailbox_reconcile`.
- Runtime path evidence: `core/runner,tool/local,runtime/config,orchestration/mailbox,orchestration/invoke,runtime/diagnostics`.
- Related contracts: `multi-agent-mailbox-contract; multi-agent-async-await-reconcile-contract`.
- Required gates: `check-multi-agent-shared-contract.*`.
- Replay fixtures: `cross-domain-primary-reason-arbitration-contract.v1`.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P2`
- `verification.semantic.anchor=mailbox.async_delayed_reconcile`
- `verification.semantic.classification=mainline.mailbox_reconcile`
- `verification.semantic.runtime_path=core/runner,tool/local,runtime/config,orchestration/mailbox,orchestration/invoke,runtime/diagnostics`
- `verification.semantic.governance=baseline`
- `verification.semantic.expected_markers=mailbox_async_delayed_dispatched,mailbox_reconcile_triggered,mailbox_timeline_reason_emitted`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`
- `result.final_answer` includes dispatch/reconcile fields: `delay_sec`, `pending`, `reconcile_batch`, `reconciled`, `late`, `timeline_reason`, `primary_reason`.

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If reconcile/timeline behavior diverges, inspect marker handlers for `mailbox_reconcile_triggered` and `mailbox_timeline_reason_emitted`.
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.
