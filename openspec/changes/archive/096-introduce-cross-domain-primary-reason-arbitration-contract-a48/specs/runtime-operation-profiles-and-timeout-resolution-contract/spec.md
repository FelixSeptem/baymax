## ADDED Requirements

### Requirement: Timeout-resolution outcomes SHALL retain top arbitration precedence for exhausted-budget and reject paths
Timeout-resolution exhausted-budget and reject outcomes MUST retain top precedence in cross-domain primary-reason arbitration.

This precedence MUST apply consistently across nested execution and parent-budget clamp/reject contexts.

#### Scenario: Parent budget exhausted alongside readiness blocked
- **WHEN** parent budget is exhausted and readiness blocked finding co-exists
- **THEN** arbitration selects timeout reject as primary reason

#### Scenario: Timeout clamp without reject co-exists with degraded readiness
- **WHEN** timeout path clamps child budget without reject and degraded readiness co-exists
- **THEN** arbitration follows canonical precedence and emits deterministic primary output
