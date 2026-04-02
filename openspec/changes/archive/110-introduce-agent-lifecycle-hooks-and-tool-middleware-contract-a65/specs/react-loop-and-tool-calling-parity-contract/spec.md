## ADDED Requirements

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
