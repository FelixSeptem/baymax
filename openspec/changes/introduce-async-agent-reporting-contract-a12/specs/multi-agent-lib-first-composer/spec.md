## ADDED Requirements

### Requirement: Composer SHALL support async child dispatch report sinks
Composer MUST support async child dispatch where child terminal outcomes are converged by report sink instead of mandatory synchronous wait.

#### Scenario: Composer dispatches child with async mode enabled
- **WHEN** composer dispatches a child task in async mode
- **THEN** composer returns accepted dispatch result and tracks child terminal through report sink updates

### Requirement: Composer async child reporting SHALL preserve scheduler terminal idempotency
Composer async child reporting integration MUST preserve scheduler terminal idempotency semantics for duplicate report deliveries.

#### Scenario: Duplicate async child terminal reports arrive
- **WHEN** same child terminal report is delivered more than once
- **THEN** composer/scheduler convergence keeps one logical terminal result and additive counters do not inflate
