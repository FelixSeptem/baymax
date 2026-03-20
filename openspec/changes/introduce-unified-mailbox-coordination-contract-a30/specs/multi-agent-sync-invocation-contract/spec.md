## MODIFIED Requirements

### Requirement: Runtime SHALL provide a shared synchronous invocation contract
The runtime MUST provide a shared synchronous invocation contract for remote execution through mailbox command->result semantics so orchestration modules consume one canonical path.

The canonical synchronous path MUST publish `command` envelope and wait for correlated terminal `result` envelope, instead of path-local direct `submit + wait` coupling.

#### Scenario: Module uses mailbox-backed shared synchronous invocation
- **WHEN** workflow, teams, composer, or scheduler dispatches a remote task synchronously
- **THEN** the module uses mailbox command->result contract and receives terminal result or explicit error

#### Scenario: Remote task remains non-terminal during polling
- **WHEN** correlated mailbox result is not terminal yet
- **THEN** invocation keeps waiting until terminal result or context termination

## REMOVED Requirements

### Requirement: Shared synchronous invocation SHALL keep callback compatibility optional
**Reason**: Synchronous invocation is converged to mailbox command->result contract and no longer uses callback compatibility as contract surface.
**Migration**: Migrate callers to mailbox-based sync invocation API and remove callback-based sync adaptation from primary path.
