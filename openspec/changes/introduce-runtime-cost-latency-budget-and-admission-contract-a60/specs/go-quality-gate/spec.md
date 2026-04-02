## ADDED Requirements

### Requirement: Quality gate SHALL include runtime budget-admission contract checks
The standard validation flow MUST include budget-admission contract checks:
- `check-runtime-budget-admission-contract.sh`
- `check-runtime-budget-admission-contract.ps1`

Shell and PowerShell checks MUST preserve equivalent blocking semantics (`non-zero exit` => gate failure).

#### Scenario: Budget-admission contract check fails
- **WHEN** budget contract suite detects threshold or decision drift
- **THEN** quality gate exits non-zero and blocks merge

#### Scenario: Cross-platform gate parity for budget checks
- **WHEN** contributors run shell and PowerShell quality gates
- **THEN** both flows execute equivalent budget contract checks with consistent pass/fail semantics

### Requirement: Budget-admission gate SHALL be exposed as independent required-check candidate
CI workflow MUST expose budget-admission validation as an independent status check suitable for branch protection.

#### Scenario: Maintainer configures branch protection for budget admission
- **WHEN** maintainer reviews available CI status checks
- **THEN** `runtime-budget-admission-gate` appears as an independent candidate required check

### Requirement: Budget-admission gate SHALL enforce same-domain closure guardrails
Budget-admission contract gate MUST enforce guardrails that prevent domain split drift:
- `budget_control_plane_absent`
- `budget_field_reuse_required`

#### Scenario: Gate detects control-plane dependency drift
- **WHEN** budget-admission checks detect hosted admission control-plane dependency
- **THEN** gate exits non-zero and blocks merge

#### Scenario: Gate detects parallel same-meaning field drift
- **WHEN** budget-admission checks detect duplicate same-meaning fields that redefine canonical A58/A59 semantics
- **THEN** gate exits non-zero and blocks merge
