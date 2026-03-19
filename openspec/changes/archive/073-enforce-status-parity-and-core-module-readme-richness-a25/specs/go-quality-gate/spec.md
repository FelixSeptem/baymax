## ADDED Requirements

### Requirement: Quality gate SHALL include release status parity validation for progress docs
Repository docs consistency checks MUST validate status parity between OpenSpec authority sources and contributor-facing progress docs.

This validation MUST be executed in both shell and PowerShell documentation consistency paths and treated as blocking in quality gate.

#### Scenario: Shell docs consistency path executes
- **WHEN** contributor runs `bash scripts/check-docs-consistency.sh`
- **THEN** release status parity validation runs and failures return non-zero

#### Scenario: PowerShell docs consistency path executes
- **WHEN** contributor runs `pwsh -File scripts/check-docs-consistency.ps1`
- **THEN** equivalent release status parity validation runs with same blocking semantics

### Requirement: Quality gate SHALL include core module README richness validation
Repository docs consistency checks MUST validate required section baseline for covered core module README files.

Failures in module README richness validation MUST fail quality gate.

#### Scenario: Covered module README misses required section
- **WHEN** docs consistency checks detect missing required section marker in covered module README
- **THEN** quality gate exits non-zero and blocks merge

#### Scenario: Covered module READMEs satisfy richness baseline
- **WHEN** all covered module READMEs include required sections or explicit N/A markers
- **THEN** docs consistency checks pass without module-readme-richness failure

### Requirement: Mainline contract index SHALL map status parity and module README gates
Mainline contract documentation MUST map status parity and module README richness checks to concrete tests or script entries.

#### Scenario: Maintainer audits governance gate traceability
- **WHEN** maintainer inspects `docs/mainline-contract-test-index.md`
- **THEN** maintainer can identify status parity and module README richness gate paths and corresponding check entries
