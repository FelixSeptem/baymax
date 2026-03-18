## ADDED Requirements

### Requirement: Composed remote execution SHALL use shared synchronous invocation contract
Composed orchestration paths that dispatch remote peer execution MUST use the shared synchronous invocation contract for `submit/wait/normalize` behavior.

#### Scenario: Workflow and Teams remote path run in one composed request
- **WHEN** composed orchestration dispatches remote execution from workflow and teams modules
- **THEN** both paths use shared synchronous invocation semantics and expose consistent terminal behavior

### Requirement: Composed synchronous remote execution SHALL preserve deterministic convergence
Composed remote execution using shared synchronous invocation MUST preserve deterministic terminal convergence for equivalent inputs and effective configuration.

#### Scenario: Equivalent composed request executes twice
- **WHEN** same composed request with remote execution runs twice
- **THEN** terminal status category and normalized error-layer mapping remain semantically equivalent
