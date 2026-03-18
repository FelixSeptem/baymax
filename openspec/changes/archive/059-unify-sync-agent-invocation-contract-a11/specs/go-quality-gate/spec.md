## ADDED Requirements

### Requirement: Quality gate SHALL include shared synchronous invocation contract tests
The shared multi-agent quality gate MUST include contract suites validating shared synchronous invocation behavior across orchestration integration paths.

#### Scenario: CI executes shared multi-agent contract gate for A11
- **WHEN** CI runs shared multi-agent contract scripts after A11 rollout
- **THEN** shared synchronous invocation contract tests are executed as blocking checks

### Requirement: Synchronous invocation gate SHALL block semantic divergence
Shared synchronous invocation contract suite MUST fail on timeout/cancellation precedence regressions, error-layer normalization drift, or Run/Stream semantic divergence.

#### Scenario: Regression changes cancellation precedence in one module path
- **WHEN** one orchestration path diverges from shared synchronous invocation cancellation semantics
- **THEN** contract suite fails and blocks merge
