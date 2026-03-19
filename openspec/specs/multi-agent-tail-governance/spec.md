# multi-agent-tail-governance Specification

## Purpose
TBD - created by archiving change close-a5-a6-tail-contract-and-governance-a7. Update Purpose after archive.
## Requirements
### Requirement: Tail governance SHALL enforce bounded-cardinality multi-agent observability contract
Tail governance MUST enforce bounded-cardinality semantics for A5/A6 additive observability fields under replay and high-concurrency workloads.

#### Scenario: High-fanout replay emits repeated scheduler/subagent events
- **WHEN** one run replays repeated scheduler/subagent events under retry and takeover paths
- **THEN** aggregate diagnostics remain replay-idempotent and do not grow unbounded by cardinality drift

### Requirement: Tail governance SHALL define additive compatibility window for A5/A6 fields
Tail governance MUST define and publish an explicit compatibility window for newly added A5/A6 fields, including nullable defaults and migration behavior for legacy consumers.

#### Scenario: Legacy consumer parses pre-A5 schema
- **WHEN** legacy consumer reads run diagnostics after A5/A6 rollout
- **THEN** existing semantics remain stable and newly added fields are additive/optional

### Requirement: Tail governance SHALL require docs-and-contract-index convergence
Tail governance MUST require synchronized updates across runtime docs and mainline contract index for A5/A6 closure before merge.

#### Scenario: A5/A6 closure change is proposed
- **WHEN** a closure change updates contract behavior or gate checks
- **THEN** documentation and contract-index entries are updated in the same change set

### Requirement: Tail governance SHALL freeze A12/A13 shared contract taxonomy and correlation constraints
Tail governance MUST freeze the combined A12/A13 reason taxonomy and required correlation markers in one shared contract source.

Minimum taxonomy scope MUST include:
- `a2a.async_submit`
- `a2a.async_report_deliver`
- `a2a.async_report_retry`
- `a2a.async_report_dedup`
- `a2a.async_report_drop`
- `scheduler.delayed_enqueue`
- `scheduler.delayed_wait`
- `scheduler.delayed_ready`

Required correlation scope MUST include scheduler attempt-level keys on scheduler-managed paths.

#### Scenario: Closure change validates A12/A13 reason completeness
- **WHEN** maintainer runs shared multi-agent contract checks after A12/A13 closure updates
- **THEN** missing async or delayed canonical reasons fail validation as blocking regressions

### Requirement: Tail governance SHALL enforce cross-mode contract matrix for communication semantics
Tail governance MUST require a traceable cross-mode matrix covering `sync`, `async`, and `delayed` communication semantics under `Run` and `Stream`, with qos/recovery key paths included.

#### Scenario: Equivalent request executes through mode matrix
- **WHEN** equivalent logical requests run through sync/async/delayed paths in Run and Stream
- **THEN** status/reason/summary semantics remain equivalent and replay-idempotent for required matrix rows

### Requirement: Tail governance SHALL enforce compatibility-window parser semantics for A12/A13 additive fields
Tail governance MUST require parser-level compatibility semantics for A12/A13 additive summary fields using `additive + nullable + default`.

#### Scenario: Legacy consumer parses run summary after A12/A13 rollout
- **WHEN** consumer reads run summary with absent or newly added A12/A13 additive fields
- **THEN** consumer behavior remains stable, missing fields resolve to documented defaults, and existing field semantics remain unchanged

### Requirement: Tail governance SHALL require docs and index convergence for A12/A13 closure
Tail governance MUST require synchronized updates across runtime diagnostics docs, roadmap status, and mainline contract index for A12/A13 closure changes.

#### Scenario: A14 closure change is prepared for merge
- **WHEN** closure change updates gate checks or matrix mappings
- **THEN** docs and contract-test index reflect the same scope in the same change set

