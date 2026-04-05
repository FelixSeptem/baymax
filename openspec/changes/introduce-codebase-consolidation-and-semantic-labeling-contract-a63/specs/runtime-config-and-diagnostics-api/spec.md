## ADDED Requirements

### Requirement: Runtime Config SHALL Support Semantic Primary Keys with Legacy Alias Compatibility
Runtime config surface touched by A63 naming convergence MUST define semantic primary keys and legacy-compatible alias read path for CA-era keys during migration.

Config precedence (`env > file > default`) and fail-fast validation semantics MUST remain unchanged.

#### Scenario: Legacy CA-era config key is provided
- **WHEN** runtime receives legacy CA-era config key that has semantic replacement
- **THEN** runtime MUST resolve effective value deterministically through alias compatibility and preserve behavior semantics

#### Scenario: Semantic primary key and legacy alias conflict
- **WHEN** both semantic primary key and legacy alias are supplied with different values
- **THEN** runtime MUST apply documented deterministic precedence and emit migration-facing diagnostics

### Requirement: Runtime Diagnostics SHALL Provide Semantic Field Migration Compatibility
Diagnostics fields touched by A63 naming convergence MUST provide semantic primary names while preserving parser compatibility for legacy field names in replay and consumers during migration window.

Additive compatibility contract (`additive + nullable + default`) MUST remain unchanged.

#### Scenario: Consumer parses run payload with legacy field names
- **WHEN** consumer or replay fixture contains legacy field naming
- **THEN** parser compatibility layer MUST keep semantic interpretation equivalent

#### Scenario: Consumer parses run payload with semantic field names
- **WHEN** diagnostics output includes semantic primary fields
- **THEN** consumers and query paths MUST parse and aggregate without semantic drift

