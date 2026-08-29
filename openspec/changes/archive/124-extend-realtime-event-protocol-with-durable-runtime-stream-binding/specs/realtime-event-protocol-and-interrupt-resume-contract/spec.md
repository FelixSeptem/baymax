## ADDED Requirements

### Requirement: Realtime SHALL provide source-owned binding integration points

Realtime SHALL remain the authority for canonical envelope taxonomy, sequence progression, deduplication, interrupt/resume, and cursor validation. It SHALL expose bounded source-owned integration points that permit an embedded durable event-stream binding to request history after an existing cursor, receive a live-tail handoff boundary, and classify retention, disconnect, and backpressure outcomes without rewriting Realtime reason taxonomy or event ordering rules.

#### Scenario: Binding reuses a valid Realtime resume cursor
- **WHEN** an embedded binding requests catch-up using a valid existing Realtime cursor
- **THEN** Realtime supplies or classifies the source-owned result using its existing cursor and sequence semantics without creating a second resume path

#### Scenario: Binding does not change interrupt/resume ownership
- **WHEN** a subscription is catching up or live and a valid interrupt/resume event is processed
- **THEN** Realtime retains interrupt freeze and resume validation ownership while the binding only projects the resulting events and correlation

### Requirement: Realtime binding integration SHALL preserve library-first boundaries

Realtime binding integration SHALL remain library-embedded and transport-neutral. It MUST NOT introduce an HTTP/SSE/WebSocket/gRPC listener, hosted connection manager, external event ledger, retention worker, or control plane. Source history availability and delivery control remain explicit source capabilities rather than implicit platform guarantees.

#### Scenario: Realtime binding gate rejects transport ownership
- **WHEN** the Realtime binding contract gate inspects dependencies and runtime wiring
- **THEN** it fails if Realtime integration creates a transport server, hosted event store, or control-plane-owned subscription lifecycle
