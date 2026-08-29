## ADDED Requirements

### Requirement: Runtime Protocol SHALL project durable event-stream binding correlation additively

The Agent Runtime Protocol SHALL project source-owned durable event-stream binding results additively with nullable/defaultable subscription correlation, binding phase/decision, source reason, requested cursor mode, and available Realtime sequence boundary. This projection SHALL preserve existing Session, Run, Event, causation, source, and trace correlation; it SHALL not create a protocol-owned event stream, authorization decision, persistence store, or parallel lifecycle state machine.

#### Scenario: Binding outcome preserves protocol event correlation
- **WHEN** a Realtime source reports a catch-up-to-live binding outcome for a correlated Run and Session
- **THEN** the protocol projection retains existing run/session/event/source correlation plus nullable binding decision and subscription reference

#### Scenario: Existing protocol v1 mapping remains compatible
- **WHEN** an existing `agent_runtime_protocol.v1` fixture has no durable binding fields
- **THEN** replay remains valid and the new binding projection fields are absent or defaulted

### Requirement: Durable binding observability SHALL use the RuntimeRecorder single-write path

Durable binding decisions that require diagnostics or OTel projection SHALL be written through `observability/event.RuntimeRecorder` with bounded, additive correlation fields. Cursor bodies, event payloads, and arbitrary subscriber metadata SHALL NOT be emitted as high-cardinality diagnostic or OTel attributes.

#### Scenario: Binding drift is observable without parallel diagnostics writing
- **WHEN** replay detects a cursor expiry, handoff gap, backpressure, or subscription-correlation drift
- **THEN** the corresponding additive diagnostics/OTel projection is attributable through `RuntimeRecorder` and no parallel diagnostics writer is used
