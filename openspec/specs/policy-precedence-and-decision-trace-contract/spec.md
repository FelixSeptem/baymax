# policy-precedence-and-decision-trace-contract Specification

## Purpose
TBD - created by archiving change introduce-policy-precedence-and-decision-trace-contract-a58. Update Purpose after archive.
## Requirements
### Requirement: Runtime SHALL apply deterministic policy-stack precedence across security and admission layers
Runtime MUST evaluate policy candidates using a canonical precedence matrix:
1. `action_gate`
2. `security_s2`
3. `sandbox_action`
4. `sandbox_egress`
5. `adapter_allowlist`
6. `readiness_admission`

For equivalent inputs and equivalent configuration, winner stage and deny source MUST remain deterministic.

#### Scenario: Multiple layers produce blocking candidates
- **WHEN** one request simultaneously hits `security_s2` and `sandbox_egress` blocking candidates
- **THEN** runtime selects winner using canonical precedence matrix and emits deterministic winner stage

#### Scenario: Lower-precedence finding co-exists with higher-precedence deny
- **WHEN** `readiness_admission` reports blocked while `action_gate` also reports deny
- **THEN** runtime keeps `action_gate` as winner and does not override with lower stage

### Requirement: Policy tie-break SHALL remain deterministic and observable
When multiple candidates exist in the same stage, runtime MUST apply deterministic tie-break using lexical canonical code and stable source ordering.

Runtime MUST expose tie-break explainability fields for debugging and replay.

#### Scenario: Two candidates collide in one stage
- **WHEN** two `sandbox_egress` deny candidates are both eligible
- **THEN** runtime selects deterministic winner and records canonical tie-break reason

#### Scenario: No same-stage collision
- **WHEN** only one candidate exists in winner stage
- **THEN** runtime emits winner without tie-break drift or conflict inflation

### Requirement: Decision trace SHALL be machine-assertable and replay-stable
Runtime MUST expose decision-trace output including:
- `policy_decision_path`
- `deny_source`
- `winner_stage`
- `tie_break_reason`

These fields MUST stay replay-idempotent under equivalent replay inputs.

#### Scenario: Consumer inspects one denied run
- **WHEN** QueryRuns returns a denied run
- **THEN** policy decision trace fields are present with canonical values

#### Scenario: Replay re-ingests equivalent decision events
- **WHEN** equivalent policy decision events are replayed multiple times
- **THEN** decision-trace aggregate semantics remain stable after first ingestion

### Requirement: Policy precedence outputs SHALL remain Run/Stream equivalent
For equivalent request context, Run and Stream MUST resolve the same winner stage and deny source.

#### Scenario: Equivalent Run and Stream request under policy conflict
- **WHEN** managed Run and Stream requests evaluate the same policy candidates
- **THEN** winner stage and deny source are semantically equivalent

#### Scenario: Deny path executes in either mode
- **WHEN** policy winner is deny in both Run and Stream
- **THEN** deny path remains side-effect-free and produces equivalent terminal classification

