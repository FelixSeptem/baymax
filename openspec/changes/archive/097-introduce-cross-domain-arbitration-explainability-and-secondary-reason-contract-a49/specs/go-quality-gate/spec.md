## ADDED Requirements

### Requirement: Quality gate SHALL include arbitration explainability contract suites
Quality gate MUST execute arbitration explainability contract suites as blocking checks in both shell and PowerShell workflows.

The suites MUST cover:
- secondary reason boundedness and deterministic ordering,
- remediation hint taxonomy stability,
- rule-version stability,
- Run/Stream explainability parity,
- replay idempotency for explainability aggregates.

#### Scenario: Contributor runs shell quality gate
- **WHEN** contributor executes `bash scripts/check-quality-gate.sh`
- **THEN** arbitration explainability suites run as required blocking checks

#### Scenario: Contributor runs PowerShell quality gate
- **WHEN** contributor executes `pwsh -File scripts/check-quality-gate.ps1`
- **THEN** equivalent arbitration explainability suites run as required blocking checks

### Requirement: Quality gate SHALL fail fast on explainability semantic drift
When explainability suites detect secondary ordering drift, hint taxonomy drift, or rule-version drift, quality gate MUST fail fast and block merge.

#### Scenario: Secondary ordering drifts from canonical rule
- **WHEN** explainability suite detects non-deterministic secondary ordering for equivalent input
- **THEN** quality gate exits non-zero and blocks validation

#### Scenario: Explainability semantics remain aligned
- **WHEN** explainability suites pass canonical assertions
- **THEN** quality gate proceeds without explainability-related failure
