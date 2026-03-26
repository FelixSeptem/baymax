## ADDED Requirements

### Requirement: Runtime config SHALL expose diagnostics cardinality governance controls with deterministic precedence
Runtime configuration MUST expose diagnostics cardinality governance controls under `diagnostics.cardinality.*` with precedence `env > file > default`.

Minimum required controls and defaults:
- `diagnostics.cardinality.enabled=true`
- `diagnostics.cardinality.max_map_entries=64`
- `diagnostics.cardinality.max_list_entries=64`
- `diagnostics.cardinality.max_string_bytes=2048`
- `diagnostics.cardinality.overflow_policy=truncate_and_record`

`overflow_policy` MUST support:
- `truncate_and_record`
- `fail_fast`

Invalid startup or hot-reload values (unsupported enum, non-positive limits, malformed booleans) MUST fail fast and MUST preserve previous valid active snapshot.

#### Scenario: Runtime starts with default diagnostics cardinality controls
- **WHEN** diagnostics cardinality fields are not explicitly configured
- **THEN** effective runtime config resolves documented defaults with `enabled=true` and `overflow_policy=truncate_and_record`

#### Scenario: Hot reload provides invalid overflow policy
- **WHEN** hot reload sets unsupported `diagnostics.cardinality.overflow_policy`
- **THEN** runtime rejects update and keeps previous active snapshot unchanged

### Requirement: Runtime diagnostics SHALL expose additive cardinality-truncation observability fields
Runtime diagnostics MUST expose additive cardinality-truncation observability fields while preserving compatibility-window semantics (`additive + nullable + default`).

Minimum required fields:
- `diagnostics_cardinality_budget_hit_total`
- `diagnostics_cardinality_truncated_total`
- `diagnostics_cardinality_fail_fast_reject_total`
- `diagnostics_cardinality_overflow_policy`
- `diagnostics_cardinality_truncated_field_summary`

`diagnostics_cardinality_truncated_field_summary` MUST remain bounded-cardinality and MUST use deterministic field-name ordering for equivalent payloads.

#### Scenario: Consumer queries diagnostics after truncation path
- **WHEN** runtime applies truncate-and-record on overflowing diagnostics payload
- **THEN** diagnostics include additive cardinality counters, overflow policy, and bounded truncated-field summary

#### Scenario: Equivalent cardinality events are replayed
- **WHEN** recorder ingests duplicate cardinality overflow events for one run
- **THEN** logical cardinality aggregate counters remain stable after first ingestion
