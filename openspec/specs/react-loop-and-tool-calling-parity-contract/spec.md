# react-loop-and-tool-calling-parity-contract Specification

## Purpose
TBD - created by archiving change introduce-react-loop-and-tool-calling-parity-contract-a56. Update Purpose after archive.
## Requirements
### Requirement: Runner SHALL execute canonical ReAct loop with deterministic step semantics
The runtime MUST execute ReAct as a canonical loop:
- model step planning,
- tool dispatch execution,
- tool result feedback into next model step,
- deterministic termination evaluation.

Run and Stream paths MUST share equivalent loop-core semantics for iteration advance, tool result merge, and termination decision.

#### Scenario: Managed Run executes multi-iteration ReAct loop
- **WHEN** a managed Run request produces tool calls for multiple model steps
- **THEN** runtime iterates through dispatch and feedback until a terminal answer or deterministic termination reason is produced

#### Scenario: Managed Stream executes equivalent multi-iteration ReAct loop
- **WHEN** an equivalent managed Stream request produces the same tool-call plan
- **THEN** runtime follows the same loop-core semantics and reaches semantically equivalent terminal classification

### Requirement: Stream path SHALL support tool dispatch and feedback without unsupported intermediate state
Stream execution MUST support tool dispatch and tool-result feedback as first-class behavior and MUST NOT terminate with `stream_tool_dispatch_not_supported` for supported configurations.

Stream event emission MAY remain incremental, but tool dispatch MUST occur at deterministic step boundary before entering the next model step.

#### Scenario: Stream step emits tool call and dispatches successfully
- **WHEN** Stream execution emits a tool-call intent in an active ReAct step
- **THEN** runtime dispatches tool execution, feeds normalized tool result, and continues to the next model step

#### Scenario: Stream request uses supported ReAct config
- **WHEN** Stream request runs with ReAct enabled and required dependencies available
- **THEN** runtime does not return `stream_tool_dispatch_not_supported` as terminal reason

### Requirement: ReAct loop governance SHALL enforce iteration and run-level tool-call budgets
Runtime MUST enforce both:
- iteration budget (`max_iterations`), and
- run-level tool-call budget (`tool_call_limit`).

Budget enforcement MUST be deterministic and fail-fast for equivalent input and configuration.

#### Scenario: Run-level tool-call budget is exhausted
- **WHEN** cumulative tool calls in one run exceed configured `tool_call_limit`
- **THEN** runtime terminates loop with canonical budget-exhausted classification and performs no further tool dispatch

#### Scenario: Iteration budget is exhausted before final answer
- **WHEN** loop iteration count reaches configured `max_iterations` before terminal answer
- **THEN** runtime terminates with canonical max-iteration classification and does not execute additional model steps

### Requirement: ReAct termination taxonomy SHALL be canonical and machine-assertable
ReAct loop termination MUST map to canonical terminal reason taxonomy with deterministic classification.

Minimum canonical reasons for this milestone:
- `react.completed`
- `react.max_iterations_exceeded`
- `react.tool_call_limit_exceeded`
- `react.tool_dispatch_failed`
- `react.provider_error`
- `react.context_canceled`

#### Scenario: Loop ends with final model answer
- **WHEN** model returns final answer with no additional tool call requirement
- **THEN** runtime terminates with canonical reason `react.completed`

#### Scenario: Tool dispatch fails in-loop
- **WHEN** tool dispatch returns non-recoverable error under fail-fast policy
- **THEN** runtime terminates with canonical reason `react.tool_dispatch_failed`

### Requirement: Run and Stream SHALL preserve ReAct semantic equivalence
For equivalent request input, effective configuration, and dependency state, Run and Stream MUST produce semantically equivalent:
- termination reason taxonomy,
- loop counters,
- budget-hit classifications,
- tool-call aggregate semantics.

Event ordering differences that do not change semantics are allowed.

#### Scenario: Equivalent Run and Stream requests hit budget termination
- **WHEN** equivalent requests in Run and Stream exhaust the same configured ReAct budget
- **THEN** both paths expose semantically equivalent budget-hit classification and loop aggregates

#### Scenario: Equivalent Run and Stream requests complete successfully
- **WHEN** equivalent requests execute ReAct loop and complete without error
- **THEN** both paths return semantically equivalent completion classification and canonical loop counters

### Requirement: Skill Preprocess Run/Stream Parity
`Discover/Compile` skill preprocess MUST run in a unified pre-stage for both `Run` and `Stream`, and MUST produce equivalent behavior under identical inputs.

#### Scenario: Equivalent preprocess enablement
- **WHEN** `runtime.skill.preprocess.enabled=true` for the same request in `Run` and `Stream`
- **THEN** preprocess execution decision and resulting skill bundle selection MUST be equivalent

#### Scenario: Equivalent preprocess fail-fast
- **WHEN** preprocess fails with `fail_mode=fail_fast`
- **THEN** `Run` and `Stream` MUST both return equivalent classified failure and stop before model/tool execution

#### Scenario: Equivalent preprocess degrade
- **WHEN** preprocess fails with `fail_mode=degrade`
- **THEN** `Run` and `Stream` MUST both continue with equivalent degraded markers and reason codes

### Requirement: Hook/Middleware Parity in ReAct Tool Loop
Lifecycle hooks and tool middleware MUST preserve ReAct loop parity between `Run` and `Stream` across reasoning/tool iterations.

#### Scenario: Equivalent iteration-level hook outcomes
- **WHEN** a ReAct flow executes multiple reasoning/tool iterations on `Run` and `Stream`
- **THEN** hook phase counts and terminal hook outcomes MUST be equivalent

#### Scenario: Equivalent middleware short-circuit outcome
- **WHEN** middleware short-circuits tool invocation during ReAct loop
- **THEN** `Run` and `Stream` MUST emit equivalent tool skip and terminal semantics

### Requirement: ReAct Plan Notebook Run-Stream Semantic Equivalence
For equivalent input, configuration, and dependency state, Run and Stream SHALL preserve semantically equivalent plan notebook lifecycle outcomes, including action sequence class, version progression, and terminal plan status.

#### Scenario: Equivalent requests perform plan revision
- **WHEN** equivalent Run and Stream flows trigger a plan revision in the same logical step
- **THEN** both paths MUST expose semantically equivalent `plan_version` progression and `revise` action classification

#### Scenario: Equivalent requests complete with same plan terminal
- **WHEN** equivalent Run and Stream flows complete the same plan branch
- **THEN** both paths MUST expose semantically equivalent plan terminal status and completion action semantics

### Requirement: Plan Change MUST NOT Introduce Parallel ReAct Loop Semantics
Plan notebook and plan-change hook execution MUST be implemented within canonical ReAct loop boundaries and MUST NOT introduce parallel loop taxonomy or separate termination channels.

#### Scenario: Plan notebook enabled with ReAct loop
- **WHEN** notebook and plan-change hooks are enabled
- **THEN** runtime MUST continue using canonical A56 ReAct termination taxonomy for final classification

#### Scenario: Plan change hook failure classification
- **WHEN** plan-change hook returns failure
- **THEN** runtime MUST map outcome to existing loop-compatible classification without introducing alternate loop family

### Requirement: Realtime Interrupt/Resume SHALL Preserve Run and Stream Semantic Equivalence
For equivalent input, effective configuration, and dependency state, Run and Stream MUST preserve semantically equivalent interrupt/resume outcomes, including interrupt acceptance, resume restoration class, and terminal classification.

#### Scenario: Equivalent interrupt accepted in Run and Stream
- **WHEN** equivalent requests in Run and Stream hit identical interrupt trigger
- **THEN** both paths MUST expose semantically equivalent interrupt acceptance and freeze boundary semantics

#### Scenario: Equivalent resume success in Run and Stream
- **WHEN** equivalent requests in Run and Stream resume from semantically equivalent cursor
- **THEN** both paths MUST expose semantically equivalent resume outcome and terminal class

#### Scenario: Equivalent invalid-resume rejection in Run and Stream
- **WHEN** equivalent requests in Run and Stream resume from invalid cursor
- **THEN** both paths MUST expose semantically equivalent rejection classification

### Requirement: Realtime Interrupt/Resume MUST NOT Introduce Parallel Loop Semantics
Realtime interrupt/resume integration MUST remain within canonical loop boundaries and MUST NOT introduce parallel loop family or alternate termination taxonomy.

#### Scenario: Realtime enabled ReAct run remains under canonical taxonomy
- **WHEN** realtime protocol and interrupt/resume are enabled for ReAct path
- **THEN** runtime MUST continue using canonical A56 termination taxonomy without alternate loop family

