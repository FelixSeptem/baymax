## ADDED Requirements

### Requirement: Runtime diagnostics SHALL expose additive arbitration explainability fields
Runtime diagnostics MUST expose additive arbitration explainability fields while preserving compatibility-window semantics (`additive + nullable + default`).

Minimum required fields:
- `runtime_secondary_reason_codes`
- `runtime_secondary_reason_count`
- `runtime_arbitration_rule_version`
- `runtime_remediation_hint_code`
- `runtime_remediation_hint_domain`

Explainability fields MUST remain bounded-cardinality and replay-idempotent.

#### Scenario: Consumer queries run diagnostics after arbitration explainability is enabled
- **WHEN** runtime emits arbitration explainability payload
- **THEN** diagnostics include additive explainability fields with canonical values

#### Scenario: Equivalent explainability events are replayed
- **WHEN** recorder ingests duplicate explainability events for one run
- **THEN** logical explainability aggregates remain stable after first ingestion
