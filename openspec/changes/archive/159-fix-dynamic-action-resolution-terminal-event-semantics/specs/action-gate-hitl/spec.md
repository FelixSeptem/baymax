## MODIFIED Requirements

### Requirement: Action Gate SHALL evaluate dynamically registered post-tool actions

In addition to pre-dispatch Action Gate evaluation, runtime MUST evaluate a valid dynamic action registration at the tool-result boundary before the next model or executor step. The dynamic evaluation MUST preserve allow, require-confirm, deny, timeout, and fail-closed semantics and MUST remain semantically equivalent between Run and Stream. For every accepted confirm, deny, or timeout decision, runtime MUST emit exactly one `run.dynamic_action.resolved` event after the checkpoint records the decision. Only deny and timeout decisions MUST additionally emit a canceled `run.finished` terminal event; confirm MUST continue the original Run/Stream without a canceled terminal projection.

#### Scenario: Dynamic action requires confirmation
- **WHEN** a completed tool registers a resumable action whose decision is `require_confirm`
- **THEN** runtime pauses the same Run as `input_required` and does not start the next model or executor step before confirmation

#### Scenario: Dynamic action confirmation is accepted
- **WHEN** a matching confirm decision is admitted for an active dynamic-action checkpoint
- **THEN** runtime emits one `run.dynamic_action.resolved` event with `decision=confirm` and accepted admission, then continues the original Run/Stream without emitting a canceled `run.finished` event

#### Scenario: Dynamic action is denied
- **WHEN** a dynamic action decision resolves to `deny`
- **THEN** runtime emits one accepted `run.dynamic_action.resolved` event, emits one canceled `run.finished` terminal event, does not resume the Run, and does not invoke the pending executor

#### Scenario: Dynamic confirmation times out
- **WHEN** a dynamic action confirmation exceeds the configured timeout
- **THEN** runtime emits one accepted `run.dynamic_action.resolved` event with the timeout decision, emits one canceled `run.finished` terminal event, and does not execute the action

#### Scenario: Duplicate dynamic action decision is idempotent
- **WHEN** the same decision and idempotency key are submitted again after an accepted resolution
- **THEN** runtime returns the prior result without emitting another `run.dynamic_action.resolved` or canceled terminal event

#### Scenario: Run and Stream dynamic gate outcomes remain equivalent
- **WHEN** equivalent Run and Stream executions register the same dynamic action and receive the same decision
- **THEN** gate checks, resolution event cardinality, pause/resume classification, executor suppression, and terminal projection remain semantically equivalent
