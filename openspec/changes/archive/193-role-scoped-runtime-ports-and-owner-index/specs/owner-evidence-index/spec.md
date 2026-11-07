## Purpose

Maintain a verifiable, read-only mapping from enrolled public semantics to
their unique owners, contract evidence, documentation and rollback boundaries.

## ADDED Requirements

### Requirement: Enrolled semantics SHALL have a unique traceable owner
The repository MUST maintain a versioned static canonical index. Each enrolled
semantic MUST have a unique stable ID, exactly one precise semantic owner, an
existing OpenSpec main specification, existing unit or integration tests,
existing public documentation, a dedicated gate, and explicit rollback and
non-goal boundaries. Its owner MUST agree with the documentation source map.
Distinct owner domains MUST be represented as separate semantic entries.

#### Scenario: Maintainer traces a semantic
- **WHEN** a maintainer selects an enrolled semantic
- **THEN** the index identifies its unique owner, specification, tests, gate,
  documentation and rollback/non-goal boundary without inferring ownership

#### Scenario: Index contains ambiguous ownership or duplicate identifiers
- **WHEN** an entry omits its owner, names multiple or broad owners, conflicts
  with the source map, or duplicates another semantic ID
- **THEN** validation fails and blocks the standard quality gate

### Requirement: Replay evidence SHALL state its applicability
Each enrolled semantic MUST reference an existing replay fixture or declare
replay not applicable with a concrete non-empty reason. A fixture or replay
result MUST NOT be interpreted as authorization, execution, recovery or effect
evidence.

#### Scenario: Replay fixture is declared
- **WHEN** an entry declares fixture replay
- **THEN** validation requires its repository-relative fixture path to exist

#### Scenario: Replay is not applicable
- **WHEN** an entry declares replay not applicable
- **THEN** validation requires a concrete reason and rejects an empty reason

### Requirement: Evidence validation SHALL be a blocking offline gate
The validator MUST reject malformed or unsupported index data, missing required
evidence paths, missing dedicated gates and dedicated gates not wired into both
standard quality paths. The repository MUST run the dedicated validation from
both shell and PowerShell quality gates, and retain existing roadmap/OpenSpec
active and archive status consistency validation.

#### Scenario: Evidence path or gate is absent
- **WHEN** an enrolled spec, test, fixture, gate or documentation path is absent,
  or the dedicated gate is not wired into either quality path
- **THEN** the relevant validation and standard quality gate fail

#### Scenario: Roadmap status contradicts OpenSpec
- **WHEN** a change is active or archived but its roadmap status disagrees
- **THEN** the existing roadmap status consistency gate fails

### Requirement: Index SHALL NOT own runtime decisions
The index and validator MUST be static and read-only. They MUST NOT create a
runtime registry, service locator, execution control plane, source owner,
authorization, recovery, effect, diagnostic writer or new runtime port. Runtime
port narrowing requires separate evidence of at least two distinct callers
crossing owner boundaries through the same broad interface.

#### Scenario: Evidence is inspected
- **WHEN** a contributor reads or validates the index
- **THEN** no runtime execution, source mutation, authorization or recovery is
  performed or implied
