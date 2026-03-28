## ADDED Requirements

### Requirement: Diagnostics-query benchmark matrix SHALL cover sandbox-enriched run summaries
Diagnostics-query performance baseline MUST include sandbox-enriched run-summary dataset coverage to detect regression introduced by sandbox additive fields.

Sandbox-enriched coverage MUST include at minimum:
- sandbox decision fields populated,
- sandbox fallback markers populated,
- sandbox failure counters populated.

#### Scenario: QueryRuns benchmark executes sandbox-enriched dataset
- **WHEN** diagnostics-query benchmark suite runs with default dataset generator
- **THEN** QueryRuns benchmark includes sandbox-enriched records in deterministic workload composition

#### Scenario: Sandbox field growth causes threshold breach
- **WHEN** sandbox-related additive fields introduce measurable regression beyond configured threshold
- **THEN** diagnostics-query regression gate fails and blocks validation

