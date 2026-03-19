## ADDED Requirements

### Requirement: Quality gate SHALL include pre-1 governance consistency checks
The standard quality gate MUST validate pre-1 governance consistency across roadmap and versioning documentation when repository remains in `0.x` phase.

This validation MUST run through repository docs consistency paths for both shell and PowerShell workflows.

#### Scenario: Contributor runs docs consistency in shell path
- **WHEN** contributor executes `bash scripts/check-docs-consistency.sh`
- **THEN** pre-1 governance consistency checks are executed as required validation

#### Scenario: Contributor runs docs consistency in PowerShell path
- **WHEN** contributor executes `pwsh -File scripts/check-docs-consistency.ps1`
- **THEN** equivalent pre-1 governance consistency checks are executed

### Requirement: Governance consistency check SHALL fail fast on stage-conflict drift
If governance docs contain semantic conflicts between pre-1 posture and stable-release claims, the docs consistency check MUST fail fast and return non-zero status.

#### Scenario: Roadmap claims stable-release posture while versioning remains pre-1
- **WHEN** docs consistency check detects conflicting release-stage semantics
- **THEN** validation exits non-zero and blocks merge

#### Scenario: Governance docs remain semantically aligned
- **WHEN** roadmap and versioning docs consistently express pre-1 posture
- **THEN** docs consistency validation passes without governance-stage failure
