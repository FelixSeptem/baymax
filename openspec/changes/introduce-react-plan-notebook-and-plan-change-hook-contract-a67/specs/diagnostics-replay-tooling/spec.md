## ADDED Requirements

### Requirement: Replay Tooling SHALL Support A67 Plan Notebook Fixture
Diagnostics replay tooling MUST support versioned fixture contract `react_plan_notebook.v1`.

Fixture validation MUST cover at minimum:
- notebook action sequence semantics (`create|revise|complete|recover`)
- plan version progression
- plan-change hook outcome semantics
- Run/Stream parity markers

#### Scenario: Replay validates canonical A67 fixture
- **WHEN** replay tooling processes valid `react_plan_notebook.v1` fixture and normalized output matches canonical expectation
- **THEN** replay validation succeeds with deterministic pass result

#### Scenario: Replay receives malformed A67 fixture schema
- **WHEN** replay tooling receives malformed or unsupported `react_plan_notebook.v1` schema
- **THEN** replay validation fails fast with deterministic validation reason code

### Requirement: Replay Drift Classification SHALL Include A67 Canonical Classes
Replay tooling MUST classify A67 semantic drift using canonical classes:
- `react_plan_version_drift`
- `react_plan_change_reason_drift`
- `react_plan_hook_semantic_drift`
- `react_plan_recover_drift`

#### Scenario: Replay detects plan version drift
- **WHEN** replay output plan version progression diverges from fixture expectation
- **THEN** replay validation fails with deterministic `react_plan_version_drift` classification

#### Scenario: Replay detects hook semantic drift
- **WHEN** replay output hook outcome semantics diverge from fixture expectation
- **THEN** replay validation fails with deterministic `react_plan_hook_semantic_drift` classification

### Requirement: A67 Fixture Support SHALL Preserve Mixed-Fixture Backward Compatibility
Adding `react_plan_notebook.v1` support MUST NOT break validation for historical fixture suites.

#### Scenario: Mixed fixture suites run in one gate flow
- **WHEN** replay gate executes archived fixtures together with `react_plan_notebook.v1`
- **THEN** parser and validation remain deterministic without regression

#### Scenario: Historical parser regression is introduced
- **WHEN** A67 fixture support breaks legacy fixture parsing
- **THEN** replay validation fails and blocks merge
