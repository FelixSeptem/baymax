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

### Requirement: A62 Example Stability Governance Gate SHALL Be Triggered by Baseline Breach
When `agent-mode` smoke stability baselines detect latency or flaky regression beyond configured thresholds, quality gate MUST execute corresponding stability-governance checks as blocking steps.

#### Scenario: Latency regression breach triggers blocking check
- **WHEN** smoke baseline comparison detects latency regression above configured threshold
- **THEN** quality gate runs the A62 stability-governance check and blocks merge on non-zero result with `example-smoke-latency-regression` classification

#### Scenario: Flaky regression breach triggers blocking check
- **WHEN** smoke baseline comparison detects flaky regression above configured threshold
- **THEN** quality gate runs the A62 stability-governance check and blocks merge on non-zero result with `example-smoke-flaky-regression` classification

### Requirement: A62 Real-Logic Contract Gate Is Mandatory
The quality gate SHALL execute agent-mode real-logic validation as a blocking step for A62-scoped example changes.

#### Scenario: Simulated engine dependency blocks gate
- **WHEN** `check-agent-mode-real-logic-contract.sh/.ps1` detects dependency on `examples/agent-modes/internal/agentmode`
- **THEN** quality gate exits non-zero and blocks merge with `agent-mode-simulated-engine-dependency` classification

#### Scenario: Placeholder-only output regression blocks gate
- **WHEN** `check-agent-mode-real-logic-contract.sh/.ps1` detects placeholder-only metadata output without mainline runtime execution evidence
- **THEN** quality gate exits non-zero and blocks merge with `agent-mode-placeholder-output-regression` classification

#### Scenario: Missing mainline runtime path evidence blocks gate
- **WHEN** `check-agent-mode-real-logic-contract.sh/.ps1` cannot find required mainline runtime path usage for an agent-mode entrypoint
- **THEN** quality gate exits non-zero and blocks merge with `agent-mode-missing-mainline-runtime-path` classification

### Requirement: A62 README Sync Contract Gate Is Mandatory
The quality gate SHALL execute readme synchronization validation when agent-mode example behavior changes.

#### Scenario: Behavior change without README update blocks gate
- **WHEN** `check-agent-mode-readme-sync-contract.sh/.ps1` detects `main.go` behavior changes without same-directory `README.md` updates
- **THEN** quality gate exits non-zero and blocks merge with `agent-mode-readme-not-updated` classification

#### Scenario: README missing required sections blocks gate
- **WHEN** `check-agent-mode-readme-sync-contract.sh/.ps1` detects missing `Run`/`Prerequisites`/`Real Runtime Path`/`Expected Output/Verification` sections
- **THEN** quality gate exits non-zero and blocks merge with `agent-mode-readme-missing-required-sections` classification
