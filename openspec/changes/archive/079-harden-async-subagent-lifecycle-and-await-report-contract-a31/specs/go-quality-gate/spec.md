## ADDED Requirements

### Requirement: Shared multi-agent gate SHALL include async-await lifecycle contract suites
The shared multi-agent quality gate MUST execute async-await lifecycle contract suites as blocking checks in both shell and PowerShell gate paths.

Required coverage MUST include:
- accepted-to-awaiting-report lifecycle transition,
- timeout terminalization behavior,
- late-report drop-and-record behavior,
- duplicate/replay idempotency behavior,
- Run/Stream semantic equivalence,
- memory/file backend parity.

#### Scenario: Shell shared gate executes async-await suites
- **WHEN** contributor executes `scripts/check-multi-agent-shared-contract.sh`
- **THEN** async-await lifecycle suites are executed as blocking checks

#### Scenario: PowerShell shared gate executes async-await suites
- **WHEN** contributor executes `scripts/check-multi-agent-shared-contract.ps1`
- **THEN** equivalent async-await lifecycle suites are executed as blocking checks

### Requirement: Async-await gate SHALL block lifecycle semantic regressions
If lifecycle state transition, timeout convergence, late-report policy, or replay-idempotency semantics drift from contract, shared gate MUST fail fast and return non-zero status.

#### Scenario: Regression changes late-report policy behavior
- **WHEN** contract suite detects late report mutates an already terminal business outcome
- **THEN** shared quality gate fails and blocks merge

