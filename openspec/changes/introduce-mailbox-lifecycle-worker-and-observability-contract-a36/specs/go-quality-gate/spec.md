## MODIFIED Requirements

### Requirement: Shared multi-agent gate SHALL include mailbox contract suites
The shared multi-agent contract gate MUST execute mailbox contract suites as blocking checks.

The mailbox suites MUST cover:
- envelope validation and idempotency,
- ack/nack/retry/ttl/dlq lifecycle semantics,
- sync/async/delayed convergence through mailbox,
- mailbox query pagination/cursor deterministic behavior,
- memory/file backend parity,
- mailbox worker lifecycle execution semantics,
- mailbox worker default policy semantics (`enabled=false`, `poll_interval=100ms`, `handler_error_policy=requeue`),
- mailbox lifecycle canonical reason taxonomy drift detection.

#### Scenario: Contributor runs shared gate in shell
- **WHEN** contributor executes `scripts/check-multi-agent-shared-contract.sh`
- **THEN** mailbox contract suites are executed as required blocking checks

#### Scenario: Contributor runs shared gate in PowerShell
- **WHEN** contributor executes `scripts/check-multi-agent-shared-contract.ps1`
- **THEN** equivalent mailbox contract suites are executed as required blocking checks

#### Scenario: Regression introduces non-canonical mailbox lifecycle reason
- **WHEN** contract suites detect mailbox lifecycle reason code outside canonical taxonomy without synchronized contract update
- **THEN** shared quality gate fails and blocks merge
