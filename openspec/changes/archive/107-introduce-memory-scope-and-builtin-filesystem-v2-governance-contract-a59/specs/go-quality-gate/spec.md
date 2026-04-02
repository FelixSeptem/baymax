## ADDED Requirements

### Requirement: Quality gate SHALL include memory scope and search contract checks
The standard validation flow MUST include memory governance contract checks for scope/search/lifecycle replay semantics.

Required checks MUST include:
- `check-memory-scope-and-search-contract.sh`
- `check-memory-scope-and-search-contract.ps1`

Both shell and PowerShell implementations MUST preserve equivalent blocking semantics (`non-zero exit` => gate failure).

#### Scenario: Memory contract check fails in pull request validation
- **WHEN** memory scope/search contract suite detects fixture drift or semantic mismatch
- **THEN** quality gate exits non-zero and blocks merge

#### Scenario: Memory contract checks run cross-platform
- **WHEN** contributors run shell and PowerShell quality gates
- **THEN** both flows execute equivalent memory contract checks with consistent pass/fail behavior

### Requirement: Memory contract gate SHALL be exposed as independent required-check candidate
CI workflow MUST expose memory governance contract validation as an independent status check suitable for branch protection.

#### Scenario: Maintainer configures branch protection for memory governance
- **WHEN** maintainer reviews available CI status checks
- **THEN** `memory-scope-search-gate` appears as an independent candidate required check
