## ADDED Requirements

### Requirement: Shared multi-agent gate SHALL include async-await reconcile contract suites
The shared multi-agent quality gate MUST execute async-await reconcile contract suites as blocking checks in both shell and PowerShell shared-gate paths.

Required coverage MUST include:
- callback-loss reconcile fallback convergence,
- first-terminal-wins arbitration and conflict recording,
- `not_found -> keep_until_timeout` behavior,
- Run/Stream semantic equivalence,
- memory/file backend parity,
- replay idempotency for callback/poll mixed events.

#### Scenario: Shell shared gate executes reconcile suites
- **WHEN** contributor executes `scripts/check-multi-agent-shared-contract.sh`
- **THEN** async-await reconcile suites run as required blocking checks

#### Scenario: PowerShell shared gate executes reconcile suites
- **WHEN** contributor executes `scripts/check-multi-agent-shared-contract.ps1`
- **THEN** equivalent async-await reconcile suites run as required blocking checks

### Requirement: Reconcile gate SHALL fail fast on terminal-arbitration or fallback semantic drift
If contract suites detect regression in first-terminal-wins arbitration, conflict recording, not-found timeout behavior, or replay-idempotency semantics, shared gate MUST fail fast and return non-zero status.

#### Scenario: Regression allows second terminal to overwrite first terminal
- **WHEN** contract suite observes later callback/poll terminal overwriting first committed terminal state
- **THEN** shared quality gate fails and blocks merge

