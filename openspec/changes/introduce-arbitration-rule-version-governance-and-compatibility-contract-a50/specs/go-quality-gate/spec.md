## ADDED Requirements

### Requirement: Quality gate SHALL include arbitration-version governance contract suites
Quality gate MUST execute arbitration-version governance suites as blocking checks in both shell and PowerShell workflows.

The suites MUST cover:
- version resolution assertions,
- compatibility-window assertions,
- Run/Stream parity assertions,
- replay idempotency assertions,
- drift classification assertions.

#### Scenario: Contributor runs shell quality gate
- **WHEN** contributor executes `bash scripts/check-quality-gate.sh`
- **THEN** arbitration-version governance suites run as required blocking checks

#### Scenario: Contributor runs PowerShell quality gate
- **WHEN** contributor executes `pwsh -File scripts/check-quality-gate.ps1`
- **THEN** equivalent arbitration-version governance suites run as required blocking checks

### Requirement: Quality gate SHALL fail fast on arbitration-version semantic drift
When arbitration-version suites detect unsupported-version handling drift, compatibility-mismatch drift, or cross-version semantic drift, quality gate MUST fail fast and block merge.

#### Scenario: Drift changes unsupported-version fail-fast behavior
- **WHEN** arbitration-version suite detects unsupported request no longer triggers fail-fast policy
- **THEN** quality gate exits non-zero and blocks validation

#### Scenario: Arbitration-version semantics remain aligned
- **WHEN** arbitration-version suites pass canonical semantic assertions
- **THEN** quality gate proceeds without arbitration-version-related failure
