## ADDED Requirements

### Requirement: Diagnostics Query Hotpath Regression Coverage
Diagnostics query performance governance SHALL cover A64 query hotspots, including QueryRuns, QueryMailbox, MailboxAggregates, and trend/percentile aggregation paths.

#### Scenario: Query hotspot benchmarks are present
- **WHEN** diagnostics query baseline validation runs
- **THEN** benchmark coverage includes QueryRuns, QueryMailbox, MailboxAggregates, and trend/percentile aggregation paths

#### Scenario: Regression in query hotspots is blocked
- **WHEN** diagnostics query benchmark metrics exceed configured regression thresholds
- **THEN** diagnostics performance checks fail and block merge

### Requirement: Query Optimization Preserves Result Semantics
Diagnostics query optimizations SHALL preserve filtering, sorting, pagination, and aggregate interpretation semantics.

#### Scenario: Query result equivalence under optimized path
- **WHEN** query hotpath optimizations are enabled
- **THEN** result ordering, cursor behavior, and aggregate values remain semantically equivalent to baseline behavior

#### Scenario: Percentile/trend optimization preserves interpretation
- **WHEN** percentile or trend computation paths are optimized
- **THEN** output schema and interpretation remain compatible with existing contract consumers

### Requirement: Diagnostics Query Regression Suite Is Mandatory for A64
The diagnostics query performance regression suite MUST be executed as a required gate step for A64 changes that touch diagnostics/recorder query hotspots.

#### Scenario: Diagnostics query regression suite is wired as blocker
- **WHEN** A64 validation includes S2 diagnostics query optimizations
- **THEN** `check-diagnostics-query-performance-regression.sh/.ps1` executes as a blocking step
