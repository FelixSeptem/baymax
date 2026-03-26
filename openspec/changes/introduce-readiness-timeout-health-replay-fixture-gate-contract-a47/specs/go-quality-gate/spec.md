## ADDED Requirements

### Requirement: Quality gate SHALL include readiness-timeout-health replay fixture suites
Quality gate MUST execute readiness-timeout-health composite replay fixture suites as blocking checks in both shell and PowerShell workflows.

The suites MUST cover:
- composite fixture matrix coverage,
- canonical taxonomy drift detection,
- Run/Stream parity assertions,
- replay idempotency assertions.

#### Scenario: Contributor runs shell quality gate
- **WHEN** contributor executes `bash scripts/check-quality-gate.sh`
- **THEN** A47 composite replay fixture suites run as required blocking checks

#### Scenario: Contributor runs PowerShell quality gate
- **WHEN** contributor executes `pwsh -File scripts/check-quality-gate.ps1`
- **THEN** equivalent A47 composite replay fixture suites run as required blocking checks

### Requirement: Quality gate SHALL fail fast on composite semantic drift
When composite replay fixture suites detect canonical semantic drift across readiness, timeout-resolution, or adapter-health domains, quality gate MUST fail fast and block merge.

#### Scenario: Composite fixture detects timeout-source drift
- **WHEN** fixture assertion detects non-canonical timeout-resolution source mapping
- **THEN** quality gate exits non-zero and blocks validation

#### Scenario: Composite fixture semantics remain aligned
- **WHEN** composite replay fixture assertions pass canonical semantic checks
- **THEN** quality gate proceeds without A47 replay fixture failure
