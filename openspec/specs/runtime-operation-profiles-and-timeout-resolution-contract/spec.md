# runtime-operation-profiles-and-timeout-resolution-contract Specification

## Purpose
TBD - created by archiving change introduce-runtime-operation-profiles-and-timeout-resolution-contract-a41. Update Purpose after archive.
## Requirements
### Requirement: Runtime SHALL expose canonical operation profile set for timeout baselines
Runtime MUST expose operation profiles for cross-domain timeout baseline governance with canonical profile set:
- `legacy`
- `interactive`
- `background`
- `batch`

Runtime MUST resolve default profile as `legacy` when caller does not provide explicit profile selection.

#### Scenario: Request omits operation profile
- **WHEN** host submits managed run request without explicit operation profile
- **THEN** runtime resolves profile to `legacy` and preserves existing default timeout behavior

#### Scenario: Request uses unsupported operation profile
- **WHEN** host submits profile outside canonical set
- **THEN** runtime fails fast with validation error and does not start execution

### Requirement: Timeout resolution SHALL follow deterministic layered precedence
Runtime MUST resolve effective timeout using deterministic layered precedence:
1. profile baseline
2. domain override
3. request override

Resolution MUST remain deterministic for equivalent inputs and MUST NOT apply implicit fallback branches outside configured precedence.

#### Scenario: Domain override supersedes profile baseline
- **WHEN** operation profile baseline and domain override both provide timeout values
- **THEN** runtime uses domain override as effective value before request-level override is evaluated

#### Scenario: Request override supersedes domain override
- **WHEN** request override provides valid timeout value
- **THEN** runtime resolves request override as final candidate before parent-budget clamp

### Requirement: Parent-child timeout convergence SHALL clamp by parent remaining budget
For subagent dispatch and nested execution, runtime MUST clamp child effective timeout by parent remaining budget:
`effective_child_timeout = min(parent_remaining_budget, child_resolved_timeout)`

If parent remaining budget is non-positive or invalid, runtime MUST fail fast and MUST NOT enqueue child execution.

#### Scenario: Child timeout exceeds parent remaining budget
- **WHEN** child resolved timeout is greater than parent remaining budget
- **THEN** runtime clamps child effective timeout to parent remaining budget and records convergence metadata

#### Scenario: Parent remaining budget is exhausted
- **WHEN** parent remaining budget is zero or below zero at child spawn boundary
- **THEN** runtime rejects child dispatch with deterministic timeout-budget error classification

### Requirement: Timeout resolution outcomes SHALL be replay-stable and mode-equivalent
Timeout resolution and parent-child convergence outcomes MUST remain semantically equivalent across Run/Stream paths and across replay/recovery boundaries for equivalent configuration and request inputs.

#### Scenario: Equivalent request under Run and Stream
- **WHEN** equivalent request/configuration is executed via Run and Stream
- **THEN** effective operation profile, resolved timeout source, and child convergence outcome remain semantically equivalent

#### Scenario: Replay restores equivalent timeout resolution
- **WHEN** runtime replays equivalent events after restore
- **THEN** timeout resolution outcome remains stable and does not inflate logical convergence counters

