## ADDED Requirements

### Requirement: Shared quality gate SHALL include long-running recovery-boundary contract suites
Shared multi-agent quality gate MUST include long-running recovery-boundary suites as blocking checks.

#### Scenario: CI executes shared gate after A17 rollout
- **WHEN** CI runs `check-multi-agent-shared-contract.*`
- **THEN** recovery-boundary suites execute as blocking checks in the shared gate path

### Requirement: Recovery-boundary gate SHALL block rewind and unbounded-reentry regressions
Recovery-boundary contract suites MUST fail on rewind of terminal tasks, unbounded timeout reentry, or boundary policy drift.

#### Scenario: Regression re-executes terminal task after restore
- **WHEN** contract suite detects restored terminal task being dispatched again
- **THEN** quality gate fails and blocks merge

### Requirement: Mainline contract index SHALL map recovery-boundary matrix coverage
Mainline contract index MUST include traceable coverage rows for crash/restart/replay/timeout boundary scenarios.

#### Scenario: Contributor audits A17 coverage mappings
- **WHEN** contributor inspects contract index for recovery-boundary entries
- **THEN** each required boundary scenario maps to concrete test cases
