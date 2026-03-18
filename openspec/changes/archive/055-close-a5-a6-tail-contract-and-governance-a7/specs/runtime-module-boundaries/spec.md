## ADDED Requirements

### Requirement: Shared contract gate SHALL include scheduler/subagent closure checks
Runtime boundary governance MUST extend shared-contract gate checks to include scheduler/subagent namespace and correlation requirements.

Minimum additional checks:
- scheduler/subagent reason namespace compliance,
- attempt-level correlation field presence,
- single-writer diagnostics path compliance.

#### Scenario: Non-canonical scheduler reason is introduced
- **WHEN** a change emits scheduler/subagent reason outside canonical taxonomy
- **THEN** shared-contract gate fails and blocks merge
