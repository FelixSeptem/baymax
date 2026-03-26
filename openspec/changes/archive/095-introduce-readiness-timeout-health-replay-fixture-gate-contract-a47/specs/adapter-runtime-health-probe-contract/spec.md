## ADDED Requirements

### Requirement: Adapter-health governance output SHALL align with composite replay fixtures
Adapter-health probe and governance output MUST remain alignable with A47 composite replay fixtures across status, reason taxonomy, and circuit-state observability.

Composite fixture assertions MUST cover:
- adapter status (`healthy|degraded|unavailable`),
- governance state (`closed|open|half_open`),
- readiness mapping for required/optional adapter paths.

#### Scenario: Composite fixture validates optional adapter degraded path
- **WHEN** fixture models optional adapter unavailable under non-strict readiness
- **THEN** replay assertion confirms degraded classification with canonical adapter-health reason taxonomy

#### Scenario: Composite fixture validates circuit-open blocking path
- **WHEN** fixture models required adapter unavailable with circuit open under strict readiness
- **THEN** replay assertion confirms blocked classification and canonical adapter-health code mapping
