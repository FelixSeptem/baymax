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

