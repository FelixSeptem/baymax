## ADDED Requirements

### Requirement: Readiness preflight SHALL include arbitration explainability alignment
Readiness preflight output MUST preserve alignment between primary reason and explainability metadata, including bounded secondary reasons and remediation hint taxonomy.

#### Scenario: Preflight returns blocked with explainability payload
- **WHEN** readiness preflight produces blocked status and arbitration metadata
- **THEN** output includes canonical primary reason plus bounded secondary reasons and remediation hint fields

#### Scenario: Equivalent preflight inputs are evaluated repeatedly
- **WHEN** runtime runs repeated preflight with unchanged inputs
- **THEN** explainability output remains semantically equivalent and deterministically ordered
