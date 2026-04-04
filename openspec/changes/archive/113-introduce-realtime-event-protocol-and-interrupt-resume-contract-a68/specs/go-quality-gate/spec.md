## ADDED Requirements

### Requirement: Quality Gate SHALL Include A68 Realtime Contract Checks
Standard quality gate MUST execute A68 contract checks as blocking validations in both shell and PowerShell flows.

Repository MUST provide:
- `scripts/check-realtime-protocol-contract.sh`
- `scripts/check-realtime-protocol-contract.ps1`

#### Scenario: Contributor runs shell quality gate
- **WHEN** contributor executes `bash scripts/check-quality-gate.sh`
- **THEN** A68 contract checks run as required blocking steps

#### Scenario: Contributor runs PowerShell quality gate
- **WHEN** contributor executes `pwsh -File scripts/check-quality-gate.ps1`
- **THEN** equivalent A68 contract checks run as required blocking steps

### Requirement: A68 Gate SHALL Fail Fast on Realtime Semantics Drift
When A68 suites detect canonical semantic drift, quality gate MUST exit non-zero and block merge.

Semantic drift for this milestone MUST include at minimum:
- realtime event order drift
- realtime interrupt semantic drift
- realtime resume semantic drift
- realtime idempotency drift
- replay drift classification mismatch

#### Scenario: Realtime suite detects interrupt semantic drift
- **WHEN** equivalent fixture or integration inputs produce non-canonical interrupt outcomes
- **THEN** quality gate fails fast and blocks validation completion

#### Scenario: Replay suite detects drift-class mismatch
- **WHEN** `realtime_event_protocol.v1` replay validation returns non-canonical drift classification
- **THEN** quality gate fails fast and blocks validation completion

### Requirement: A68 Gate SHALL Enforce `realtime_control_plane_absent`
A68 gate MUST assert boundary condition `realtime_control_plane_absent` and fail on hosted realtime control-plane dependency introduction.

#### Scenario: Gate detects hosted realtime control-plane dependency
- **WHEN** A68 scope introduces dependency on hosted realtime gateway/control-plane runtime
- **THEN** gate fails with deterministic boundary-violation classification

#### Scenario: Library-embedded realtime implementation passes boundary assertion
- **WHEN** realtime implementation remains library-embedded without hosted control-plane dependency
- **THEN** boundary assertion passes

### Requirement: A68 Impacted Contract Suites Enforcement
Gate execution MUST enforce impacted suites for A68 scope changes and MUST reject merges when required suites are missing or failing.

#### Scenario: Realtime scope requires parity suites
- **WHEN** A68 changes touch interrupt/resume runtime boundaries
- **THEN** gate MUST require Run/Stream parity suites before merge

#### Scenario: Replay scope requires replay suites
- **WHEN** A68 changes touch fixture parser or drift classification logic
- **THEN** gate MUST require replay contract suites before merge

