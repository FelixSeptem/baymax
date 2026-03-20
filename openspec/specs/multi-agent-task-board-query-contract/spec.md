# multi-agent-task-board-query-contract Specification

## Purpose
TBD - created by archiving change introduce-task-board-query-contract-a29. Update Purpose after archive.
## Requirements
### Requirement: Task Board query API SHALL provide read-only scheduler task board retrieval
The runtime MUST provide a library-level Task Board query API that returns scheduler task records without mutating scheduler state.

The API MUST support canonical filters: `task_id`, `run_id`, `workflow_id`, `team_id`, `state`, `priority`, `agent_id`, `peer_id`, `parent_run_id`, and `time_range`.

When multiple filters are provided, evaluation MUST use `AND` semantics.

#### Scenario: Query uses multiple filters
- **WHEN** a consumer submits `team_id`, `state`, and `priority` in one request
- **THEN** returned task items satisfy all provided filters at the same time

#### Scenario: Query is read-only
- **WHEN** a consumer executes Task Board query
- **THEN** scheduler queue/task runtime state is not mutated by the query path

### Requirement: Task Board query API SHALL enforce deterministic pagination and sorting defaults
The Task Board query API MUST apply `page_size=50` when caller omits page size.

The query API MUST enforce `page_size <= 200`, and out-of-range values MUST fail fast.

The query API MUST apply default sort `updated_at desc` when caller omits explicit sort options.

The first milestone MUST only accept sort fields `updated_at` and `created_at`.

#### Scenario: Caller omits page size and sort
- **WHEN** query request does not include page size and sort fields
- **THEN** runtime applies `page_size=50` and `updated_at desc`

#### Scenario: Caller uses unsupported sort field
- **WHEN** query request sets sort field outside `updated_at|created_at`
- **THEN** runtime returns validation error immediately

### Requirement: Task Board query API SHALL use opaque cursor pagination
The Task Board query API MUST return opaque cursor strings and MUST NOT expose internal storage offsets or index keys.

Cursor advancement MUST be deterministic for the same logical query boundary.

Malformed cursor or query-boundary mismatch MUST fail fast.

#### Scenario: Query returns next-page cursor
- **WHEN** first page has more items than requested page size
- **THEN** response includes an opaque `next_cursor`

#### Scenario: Query uses invalid cursor
- **WHEN** request includes malformed or boundary-mismatched cursor
- **THEN** runtime returns validation error and does not execute partial pagination

### Requirement: Task Board query API SHALL distinguish invalid input from no-match results
For syntactically valid queries, unmatched filters MUST return an empty result set without error.

Invalid filter values (including unsupported `state` or invalid `time_range`) MUST fail fast.

#### Scenario: Query references non-existent task identifier
- **WHEN** caller submits a syntactically valid `task_id` that has no matching task record
- **THEN** runtime returns empty items and no error

#### Scenario: Query includes invalid time range
- **WHEN** request sets `time_range.start` later than `time_range.end`
- **THEN** runtime returns validation error immediately

### Requirement: Task Board query API SHALL preserve backend semantic parity
For equivalent task snapshots, Task Board query behavior MUST be semantically equivalent across memory and file scheduler backends.

#### Scenario: Equivalent snapshot queried on memory and file backends
- **WHEN** same logical query executes on memory and file scheduler backends with equivalent snapshot data
- **THEN** returned item set, ordering, and cursor traversal semantics remain equivalent

### Requirement: Task Board query state filter SHALL include awaiting-report state
Task Board query state filter MUST support `awaiting_report` as first-class lifecycle state while preserving existing deterministic pagination and cursor semantics.

#### Scenario: Query filters awaiting-report tasks
- **WHEN** caller submits Task Board query with `state=awaiting_report`
- **THEN** runtime returns only tasks in awaiting-report lifecycle with existing sort/page/cursor semantics unchanged

### Requirement: Task Board validation SHALL keep fail-fast behavior for unsupported states
Task Board query validation MUST continue fail-fast behavior for unsupported state values, and `awaiting_report` MUST be treated as supported state.

#### Scenario: Query uses unknown state value
- **WHEN** caller submits `state` outside supported set including `awaiting_report`
- **THEN** runtime returns validation error immediately

