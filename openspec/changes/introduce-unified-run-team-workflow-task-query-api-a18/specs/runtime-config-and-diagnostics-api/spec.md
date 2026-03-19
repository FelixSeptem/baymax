## ADDED Requirements

### Requirement: Runtime diagnostics API SHALL expose unified multi-dimensional query entrypoint
The runtime diagnostics manager and its backing store MUST expose a unified query entrypoint for `run_id`, `team_id`, `workflow_id`, `task_id`, `status`, and `time_range` filtering.

The unified query entrypoint MUST be available by default without introducing a dedicated feature flag for this capability.

#### Scenario: Consumer uses unified query API from runtime manager
- **WHEN** application calls the new diagnostics query entrypoint with supported filters
- **THEN** runtime manager executes the query and returns normalized result payload

#### Scenario: Runtime starts with default configuration
- **WHEN** no query-feature toggle is provided in config
- **THEN** unified query API remains available as standard diagnostics capability

### Requirement: Runtime diagnostics compatibility SHALL preserve existing Recent APIs
Adding unified query capability MUST NOT break or remove existing `RecentRuns`, `RecentCalls`, or `RecentSkills` behaviors.

New diagnostics response fields introduced for unified query MUST follow `additive + nullable + default` compatibility semantics.

#### Scenario: Legacy consumer keeps using Recent APIs
- **WHEN** consumer only invokes existing `RecentRuns/RecentCalls/RecentSkills` APIs
- **THEN** behavior and field semantics remain backward compatible

#### Scenario: Legacy parser reads unified-query-capable diagnostics payload
- **WHEN** diagnostics payload contains newly added unified-query-related fields
- **THEN** older consumers can safely ignore unknown optional fields without semantic regression

### Requirement: Runtime diagnostics query validation SHALL fail fast for invalid parameters
Invalid query inputs MUST be rejected before execution, including malformed cursor, invalid status filter value, out-of-range page size, and invalid time range boundaries.

For syntactically valid but unmatched `task_id`, runtime MUST return empty result set rather than returning parameter error.

#### Scenario: Consumer sends invalid page size
- **WHEN** query request sets `page_size` outside supported range
- **THEN** runtime diagnostics returns fail-fast validation error

#### Scenario: Consumer sends non-existent but valid task identifier
- **WHEN** query request includes a valid `task_id` format that matches no records
- **THEN** runtime diagnostics returns empty items with no error

