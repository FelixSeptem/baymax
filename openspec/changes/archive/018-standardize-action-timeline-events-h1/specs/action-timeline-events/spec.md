## ADDED Requirements

### Requirement: Runtime SHALL emit normalized action timeline events across execution phases
The runtime MUST emit structured Action Timeline events for agent execution with deterministic phase transition semantics. Timeline output MUST cover at least `run`, `context_assembler`, `model`, `tool`, `mcp`, and `skill` phases when those phases are involved in a run.

#### Scenario: Timeline covers active phases in a normal run
- **WHEN** a run executes with context assembly, model generation, and tool invocation
- **THEN** runtime emits timeline events that include `context_assembler`, `model`, and `tool` phases in execution order

#### Scenario: Timeline omits inactive phases without synthetic noise
- **WHEN** a run does not execute MCP or skill loading
- **THEN** timeline output does not emit fake phase events for `mcp` or `skill`

### Requirement: Action timeline status enum SHALL be stable and include cancellation semantics
The runtime MUST use normalized timeline status enums: `pending`, `running`, `succeeded`, `failed`, `skipped`, and `canceled`. Producers and consumers MUST treat these values as canonical status semantics for H1.

#### Scenario: Successful phase transition
- **WHEN** a phase completes without error
- **THEN** the phase timeline status transitions to `succeeded`

#### Scenario: Explicit cancellation transition
- **WHEN** execution is canceled by timeout or caller cancellation path
- **THEN** affected phase or run timeline status is emitted as `canceled`

### Requirement: Run and Stream paths SHALL preserve timeline semantic equivalence
The runtime MUST preserve semantic equivalence of timeline phase/status transitions between Run and Stream paths for equivalent execution outcomes.

#### Scenario: Equivalent successful execution via Run and Stream
- **WHEN** Run and Stream process the same request and both complete successfully
- **THEN** timeline phase/status sequences are semantically equivalent

#### Scenario: Equivalent degraded or failed execution via Run and Stream
- **WHEN** Run and Stream hit the same failure or skip condition
- **THEN** timeline phase/status semantics remain equivalent, including failure/skip reason category

### Requirement: Timeline adoption SHALL preserve backward compatibility for existing event consumers
The runtime MUST keep existing non-timeline event payloads available during H1 so current consumers are not broken by timeline adoption.

#### Scenario: Legacy consumer reads existing event payload
- **WHEN** an integration reads pre-H1 event fields only
- **THEN** runtime still provides compatible existing fields without requiring timeline migration
