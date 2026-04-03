## ADDED Requirements

### Requirement: ReAct Plan Notebook Lifecycle Contract
Runtime SHALL provide a versioned plan notebook lifecycle for ReAct flows with canonical actions `create|revise|complete|recover`.

Notebook entries MUST include deterministic `plan_id`, monotonic `plan_version`, action type, and action reason.

#### Scenario: Initial planning creates notebook entry
- **WHEN** a ReAct run generates its first executable plan
- **THEN** runtime MUST create notebook state with `plan_version=1` and action `create`

#### Scenario: Plan revision increments version
- **WHEN** ReAct loop revises an existing plan
- **THEN** runtime MUST append a `revise` entry and increment `plan_version` monotonically

#### Scenario: Plan completion freezes notebook terminal state
- **WHEN** run reaches completed terminal plan outcome
- **THEN** notebook status MUST transition to completed and reject further mutate actions for the same terminal branch

#### Scenario: Plan recovery rebuilds notebook deterministically
- **WHEN** runtime resumes an interrupted ReAct flow with recoverable notebook snapshot
- **THEN** notebook MUST restore to semantically equivalent active state and append `recover` action

### Requirement: Plan-Change Hook Contract
Runtime SHALL expose deterministic hooks for plan-change boundaries:
- `before_plan_change`
- `after_plan_change`

Hook execution MUST preserve canonical ordering, context propagation, and deterministic failure semantics.

#### Scenario: Before hook blocks plan change under fail-fast mode
- **WHEN** `before_plan_change` returns blocking failure and fail mode is `fail_fast`
- **THEN** runtime MUST abort the current plan mutation and emit canonical failure classification

#### Scenario: After hook runs once after successful mutation
- **WHEN** a plan mutation succeeds
- **THEN** runtime MUST invoke `after_plan_change` exactly once for that mutation

#### Scenario: Hook execution parity for Run and Stream
- **WHEN** equivalent ReAct requests execute in Run and Stream
- **THEN** hook phase counts and terminal hook outcomes MUST remain semantically equivalent

### Requirement: Plan Notebook Actions MUST Be Idempotent Under Replay
Plan notebook action ingestion and recovery MUST remain idempotent under equivalent replay input.

#### Scenario: Replayed recover event does not inflate counters
- **WHEN** equivalent recover action is replayed for the same run and plan version
- **THEN** logical action counters and final notebook state MUST remain stable after first ingestion

#### Scenario: Replayed revise action preserves final state equivalence
- **WHEN** identical revise action payload is replayed multiple times
- **THEN** final notebook state MUST remain semantically equivalent with no duplicate terminal mutation
