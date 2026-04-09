## ADDED Requirements

### Requirement: A71 Real Runtime Semantic Gate Is Mandatory
The quality gate SHALL execute `check-agent-mode-real-runtime-semantic-contract.sh/.ps1` as a blocking step for a71-scoped example changes.

#### Scenario: Generic template-only implementation blocks gate
- **WHEN** real-runtime-semantic gate detects template-only implementation without mode-specific semantic anchors
- **THEN** quality gate exits non-zero and blocks merge with `agent-mode-semantic-template-regression` classification

#### Scenario: Missing runtime path evidence blocks gate
- **WHEN** real-runtime-semantic gate detects missing mode-required runtime path evidence in execution output
- **THEN** quality gate exits non-zero and blocks merge with `agent-mode-missing-runtime-path-evidence` classification

### Requirement: A71 README Runtime Sync Gate Is Mandatory
The quality gate SHALL execute `check-agent-mode-readme-runtime-sync-contract.sh/.ps1` as a blocking step when a71 changes include agent-mode behavior changes.

#### Scenario: Behavior change without README update blocks gate
- **WHEN** readme-runtime-sync gate detects `main.go` behavior changes without corresponding README updates
- **THEN** quality gate exits non-zero and blocks merge with `agent-mode-readme-runtime-desync` classification

#### Scenario: README required sections missing blocks gate
- **WHEN** readme-runtime-sync gate detects missing required sections (`Run`, `Prerequisites`, `Real Runtime Path`, `Expected Output/Verification`, `Failure/Rollback Notes`)
- **THEN** quality gate exits non-zero and blocks merge with `agent-mode-readme-required-sections-missing` classification

### Requirement: A71 Example Smoke SHALL Validate Dual Variants and Semantic Evidence
The quality gate SHALL enforce smoke validation for both `minimal` and `production-ish` variants and SHALL validate semantic evidence outputs rather than only process exit status.

#### Scenario: Dual-variant smoke is required
- **WHEN** quality gate executes agent-mode smoke for a71
- **THEN** both `minimal` and `production-ish` variants are executed for required modes

#### Scenario: Semantic evidence missing fails smoke
- **WHEN** smoke output lacks required semantic evidence markers for a mode
- **THEN** smoke validation exits non-zero and blocks merge with `agent-mode-smoke-semantic-evidence-missing` classification

### Requirement: A71 Gates SHALL Preserve Shell and PowerShell Parity
A71 gate outcomes MUST remain equivalent between shell and PowerShell for the same repository state and inputs.

#### Scenario: Parity is enforced
- **WHEN** A71 gates run in shell and PowerShell environments
- **THEN** pass/fail outcomes and failure classifications are equivalent
