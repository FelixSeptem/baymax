## ADDED Requirements

### Requirement: Scheduler SHALL resolve child timeout via shared operation-profile resolver
Scheduler child-dispatch path MUST consume shared timeout resolution output derived from operation profile precedence and MUST NOT apply path-local ad-hoc timeout overrides.

Resolved child timeout MUST be persisted into task attempt metadata for query/replay parity.

#### Scenario: Scheduler enqueues child with resolved timeout metadata
- **WHEN** composer submits child task with operation profile and optional overrides
- **THEN** scheduler stores resolved timeout and resolution source in attempt metadata

#### Scenario: Path-local timeout override is attempted
- **WHEN** scheduler path attempts to bypass shared resolver with unsupported local override
- **THEN** scheduler fails fast with deterministic validation error and no partial enqueue mutation

### Requirement: Scheduler child timeout SHALL converge with parent remaining budget deterministically
Before enqueue/claim activation, scheduler MUST apply parent-budget convergence using shared clamp rule.

If convergence detects exhausted parent budget, scheduler MUST reject child task activation with canonical timeout-budget rejection classification.

#### Scenario: Parent budget clamp is applied during enqueue
- **WHEN** parent remaining budget is lower than child resolved timeout
- **THEN** scheduler stores clamped timeout and marks convergence reason in diagnostics metadata

#### Scenario: Parent budget is exhausted at activation boundary
- **WHEN** parent remaining budget is non-positive
- **THEN** scheduler rejects child activation and emits canonical timeout-budget reject reason

### Requirement: Scheduler timeout-resolution metadata SHALL remain recovery-compatible
Scheduler restore and replay paths MUST preserve timeout-resolution metadata and convergence semantics without changing terminal commit idempotency behavior.

#### Scenario: Restore keeps timeout-resolution metadata stable
- **WHEN** scheduler restores from snapshot containing timeout-resolution metadata
- **THEN** query and subsequent claim behavior observe same effective timeout semantics as pre-restore path

#### Scenario: Replay does not inflate convergence counters
- **WHEN** equivalent convergence events are replayed after restore
- **THEN** scheduler diagnostics counters remain logically idempotent
