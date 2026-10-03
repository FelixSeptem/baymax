## ADDED Requirements

### Requirement: Action Gate SHALL evaluate dynamically registered post-tool actions

In addition to pre-dispatch Action Gate evaluation, runtime MUST evaluate a valid dynamic action registration at the tool-result boundary before the next model or executor step. The dynamic evaluation MUST preserve allow, require-confirm, deny, timeout, and fail-closed semantics and MUST remain semantically equivalent between Run and Stream.

#### Scenario: Dynamic action requires confirmation
- **WHEN** a completed tool registers a resumable action whose decision is `require_confirm`
- **THEN** runtime pauses the same Run as `input_required` and does not start the next model or executor step before confirmation

#### Scenario: Dynamic action is denied
- **WHEN** a dynamic action decision resolves to `deny`
- **THEN** runtime preserves the existing classified denial semantics, does not resume the Run, and does not invoke the pending executor

#### Scenario: Dynamic confirmation times out
- **WHEN** a dynamic action confirmation exceeds the configured timeout
- **THEN** runtime records timeout-deny semantics and leaves the Run in its source-owned terminal or resumable outcome without executing the action

#### Scenario: Run and Stream dynamic gate outcomes remain equivalent
- **WHEN** equivalent Run and Stream executions register the same dynamic action and receive the same decision
- **THEN** gate checks, pause/resume classification, executor suppression, and terminal projection remain semantically equivalent
