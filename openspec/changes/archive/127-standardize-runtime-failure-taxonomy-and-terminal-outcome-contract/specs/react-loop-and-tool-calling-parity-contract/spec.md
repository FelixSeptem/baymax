## MODIFIED Requirements

### Requirement: ReAct termination taxonomy SHALL be canonical and machine-assertable
ReAct loop termination MUST map to canonical terminal reason taxonomy with deterministic classification. Existing canonical reasons remain authoritative, and each terminal projection MUST additionally expose the shared normalized failure family and execution phase when applicable. Minimum canonical reasons for this milestone remain:
- `react.completed`
- `react.max_iterations_exceeded`
- `react.tool_call_limit_exceeded`
- `react.tool_dispatch_failed`
- `react.provider_error`
- `react.context_canceled`

#### Scenario: Loop ends with final model answer
- **WHEN** model returns final answer with no additional tool call requirement
- **THEN** runtime terminates with canonical reason `react.completed`, terminal state `completed`, and normalized family `none`

#### Scenario: Tool dispatch fails in-loop
- **WHEN** tool dispatch returns non-recoverable error under fail-fast policy
- **THEN** runtime terminates with canonical reason `react.tool_dispatch_failed`, preserves the source tool classification, and exposes normalized family `runtime_failed` or `policy_denied` according to the owner decision

#### Scenario: ReAct context is canceled
- **WHEN** the canonical loop is canceled before completion
- **THEN** runtime terminates with `react.context_canceled`, terminal state `canceled`, normalized family `canceled`, and no retry claim is synthesized

### Requirement: Run and Stream SHALL preserve ReAct semantic equivalence
For equivalent request input, effective configuration, and dependency state, Run and Stream MUST produce semantically equivalent termination reason taxonomy, normalized failure family, execution phase, loop counters, budget-hit classifications, tool-call aggregate semantics, and terminal outcome. Event ordering differences that do not change semantics are allowed.

#### Scenario: Equivalent Run and Stream hit budget termination
- **WHEN** equivalent requests in Run and Stream exhaust the same configured ReAct budget
- **THEN** both expose the same canonical budget reason, normalized family `runtime_failed`, phase `post_start`, and equivalent loop aggregates

#### Scenario: Equivalent Run and Stream complete successfully
- **WHEN** equivalent requests execute ReAct loop and complete without error
- **THEN** both expose semantically equivalent completion classification, terminal state, and canonical loop counters
