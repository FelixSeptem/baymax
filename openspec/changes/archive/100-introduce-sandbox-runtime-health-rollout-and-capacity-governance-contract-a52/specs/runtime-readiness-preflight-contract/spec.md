## ADDED Requirements

### Requirement: Readiness preflight SHALL evaluate sandbox rollout-governance readiness findings
Runtime readiness preflight MUST evaluate rollout-governance state and emit canonical findings when rollout configuration or state is unsafe for managed execution.

Canonical finding codes for this milestone MUST include:
- `sandbox.rollout.phase_invalid`
- `sandbox.rollout.health_budget_breached`
- `sandbox.rollout.frozen`
- `sandbox.rollout.capacity_unavailable`

#### Scenario: Preflight detects frozen rollout state
- **WHEN** effective rollout phase is `frozen`
- **THEN** readiness returns canonical `sandbox.rollout.frozen` finding and deterministic status classification

#### Scenario: Preflight detects health budget breach
- **WHEN** rollout health budget classification is `breached`
- **THEN** readiness returns canonical `sandbox.rollout.health_budget_breached` finding with machine-readable metadata

### Requirement: Rollout-governance findings SHALL preserve strict/non-strict mapping semantics
Readiness preflight MUST map rollout-governance findings using existing strict/non-strict policy semantics:
- `strict=false` may classify recoverable rollout risk as `degraded`
- `strict=true` MUST escalate equivalent blocking-class finding to `blocked`

#### Scenario: Non-strict policy with capacity pressure
- **WHEN** rollout finding indicates capacity pressure under non-strict readiness policy
- **THEN** readiness status is `degraded` with canonical rollout-capacity finding

#### Scenario: Strict policy with equivalent capacity pressure
- **WHEN** the same rollout-capacity finding is evaluated with strict readiness policy
- **THEN** readiness status escalates to `blocked`
