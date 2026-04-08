## ADDED Requirements

### Requirement: A62 Agent Mode Smoke Gate Is Mandatory
The quality gate SHALL execute agent mode smoke validation as a blocking step for A62-scoped example changes.

#### Scenario: Smoke gate blocks on runnable failure
- **WHEN** `check-agent-mode-examples-smoke.sh/.ps1` reports a failed required mode run
- **THEN** quality gate validation fails and merge is blocked

### Requirement: A62 Pattern Coverage Gate Is Mandatory
The quality gate SHALL execute pattern coverage validation to ensure required mode families and matrix mappings are complete.

#### Scenario: Missing required mode family blocks gate
- **WHEN** `check-agent-mode-pattern-coverage.sh/.ps1` detects missing required mode families or incomplete matrix rows
- **THEN** quality gate exits non-zero and blocks merge

### Requirement: A62 Migration Playbook Consistency Gate Is Mandatory
The quality gate SHALL execute migration playbook consistency validation between `MATRIX.md`, `PLAYBOOK.md`, and `production-ish` readme `prod delta` sections.

#### Scenario: Missing playbook mapping blocks gate
- **WHEN** `check-agent-mode-migration-playbook-consistency.sh/.ps1` detects missing checklist mapping or missing required gate references
- **THEN** quality gate exits non-zero and blocks merge with `missing-checklist` or `missing-gate` classification

### Requirement: A62 Example Gates SHALL Preserve Shell and PowerShell Parity
A62 example-related gates MUST preserve pass/fail parity across shell and PowerShell for equivalent repository state.

#### Scenario: Gate parity is enforced
- **WHEN** A62 quality gate steps are executed on shell and PowerShell environments
- **THEN** pass/fail outcomes are equivalent for the same inputs and fixtures

### Requirement: A62 Legacy Example TODO Cleanup Gate Is Mandatory
The quality gate SHALL execute legacy example placeholder cleanup validation for A62-scoped changes.

#### Scenario: Legacy TODO cleanup gate blocks unresolved placeholders
- **WHEN** `check-agent-mode-legacy-todo-cleanup.sh/.ps1` detects `TODO/TBD/FIXME/待补` markers in `examples/`
- **THEN** quality gate exits non-zero and blocks merge

### Requirement: A62 Context-Governed Validation SHALL Require A69 Context Compression Gates
When A62 changes touch `context-governed` example scope, quality gate MUST execute A69 context compression and A67-CTX context organization contract checks as blocking steps.

#### Scenario: Context-governed example change triggers A69 and A67-CTX gates
- **WHEN** changed files map to `examples/agent-modes/context-governed-reference-first` or equivalent context-governed paths
- **THEN** `check-context-compression-production-contract.sh/.ps1` and `check-context-jit-organization-contract.sh/.ps1` are both required blocking steps

#### Scenario: A69 gate failure blocks A62 context-governed completion
- **WHEN** context-governed validation runs and A69 contract checks fail
- **THEN** quality gate exits non-zero and blocks merge
