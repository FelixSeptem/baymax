# realtime-interrupt-resume (minimal)

## Purpose
Real runtime semantic example for `realtime-interrupt-resume` with `minimal` evidence profile.
This variant executes a concrete realtime recovery chain: cursor idempotency dedupe, interrupt capture, checkpoint resume recovery, and the documentation baseline for an embedded host command/response/event contract.

## Run
go run ./examples/agent-modes/realtime-interrupt-resume/minimal

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `embedded_host.command_response_event_correlation+runtime_input.steering_follow_up_safe_point` (with source recovery anchor `realtime.event_stream_terminal_recovery`).
- Classification: `realtime.resume_recovery`.
- Runtime path evidence: `core/types,core/runner,host,host/jsonl,tool/local,runtime/config,orchestration/composer,observability/event,observability/trace,runtime/diagnostics,tool/diagnosticsreplay`.
- Related contracts: `embedded-host-command-response-and-event-correlation-contract; runtime-steering-and-follow-up-input-contract; realtime-event-protocol-and-interrupt-resume-contract; durable-runtime-event-stream-binding; runtime-event-stream-terminal-recovery`.
- Required gates: `check-realtime-protocol-contract.*; check-agent-runtime-protocol-contract.*; check-runtime-event-stream-terminal-recovery-contract.*; check-runtime-steering-follow-up-contract.*; host-contract/jsonl/replay/parity gates planned by the change`.
- Replay fixtures: `embedded_host_protocol.v1; realtime_event_protocol.v1; agent_runtime_protocol.v1/stream-binding.json; runtime_event_stream_terminal_recovery.v1`.

## Host Contract Expectations

- The host negotiates one supported profile before mutation commands and correlates each command response with its request, `session_id`, `run_id`, causation, and source facts when known.
- Admission is two-phase: `accepted|rejected|duplicate` reports only command admission; asynchronous progress and the single source-owned terminal outcome arrive as `runtime_event` envelopes. An accepted cancel or interrupt is not a synthetic canceled/completed result.
- `cancel` and active `realtime.interrupt|resume` are delegated to the source-owned Run control. Realtime validates session/run identity, sequence, dedupe, cursor, and safe-point lifecycle; disconnect does not submit an implicit control command.
- The minimal profile documents the resolver boundary but does not claim a live HITL bridge: any future `host_request`/`host_response` must preserve existing RequestID, timeout, clarification cancellation, and Action Gate fail-closed semantics.
- `steering` and `follow_up` are explicit versioned input kinds. Admission is source-owned and bounded: one pending steering item per active Run plus a bounded follow-up lane on the causal chain. A command response reports only `accepted|rejected|duplicate`; application and promotion arrive later as correlated Runtime facts.
- Steering is applied only at a safe point after an atomic model/tool/HITL operation and before the next model decision. Follow-up remains pending until the existing idle/terminal boundary and, when promoted, creates a distinct causal Run without rewriting the original terminal Run.
- Cancel, Realtime interrupt/resume, terminal first-wins, disconnect cleanup, and Run/Stream parity retain precedence. Disconnect settles accepted-but-unapplied input as `not_applied`/`disconnected`; it never cancels the Run or creates a host-owned queue.
- Reconnection uses durable cursor catch-up and terminal recovery; a disconnected observer does not cancel or rewrite the business Run.
- The strict JSONL binding is LF-delimited, one object per frame, bounded to the planned 1 MiB default, and keeps stdout protocol-only. Logs belong on the injected non-stdout writer (stderr by default); malformed/oversized frames are rejected before source mutation.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P2`
- `verification.semantic.anchor=embedded_host.command_response_event_correlation+runtime_input.steering_follow_up_safe_point`
- `verification.semantic.classification=realtime.resume_recovery`
- `verification.semantic.runtime_path=core/types,core/runner,host,host/jsonl,tool/local,runtime/config,orchestration/composer,observability/event,observability/trace,runtime/diagnostics,tool/diagnosticsreplay`
- `verification.semantic.governance=baseline`
- `verification.semantic.expected_markers=host_negotiation_completed,host_command_correlated,host_command_admission_separated,runtime_steering_admitted,runtime_steering_applied_at_safe_point,runtime_follow_up_pending,host_active_interrupt_resume,host_terminal_authoritative,realtime_cursor_idempotent,realtime_interrupt_captured,realtime_resume_recovered,realtime_stream_binding_live,realtime_stream_binding_catch_up,realtime_stream_binding_handoff_dedup,realtime_stream_terminal_available,realtime_stream_recovery_retained_facts`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`

## Failure/Rollback Notes
- If runtime path check fails, verify the documented source-owner path (`core/types -> core/runner -> host -> host/jsonl`) and rerun this variant.
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If cursor/interrupt/resume outputs are unexpected, inspect event/signal fixtures in `semantic_example.go`; do not infer terminal state from an admission response.
- If input markers are missing, inspect the source-owned admission/safe-point path and verify that steering was not injected into an in-flight Provider call or that follow-up was not appended to a terminal Run.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together and remove the input projection from the root README/MATRIX row. Existing Realtime, resolver, durable-stream, terminal-recovery, and Run/Stream owners remain unchanged; reject steering/follow-up as unsupported rather than retaining a partial queue.
