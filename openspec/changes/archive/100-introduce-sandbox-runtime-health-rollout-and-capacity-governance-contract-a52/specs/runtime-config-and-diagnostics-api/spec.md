## ADDED Requirements

### Requirement: Runtime SHALL expose sandbox rollout-governance config with deterministic precedence
Runtime configuration MUST expose `security.sandbox.rollout.*` with precedence `env > file > default`.

At minimum, rollout-governance config MUST include:
- `phase` (`observe|canary|baseline|full|frozen`)
- `traffic_ratio`
- `health_window`
- `error_budget`
- `freeze_on_breach`
- `cooldown`
- `manual_unfreeze_token`

At minimum, capacity-governance config MUST include:
- `max_inflight`
- `max_queue`
- `throttle_threshold`
- `deny_threshold`
- `degraded_policy` (`allow_and_record|fail_fast`)

Invalid enum/range/transition values MUST fail fast at startup and MUST rollback atomically on hot reload.

#### Scenario: Invalid rollout traffic ratio at startup
- **WHEN** configuration sets rollout `traffic_ratio` outside valid range
- **THEN** runtime startup fails fast with validation error

#### Scenario: Invalid rollout transition in hot reload payload
- **WHEN** hot reload payload requests illegal phase transition
- **THEN** runtime rejects update and preserves previous active snapshot

### Requirement: Runtime diagnostics SHALL include additive rollout and capacity governance fields
Run diagnostics MUST include additive rollout/capacity fields while preserving backward compatibility (`additive + nullable + default`).

At minimum, diagnostics MUST include:
- `sandbox_rollout_phase`
- `sandbox_rollout_effective_ratio`
- `sandbox_health_budget_status`
- `sandbox_health_budget_breach_total`
- `sandbox_freeze_state`
- `sandbox_freeze_reason_code`
- `sandbox_capacity_action`
- `sandbox_capacity_queue_depth`
- `sandbox_capacity_inflight`

#### Scenario: Consumer queries run diagnostics after rollout admission
- **WHEN** diagnostics API returns run summary for sandbox-enabled request
- **THEN** response includes rollout/capacity additive fields with canonical enum semantics

#### Scenario: Consumer queries run diagnostics for pre-A52 historical record
- **WHEN** diagnostics API returns legacy run record without rollout governance path
- **THEN** rollout/capacity additive fields are nullable/default without breaking schema compatibility
