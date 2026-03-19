## ADDED Requirements

### Requirement: Quality gate SHALL include adapter scaffold drift validation as blocking step
The repository quality gate MUST execute adapter scaffold drift validation and MUST treat failures as blocking.

This validation MUST be integrated into both `scripts/check-quality-gate.sh` and `scripts/check-quality-gate.ps1` with equivalent semantics.

#### Scenario: Contributor runs shell quality gate
- **WHEN** contributor executes `bash scripts/check-quality-gate.sh`
- **THEN** adapter scaffold drift validation is executed as a required blocking step

#### Scenario: Contributor runs PowerShell quality gate
- **WHEN** contributor executes `pwsh -File scripts/check-quality-gate.ps1`
- **THEN** adapter scaffold drift validation is executed with equivalent blocking behavior

### Requirement: Scaffold drift validation SHALL fail fast with deterministic non-zero status
If generated scaffold output diverges from repository source-of-truth templates or expected fixture mapping, drift validation MUST fail fast and return non-zero status.

#### Scenario: Template drift is detected
- **WHEN** drift validation detects mismatch between generated scaffold and committed expectation
- **THEN** quality gate exits non-zero and blocks merge

#### Scenario: Scaffold output matches source-of-truth
- **WHEN** drift validation confirms all required scaffold outputs are aligned
- **THEN** quality gate continues without scaffold drift failure

### Requirement: Quality gate SHALL preserve traceability between scaffold generation and conformance bootstrap checks
Repository validation flow MUST keep traceable linkage between scaffold generation outputs and adapter conformance bootstrap coverage.

#### Scenario: Maintainer audits scaffold-conformance traceability
- **WHEN** maintainer reviews quality-gate scripts and contract index
- **THEN** maintainer can identify how scaffold drift checks and conformance bootstrap checks map to concrete validation entries
