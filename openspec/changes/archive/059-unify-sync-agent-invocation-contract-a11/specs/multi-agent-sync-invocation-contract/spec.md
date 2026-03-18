## ADDED Requirements

### Requirement: Runtime SHALL provide a shared synchronous invocation contract
The runtime MUST provide a shared synchronous invocation contract for A2A-based remote execution so orchestration modules can use one canonical `submit + wait + normalize` path.

#### Scenario: Module uses shared synchronous invocation
- **WHEN** workflow, teams, composer, or scheduler dispatches a remote task
- **THEN** the module consumes the same shared synchronous invocation contract instead of module-local duplicated flow

### Requirement: Shared synchronous invocation SHALL return terminal outcome only
Shared synchronous invocation MUST return either a terminal task outcome or an explicit error, and MUST NOT expose non-terminal intermediate statuses as final return.

#### Scenario: Remote task is still running during polling
- **WHEN** shared invocation polls a non-terminal task status
- **THEN** invocation continues waiting until terminal status or context termination

### Requirement: Shared synchronous invocation SHALL enforce context-first cancellation semantics
Shared synchronous invocation MUST treat caller context cancellation/timeout as authoritative and MUST stop waiting immediately when context is done.

#### Scenario: Parent context is canceled while waiting
- **WHEN** caller context is canceled before remote task reaches terminal status
- **THEN** invocation exits with context-derived error and does not continue polling

### Requirement: Shared synchronous invocation SHALL normalize error taxonomy and retryability
Shared synchronous invocation MUST expose normalized error taxonomy and retryability hints so scheduler/composer paths can converge terminal mapping consistently.

#### Scenario: Remote call fails with transport-layer error
- **WHEN** submit or wait path returns a transport-class failure
- **THEN** invocation result includes normalized transport layer classification and retryable hint

### Requirement: Shared synchronous invocation SHALL keep callback compatibility optional
Shared synchronous invocation MUST keep callback hook optional for compatibility and MUST NOT require callback registration to complete synchronous call.

#### Scenario: Invocation executes without callback
- **WHEN** caller does not provide callback handler
- **THEN** invocation still completes with terminal result or explicit error

### Requirement: Shared synchronous invocation SHALL preserve Run/Stream semantic equivalence
For equivalent requests and effective configuration, shared synchronous invocation outcomes consumed by Run and Stream paths MUST remain semantically equivalent.

#### Scenario: Equivalent request through Run and Stream paths
- **WHEN** same remote task semantics are invoked by Run and Stream flows
- **THEN** terminal category and normalized error-layer semantics remain equivalent
