# realtime-interrupt-resume (production-ish)

## Purpose
Real runtime semantic example for `realtime-interrupt-resume` with `production-ish` evidence profile.

## Variant Delta (vs minimal)
- Reuses the embedded-host command/response/event semantic anchor and runtime path baseline as minimal.
- Adds `governance_realtime_gate_enforced`: classify recovery result into `allow|allow_with_record|block`.
- Adds `governance_realtime_replay_bound`: bind replay signature from cursor trajectory and governance decision.
- Preserves minimal interrupt/resume chain and appends governance enforcement.
- Adds reverse HITL request/response correlation, duplicate/late response rejection, disconnect recovery without implicit Run mutation, bounded ingress/output backpressure, deterministic JSONL frame rejection, and stdout protocol purity expectations.
- Adds source-owned steering/follow-up admission, safe-point application, bounded causal follow-up promotion, duplicate/stale/terminal race classifications, disconnect settlement, and normalized Run/Stream parity expectations.
- Requires verification.semantic.governance=enforced.
- Requires verification.semantic.expected_markers and result.signature to differ from minimal.

## Run
go run ./examples/agent-modes/realtime-interrupt-resume/production-ish

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `embedded_host.command_response_event_correlation+runtime_input.steering_follow_up_safe_point` (with source recovery anchor `realtime.event_stream_terminal_recovery`).
- Classification: `realtime.resume_recovery`.
- Runtime path evidence: `core/types,core/runner,host,host/jsonl,tool/local,runtime/config,orchestration/composer,observability/event,observability/trace,runtime/diagnostics,tool/diagnosticsreplay`.
- Related contracts: `embedded-host-command-response-and-event-correlation-contract; runtime-steering-and-follow-up-input-contract; realtime-event-protocol-and-interrupt-resume-contract; durable-runtime-event-stream-binding; runtime-event-stream-terminal-recovery`.
- Required gates: `check-realtime-protocol-contract.*; check-agent-runtime-protocol-contract.*; check-runtime-event-stream-terminal-recovery-contract.*; check-runtime-steering-follow-up-contract.*; check-embedded-host-contract.*`.
- Replay fixtures: `embedded_host_protocol.v1; realtime_event_protocol.v1; agent_runtime_protocol.v1/stream-binding.json; runtime_event_stream_terminal_recovery.v1`.

## Host Contract Expectations

- Every command receives one immediate `accepted|rejected|duplicate` admission response; progress, reverse `host_request`, and terminal outcome are separate correlated Runtime envelopes. The first source-owned terminal wins races and remains immutable.
- Steering and follow-up commands are explicit input kinds with stable identity, Session/Run correlation, causation, profile version, and bounded payload. Admission never claims input application or follow-up execution completion.
- Steering uses a source-owned pending slot and applies only between atomic model/tool/HITL operations. Follow-up uses a bounded source-owned causal lane and is promoted only at the existing idle/terminal boundary into a distinct Run; a terminal Run is never reopened.
- Active cancel and Realtime interrupt/resume delegate to the source Run control. Unsupported, unauthorized, unknown, inactive, duplicate, late, sequence-gap, invalid-cursor, full, or closed-ingress operations are rejected with stable classifications and no synthetic terminal event.
- HITL reverse requests preserve the existing RequestID and timeout semantics. A valid clarification response resumes once; duplicate/late responses are rejected. Action Gate transport loss remains fail closed. Disconnect closes pending transport correlation but does not cancel unrelated Runs.
- Reconnect uses durable cursor catch-up/live handoff and terminal recovery. The adapter owns only bounded connection correlation and delivery state; it does not own event history, cursor allocation, terminal arbitration, or diagnostics writes.
- JSONL is raw LF framing (1 MiB default bound, validated override, U+2028/U+2029 retained inside a frame). Malformed/oversized frames fail before source mutation. A serialized writer keeps complete frames atomic, classifies backpressure/write failure, and keeps stdout protocol-only with logs on stderr.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P2`
- `verification.semantic.anchor=embedded_host.command_response_event_correlation+runtime_input.steering_follow_up_safe_point`
- `verification.semantic.classification=realtime.resume_recovery`
- `verification.semantic.runtime_path=core/types,core/runner,host,host/jsonl,tool/local,runtime/config,orchestration/composer,observability/event,observability/trace,runtime/diagnostics,tool/diagnosticsreplay`
- `verification.semantic.governance=enforced`
- `verification.semantic.expected_markers=host_negotiation_completed,host_command_correlated,host_command_admission_separated,runtime_steering_admitted,runtime_steering_applied_at_safe_point,runtime_follow_up_pending,host_active_interrupt_resume,host_terminal_authoritative,realtime_cursor_idempotent,realtime_interrupt_captured,realtime_resume_recovered,realtime_stream_binding_live,realtime_stream_binding_catch_up,realtime_stream_binding_handoff_dedup,realtime_stream_terminal_available,realtime_stream_recovery_retained_facts,runtime_steering_duplicate_rejected,runtime_follow_up_promoted,runtime_follow_up_backpressure,runtime_input_not_applied_on_disconnect,host_hitl_reverse_request,host_disconnect_recovery,host_jsonl_frame_rejected,host_stdout_pure,governance_realtime_gate_enforced,governance_realtime_replay_bound,realtime_stream_binding_expired,realtime_stream_binding_backpressure,realtime_stream_binding_disconnect_recovery`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`

## Failure/Rollback Notes
- If runtime path check fails, verify source ownership and host adapter boundaries (`core/types`, `core/runner`, `host`, `host/jsonl`) and rerun this variant.
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If governance/replay output is unexpected, inspect `governance_realtime_*` branches in `semantic_example.go`; then inspect host transcript normalization for admission/event separation and terminal first-wins facts.
- If input outcomes drift, inspect bounded source-owned lanes, safe-point application, follow-up promotion, and cancel/Realtime arbitration before changing host JSONL behavior; host state must not become an input or terminal owner.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together with the root README/MATRIX input row. Disable steering/follow-up capability advertisement and reject those inputs as unsupported; source Realtime, HITL, durable stream, terminal recovery, and diagnostics owners remain valid.


