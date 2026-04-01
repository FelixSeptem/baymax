## ADDED Requirements

### Requirement: Quality gate SHALL include sandbox egress and adapter allowlist contract checks
Standard quality gate MUST execute A57 contract checks as blocking validations in both shell and PowerShell flows.

Repository MUST provide:
- `scripts/check-sandbox-egress-allowlist-contract.sh`
- `scripts/check-sandbox-egress-allowlist-contract.ps1`

#### Scenario: Contributor runs shell quality gate
- **WHEN** contributor executes `bash scripts/check-quality-gate.sh`
- **THEN** A57 contract checks run as required blocking steps

#### Scenario: Contributor runs PowerShell quality gate
- **WHEN** contributor executes `pwsh -File scripts/check-quality-gate.ps1`
- **THEN** equivalent A57 contract checks run as required blocking steps

### Requirement: Quality gate SHALL fail fast on A57 semantic drift
When A57 suites detect canonical semantic drift, quality gate MUST exit non-zero and block merge.

Semantic drift for this milestone MUST include:
- egress policy decision drift,
- allowlist activation drift,
- readiness or admission finding taxonomy drift,
- replay drift classification mismatch.

#### Scenario: Egress contract suite detects policy decision drift
- **WHEN** equivalent fixtures produce non-equivalent egress action semantics
- **THEN** quality gate fails fast and blocks validation completion

#### Scenario: Allowlist suite detects activation taxonomy drift
- **WHEN** allowlist activation outcome taxonomy diverges from canonical contract
- **THEN** quality gate fails fast and blocks validation completion

### Requirement: CI SHALL expose A57 gate as independent required-check candidate
CI workflow MUST expose sandbox egress plus adapter allowlist contract validation as a standalone status check suitable for branch protection.

#### Scenario: Maintainer configures branch protection for A57 contract
- **WHEN** maintainer reviews available CI status checks
- **THEN** A57 contract gate appears as a distinct required-check candidate

#### Scenario: Independent A57 gate fails while other jobs pass
- **WHEN** A57 gate fails and unrelated quality jobs succeed
- **THEN** branch protection can still block merge based on A57 gate status
