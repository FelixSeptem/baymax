## ADDED Requirements

### Requirement: Quality gate SHALL include composer contract suite in shared multi-agent gate
The quality gate MUST include composer integration contract tests in the existing shared multi-agent gate pipeline, rather than introducing a disconnected parallel gate.

#### Scenario: CI executes multi-agent shared-contract gate after A8
- **WHEN** CI runs shared multi-agent contract gate scripts
- **THEN** composer contract suites run as blocking checks within the same gate path

### Requirement: Composer contract suite SHALL cover fallback and semantic equivalence
Composer contract suites MUST cover scheduler fallback-to-memory behavior, Run/Stream semantic equivalence, and replay/idempotency behavior for scheduler-managed child execution.

#### Scenario: Regression introduces Run/Stream summary divergence
- **WHEN** equivalent composer-managed Run and Stream requests produce non-equivalent aggregate summaries
- **THEN** composer contract suite fails and blocks merge
