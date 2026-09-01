# realtime-interrupt-resume (minimal)

## Purpose
Real runtime semantic example for `realtime-interrupt-resume` with `minimal` evidence profile.
This variant executes a concrete realtime recovery chain: cursor idempotency dedupe, interrupt capture, and checkpoint resume recovery.

## Run
go run ./examples/agent-modes/realtime-interrupt-resume/minimal

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `realtime.event_stream_terminal_recovery`.
- Classification: `realtime.resume_recovery`.
- Runtime path evidence: `core/runner,tool/local,runtime/config,core/types,observability/event,observability/trace,runtime/diagnostics,tool/diagnosticsreplay`.
- Related contracts: `realtime-event-protocol-and-interrupt-resume-contract; durable-runtime-event-stream-binding; runtime-event-stream-terminal-recovery`.
- Required gates: `check-realtime-protocol-contract.*; check-agent-runtime-protocol-contract.*; check-runtime-event-stream-terminal-recovery-contract.*`.
- Replay fixtures: `realtime_event_protocol.v1; agent_runtime_protocol.v1/stream-binding.json; runtime_event_stream_terminal_recovery.v1`.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P2`
- `verification.semantic.anchor=realtime.event_stream_terminal_recovery`
- `verification.semantic.classification=realtime.resume_recovery`
- `verification.semantic.runtime_path=core/runner,tool/local,runtime/config,core/types,observability/event,observability/trace,runtime/diagnostics,tool/diagnosticsreplay`
- `verification.semantic.governance=baseline`
- `verification.semantic.expected_markers=realtime_cursor_idempotent,realtime_interrupt_captured,realtime_resume_recovered,realtime_stream_binding_live,realtime_stream_binding_catch_up,realtime_stream_binding_handoff_dedup,realtime_stream_terminal_available,realtime_stream_recovery_retained_facts`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If cursor/interrupt/resume outputs are unexpected, inspect event/signal fixtures in `semantic_example.go`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.
