## ADDED Requirements

### Requirement: Composer SHALL expose operation-profile selection in managed orchestration requests
Composer MUST allow managed orchestration requests to specify operation profile selection and request-level timeout overrides through library-level API.

Composer MUST validate profile selection against canonical profile set before dispatching to scheduler.

#### Scenario: Host submits managed request with explicit profile
- **WHEN** caller invokes composer with `operation_profile=interactive`
- **THEN** composer accepts request, validates profile, and forwards resolved timeout context to scheduler

#### Scenario: Host submits managed request with unsupported profile
- **WHEN** caller invokes composer with non-canonical profile value
- **THEN** composer fails fast and does not create child dispatch attempt

### Requirement: Composer SHALL propagate timeout-resolution summary as additive diagnostics context
Composer run summary MUST include additive timeout-resolution context sufficient to explain effective profile and parent-child convergence outcome without breaking existing diagnostics consumers.

Minimum summary context:
- effective operation profile
- final child timeout budget
- resolution source classification

#### Scenario: Consumer inspects composer diagnostics after child dispatch
- **WHEN** managed run performs child dispatch with profile-based timeout resolution
- **THEN** composer diagnostics include additive timeout-resolution summary fields

#### Scenario: Equivalent Run and Stream execution paths
- **WHEN** equivalent inputs execute through Run and Stream with same profile and overrides
- **THEN** composer timeout-resolution summary semantics remain equivalent across modes
