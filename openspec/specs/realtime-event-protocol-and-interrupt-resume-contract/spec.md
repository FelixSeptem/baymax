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

### Requirement: Realtime SHALL expose source-owned ingress during an active Run
The Realtime owner MUST expose an optional bounded ingress through the active source Run control so a host can submit valid interrupt and resume envelopes after Run or Stream admission. The ingress MUST apply the existing envelope validation, session/Run correlation, monotonic sequence, deduplication key, cursor, and interrupt/resume state rules before mutation.

An inactive or unknown Run, mismatched Session, sequence gap, duplicate event, invalid cursor, closed ingress, or full ingress boundary MUST return a deterministic admission outcome. The host adapter MUST NOT maintain a second Realtime cursor, sequence allocator, deduplication store, or state machine.

#### Scenario: Active interrupt enters through source ingress
- **WHEN** a host submits a valid next-sequence interrupt envelope for an active Run
- **THEN** the source Realtime owner accepts it once, freezes mutable output progression, and exposes the existing input-required and resume-cursor projection

#### Scenario: Unknown Run rejects realtime command
- **WHEN** a host targets a Run with no active source ingress
- **THEN** admission is rejected without creating a cursor, event record, or synthetic terminal outcome for that Run

#### Scenario: Duplicate active event remains idempotent
- **WHEN** the same accepted interrupt or resume envelope is submitted again while the Run is active
- **THEN** the source deduplication rule reports a duplicate and does not repeat the interrupt or resume mutation

### Requirement: Active ingress SHALL preserve interrupt, resume, and disconnect ownership
An accepted interrupt MUST retain the existing freeze and `input_required` semantics. Resume MUST require the current source-owned cursor and valid lifecycle state before output progression continues. Closing the host connection or event observation path MUST NOT itself submit an interrupt, resume, or cancel command and MUST NOT change the source Run state.

Ingress backpressure MUST be bounded and explicit. A full or closed ingress MUST reject the command rather than block indefinitely, silently drop a control envelope, or transfer Realtime ownership to the host adapter.

#### Scenario: Resume uses current source cursor
- **WHEN** a host submits a resume envelope carrying the active Run's current source-owned cursor
- **THEN** the Realtime owner resumes progression once and preserves existing sequence and protocol lifecycle mapping

#### Scenario: Host disconnect does not mutate realtime state
- **WHEN** an observer disconnects while a Run is active and sends no explicit control command
- **THEN** the Run's interrupt/resume state remains unchanged and later observation uses the durable stream recovery contract

#### Scenario: Full ingress rejects without blocking
- **WHEN** the bounded source ingress cannot accept another control envelope within its declared admission boundary
- **THEN** the command is rejected with a stable backpressure classification and no envelope is silently discarded as accepted

### Requirement: Active realtime ingress SHALL be replayable and Run/Stream equivalent
Versioned Realtime and host-contract fixtures MUST cover mid-run interrupt, valid and invalid resume, duplicate and late delivery, sequence gap, unknown Run, ingress backpressure, disconnect without mutation, terminal race, and Run/Stream parity. Equivalent Run and Stream workloads MUST preserve the same normalized Realtime admission, cursor, deduplication, lifecycle, and terminal semantics.

#### Scenario: Mid-run control transcript replays deterministically
- **WHEN** replay processes a valid active-Run interrupt/resume transcript
- **THEN** it reproduces the expected source-owned admission, cursor, sequence, deduplication, and lifecycle facts without live transport

#### Scenario: Active ingress parity drifts
- **WHEN** equivalent Run and Stream transcripts differ in interrupt/resume admission or resulting lifecycle semantics
- **THEN** replay returns a stable parity-drift classification and the Realtime contract gate fails
