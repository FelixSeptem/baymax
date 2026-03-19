# multi-agent-unified-query-api Specification

## Purpose
TBD - created by archiving change introduce-unified-run-team-workflow-task-query-api-a18. Update Purpose after archive.
## Requirements
### Requirement: Unified query API SHALL support canonical multi-dimensional filters
The runtime diagnostics unified query API MUST accept `run_id`, `team_id`, `workflow_id`, `task_id`, `status`, and `time_range` filters in a single request model.

When multiple filters are provided, filter evaluation MUST use `AND` semantics.

#### Scenario: Query uses multiple filters
- **WHEN** a consumer submits `team_id`, `workflow_id`, and `status` in one request
- **THEN** returned records satisfy all provided filters at the same time

#### Scenario: Query uses one filter only
- **WHEN** a consumer submits only `run_id`
- **THEN** returned records are filtered by `run_id` and no implicit extra filter is applied

### Requirement: Unified query API SHALL enforce deterministic pagination and sorting defaults
The query API MUST apply `page_size=50` when caller input omits page size.

The query API MUST enforce `page_size <= 200`, and values outside valid range MUST fail fast.

The query API MUST apply default sorting `time desc` when caller input omits explicit sort fields.

#### Scenario: Caller omits page size and sort
- **WHEN** query request does not include pagination or sort options
- **THEN** runtime uses `page_size=50` and `time desc` for result ordering

#### Scenario: Caller exceeds page size limit
- **WHEN** query request sets `page_size` greater than `200`
- **THEN** runtime returns validation error immediately and does not execute the query

### Requirement: Unified query API SHALL use opaque cursor pagination
The query API MUST return opaque cursor strings and MUST NOT expose internal storage offsets or index keys.

For the same logical query boundary, cursor advancement MUST produce deterministic next-page traversal.

#### Scenario: Query returns next-page cursor
- **WHEN** first page contains more records than the requested page size
- **THEN** response includes an opaque cursor for next-page retrieval

#### Scenario: Query uses invalid cursor
- **WHEN** request includes malformed or non-decodable cursor
- **THEN** runtime returns fail-fast validation error

### Requirement: Unified query API SHALL distinguish invalid input from no-match results
For syntactically valid queries, unmatched filters MUST return an empty result set without error.

Specifically, when `task_id` is valid as a parameter but does not exist in stored diagnostics, runtime MUST return empty items and MUST NOT return an error.

#### Scenario: Query references non-existent task identifier
- **WHEN** caller submits a syntactically valid `task_id` that has no matching record
- **THEN** runtime returns an empty result set with no error

#### Scenario: Query includes invalid time range
- **WHEN** request sets `time_range.start` later than `time_range.end`
- **THEN** runtime returns fail-fast validation error instead of returning partial data

