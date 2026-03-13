## ADDED Requirements

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

## MODIFIED Requirements

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
