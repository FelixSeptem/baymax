## ADDED Requirements

### Requirement: Composed orchestration SHALL support async dispatch with later report convergence
Composed orchestration MUST support non-blocking async dispatch where terminal convergence can occur through report sink callbacks.

#### Scenario: Composed run dispatches remote task asynchronously
- **WHEN** composed orchestration chooses async dispatch for a remote step
- **THEN** run execution can continue without blocking and terminal convergence is provided through report flow

### Requirement: Composed async reporting SHALL preserve deterministic terminal convergence
For equivalent inputs and effective config, composed async report convergence MUST preserve deterministic terminal category and aggregate counter semantics.

#### Scenario: Equivalent async composed request is replayed
- **WHEN** equivalent async composed request is replayed under same configuration
- **THEN** final terminal category and additive aggregates remain semantically equivalent
