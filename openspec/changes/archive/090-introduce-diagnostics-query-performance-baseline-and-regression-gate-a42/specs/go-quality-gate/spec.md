## ADDED Requirements

### Requirement: Quality gate SHALL include diagnostics-query performance regression checks
The standard repository quality gate MUST execute diagnostics-query benchmark regression checks as blocking validation.

This check MUST run in both shell and PowerShell quality-gate scripts to preserve cross-platform parity.

#### Scenario: Contributor runs shell quality gate
- **WHEN** contributor executes `bash scripts/check-quality-gate.sh`
- **THEN** diagnostics-query performance regression check is executed as a required blocking step

#### Scenario: Contributor runs PowerShell quality gate
- **WHEN** contributor executes `pwsh -File scripts/check-quality-gate.ps1`
- **THEN** equivalent diagnostics-query performance regression check is executed as a required blocking step

### Requirement: Quality gate SHALL block merge on diagnostics-query threshold regression
When diagnostics-query benchmark regression check reports degradation beyond configured thresholds, quality gate MUST fail and block merge.

#### Scenario: Diagnostics-query regression exceeds configured threshold
- **WHEN** one or more diagnostics-query benchmark metrics exceed configured degradation limits
- **THEN** quality gate exits non-zero and validation is blocked

#### Scenario: Diagnostics-query regression remains within configured threshold
- **WHEN** all diagnostics-query benchmark metrics remain within configured limits
- **THEN** quality gate proceeds without diagnostics-query performance failure
