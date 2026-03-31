## ADDED Requirements

### Requirement: Sandbox action resolution SHALL remain deterministic across ReAct iterations
For equivalent selector, effective sandbox policy, and runtime snapshot, sandbox action resolution (`host|sandbox|deny`) MUST remain deterministic across multiple ReAct loop iterations.

#### Scenario: Equivalent tool selector appears in multiple ReAct iterations
- **WHEN** the same tool selector is invoked in successive ReAct iterations with unchanged effective config
- **THEN** runtime resolves semantically equivalent sandbox action on each iteration

#### Scenario: Equivalent selector appears in Run and Stream ReAct loops
- **WHEN** equivalent Run and Stream ReAct loops invoke the same tool selector under unchanged config
- **THEN** both paths resolve semantically equivalent sandbox action classification

### Requirement: Sandbox fallback behavior in ReAct loops SHALL preserve canonical taxonomy
When sandbox execution fails during ReAct loop, runtime MUST apply configured fallback policy deterministically and MUST preserve canonical fallback reason taxonomy per iteration.

#### Scenario: ReAct iteration hits sandbox launch failure with allow-and-record fallback
- **WHEN** sandbox launch fails during a ReAct iteration and fallback policy is `allow_and_record`
- **THEN** runtime executes host fallback and records canonical fallback reason for that iteration

#### Scenario: ReAct iteration hits sandbox launch failure with deny fallback
- **WHEN** sandbox launch fails during a ReAct iteration and fallback policy is `deny`
- **THEN** runtime fail-fast denies the tool call and maps loop termination to canonical sandbox failure classification

### Requirement: Sandbox capability mismatch in ReAct loop SHALL terminate deterministically
If sandbox required capability negotiation fails for a ReAct tool call in enforce mode, runtime MUST terminate deterministically with canonical capability-mismatch semantics and MUST NOT silently downgrade execution path.

#### Scenario: ReAct tool call requires unsupported sandbox capability
- **WHEN** sandbox capability probe reports missing required capability for dispatched ReAct tool call
- **THEN** runtime terminates tool step with canonical `sandbox.capability_mismatch` classification

#### Scenario: Equivalent capability mismatch under Run and Stream
- **WHEN** equivalent Run and Stream ReAct loops encounter same capability mismatch
- **THEN** both paths produce semantically equivalent terminal classification and fallback usage semantics
