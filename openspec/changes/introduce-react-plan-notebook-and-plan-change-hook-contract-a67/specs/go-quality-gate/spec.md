## ADDED Requirements

### Requirement: Quality Gate SHALL Include A67 Plan Notebook Contract Checks
Standard quality gate MUST execute A67 contract checks as blocking validations in both shell and PowerShell flows.

Repository MUST provide:
- `scripts/check-react-plan-notebook-contract.sh`
- `scripts/check-react-plan-notebook-contract.ps1`

#### Scenario: Contributor runs shell quality gate
- **WHEN** contributor executes `bash scripts/check-quality-gate.sh`
- **THEN** A67 contract checks run as required blocking steps

#### Scenario: Contributor runs PowerShell quality gate
- **WHEN** contributor executes `pwsh -File scripts/check-quality-gate.ps1`
- **THEN** equivalent A67 contract checks run as required blocking steps

### Requirement: A67 Gate SHALL Fail Fast on Plan Semantics Drift
When A67 suites detect canonical semantic drift, quality gate MUST exit non-zero and block merge.

Semantic drift for this milestone MUST include at minimum:
- plan lifecycle transition drift
- plan-change hook semantic drift
- Run/Stream parity drift
- replay drift classification mismatch

#### Scenario: Plan lifecycle suite detects transition drift
- **WHEN** equivalent fixture or integration inputs produce non-canonical lifecycle transitions
- **THEN** quality gate fails fast and blocks validation completion

#### Scenario: Replay suite detects drift-class mismatch
- **WHEN** `react_plan_notebook.v1` replay validation returns non-canonical drift classification
- **THEN** quality gate fails fast and blocks validation completion

### Requirement: A67 Impacted Contract Suites Enforcement
Gate execution MUST enforce impacted suites for A67 scope changes and MUST reject merges when required suites are missing or failing.

#### Scenario: ReAct scope requires parity suites
- **WHEN** A67 changes touch ReAct loop and plan lifecycle boundaries
- **THEN** gate MUST require Run/Stream parity suites before merge

#### Scenario: Replay scope requires replay suites
- **WHEN** A67 changes touch fixture parser or drift classification logic
- **THEN** gate MUST require replay contract suites before merge
