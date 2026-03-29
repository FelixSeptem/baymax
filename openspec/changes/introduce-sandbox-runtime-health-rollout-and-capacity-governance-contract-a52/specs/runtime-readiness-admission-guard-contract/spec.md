## ADDED Requirements

### Requirement: Admission guard SHALL enforce sandbox rollout freeze semantics before execution
Runtime readiness-admission guard MUST deny managed execution when rollout phase is `frozen` under enforce policy.

Deny path MUST remain side-effect free (no scheduler enqueue, mailbox publish, or lifecycle mutation).

#### Scenario: Admission evaluates frozen rollout under enforce policy
- **WHEN** admission receives readiness output indicating `sandbox.rollout.frozen`
- **THEN** admission decision is `deny` with deterministic freeze reason classification

#### Scenario: Frozen deny path remains side-effect free
- **WHEN** managed Run/Stream request is denied because rollout is frozen
- **THEN** runtime does not mutate scheduler/mailbox/task lifecycle state

### Requirement: Admission guard SHALL apply sandbox capacity action deterministically
Admission guard MUST consume canonical capacity action (`allow|throttle|deny`) and apply deterministic policy mapping.

For `throttle`, admission MUST follow configured degraded policy:
- `allow_and_record`
- `fail_fast`

#### Scenario: Capacity action is throttle with allow-and-record policy
- **WHEN** readiness reports capacity action `throttle` and degraded policy is `allow_and_record`
- **THEN** admission allows execution and records canonical throttle observability markers

#### Scenario: Capacity action is throttle with fail-fast policy
- **WHEN** readiness reports capacity action `throttle` and degraded policy is `fail_fast`
- **THEN** admission denies execution with deterministic throttle classification

#### Scenario: Capacity action is deny
- **WHEN** readiness reports capacity action `deny`
- **THEN** admission denies execution deterministically and preserves side-effect-free deny semantics

### Requirement: Run and Stream SHALL preserve rollout-capacity admission equivalence
For equivalent inputs and effective configuration, Run and Stream admission decisions MUST remain semantically equivalent for frozen and capacity-governed paths.

#### Scenario: Equivalent Run and Stream requests under throttle policy
- **WHEN** equivalent requests are evaluated with capacity action `throttle`
- **THEN** both paths produce semantically equivalent allow-or-deny admission outcomes according to degraded policy

#### Scenario: Equivalent Run and Stream requests under frozen rollout
- **WHEN** equivalent requests are evaluated with rollout phase `frozen`
- **THEN** both paths return semantically equivalent deny classification
