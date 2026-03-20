## ADDED Requirements

### Requirement: Shared multi-agent gate SHALL include mailbox contract suites
The shared multi-agent contract gate MUST execute mailbox contract suites as blocking checks.

The mailbox suites MUST cover:
- envelope validation and idempotency,
- ack/nack/retry/ttl/dlq lifecycle semantics,
- sync/async/delayed convergence through mailbox,
- mailbox query pagination/cursor deterministic behavior,
- memory/file backend parity.

#### Scenario: Contributor runs shared gate in shell
- **WHEN** contributor executes `scripts/check-multi-agent-shared-contract.sh`
- **THEN** mailbox contract suites are executed as required blocking checks

#### Scenario: Contributor runs shared gate in PowerShell
- **WHEN** contributor executes `scripts/check-multi-agent-shared-contract.ps1`
- **THEN** equivalent mailbox contract suites are executed as required blocking checks

### Requirement: Quality gate SHALL track mailbox migration as canonical multi-agent path
Quality gate and contract index mapping MUST treat mailbox path as canonical for sync/async/delayed coordination flows after migration.

#### Scenario: Maintainer audits shared contract index after mailbox rollout
- **WHEN** maintainer reviews gate scripts and mainline contract index
- **THEN** mailbox-based rows are canonical and legacy path mapping is marked deprecated
