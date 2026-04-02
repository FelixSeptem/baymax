## ADDED Requirements

### Requirement: Admission guard SHALL evaluate budget-admission decision before managed execution
Runtime readiness-admission guard MUST evaluate unified budget-admission output before managed Run/Stream execution starts.

Budget decision mapping MUST support:
- `allow`
- `degrade`
- `deny`

When budget decision is `deny`, admission deny path MUST remain side-effect free.

#### Scenario: Admission receives budget decision allow
- **WHEN** readiness passes and budget decision is `allow`
- **THEN** managed execution proceeds without degrade action

#### Scenario: Admission receives budget decision deny
- **WHEN** readiness passes but budget decision is `deny`
- **THEN** admission denies execution with deterministic budget classification and no scheduler/mailbox/task side effects

### Requirement: Admission explainability SHALL preserve budget decision fields alongside policy fields
Admission outputs MUST preserve budget decision explainability without remapping drift and MUST NOT redefine A58 policy fields.

Minimum preserved fields:
- `budget_decision`
- `degrade_action`
- referenced policy decision fields (`winner_stage`, `deny_source`) when present

#### Scenario: Deny driven by budget over-threshold
- **WHEN** budget decision is `deny` and policy winner is non-blocking
- **THEN** admission output includes canonical budget fields and preserves referenced policy fields unchanged

#### Scenario: Degrade decision under equivalent Run and Stream
- **WHEN** equivalent requests in Run and Stream enter budget degrade range
- **THEN** both outputs preserve semantically equivalent budget and policy explainability fields

### Requirement: Budget admission extension SHALL be absorbed within A60 contract scope
Budget-admission domain extensions (budget dimensions, threshold bands, degrade actions, replay classes) MUST be absorbed as additive updates within this capability contract and MUST NOT require parallel same-domain proposal semantics.

#### Scenario: New budget dimension is introduced
- **WHEN** maintainers add a new budget dimension under runtime admission
- **THEN** change is expressed as additive update in this capability contract with replay and gate updates, without creating parallel budget-admission semantics
