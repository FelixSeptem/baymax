# adapter-health-backoff-and-circuit-governance Specification

## Purpose
TBD - created by archiving change introduce-adapter-health-backoff-and-circuit-governance-contract-a46. Update Purpose after archive.
## Requirements
### Requirement: Adapter health probing SHALL enforce backoff and circuit-governed scheduling
Adapter health probing MUST apply exponential backoff with jitter and circuit-governed probing windows to prevent probe storms under repeated failures.

Circuit state model MUST include:
- `closed`
- `open`
- `half_open`

Backoff behavior MUST be deterministic for equivalent input sequence and equivalent effective configuration.

#### Scenario: Repeated probe failures trigger governed slowdown
- **WHEN** one adapter probe path repeatedly fails and reaches configured threshold
- **THEN** probe scheduling is throttled by backoff and circuit open-window semantics

#### Scenario: Recovery path returns to normal probe cadence
- **WHEN** open window expires and half-open probes satisfy configured success threshold
- **THEN** circuit transitions back to `closed` and normal probe cadence resumes

### Requirement: Circuit transitions SHALL be canonical and replay-stable
Circuit transitions MUST follow canonical state-transfer rules and MUST remain semantically stable under replay.

Canonical transitions:
- `closed -> open` on consecutive failures meeting threshold
- `open -> half_open` on open-window expiry
- `half_open -> open` on any failed half-open probe
- `half_open -> closed` on consecutive successful half-open probes meeting threshold

#### Scenario: Half-open failure immediately reopens circuit
- **WHEN** circuit is `half_open` and next probe fails
- **THEN** circuit transitions to `open` and open-window cooldown is re-applied

#### Scenario: Replayed equivalent probe events preserve terminal circuit state
- **WHEN** equivalent probe events are replayed for one adapter in one evaluation window
- **THEN** resulting logical circuit state and transition counters remain stable after first logical ingestion

