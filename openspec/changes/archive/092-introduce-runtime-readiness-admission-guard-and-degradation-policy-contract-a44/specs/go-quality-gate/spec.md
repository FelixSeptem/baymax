## ADDED Requirements

### Requirement: Quality gate SHALL include readiness-admission contract suites
Quality gate MUST execute readiness-admission contract suites as blocking checks in both shell and PowerShell workflows.

The suites MUST cover:
- admission config validation fail-fast and rollback behavior,
- blocked/degraded policy mapping semantics,
- deny-path side-effect-free assertions,
- Run/Stream admission equivalence,
- diagnostics additive schema and replay idempotency.

#### Scenario: Contributor runs shell quality gate
- **WHEN** contributor executes `bash scripts/check-quality-gate.sh`
- **THEN** readiness-admission contract suites run as required blocking checks

#### Scenario: Contributor runs PowerShell quality gate
- **WHEN** contributor executes `pwsh -File scripts/check-quality-gate.ps1`
- **THEN** equivalent readiness-admission contract suites run as required blocking checks

### Requirement: Quality gate SHALL block merge on readiness-admission semantic drift
When readiness-admission suites detect mapping drift, non-canonical admission reason taxonomy, or deny-path side-effect regressions, quality gate MUST fail and block merge.

#### Scenario: Admission deny path mutates scheduler state
- **WHEN** contract suites detect task lifecycle mutation after admission deny
- **THEN** quality gate exits non-zero and blocks validation

#### Scenario: Readiness-admission semantics remain aligned
- **WHEN** readiness-admission suites pass canonical semantic assertions
- **THEN** quality gate proceeds without readiness-admission failure
