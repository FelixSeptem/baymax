# realtime-interrupt-resume (production-ish)

## Purpose
Real runtime semantic example for `realtime-interrupt-resume` with `production-ish` evidence profile.

## Variant Delta (vs minimal)
- Reuses the same semantic anchor and runtime path baseline as minimal.
- Adds `governance_realtime_gate_enforced`: classify recovery result into `allow|allow_with_record|block`.
- Adds `governance_realtime_replay_bound`: bind replay signature from cursor trajectory and governance decision.
- Preserves minimal interrupt/resume chain and appends governance enforcement.
- Requires verification.semantic.governance=enforced.
- Requires verification.semantic.expected_markers and result.signature to differ from minimal.

## Run
go run ./examples/agent-modes/realtime-interrupt-resume/production-ish

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `realtime.durable_runtime_stream_binding`.
- Classification: `realtime.resume_recovery`.
- Runtime path evidence: `core/runner,tool/local,runtime/config,core/types,observability/event,observability/trace,runtime/diagnostics`.
- Related contracts: `realtime-event-protocol-and-interrupt-resume-contract; durable-runtime-event-stream-binding`.
- Required gates: `check-realtime-protocol-contract.*; check-agent-runtime-protocol-contract.*`.
- Replay fixtures: `realtime_event_protocol.v1; agent_runtime_protocol.v1/stream-binding.json`.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P2`
- `verification.semantic.anchor=realtime.durable_runtime_stream_binding`
- `verification.semantic.classification=realtime.resume_recovery`
- `verification.semantic.runtime_path=core/runner,tool/local,runtime/config,core/types,observability/event,observability/trace,runtime/diagnostics`
- `verification.semantic.governance=enforced`
- `verification.semantic.expected_markers=realtime_cursor_idempotent,realtime_interrupt_captured,realtime_resume_recovered,realtime_stream_binding_live,realtime_stream_binding_catch_up,realtime_stream_binding_handoff_dedup,governance_realtime_gate_enforced,governance_realtime_replay_bound,realtime_stream_binding_expired,realtime_stream_binding_backpressure,realtime_stream_binding_disconnect_recovery`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If governance/replay output is unexpected, inspect `governance_realtime_*` branches in `semantic_example.go`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.


