## ADDED Requirements

### Requirement: Runner SHALL enforce Action Gate before high-risk tool execution
The runtime MUST evaluate an Action Gate decision before dispatching high-risk tool actions. Gate decision MUST support `allow`, `require_confirm`, and `deny` semantics. In H2 scope, high-risk detection MUST be based on tool name and keyword rules.

#### Scenario: High-risk tool requires confirmation
- **WHEN** a tool call matches configured high-risk tool name or keyword rule
- **THEN** runner evaluates gate decision as `require_confirm` and waits for resolver outcome before tool dispatch

#### Scenario: Non-risk tool bypasses gate confirmation
- **WHEN** a tool call does not match high-risk rule set
- **THEN** runner continues tool dispatch without confirmation blocking

### Requirement: Action Gate default policy SHALL be require_confirm
The runtime MUST default Action Gate policy to `require_confirm`. If confirmation is required but resolver is not configured, runtime MUST treat the outcome as denied and MUST NOT execute the tool action.

#### Scenario: Confirmation required without resolver
- **WHEN** gate decision is `require_confirm` and no resolver is configured
- **THEN** runner denies execution and returns fail-fast result with normalized error classification

### Requirement: Action Gate timeout SHALL deny execution
When confirmation resolver times out, runtime MUST classify gate outcome as timeout-denied and MUST block tool execution.

#### Scenario: Resolver timeout
- **WHEN** resolver does not return within configured timeout
- **THEN** runner marks gate outcome as timeout and denies tool execution

### Requirement: Run and Stream SHALL keep Action Gate semantic equivalence
Run and Stream paths MUST produce semantically equivalent Action Gate outcomes for the same input and configuration, including allow, deny, and timeout cases.

#### Scenario: Equivalent deny behavior in Run and Stream
- **WHEN** the same high-risk tool request is executed in Run and Stream with deny outcome
- **THEN** both paths terminate the gated action with equivalent status and error semantics

#### Scenario: Equivalent timeout behavior in Run and Stream
- **WHEN** confirmation resolver times out in both Run and Stream for equivalent inputs
- **THEN** both paths produce timeout-deny semantics with equivalent observability fields
