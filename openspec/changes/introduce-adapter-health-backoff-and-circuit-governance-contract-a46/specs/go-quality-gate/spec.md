## ADDED Requirements

### Requirement: Quality gate SHALL include adapter-health governance contract suites
Quality gate MUST execute adapter-health governance contract suites as blocking checks in both shell and PowerShell workflows.

The suites MUST cover:
- adapter-health backoff/circuit config validation fail-fast and rollback behavior,
- circuit transition determinism and half-open budget semantics,
- readiness strict/non-strict mapping stability under governance paths,
- diagnostics additive schema stability and replay idempotency,
- adapter conformance governance matrix parity.

#### Scenario: Contributor runs shell quality gate
- **WHEN** contributor executes `bash scripts/check-quality-gate.sh`
- **THEN** adapter-health governance suites run as required blocking checks

#### Scenario: Contributor runs PowerShell quality gate
- **WHEN** contributor executes `pwsh -File scripts/check-quality-gate.ps1`
- **THEN** equivalent adapter-health governance suites run as required blocking checks

### Requirement: Quality gate SHALL block merge on adapter-health governance semantic drift
When governance suites detect transition drift, canonical reason-code drift, or replay-idempotency regressions, quality gate MUST fail and block merge.

#### Scenario: Regression alters half-open transition semantics
- **WHEN** governance suites detect `half_open` no longer reopens on failed probe
- **THEN** quality gate exits non-zero and blocks validation

#### Scenario: Governance semantics remain aligned
- **WHEN** governance suites pass canonical assertions
- **THEN** quality gate proceeds without adapter-health-governance failures
