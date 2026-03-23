## MODIFIED Requirements

### Requirement: Shared multi-agent gate SHALL include mailbox contract suites
The shared multi-agent contract gate MUST execute mailbox contract suites as blocking checks.

The mailbox suites MUST cover:
- envelope validation and idempotency,
- ack/nack/retry/ttl/dlq lifecycle semantics,
- sync/async/delayed convergence through mailbox,
- mailbox query pagination/cursor deterministic behavior,
- memory/file backend parity,
- runtime mailbox wiring activation from effective config,
- file-backend init failure fallback-to-memory semantics with deterministic reason traceability,
- mailbox diagnostics query/aggregate traceability for managed orchestration traffic.

#### Scenario: Contributor runs shared gate in shell
- **WHEN** contributor executes `scripts/check-multi-agent-shared-contract.sh`
- **THEN** mailbox contract suites are executed as required blocking checks

#### Scenario: Contributor runs shared gate in PowerShell
- **WHEN** contributor executes `scripts/check-multi-agent-shared-contract.ps1`
- **THEN** equivalent mailbox contract suites are executed as required blocking checks

#### Scenario: Regression bypasses shared runtime mailbox wiring
- **WHEN** change reintroduces per-call ephemeral mailbox bridge behavior on managed orchestration path
- **THEN** shared mailbox contract suites fail and block merge
