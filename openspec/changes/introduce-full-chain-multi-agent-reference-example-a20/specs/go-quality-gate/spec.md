## ADDED Requirements

### Requirement: Quality gate SHALL include full-chain example smoke validation
The standard quality gate MUST execute smoke validation for the full-chain multi-agent reference example as a blocking step.

This validation MUST be included in both shell and PowerShell quality-gate scripts.

#### Scenario: Shell quality gate runs
- **WHEN** contributor executes `bash scripts/check-quality-gate.sh`
- **THEN** full-chain example smoke validation runs as a required blocking step

#### Scenario: PowerShell quality gate runs
- **WHEN** contributor executes `pwsh -File scripts/check-quality-gate.ps1`
- **THEN** equivalent full-chain example smoke validation runs as a required blocking step

### Requirement: Example smoke gate SHALL fail fast on execution drift
If full-chain example smoke command fails, times out, or misses required success markers, quality gate MUST fail and return non-zero status.

#### Scenario: Example execution fails
- **WHEN** full-chain example smoke command exits with non-zero status
- **THEN** quality gate fails and blocks merge

#### Scenario: Example output misses required convergence markers
- **WHEN** smoke validation cannot find required success/checkpoint markers
- **THEN** quality gate fails with explicit example-smoke classification

### Requirement: Mainline index SHALL trace full-chain example smoke coverage
The repository MUST update mainline contract/index documentation to include traceability between full-chain example smoke checks and corresponding gate paths.

#### Scenario: Contributor audits full-chain example validation mapping
- **WHEN** contributor inspects mainline index after A20
- **THEN** full-chain example smoke check has explicit mapping to quality-gate execution path

