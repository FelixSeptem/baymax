## ADDED Requirements

### Requirement: Runner SHALL enforce namespace+tool permission policy before tool dispatch
The runtime MUST evaluate tool permission policy keyed by `namespace+tool` before executing a tool call.

If a matching policy resolves to `deny`, execution MUST be fail-fast denied and the tool MUST NOT be dispatched.

#### Scenario: Permission allow for matched namespace+tool
- **WHEN** a tool call matches configured `namespace+tool` policy with `allow`
- **THEN** runtime permits tool dispatch and continues normal execution flow

#### Scenario: Permission deny for matched namespace+tool
- **WHEN** a tool call matches configured `namespace+tool` policy with `deny`
- **THEN** runtime fail-fast denies execution and does not dispatch the tool

### Requirement: Runner SHALL enforce process-scoped rate limit on namespace+tool
The runtime MUST enforce rate limits using process-scoped counters keyed by `namespace+tool` and configured time windows.

When rate limit is exceeded, runtime MUST return `deny` outcome and MUST NOT execute the tool call.

#### Scenario: Tool call within configured quota
- **WHEN** call count for a `namespace+tool` key remains within configured limit for the current window
- **THEN** runtime allows tool dispatch

#### Scenario: Tool call exceeds configured quota
- **WHEN** call count for a `namespace+tool` key exceeds configured limit in the current window
- **THEN** runtime returns fail-fast `deny` and blocks tool dispatch

### Requirement: Tool security governance SHALL emit normalized deny diagnostics
Permission-denied and rate-limit-denied outcomes MUST emit normalized diagnostics fields and reason codes for audit and triage.

#### Scenario: Permission deny is recorded
- **WHEN** runtime denies a tool call by permission policy
- **THEN** diagnostics include policy type `permission`, matched `namespace+tool`, and normalized deny reason code

#### Scenario: Rate limit deny is recorded
- **WHEN** runtime denies a tool call due to rate-limit exceedance
- **THEN** diagnostics include policy type `rate_limit`, matched `namespace+tool`, window context, and normalized deny reason code

### Requirement: Run and Stream SHALL keep tool-security governance semantic equivalence
For equivalent input and effective configuration, Run and Stream MUST produce semantically equivalent permission and rate-limit decisions.

#### Scenario: Equivalent permission deny in Run and Stream
- **WHEN** equivalent requests hit the same deny permission policy in both Run and Stream
- **THEN** both paths deny tool execution with semantically equivalent outcome and diagnostics semantics

#### Scenario: Equivalent rate-limit deny in Run and Stream
- **WHEN** equivalent request patterns exceed the same process-scoped rate limit in both Run and Stream
- **THEN** both paths deny tool execution with semantically equivalent outcome and diagnostics semantics