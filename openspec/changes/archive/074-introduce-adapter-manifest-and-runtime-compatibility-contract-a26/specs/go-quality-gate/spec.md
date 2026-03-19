## ADDED Requirements

### Requirement: Quality gate SHALL include adapter manifest contract validation as blocking step
The standard quality gate MUST execute adapter manifest contract validation and MUST treat failures as blocking.

This validation MUST be integrated into both shell and PowerShell quality-gate paths.

#### Scenario: Contributor runs shell quality gate
- **WHEN** contributor executes `bash scripts/check-quality-gate.sh`
- **THEN** adapter manifest contract validation runs as required blocking step

#### Scenario: Contributor runs PowerShell quality gate
- **WHEN** contributor executes `pwsh -File scripts/check-quality-gate.ps1`
- **THEN** equivalent adapter manifest contract validation runs as required blocking step

### Requirement: Manifest contract gate SHALL fail fast with deterministic non-zero status
If manifest schema, compatibility range, or required capability checks fail, validation MUST fail fast and return deterministic non-zero status.

#### Scenario: Manifest compatibility check fails
- **WHEN** manifest contract validation detects incompatible `baymax_compat` or invalid semver expression
- **THEN** quality gate exits non-zero and blocks merge

#### Scenario: Manifest contract checks pass
- **WHEN** all required manifest schema and compatibility checks pass
- **THEN** quality gate proceeds without manifest-gate failure
