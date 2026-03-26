## ADDED Requirements

### Requirement: Runtime diagnostics SHALL expose additive cross-domain primary-reason fields
Runtime diagnostics MUST expose additive cross-domain primary-reason fields while preserving compatibility-window semantics (`additive + nullable + default`).

Minimum required fields:
- `runtime_primary_domain`
- `runtime_primary_code`
- `runtime_primary_source`
- `runtime_primary_conflict_total`

Primary-reason fields MUST remain replay-idempotent and bounded-cardinality.

#### Scenario: Consumer queries diagnostics after arbitration
- **WHEN** runtime evaluates cross-domain findings and selects primary reason
- **THEN** diagnostics include additive primary-reason fields with canonical values

#### Scenario: Equivalent arbitration events are replayed
- **WHEN** recorder ingests duplicate arbitration events for one run
- **THEN** primary-reason logical counters remain stable after first ingestion
