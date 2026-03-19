## ADDED Requirements

### Requirement: Quality gate SHALL include multi-agent performance regression checks
The standard repository quality gate MUST execute multi-agent mainline benchmark regression checks as blocking validation.

This check MUST run in both shell and PowerShell quality-gate scripts to preserve cross-platform parity.

#### Scenario: Contributor runs shell quality gate
- **WHEN** contributor executes `bash scripts/check-quality-gate.sh`
- **THEN** multi-agent performance regression check is executed as a required step

#### Scenario: Contributor runs PowerShell quality gate
- **WHEN** contributor executes `pwsh -File scripts/check-quality-gate.ps1`
- **THEN** equivalent multi-agent performance regression check is executed as a required step

### Requirement: Quality gate SHALL block merge on multi-agent performance threshold regression
When multi-agent benchmark regression check reports degradation beyond configured thresholds, quality gate MUST fail and block merge.

#### Scenario: Candidate regression exceeds configured threshold
- **WHEN** one or more multi-agent benchmark metrics exceed configured degradation limits
- **THEN** quality gate exits non-zero and validation is blocked

#### Scenario: Candidate regression stays within configured thresholds
- **WHEN** all required multi-agent benchmark metrics remain within configured limits
- **THEN** quality gate proceeds without performance-regression failure

### Requirement: CI quality workflow SHALL preserve local parity for multi-agent performance gate
Default CI workflow MUST invoke quality gate steps that include the same multi-agent performance regression semantics used in local scripts.

#### Scenario: CI executes test-and-lint quality path
- **WHEN** CI runs the default quality-gate job
- **THEN** multi-agent performance regression check behavior matches local quality-gate scripts

