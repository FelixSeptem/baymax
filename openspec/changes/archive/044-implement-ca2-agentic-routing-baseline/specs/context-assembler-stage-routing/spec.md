## MODIFIED Requirements

### Requirement: Context assembler SHALL support CA2 two-stage assembly routing
Context assembler MUST execute Stage1 before Stage2. Stage2 invocation MUST be decided by configured CA2 routing mode and MUST remain deterministic and traceable.

When `routing_mode=rules`, Stage2 decision MUST follow existing rule-based routing conditions.
When `routing_mode=agentic`, Stage2 decision MUST follow agentic router callback output, subject to callback failure fallback policy.

#### Scenario: Rules mode skips Stage2 when routing threshold is not met
- **WHEN** `routing_mode=rules` and Stage1 output does not satisfy Stage2 trigger conditions
- **THEN** assembler skips Stage2 and records a normalized skip reason

#### Scenario: Rules mode triggers Stage2 when routing threshold is met
- **WHEN** `routing_mode=rules` and Stage1 output satisfies Stage2 trigger conditions
- **THEN** assembler invokes Stage2 provider and merges Stage2 output into assembled context

#### Scenario: Agentic mode triggers Stage2 from callback decision
- **WHEN** `routing_mode=agentic` and callback returns `run_stage2=true` with valid reason
- **THEN** assembler invokes Stage2 and records router decision metadata

#### Scenario: Agentic mode skips Stage2 from callback decision
- **WHEN** `routing_mode=agentic` and callback returns `run_stage2=false` with valid reason
- **THEN** assembler skips Stage2 and records router decision metadata

### Requirement: Routing engine SHALL provide agentic extension hook placeholder
CA2 routing MUST provide a host callback extension for agentic decisioning. In `routing_mode=agentic`, assembler MUST call the registered callback with bounded timeout.

If callback is missing, times out, returns an error, or returns an invalid decision payload, assembler MUST fallback to `rules` routing under `best_effort` policy and MUST NOT terminate assemble flow solely due to agentic callback failure.

#### Scenario: Agentic callback is available and returns valid decision
- **WHEN** runtime runs CA2 with `routing_mode=agentic` and registered callback returns valid decision
- **THEN** assembler applies callback decision and continues assemble flow

#### Scenario: Agentic callback is not registered
- **WHEN** runtime runs CA2 with `routing_mode=agentic` and no callback is registered
- **THEN** assembler falls back to `rules` routing and records fallback reason

#### Scenario: Agentic callback times out
- **WHEN** runtime runs CA2 with `routing_mode=agentic` and callback exceeds configured timeout
- **THEN** assembler falls back to `rules` routing, records timeout reason, and continues assemble flow

#### Scenario: Agentic callback returns error or invalid payload
- **WHEN** runtime runs CA2 with `routing_mode=agentic` and callback returns error or invalid decision payload
- **THEN** assembler falls back to `rules` routing, records normalized router error, and continues assemble flow

## ADDED Requirements

### Requirement: CA2 routing decisions SHALL remain semantically equivalent between Run and Stream
For equivalent inputs and effective configuration, Run and Stream MUST produce semantically equivalent CA2 routing outcomes in both `rules` and `agentic` modes, allowing implementation-level event ordering differences.

#### Scenario: Equivalent callback-driven decision in Run and Stream
- **WHEN** equivalent requests execute in `routing_mode=agentic` with the same callback behavior
- **THEN** Run and Stream expose semantically equivalent Stage2 invoke/skip outcomes and router reason classes

#### Scenario: Equivalent callback failure fallback in Run and Stream
- **WHEN** equivalent requests execute in `routing_mode=agentic` and callback path fails with the same failure class
- **THEN** Run and Stream both fallback to `rules` and expose semantically equivalent fallback reason classes
