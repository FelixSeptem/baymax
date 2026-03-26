## ADDED Requirements

### Requirement: Replay fixture gate SHALL define canonical cross-domain scenario matrix
Replay fixture gate MUST define a canonical cross-domain matrix that covers readiness, timeout-resolution, and adapter-health governance combinations.

Minimum matrix dimensions:
- readiness status and policy path (`ready|degraded|blocked`, strict/non-strict)
- timeout resolution path (`profile -> domain -> request`, parent-budget clamp/reject)
- adapter-health path (`healthy|degraded|unavailable`, `closed|open|half_open`)
- execution mode parity (Run and Stream)

#### Scenario: Maintainer executes canonical matrix suite
- **WHEN** replay fixture suites run for A47 cross-domain matrix
- **THEN** each matrix dimension is covered by at least one deterministic fixture case

#### Scenario: Matrix case is missing required cross-domain axis
- **WHEN** fixture set omits one required dimension
- **THEN** fixture validation fails and blocks quality gate

### Requirement: Replay assertions SHALL be deterministic and replay-idempotent
For equivalent fixture input and equivalent runtime normalization rules, replay assertions MUST produce deterministic results and MUST preserve replay-idempotent logical aggregates.

#### Scenario: Equivalent fixture replay is executed repeatedly
- **WHEN** identical fixture case is replayed more than once
- **THEN** assertion result and logical aggregate counters remain stable after first logical ingestion

#### Scenario: Equivalent fixture is validated across Run and Stream
- **WHEN** same semantic fixture case is mapped to Run and Stream paths
- **THEN** normalized assertion outcome remains semantically equivalent

### Requirement: Fixture gate drift policy SHALL fail fast on canonical semantic mismatch
If fixture output diverges from canonical semantic fields (status/code/reason taxonomy/timeout source/circuit state), fixture gate MUST fail fast and block merge.

#### Scenario: Canonical reason taxonomy drifts
- **WHEN** fixture assertion detects non-canonical reason or finding code
- **THEN** fixture gate exits non-zero and blocks validation

#### Scenario: Non-semantic additive nullable field is absent
- **WHEN** additive nullable optional field is absent while canonical semantic fields match
- **THEN** fixture assertion remains pass according to compatibility-window policy
