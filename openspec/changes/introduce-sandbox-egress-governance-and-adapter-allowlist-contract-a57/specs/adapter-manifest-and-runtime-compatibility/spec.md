## ADDED Requirements

### Requirement: Adapter manifest SHALL declare allowlist identity metadata
Adapter manifest schema MUST support allowlist identity metadata for activation governance.

Minimum required fields for this milestone:
- `allowlist.adapter_id`
- `allowlist.publisher`
- `allowlist.version`
- `allowlist.signature_status`

#### Scenario: Manifest misses required allowlist field
- **WHEN** adapter manifest omits `allowlist.publisher`
- **THEN** manifest validation fails fast before activation

#### Scenario: Manifest includes complete allowlist metadata
- **WHEN** adapter manifest provides all required allowlist identity fields
- **THEN** runtime can evaluate activation eligibility deterministically

### Requirement: Runtime SHALL enforce manifest allowlist compatibility before activation
Runtime MUST validate adapter manifest allowlist metadata against effective allowlist policy before loading adapter into active runtime graph.

#### Scenario: Manifest identity not allowed by policy
- **WHEN** runtime activation evaluates adapter metadata not present in effective allowlist
- **THEN** runtime blocks activation with canonical allowlist classification

#### Scenario: Manifest identity allowed by policy
- **WHEN** runtime activation evaluates adapter metadata matching effective allowlist policy
- **THEN** adapter activation proceeds without allowlist conflict
