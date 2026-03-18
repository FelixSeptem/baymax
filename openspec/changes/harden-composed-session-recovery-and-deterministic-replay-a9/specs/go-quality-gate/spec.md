## ADDED Requirements

### Requirement: Shared multi-agent gate SHALL include session recovery contract suite
Quality gate contract MUST include session recovery and deterministic replay tests in the existing shared multi-agent gate scripts.

#### Scenario: CI runs shared multi-agent gate after recovery rollout
- **WHEN** CI executes `check-multi-agent-shared-contract.*`
- **THEN** recovery/replay contract suites run as blocking checks in the same gate path

### Requirement: Recovery gate SHALL block semantic divergence and conflict-policy regressions
Recovery contract suite MUST fail on Run/Stream semantic divergence, replay counter inflation, or non-fail-fast conflict handling.

#### Scenario: Regression changes conflict handling away from fail-fast
- **WHEN** recovery conflict handling regresses to non-fail-fast behavior
- **THEN** recovery contract tests fail and block merge
