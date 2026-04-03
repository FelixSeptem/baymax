## ADDED Requirements

### Requirement: Recovery Import via Unified Snapshot Contract
Composer and scheduler recovery paths MUST accept unified snapshot manifest input and MUST apply restore policy deterministically.

#### Scenario: Recovery strict mode boundary enforcement
- **WHEN** imported snapshot violates recovery boundary under strict restore mode
- **THEN** recovery MUST fail fast before mutating scheduler/composer runtime state

#### Scenario: Recovery compatible mode deterministic action
- **WHEN** compatible mode accepts an in-window snapshot
- **THEN** recovery MUST emit deterministic restore action and preserve canonical terminal arbitration semantics

### Requirement: Cross-Module Recovery Consistency
Recovery from unified snapshots MUST preserve Run/Stream semantic equivalence and memory/file backend parity.

#### Scenario: Run/Stream equivalence after restore
- **WHEN** the same snapshot is restored and resumed through `Run` and `Stream`
- **THEN** resulting terminal classification and recovery aggregates MUST be equivalent

#### Scenario: Backend parity after restore
- **WHEN** recovery is executed against memory and file scheduler backends from equivalent snapshots
- **THEN** restored task/session semantics MUST remain equivalent modulo additive metadata ordering
