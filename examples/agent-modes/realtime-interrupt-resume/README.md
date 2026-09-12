# realtime-interrupt-resume

## Prerequisites

Go 1.22+ with module dependencies resolved; no external network service is required.

## Semantic Anchor

`embedded_host.command_response_event_correlation` extends the existing source-owned interrupt/resume example with a bounded, transport-neutral embedded-host contract. The host receives one admission response per command and observes later source-owned Runtime events, reverse HITL requests, and the authoritative terminal outcome through stable message/request/run/session/source correlations. Realtime remains the owner of event history, cursor validation, sequence, deduplication, interrupt freeze, resume semantics, retention, and slow-consumer outcomes. The terminal arbiter remains the owner of first-terminal-wins semantics.

## Runtime Path

`core/types` validates host envelopes, command admission, Realtime ingress, source history/live handoff, and terminal recovery; `core/runner` owns the opt-in active Run control and equivalent Run/Stream safe points; `host` coordinates bounded connection correlation and existing resolver/event/terminal owners; `host/jsonl` provides the strict local binding; `observability/event.RuntimeRecorder` records nullable binding/recovery correlation; `observability/trace` exports finite binding state/reason attributes; `runtime/diagnostics` stores additive fields.

No listener, hosted Event/Session service, external event store, global queue, synthetic cursor, adapter-owned terminal state machine, steering/follow-up queue, or transport gateway is part of this example. The host adapter never becomes a second Run, Realtime, terminal, cursor, or diagnostics owner.

## Real Runtime Path

The executable variants exercise `core/runner`, `tool/local`, `runtime/config`, `core/types`, `observability/event`, `observability/trace`, and `runtime/diagnostics`.

## Expected Markers

The executable variants retain the existing markers `realtime_cursor_idempotent`, `realtime_interrupt_captured`, and `realtime_resume_recovered`, and add the host-contract baseline:

- `host_negotiation_completed`
- `host_command_correlated`
- `host_command_admission_separated`
- `host_active_interrupt_resume`
- `host_terminal_authoritative`

The production-ish variant additionally exercises:

- `host_hitl_reverse_request`
- `host_disconnect_recovery`
- `host_jsonl_frame_rejected`
- `host_stdout_pure`
- `governance_realtime_gate_enforced`
- `governance_realtime_replay_bound`

The existing stream-binding markers remain part of the source-owned recovery evidence: `realtime_stream_binding_live`, `realtime_stream_binding_catch_up`, `realtime_stream_binding_handoff_dedup`, `realtime_stream_binding_expired`, `realtime_stream_binding_backpressure`, `realtime_stream_binding_disconnect_recovery`, `realtime_stream_terminal_available`, and `realtime_stream_recovery_retained_facts`.

## Expected Output/Verification

Each variant prints `verification.mainline_runtime_path=ok`, `verification.semantic.phase=P2`, the semantic anchor above, one marker line per expected marker, and a deterministic `result.signature`.

## Reconnect, Catch-Up, and Handoff

`latest` starts a source-owned live tail. `after_cursor` asks Realtime for bounded history, reports `catching_up`, then transitions to `live` only with a valid handoff boundary. A bounded history/live overlap is deduplicated by the canonical Realtime event ID. A missing sequence is reported as `gap`; the binding does not synthesize a cursor or claim exactly-once delivery. Host command admission (`accepted|rejected|duplicate`) is separate from asynchronous progress and terminal events: accepted cancel/interrupt does not imply a canceled terminal, and rejected commands emit no synthetic business terminal. Observer disconnect/stop does not mutate the Run or trigger retry/resume. Recovery can expose a source-owned terminal snapshot before its terminal event arrives; the existing terminal arbiter preserves the first business terminal and records late conflicts without overwriting it.

## Expiry and Backpressure

An expired cursor is reported as `expired` and never silently falls back to `latest`. Unknown history is `unresolved`. Backpressure is classified from the source-owned result (`drop_with_record`, `pause_source`, or `unknown`) and does not allocate a binding queue or pause Runner work itself. Host transport dropping is limited to Realtime `delta` events when the returned projection declares both `drop_with_record` and a source-owned outcome and a standard EventSink can record the drop; direct Runner callbacks and all non-delta events remain non-droppable.

## Rollback

Rollback removes the opt-in host projection and its nullable correlation/diagnostics fields, then removes the strict JSONL adapter wiring. The existing Realtime envelope, cursor, interrupt/resume, durable binding, Run/Stream parity, resolver behavior, and legacy example markers remain valid. Revert the root README, MATRIX row, mode README files, and both executable variants together if the documentation baseline must be withdrawn; no source Run or terminal owner is rolled back by removing the host adapter.

## Failure/Rollback Notes

If a marker or runtime path is missing, run the agent-mode semantic and README sync gates plus the host transcript/replay and JSONL framing gates once they land. Roll back the root README, MATRIX row, mode README files, and both executable variants together. A transport disconnect is an observation/pending-correlation cleanup, not an implicit cancel or terminal mutation.

## Variants

- `minimal/README.md`: baseline source-owned recovery plus host negotiation, command correlation, admission/event separation, active interrupt/resume, and authoritative terminal markers.
- `production-ish/README.md`: baseline plus reverse HITL request, duplicate/late response handling, disconnect recovery, JSONL frame rejection/backpressure, stdout purity, governance gate, and replay signature.

## Contract, Replay, and Gate Mapping

- Contracts: `embedded-host-command-response-and-event-correlation-contract`; `realtime-event-protocol-and-interrupt-resume-contract`; `durable-runtime-event-stream-binding`; `runtime-event-stream-terminal-recovery`.
- Planned replay fixtures: `embedded_host_transcript.v1`; `realtime_event_protocol.v1`; `agent_runtime_protocol.v1/stream-binding.json`; `runtime_event_stream_terminal_recovery.v1`.
- Required gates: `check-realtime-protocol-contract.*`; `check-agent-runtime-protocol-contract.*`; `check-runtime-event-stream-terminal-recovery-contract.*`; the host-contract, JSONL conformance, replay, and shell/PowerShell parity gates introduced by this change.
- Example impact assessment: `修改示例`. The baseline is documentation-only until tasks 1.1 and 1.2 are complete; implementation tasks must not be checked before this anchor/path/marker/rollback mapping is present.
