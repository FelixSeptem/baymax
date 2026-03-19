## ADDED Requirements

### Requirement: Composer recovery SHALL enforce next_attempt_only resume boundary
Composer recovery flow MUST enforce next-attempt-only boundary when restoring scheduler/workflow/A2A state.

#### Scenario: Composer restores run with active in-flight child attempts
- **WHEN** composer recovery resumes run containing in-flight child attempts
- **THEN** current attempt semantics remain stable and updated controls apply on next attempt boundary

### Requirement: Composer recovery SHALL preserve no_rewind semantics for completed child tasks
Composer recovery MUST not dispatch already terminal child tasks again after restore.

#### Scenario: Restored snapshot contains completed child tasks
- **WHEN** composer finishes recovery initialization and resumes orchestration
- **THEN** completed child tasks remain terminal and are excluded from new dispatch

### Requirement: Composer timeout reentry semantics SHALL remain bounded and deterministic
Composer-managed timeout reentry after restore MUST follow bounded single reentry policy and deterministic fail convergence.

#### Scenario: Restored child execution times out beyond reentry budget
- **WHEN** child task exceeds configured timeout reentry budget after recovery
- **THEN** composer emits deterministic terminal failure and no further reentry is attempted
