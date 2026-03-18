## ADDED Requirements

### Requirement: Runtime diagnostics SHALL enforce bounded-cardinality for scheduler/subagent additive fields
Runtime diagnostics MUST enforce bounded-cardinality aggregation for scheduler/subagent additive fields and keep replay-idempotent semantics.

#### Scenario: Repeated takeover events are ingested
- **WHEN** repeated takeover-related events for the same run are ingested
- **THEN** queue/claim/reclaim counters remain stable after first logical ingestion

### Requirement: Runtime diagnostics contract SHALL publish compatibility-window guidance for A5/A6 fields
Diagnostics contract documentation MUST specify compatibility-window guidance for A5/A6 additive fields, including nullability, defaults, and consumer migration expectations.

#### Scenario: Consumer audits A5/A6 field behavior
- **WHEN** consumer reads diagnostics contract documentation after A5/A6 closure
- **THEN** compatibility-window semantics are explicit and testable
