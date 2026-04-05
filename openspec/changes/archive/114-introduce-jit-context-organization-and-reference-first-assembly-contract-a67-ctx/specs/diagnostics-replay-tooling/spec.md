## ADDED Requirements

### Requirement: Replay Tooling SHALL Support A67-CTX Context Fixtures
Diagnostics replay tooling MUST support versioned A67-CTX fixture contracts:
- `context_reference_first.v1`
- `context_isolate_handoff.v1`
- `context_edit_gate.v1`
- `context_relevance_swapback.v1`
- `context_lifecycle_tiering.v1`

Fixture validation MUST cover at minimum:
- reference discovery and selected-resolution semantics,
- isolate handoff ingestion semantics,
- clear-at-least edit-gate decisions,
- swap-back relevance routing semantics,
- lifecycle tier transitions and recap source semantics,
- Run/Stream parity markers.

#### Scenario: Replay validates canonical A67-CTX fixture set
- **WHEN** replay tooling processes valid A67-CTX fixtures and normalized output matches canonical expectation
- **THEN** replay validation MUST succeed with deterministic pass result

#### Scenario: Replay receives malformed A67-CTX fixture schema
- **WHEN** replay tooling receives malformed or unsupported A67-CTX fixture schema
- **THEN** replay validation MUST fail fast with deterministic validation reason

### Requirement: Replay Drift Classification SHALL Include A67-CTX Canonical Classes
Replay tooling MUST classify A67-CTX semantic drift using canonical classes:
- `reference_resolution_drift`
- `isolate_handoff_drift`
- `edit_gate_threshold_drift`
- `swapback_relevance_drift`
- `lifecycle_tiering_drift`
- `recap_semantic_drift`

#### Scenario: Replay detects reference-resolution drift
- **WHEN** replay output diverges from canonical reference selection/resolution semantics
- **THEN** replay validation MUST fail with deterministic `reference_resolution_drift` classification

#### Scenario: Replay detects recap semantic drift
- **WHEN** replay output recap source semantics diverge from fixture expectation
- **THEN** replay validation MUST fail with deterministic `recap_semantic_drift` classification

### Requirement: A67-CTX Fixture Support SHALL Preserve Mixed-Fixture Backward Compatibility
Adding A67-CTX fixture support MUST NOT break validation for historical fixture suites.

#### Scenario: Mixed fixture suites run in one gate flow
- **WHEN** replay gate executes archived fixtures together with A67-CTX fixtures
- **THEN** parser and validation MUST remain deterministic without regression

#### Scenario: Historical parser regression is introduced
- **WHEN** A67-CTX fixture support breaks legacy fixture parsing
- **THEN** replay validation MUST fail and block merge
