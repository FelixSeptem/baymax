## ADDED Requirements

### Requirement: Scheduler A2A adapter SHALL use shared synchronous invocation contract
Scheduler-managed A2A dispatch adapter MUST use shared synchronous invocation contract for submit/wait/normalize behavior instead of path-local duplicated logic.

#### Scenario: Scheduler claim executes remote child through A2A
- **WHEN** scheduler worker executes claimed task through A2A bridge
- **THEN** adapter invokes shared synchronous invocation and receives normalized terminal mapping

### Requirement: Scheduler retryability mapping SHALL follow normalized transport classification
Scheduler retryability decision for A2A execution MUST be derived from normalized error-layer classification where transport-layer failures are retryable and non-transport failures are non-retryable by default.

#### Scenario: Scheduler receives protocol-layer failure
- **WHEN** shared synchronous invocation returns protocol-layer failure
- **THEN** scheduler marks commit as failed and non-retryable

### Requirement: Scheduler canceled remote terminal SHALL converge deterministically
When remote A2A terminal state is `canceled`, scheduler terminal commit path MUST converge deterministically under existing terminal commit contract.

#### Scenario: A2A terminal status is canceled during scheduler-managed execution
- **WHEN** scheduler adapter receives canceled terminal from A2A
- **THEN** scheduler produces deterministic terminal commit outcome compatible with existing commit API
