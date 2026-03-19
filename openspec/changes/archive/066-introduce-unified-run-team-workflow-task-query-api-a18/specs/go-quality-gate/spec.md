## ADDED Requirements

### Requirement: Shared quality gate SHALL include unified query contract suites
The shared multi-agent quality gate MUST include blocking contract suites for unified diagnostics query behavior.

The suites MUST run in repository gate scripts for both shell and PowerShell flows.

#### Scenario: CI runs shared quality gate after unified query rollout
- **WHEN** CI executes shared multi-agent contract gate scripts
- **THEN** unified query contract suites run as required blocking checks

#### Scenario: Local contributor runs PowerShell shared gate
- **WHEN** contributor executes `pwsh -File scripts/check-multi-agent-shared-contract.ps1`
- **THEN** unified query contract suites are executed with equivalent blocking semantics

### Requirement: Unified query contract suites SHALL enforce canonical query semantics
Contract tests MUST cover at least:
- multi-filter `AND` semantics,
- default pagination `page_size=50`,
- maximum page size `200` with fail-fast on invalid values,
- default sort `time desc`,
- opaque cursor pagination behavior and invalid cursor fail-fast behavior,
- non-existent `task_id` returns empty result set without error.

#### Scenario: Regression changes filter semantics to OR
- **WHEN** implementation returns records matching any filter instead of all filters
- **THEN** contract suite fails and blocks merge

#### Scenario: Regression changes missing task behavior to error
- **WHEN** implementation returns error for unmatched but syntactically valid `task_id`
- **THEN** contract suite fails and blocks merge

### Requirement: Mainline contract index SHALL trace unified query coverage
Repository documentation and contract index MUST include traceable mapping from unified query semantic rows to concrete test cases and gate script entries.

#### Scenario: Contributor audits unified query coverage
- **WHEN** contributor inspects mainline contract index after A18
- **THEN** each required unified query semantic row maps to concrete test and gate path

