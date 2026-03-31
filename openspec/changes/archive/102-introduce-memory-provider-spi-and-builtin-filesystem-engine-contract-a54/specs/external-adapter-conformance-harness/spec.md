## ADDED Requirements

### Requirement: Conformance harness SHALL include mainstream memory adapter matrix suites
External adapter conformance harness MUST include memory adapter matrix suites for:
- `mem0`
- `zep`
- `openviking`
- `generic`

Memory matrix execution MUST be offline deterministic by default and MUST NOT require external network or provider credentials.

#### Scenario: CI executes memory matrix in disconnected environment
- **WHEN** conformance harness runs memory adapter suites without network access
- **THEN** suites execute deterministically using fixtures or fakes and produce stable pass fail semantics

#### Scenario: Contributor declares unsupported memory profile in matrix
- **WHEN** memory conformance matrix input references unsupported profile id
- **THEN** harness fails fast with deterministic profile-unknown classification

### Requirement: Memory conformance suites SHALL validate canonical operation and error taxonomy semantics
Memory adapter conformance suites MUST validate canonical `Query/Upsert/Delete` semantics, required and optional capability behavior, and normalized error taxonomy mapping.

#### Scenario: Adapter misses required memory operation
- **WHEN** adapter manifest declares required memory operation but implementation does not satisfy it
- **THEN** conformance fails with deterministic required-capability-missing classification

#### Scenario: Adapter misses optional memory capability
- **WHEN** adapter lacks optional capability such as TTL or metadata filter
- **THEN** conformance verifies deterministic downgrade path with canonical downgrade reason

### Requirement: Memory conformance SHALL assert fallback and Run Stream semantic equivalence
Memory conformance harness MUST assert:
- fallback policy behavior (`fail_fast|degrade_to_builtin|degrade_without_memory`),
- Run and Stream semantic equivalence under equivalent memory inputs.

#### Scenario: Equivalent Run and Stream memory sequence
- **WHEN** equivalent memory request sequence runs through Run and Stream suites
- **THEN** conformance output remains semantically equivalent for operation outcomes and reason taxonomy

#### Scenario: Fallback policy behavior drifts from canonical expectation
- **WHEN** memory fallback execution diverges from canonical fixture expectation
- **THEN** conformance fails with deterministic fallback-drift classification
