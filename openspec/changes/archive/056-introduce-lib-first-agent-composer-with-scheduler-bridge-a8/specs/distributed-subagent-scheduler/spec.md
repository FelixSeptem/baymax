## ADDED Requirements

### Requirement: Scheduler initialization SHALL support fallback-to-memory backend
When configured scheduler backend initialization fails, composer-managed runtime MUST fallback to `memory` backend and MUST emit deterministic fallback diagnostics markers.

#### Scenario: File backend initialization fails at startup
- **WHEN** scheduler backend is configured as `file` and backend initialization fails
- **THEN** runtime falls back to `memory` backend, continues execution, and records fallback usage with an explicit reason marker

### Requirement: Scheduler config reload SHALL use next-attempt-only semantics
Scheduler-related hot-reload updates MUST apply to newly created or newly claimed attempts only, and MUST NOT retroactively change lease semantics of in-flight attempts.

#### Scenario: Scheduler lease config changes during an active attempt
- **WHEN** hot reload updates scheduler lease-related settings while a task attempt is already running
- **THEN** the running attempt keeps its existing lease semantics, and the updated settings apply from the next attempt boundary

### Requirement: Scheduler bridge SHALL converge local and A2A child terminals uniformly
Scheduler-managed local child-run and A2A child-run execution paths MUST converge through the same terminal commit idempotency contract.

#### Scenario: Duplicate terminal commits from mixed child targets
- **WHEN** duplicate terminal commits arrive for local and A2A child attempts
- **THEN** scheduler preserves a single logical terminal outcome and does not inflate additive counters
