## ADDED Requirements

### Requirement: A72 Anti-Template Gate Is Mandatory
The quality gate SHALL execute `check-agent-mode-anti-template-contract.sh/.ps1` as a required blocking step for agent-mode changes scoped by this contract.

#### Scenario: Template skeleton regression blocks gate
- **WHEN** anti-template validation detects cross-mode structural template regression or wrapper-only semantic ownership
- **THEN** quality gate exits non-zero and blocks merge with deterministic anti-template classification

#### Scenario: Missing mode-owned semantic execution blocks gate
- **WHEN** anti-template validation detects that mode business semantics are not mode-owned
- **THEN** quality gate exits non-zero and blocks merge

### Requirement: A72 Doc-First Delivery Gate Is Mandatory
The quality gate SHALL execute `check-agent-mode-doc-first-delivery-contract.sh/.ps1` as a required blocking step for agent-mode changes scoped by this contract.

#### Scenario: Code change without prior documentation baseline blocks gate
- **WHEN** doc-first validation detects mode semantic code changes without required matrix/readme baseline updates
- **THEN** quality gate exits non-zero and blocks merge

#### Scenario: Missing required readme sections blocks gate
- **WHEN** doc-first validation detects missing required sections in mode readme
- **THEN** quality gate exits non-zero and blocks merge

### Requirement: A72 Task Completion Evidence Integrity SHALL Be Enforced
For this contract scope, quality validation MUST verify that task completion claims are backed by code/test/documentation/gate evidence references.

#### Scenario: Incomplete evidence claim blocks gate
- **WHEN** task completion metadata indicates completion without full evidence coverage
- **THEN** quality gate exits non-zero and blocks merge

### Requirement: A72 Gates SHALL Preserve Shell and PowerShell Parity
A72 anti-template and doc-first gates MUST produce equivalent pass/fail semantics between shell and PowerShell for the same repository state.

#### Scenario: Gate parity is preserved
- **WHEN** A72 gates run on shell and PowerShell paths
- **THEN** pass/fail outcomes remain equivalent
