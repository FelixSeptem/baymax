## ADDED Requirements

### Requirement: CA4 threshold precedence SHALL be explicit and testable
Memory pressure control MUST define and test precedence of global and stage-level thresholds, including conflict resolution for mixed threshold triggers.

#### Scenario: Global and stage thresholds both configured
- **WHEN** stage-level thresholds are present for active stage
- **THEN** active stage uses stage-level thresholds and does not mix global values for that stage

#### Scenario: Trigger conflict during pressure evaluation
- **WHEN** percent and absolute threshold evaluations produce different zones
- **THEN** higher-pressure zone is selected consistently and diagnostics include trigger source

### Requirement: CA4 counting fallback SHALL preserve pressure safety
If provider counting is unavailable, memory pressure control MUST still produce stable zone computation through local tokenizer and fallback estimator paths.

#### Scenario: Provider counting unavailable in sdk_preferred mode
- **WHEN** provider counting fails in pressure evaluation
- **THEN** fallback estimates are used and pressure safety actions continue to work
