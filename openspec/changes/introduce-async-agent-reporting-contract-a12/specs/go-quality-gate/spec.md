## ADDED Requirements

### Requirement: Quality gate SHALL include async reporting contract suites
Shared multi-agent quality gate MUST include async reporting contract suites as blocking checks.

#### Scenario: CI executes shared gate after A12 rollout
- **WHEN** CI runs `check-multi-agent-shared-contract.*`
- **THEN** async reporting contract tests are executed as blocking checks in the same gate path

### Requirement: Async reporting gate SHALL block retry-idempotency regressions
Async reporting contract suites MUST fail on delivery retry drift, dedup regression, or replay-idempotency violations.

#### Scenario: Regression causes duplicate async reports to inflate aggregates
- **WHEN** duplicate async reports increase logical counters
- **THEN** contract suite fails and blocks merge
