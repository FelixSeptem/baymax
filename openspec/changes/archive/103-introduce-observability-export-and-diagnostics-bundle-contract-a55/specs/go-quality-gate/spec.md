## ADDED Requirements

### Requirement: Quality gate SHALL include observability export and diagnostics bundle contract checks
The standard quality validation flow MUST include observability export and diagnostics bundle contract checks.

Repository MUST provide cross-platform gate scripts:
- `scripts/check-observability-export-and-bundle-contract.sh`
- `scripts/check-observability-export-and-bundle-contract.ps1`

Failures in observability contract checks MUST block merge.

#### Scenario: Observability contract check fails in pull request
- **WHEN** CI runs observability export and bundle contract suite and expected semantics diverge from fixtures
- **THEN** observability contract gate exits non-zero and pull request cannot pass required validation

#### Scenario: Observability contract check passes in pull request
- **WHEN** CI runs observability export and bundle contract suite and fixtures match expected semantics
- **THEN** observability contract gate reports success and does not block merge

### Requirement: Observability contract gate SHALL be exposed as independent required-check candidate
CI workflow MUST expose observability export and diagnostics bundle validation as an independent job suitable for branch-protection required status checks.

#### Scenario: Maintainer configures branch protection for observability contract
- **WHEN** maintainer reviews available CI status checks
- **THEN** observability export and diagnostics bundle gate appears as a distinct check that can be configured as required

### Requirement: Observability gate SHALL preserve shell and PowerShell parity
Shell and PowerShell observability contract gate scripts MUST provide equivalent pass/fail semantics, including deterministic failure propagation and exit code behavior.

#### Scenario: Equivalent contract failure under shell and PowerShell
- **WHEN** same observability fixture regression is validated in shell and PowerShell gate flows
- **THEN** both scripts fail deterministically with non-zero exit status
