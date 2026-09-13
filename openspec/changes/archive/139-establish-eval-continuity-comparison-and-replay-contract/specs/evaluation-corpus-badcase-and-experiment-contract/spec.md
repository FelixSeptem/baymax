## ADDED Requirements

### Requirement: Experiment evaluation SHALL expose additive continuity comparison

Evaluation comparison MUST be able to associate a bounded continuity comparison with a corpus item or experiment result by stable run/session and checkpoint/reference identity. This association MUST remain nullable and reference-only, MUST NOT change existing corpus or experiment aggregation semantics, and MUST classify continuity drift independently from metric or rubric drift.

#### Scenario: Experiment includes a valid continuity association
- **WHEN** an experiment result references a valid continuity comparison for the same run and corpus item
- **THEN** the result preserves existing metric comparison behavior and exposes the normalized continuity outcome

#### Scenario: Continuity drift does not mutate experiment aggregates
- **WHEN** a continuity comparison reports drift for an otherwise valid experiment shard
- **THEN** the aggregate retains its existing deterministic semantics and reports continuity drift as an additive classification

