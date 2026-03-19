## ADDED Requirements

### Requirement: Quality gate SHALL include delayed-dispatch contract suites
Shared multi-agent quality gate MUST include delayed-dispatch contract suites as blocking checks.

#### Scenario: CI executes shared gate after delayed-dispatch rollout
- **WHEN** CI runs `check-multi-agent-shared-contract.*`
- **THEN** delayed-dispatch suites execute as blocking checks in the same gate path

### Requirement: Delayed-dispatch gate SHALL block early-claim and recovery-drift regressions
Delayed-dispatch contract suites MUST fail on early-claim regressions, delayed-ready ordering drift, or restore-time semantic drift.

#### Scenario: Regression claims task before not_before
- **WHEN** scheduler claims a delayed task before `not_before`
- **THEN** delayed-dispatch contract suite fails and blocks merge
