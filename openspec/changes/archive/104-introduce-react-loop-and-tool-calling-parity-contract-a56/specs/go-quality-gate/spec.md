## ADDED Requirements

### Requirement: Quality gate SHALL include ReAct contract checks in shell and PowerShell flows
Standard quality gate MUST execute ReAct contract checks as blocking validations in both shell and PowerShell paths.

Repository MUST provide:
- `scripts/check-react-contract.sh`
- `scripts/check-react-contract.ps1`

ReAct contract checks MUST cover:
- Run and Stream ReAct loop parity,
- iteration and tool-call budget enforcement,
- provider tool-calling normalization,
- readiness and admission mapping,
- sandbox decision consistency in loop iterations,
- replay fixture validation for `react.v1`.

#### Scenario: Contributor runs shell quality gate
- **WHEN** contributor executes `bash scripts/check-quality-gate.sh`
- **THEN** ReAct contract checks run as required blocking steps

#### Scenario: Contributor runs PowerShell quality gate
- **WHEN** contributor executes `pwsh -File scripts/check-quality-gate.ps1`
- **THEN** equivalent ReAct contract checks run as required blocking steps

### Requirement: Quality gate SHALL fail fast on ReAct semantic drift
When ReAct contract suites detect canonical semantic drift, quality gate MUST exit non-zero and block merge.

Semantic drift for this milestone MUST include at minimum:
- Run and Stream loop parity drift,
- budget enforcement drift,
- provider mapping drift,
- readiness or admission reason taxonomy drift,
- sandbox loop decision drift,
- replay drift classification mismatch.

#### Scenario: ReAct parity suite detects Run Stream divergence
- **WHEN** equivalent ReAct requests produce non-equivalent terminal reason or loop aggregates across Run and Stream
- **THEN** quality gate fails fast and blocks validation completion

#### Scenario: ReAct replay suite detects drift classification mismatch
- **WHEN** `react.v1` fixture validation returns non-canonical drift class for equivalent mismatch
- **THEN** quality gate fails fast and blocks validation completion

### Requirement: CI SHALL expose ReAct contract validation as independent required-check candidate
CI workflow MUST expose ReAct contract validation as a standalone status check suitable for branch protection required-check configuration.

#### Scenario: Maintainer configures branch protection for ReAct contract
- **WHEN** maintainer reviews available CI status checks
- **THEN** ReAct contract gate appears as a distinct required-check candidate

#### Scenario: Independent ReAct gate fails while other jobs pass
- **WHEN** ReAct contract gate job fails and unrelated quality jobs succeed
- **THEN** branch protection can still block merge based on ReAct gate status

### Requirement: ReAct quality gate SHALL enforce docs consistency for contract index and roadmap alignment
Quality gate MUST validate that ReAct-related config fields, diagnostics fields, replay fixtures, and roadmap plus contract-index entries remain synchronized in repository docs.

#### Scenario: ReAct diagnostics field is added without docs update
- **WHEN** docs consistency checks detect missing ReAct field mapping documentation
- **THEN** quality gate exits non-zero and blocks merge

#### Scenario: ReAct roadmap and contract index are synchronized
- **WHEN** docs include aligned ReAct milestone and contract-test index references
- **THEN** docs consistency checks pass without blocking quality gate
