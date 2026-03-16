## ADDED Requirements

### Requirement: CI workflow SHALL pin critical lint tooling version
The default CI workflow MUST use a pinned `golangci-lint` version rather than floating `latest` to ensure reproducible validation behavior across time.

#### Scenario: CI executes lint job on different dates
- **WHEN** the same commit is validated at different times
- **THEN** lint pass/fail semantics remain stable unless repository-owned version pin is intentionally changed

### Requirement: CI workflow SHALL avoid duplicated quality-gate stages
The default CI workflow MUST avoid duplicate execution of repository hygiene checks when the quality-gate script already includes the same check.

#### Scenario: Workflow runs standard validation
- **WHEN** CI executes the default quality gate
- **THEN** repository hygiene is executed exactly once in the canonical validation path

### Requirement: CI workflow SHALL declare least-privilege permissions and timeout
The default CI workflow MUST explicitly declare minimum required GitHub Actions permissions and define job timeout to prevent unbounded execution.

#### Scenario: Workflow job starts on pull request
- **WHEN** CI job initializes
- **THEN** the job runs under explicit least-privilege permissions and bounded timeout settings
