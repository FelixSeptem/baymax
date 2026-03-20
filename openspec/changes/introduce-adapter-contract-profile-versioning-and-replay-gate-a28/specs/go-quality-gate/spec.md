## ADDED Requirements

### Requirement: Quality gate SHALL include adapter contract replay validation
The standard quality gate MUST execute adapter contract replay validation and treat failures as blocking.

This validation MUST run in both shell and PowerShell gate paths.

#### Scenario: Contributor runs shell quality gate
- **WHEN** contributor executes `bash scripts/check-quality-gate.sh`
- **THEN** adapter contract replay validation executes as required blocking step

#### Scenario: Contributor runs PowerShell quality gate
- **WHEN** contributor executes `pwsh -File scripts/check-quality-gate.ps1`
- **THEN** equivalent adapter contract replay validation executes as required blocking step

### Requirement: Contract replay gate SHALL fail fast on profile drift
If replay fixtures diverge from runtime outputs for supported profile versions, validation MUST fail fast and return deterministic non-zero status.

#### Scenario: Replay detects taxonomy drift
- **WHEN** replay validation detects reason taxonomy output differs from fixture baseline
- **THEN** quality gate exits non-zero and blocks merge

#### Scenario: Replay detects profile compatibility window mismatch
- **WHEN** replay validation detects unsupported profile handling diverges from contract expectations
- **THEN** quality gate exits non-zero and blocks merge
