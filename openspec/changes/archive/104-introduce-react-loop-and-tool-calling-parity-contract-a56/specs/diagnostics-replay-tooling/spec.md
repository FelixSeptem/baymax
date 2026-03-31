## ADDED Requirements

### Requirement: Replay tooling SHALL support ReAct fixture contract version react.v1
Diagnostics replay tooling MUST support versioned ReAct fixture contract `react.v1`.

`react.v1` fixture validation MUST cover at minimum:
- loop step sequence,
- iteration and tool-call counters,
- terminal reason classification,
- Stream dispatch parity markers,
- provider tool-calling normalization summary.

#### Scenario: Replay validates canonical react.v1 fixture
- **WHEN** tooling replays valid `react.v1` fixture with expected canonical output
- **THEN** replay validation succeeds with deterministic pass result

#### Scenario: Replay receives malformed react.v1 schema
- **WHEN** tooling receives malformed or unsupported `react.v1` fixture payload
- **THEN** replay fails fast with deterministic validation reason code

### Requirement: Replay drift classification SHALL include ReAct-specific drift classes
Replay tooling MUST classify ReAct semantic drift using canonical classes:
- `react_loop_step_drift`
- `react_tool_call_budget_drift`
- `react_iteration_budget_drift`
- `react_termination_reason_drift`
- `react_stream_dispatch_drift`
- `react_provider_mapping_drift`

#### Scenario: Replay detects terminal reason drift
- **WHEN** actual replay output termination reason differs from fixture expectation
- **THEN** replay validation fails with deterministic `react_termination_reason_drift` classification

#### Scenario: Replay detects stream dispatch parity drift
- **WHEN** replay output indicates Stream dispatch semantics diverge from canonical fixture expectation
- **THEN** replay validation fails with deterministic `react_stream_dispatch_drift` classification

### Requirement: ReAct fixture support SHALL preserve backward-compatible mixed-fixture validation
Adding `react.v1` support MUST NOT break existing fixture generations and mixed fixture replay flows.

#### Scenario: Mixed fixture suite includes A52 A53 memory v1 observability v1 and react.v1
- **WHEN** replay gate runs mixed fixture suite containing historical fixtures and `react.v1`
- **THEN** all fixture generations are parsed and validated deterministically without parser regression

#### Scenario: Historical fixture parser regression is introduced by react.v1 changes
- **WHEN** replay tooling update for `react.v1` breaks historical fixture parsing
- **THEN** replay validation fails and blocks merge
