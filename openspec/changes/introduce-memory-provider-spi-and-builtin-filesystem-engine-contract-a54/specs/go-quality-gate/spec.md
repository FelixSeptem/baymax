## ADDED Requirements

### Requirement: Quality gate SHALL include memory contract checks in shell and PowerShell flows
Standard quality gate MUST execute memory contract checks as blocking validations in both shell and PowerShell paths.

Memory contract checks MUST include:
- memory adapter conformance matrix suites,
- memory readiness finding contract suites,
- memory replay fixture suites.

#### Scenario: Contributor runs shell quality gate
- **WHEN** contributor executes `bash scripts/check-quality-gate.sh`
- **THEN** memory contract checks run as required blocking steps

#### Scenario: Contributor runs PowerShell quality gate
- **WHEN** contributor executes `pwsh -File scripts/check-quality-gate.ps1`
- **THEN** equivalent memory contract checks run as required blocking steps

### Requirement: Quality gate SHALL fail fast on memory semantic drift
When memory contract suites detect canonical semantic drift, quality gate MUST exit non-zero and block merge.

Semantic drift for this milestone MUST include at minimum:
- mode and profile mismatch,
- fallback policy drift,
- reason taxonomy drift,
- replay aggregate drift.

#### Scenario: Memory conformance detects profile mismatch drift
- **WHEN** memory matrix suite detects profile behavior inconsistent with canonical contract
- **THEN** quality gate fails fast with deterministic memory drift classification

#### Scenario: Memory replay detects taxonomy drift
- **WHEN** replay suite detects memory reason code taxonomy mismatch
- **THEN** quality gate fails fast and blocks validation completion

### Requirement: CI SHALL expose memory contract validation as independent required-check candidate
CI workflow MUST expose memory contract validation as a standalone status check suitable for branch protection required-check configuration.

#### Scenario: Maintainer configures branch protection for memory contract
- **WHEN** maintainer reviews available CI status checks
- **THEN** memory contract gate appears as a distinct required-check candidate

#### Scenario: Independent memory gate fails
- **WHEN** memory contract gate job fails while other quality jobs pass
- **THEN** branch protection can still block merge based on memory gate status

### Requirement: Quality gate SHALL enforce docs consistency for memory contract index and roadmap entries
Quality gate MUST validate that memory-related config fields, diagnostics fields, conformance suites, and roadmap milestone entries remain synchronized in repository docs.

#### Scenario: Memory diagnostics field is added without docs update
- **WHEN** docs consistency checks detect missing memory field mapping documentation
- **THEN** quality gate exits non-zero and blocks merge

#### Scenario: Memory roadmap and contract index are synchronized
- **WHEN** docs include aligned memory milestone and contract test index references
- **THEN** docs consistency checks pass without blocking quality gate
