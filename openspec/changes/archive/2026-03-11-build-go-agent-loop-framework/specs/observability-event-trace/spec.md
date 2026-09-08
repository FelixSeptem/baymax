## ADDED Requirements

### Requirement: Runner SHALL emit standardized lifecycle events
The system MUST emit a stable minimum lifecycle event set for each run and MUST preserve causal ordering within a run.

#### Scenario: Minimum event coverage
- **WHEN** a run executes from start to finish
- **THEN** events MUST include `run.started`, model lifecycle events, optional tool/MCP events, and `run.finished`

#### Scenario: Event order correctness
- **WHEN** a model step is executed
- **THEN** `model.requested` MUST precede `model.completed` for the same step identifier

### Requirement: Event payload SHALL include correlation identifiers
Each emitted event MUST include identifiers needed to correlate logs, traces, and run artifacts.

#### Scenario: Correlation fields present
- **WHEN** any runtime event is emitted
- **THEN** payload MUST include `run_id` and SHOULD include iteration and call identifiers when applicable

#### Scenario: Tool correlation
- **WHEN** tool lifecycle events are emitted
- **THEN** `tool.requested` and `tool.completed/failed` MUST share the same call identifier

### Requirement: Tracing SHALL provide hierarchical spans
The runtime MUST create OTel spans with `agent.run` as root and nested spans for skill loading, model generation, tool invocation, and MCP calls.

#### Scenario: Root span creation
- **WHEN** a run starts
- **THEN** the runtime MUST open root span `agent.run` and close it at terminal state

#### Scenario: Child span linkage
- **WHEN** a tool invocation occurs
- **THEN** the corresponding `tool.invoke` span MUST be a child of the current run span context

### Requirement: Structured logs SHALL be trace-aware JSON
The system MUST emit JSON logs to stdout containing enough fields to join with traces and event records.

#### Scenario: Trace-aware log line
- **WHEN** runtime logs an operational event
- **THEN** log entry MUST include `trace_id` and `span_id` when span context exists

#### Scenario: Error log consistency
- **WHEN** runtime records classified errors
- **THEN** JSON log MUST include error class and run correlation fields
