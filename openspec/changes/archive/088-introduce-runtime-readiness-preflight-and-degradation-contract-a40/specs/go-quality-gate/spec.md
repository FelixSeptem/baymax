## ADDED Requirements

### Requirement: Quality gate SHALL include runtime readiness contract suites
Quality gate MUST execute runtime readiness contract suites as blocking checks in both shell and PowerShell workflows.

The suites MUST cover at minimum:
- readiness status classification (`ready|degraded|blocked`),
- strict policy escalation (`degraded -> blocked` when strict enabled),
- canonical finding schema and code stability,
- diagnostics additive readiness fields and replay idempotency,
- composer readiness passthrough parity with runtime readiness.

#### Scenario: Contributor runs quality gate in shell
- **WHEN** contributor executes `scripts/check-quality-gate.sh`
- **THEN** readiness contract suites run as required blocking checks

#### Scenario: Contributor runs quality gate in PowerShell
- **WHEN** contributor executes `scripts/check-quality-gate.ps1`
- **THEN** equivalent readiness contract suites run as required blocking checks

#### Scenario: Regression breaks readiness code taxonomy or strict escalation
- **WHEN** readiness contract suite detects non-canonical finding code or strict escalation mismatch
- **THEN** quality gate fails and blocks merge
