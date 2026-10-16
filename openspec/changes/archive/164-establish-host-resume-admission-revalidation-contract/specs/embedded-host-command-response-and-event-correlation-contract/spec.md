## ADDED Requirements

### Requirement: Action-resume SHALL delegate required revalidation to the source boundary

The existing action-resume command MUST retain general command readiness and authorization checks and delegate required checkpoint/business revalidation to the same source-owned resume boundary used by direct native API callers. A command adapter MUST NOT construct authority from caller digests, bypass an installed source verifier, maintain a parallel checkpoint registry, or treat command/message deduplication as business resume admission. Source rejection classifications MUST remain bounded and preserve the existing command-response-before-event delivery contract; no rejected resume may synthesize resolution or terminal events.

#### Scenario: Host and direct callers share resume guard
- **WHEN** an enabled source guard rejects a current revision or authority for an action-resume command or direct API call
- **THEN** both paths reject before source decision commit with equivalent resume semantics

#### Scenario: Generic command authorization passes but resume binding is stale
- **WHEN** command readiness and authorization pass but source checkpoint/business revalidation fails
- **THEN** the command reports rejection and no resolution, terminal or execution effect is emitted

#### Scenario: New message repeats an already admitted decision
- **WHEN** a new transport message carries an exact duplicate of a source-retained action-resume decision
- **THEN** source idempotency prevents repeated events or execution regardless of connection-scoped message tracking

#### Scenario: Command acceptance remains distinct from business commit
- **WHEN** a command delegates an accepted confirm and receives source events
- **THEN** response delivery remains ahead of those events and neither command acceptance nor runner completion is represented as host business commit
