## ADDED Requirements

### Requirement: Session recovery SHALL apply explicit resume boundary policy for long-running execution
Session recovery contract MUST apply explicit resume boundary policy and reject ambiguous recovery continuation paths.

#### Scenario: Recovery request lacks deterministic continuation boundary
- **WHEN** restored state cannot be mapped to deterministic next-attempt continuation
- **THEN** recovery fails fast with explicit boundary conflict classification

### Requirement: Session recovery timeout reentry budget SHALL be deterministic and bounded
Session recovery MUST track timeout reentry budget per task and enforce configured maximum reentry count.

#### Scenario: Recovery resumes timeout-sensitive task
- **WHEN** recovered task exceeds timeout reentry budget
- **THEN** runtime converges task to deterministic terminal failure with bounded reentry counter

### Requirement: Session recovery SHALL preserve replay-idempotent outcomes under boundary enforcement
Recovery boundary enforcement MUST preserve replay idempotency for terminal outcomes and additive counters under repeated replay attempts.

#### Scenario: Same recovery batch is replayed with boundary checks enabled
- **WHEN** identical recovery snapshot and events are replayed multiple times
- **THEN** logical terminal outcomes and recovery counters remain stable after first logical ingestion
