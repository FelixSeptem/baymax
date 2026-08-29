## Why

The archived Realtime Event Protocol contract already defines canonical envelopes, monotonic sequence, deduplication, interrupt/resume, and resumable cursors, but a host still cannot consume a long-running Runtime event stream across disconnects without learning source-specific internals. This change adds a bounded, transport-neutral binding for subscription, cursor catch-up, live-tail handoff, retention, and backpressure classification while keeping event ownership in the existing Realtime and Runtime sources.

## What Changes

- Add a versioned `durable-runtime-event-stream-binding` capability that describes subscription requests, cursor-based catch-up, live-tail handoff, disconnect recovery, retention/expiry, and consumer backpressure outcomes.
- Define deterministic stream lifecycle classifications for accepted, catching-up, live, expired-cursor, gap, backpressured, disconnected, and closed outcomes without adding a second event taxonomy.
- Preserve the existing Realtime source as the authority for event sequence, deduplication, interrupt/resume, cursor validation, and reason codes; the binding only projects source-owned results.
- Add replay fixtures and Run/Stream parity coverage for catch-up/live-tail transitions, duplicate events, cursor expiry, sequence gaps, disconnect recovery, and backpressure.
- Extend diagnostics and OTel projection additively with nullable subscription/binding correlation through `RuntimeRecorder`; preserve cardinality limits and existing v1 fixtures.
- Update the `realtime-interrupt-resume` example documentation and smoke path before implementation to demonstrate reconnect and catch-up semantics.
- Do not introduce a transport gateway, hosted Session/Event service, external event store, Redis/Kafka dependency, global queue, or control plane.

## Example Impact Assessment

修改示例

The existing `realtime-interrupt-resume` example will be updated with a documentation-first subscription, cursor catch-up, live-tail handoff, disconnect, and backpressure projection. The example will retain existing interrupt/resume lifecycle behavior and document rollback to the current source-owned event path.

## Capabilities

### New Capabilities

- `durable-runtime-event-stream-binding`: Transport-neutral, bounded subscription and reconnect semantics for Runtime event streams, including catch-up, live-tail handoff, retention, disconnect, and backpressure classifications.

### Modified Capabilities

- `realtime-event-protocol-and-interrupt-resume-contract`: Extend the existing Realtime contract with binding integration points while preserving its event ordering, deduplication, interrupt/resume, cursor validation, and control-plane-absence requirements.
- `agent-runtime-protocol-contract`: Add an additive mapping from source-owned event-stream binding outcomes to protocol-visible correlation and lifecycle metadata without changing the six-object lifecycle model.

## Impact

- `core/types` and `core/runner`: additive stream-binding DTOs, validation, and source-owned projection helpers; no new scheduler or event store.
- Realtime and protocol mapping adapters: expose subscription/catch-up/live-tail outcomes while retaining Realtime as the source of truth.
- Diagnostics, OTel, replay fixtures, and contract gates: nullable binding correlation, deterministic drift classifications, shell/PowerShell parity, and `RuntimeRecorder` single-write enforcement.
- `examples/agent-modes/realtime-interrupt-resume`: documentation-first update and executable smoke assertions.
- Existing Realtime, Agent Runtime Protocol v1, Snapshot, policy, and Run/Stream behavior remains additive, nullable, and backward compatible.
