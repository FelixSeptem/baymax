## ADDED Requirements

### Requirement: Scaffold generator SHALL include capability negotiation and fallback test skeleton
Generated adapter scaffold MUST include minimal negotiation/fallback contract test skeletons aligned with repository taxonomy and strategy defaults.

The generated skeleton MUST cover:
- required capability missing fail-fast path,
- optional capability downgrade path,
- Run/Stream equivalence assertion for negotiation outcomes.

#### Scenario: Contributor generates scaffold and inspects tests
- **WHEN** contributor generates adapter scaffold
- **THEN** generated test skeleton includes negotiation and fallback contract cases with repository-default taxonomy markers

### Requirement: Scaffold defaults SHALL use fail_fast strategy and expose override hook
Generated scaffold configuration MUST default to `fail_fast` strategy and provide explicit request-level override hook for `best_effort`.

#### Scenario: Contributor uses default scaffold strategy
- **WHEN** contributor runs generated scaffold without strategy override
- **THEN** negotiation default behavior uses `fail_fast`

#### Scenario: Contributor uses generated override hook
- **WHEN** contributor applies request-level override to `best_effort`
- **THEN** generated scaffold path exercises downgrade behavior with deterministic reason output
