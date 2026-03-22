## ADDED Requirements

### Requirement: Shared multi-agent gate SHALL include collaboration retry contract suites
The shared multi-agent quality gate MUST execute collaboration retry contract suites as blocking checks in both shell and PowerShell shared-gate paths.

Required coverage MUST include:
- retry-disabled default behavior,
- bounded retry with exponential backoff+jitter under enabled policy,
- `retry_on=transport_only` classification behavior,
- scheduler-managed single-owner retry behavior (no compounded retries),
- Run/Stream semantic equivalence and replay-idempotent aggregate behavior.

#### Scenario: Shell shared gate executes collaboration retry suites
- **WHEN** contributor executes `scripts/check-multi-agent-shared-contract.sh`
- **THEN** collaboration retry suites are executed as required blocking checks

#### Scenario: PowerShell shared gate executes collaboration retry suites
- **WHEN** contributor executes `scripts/check-multi-agent-shared-contract.ps1`
- **THEN** equivalent collaboration retry suites are executed as required blocking checks

### Requirement: Collaboration retry gate SHALL fail fast on retry-policy semantic drift
If contract suites detect retry-boundary drift, retry-classification drift, or compounded retry behavior drift, shared gate MUST fail fast and return non-zero status.

#### Scenario: Regression introduces compounded primitive+scheduler retries
- **WHEN** contract suite observes one logical failure triggering both primitive retry and scheduler retry loops simultaneously
- **THEN** shared quality gate fails and blocks merge

