## ADDED Requirements

### Requirement: Quality Gate SHALL Include Proposal Example Impact Declaration Check
Standard quality gate flow MUST execute proposal example-impact declaration validation as a blocking step.

Required commands:
- `scripts/check-openspec-example-impact-declaration.sh`
- `scripts/check-openspec-example-impact-declaration.ps1`

#### Scenario: Missing declaration blocks merge
- **WHEN** proposal validation detects missing or invalid example-impact declaration
- **THEN** quality gate exits non-zero and blocks merge

### Requirement: Docs Consistency Gate SHALL Include Roadmap and OpenSpec Status Consistency Check
Repository docs consistency flow MUST execute roadmap/open spec status consistency validation as a blocking step.

Required commands:
- `scripts/check-openspec-roadmap-status-consistency.sh`
- `scripts/check-openspec-roadmap-status-consistency.ps1`

#### Scenario: Status drift blocks merge
- **WHEN** roadmap status disagrees with OpenSpec active/archive sources
- **THEN** docs consistency check exits non-zero and blocks merge

### Requirement: A70 Governance Checks SHALL Preserve Shell and PowerShell Parity
A70 governance checks MUST preserve pass/fail parity across shell and PowerShell for equivalent repository state.

#### Scenario: Equivalent failure on shell and PowerShell paths
- **WHEN** one governance check fails under shell execution
- **THEN** equivalent PowerShell execution yields the same blocking outcome for the same input
