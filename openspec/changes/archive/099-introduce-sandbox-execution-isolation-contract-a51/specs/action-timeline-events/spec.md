## ADDED Requirements

### Requirement: Action timeline SHALL include canonical sandbox reason taxonomy
Action timeline events MUST include canonical sandbox reason codes for sandbox-governed tool execution outcomes.

At minimum, reason codes MUST cover:
- `sandbox.policy_deny`
- `sandbox.launch_failed`
- `sandbox.timeout`
- `sandbox.fallback_allow_and_record`
- `sandbox.capability_mismatch`
- `sandbox.tool_not_adapted`

#### Scenario: Timeline records sandbox policy deny
- **WHEN** sandbox policy resolves a tool call to deny
- **THEN** timeline event reason code is `sandbox.policy_deny`

#### Scenario: Timeline records sandbox timeout
- **WHEN** sandbox executor times out while executing a tool call
- **THEN** timeline event reason code is `sandbox.timeout`

### Requirement: Run and Stream SHALL preserve sandbox timeline semantic equivalence
For equivalent input and effective configuration, Run and Stream MUST emit semantically equivalent sandbox timeline reason and status transitions.

#### Scenario: Equivalent sandbox fallback in Run and Stream timeline
- **WHEN** equivalent requests in Run and Stream hit sandbox launch failure with allow-and-record fallback
- **THEN** timeline outputs contain semantically equivalent `sandbox.fallback_allow_and_record` reason semantics

#### Scenario: Equivalent sandbox deny in Run and Stream timeline
- **WHEN** equivalent requests in Run and Stream are denied by sandbox policy
- **THEN** timeline outputs contain semantically equivalent `sandbox.policy_deny` reason semantics
