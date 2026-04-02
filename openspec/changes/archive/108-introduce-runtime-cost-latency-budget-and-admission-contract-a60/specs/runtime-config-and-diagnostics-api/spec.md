## ADDED Requirements

### Requirement: Runtime config SHALL expose budget and degrade-policy fields with deterministic precedence
Runtime configuration MUST expose budget-admission fields with precedence `env > file > default`:
- `runtime.admission.budget.cost.*`
- `runtime.admission.budget.latency.*`
- `runtime.admission.degrade_policy.*`

Invalid threshold ranges, malformed policy definitions, or unsupported enum values MUST fail fast at startup and rollback atomically on hot reload.

#### Scenario: Startup resolves effective budget config from env and file
- **WHEN** both YAML and env provide budget/admission fields
- **THEN** effective config resolves by `env > file > default`

#### Scenario: Hot reload contains invalid budget threshold
- **WHEN** hot reload payload sets malformed budget threshold range
- **THEN** runtime rejects update and preserves previous active snapshot

### Requirement: Runtime diagnostics SHALL expose additive budget-admission fields
Run diagnostics MUST expose additive budget-admission fields:
- `budget_snapshot`
- `budget_decision`
- `degrade_action`

These fields MUST remain compatible under `additive + nullable + default` contract and MUST be emitted via `RuntimeRecorder` single-writer path.

#### Scenario: Consumer inspects run with budget degrade
- **WHEN** run is admitted with degrade decision
- **THEN** diagnostics include canonical `budget_snapshot`, `budget_decision=degrade`, and `degrade_action`

#### Scenario: Consumer inspects run without degrade
- **WHEN** run is admitted with allow decision and no degrade action
- **THEN** diagnostics remain schema-compatible with nullable/default `degrade_action`
