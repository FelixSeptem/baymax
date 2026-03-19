## ADDED Requirements

### Requirement: Quality gate SHALL validate adapter template and migration-doc consistency
The repository quality validation flow MUST verify that external adapter template documentation and migration mapping indexes are synchronized with declared navigation entries.

Validation MUST run through existing docs consistency and contribution check paths.

#### Scenario: Docs index misses adapter mapping entry
- **WHEN** adapter template docs are added or renamed without index synchronization
- **THEN** docs consistency or contribution checks fail and block validation

#### Scenario: Migration mapping link is stale
- **WHEN** migration mapping reference points to missing or moved document path
- **THEN** validation fails with explicit documentation consistency error

### Requirement: Quality gate SHALL keep traceability for adapter migration guidance
Mainline documentation checks MUST preserve traceability between adapter templates, migration mapping docs, and repository entry points.

#### Scenario: Maintainer audits adapter onboarding coverage
- **WHEN** maintainer reviews contribution check outputs and docs index
- **THEN** template and migration mapping paths are traceable from repository documentation entry points

