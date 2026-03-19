## ADDED Requirements

### Requirement: Quality gate SHALL include adapter conformance validation as blocking step
The standard quality gate MUST execute adapter conformance validation and treat failures as blocking.

This validation MUST be integrated into both shell and PowerShell quality-gate paths.

#### Scenario: Contributor runs shell quality gate
- **WHEN** contributor executes `bash scripts/check-quality-gate.sh`
- **THEN** adapter conformance validation is executed as required blocking step

#### Scenario: Contributor runs PowerShell quality gate
- **WHEN** contributor executes `pwsh -File scripts/check-quality-gate.ps1`
- **THEN** equivalent adapter conformance validation is executed as required blocking step

### Requirement: Adapter conformance gate SHALL fail fast and return deterministic non-zero status
If any conformance case fails, quality gate MUST fail fast and return deterministic non-zero status without continuing as success.

#### Scenario: Conformance case fails during validation
- **WHEN** one adapter conformance scenario reports semantic mismatch
- **THEN** quality gate exits non-zero and blocks merge

#### Scenario: All conformance cases pass
- **WHEN** all required adapter conformance scenarios pass
- **THEN** quality gate proceeds without adapter conformance failure

### Requirement: Mainline contract index SHALL map adapter conformance coverage and gate paths
Repository documentation MUST map adapter conformance scenarios to concrete test entries and gate scripts for traceability.

#### Scenario: Maintainer audits adapter contract coverage
- **WHEN** maintainer inspects mainline contract index after A22
- **THEN** adapter conformance rows map to concrete harness test entries and quality-gate script paths

