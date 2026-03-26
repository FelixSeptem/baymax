# diagnostics-cardinality-budget-and-truncation-governance Specification

## Purpose
TBD - created by archiving change introduce-diagnostics-cardinality-budget-and-truncation-governance-contract-a45. Update Purpose after archive.
## Requirements
### Requirement: Diagnostics pipeline SHALL enforce bounded cardinality budgets
Diagnostics pipeline MUST enforce configurable cardinality budgets for high-growth field types:
- map entry count
- list entry count
- string byte size

Budget checks MUST run before record persistence and query exposure.

#### Scenario: Record exceeds map entry budget
- **WHEN** diagnostics record contains map field exceeding configured `max_map_entries`
- **THEN** runtime applies configured overflow policy and records deterministic budget-hit observability markers

#### Scenario: Record remains within configured budgets
- **WHEN** diagnostics record fields are all within configured cardinality budgets
- **THEN** record is persisted without truncation markers

### Requirement: Overflow policy SHALL support deterministic truncate or fail-fast behavior
Diagnostics cardinality overflow policy MUST support:
- `truncate_and_record`
- `fail_fast`

Policy behavior MUST be deterministic for equivalent input and configuration.

#### Scenario: Overflow policy is truncate-and-record
- **WHEN** record exceeds configured budget and `overflow_policy=truncate_and_record`
- **THEN** runtime truncates field content deterministically and emits truncation observability fields

#### Scenario: Overflow policy is fail-fast
- **WHEN** record exceeds configured budget and `overflow_policy=fail_fast`
- **THEN** runtime returns deterministic validation error and refuses to persist overflowing record

### Requirement: Truncation output SHALL be replay-stable and mode-equivalent
For equivalent input, equivalent configuration, and equivalent record shape, truncation output MUST remain semantically equivalent across Run and Stream and across replay paths.

#### Scenario: Equivalent truncation in Run and Stream
- **WHEN** equivalent overflowing diagnostics payload is emitted through Run and Stream
- **THEN** truncation result and field-level truncation markers remain semantically equivalent

#### Scenario: Replay preserves truncation aggregates
- **WHEN** equivalent truncated diagnostics events are replayed
- **THEN** logical truncation counters remain stable after first ingestion

