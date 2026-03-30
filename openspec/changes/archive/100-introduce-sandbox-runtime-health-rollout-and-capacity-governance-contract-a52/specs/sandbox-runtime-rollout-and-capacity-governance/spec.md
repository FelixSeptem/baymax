## ADDED Requirements

### Requirement: Runtime SHALL provide deterministic sandbox rollout phase governance
Runtime MUST expose canonical sandbox rollout phases:
- `observe`
- `canary`
- `baseline`
- `full`
- `frozen`

Runtime MUST enforce deterministic legal transitions:
- `observe -> canary`
- `canary -> baseline`
- `baseline -> full`
- any active phase (`canary|baseline|full`) -> `frozen`
- `frozen -> canary|observe`

Illegal transitions MUST fail fast during startup or hot reload activation.

#### Scenario: Valid rollout transition from canary to baseline
- **WHEN** effective configuration changes rollout phase from `canary` to `baseline`
- **THEN** runtime accepts configuration and applies deterministic rollout transition semantics

#### Scenario: Invalid rollout transition from full to observe
- **WHEN** effective configuration attempts direct transition `full -> observe`
- **THEN** runtime fails fast and preserves previous active configuration snapshot

### Requirement: Runtime SHALL evaluate sandbox health budgets using canonical SLI set
Runtime MUST evaluate rollout health against canonical SLI inputs:
- launch failure rate
- timeout rate
- violation rate
- p95 latency delta
- admission deny rate

Runtime MUST classify budget evaluation deterministically as:
- `within_budget`
- `near_budget`
- `breached`

#### Scenario: Health SLI stays within configured budget
- **WHEN** canonical SLI values remain below configured thresholds in evaluation window
- **THEN** runtime keeps current rollout phase and records `within_budget` classification

#### Scenario: Health SLI breaches configured budget
- **WHEN** one or more canonical SLI values exceed configured threshold for breach window
- **THEN** runtime classifies health as `breached` and triggers configured freeze policy

### Requirement: Capacity governance SHALL expose deterministic admission actions
Runtime MUST derive a canonical sandbox capacity action before managed execution:
- `allow`
- `throttle`
- `deny`

Capacity action MUST be computed from deterministic queue depth and inflight budget evaluation.

#### Scenario: Capacity is within inflight and queue limits
- **WHEN** queue depth and inflight counts are below configured limits
- **THEN** capacity action is `allow`

#### Scenario: Queue depth exceeds soft threshold
- **WHEN** queue depth exceeds throttle threshold but remains below hard deny threshold
- **THEN** capacity action is `throttle`

#### Scenario: Queue depth exceeds hard deny threshold
- **WHEN** queue depth exceeds hard deny threshold
- **THEN** capacity action is `deny`

### Requirement: Rollout breach handling SHALL support automatic freeze and controlled unfreeze
When health budget is breached under enforce rollout policy, runtime MUST support automatic transition to `frozen` with canonical freeze reason code.

Exiting `frozen` MUST require configured cooldown completion and explicit unfreeze token validation.

#### Scenario: Automatic freeze is triggered by repeated budget breach
- **WHEN** breach classification persists for configured consecutive evaluation windows
- **THEN** runtime transitions to `frozen` and emits canonical freeze reason metadata

#### Scenario: Frozen runtime is manually unfrozen after cooldown
- **WHEN** cooldown has elapsed and a valid unfreeze token is provided
- **THEN** runtime transitions from `frozen` to configured target phase (`canary` or `observe`)

### Requirement: Run and Stream SHALL preserve rollout and capacity semantic equivalence
For equivalent request and effective configuration, Run and Stream MUST produce semantically equivalent rollout phase usage, capacity action, and freeze-driven terminal classification.

#### Scenario: Equivalent requests under frozen phase
- **WHEN** equivalent Run and Stream requests are processed while rollout phase is `frozen`
- **THEN** both paths produce semantically equivalent deny classification and no execution side effects

#### Scenario: Equivalent requests under throttle action
- **WHEN** equivalent Run and Stream requests are processed with capacity action `throttle`
- **THEN** both paths produce semantically equivalent admission classification and observability markers
