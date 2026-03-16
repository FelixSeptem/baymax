# action-gate-hitl Specification

## Purpose
Defines the H2 Action Gate behavior for human-in-the-loop control before high-risk tool execution, including default policy, timeout-deny semantics, and Run/Stream semantic equivalence.
## Requirements
### Requirement: Runner SHALL enforce Action Gate before high-risk tool execution
The runtime MUST evaluate an Action Gate decision before dispatching high-risk tool actions. Gate decision MUST support `allow`, `require_confirm`, and `deny` semantics. High-risk detection MUST support parameter-level rule evaluation in addition to existing tool-name and keyword rules.

Parameter-rule evaluation MUST execute before tool-name and keyword rule resolution. If a parameter rule matches, its action resolution MUST be used (explicit rule action first; otherwise inherit global policy).

#### Scenario: High-risk tool requires confirmation
- **WHEN** a tool call matches configured high-risk rule
- **THEN** runner evaluates gate decision as `require_confirm` and waits for resolver outcome before tool dispatch

#### Scenario: Non-risk tool bypasses gate confirmation
- **WHEN** a tool call does not match high-risk rule set
- **THEN** runner continues tool dispatch without confirmation blocking

#### Scenario: Parameter-rule match takes priority over keyword rule
- **WHEN** a tool call matches both parameter rule and keyword rule with different decisions
- **THEN** runner applies parameter-rule decision as the effective gate outcome

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

For parameter-level rules, Run and Stream MUST preserve equivalent outcomes for matched and unmatched conditions, including composite condition evaluation and inherited-action behavior.

#### Scenario: Equivalent deny behavior in Run and Stream
- **WHEN** the same high-risk tool request is executed in Run and Stream with deny outcome
- **THEN** both paths terminate the gated action with equivalent status and error semantics

#### Scenario: Equivalent timeout behavior in Run and Stream
- **WHEN** confirmation resolver times out in both Run and Stream for equivalent inputs
- **THEN** both paths produce timeout-deny semantics with equivalent observability fields

#### Scenario: Equivalent parameter-rule inherited action in Run and Stream
- **WHEN** equivalent requests match a parameter rule without explicit action in both Run and Stream
- **THEN** both paths inherit global policy and produce equivalent gate outcomes

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

