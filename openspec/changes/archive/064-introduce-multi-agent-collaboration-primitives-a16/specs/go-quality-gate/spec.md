## ADDED Requirements

### Requirement: Shared multi-agent quality gate SHALL include collaboration primitive contract suites
Shared multi-agent quality gate MUST include collaboration primitive contract suites as blocking checks.

#### Scenario: CI executes shared gate after collaboration rollout
- **WHEN** CI runs `check-multi-agent-shared-contract.*`
- **THEN** collaboration primitive suites run as blocking checks in the same shared gate path

### Requirement: Collaboration primitive gate SHALL block semantic drift across modes
Collaboration primitive contract suites MUST fail on semantic drift for sync/async/delayed mode composition, Run/Stream equivalence, or replay-idempotency behavior.

#### Scenario: Regression causes mode-dependent terminal divergence
- **WHEN** same collaboration primitive request produces divergent terminal semantics across modes
- **THEN** contract gate fails and blocks merge

### Requirement: Mainline contract index SHALL map collaboration primitive coverage
Mainline contract index MUST provide traceable mapping for collaboration primitive coverage including handoff, delegation, aggregation strategy semantics, and failure-policy behavior.

#### Scenario: Contributor audits collaboration primitive coverage
- **WHEN** contributor checks contract index and integration suites
- **THEN** each required collaboration primitive scenario has a concrete test mapping
