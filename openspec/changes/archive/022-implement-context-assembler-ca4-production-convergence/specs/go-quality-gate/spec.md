## ADDED Requirements

### Requirement: Quality gate SHALL include CA4 benchmark regression checks
The standard validation flow MUST include CA4-related benchmark checks evaluated by relative percentage thresholds, including P95 latency constraints.

#### Scenario: CA4 benchmark regression exceeds threshold
- **WHEN** candidate benchmark result exceeds configured relative degradation or P95 threshold
- **THEN** validation fails and change cannot be completed until regression is mitigated or explicitly re-baselined

### Requirement: CA4 benchmark policy SHALL align with documented performance rules
CA4 benchmark acceptance criteria MUST align with repository performance policy and remain documented for local and CI execution parity.

#### Scenario: Contributor runs CA4 performance validation
- **WHEN** contributor follows documented commands
- **THEN** contributor can reproduce the same pass/fail semantics locally and in CI
