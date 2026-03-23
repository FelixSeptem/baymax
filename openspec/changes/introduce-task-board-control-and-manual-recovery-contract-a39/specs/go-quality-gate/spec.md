## ADDED Requirements

### Requirement: Shared gate SHALL include task-board manual-control contract suites
The shared multi-agent contract gate MUST execute task-board manual-control suites as blocking checks.

The suites MUST cover at minimum:
- action validation and state-matrix fail-fast behavior,
- `operation_id` idempotent dedup and replay stability,
- manual retry budget enforcement (`max_manual_retry_per_task`),
- canonical reason taxonomy coverage (`scheduler.manual_cancel`, `scheduler.manual_retry`),
- memory/file backend parity and Run/Stream semantic equivalence.

#### Scenario: Contributor runs shared gate in shell
- **WHEN** contributor executes `scripts/check-multi-agent-shared-contract.sh`
- **THEN** task-board manual-control suites run as required blocking checks

#### Scenario: Contributor runs shared gate in PowerShell
- **WHEN** contributor executes `scripts/check-multi-agent-shared-contract.ps1`
- **THEN** equivalent task-board manual-control suites run as required blocking checks

#### Scenario: Regression introduces non-canonical manual-control reason
- **WHEN** contract suites detect manual-control reason drift outside canonical scheduler namespace without synchronized contract update
- **THEN** shared quality gate fails and blocks merge
