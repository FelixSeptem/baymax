# realtime-interrupt-resume (production-ish)

## Purpose
Real runtime semantic example for `realtime-interrupt-resume` with `production-ish` evidence profile.

## Run
go run ./examples/agent-modes/realtime-interrupt-resume/production-ish

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `realtime.cursor_idempotent_interrupt_resume`.
- Classification: `realtime.resume_recovery`.
- Runtime path evidence: `core/runner,tool/local,runtime/config,runtime/diagnostics`.
- Related contracts: `realtime-event-protocol-and-interrupt-resume-contract`.
- Required gates: `check-realtime-protocol-contract.*`.
- Replay fixtures: `realtime_event_protocol.v1`.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P0`
- `verification.semantic.anchor=realtime.cursor_idempotent_interrupt_resume`
- `verification.semantic.classification=realtime.resume_recovery`
- `verification.semantic.runtime_path=core/runner,tool/local,runtime/config,runtime/diagnostics`
- `verification.semantic.governance=enforced`
- `verification.semantic.expected_markers=realtime_cursor_idempotent,realtime_interrupt_captured,realtime_resume_recovered,governance_realtime_gate_enforced,governance_realtime_replay_bound`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.
