# realtime-interrupt-resume

## Prerequisites

Go 1.22+ with module dependencies resolved; no external network service is required.

## Semantic Anchor

`realtime.durable_runtime_stream_binding` extends the existing source-owned interrupt/resume example with a bounded, transport-neutral event-stream subscription. Realtime remains the owner of event history, cursor validation, sequence, deduplication, interrupt freeze, and resume semantics.

## Runtime Path

`core/types` validates the subscription and normalizes source history/live handoff; `core/runner` exposes the opt-in projection; `observability/event.RuntimeRecorder` records nullable binding correlation; `observability/trace` exports finite binding state/reason attributes; `runtime/diagnostics` stores the additive fields.

No listener, connection manager, hosted Event/Session service, external event store, global queue, synthetic cursor, or transport gateway is part of this example.

## Real Runtime Path

The executable variants exercise `core/runner`, `tool/local`, `runtime/config`, `core/types`, `observability/event`, `observability/trace`, and `runtime/diagnostics`.

## Expected Markers

The executable variants retain the existing markers `realtime_cursor_idempotent`, `realtime_interrupt_captured`, and `realtime_resume_recovered`, and add:

- `realtime_stream_binding_live`
- `realtime_stream_binding_catch_up`
- `realtime_stream_binding_handoff_dedup`
- `realtime_stream_binding_expired`
- `realtime_stream_binding_backpressure`
- `realtime_stream_binding_disconnect_recovery`

Production-ish additionally emits `governance_realtime_gate_enforced` and `governance_realtime_replay_bound`.

## Expected Output/Verification

Each variant prints `verification.mainline_runtime_path=ok`, `verification.semantic.phase=P2`, the semantic anchor above, one marker line per expected marker, and a deterministic `result.signature`.

## Reconnect, Catch-Up, and Handoff

`latest` starts a source-owned live tail. `after_cursor` asks Realtime for bounded history, reports `catching_up`, then transitions to `live` only with a valid handoff boundary. A bounded history/live overlap is deduplicated by the canonical Realtime event ID. A missing sequence is reported as `gap`; the binding does not synthesize a cursor or claim exactly-once delivery.

## Expiry and Backpressure

An expired cursor is reported as `expired` and never silently falls back to `latest`. Unknown history is `unresolved`. Backpressure is classified from the source-owned result (`drop_with_record`, `pause_source`, or `unknown`) and does not allocate a binding queue or pause Runner work itself.

## Rollback

Rollback removes the opt-in binding projection and omits its nullable diagnostics fields. The existing Realtime envelope, cursor, interrupt/resume, Run/Stream parity, and example markers remain valid. Revert the root README, MATRIX row, and both executable variants together if the documentation baseline must be withdrawn.

## Failure/Rollback Notes

If a marker or runtime path is missing, run the agent-mode semantic and README sync gates. Roll back the root README, MATRIX row, and both executable variants together.

## Variants

- `minimal/README.md`: baseline source-owned recovery and binding classifications.
- `production-ish/README.md`: baseline plus governance gate and replay signature.
