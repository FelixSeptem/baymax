## ADDED Requirements

### Requirement: Runtime SHALL expose scheduler and subagent guardrail config with deterministic precedence
Runtime configuration MUST expose scheduler and subagent guardrail controls with precedence `env > file > default`.

At minimum, this milestone MUST support:
- scheduler enablement,
- scheduler backend selector,
- lease timeout and heartbeat interval,
- queue and retry limits,
- subagent depth and active-child guardrails,
- child execution timeout guardrail.

#### Scenario: Scheduler config is defined in file and environment
- **WHEN** scheduler/subagent config appears in both YAML and environment variables
- **THEN** runtime resolves effective config by `env > file > default` and validates constraints fail-fast

#### Scenario: Invalid scheduler lease configuration at hot reload
- **WHEN** hot reload applies non-positive lease timeout or invalid heartbeat relationship
- **THEN** runtime rejects update and rolls back to previous valid snapshot

### Requirement: Runtime diagnostics SHALL expose additive scheduler and subagent summary fields
Run diagnostics MUST expose additive scheduler/subagent summary fields including backend, queue/claim/reclaim counters, and child-run aggregate counters.

#### Scenario: Consumer inspects scheduler-managed run diagnostics
- **WHEN** a run executes with scheduler-managed subagent dispatch
- **THEN** diagnostics include additive scheduler/subagent fields without breaking existing summary schema

### Requirement: Scheduler diagnostics SHALL remain replay-idempotent
Repeated ingestion of equivalent scheduler/subagent events for the same run MUST NOT inflate logical scheduler counters.

#### Scenario: Duplicate scheduler events are replayed
- **WHEN** scheduler timeline and summary payloads are replayed multiple times for one run
- **THEN** diagnostics preserve stable queue/claim/reclaim and child-run aggregate counters
