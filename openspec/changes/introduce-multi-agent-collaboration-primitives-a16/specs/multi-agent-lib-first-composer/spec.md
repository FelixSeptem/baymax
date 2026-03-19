## ADDED Requirements

### Requirement: Composer SHALL expose collaboration primitive entrypoints
Composer MUST expose library-first entrypoints to run collaboration primitives (`handoff`, `delegation`, `aggregation`) through unified contracts.

#### Scenario: Host dispatches collaboration primitive through composer
- **WHEN** host invokes composer collaboration primitive API
- **THEN** runtime executes primitive through unified contract and returns normalized terminal outcome

### Requirement: Composer collaboration primitive execution SHALL compose with sync async delayed modes
Composer collaboration primitive execution MUST compose with synchronous invocation, async reporting, and delayed dispatch paths.

#### Scenario: Composer delegation uses delayed dispatch and async terminal reporting
- **WHEN** composer executes delegation primitive with delayed scheduling and async reporting enabled
- **THEN** terminal convergence remains deterministic and additive counters stay replay-idempotent

### Requirement: Composer collaboration primitive execution SHALL preserve Run Stream semantic equivalence
For equivalent collaboration primitive requests, composer Run and Stream paths MUST preserve semantically equivalent terminal category and additive summary fields.

#### Scenario: Equivalent collaboration primitive request via Run and Stream
- **WHEN** same collaboration primitive request is executed once via Run and once via Stream
- **THEN** summary semantics and terminal category remain semantically equivalent
