## MODIFIED Requirements

### Requirement: Runtime SHALL provide a shared synchronous invocation contract
The runtime MUST provide a shared synchronous invocation contract for remote execution through mailbox command->result semantics so orchestration modules consume one canonical path.

The canonical synchronous path MUST publish `command` envelope and wait for correlated terminal `result` envelope, instead of path-local direct `submit + wait` coupling.

Public synchronous invocation entrypoints MUST be mailbox-backed canonical APIs only, and legacy direct invoke entrypoints MUST NOT remain as supported public contract surface.

#### Scenario: Module uses mailbox-backed shared synchronous invocation
- **WHEN** workflow, teams, composer, or scheduler dispatches a remote task synchronously
- **THEN** the module uses mailbox command->result contract and receives terminal result or explicit error

#### Scenario: Remote task remains non-terminal during polling
- **WHEN** correlated mailbox result is not terminal yet
- **THEN** invocation keeps waiting until terminal result or context termination

#### Scenario: Maintainer audits synchronous invocation public entrypoints
- **WHEN** repository exposes synchronous multi-agent invocation APIs
- **THEN** only mailbox-backed canonical entrypoints are treated as supported contract surface
