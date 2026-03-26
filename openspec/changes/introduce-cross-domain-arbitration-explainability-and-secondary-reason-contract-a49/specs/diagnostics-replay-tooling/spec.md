## ADDED Requirements

### Requirement: Replay tooling SHALL validate arbitration explainability fixtures
Diagnostics replay tooling MUST validate arbitration explainability fixtures, including secondary reason ordering, bounded count, remediation hint taxonomy, and rule-version stability.

Replay drift classes MUST include at minimum:
- `secondary_order_drift`
- `secondary_count_drift`
- `hint_taxonomy_drift`
- `rule_version_drift`

#### Scenario: Explainability fixture matches canonical output
- **WHEN** expected explainability fixture matches normalized replay output
- **THEN** replay validation passes deterministically

#### Scenario: Explainability fixture detects secondary-order drift
- **WHEN** replay output secondary reason ordering differs from canonical expectation
- **THEN** replay validation fails with deterministic `secondary_order_drift` classification
