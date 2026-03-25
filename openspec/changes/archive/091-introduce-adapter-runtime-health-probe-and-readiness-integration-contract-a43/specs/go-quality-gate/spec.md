## ADDED Requirements

### Requirement: Quality gate SHALL include adapter-health contract suites
The standard quality gate MUST execute adapter-health contract suites as blocking validation in both shell and PowerShell paths.

The suites MUST cover:
- adapter-health configuration validation,
- readiness mapping strict/non-strict behavior,
- diagnostics additive schema and replay idempotency,
- adapter conformance health matrix.

#### Scenario: Contributor runs shell quality gate
- **WHEN** contributor executes `bash scripts/check-quality-gate.sh`
- **THEN** adapter-health contract suites run as required blocking checks

#### Scenario: Contributor runs PowerShell quality gate
- **WHEN** contributor executes `pwsh -File scripts/check-quality-gate.ps1`
- **THEN** equivalent adapter-health contract suites run as required blocking checks

### Requirement: Quality gate SHALL block merge on adapter-health semantic drift
When adapter-health suites detect readiness mapping drift, non-canonical reason taxonomy, or replay-idempotency regression, quality gate MUST fail and block merge.

#### Scenario: Adapter-health mapping drifts from contract
- **WHEN** contract suites detect divergence in required/optional mapping or strict escalation behavior
- **THEN** quality gate exits non-zero and blocks validation

#### Scenario: Adapter-health semantics remain within contract
- **WHEN** adapter-health suites pass all canonical semantic assertions
- **THEN** quality gate proceeds without adapter-health failure
