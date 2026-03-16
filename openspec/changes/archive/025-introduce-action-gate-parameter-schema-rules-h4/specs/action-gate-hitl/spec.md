## MODIFIED Requirements

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
