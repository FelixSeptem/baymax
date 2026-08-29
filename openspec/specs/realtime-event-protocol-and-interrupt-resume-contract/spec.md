# realtime-event-protocol-and-interrupt-resume-contract Specification

## Purpose
TBD - created by archiving change introduce-realtime-event-protocol-and-interrupt-resume-contract-a68. Update Purpose after archive.
## Requirements
### Requirement: Runtime SHALL Provide Canonical Realtime Event Envelope and Taxonomy
Runtime MUST provide a canonical realtime event envelope including:
- `event_id`
- `session_id`
- `run_id`
- `seq`
- `type`
- `ts`
- `payload`

Runtime MUST provide canonical event taxonomy covering:
- `request`
- `delta`
- `interrupt`
- `resume`
- `ack`
- `error`
- `complete`

#### Scenario: Realtime stream emits canonical envelope
- **WHEN** runtime emits realtime events for an active session
- **THEN** each event MUST include required canonical envelope fields with valid types

#### Scenario: Unsupported realtime event type is rejected
- **WHEN** runtime receives event type outside canonical taxonomy
- **THEN** runtime MUST fail validation with deterministic protocol error classification

### Requirement: Realtime Sequence and Idempotency Semantics
Realtime event processing MUST preserve monotonic sequence semantics and idempotent ingestion.

#### Scenario: Equivalent repeated event is deduplicated
- **WHEN** the same `event_id` (or dedup key) is ingested repeatedly
- **THEN** runtime MUST preserve semantically equivalent state and MUST NOT inflate logical counters

#### Scenario: Sequence gap is detected
- **WHEN** incoming event sequence skips required monotonic progression
- **THEN** runtime MUST classify deterministic sequence-gap protocol error

### Requirement: Interrupt and Resume Contract
Runtime MUST provide canonical interrupt/resume semantics with explicit resume cursor boundary.

#### Scenario: Interrupt freezes mutable output progression
- **WHEN** runtime accepts interrupt event for active stream
- **THEN** mutable output progression MUST stop at deterministic boundary and record resumable cursor

#### Scenario: Resume from valid cursor restores progression
- **WHEN** runtime receives resume event with valid cursor state
- **THEN** runtime MUST restore output progression from semantically equivalent boundary

#### Scenario: Resume with invalid cursor is rejected
- **WHEN** runtime receives resume event with non-resumable cursor
- **THEN** runtime MUST fail fast with deterministic resume-classified error

### Requirement: Realtime Contract MUST Keep Library-First Boundary
Realtime contract implementation MUST remain library-embedded and MUST NOT require platform control plane dependencies.

#### Scenario: Realtime contract gate validates control-plane absence
- **WHEN** contract gate validates realtime contract requirements
- **THEN** gate MUST assert `realtime_control_plane_absent` and fail on hosted control-plane dependency introduction

### Requirement: Realtime events SHALL map to canonical protocol lifecycle and event references
The existing realtime envelope and resume cursor MUST map deterministically to the Agent Runtime Protocol Event and Run lifecycle references. Realtime MUST remain the authority for event taxonomy, sequence, idempotency, interrupt freeze, and resume cursor validation.

#### Scenario: Valid resume maps without transport rewrite
- **WHEN** a valid realtime resume event is accepted
- **THEN** its canonical protocol mapping exposes the existing run/session/event correlation and resume causal relationship without adding a hosted transport dependency

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

