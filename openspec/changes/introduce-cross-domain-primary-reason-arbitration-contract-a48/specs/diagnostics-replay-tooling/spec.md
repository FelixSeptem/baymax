## ADDED Requirements

### Requirement: Replay tooling SHALL validate cross-domain primary-reason arbitration fixtures
Diagnostics replay tooling MUST support cross-domain primary-reason arbitration fixtures and MUST return deterministic drift classification on mismatch.

Drift classes MUST include at minimum:
- precedence drift
- tie-break drift
- taxonomy drift

#### Scenario: Replay fixture matches canonical arbitration output
- **WHEN** fixture expected arbitration output matches normalized actual output
- **THEN** replay validation succeeds with deterministic pass result

#### Scenario: Replay fixture detects precedence drift
- **WHEN** actual primary reason violates canonical precedence order
- **THEN** replay validation fails with deterministic precedence-drift classification
