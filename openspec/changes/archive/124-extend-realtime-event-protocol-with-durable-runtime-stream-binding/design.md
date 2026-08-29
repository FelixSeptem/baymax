## Context

The archived Realtime Event Protocol defines a canonical event envelope, monotonic sequence, deduplication, interrupt/resume, and a resumable cursor. The Agent Runtime Protocol maps those source-owned events to canonical Run/Event references. Neither contract currently describes how an embedded host starts a durable consumption session, receives history after a cursor, transitions atomically to live events, interprets retention or slow-consumer outcomes, or correlates a reconnect attempt without inspecting Runner internals.

This change is constrained by Baymax's library-first architecture. `core/runner` remains responsible for Realtime execution and source-owned cursor validation; `runtime/*` must not depend on `mcp/http` or `mcp/stdio`; diagnostics writes continue exclusively through `observability/event.RuntimeRecorder`; and all QueryRuns/diagnostic additions remain nullable/defaultable. Existing Run/Stream, interrupt/resume, policy, snapshot, and adapter semantics must remain equivalent under unchanged effective configuration.

## Example Impact Assessment

修改示例

The `realtime-interrupt-resume` documentation baseline will be updated before its implementation and smoke assertions. It will show a bounded subscription request, reconnect from cursor, catch-up completion, live-tail handoff, expired cursor, backpressure classification, and rollback to the current source-owned path.

## Goals / Non-Goals

**Goals:**

- Define an embeddable, versioned event-stream binding DTO for subscription identity, source filter, cursor, bounded delivery settings, and source-owned outcomes.
- Make catch-up, live-tail handoff, duplicate suppression, cursor expiry, sequence gap, disconnect recovery, and slow-consumer/backpressure decisions deterministic and replayable.
- Reuse existing Realtime envelope fields, sequence, deduplication keys, interrupt/resume cursor validation, error taxonomy, and source ownership.
- Expose additive protocol/diagnostic/OTel correlation without creating a second event ledger or a parallel observability writer.
- Preserve v1 event and protocol fixtures, source behavior, and Run/Stream semantic equivalence.

**Non-Goals:**

- No SSE, WebSocket, gRPC, JSON-RPC, HTTP gateway, listener, connection manager, or transport implementation.
- No hosted Event/Session service, external event store, Redis/Kafka dependency, retention worker, global queue, or remote scheduler.
- No new event taxonomy, sequence allocator, deduplication store, interrupt/resume state machine, authorization model, or policy-precedence model.
- No guaranteed event durability where a source cannot already provide history; that source reports a bounded unresolved/expired outcome rather than synthesizing persistence.
- No provider-specific event payload or unbounded metadata/body projection.

## Decisions

### 1. The binding is a pure, source-owned projection

Introduce `EventStreamSubscription`, `EventStreamCursor`, `EventStreamDeliveryPolicy`, `EventStreamBindingOutcome`, and a finite binding state/reason taxonomy in the existing cross-module protocol owner. The binding accepts source-produced history/live event slices and source admission/retention outcomes; it performs validation and normalization only. It does not open a socket, hold a goroutine, retain events, mutate a cursor, create a queue, or authorize a caller.

Alternative considered: add an always-on in-process subscription service. Rejected because it would become a new event owner and implicitly create retention/backpressure behavior that source runtimes have not declared.

### 2. Cursor semantics remain owned by Realtime

Subscription start is explicitly one of `latest` or `after_cursor`. A cursor is opaque to the binding except for bounded validation and correlation. Realtime retains responsibility for cursor construction, validity, sequence relation, deduplication, interrupt freeze, and resume behavior. The binding can classify source results as `catching_up`, `live`, `expired`, `gap`, `disconnected`, `backpressured`, or `closed`, but cannot reinterpret a cursor or manufacture a replacement.

Alternative considered: derive a synthetic cursor from protocol event IDs. Rejected because non-Realtime sources do not offer the same sequence guarantees and it would split the Realtime cursor authority.

### 3. Handoff is explicit and overlap-safe

Catch-up and live-tail are separate phases. A source hands off by reporting the last delivered Realtime cursor/sequence and then providing subsequent live events. The binding normalizes an overlap through the existing Realtime event ID/dedup semantics and rejects a gap deterministically. It never claims a globally atomic handoff across independent storage and transport systems; host bindings must tolerate one bounded overlap and deduplicate it.

Alternative considered: promise exactly-once delivery. Rejected because existing contracts promise idempotent ingestion, not a distributed exactly-once transport guarantee.

### 4. Backpressure is classified, not implemented

Define a bounded consumer delivery policy (`reject`, `drop_with_record`, `pause_source`, `unknown`) and compatible source-owned outcome classifications. `pause_source` is only a reported source capability/outcome; the binding does not pause Runner execution. Unknown support remains explicit. Source reason codes remain primary; binding reason codes only classify contract validation failures such as incompatible policy/outcome or invalid subscription bounds.

Alternative considered: add a universal buffering queue. Rejected because it creates a global scheduler/queue owner and changes source resource semantics.

### 5. Subscription identity and observability are additive and bounded

Subscription IDs are caller supplied or source generated, bounded opaque references. Protocol mappings and diagnostics include nullable `stream_subscription_id`, binding phase/decision/reason, requested cursor mode, and source sequence correlation. OTel attributes use low-cardinality finite state/reason values and do not emit cursor or payload bodies. All diagnostic writes remain through `RuntimeRecorder`.

Alternative considered: expose arbitrary subscriber metadata. Rejected due to PII/cardinality risk and because authorization/tenant models are explicitly out of scope.

### 6. Replays compare normalized event streams, not transport timing

Fixtures encode source history, requested cursor, handoff marker, live events, binding outcome, and expected normalized protocol/diagnostic decisions. Replay permits transport timing/order normalization only where existing Realtime idempotency allows it; it must fail for cursor expiry changes, handoff gaps, backpressure policy/outcome mismatch, subscription correlation loss, or reason taxonomy drift. Equivalent Run and Stream inputs must yield equivalent classifications.

Alternative considered: integration-only reconnect tests. Rejected because drift in deterministic cursor/outcome classification must be caught without transport infrastructure.

## Risks / Trade-offs

- [Binding is mistaken for a durable transport] → Use transport-neutral names, gate against listeners/gateways/event stores, and document source-owned durability limits.
- [Catch-up/live-tail duplicates inflate events] → Reuse canonical `event_id`/dedup keys, require overlap fixtures, and preserve existing idempotency semantics.
- [A source reports unsupported retention ambiguously] → Require finite `expired|unresolved` outcomes and stable reason codes; never silently start from latest.
- [Backpressure implies a new global queue] → Validate projection-only helpers and add control-plane/queue-absence gate assertions.
- [Cursor/subscriber fields expand diagnostics cardinality] → Use opaque bounded identifiers, omit cursor bodies from OTel, and retain existing cardinality budgets.
- [Run and Stream diverge after reconnect] → Require parity fixtures with the same source history, cursor, delivery policy, and handoff outcome.

## Migration Plan

1. Add additive DTOs, validators, and pure projection helpers with empty/default binding fields for existing callers.
2. Add opt-in Realtime source mapping that consumes only source-owned event history, cursor, retention, and backpressure outcomes; leave existing Run/Stream event paths unchanged when no binding is requested.
3. Add nullable diagnostics/OTel fields through `RuntimeRecorder`, replay fixtures, and shell/PowerShell contract-gate coverage.
4. Update the Realtime example documentation baseline, then its executable projection and smoke coverage.
5. Roll back by disabling binding projection wiring and omitting the new fields; the existing event envelope, cursor, interrupt/resume, and Run/Stream behavior continue unchanged.

## Open Questions

- Which existing embedded sources can expose bounded history immediately, and which must report `unresolved` until their own storage contract exposes history?
- Should `pause_source` be included in the initial delivery policy vocabulary, or only represented as an unavailable/unknown source capability until a source runtime owns it?
- What maximum subscription identifier length, event batch size, and catch-up window fit the current diagnostics/cardinality budget without configuration expansion?
