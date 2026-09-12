## ADDED Requirements

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

