## ADDED Requirements

### Requirement: Quality gate SHALL include scheduler QoS and dead-letter contract suites
The shared multi-agent quality gate MUST include scheduler qos/fairness/dead-letter contract tests as blocking checks.

#### Scenario: CI executes shared multi-agent gate after A10 rollout
- **WHEN** CI runs `check-multi-agent-shared-contract.*`
- **THEN** scheduler QoS and dead-letter suites execute as blocking checks in that gate path

### Requirement: QoS gate SHALL block fairness and dead-letter regressions
QoS contract suites MUST fail on fairness-window violations, dead-letter transfer regressions, or retry-backoff policy drift.

#### Scenario: Regression bypasses fairness threshold
- **WHEN** high-priority claims exceed configured fairness window without yielding
- **THEN** QoS contract tests fail and block merge
