## ADDED Requirements

### Requirement: Quality gate SHALL include policy precedence contract suites
Repository quality gate MUST include policy precedence contract checks as blocking suites in both shell and PowerShell paths.

Minimum required scripts:
- `scripts/check-policy-precedence-contract.sh`
- `scripts/check-policy-precedence-contract.ps1`

#### Scenario: Shell quality gate executes policy precedence contract suite
- **WHEN** `scripts/check-quality-gate.sh` runs in CI or local pre-merge flow
- **THEN** policy precedence contract checks execute and fail-fast on non-zero exit

#### Scenario: PowerShell quality gate executes policy precedence contract suite
- **WHEN** `scripts/check-quality-gate.ps1` runs in CI or local pre-merge flow
- **THEN** policy precedence contract checks execute with equivalent blocking semantics

### Requirement: Policy precedence gate SHALL provide deterministic required-check candidate
CI SHOULD expose independent required-check candidate `policy-precedence-gate` for policy-stack contract regressions.

#### Scenario: Policy precedence gate detects replay drift
- **WHEN** `policy_stack.v1` replay validation fails
- **THEN** `policy-precedence-gate` fails deterministically and blocks merge

#### Scenario: Policy precedence gate passes all suites
- **WHEN** config, integration, replay, and docs parity checks pass
- **THEN** `policy-precedence-gate` reports deterministic success
