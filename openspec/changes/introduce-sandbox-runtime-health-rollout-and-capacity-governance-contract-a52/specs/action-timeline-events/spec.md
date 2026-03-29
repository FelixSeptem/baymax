## ADDED Requirements

### Requirement: Action timeline SHALL emit canonical sandbox rollout-governance reasons
When sandbox rollout governance affects execution path, action timeline events MUST emit canonical reason codes in `sandbox.rollout.*` namespace.

Canonical reasons for this milestone MUST include:
- `sandbox.rollout.phase_canary`
- `sandbox.rollout.phase_frozen`
- `sandbox.rollout.health_budget_breached`
- `sandbox.rollout.capacity_throttle`
- `sandbox.rollout.capacity_denied`

#### Scenario: Timeline records canary rollout phase usage
- **WHEN** request is processed under sandbox rollout phase `canary`
- **THEN** related timeline event includes reason `sandbox.rollout.phase_canary`

#### Scenario: Timeline records frozen deny reason
- **WHEN** admission denies execution because rollout is frozen
- **THEN** timeline event includes reason `sandbox.rollout.phase_frozen`

#### Scenario: Timeline records capacity throttle reason
- **WHEN** admission applies throttle action under rollout capacity policy
- **THEN** timeline event includes reason `sandbox.rollout.capacity_throttle`

### Requirement: Run and Stream SHALL preserve rollout-governance timeline equivalence
For equivalent inputs and rollout-governance state, Run and Stream timeline reason semantics MUST remain equivalent.

#### Scenario: Equivalent frozen-path requests in Run and Stream
- **WHEN** equivalent Run and Stream requests are denied by frozen rollout phase
- **THEN** both paths emit semantically equivalent timeline reason classification

#### Scenario: Equivalent throttle-path requests in Run and Stream
- **WHEN** equivalent Run and Stream requests are handled under throttle action
- **THEN** both paths emit semantically equivalent timeline reason and status semantics
