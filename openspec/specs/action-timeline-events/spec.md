# action-timeline-events Specification

## Purpose
TBD - created by archiving change standardize-action-timeline-events-h1. Update Purpose after archive.
## Requirements
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

For H1.5 observability convergence, the runtime MUST additionally preserve equivalence of phase-level aggregate distribution between Run and Stream for equivalent scenarios, without requiring byte-level event sequence identity.

#### Scenario: Equivalent successful execution via Run and Stream
- **WHEN** Run and Stream process the same request and both complete successfully
- **THEN** timeline phase/status sequences are semantically equivalent

#### Scenario: Equivalent degraded or failed execution via Run and Stream
- **WHEN** Run and Stream hit the same failure or skip condition
- **THEN** timeline phase/status semantics remain equivalent, including failure/skip reason category

#### Scenario: Equivalent aggregate observability via Run and Stream
- **WHEN** Run and Stream execute equivalent scenarios with timeline aggregation enabled
- **THEN** diagnostics expose semantically equivalent phase-level aggregate distributions for both paths

### Requirement: Timeline adoption SHALL preserve backward compatibility for existing event consumers
The runtime MUST keep existing non-timeline event payloads available during H1 so current consumers are not broken by timeline adoption.

#### Scenario: Legacy consumer reads existing event payload
- **WHEN** an integration reads pre-H1 event fields only
- **THEN** runtime still provides compatible existing fields without requiring timeline migration

### Requirement: Action timeline observability SHALL provide phase-level aggregate metrics per run
The runtime MUST aggregate Action Timeline events into run-level, phase-scoped observability metrics. The minimum metric set per phase MUST include `count_total`, `failed_total`, `canceled_total`, `skipped_total`, `latency_ms`, and `latency_p95_ms`.

#### Scenario: Successful run exposes per-phase aggregate counts and latency
- **WHEN** a run completes with timeline events across one or more phases
- **THEN** diagnostics expose per-phase aggregate metrics including count and latency fields for all active phases

#### Scenario: Phase not activated in run
- **WHEN** a run does not execute a specific phase
- **THEN** diagnostics do not fabricate aggregates for that inactive phase

### Requirement: Action timeline aggregation SHALL be idempotent under replay
For the same run, replayed or duplicated timeline events MUST NOT increase aggregate counters or latency samples more than once.

#### Scenario: Duplicate timeline replay for same run
- **WHEN** timeline events for the same run are submitted more than once due to retry or replay
- **THEN** aggregate metrics remain unchanged after the first logical submission

### Requirement: Action timeline SHALL encode Action Gate reason semantics
When Action Gate is evaluated for tool execution, timeline events MUST expose normalized reason codes for gate control outcomes. At minimum, reason codes MUST include `gate.rule_match`, `gate.require_confirm`, `gate.denied`, and `gate.timeout`.

#### Scenario: Timeline records parameter-rule match reason
- **WHEN** runner hits a parameter-level Action Gate rule
- **THEN** corresponding timeline event includes reason code `gate.rule_match`

#### Scenario: Timeline records confirmation-required reason
- **WHEN** runner marks a tool action as `require_confirm`
- **THEN** corresponding timeline event includes reason code `gate.require_confirm`

#### Scenario: Timeline records denied reason
- **WHEN** gate outcome denies tool execution
- **THEN** corresponding timeline event includes reason code `gate.denied`

#### Scenario: Timeline records timeout reason
- **WHEN** confirmation resolver times out and execution is denied
- **THEN** corresponding timeline event includes reason code `gate.timeout`

### Requirement: Action timeline SHALL encode clarification HITL lifecycle semantics
When clarification HITL is triggered, timeline events MUST expose normalized reason semantics for await/resume/cancel transitions.

#### Scenario: Timeline records await-user transition
- **WHEN** runner enters clarification waiting state
- **THEN** timeline event includes reason code `hitl.await_user`

#### Scenario: Timeline records resumed transition
- **WHEN** runner resumes after receiving clarification
- **THEN** timeline event includes reason code `hitl.resumed`

#### Scenario: Timeline records cancel-by-user transition
- **WHEN** clarification timeout policy resolves to cancel
- **THEN** timeline event includes reason code `hitl.canceled_by_user`

### Requirement: Clarification request event payload SHALL be structured
Clarification events MUST include a structured `clarification_request` payload for direct consumer rendering.

#### Scenario: Consumer reads clarification request event
- **WHEN** runtime emits clarification request event
- **THEN** payload includes at least `request_id`, `questions`, `context_summary`, and `timeout_ms`

### Requirement: Action timeline SHALL expose cancellation-propagation reason semantics
When cancellation storm controls are triggered, action timeline events MUST expose normalized reason semantics indicating cancellation propagation outcomes across execution phases.

#### Scenario: Timeline records cancellation propagation during tool phase
- **WHEN** runner propagates parent cancellation while tool fanout is active
- **THEN** corresponding timeline event includes cancellation-propagation reason semantics and terminal status consistency

#### Scenario: Timeline records cancellation propagation during mcp or skill phase
- **WHEN** runner propagates parent cancellation while mcp or skill work is active
- **THEN** corresponding timeline event includes cancellation-propagation reason semantics aligned with run terminal classification

### Requirement: Action timeline SHALL preserve backpressure observability consistency with diagnostics
Timeline and diagnostics outputs MUST remain semantically consistent for backpressure and cancellation outcomes in the same run.

#### Scenario: Consumer correlates timeline and diagnostics under block policy
- **WHEN** a high-fanout run triggers backpressure with policy `block`
- **THEN** timeline events and run diagnostics present non-conflicting outcome semantics, and `backpressure_drop_count` remains zero

#### Scenario: Consumer correlates timeline and diagnostics under canceled run
- **WHEN** a run is canceled and cancellation is propagated across branches
- **THEN** timeline terminal semantics match diagnostics counters and final run status category

