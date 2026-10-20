## ADDED Requirements

### Requirement: Checkpoint replay SHALL be read-only and crash-window complete

Diagnostics fixtures SHALL represent checkpoint construction, persistence, host admission, claim, continuation start, terminal recording, expiry, corruption, foreign ownership, duplicate, conflict, and store-unavailable windows. Replay MUST compare normalized source facts and MUST NOT invoke model, tool, executor, retry, restore, or business effect code.

#### Scenario: Replay detects invented completion
- **WHEN** a fixture projects completed restore without a source terminal or committed paused successor while the source only records a claimed/running attempt
- **THEN** replay reports terminal-evidence drift without attempting recovery; a correctly projected recovery-required crash window remains valid

#### Scenario: Replay preserves unsupported boundaries
- **WHEN** a fixture requests composite restore without source evidence
- **THEN** replay records `unsupported_capability` and verifies no activation event exists
