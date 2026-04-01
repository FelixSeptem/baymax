## ADDED Requirements

### Requirement: Admission guard SHALL consume canonical policy precedence output
Runtime readiness-admission guard MUST consume policy precedence winner output and apply deterministic deny/allow mapping without entrypoint-specific drift.

#### Scenario: Admission receives policy winner from sandbox egress stage
- **WHEN** policy winner stage is `sandbox_egress`
- **THEN** admission returns deterministic deny classification and preserves stage/source semantics

#### Scenario: Equivalent Run and Stream admission mapping
- **WHEN** equivalent Run and Stream requests consume the same policy winner output
- **THEN** both paths return semantically equivalent admission decision and reason taxonomy

### Requirement: Admission deny path SHALL remain side-effect-free under policy precedence
When policy precedence yields deny, admission MUST reject execution before scheduler/mailbox/tool dispatch side effects.

#### Scenario: Policy winner is blocking before execution
- **WHEN** admission receives blocking winner stage from policy evaluator
- **THEN** runtime denies request and does not emit runtime execution side effects

#### Scenario: Policy winner changes after hot reload rollback
- **WHEN** invalid hot reload is rejected and previous snapshot is restored
- **THEN** admission uses restored policy winner semantics deterministically

### Requirement: Admission explainability SHALL preserve decision trace fields
Admission response MUST preserve canonical decision-trace fields for policy precedence winners.

#### Scenario: Deny response includes decision trace
- **WHEN** admission denies due to policy precedence
- **THEN** response includes canonical `deny_source` and `winner_stage`

#### Scenario: Tie-break deny includes tie-break reason
- **WHEN** deny winner is selected via same-stage tie-break
- **THEN** response includes canonical `tie_break_reason` without remapping drift
