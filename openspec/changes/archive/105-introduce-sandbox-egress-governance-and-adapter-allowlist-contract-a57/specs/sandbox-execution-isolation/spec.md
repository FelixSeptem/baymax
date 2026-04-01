## ADDED Requirements

### Requirement: Sandbox execution SHALL apply canonical egress policy resolution per tool selector
For each sandbox-governed tool selector, runtime MUST resolve egress action deterministically using effective egress policy and allowlist rules.

#### Scenario: Equivalent selector with unchanged policy
- **WHEN** the same selector executes multiple times under unchanged egress configuration
- **THEN** runtime resolves semantically equivalent egress action and reason code each time

#### Scenario: Selector-specific override exists
- **WHEN** per-selector egress override is configured
- **THEN** runtime uses selector override instead of global default action

### Requirement: Egress deny path SHALL preserve sandbox fail-fast semantics
When egress action resolves to deny in enforce mode, runtime MUST block the outbound action before execution side effects and emit canonical violation classification.

#### Scenario: Outbound request denied in enforce mode
- **WHEN** sandbox egress policy resolves outbound target to deny
- **THEN** runtime blocks request and returns canonical `sandbox.egress_deny` classification

#### Scenario: Equivalent deny under Run and Stream
- **WHEN** equivalent Run and Stream requests hit identical egress deny condition
- **THEN** both paths produce semantically equivalent deny classification

### Requirement: Egress allow-and-record SHALL keep deterministic observability semantics
When policy action is `allow_and_record`, runtime MUST allow outbound action and persist deterministic observability markers for policy source and decision.

#### Scenario: Allow-and-record action executes outbound request
- **WHEN** egress policy resolves to `allow_and_record`
- **THEN** runtime executes request and records canonical decision metadata

#### Scenario: Equivalent allow-and-record events are replayed
- **WHEN** duplicate equivalent egress decision events are ingested
- **THEN** logical egress aggregate counters remain replay-idempotent
