## ADDED Requirements

### Requirement: Quality gate SHALL include cross-domain primary-reason arbitration contract suites
Quality gate MUST execute cross-domain primary-reason arbitration suites as blocking checks in both shell and PowerShell workflows.

The suites MUST cover:
- precedence-order assertions,
- tie-break determinism assertions,
- Run/Stream parity assertions,
- replay idempotency assertions,
- taxonomy drift assertions.

#### Scenario: Contributor runs shell quality gate
- **WHEN** contributor executes `bash scripts/check-quality-gate.sh`
- **THEN** arbitration contract suites run as required blocking checks

#### Scenario: Contributor runs PowerShell quality gate
- **WHEN** contributor executes `pwsh -File scripts/check-quality-gate.ps1`
- **THEN** equivalent arbitration contract suites run as required blocking checks

### Requirement: Quality gate SHALL fail fast on primary-reason arbitration semantic drift
When arbitration suites detect precedence drift, tie-break drift, or canonical taxonomy drift, quality gate MUST fail fast and block merge.

#### Scenario: Drift changes top-level timeout precedence
- **WHEN** arbitration suite detects timeout reject no longer outranks blocked readiness
- **THEN** quality gate exits non-zero and blocks validation

#### Scenario: Arbitration semantics remain aligned
- **WHEN** arbitration suites pass canonical semantic assertions
- **THEN** quality gate proceeds without arbitration-related failure
