## ADDED Requirements

### Requirement: Runner SHALL support native clarification HITL lifecycle in H3
The runtime MUST support a native clarification lifecycle for human-in-the-loop interactions during execution, including `await_user`, `resumed`, and `canceled_by_user` outcomes, within single-process scope.

#### Scenario: Clarification is requested during run
- **WHEN** agent determines required user information is missing
- **THEN** runner enters `await_user` state and emits structured clarification request payload

#### Scenario: Clarification answer resumes execution
- **WHEN** resolver returns user clarification data within timeout
- **THEN** runner marks lifecycle as `resumed` and continues execution with injected clarification context

#### Scenario: Clarification timeout cancels run
- **WHEN** clarification wait exceeds configured timeout
- **THEN** runner marks lifecycle as `canceled_by_user` and terminates run fail-fast

### Requirement: H3 clarification integration SHALL remain library-first
The runtime MUST expose clarification interaction via library interfaces and MUST NOT require CLI support.

#### Scenario: Host application provides clarification resolver
- **WHEN** host application injects clarification resolver callback
- **THEN** runner uses callback for HITL interaction without depending on CLI

### Requirement: Run and Stream SHALL keep H3 semantic equivalence
Run and Stream paths MUST produce semantically equivalent clarification outcomes for the same input/configuration.

#### Scenario: Equivalent await/resume semantics
- **WHEN** Run and Stream process equivalent clarification-required requests
- **THEN** both paths emit equivalent await/resume lifecycle semantics

#### Scenario: Equivalent timeout-cancel semantics
- **WHEN** Run and Stream both hit clarification timeout
- **THEN** both paths terminate with equivalent `canceled_by_user` semantics and error classification
